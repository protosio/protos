package core

import (
	"net"

	"github.com/attic-labs/noms/go/datas"
)

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
	SyncServer(address net.IP) (func() error, error)
	Close() error
}
