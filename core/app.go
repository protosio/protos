package core

type AppManager interface {
	Get(string) App
}

type App interface {
	Start() error
	GetID() string
}
