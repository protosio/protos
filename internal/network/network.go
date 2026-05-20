package network

import (
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

func (m *Manager) ConfigurePeers(instances []provisioners.InstanceInfo, devices []user.UserDevice, appRoutes []AppRoute) error {
	peers := networkmodule.Peers{
		Instances: make([]networkmodule.InstancePeer, 0, len(instances)),
		Devices:   make([]networkmodule.DevicePeer, 0, len(devices)),
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
			ID:        instance.ID,
			Name:      instance.Name,
			PublicKey: instance.PublicKey,
			PublicIP:  instance.PublicIP,
			Routes:    routesByInstance[instance.ID],
		})
	}

	for _, device := range devices {
		peers.Devices = append(peers.Devices, networkmodule.DevicePeer{
			Name:      device.Name,
			PublicKey: device.PublicKey,
		})
	}

	return m.module.ConfigurePeers(m.config, peers)
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
