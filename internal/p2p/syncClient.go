package p2p

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/nbs"
	"github.com/attic-labs/noms/go/util/verbose"
	"github.com/golang/snappy"
	"github.com/libp2p/go-libp2p-core/peer"
)

const (
	readThreshold = 1 << 12 // 4K
)

// NewRemoteChunkStore creates a new remote chunk store based on libp2p
func NewRemoteChunkStore(p2p *P2P, peerID peer.ID) *ClientChunkStore {
	p2pcs := &ClientChunkStore{
		getQueue:      make(chan chunks.ReadRequest),
		hasQueue:      make(chan chunks.ReadRequest),
		finishedChan:  make(chan struct{}),
		rateLimit:     make(chan struct{}, 6),
		workerWg:      &sync.WaitGroup{},
		cacheMu:       &sync.RWMutex{},
		unwrittenPuts: nbs.NewCache(),
		rootMu:        &sync.RWMutex{},

		p2p:    p2p,
		peerID: peerID,
	}
	p2pcs.root, p2pcs.version = p2pcs.getRoot(false)
	p2pcs.batchReadRequests(p2pcs.getQueue, p2pcs.getRefs)
	p2pcs.batchReadRequests(p2pcs.hasQueue, p2pcs.hasRefs)
	return p2pcs
}

func serializeHashes(w io.Writer, batch chunks.ReadBatch) {
	err := binary.Write(w, binary.BigEndian, uint32(len(batch))) // 4 billion hashes is probably absurd. Maybe this should be smaller?
	d.PanicIfError(err)
	for h := range batch {
		serializeHash(w, h)
	}
}

func serializeHash(w io.Writer, h hash.Hash) {
	_, err := w.Write(h[:])
	d.PanicIfError(err)
}

func buildHashesRequest(batch chunks.ReadBatch) io.ReadCloser {
	body, pw := io.Pipe()
	go func() {
		defer pw.Close()
		serializeHashes(pw, batch)
	}()
	return body
}

// ClientChunkStore is a client to a remote chunk store server
type ClientChunkStore struct {
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

	p2p    *P2P
	peerID peer.ID
}

func expectVersion(expected string, received string) {
	if expected != received {
		d.Panic(
			"Version skew\n\r"+
				"\tServer data version changed from '%s' to '%s'\n\r",
			expected, received)
	}
}

func (p2pcs *ClientChunkStore) getRoot(checkVers bool) (root hash.Hash, vers string) {

	respData := &getRootResp{}

	// send the request
	log.Infof("Sending getRoot request '%s'", p2pcs.peerID.String())
	err := p2pcs.p2p.sendRequest(p2pcs.peerID, getRootHandler, emptyReq{}, respData)
	d.PanicIfError(err)

	if checkVers && p2pcs.version != respData.NomsVersion {
		expectVersion(p2pcs.version, respData.NomsVersion)
	}

	return hash.Parse(respData.Root), respData.NomsVersion
}

func (p2pcs *ClientChunkStore) setRoot(current, last hash.Hash) (*setRootResp, error) {
	reqData := &setRootReq{
		Last:    last.String(),
		Current: current.String(),
	}

	respData := &setRootResp{}

	// send the request
	log.Infof("Sending setRoot request '%s'", p2pcs.peerID.String())
	err := p2pcs.p2p.sendRequest(p2pcs.peerID, getRootHandler, reqData, respData)
	if err != nil {
		return nil, fmt.Errorf("setRoot request to '%s' failed: %s", p2pcs.peerID.String(), err.Error())
	}

	return respData, nil
}

type batchGetter func(batch chunks.ReadBatch)

func (p2pcs *ClientChunkStore) batchReadRequests(queue <-chan chunks.ReadRequest, getter batchGetter) {
	p2pcs.workerWg.Add(1)
	go func() {
		defer p2pcs.workerWg.Done()
		for done := false; !done; {
			select {
			case req := <-queue:
				p2pcs.sendReadRequests(req, queue, getter)
			case <-p2pcs.finishedChan:
				done = true
			}
		}
	}()
}

func (p2pcs *ClientChunkStore) sendReadRequests(req chunks.ReadRequest, queue <-chan chunks.ReadRequest, getter batchGetter) {
	batch := chunks.ReadBatch{}

	addReq := func(req chunks.ReadRequest) {
		for h := range req.Hashes() {
			batch[h] = append(batch[h], req.Outstanding())
		}
	}

	addReq(req)
	for drained := false; !drained && len(batch) < readThreshold; {
		select {
		case req := <-queue:
			addReq(req)
		default:
			drained = true
		}
	}

	p2pcs.rateLimit <- struct{}{}
	go func() {
		defer batch.Close()
		defer func() { <-p2pcs.rateLimit }()

		getter(batch)
	}()
}

func (p2pcs *ClientChunkStore) getRefs(batch chunks.ReadBatch) {

	// FIXME: figure out the query
	// Indicate to the server that we're OK reading chunks from any store that knows about our root
	// q := "root=" + p2pcs.root.String()
	// if u.RawQuery != "" {
	// 	q = u.RawQuery + "&" + q
	// }
	// u.RawQuery = q

	hashes := buildHashesRequest(batch)
	nb := &bytes.Buffer{}
	_, err := io.Copy(nb, hashes)
	d.PanicIfError(err)

	encodedBody := base64.StdEncoding.EncodeToString(nb.Bytes())

	resp := &getRefsResp{}

	err = p2pcs.p2p.sendRequest(p2pcs.peerID, getRefsHandler, getRefsReq{Hashes: encodedBody}, resp)
	d.Chk.NoError(err)

	// FIXME: check version in every call

	byteChunks, err := base64.StdEncoding.DecodeString(resp.Chunks)
	d.Chk.NoError(err)

	reader := ioutil.NopCloser(snappy.NewReader(bytes.NewReader(byteChunks)))

	chunkChan := make(chan *chunks.Chunk, 16)
	go func() { defer close(chunkChan); chunks.Deserialize(reader, chunkChan) }()

	for c := range chunkChan {
		h := c.Hash()
		for _, or := range batch[h] {
			go or.Satisfy(h, c)
		}
		delete(batch, c.Hash())
	}
}

func (p2pcs *ClientChunkStore) hasRefs(batch chunks.ReadBatch) {

	hashes := buildHashesRequest(batch)
	nb := &bytes.Buffer{}
	_, err := io.Copy(nb, hashes)
	d.PanicIfError(err)

	encodedBody := base64.StdEncoding.EncodeToString(nb.Bytes())

	resp := &hasRefsResp{}

	err = p2pcs.p2p.sendRequest(p2pcs.peerID, hasRefsHandler, hasRefsReq{Hashes: encodedBody}, resp)
	d.Chk.NoError(err)

	// FIXME: check version in every call

	byteChunks, err := base64.StdEncoding.DecodeString(resp.Hashes)
	d.Chk.NoError(err)

	reader := ioutil.NopCloser(snappy.NewReader(bytes.NewReader(byteChunks)))

	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		h := hash.Parse(scanner.Text())
		for _, outstanding := range batch[h] {
			outstanding.Satisfy(h, &chunks.EmptyChunk)
		}
		delete(batch, h)
	}
}

//
// public methods
//

func (p2pcs *ClientChunkStore) Get(h hash.Hash) chunks.Chunk {
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

func (p2pcs *ClientChunkStore) GetMany(hashes hash.HashSet, foundChunks chan *chunks.Chunk) {
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

func (p2pcs *ClientChunkStore) Has(h hash.Hash) bool {
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

func (p2pcs *ClientChunkStore) HasMany(hashes hash.HashSet) (absent hash.HashSet) {
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

func (p2pcs *ClientChunkStore) Put(c chunks.Chunk) {
	p2pcs.cacheMu.RLock()
	defer p2pcs.cacheMu.RUnlock()
	select {
	case <-p2pcs.finishedChan:
		d.Panic("Tried to Put %s into closed ChunkStore", c.Hash())
	default:
	}
	p2pcs.unwrittenPuts.Insert(c)
}

func (p2pcs *ClientChunkStore) Version() string {
	return p2pcs.version
}

func (p2pcs *ClientChunkStore) Rebase() {
	root, _ := p2pcs.getRoot(true)
	p2pcs.rootMu.Lock()
	defer p2pcs.rootMu.Unlock()
	p2pcs.root = root
}

func (p2pcs *ClientChunkStore) Root() hash.Hash {
	p2pcs.rootMu.RLock()
	defer p2pcs.rootMu.RUnlock()
	return p2pcs.root
}

func (p2pcs *ClientChunkStore) Commit(current, last hash.Hash) bool {
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
		verbose.Log("Sending %d chunks", count)

		chunkChan := make(chan *chunks.Chunk, 1024)
		go func() {
			p2pcs.unwrittenPuts.ExtractChunks(chunkChan)
			close(chunkChan)
		}()

		body := datas.BuildWriteValueRequest(chunkChan)

		nb := &bytes.Buffer{}
		var err error
		_, err = io.Copy(nb, body)
		d.PanicIfError(err)

		encodedBody := base64.StdEncoding.EncodeToString(nb.Bytes())

		// send the write value request
		// FIXME: check version
		log.Infof("Sending writeValue request '%s'", p2pcs.peerID.String())
		err = p2pcs.p2p.sendRequest(p2pcs.peerID, writeValueHandler, writeValueReq{Data: encodedBody}, emptyResp{})
		d.PanicIfError(fmt.Errorf("writeValue request to '%s' failed: %s", p2pcs.peerID.String(), err.Error()))
		verbose.Log("Finished sending %d hashes", count)

		p2pcs.unwrittenPuts.Destroy()
		p2pcs.unwrittenPuts = nbs.NewCache()
	}

	// POST http://<host>/root?current=<ref>&last=<ref>. Response will be 200 on success, 409 if current is outdated. Regardless, the server returns its current root for this store
	resp, err := p2pcs.setRoot(current, last)
	d.PanicIfError(err)
	expectVersion(p2pcs.version, resp.NomsVersion)

	var success bool
	switch resp.status {
	case http.StatusOK:
		success = true
	case http.StatusConflict:
		success = false
	default:
		d.Chk.Fail(
			fmt.Sprintf("Unexpected status: %s",
				http.StatusText(resp.status)))
		return false
	}
	p2pcs.root = hash.Parse(resp.Root)
	return success
}

func (p2pcs *ClientChunkStore) Stats() interface{} {
	return nil
}

func (p2pcs *ClientChunkStore) StatsSummary() string {
	respData := &getStatsSummaryHandlerResp{}
	err := p2pcs.p2p.sendRequest(p2pcs.peerID, getStatsSummaryHandler, emptyReq{}, respData)
	d.PanicIfError(err)

	return respData.Stats
}

func (p2pcs *ClientChunkStore) Close() error {
	p2pcs.rootMu.Lock()
	defer p2pcs.rootMu.Unlock()

	close(p2pcs.finishedChan)
	p2pcs.workerWg.Wait()

	close(p2pcs.getQueue)
	close(p2pcs.hasQueue)
	close(p2pcs.rateLimit)

	p2pcs.cacheMu.Lock()
	defer p2pcs.cacheMu.Unlock()
	p2pcs.unwrittenPuts.Destroy()
	return nil
}
