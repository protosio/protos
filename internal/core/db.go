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
