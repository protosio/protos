package p2p

import (
	"fmt"
	"sync"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/nbs"
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

func (p2pcs *P2PRemoteChunkStore) getRoot(checkVers bool) (root hash.Hash, vers string) {
	res, err := p2pcs.requestRoot(hash.Hash{}, hash.Hash{})
	d.PanicIfError(err)

	if checkVers && p2pcs.version != res.nomsVersion {
		d.Panic(
			"Version skew\n\r"+
				"\tServer data version changed from '%s' to '%s'\n\r"+
				"\tResponse: %s\n",
			p2pcs.version, res.nomsVersion, res.root)
	}

	return hash.Parse(res.root), res.nomsVersion
}

func (p2pcs *P2PRemoteChunkStore) requestRoot(current, last hash.Hash) (*getRootResp, error) {
	peerID, err := peer.IDFromString(p2pcs.id)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse peer ID from string: %w", err)
	}

	respData := &getRootResp{}

	// send the request
	log.Infof("Sending getRoot request '%s'", peerID.String())
	err = p2pcs.p2p.sendRequest(peerID, getRootHandler, emptyReq{}, respData)
	if err != nil {
		return nil, fmt.Errorf("getRoot request to '%s' failed: %s", peerID.String(), err.Error())
	}

	return respData, nil
}

func (p2pcs *P2PRemoteChunkStore) setRoot(current, last hash.Hash) (*getRootResp, error) {
	peerID, err := peer.IDFromString(p2pcs.id)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse peer ID from string: %w", err)
	}

	req := &setRootReq{
		last:    last.String(),
		current: current.String(),
	}

	respData := &getRootResp{}

	// send the request
	log.Infof("Sending setRoot request '%s'", peerID.String())
	err = p2pcs.p2p.sendRequest(peerID, getRootHandler, req, respData)
	if err != nil {
		return nil, fmt.Errorf("setRoot request to '%s' failed: %s", peerID.String(), err.Error())
	}

	return respData, nil
}

//
// public methods
//

func (p2pcs *P2PRemoteChunkStore) Get(h hash.Hash) chunks.Chunk {
	return chunks.Chunk{}
}

func (p2pcs *P2PRemoteChunkStore) GetMany(hashes hash.HashSet, foundChunks chan *chunks.Chunk) {
	return
}

func (p2pcs *P2PRemoteChunkStore) Has(h hash.Hash) bool {
	return false
}

func (p2pcs *P2PRemoteChunkStore) HasMany(hashes hash.HashSet) (absent hash.HashSet) {
	return hash.HashSet{}
}

func (p2pcs *P2PRemoteChunkStore) Put(c chunks.Chunk) {
	return
}

func (p2pcs *P2PRemoteChunkStore) Version() string {
	return ""
}

func (p2pcs *P2PRemoteChunkStore) Rebase() {
	return
}

func (p2pcs *P2PRemoteChunkStore) Root() hash.Hash {
	return hash.Hash{}
}

func (p2pcs *P2PRemoteChunkStore) Commit(current, last hash.Hash) bool {
	return false
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
