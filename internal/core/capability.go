package core

// CapabilityManager manages capabilities and their validation
type CapabilityManager interface {
	Validate(methodcap Capability, appcap string) bool
	SetMethodCap(method string, cap Capability)
	GetMethodCap(method string) (Capability, error)
	GetByName(name string) (Capability, error)
	GetOrPanic(name string) Capability
}

// Capability represents a capability, which is the main way access is granted in Protos
type Capability interface {
	GetName() string
	GetParent() Capability
}

// CapabilityChecker is an interface that implements methods for checking a capability
type CapabilityChecker interface {
	ValidateCapability(cap Capability) error
}
