package db

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
func Open(protosDir string, protosDB string) (core.DB, error) {
	dbpath := path.Join(protosDir, protosDB)
	if _, err := os.Stat(dbpath); os.IsNotExist(err) {
		err := os.Mkdir(dbpath, 0755)
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
func (db *dbNoms) InitSet(dataset string) error {
	ds := db.dbn.GetDataset(dataset)

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
func (db *dbNoms) InitMap(dataset string) error {
	ds := db.dbn.GetDataset(dataset)

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
