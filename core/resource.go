package core

type RType string

type ResourceManager interface {
	GetType(string) (RType, error)
}

type Resource interface {
	Save()
	Delete()
	UpdateValue([]byte) error
	SetStatus(string) error
}
