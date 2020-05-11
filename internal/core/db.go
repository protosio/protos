package core

// DB is used to interact with the database
type DB interface {
	Exists() bool
	Save(interface{}) error
	One(fieldName string, value interface{}, to interface{}) error
	All(to interface{}) error
	Remove(data interface{}) error
	Register(interface{})
	Open()
	Close()
}

// DBCLI represents a DB client instance, used to interract with the database
type DBCLI interface {
	SaveStruct(dataset string, data interface{}) error
	GetStruct(dataset string, to interface{}) error
	GetSet(dataset string, to interface{}) error
	InsertInSet(dataset string, data interface{}) error
	RemoveFromSet(dataset string, data interface{}) error
	Close() error
}
