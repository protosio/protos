package core

type AppManager interface {
	Read(string) (App, error)
	GetAllPublic() map[string]App
}

type App interface {
	Start() error
	GetID() string
	GetName() string
}
