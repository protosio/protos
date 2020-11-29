package core

// // ResourceType is a string wrapper used for typechecking the resource types
// type ResourceType string

// const (
// 	// Certificate represents a TLS/SSL certificate
// 	Certificate = ResourceType("certificate")
// 	// DNS represents a DNS record
// 	DNS = ResourceType("dns")
// 	// Mail is not used yet
// 	Mail = ResourceType("mail")
// )

// // ResourceStatus is a string wrapper used for typechecking the resource status
// type ResourceStatus string

// const (
// 	// Requested status is set at creation time and indicates that a resource provider should create this resource
// 	Requested = ResourceStatus("requested")
// 	// Created status is the final final state of a resource, ready to be used by an application
// 	Created = ResourceStatus("created")
// 	// Unknown status is for error or uknown states
// 	Unknown = ResourceStatus("unknown")
// )

// // ErrResourceExists is returned when trying to create a resource that already exists
// type ErrResourceExists struct{}

// func (e ErrResourceExists) Error() string {
// 	return "Resource exists"
// }

// // ResourceManager manages the list of resources
// type ResourceManager interface {
// 	Get(id string) (Resource, error)
// 	Delete(id string) error
// 	GetType(name string) (ResourceType, ResourceValue, error)
// 	GetAll(sanitize bool) map[string]Resource
// 	Select(func(Resource) bool) map[string]Resource
// 	StringToStatus(string) (ResourceStatus, error)

// 	// Create resources
// 	CreateDNS(appID string, name string, rtype string, value string, ttl int) (Resource, error)
// 	CreateCert(appID string, domains []string) (Resource, error)
// 	CreateFromJSON(rscJSON []byte, appID string) (Resource, error)
// }

// // Resource represents a Protos resource
// type Resource interface {
// 	Save()
// 	GetID() string
// 	GetType() ResourceType
// 	GetValue() ResourceValue
// 	UpdateValue(ResourceValue)
// 	GetStatus() ResourceStatus
// 	SetStatus(ResourceStatus)
// 	GetAppID() string
// 	Sanitize() Resource
// }

// // ResourceValue is an interface that satisfies all the resource values
// type ResourceValue interface {
// 	Update(ResourceValue)
// 	Sanitize() ResourceValue
// }
