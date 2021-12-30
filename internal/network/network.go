package network

import (
	"fmt"
	"net"

	"github.com/foxcpp/wirebox/linkmgr"
	"github.com/protosio/protos/internal/util"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// var wgPort int = 10999
var log = util.GetLogger("network")

type Manager struct {
	privateKey  wgtypes.Key
	network     net.IPNet
	gateway     net.IP
	domain      string
	linkManager linkmgr.Manager
}

func (m *Manager) Init(network net.IPNet, gateway net.IP, privateKey wgtypes.Key, domain string) error {
	m.network = network
	m.privateKey = privateKey
	m.domain = domain
	m.gateway = gateway
	err := m.Up()
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) GetInternalIP() net.IP {
	return m.gateway
}

func NewManager() (*Manager, error) {
	linkManager, err := linkmgr.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize network: %w", err)
	}
	return &Manager{linkManager: linkManager}, nil
}
