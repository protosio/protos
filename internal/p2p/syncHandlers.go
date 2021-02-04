package p2p

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"runtime"
	"time"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/d"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/hash"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/verbose"
	"github.com/golang/snappy"
)

const (
	getRootHandler    = "getRoot"
	setRootHandler    = "setRoot"
	writeValueHandler = "writeValue"
)

type setRootReq struct {
	last    string
	current string
}

type getRootResp struct {
	root        string
	nomsVersion string
}

type setRootResp struct {
	root        string
	nomsVersion string
	status      int
}

type writeValueReq struct {
	data string
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
			return setRootResp{
				root:        p2pcs.cs.Root().String(),
				nomsVersion: p2pcs.cs.Version(),
				status:      http.StatusConflict,
			}, nil
		}
		to, from = vs.WriteValue(merged).TargetHash(), root
	}

	return setRootResp{
		root:        p2pcs.cs.Root().String(),
		nomsVersion: p2pcs.cs.Version(),
		status:      http.StatusOK,
	}, nil

}

func (p2pcs *P2PServerChunkStore) writeValue(data interface{}) (interface{}, error) {
	req, ok := data.(*writeValueReq)
	if !ok {
		return emptyResp{}, fmt.Errorf("Unknown data struct for writeValue request")
	}

	byteData, err := base64.StdEncoding.DecodeString(req.data)
	if !ok {
		return emptyResp{}, fmt.Errorf("Failed to base64 decode data in writeValue request: %s", err.Error())
	}

	t1 := time.Now()
	totalDataWritten := 0
	chunkCount := 0

	defer func() {
		verbose.Log("Wrote %d Kb as %d chunks from remote peer in %s", totalDataWritten/1024, chunkCount, time.Since(t1))
	}()

	reader := ioutil.NopCloser(snappy.NewReader(bytes.NewReader(byteData)))
	defer func() {
		// Ensure all data on reader is consumed
		io.Copy(ioutil.Discard, reader)
		reader.Close()
	}()
	vdc := types.NewValidatingDecoder(p2pcs.cs)

	// Deserialize chunks from reader in background, recovering from errors
	errChan := make(chan error)
	chunkChan := make(chan *chunks.Chunk, runtime.NumCPU())

	go func() {
		var err error
		defer func() { errChan <- err; close(errChan) }()
		defer close(chunkChan)
		err = chunks.Deserialize(reader, chunkChan)
	}()

	decoded := make(chan chan types.DecodedChunk, runtime.NumCPU())

	go func() {
		defer close(decoded)
		for c := range chunkChan {
			ch := make(chan types.DecodedChunk)
			decoded <- ch

			go func(ch chan types.DecodedChunk, c *chunks.Chunk) {
				ch <- vdc.Decode(c)
			}(ch, c)
		}
	}()

	unresolvedRefs := hash.HashSet{}
	for ch := range decoded {
		dc := <-ch
		if dc.Chunk != nil && dc.Value != nil {
			(*dc.Value).WalkRefs(func(r types.Ref) {
				unresolvedRefs.Insert(r.TargetHash())
			})

			totalDataWritten += len(dc.Chunk.Data())
			p2pcs.cs.Put(*dc.Chunk)
			chunkCount++
			if chunkCount%100 == 0 {
				verbose.Log("Enqueued %d chunks", chunkCount)
			}
		}
	}

	// If there was an error during chunk deserialization, raise so it can be logged and responded to.
	if err := <-errChan; err != nil {
		d.Panic("Deserialization failure: %v", err)
	}

	if chunkCount > 0 {
		types.PanicIfDangling(unresolvedRefs, p2pcs.cs)
		for !p2pcs.cs.Commit(p2pcs.cs.Root(), p2pcs.cs.Root()) {
		}
	}

	return emptyResp{}, nil
}
