package core

// DB represents a DB client instance, used to interract with the database
type DB interface {
	SaveStruct(dataset string, data interface{}) error
	GetStruct(dataset string, to interface{}) error
	InitSet(dataset string) error
	GetSet(dataset string, to interface{}) error
	InsertInSet(dataset string, data interface{}) error
	RemoveFromSet(dataset string, data interface{}) error
	InitMap(dataset string) error
	GetMap(dataset string, to interface{}) error
	InsertInMap(dataset string, id string, data interface{}) error
	RemoveFromMap(dataset string, id string) error
	SyncTo(dataset string, ip string) error
	SyncServer() error
	Close() error
}
