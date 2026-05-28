package module

import (
	"net"
	"net/netip"
)

type Config struct {
	IPv6Address         netip.Addr
	IPv4Address         netip.Addr
	LocalPeerID         string
	WireGuardPrivateKey string
	Domain              string
}

type Peers struct {
	Instances  []InstancePeer
	Devices    []DevicePeer
	ExitRoutes []ExitRoute
}

type InstancePeer struct {
	ID          string
	Name        string
	PublicKey   string
	PublicIP    string
	IPv4Address netip.Addr
	Routes      []netip.Addr
}

type DevicePeer struct {
	ID          string
	Name        string
	PublicKey   string
	IPv4Address netip.Addr
}

type ExitRoute struct {
	ID         string
	DeviceID   string
	InstanceID string
	CIDRs      []string
}

type State struct {
	Module         string
	Up             bool
	InterfaceName  string
	Interfaces     []InterfaceState
	Addresses      []AddressState
	Routes         []RouteState
	WireGuardPeers []WireGuardPeerState
	FirewallTables []FirewallTableState
	DNS            []DNSState
	Messages       []string
}

type InterfaceState struct {
	Name       string
	Type       string
	Index      int
	MTU        int
	Up         bool
	Master     string
	MacAddress string
	Kind       string
}

type AddressState struct {
	InterfaceName string
	CIDR          string
	Scope         string
}

type RouteState struct {
	InterfaceName string
	Destination   string
	Gateway       string
	Source        string
	Family        string
	Table         string
	Protocol      string
	Scope         string
	Priority      string
	Kind          string
}

type WireGuardPeerState struct {
	PublicKey       string
	Endpoint        string
	AllowedIPs      []string
	LatestHandshake string
	RxBytes         uint64
	TxBytes         uint64
}

type FirewallTableState struct {
	Family string
	Name   string
	Chains []FirewallChainState
}

type FirewallChainState struct {
	Name     string
	Type     string
	Hook     string
	Priority string
	Rules    []FirewallRuleState
}

type FirewallRuleState struct {
	Expressions []string
	Packets     uint64
	Bytes       uint64
}

type DNSState struct {
	Scope   string
	Domain  string
	Servers []string
	Port    int
	Active  bool
	Source  string
}

type Module interface {
	Name() string
	Up(Config) error
	Down() error
	ConfigurePeers(Config, Peers) error
	State() (State, error)
	Close() error
}

type NamespacedInterfaceModule interface {
	CreateNamespacedInterface(Config, string, net.IP) error
}
