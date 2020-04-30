package core

import (
	"net"

	"github.com/protosio/protos/internal/util"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
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
	// crypto related methods
	GetKey() wgtypes.Key
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
