package db

import (
	"fmt"
	"os"
	"os/signal"
	"path"
	"syscall"
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
	"github.com/protosio/protos/internal/core"
	"github.com/protosio/protos/internal/util"
)

var log = util.GetLogger("db")

const (
	dbPort = 19199
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
func Open(protosDir string, protosDB string) (core.DB, error) {
	dbpath := path.Join(protosDir, protosDB)
	if _, err := os.Stat(dbpath); os.IsNotExist(err) {
		err := os.Mkdir(dbpath, 0755)
		if err != nil {
			return &dbNoms{}, fmt.Errorf("Failed to open database: %w", err)
		}
	}

	cs := nbs.NewLocalStore(dbpath, clienttest.DefaultMemTableSize)
	db := datas.NewDatabase(cs)
	return &dbNoms{dbn: db, cs: cs, datasetsSync: map[string]bool{}}, nil
}

//
// db storm methods for implementing the DB interface
//

type dbNoms struct {
	uri          string
	cs           chunks.ChunkStore
	dbn          datas.Database
	datasetsSync map[string]bool
}

func (db *dbNoms) SyncAll(ips []string) error {
	for ds, syncable := range db.datasetsSync {
		if syncable {
			for _, ip := range ips {
				log.Tracef("Syncing dataset '%s' to '%s'", ds, ip)
				err := db.SyncTo(ds, ip)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (db *dbNoms) SyncTo(dataset string, ip string) error {

	// prepare destination db
	dst := fmt.Sprintf("http://%s:%d::%s", ip, dbPort, dataset)
	cfg := config.NewResolver()
	dstStore, dstObj, err := cfg.GetPath(dst)
	if err != nil {
		return err
	}

	dstDataset := dstStore.GetDataset(dataset)

	defer dstStore.Close()

	if dstObj == nil {
		return fmt.Errorf("Object for dataset '%s' not found on '%s'", dataset, ip)
	}

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
	srcStore := db.dbn
	srcObj, found := srcStore.GetDataset(dataset).MaybeHead()
	if !found {
		return fmt.Errorf("Object not found for local db")
	}
	srcRef := types.NewRef(srcObj)

	dstRef, dstExists := dstDataset.MaybeHeadRef()
	nonFF := false

	datas.Pull(srcStore, dstStore, srcRef, progressCh)

	dstDataset, err = dstStore.FastForward(dstDataset, srcRef)
	if err == datas.ErrMergeNeeded {
		dstDataset, err = dstStore.SetHead(dstDataset, srcRef)
		nonFF = true
	}

	close(progressCh)
	if last := <-lastProgressCh; last.DoneCount > 0 {
		status.Printf("Done - Synced %s in %s (%s/s)",
			humanize.Bytes(last.ApproxWrittenBytes), since(start), bytesPerSec(last.ApproxWrittenBytes, start))
		status.Done()
	} else if !dstExists {
		fmt.Printf("All chunks already exist at destination! Created new dataset %s.\n", dst)
	} else if nonFF && !srcRef.Equals(dstRef) {
		fmt.Printf("Abandoning %s; new head is %s\n", dstRef.TargetHash(), srcRef.TargetHash())
	} else {
		fmt.Printf("Dataset %s is already up to date.\n", dst)
	}

	return nil
}

func (db *dbNoms) SyncServer() error {
	server := datas.NewRemoteDatabaseServer(db.cs, dbPort)

	// Shutdown server gracefully so that profile may be written
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		server.Stop()
	}()
	go server.Run()
	return nil
}

func (db *dbNoms) Close() error {
	return db.dbn.Close()
}

// SaveStruct writes a new value for a given struct
func (db *dbNoms) SaveStruct(dataset string, data interface{}) error {

	ds := db.dbn.GetDataset(dataset)

	marshaled, err := marshal.Marshal(db.dbn, data)
	if err != nil {
		return fmt.Errorf("Failed to marshal db data: %w", err)
	}

	_, err = db.dbn.CommitValue(ds, marshaled)
	if err != nil {
		return fmt.Errorf("Error committing: %w", err)
	}
	return nil
}

// GetStruct retrieves a struct from a dataset
func (db *dbNoms) GetStruct(dataset string, to interface{}) error {
	ds := db.dbn.GetDataset(dataset)
	hv, ok := ds.MaybeHeadValue()
	if !ok {
		return fmt.Errorf("No record found")
	}

	err := marshal.Unmarshal(hv.Value(), to)
	if err != nil {
		return fmt.Errorf("Failed to unmarshall: %w", err)
	}
	return nil
}

// GetSet retrieves all records in a set
func (db *dbNoms) GetSet(dataset string, to interface{}) error {
	ds := db.dbn.GetDataset(dataset)

	hv, ok := ds.MaybeHeadValue()
	if !ok {
		return fmt.Errorf("Dataset '%s' not found", dataset)
	}

	set := hv.(types.Set)

	err := marshal.Unmarshal(set.Value(), to)
	if err != nil {
		return fmt.Errorf("Failed to unmarshall: %w", err)
	}

	return nil
}

// InsertInSet inserts an element in a set
func (db *dbNoms) InsertInSet(dataset string, data interface{}) error {
	ds := db.dbn.GetDataset(dataset)

	hv, ok := ds.MaybeHeadValue()
	if !ok {
		return fmt.Errorf("Dataset '%s' not found", dataset)
	}

	set := hv.(types.Set)

	marshaled, err := marshal.Marshal(db.dbn, data)
	if err != nil {
		return fmt.Errorf("Failed to marshal db data: %w", err)
	}

	_, err = db.dbn.CommitValue(ds, set.Edit().Insert(marshaled).Set())
	if err != nil {
		return fmt.Errorf("Error committing: %w", err)
	}
	return nil
}

// RemoveFromSet removes an element from a set
func (db *dbNoms) RemoveFromSet(dataset string, data interface{}) error {
	ds := db.dbn.GetDataset(dataset)

	hv, ok := ds.MaybeHeadValue()
	if !ok {
		return fmt.Errorf("Dataset '%s' not found", dataset)
	}

	set := hv.(types.Set)

	marshaled, err := marshal.Marshal(db.dbn, data)
	if err != nil {
		return fmt.Errorf("Failed to marshal db data: %w", err)
	}

	_, err = db.dbn.CommitValue(ds, set.Edit().Remove(marshaled).Set())
	if err != nil {
		return fmt.Errorf("Error committing: %w", err)
	}
	return nil
}

// InitSet initializes a set dataset in the db
func (db *dbNoms) InitSet(dataset string, sync bool) error {
	ds := db.dbn.GetDataset(dataset)

	if sync {
		db.datasetsSync[dataset] = true
	}

	_, ok := ds.MaybeHeadValue()
	if ok {
		return nil
	}

	set := types.NewSet(ds.Database())

	_, err := db.dbn.CommitValue(ds, set)
	if err != nil {
		return fmt.Errorf("Error committing: %w", err)
	}

	return nil
}

// GetMap retrieves all records in a map
func (db *dbNoms) GetMap(dataset string, to interface{}) error {
	ds := db.dbn.GetDataset(dataset)

	hv, ok := ds.MaybeHeadValue()
	if !ok {
		return fmt.Errorf("Dataset '%s' not found", dataset)
	}

	mapi := hv.(types.Map)

	err := marshal.Unmarshal(mapi.Value(), to)
	if err != nil {
		return fmt.Errorf("Failed to unmarshall: %w", err)
	}

	return nil
}

// InsertInMap inserts an element in a map, or updates an existing one
func (db *dbNoms) InsertInMap(dataset string, id string, data interface{}) error {
	ds := db.dbn.GetDataset(dataset)

	hv, ok := ds.MaybeHeadValue()
	if !ok {
		return fmt.Errorf("Dataset '%s' not found", dataset)
	}

	mapi := hv.(types.Map)

	marshaled, err := marshal.Marshal(db.dbn, data)
	if err != nil {
		return fmt.Errorf("Failed to marshal db data: %w", err)
	}

	_, err = db.dbn.CommitValue(ds, mapi.Edit().Set(types.String(id), marshaled).Map())
	if err != nil {
		return fmt.Errorf("Error committing: %w", err)
	}
	return nil
}

// RemoveFromMap removes an element from a map
func (db *dbNoms) RemoveFromMap(dataset string, id string) error {
	ds := db.dbn.GetDataset(dataset)

	hv, ok := ds.MaybeHeadValue()
	if !ok {
		return fmt.Errorf("Dataset '%s' not found", dataset)
	}

	mapi := hv.(types.Map)

	_, err := db.dbn.CommitValue(ds, mapi.Edit().Remove(types.String(id)).Map())
	if err != nil {
		return fmt.Errorf("Error committing: %w", err)
	}
	return nil
}

// InitMap initializes a map dataset in the db
func (db *dbNoms) InitMap(dataset string, sync bool) error {
	ds := db.dbn.GetDataset(dataset)

	if sync {
		db.datasetsSync[dataset] = true
	}

	_, ok := ds.MaybeHeadValue()
	if ok {
		return nil
	}

	mapi := types.NewMap(ds.Database())

	_, err := db.dbn.CommitValue(ds, mapi)
	if err != nil {
		return fmt.Errorf("Error committing: %w", err)
	}

	return nil
}
