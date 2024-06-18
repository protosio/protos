package network

import (
	"fmt"
	"net"
	"net/netip"

	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/util"
	"github.com/protosio/protos/internal/wireguard"
)

// var wgPort int = 10999
var log = util.GetLogger("network")

func NewManager() (*Manager, error) {
	linkManager, err := wireguard.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize network: %w", err)
	}

	dnsManager, err := NewDNSManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize network: %w", err)
	}

	return &Manager{linkManager: linkManager, dnsManager: dnsManager}, nil
}

type Manager struct {
	key         *pcrypto.Key
	domain      string
	linkManager wireguard.Manager
	dnsManager  *DNSManager
}

func (m *Manager) Init(key *pcrypto.Key, domain string) error {
	m.key = key
	m.domain = domain
	err := m.Up()
	if err != nil {
		return err
	}
	return nil
}

func createIPv6Net(addr netip.Addr) *net.IPNet {
	if !addr.Is6() {
		return nil
	}

	ip := net.IP(addr.AsSlice())
	mask := net.CIDRMask(128, 128)
	return &net.IPNet{
		IP:   ip,
		Mask: mask,
	}
}
