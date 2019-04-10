package core

type AppManager interface {
	Read(string) (App, error)
}

type App interface {
	Start() error
	GetID() string
	GetName() string
}
