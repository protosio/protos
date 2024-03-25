package meta

import (
	"context"
	"net"

	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/pcrypto"

	"github.com/protosio/protos/internal/util"
)

const (
	metaDS      = "meta"
	metaKeyFile = "protos_key.pub"
)

var log = util.GetLogger("meta")
var gconfig = config.Get()

// Meta contains information about the Protos instance
type Meta struct {
	db               *db.DB
	version          string
	networkSetSignal chan net.IP

	// Public members
	ID           string
	InstanceName string
	PublicIP     net.IP
	Resources    []string
	Network      net.IPNet
	InternalIP   net.IP
}

// Setup reads the domain and other information on first run and save this information to the database
func Setup(db *db.DB, keymngr *pcrypto.Manager, version string) *Meta {
	if db == nil || keymngr == nil {
		log.Panic("Failed to setup meta package: none of the inputs can be nil")
	}

	metaRoot := Meta{}
	log.Debug("Reading instance information from database")

	metaRoot.db = db
	metaRoot.version = version
	metaRoot.networkSetSignal = make(chan net.IP, 1)
	return &metaRoot
}

// SetInstanceName sets the name of the instance
func (m *Meta) SetInstanceName(name string) {
	log.Debugf("Setting instance name to '%s'", name)
	m.InstanceName = name
}

// GetInstanceName retrieves the name of the instance
func (m *Meta) GetInstanceName() string {
	return m.InstanceName
}

// SetNetwork sets the instance network
func (m *Meta) SetNetwork(network net.IPNet) net.IP {
	log.Debugf("Setting instance network to '%s'", network.String())
	ip := network.IP.Mask(network.Mask)
	ip[3]++
	m.InternalIP = ip
	m.Network = network
	m.networkSetSignal <- ip
	return ip
}

// GetNetwork gets the instance network
func (m *Meta) GetNetwork() net.IPNet {
	return m.Network
}

// GetInternalIP gets the instance IP
func (m *Meta) GetInternalIP() net.IP {
	return m.InternalIP
}

// GetPublicIP returns the public IP of the Protos instance
func (m *Meta) GetPublicIP() string {
	return m.PublicIP.String()
}

// GetVersion returns current version
func (m *Meta) GetVersion() string {
	return m.version
}

// InitMode returns the status of the init process
func (m *Meta) InitMode() bool {
	if m.InternalIP == nil {
		log.Warnf("Instance info is not set. Running in init mode")
		return true
	}

	return false
}

// WaitForInit returns when both the domain and network has been set
func (m *Meta) WaitForInit(ctx context.Context) (net.IP, net.IPNet) {
	if m.InternalIP != nil {
		return m.InternalIP, m.Network
	}

	var internalIP net.IP

	initialized := make(chan bool)

	go func() {
		log.Debug("Waiting for initialisation to complete")
		internalIP = <-m.networkSetSignal
		initialized <- true
	}()

	select {
	case <-ctx.Done():
		log.Debug("Init did not finish. Canceled by user")
		return internalIP, m.Network
	case <-initialized:
		log.Debug("Meta init finished")
		return internalIP, m.Network
	}

}
