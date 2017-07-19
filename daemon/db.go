package daemon

import (
	"path"

	"github.com/boltdb/bolt"
)

func openDatabase() error {

	// open the database
	var err error
	var dbpath string
	dbpath = path.Join(Gconfig.WorkDir, "protos.db")
	log.Info("Opening database [", dbpath, "]")
	Gconfig.Db, err = bolt.Open(dbpath, 0600, nil)
	return err

}
