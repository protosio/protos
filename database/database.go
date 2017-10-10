package database

import (
	"encoding/json"
	"errors"
	"path"
	"protos/config"
	"protos/util"

	"github.com/boltdb/bolt"
)

var gconfig = config.Gconfig
var log = util.Log

// DB allows direct interaction with the db from other packages
var DB *bolt.DB

// Open opens a a boltdb database
func Open() {

	// open the database
	var err error
	dbpath := path.Join(gconfig.WorkDir, "protos.db")
	log.Info("Opening database [", dbpath, "]")
	DB, err = bolt.Open(dbpath, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

}

// Close closes the boltdb database
func Close() {
	log.Info("Closing database")
	DB.Close()
}

// Update writes a new value for a specific key in a bucket
func Update(bucket string, key string, value interface{}) error {
	return DB.Update(func(tx *bolt.Tx) error {
		userBucket := tx.Bucket([]byte(bucket))

		valuebuf, err := json.Marshal(value)
		if err != nil {
			return err
		}

		err = userBucket.Put([]byte(key), valuebuf)
		if err != nil {
			return err
		}

		return nil
	})
}

// Get retuns the value of a specific key, from a specific bucket
func Get(bucket string, key string) ([]byte, error) {
	var v []byte
	err := DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		v = b.Get([]byte(key))
		if v == nil {
			return errors.New("Can't find key " + key + " in bucket " + bucket)
		}

		return nil
	})
	if err != nil {
		return []byte{}, err
	}
	return v, nil
}

// Initialize creates the required database tables
func Initialize() {
	log.Info("Setting up database")
	err := DB.Update(func(tx *bolt.Tx) error {

		buckets := [...]string{"installer", "app", "user"}

		for _, bname := range buckets {
			_, err := tx.CreateBucketIfNotExists([]byte(bname))
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
}
