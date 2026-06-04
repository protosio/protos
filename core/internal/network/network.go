package network

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/netip"

	networkmodule "github.com/protosio/protos/internal/network/module"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/provisioners"
	"github.com/protosio/protos/internal/user"
	"github.com/protosio/protos/internal/util"
)

// var wgPort int = 10999
var log = util.GetLogger("network")

func NewManager() (*Manager, error) {
	mod, err := newModule()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize network: %w", err)
	}

	log.Debugf("Using network module %q", mod.Name())
	return &Manager{module: mod}, nil
}

type Manager struct {
	config networkmodule.Config
	module networkmodule.Module
}

type AppRoute struct {
	InstanceID string
	IP         net.IP
}

func (m *Manager) Init(key *pcrypto.Key, domain string) error {
	m.config = networkmodule.Config{
		IPv6Address:         key.IPv6Address(),
		IPv4Address:         TunnelIPv4ForPublicKey(key.PublicString()),
		LocalPeerID:         key.GetID(),
		WireGuardPrivateKey: key.PrivateWG().String(),
		Domain:              domain,
	}
	err := m.module.Up(m.config)
	if err != nil {
		return err
	}
	return nil
}

func (m *Manager) Up() error {
	return m.module.Up(m.config)
}

func (m *Manager) Down() error {
	return m.module.Down()
}

func (m *Manager) ConfigurePeers(instances []provisioners.InstanceInfo, devices []user.UserDevice, appRoutes []AppRoute, exitRoutes []ExitRoute) error {
	peers := networkmodule.Peers{
		Instances:  make([]networkmodule.InstancePeer, 0, len(instances)),
		Devices:    make([]networkmodule.DevicePeer, 0, len(devices)),
		ExitRoutes: make([]networkmodule.ExitRoute, 0, len(exitRoutes)),
	}

	routesByInstance := map[string][]netip.Addr{}
	for _, route := range appRoutes {
		if route.IP == nil || route.InstanceID == "" {
			continue
		}
		addr, ok := netip.AddrFromSlice(route.IP.To16())
		if !ok || !addr.Is6() {
			continue
		}
		routesByInstance[route.InstanceID] = append(routesByInstance[route.InstanceID], addr)
	}

	for _, instance := range instances {
		peers.Instances = append(peers.Instances, networkmodule.InstancePeer{
			ID:          instance.ID,
			Name:        instance.Name,
			PublicKey:   instance.PublicKey,
			PublicIP:    instance.PublicIP,
			IPv4Address: TunnelIPv4ForPublicKey(instance.PublicKey),
			Routes:      routesByInstance[instance.ID],
		})
	}

	for _, device := range devices {
		peers.Devices = append(peers.Devices, networkmodule.DevicePeer{
			ID:          device.ID,
			Name:        device.Name,
			PublicKey:   device.PublicKey,
			IPv4Address: TunnelIPv4ForPublicKey(device.PublicKey),
		})
	}

	for _, route := range exitRoutes {
		if NormalizeExitRouteStatus(route.DesiredStatus) != ExitRouteStatusActive {
			continue
		}
		peers.ExitRoutes = append(peers.ExitRoutes, networkmodule.ExitRoute{
			ID:         route.ID,
			DeviceID:   route.DeviceID,
			InstanceID: route.InstanceID,
			CIDRs:      route.CIDRs,
		})
	}

	return m.module.ConfigurePeers(m.config, peers)
}

func (m *Manager) State() (networkmodule.State, error) {
	if m == nil || m.module == nil {
		return networkmodule.State{}, fmt.Errorf("network manager is not configured")
	}
	return m.module.State()
}

func (m *Manager) Close() error {
	return m.module.Close()
}

func (m *Manager) CreateNamespacedInterface(netNSpath string, ip net.IP) error {
	namespacedModule, ok := m.module.(networkmodule.NamespacedInterfaceModule)
	if !ok {
		return fmt.Errorf("network module %q does not support namespaced interfaces", m.module.Name())
	}
	return namespacedModule.CreateNamespacedInterface(m.config, netNSpath, ip)
}

func TunnelIPv4ForPublicKey(publicKey string) netip.Addr {
	sum := sha256.Sum256([]byte(publicKey))
	return netip.AddrFrom4([4]byte{
		100,
		64 | (sum[0] & 0x3f),
		sum[1],
		sum[2],
	})
}
