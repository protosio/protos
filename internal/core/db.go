package core

// DB represents a DB client instance, used to interract with the database
type DB interface {
	SaveStruct(dataset string, data interface{}) error
	GetStruct(dataset string, to interface{}) error
	GetSet(dataset string, to interface{}) error
	InsertInSet(dataset string, data interface{}) error
	RemoveFromSet(dataset string, data interface{}) error
	GetMap(dataset string, to interface{}) error
	InsertInMap(dataset string, id string, data interface{}) error
	RemoveFromMap(dataset string, id string) error
	Close() error
}
