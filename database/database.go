package database

import (
	"os"
	"path"

	"github.com/nustiueudinastea/protos/config"
	"github.com/nustiueudinastea/protos/util"

	"github.com/asdine/storm"
	"github.com/asdine/storm/codec/gob"
)

var gconfig = config.Get()
var log = util.Log

// db - package wide db reference
var db *storm.DB

// Exists checks if the database file exists on disk
func Exists() bool {
	dbpath := path.Join(gconfig.WorkDir, "protos.db")
	if _, err := os.Stat(dbpath); os.IsNotExist(err) {
		return false
	}
	return true
}

// Open opens a a boltdb database
func Open() {

	var err error
	dbpath := path.Join(gconfig.WorkDir, "protos.db")
	log.Info("Opening database [", dbpath, "]")
	db, err = storm.Open(dbpath, storm.Codec(gob.Codec))
	if err != nil {
		log.Fatalf("Failed to open database at path %s, %s", dbpath, err.Error())
	}

}

// Close closes the boltdb database
func Close() {
	log.Info("Closing database")
	db.Close()
}

// Save writes a new value for a specific key in a bucket
func Save(data interface{}) error {
	return db.Save(data)
}

// One retrieves one record from the database based on the field name
func One(fieldName string, value interface{}, to interface{}) error {
	return db.One(fieldName, value, to)
}

// All retrieves all records for a specific type
func All(to interface{}) error {
	return db.All(to)
}

// Remove removes a record of specific type
func Remove(data interface{}) error {
	return db.DeleteStruct(data)
}
