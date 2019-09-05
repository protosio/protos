package core

// RType is a string wrapper used for typechecking the resource types
type RType string

const (
	// Certificate represents a TLS/SSL certificate
	Certificate = RType("certificate")
	// DNS represents a DNS record
	DNS = RType("dns")
	// Mail is not used yet
	Mail = RType("mail")
)

type RStatus string

const (
	// Requested status is set at creation time and indicates that a resource provider should create this resource
	Requested = RStatus("requested")
	// Created status is the final final state of a resource, ready to be used by an application
	Created = RStatus("created")
	// Unknown status is for error or uknown states
	Unknown = RStatus("unknown")
)

// ResourceManager manages the list of resources
type ResourceManager interface {
	// Create(rtype RType, value Type, appID string) (Resource, error)
	Get(id string) (Resource, error)
	Delete(id string) error
	GetType(name string) (RType, Type, error)
	GetAll(sanitize bool) map[string]Resource
	Select(func(Resource) bool) map[string]Resource
	GetStatus(string) (RStatus, error)
}

// ResourceCreator allows for the creation of all the supported Resource types
type ResourceCreator interface {
	CreateDNS(appID string, name string, rtype string, value string, ttl int) (Resource, error)
	CreateCert(appID string, domains []string) (Resource, error)
	CreateFromJSON(rscJSON []byte, appID string) (Resource, error)
}

// Resource represents a Protos resource
type Resource interface {
	Save()
	GetID() string
	GetType() RType
	GetValue() Type
	UpdateValue(Type)
	SetStatus(RStatus)
	GetAppID() string
	Sanitize() Resource
}

// Type is an interface that satisfies all the resource types
type Type interface {
	Update(Type)
	Sanitize() Type
}
