package module

import (
	"net"
	"net/netip"
)

type Config struct {
	IPv6Address         netip.Addr
	WireGuardPrivateKey string
	Domain              string
}

type Peers struct {
	Instances []InstancePeer
	Devices   []DevicePeer
}

type InstancePeer struct {
	ID        string
	Name      string
	PublicKey string
	PublicIP  string
	Routes    []netip.Addr
}

type DevicePeer struct {
	Name      string
	PublicKey string
}

type Module interface {
	Name() string
	Up(Config) error
	Down() error
	ConfigurePeers(Config, Peers) error
	Close() error
}

type NamespacedInterfaceModule interface {
	CreateNamespacedInterface(Config, string, net.IP) error
}
