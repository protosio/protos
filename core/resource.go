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
	Get(string) (Resource, error)
	Delete(string) error
	GetType(string) (RType, Type, error)
	Select(func(Resource) bool) map[string]Resource
}

type Resource interface {
	Save()
	UpdateValue(Type)
	SetStatus(RStatus)
	GetType() RType
}

// Type is an interface that satisfies all the resource types
type Type interface {
	Update(Type)
	Sanitize() Type
}
