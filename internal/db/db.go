package db

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/attic-labs/noms/go/config"
	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/nbs"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/clienttest"
	"github.com/attic-labs/noms/go/util/status"
	"github.com/dustin/go-humanize"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/protosio/protos/internal/util"
)

const (
	defaultDataset = "protos"
)

var log = util.GetLogger("db")

type Refresher interface {
	Refresh() error
}

type Publisher interface {
	Broadcast(dataset string, head string) error
}

// DB represents a DB client instance, used to interract with the database
type DB interface {
	SaveStruct(id string, data interface{}) error
	GetStruct(id string, to interface{}) error
	InitDataset(dataset string, sync bool) error
	GetMap(dataset string, to interface{}) error
	InsertInMap(dataset string, id string, data interface{}) error
	RemoveFromMap(dataset string, id string) error
	GetChunkStore() chunks.ChunkStore
	HasCS(id string) bool
	AddRemoteCS(id string, cs chunks.ChunkStore)
	DeleteRemoteCS(id string)
	Sync(peerID string, dataset string, head string)
	AddRefresher(name string, refresher Refresher)
	AddPublisher(publisher Publisher)
	BroadcastHead()
	Close() error
}

func bytesPerSec(bytes uint64, start time.Time) string {
	bps := float64(bytes) / float64(time.Since(start).Seconds())
	return humanize.Bytes(uint64(bps))
}

func since(start time.Time) string {
	round := time.Second / 100
	now := time.Now().Round(round)
	return now.Sub(start.Round(round)).String()
}

// Open opens a noms database on the provided path
func Open(protosDir string, protosDB string) (DB, error) {
	dbpath := path.Join(protosDir, protosDB)
	if _, err := os.Stat(dbpath); os.IsNotExist(err) {
		err := os.Mkdir(dbpath, 0755)
		if err != nil {
			return &dbNoms{}, fmt.Errorf("failed to open database: %w", err)
		}
	}

	cs := nbs.NewLocalStore(dbpath, clienttest.DefaultMemTableSize)
	dbn := datas.NewDatabase(cs)
	db := &dbNoms{
		dbn:               dbn,
		cs:                cs,
		sharedDatasets:    map[string]bool{},
		remoteChunkStores: cmap.New(),
		refreshers:        cmap.New(),
	}

	err := db.InitDataset(defaultDataset, false)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize db: %w", err)
	}

	return db, nil
}

//
// db storm methods for implementing the DB interface
//

type dbNoms struct {
	cs                chunks.ChunkStore
	remoteChunkStores cmap.ConcurrentMap
	dbn               datas.Database
	sharedDatasets    map[string]bool
	refreshers        cmap.ConcurrentMap
	publisher         Publisher
}

//
// private methods
//

func (db *dbNoms) publishHead(dataset string, head string) {
	if db.publisher != nil {
		log.Debugf("Publishing dataset '%s' head '%s'", dataset, head)
		err := db.publisher.Broadcast(dataset, head)
		if err != nil {
			log.Errorf("Failed to publish DB head: %s", err.Error())
		}
	}
}

func (db *dbNoms) BroadcastHead() {
	for dsName := range db.sharedDatasets {
		ds := db.dbn.GetDataset(dsName)
		db.publishHead(dsName, ds.Head().Hash().String())
	}
}

func (db *dbNoms) getDataset(name string) (datas.Dataset, bool) {
	_, found := db.sharedDatasets[name]
	shared := false
	if found {
		shared = true
	}
	ds := db.dbn.GetDataset(name)
	return ds, shared
}

func (db *dbNoms) refresh() {
	for refresher := range db.refreshers.IterBuffered() {
		go func(ref cmap.Tuple) {
			cRefresher := ref.Val.(Refresher)
			err := cRefresher.Refresh()
			if err != nil {
				log.Errorf("Failed to refresh '%s' in db: %s", ref.Key, err.Error())
			}
		}(refresher)
	}
}

//
// public methods
//

func (db *dbNoms) AddRefresher(name string, refresher Refresher) {
	db.refreshers.Set(name, refresher)
}

func (db *dbNoms) AddPublisher(publisher Publisher) {
	db.publisher = publisher
}

func (db *dbNoms) Close() error {
	return db.dbn.Close()
}

func (db *dbNoms) GetChunkStore() chunks.ChunkStore {
	return db.cs
}

// HasCS checks if a remote chunk store is present
func (db *dbNoms) HasCS(id string) bool {
	return db.remoteChunkStores.Has(id)
}

// AddRemoteCS adds a remote chunk store which can be synced
func (db *dbNoms) AddRemoteCS(id string, cs chunks.ChunkStore) {
	db.remoteChunkStores.Set(id, cs)
}

// DeleteRemoteCS removes a remote chunk store
func (db *dbNoms) DeleteRemoteCS(id string) {
	db.remoteChunkStores.Remove(id)
}

// // SyncAll syncs (push) too all the available peers
// func (db *dbNoms) SyncAll() {
// 	for id, cs := range db.remoteChunkStores {
// 		localCS := cs
// 		localID := id
// 		go func() {
// 			defer func() {
// 				if err := recover(); err != nil {
// 					log.Errorf("Exception during db sync to '%s': %v", localID, err)
// 				}
// 			}()

// 			err := db.pushToRemoteCS(localCS)
// 			if err != nil {
// 				log.Errorf("Failed to sync db to '%s': %s", localID, err.Error())
// 			}
// 		}()
// 	}
// }

// Sync syncs (pull) from a specific peer
func (db *dbNoms) Sync(id string, dataset string, head string) {
	if csRemoteI, found := db.remoteChunkStores.Get(id); found {
		csRemote := csRemoteI.(chunks.ChunkStore)
		go func() {
			defer func() {
				if err := recover(); err != nil {
					log.Errorf("Exception during dataset '%s' sync to '%s': %v", dataset, id, err)
				}
			}()

			csRemote.Rebase()
			localDataset := db.dbn.GetDataset(dataset)
			if localDataset.Head().Hash().String() != head {
				err := db.pullFromRemoteCS(csRemote, dataset)
				if err != nil {
					log.Errorf("Failed to sync db head '%s' from '%s': %s", head, id, err.Error())
				}
			}

		}()
	} else {
		log.Errorf("Failed to sync db head '%s' from '%s': could not find peer '%s'", head, id, id)
	}
}

// // pushToRemoteCS syncs a remote chunk store
// func (db *dbNoms) pushToRemoteCS(cs chunks.ChunkStore) error {
// 	cfg := config.NewResolver()
// 	remoteDB, _, err := cfg.GetDatasetFromChunkStore(cs, sharedData)
// 	if err != nil {
// 		return err
// 	}

// 	// sync local -> remote
// 	err = db.SyncTo(db.dbn, remoteDB)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// pullFromRemoteCS syncs by pulling from a remote chunk store
func (db *dbNoms) pullFromRemoteCS(cs chunks.ChunkStore, dataset string) error {
	cfg := config.NewResolver()
	remoteDB, _, err := cfg.GetDatasetFromChunkStore(cs, dataset)
	if err != nil {
		return err
	}

	// sync local <- remote
	err = db.SyncTo(remoteDB, db.dbn, dataset)
	if err != nil {
		return err
	}

	return nil
}

func (db *dbNoms) SyncTo(srcStore, dstStore datas.Database, dataset string) error {

	// prepare destination db
	dstDataset := dstStore.GetDataset(dataset)

	// sync
	start := time.Now()
	progressCh := make(chan datas.PullProgress)
	lastProgressCh := make(chan datas.PullProgress)

	go func() {
		var last datas.PullProgress

		for info := range progressCh {
			last = info
			if info.KnownCount == 1 {
				// It's better to print "up to date" than "0% (0/1); 100% (1/1)".
				continue
			}

			if status.WillPrint() {
				pct := 100.0 * float64(info.DoneCount) / float64(info.KnownCount)
				status.Printf("Syncing - %.2f%% (%s/s)\n", pct, bytesPerSec(info.ApproxWrittenBytes, start))
			}
		}
		lastProgressCh <- last
	}()

	// prepare src db
	srcObj, found := srcStore.GetDataset(dataset).MaybeHead()
	if !found {
		return fmt.Errorf("head not found for source db dataset '%s'", dataset)
	}
	srcRef := types.NewRef(srcObj)

	dstRef, dstExists := dstDataset.MaybeHeadRef()
	nonFF := false

	// pull the data from src towards dst
	datas.Pull(srcStore, dstStore, srcRef, progressCh)

	dstDataset, err := dstStore.FastForward(dstDataset, srcRef)
	if err == datas.ErrMergeNeeded {
		dstDataset, err = dstStore.SetHead(dstDataset, srcRef)
		if err != nil {
			return fmt.Errorf("failed to set head on destination dataset '%s': %w", dataset, err)
		}
		nonFF = true
	}

	close(progressCh)
	if last := <-lastProgressCh; last.DoneCount > 0 {
		log.Debugf("Done - Synced %s in %s (%s/s)", humanize.Bytes(last.ApproxWrittenBytes), since(start), bytesPerSec(last.ApproxWrittenBytes, start))
		status.Done()
		db.refresh()
	} else if !dstExists {
		log.Debugf("All chunks already exist at destination")
	} else if nonFF && !srcRef.Equals(dstRef) {
		log.Debugf("Abandoning %s; new head is %s\n", dstRef.TargetHash(), srcRef.TargetHash())
	} else {
		log.Debugf("Dataset '%s' is already up to date.\n", dataset)
	}

	return nil
}

// SaveStruct writes a new value for a given struct, in the default protos dataset
func (db *dbNoms) SaveStruct(id string, data interface{}) error {

	ds, _ := db.getDataset(defaultDataset)
	hv, ok := ds.MaybeHeadValue()
	if !ok {
		return fmt.Errorf("dataset '%s' does not have a head value", defaultDataset)
	}

	marshaled, err := marshal.Marshal(db.dbn, data)
	if err != nil {
		return fmt.Errorf("failed to marshal db data: %w", err)
	}

	currentMap := hv.(types.Map)
	newMap := currentMap.Edit().Set(types.String(id), marshaled).Map()
	ds, err = db.dbn.CommitValue(ds, marshal.MustMarshal(db.dbn, newMap))
	if err != nil {
		return fmt.Errorf("error committing to db: %w", err)
	}

	return nil
}

// GetStruct retrieves a struct from the default dataset
func (db *dbNoms) GetStruct(id string, to interface{}) error {
	ds, _ := db.getDataset(defaultDataset)
	hv, ok := ds.MaybeHeadValue()
	if !ok {
		return fmt.Errorf("dataset '%s' does not have a head value", defaultDataset)
	}

	currentMap := hv.(types.Map)
	existingValue, found := currentMap.MaybeGet(types.String(id))
	if !found {
		return fmt.Errorf("db struct '%s' not found in dataset '%s'", id, defaultDataset)
	}

	err := marshal.Unmarshal(existingValue.Value(), to)
	if err != nil {
		return fmt.Errorf("failed to unmarshall data from db: %w", err)
	}
	return nil
}

// GetMap retrieves all records in a map
func (db *dbNoms) GetMap(dataset string, to interface{}) error {
	ds, _ := db.getDataset(dataset)
	hv, ok := ds.MaybeHeadValue()
	if !ok {
		return fmt.Errorf("dataset '%s' does not have a head value", dataset)
	}

	currentMap := hv.(types.Map)
	err := marshal.Unmarshal(currentMap.Value(), to)
	if err != nil {
		return fmt.Errorf("failed to unmarshall data from dataset '%s': %w", dataset, err)
	}

	return nil
}

// InsertInMap inserts an element in a map, or updates an existing one
func (db *dbNoms) InsertInMap(dataset string, id string, data interface{}) error {
	ds, shared := db.getDataset(dataset)
	hv, ok := ds.MaybeHeadValue()
	if !ok {
		return fmt.Errorf("dataset '%s' does not have a head value", dataset)
	}

	marshaled, err := marshal.Marshal(db.dbn, data)
	if err != nil {
		return fmt.Errorf("failed to marshal db data: %w", err)
	}

	currentMap := hv.(types.Map)
	newMap := currentMap.Edit().Set(types.String(id), marshaled).Map()
	ds, err = db.dbn.CommitValue(ds, marshal.MustMarshal(db.dbn, newMap))
	if err != nil {
		return fmt.Errorf("error committing to db: %w", err)
	}

	if shared {
		db.refresh()
		db.publishHead(dataset, ds.Head().Hash().String())
	}
	return nil
}

// RemoveFromMap removes an element from a map
func (db *dbNoms) RemoveFromMap(dataset string, id string) error {
	ds, shared := db.getDataset(dataset)
	hv, ok := ds.MaybeHeadValue()
	if !ok {
		return fmt.Errorf("dataset '%s' does not have a head value", dataset)
	}

	currentMap := hv.(types.Map)
	newMap := currentMap.Edit().Remove(types.String(id)).Map()
	ds, err := db.dbn.CommitValue(ds, marshal.MustMarshal(db.dbn, newMap))
	if err != nil {
		return fmt.Errorf("error committing to db: %w", err)
	}

	if shared {
		db.refresh()
		db.publishHead(dataset, ds.Head().Hash().String())
	}
	return nil
}

// InitDataset initializes a map dataset in the db
func (db *dbNoms) InitDataset(name string, sync bool) error {
	log.Debugf("Initializing dataset '%s'(sync: '%t')", name, sync)
	var err error
	if sync {
		db.sharedDatasets[name] = true
	}

	// create dataset
	ds := db.dbn.GetDataset(name)
	_, found := ds.MaybeHeadValue()
	if !found {
		newMap := types.NewMap(ds.Database())
		ds, err = db.dbn.CommitValue(ds, newMap)
		if err != nil {
			return fmt.Errorf("error creating dataset '%s'(sync: '%t'): %w", name, sync, err)
		}
	}

	return nil
}
