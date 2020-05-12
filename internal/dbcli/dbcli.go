package dbcli

import (
	"fmt"
	"os"
	"path"

	"github.com/attic-labs/noms/go/datas"
	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/nbs"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/clienttest"
	"github.com/protosio/protos/internal/core"
)

// Open opens a noms database on the provided path
func Open(protosDir string, protosDB string) (core.DBCLI, error) {
	dbpath := path.Join(protosDir, protosDB)
	if _, err := os.Stat(dbpath); os.IsNotExist(err) {
		err := os.Mkdir(dbpath, os.ModeDir)
		if err != nil {
			return &dbNoms{}, fmt.Errorf("Failed to open database: %w", err)
		}
	}
	db := datas.NewDatabase(nbs.NewLocalStore(dbpath, clienttest.DefaultMemTableSize))
	return &dbNoms{dbn: db}, nil
}

//
// db storm methods for implementing the DB interface
//

type dbNoms struct {
	uri string
	dbn datas.Database
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

	var set types.Set
	hv, ok := ds.MaybeHeadValue()
	if ok {
		set = hv.(types.Set)
	} else {
		set = types.NewSet(ds.Database())
	}

	err := marshal.Unmarshal(set.Value(), to)
	if err != nil {
		return fmt.Errorf("Failed to unmarshall: %w", err)
	}

	return nil
}

// InsertInSet inserts an element in a set, or updates an existing one
func (db *dbNoms) InsertInSet(dataset string, data interface{}) error {
	ds := db.dbn.GetDataset(dataset)

	var set types.Set
	hv, ok := ds.MaybeHeadValue()
	if ok {
		set = hv.(types.Set)
	} else {
		set = types.NewSet(ds.Database())
	}

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

	var set types.Set
	hv, ok := ds.MaybeHeadValue()
	if ok {
		set = hv.(types.Set)
	} else {
		set = types.NewSet(ds.Database())
	}

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
