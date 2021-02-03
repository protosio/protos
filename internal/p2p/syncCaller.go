package p2p

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/constants"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/nbs"
	"github.com/attic-labs/noms/go/util/verbose"
	"github.com/julienschmidt/httprouter"
	"github.com/libp2p/go-libp2p-core/peer"
)

// NewChunkStore creates a new remote chunk store
func NewRemoteChunkStore(p2p *P2P, id string) *P2PRemoteChunkStore {
	return &P2PRemoteChunkStore{
		getQueue:      make(chan chunks.ReadRequest),
		hasQueue:      make(chan chunks.ReadRequest),
		finishedChan:  make(chan struct{}),
		rateLimit:     make(chan struct{}, 6),
		workerWg:      &sync.WaitGroup{},
		cacheMu:       &sync.RWMutex{},
		unwrittenPuts: nbs.NewCache(),
		rootMu:        &sync.RWMutex{},

		p2p: p2p,
		id:  id,
	}
}

type P2PRemoteChunkStore struct {
	getQueue     chan chunks.ReadRequest
	hasQueue     chan chunks.ReadRequest
	finishedChan chan struct{}
	rateLimit    chan struct{}
	workerWg     *sync.WaitGroup

	cacheMu       *sync.RWMutex
	unwrittenPuts *nbs.NomsBlockCache

	rootMu  *sync.RWMutex
	root    hash.Hash
	version string

	p2p *P2P
	id  string
}

func expectVersion(expected string, received string) {
	if expected != received {
		d.Panic(
			"Version skew\n\r"+
				"\tServer data version changed from '%s' to '%s'\n\r",
			expected, received)
	}
}

func (p2pcs *P2PRemoteChunkStore) getRoot(checkVers bool) (root hash.Hash, vers string) {
	peerID, err := peer.IDFromString(p2pcs.id)
	d.PanicIfError(fmt.Errorf("Failed to parse peer ID from string: %w", err))

	respData := &getRootResp{}

	// send the request
	log.Infof("Sending getRoot request '%s'", peerID.String())
	err = p2pcs.p2p.sendRequest(peerID, getRootHandler, emptyReq{}, respData)
	d.PanicIfError(fmt.Errorf("getRoot request to '%s' failed: %s", peerID.String(), err.Error()))

	if checkVers && p2pcs.version != respData.nomsVersion {
		expectVersion(p2pcs.version, respData.nomsVersion)
	}

	return hash.Parse(respData.root), respData.nomsVersion
}

func (p2pcs *P2PRemoteChunkStore) setRoot(current, last hash.Hash) (*getRootResp, error) {
	peerID, err := peer.IDFromString(p2pcs.id)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse peer ID from string: %w", err)
	}

	reqData := &setRootReq{
		last:    last.String(),
		current: current.String(),
	}

	respData := &getRootResp{}

	// send the request
	log.Infof("Sending setRoot request '%s'", peerID.String())
	err = p2pcs.p2p.sendRequest(peerID, getRootHandler, reqData, respData)
	if err != nil {
		return nil, fmt.Errorf("setRoot request to '%s' failed: %s", peerID.String(), err.Error())
	}

	return respData, nil
}

//
// public methods
//

func (p2pcs *P2PRemoteChunkStore) Get(h hash.Hash) chunks.Chunk {
	checkCache := func(h hash.Hash) chunks.Chunk {
		p2pcs.cacheMu.RLock()
		defer p2pcs.cacheMu.RUnlock()
		return p2pcs.unwrittenPuts.Get(h)
	}
	if pending := checkCache(h); !pending.IsEmpty() {
		return pending
	}

	ch := make(chan *chunks.Chunk)
	defer close(ch)

	select {
	case <-p2pcs.finishedChan:
		d.Panic("Tried to Get %s from closed ChunkStore", h)
	case p2pcs.getQueue <- chunks.NewGetRequest(h, ch):
	}

	return *(<-ch)
}

func (p2pcs *P2PRemoteChunkStore) GetMany(hashes hash.HashSet, foundChunks chan *chunks.Chunk) {
	cachedChunks := make(chan *chunks.Chunk)
	go func() {
		p2pcs.cacheMu.RLock()
		defer p2pcs.cacheMu.RUnlock()
		defer close(cachedChunks)
		p2pcs.unwrittenPuts.GetMany(hashes, cachedChunks)
	}()
	remaining := hash.HashSet{}
	for h := range hashes {
		remaining.Insert(h)
	}
	for c := range cachedChunks {
		remaining.Remove(c.Hash())
		foundChunks <- c
	}

	if len(remaining) == 0 {
		return
	}
	wg := &sync.WaitGroup{}
	wg.Add(len(remaining))
	select {
	case <-p2pcs.finishedChan:
		d.Panic("Tried to GetMany from closed ChunkStore")
	case p2pcs.getQueue <- chunks.NewGetManyRequest(remaining, wg, foundChunks):
	}
	wg.Wait()
}

func (p2pcs *P2PRemoteChunkStore) Has(h hash.Hash) bool {
	checkCache := func(h hash.Hash) bool {
		p2pcs.cacheMu.RLock()
		defer p2pcs.cacheMu.RUnlock()
		return p2pcs.unwrittenPuts.Has(h)
	}
	if checkCache(h) {
		return true
	}

	ch := make(chan bool)
	defer close(ch)
	select {
	case <-p2pcs.finishedChan:
		d.Panic("Tried to Has %s on closed ChunkStore", h)
	case p2pcs.hasQueue <- chunks.NewAbsentRequest(h, ch):
	}

	return <-ch
}

func (p2pcs *P2PRemoteChunkStore) HasMany(hashes hash.HashSet) (absent hash.HashSet) {
	var remaining hash.HashSet
	func() {
		p2pcs.cacheMu.RLock()
		defer p2pcs.cacheMu.RUnlock()
		remaining = p2pcs.unwrittenPuts.HasMany(hashes)
	}()
	if len(remaining) == 0 {
		return remaining
	}

	notFoundChunks := make(chan hash.Hash)
	wg := &sync.WaitGroup{}
	wg.Add(len(remaining))
	select {
	case <-p2pcs.finishedChan:
		d.Panic("Tried to HasMany on closed ChunkStore")
	case p2pcs.hasQueue <- chunks.NewAbsentManyRequest(remaining, wg, notFoundChunks):
	}
	go func() { defer close(notFoundChunks); wg.Wait() }()

	absent = hash.HashSet{}
	for notFound := range notFoundChunks {
		absent.Insert(notFound)
	}
	return absent
}

func (p2pcs *P2PRemoteChunkStore) Put(c chunks.Chunk) {
	p2pcs.cacheMu.RLock()
	defer p2pcs.cacheMu.RUnlock()
	select {
	case <-p2pcs.finishedChan:
		d.Panic("Tried to Put %s into closed ChunkStore", c.Hash())
	default:
	}
	p2pcs.unwrittenPuts.Insert(c)
}

func (p2pcs *P2PRemoteChunkStore) Version() string {
	return p2pcs.version
}

func (p2pcs *P2PRemoteChunkStore) Rebase() {
	root, _ := p2pcs.getRoot(true)
	p2pcs.rootMu.Lock()
	defer p2pcs.rootMu.Unlock()
	p2pcs.root = root
}

func (p2pcs *P2PRemoteChunkStore) Root() hash.Hash {
	p2pcs.rootMu.RLock()
	defer p2pcs.rootMu.RUnlock()
	return p2pcs.root
}

func (p2pcs *P2PRemoteChunkStore) Commit(current, last hash.Hash) bool {
	p2pcs.rootMu.Lock()
	defer p2pcs.rootMu.Unlock()
	p2pcs.cacheMu.Lock()
	defer p2pcs.cacheMu.Unlock()

	select {
	case <-p2pcs.finishedChan:
		d.Panic("Tried to Commit %s to closed ChunkStore", current)
	case p2pcs.rateLimit <- struct{}{}:
		defer func() { <-p2pcs.rateLimit }()
	}

	if count := p2pcs.unwrittenPuts.Count(); count > 0 {
		url := *p2pcs.host
		url.Path = httprouter.CleanPath(hcs.host.Path + constants.WriteValuePath)
		verbose.Log("Sending %d chunks", count)
		sendWriteRequest(url, hcs.auth, p2pcs.version, p2pcs.unwrittenPuts, hcs.httpClient)
		verbose.Log("Finished sending %d hashes", count)
		p2pcs.unwrittenPuts.Destroy()
		p2pcs.unwrittenPuts = nbs.NewCache()
	}

	// POST http://<host>/root?current=<ref>&last=<ref>. Response will be 200 on success, 409 if current is outdated. Regardless, the server returns its current root for this store
	resp, err := p2pcs.setRoot(current, last)
	d.PanicIfError(err)
	expectVersion(p2pcs.version, resp.nomsVersion)

	// FIXME: figure out a way to pass status codes. They are used all over the place in noms

	var success bool
	switch resp.StatusCode {
	case http.StatusOK:
		success = true
	case http.StatusConflict:
		success = false
	default:
		buf := bytes.Buffer{}
		buf.ReadFrom(res.Body)
		body := buf.String()
		d.Chk.Fail(
			fmt.Sprintf("Unexpected response: %s: %s",
				http.StatusText(res.StatusCode),
				body))
		return false
	}
	data, err := ioutil.ReadAll(res.Body)
	d.PanicIfError(err)
	p2pcs.root = hash.Parse(string(data))
	return success
}

func (p2pcs *P2PRemoteChunkStore) Stats() interface{} {
	return nil
}

func (p2pcs *P2PRemoteChunkStore) StatsSummary() string {
	return ""
}

func (p2pcs *P2PRemoteChunkStore) Close() error {
	return nil
}
