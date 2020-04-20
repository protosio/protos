package core

import (
	"net"

	"github.com/protosio/protos/internal/util"
)

// Meta holds information about the protos instance
type Meta interface {
	// network related methods
	GetPublicIP() string
	SetDomain(string)
	GetDomain() string
	SetNetwork(net.IPNet)
	GetNetwork() net.IPNet
	SetInternalIP(net.IP)
	GetInternalIP() net.IP

	GetTLSCertificate() Resource
	SetAdminUser(string)
	CreateProtosResources() (map[string]Resource, error)
	GetProtosResources() map[string]Resource
	CleanProtosResources() error
	GetDashboardDomain() string
	GetService() util.Service
	GetAdminUser() string
	GetVersion() string
	InitMode() bool
}
