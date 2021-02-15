package db

import (
	"fmt"
	"net"
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
	dbPort   = 19199
	sharedDS = "protosshared"
	localDS  = "protoslocal"
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
			return &dbNoms{}, fmt.Errorf("Failed to open database: %w", err)
		}
	}

	cs := nbs.NewLocalStore(dbpath, clienttest.DefaultMemTableSize)
	db := datas.NewDatabase(cs)

	var err error

	// create local dataset
	lds := db.GetDataset(localDS)
	_, found := lds.MaybeHeadValue()
	if !found {
		mapi := types.NewMap(lds.Database())
		lds, err = db.CommitValue(lds, mapi)
		if err != nil {
			return &dbNoms{}, fmt.Errorf("Error creating local dataset: %w", err)
		}
	}

	// create shared dataset
	sds := db.GetDataset(sharedDS)
	_, found = sds.MaybeHeadValue()
	if !found {
		mapi := types.NewMap(sds.Database())
		sds, err = db.CommitValue(sds, mapi)
		if err != nil {
			return &dbNoms{}, fmt.Errorf("Error creating local dataset: %w", err)
		}
	}

	return &dbNoms{dbn: db, cs: cs, sharedDatasets: map[string]bool{}}, nil
}

// DB represents a DB client instance, used to interract with the database
type DB interface {
	SaveStruct(dataset string, data interface{}) error
	GetStruct(dataset string, to interface{}) error
	InitMap(dataset string, sync bool) error
	GetMap(dataset string, to interface{}) error
	InsertInMap(dataset string, id string, data interface{}) error
	RemoveFromMap(dataset string, id string) error
	SyncAll(ips []string) error
	SyncTo(srcStore, dstStore datas.Database) error
	SyncCS(cs chunks.ChunkStore) error
	SyncServer(address net.IP) (func() error, error)
	GetChunkStore() chunks.ChunkStore
	Close() error
}

//
// db storm methods for implementing the DB interface
//

type dbNoms struct {
	uri            string
	cs             chunks.ChunkStore
	dbn            datas.Database
	sharedDatasets map[string]bool
}

//
// private methods
//

func (db *dbNoms) getHeadMap(name string) (datas.Dataset, types.Map) {
	var ds datas.Dataset
	_, found := db.sharedDatasets[name]
	if found {
		ds = db.dbn.GetDataset(sharedDS)
	} else {
		ds = db.dbn.GetDataset(localDS)
	}

	hv, ok := ds.MaybeHeadValue()
	if !ok {
		panic("Local or Shared dataset does not have a head value")
	}

	mapHead := hv.(types.Map)
	return ds, mapHead
}

func (db *dbNoms) getSharedHeadMap() (datas.Dataset, types.Map) {

	ds := db.dbn.GetDataset(sharedDS)
	hv, ok := ds.MaybeHeadValue()
	if !ok {
		panic("Shared dataset does not have a head value")
	}

	mapHead := hv.(types.Map)
	return ds, mapHead
}

func (db *dbNoms) getLocalHeadMap() (datas.Dataset, types.Map) {

	ds := db.dbn.GetDataset(localDS)
	hv, ok := ds.MaybeHeadValue()
	if !ok {
		panic("Shared dataset does not have a head value")
	}

	mapHead := hv.(types.Map)
	return ds, mapHead
}

//
// public methods
//

func (db *dbNoms) Close() error {
	return db.dbn.Close()
}

func (db *dbNoms) GetChunkStore() chunks.ChunkStore {
	return db.cs
}

func (db *dbNoms) SyncAll(ips []string) error {

	for _, ip := range ips {
		log.Tracef("Syncing dataset '%s' to '%s'", sharedDS, ip)

		dst := fmt.Sprintf("http://%s:%d::%s", ip, dbPort, sharedDS)
		cfg := config.NewResolver()
		remoteDB, remoteObj, err := cfg.GetPath(dst)
		if err != nil {
			return err
		}

		if remoteObj == nil {
			return fmt.Errorf("Object for dataset '%s' not found on '%s'", sharedDS, "destination")
		}

		// sync local -> remote
		err = db.SyncTo(db.dbn, remoteDB)
		if err != nil {
			return err
		}

		// sync remote -> local
		err = db.SyncTo(remoteDB, db.dbn)
		if err != nil {
			return err
		}

		err = remoteDB.Close()
		if err != nil {
			return err
		}

	}

	return nil
}

func (db *dbNoms) SyncCS(cs chunks.ChunkStore) error {
	cfg := config.NewResolver()
	remoteDB, _, err := cfg.GetDatasetFromChunkStore(cs, sharedDS)
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
	dstDataset := dstStore.GetDataset(sharedDS)

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
				status.Printf("Syncing - %.2f%% (%s/s)", pct, bytesPerSec(info.ApproxWrittenBytes, start))
			}
		}
		lastProgressCh <- last
	}()

	// prepare local db
	srcObj, found := srcStore.GetDataset(sharedDS).MaybeHead()
	if !found {
		return fmt.Errorf("Head not found for local db")
	}
	srcRef := types.NewRef(srcObj)

	dstRef, dstExists := dstDataset.MaybeHeadRef()
	nonFF := false

	// pull the data from, from src towards dst
	datas.Pull(srcStore, dstStore, srcRef, progressCh)

	dstDataset, err := dstStore.FastForward(dstDataset, srcRef)
	if err == datas.ErrMergeNeeded {
		dstDataset, err = dstStore.SetHead(dstDataset, srcRef)
		nonFF = true
	}

	close(progressCh)
	if last := <-lastProgressCh; last.DoneCount > 0 {
		log.Tracef("Done - Synced %s in %s (%s/s)",
			humanize.Bytes(last.ApproxWrittenBytes), since(start), bytesPerSec(last.ApproxWrittenBytes, start))
		status.Done()
	} else if !dstExists {
		log.Tracef("All chunks already exist at destination! Created new dataset %s.\n", sharedDS)
	} else if nonFF && !srcRef.Equals(dstRef) {
		log.Tracef("Abandoning %s; new head is %s\n", dstRef.TargetHash(), srcRef.TargetHash())
	} else {
		log.Tracef("Dataset '%s' is already up to date.\n", sharedDS)
	}

	return nil
}

func (db *dbNoms) SyncServer(address net.IP) (func() error, error) {
	server := datas.NewRemoteDatabaseServer(db.cs, address.String(), dbPort)

	go server.Run()

	stopper := func() error {
		log.Debug("Shutting down DB sync server")
		server.Stop()
		return nil
	}

	return stopper, nil
}

// SaveStruct writes a new value for a given struct
func (db *dbNoms) SaveStruct(dataset string, data interface{}) error {

	ds, mapHead := db.getHeadMap(dataset)

	marshaled, err := marshal.Marshal(db.dbn, data)
	if err != nil {
		return fmt.Errorf("Failed to marshal db data: %w", err)
	}

	_, err = db.dbn.CommitValue(ds, mapHead.Edit().Set(types.String(dataset), marshaled).Map())
	if err != nil {
		return fmt.Errorf("Error committing: %w", err)
	}
	return nil
}

// GetStruct retrieves a struct from a dataset
func (db *dbNoms) GetStruct(dataset string, to interface{}) error {
	_, mapHead := db.getHeadMap(dataset)

	iv, found := mapHead.MaybeGet(types.String(dataset))
	if !found {
		return fmt.Errorf("Struct dataset '%s' not found", dataset)
	}

	err := marshal.Unmarshal(iv.Value(), to)
	if err != nil {
		return fmt.Errorf("Failed to unmarshall: %w", err)
	}
	return nil
}

// GetMap retrieves all records in a map
func (db *dbNoms) GetMap(dataset string, to interface{}) error {
	_, mapHead := db.getHeadMap(dataset)

	iv, found := mapHead.MaybeGet(types.String(dataset))
	if !found {
		return fmt.Errorf("Map dataset '%s' not found", dataset)
	}

	mapi := iv.(types.Map)

	err := marshal.Unmarshal(mapi.Value(), to)
	if err != nil {
		return fmt.Errorf("Failed to unmarshall: %w", err)
	}

	return nil
}

// InsertInMap inserts an element in a map, or updates an existing one
func (db *dbNoms) InsertInMap(dataset string, id string, data interface{}) error {
	ds, mapHead := db.getHeadMap(dataset)

	iv, found := mapHead.MaybeGet(types.String(dataset))
	if !found {
		return fmt.Errorf("Map dataset '%s' not found", dataset)
	}

	mapi := iv.(types.Map)

	marshaled, err := marshal.Marshal(db.dbn, data)
	if err != nil {
		return fmt.Errorf("Failed to marshal db data: %w", err)
	}

	newMapi := mapi.Edit().Set(types.String(id), marshaled).Map()
	newMapHead := mapHead.Edit().Set(types.String(dataset), marshal.MustMarshal(db.dbn, newMapi)).Map()
	_, err = db.dbn.CommitValue(ds, marshal.MustMarshal(db.dbn, newMapHead))
	if err != nil {
		return fmt.Errorf("Error committing: %w", err)
	}
	return nil
}

// RemoveFromMap removes an element from a map
func (db *dbNoms) RemoveFromMap(dataset string, id string) error {
	ds, mapHead := db.getHeadMap(dataset)

	iv, found := mapHead.MaybeGet(types.String(dataset))
	if !found {
		return fmt.Errorf("Map dataset '%s' not found", dataset)
	}

	mapi := iv.(types.Map)

	newMapi := mapi.Edit().Remove(types.String(id)).Map()
	newMapHead := mapHead.Edit().Set(types.String(dataset), marshal.MustMarshal(db.dbn, newMapi)).Map()
	_, err := db.dbn.CommitValue(ds, marshal.MustMarshal(db.dbn, newMapHead))
	if err != nil {
		return fmt.Errorf("Error committing: %w", err)
	}
	return nil
}

// InitMap initializes a map dataset in the db
func (db *dbNoms) InitMap(name string, sync bool) error {
	log.Tracef("Initializing map '%s' (sync: '%t')", name, sync)
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
		return fmt.Errorf("Error committing map '%s': %w", name, err)
	}

	return nil
}
