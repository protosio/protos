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
	"github.com/protosio/protos/internal/util"
)

var log = util.GetLogger("db")

const (
	dbPort     = 19199
	sharedData = "protosshared"
	localData  = "protoslocal"
)

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
	db := datas.NewDatabase(cs)

	var err error

	// create local dataset
	lds := db.GetDataset(localData)
	_, found := lds.MaybeHeadValue()
	if !found {
		mapi := types.NewMap(lds.Database())
		lds, err = db.CommitValue(lds, mapi)
		if err != nil {
			return &dbNoms{}, fmt.Errorf("error creating local dataset: %w", err)
		}
	}

	// create shared dataset
	sds := db.GetDataset(sharedData)
	_, found = sds.MaybeHeadValue()
	if !found {
		mapi := types.NewMap(sds.Database())
		sds, err = db.CommitValue(sds, mapi)
		if err != nil {
			return &dbNoms{}, fmt.Errorf("error creating local dataset: %w", err)
		}
	}

	return &dbNoms{dbn: db, cs: cs, sharedDatasets: map[string]bool{}, remoteChunkStores: map[string]chunks.ChunkStore{}}, nil
}

type Refresher interface {
	Refresh() error
}

// DB represents a DB client instance, used to interract with the database
type DB interface {
	SaveStruct(dataset string, data interface{}) error
	GetStruct(dataset string, to interface{}) error
	InitMap(dataset string, sync bool) error
	GetMap(dataset string, to interface{}) error
	InsertInMap(dataset string, id string, data interface{}) error
	RemoveFromMap(dataset string, id string) error
	SyncCS(cs chunks.ChunkStore) error
	GetChunkStore() chunks.ChunkStore
	SyncAll()
	AddRemoteCS(id string, cs chunks.ChunkStore)
	DeleteRemoteCS(id string)
	AddRefresher(refresher Refresher)
	Close() error
}

//
// db storm methods for implementing the DB interface
//

type dbNoms struct {
	cs                chunks.ChunkStore
	remoteChunkStores map[string]chunks.ChunkStore
	dbn               datas.Database
	sharedDatasets    map[string]bool
	refresher         Refresher
}

//
// private methods
//

func (db *dbNoms) getHeadMap(name string) (datas.Dataset, types.Map, bool) {
	var ds datas.Dataset
	_, found := db.sharedDatasets[name]
	var shared bool
	if found {
		ds = db.dbn.GetDataset(sharedData)
		shared = true
	} else {
		ds = db.dbn.GetDataset(localData)
		shared = false
	}

	hv, ok := ds.MaybeHeadValue()
	if !ok {
		panic("Local or Shared dataset does not have a head value")
	}

	mapHead := hv.(types.Map)
	return ds, mapHead, shared
}

func (db *dbNoms) getSharedHeadMap() (datas.Dataset, types.Map) {

	ds := db.dbn.GetDataset(sharedData)
	hv, ok := ds.MaybeHeadValue()
	if !ok {
		panic("Shared dataset does not have a head value")
	}

	mapHead := hv.(types.Map)
	return ds, mapHead
}

func (db *dbNoms) getLocalHeadMap() (datas.Dataset, types.Map) {

	ds := db.dbn.GetDataset(localData)
	hv, ok := ds.MaybeHeadValue()
	if !ok {
		panic("Shared dataset does not have a head value")
	}

	mapHead := hv.(types.Map)
	return ds, mapHead
}

func (db *dbNoms) refresh() {
	if db.refresher != nil {
		go func() {
			err := db.refresher.Refresh()
			if err != nil {
				log.Errorf("Failed to do refresh in db: %s", err.Error())
			}
		}()
	}
}

//
// public methods
//

func (db *dbNoms) AddRefresher(refresher Refresher) {
	db.refresher = refresher
}

func (db *dbNoms) Close() error {
	return db.dbn.Close()
}

func (db *dbNoms) GetChunkStore() chunks.ChunkStore {
	return db.cs
}

// AddRemoteCS adds a remote chunk store which can be synced
func (db *dbNoms) AddRemoteCS(id string, cs chunks.ChunkStore) {
	db.remoteChunkStores[id] = cs
}

// DeleteRemoteCS removes a remote chunk store
func (db *dbNoms) DeleteRemoteCS(id string) {
	delete(db.remoteChunkStores, id)
}

// AddRemoteCS adds a remote chunk store which can be synced
func (db *dbNoms) SyncAll() {
	for id, cs := range db.remoteChunkStores {
		localCS := cs
		localID := id
		go func() {
			err := db.SyncCS(localCS)
			if err != nil {
				log.Errorf("Failed to sync db to '%s': %w", localID, err)
			}
		}()
	}
}

// SyncCS syncs a remote chunk store
func (db *dbNoms) SyncCS(cs chunks.ChunkStore) error {
	cfg := config.NewResolver()
	remoteDB, _, err := cfg.GetDatasetFromChunkStore(cs, sharedData)
	if err != nil {
		return err
	}

	// sync local -> remote
	err = db.SyncTo(db.dbn, remoteDB)
	if err != nil {
		return err
	}

	return nil
}

func (db *dbNoms) SyncTo(srcStore, dstStore datas.Database) error {

	// prepare destination db
	dstDataset := dstStore.GetDataset(sharedData)

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

	// prepare local db
	srcObj, found := srcStore.GetDataset(sharedData).MaybeHead()
	if !found {
		return fmt.Errorf("head not found for local db")
	}
	srcRef := types.NewRef(srcObj)

	dstRef, dstExists := dstDataset.MaybeHeadRef()
	nonFF := false

	// pull the data from, from src towards dst
	datas.Pull(srcStore, dstStore, srcRef, progressCh)

	dstDataset, err := dstStore.FastForward(dstDataset, srcRef)
	if err == datas.ErrMergeNeeded {
		_, err = dstStore.SetHead(dstDataset, srcRef)
		if err != nil {
			return fmt.Errorf("failed to set head on destination store: %w", err)
		}
		nonFF = true
	}

	close(progressCh)
	if last := <-lastProgressCh; last.DoneCount > 0 {
		log.Debugf("Done - Synced %s in %s (%s/s)", humanize.Bytes(last.ApproxWrittenBytes), since(start), bytesPerSec(last.ApproxWrittenBytes, start))
		status.Done()
	} else if !dstExists {
		log.Debugf("All chunks already exist at destination")
	} else if nonFF && !srcRef.Equals(dstRef) {
		log.Debugf("Abandoning %s; new head is %s\n", dstRef.TargetHash(), srcRef.TargetHash())
	} else {
		log.Debugf("Dataset '%s' is already up to date.\n", sharedData)
	}

	return nil
}

// SaveStruct writes a new value for a given struct
func (db *dbNoms) SaveStruct(dataset string, data interface{}) error {

	ds, mapHead, _ := db.getHeadMap(dataset)

	marshaled, err := marshal.Marshal(db.dbn, data)
	if err != nil {
		return fmt.Errorf("failed to marshal db data: %w", err)
	}

	_, err = db.dbn.CommitValue(ds, mapHead.Edit().Set(types.String(dataset), marshaled).Map())
	if err != nil {
		return fmt.Errorf("error committing to DB: %w", err)
	}
	return nil
}

// GetStruct retrieves a struct from a dataset
func (db *dbNoms) GetStruct(dataset string, to interface{}) error {
	_, mapHead, _ := db.getHeadMap(dataset)

	iv, found := mapHead.MaybeGet(types.String(dataset))
	if !found {
		return fmt.Errorf("db struct dataset '%s' not found", dataset)
	}

	err := marshal.Unmarshal(iv.Value(), to)
	if err != nil {
		return fmt.Errorf("failed to unmarshall data from db: %w", err)
	}
	return nil
}

// GetMap retrieves all records in a map
func (db *dbNoms) GetMap(dataset string, to interface{}) error {
	_, mapHead, _ := db.getHeadMap(dataset)

	iv, found := mapHead.MaybeGet(types.String(dataset))
	if !found {
		return fmt.Errorf("db map dataset '%s' not found", dataset)
	}

	mapi := iv.(types.Map)

	err := marshal.Unmarshal(mapi.Value(), to)
	if err != nil {
		return fmt.Errorf("failed to unmarshall data from db: %w", err)
	}

	return nil
}

// InsertInMap inserts an element in a map, or updates an existing one
func (db *dbNoms) InsertInMap(dataset string, id string, data interface{}) error {
	ds, mapHead, shared := db.getHeadMap(dataset)

	iv, found := mapHead.MaybeGet(types.String(dataset))
	if !found {
		return fmt.Errorf("db map dataset '%s' not found", dataset)
	}

	mapi := iv.(types.Map)

	marshaled, err := marshal.Marshal(db.dbn, data)
	if err != nil {
		return fmt.Errorf("failed to marshal db data: %w", err)
	}

	newMapi := mapi.Edit().Set(types.String(id), marshaled).Map()
	newMapHead := mapHead.Edit().Set(types.String(dataset), marshal.MustMarshal(db.dbn, newMapi)).Map()
	_, err = db.dbn.CommitValue(ds, marshal.MustMarshal(db.dbn, newMapHead))
	if err != nil {
		return fmt.Errorf("error committing to db: %w", err)
	}

	db.refresh()
	if shared {
		db.SyncAll()
	}
	return nil
}

// RemoveFromMap removes an element from a map
func (db *dbNoms) RemoveFromMap(dataset string, id string) error {
	ds, mapHead, shared := db.getHeadMap(dataset)

	iv, found := mapHead.MaybeGet(types.String(dataset))
	if !found {
		return fmt.Errorf("db map dataset '%s' not found", dataset)
	}

	mapi := iv.(types.Map)

	newMapi := mapi.Edit().Remove(types.String(id)).Map()
	newMapHead := mapHead.Edit().Set(types.String(dataset), marshal.MustMarshal(db.dbn, newMapi)).Map()
	_, err := db.dbn.CommitValue(ds, marshal.MustMarshal(db.dbn, newMapHead))
	if err != nil {
		return fmt.Errorf("error committing to db: %w", err)
	}

	db.refresh()
	if shared {
		db.SyncAll()
	}
	return nil
}

// InitMap initializes a map dataset in the db
func (db *dbNoms) InitMap(name string, sync bool) error {
	log.Tracef("Initializing db map '%s' (sync: '%t')", name, sync)
	var ds datas.Dataset
	var mapHead types.Map
	if sync {
		ds, mapHead = db.getSharedHeadMap()
		db.sharedDatasets[name] = true
	} else {
		ds, mapHead = db.getLocalHeadMap()
	}

	// if item found in head map, return without doing anything
	_, found := mapHead.MaybeGet(types.String(name))
	if found {
		return nil
	}

	// if item not found in head map, create a new map and add it
	mapNew := types.NewMap(ds.Database())
	newMapHead := mapHead.Edit().Set(types.String(name), marshal.MustMarshal(db.dbn, mapNew)).Map()
	_, err := db.dbn.CommitValue(ds, marshal.MustMarshal(db.dbn, newMapHead))
	if err != nil {
		return fmt.Errorf("error committing map '%s' to db: %w", name, err)
	}

	return nil
}
