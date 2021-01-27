package p2p

import (
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/constants"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/nbs"
	"github.com/julienschmidt/httprouter"
)

type SyncRemote struct {
	p2p *P2P
}

func GetRemoteChunkStore() chunks.ChunkStore {
	return &P2PChunkStore{
		getQueue:      make(chan chunks.ReadRequest),
		hasQueue:      make(chan chunks.ReadRequest),
		finishedChan:  make(chan struct{}),
		rateLimit:     make(chan struct{}, 6),
		workerWg:      &sync.WaitGroup{},
		cacheMu:       &sync.RWMutex{},
		unwrittenPuts: nbs.NewCache(),
		rootMu:        &sync.RWMutex{},
	}
}

type P2PChunkStore struct {
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
}

func (p2pcs *P2PChunkStore) getRoot(checkVers bool) (root hash.Hash, vers string) {
	// GET http://<host>/root. Response will be ref of root.
	res := p2pcs.requestRoot("GET", hash.Hash{}, hash.Hash{})

	// FIXME: check expected version

	defer closeResponse(res.Body)

	checkStatus(http.StatusOK, res, res.Body)
	data, err := ioutil.ReadAll(res.Body)
	d.PanicIfError(err)

	return hash.Parse(string(data)), res.Header.Get(NomsVersionHeader)
}

func (p2pcs *P2PChunkStore) requestRoot(method string, current, last hash.Hash) *http.Response {
	u := *hcs.host
	u.Path = httprouter.CleanPath(hcs.host.Path + constants.RootPath)
	if method == "POST" {
		params := u.Query()
		params.Add("last", last.String())
		params.Add("current", current.String())
		u.RawQuery = params.Encode()
	}

	req := newRequest(method, hcs.auth, u.String(), nil, nil)

	res, err := hcs.httpClient.Do(req)
	d.PanicIfError(err)

	return res
}

//
// public methods
//

func (p2pcs *P2PChunkStore) Get(h hash.Hash) chunks.Chunk {
	return chunks.Chunk{}
}

func (p2pcs *P2PChunkStore) GetMany(hashes hash.HashSet, foundChunks chan *chunks.Chunk) {
	return
}

func (p2pcs *P2PChunkStore) Has(h hash.Hash) bool {
	return false
}

func (p2pcs *P2PChunkStore) HasMany(hashes hash.HashSet) (absent hash.HashSet) {
	return hash.HashSet{}
}

func (p2pcs *P2PChunkStore) Put(c chunks.Chunk) {
	return
}

func (p2pcs *P2PChunkStore) Version() string {
	return ""
}

func (p2pcs *P2PChunkStore) Rebase() {
	return
}

func (p2pcs *P2PChunkStore) Root() hash.Hash {
	return hash.Hash{}
}

func (p2pcs *P2PChunkStore) Commit(current, last hash.Hash) bool {
	return false
}

func (p2pcs *P2PChunkStore) Stats() interface{} {
	return nil
}

func (p2pcs *P2PChunkStore) StatsSummary() string {
	return ""
}

func (p2pcs *P2PChunkStore) Close() error {
	return nil
}

// NewSyncRemote creates a new remote sync handler
func NewSyncRemote(p2p *P2P) *SyncRemote {
	syncRemote := &SyncRemote{
		p2p: p2p,
	}
	return syncRemote
}
