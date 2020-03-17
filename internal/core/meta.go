package core

import "github.com/protosio/protos/internal/util"

// Meta holds information about the protos instance
type Meta interface {
	InitCheck()
	GetPublicIP() string
	GetDomain() string
	GetTLSCertificate() Resource
	SetDomain(string)
	SetAdminUser(string)
	CreateProtosResources() (map[string]Resource, error)
	GetProtosResources() map[string]Resource
	CleanProtosResources() error
	GetDashboardDomain() string
	GetService() util.Service
	GetAdminUser() string
	GetVersion() string
}
