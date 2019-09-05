package core

type DB interface {
	Save(interface{}) error
	One(fieldName string, value interface{}, to interface{}) error
	All(to interface{}) error
	Remove(data interface{}) error
	Register(interface{})
	Open()
	Close()
}
