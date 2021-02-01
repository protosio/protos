package p2p

import (
	"fmt"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/types"
)

const getRootHandler = "getRoot"

type setRootReq struct {
	last    string
	current string
}

type getRootResp struct {
	root        string
	nomsVersion string
}

type P2PServerChunkStore struct {
	cs chunks.ChunkStore
}

func (p2pcs *P2PServerChunkStore) getRoot(data interface{}) (interface{}, error) {
	resp := getRootResp{
		root:        p2pcs.cs.Root().String(),
		nomsVersion: p2pcs.cs.Version(),
	}

	return resp, nil
}

func (p2pcs *P2PServerChunkStore) setRoot(data interface{}) (interface{}, error) {
	req, ok := data.(*setRootReq)
	if !ok {
		return getRootResp{}, fmt.Errorf("Unknown data struct for setRoot request")
	}

	last := hash.Parse(req.last)
	proposed := hash.Parse(req.current)

	vs := types.NewValueStore(p2pcs.cs)

	// Even though the Root is actually a Map<String, Ref<Commit>>, its Noms Type is Map<String, Ref<Value>> in order to prevent the root chunk from getting bloated with type info. That means that the Value of the proposed new Root needs to be manually type-checked. The simplest way to do that would be to iterate over the whole thing and pull the target of each Ref from |cs|. That's a lot of reads, though, and it's more efficient to just read the Value indicated by |last|, diff the proposed new root against it, and validate whatever new entries appear.
	lastMap := datas.ValidateLast(last, vs)

	proposedMap := datas.ValidateProposed(proposed, last, vs)
	if !proposedMap.Empty() {
		datas.AssertMapOfStringToRefOfCommit(proposedMap, lastMap, vs)
	}

	for to, from := proposed, last; !vs.Commit(to, from); {
		// If committing failed, we go read out the map of Datasets at the root of the store, which is a Map[string]Ref<Commit>
		rootMap := types.NewMap(vs)
		root := vs.Root()
		if v := vs.ReadValue(root); v != nil {
			rootMap = v.(types.Map)
		}

		// Since we know that lastMap is an ancestor of both proposedMap and
		// rootMap, we can try to do a three-way merge here. We don't want to
		// traverse the Ref<Commit>s stored in the maps, though, just
		// basically merge the maps together as long the changes to rootMap
		// and proposedMap were in different Datasets.
		merged, err := datas.MergeDatasetMaps(proposedMap, rootMap, lastMap, vs)
		if err != nil {
			return getRootResp{
				root:        p2pcs.cs.Root().String(),
				nomsVersion: p2pcs.cs.Version(),
			}, fmt.Errorf("Attempted root map auto-merge failed: %s", err)
		}
		to, from = vs.WriteValue(merged).TargetHash(), root
	}

	return getRootResp{
		root:        p2pcs.cs.Root().String(),
		nomsVersion: p2pcs.cs.Version(),
	}, nil

}
