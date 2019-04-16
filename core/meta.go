package core

// Meta holds information about the protos instance
type Meta interface {
	InitCheck()
	GetPublicIP() string
	GetDomain() string
}
