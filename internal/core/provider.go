package core

type ProviderManager interface {
	Register(App, ResourceType) error
	Deregister(App, ResourceType) error
	Get(App) (Provider, error)
}

type Provider interface {
	GetResources() map[string]Resource
	GetResource(resourceID string) Resource
	TypeName() string
}
