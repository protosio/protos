package core

type ProviderManager interface {
	Register(App, RType) error
	Deregister(App, RType) error
	Get(App) (Provider, error)
}

type Provider interface {
	GetResources() map[string]Resource
	GetResource(resourceID string) Resource
	TypeName() string
}
