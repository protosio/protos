package ipc

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"time"

	hostagentipc "github.com/protosio/protos/internal/hostagent/ipc"
	hostagentpb "github.com/protosio/protos/internal/hostagent/proto"
	networkmodule "github.com/protosio/protos/internal/network/module"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const defaultRPCTimeout = 30 * time.Second

type Module struct {
	socket string
	conn   *grpc.ClientConn
	client hostagentpb.HostAgentClient
}

var _ networkmodule.Module = (*Module)(nil)
var _ networkmodule.NamespacedInterfaceModule = (*Module)(nil)

func New() (*Module, error) {
	socket := hostagentipc.SocketPath()
	conn, err := grpc.NewClient("unix://"+socket, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("host agent ipc network module: %w", err)
	}

	return &Module{
		socket: socket,
		conn:   conn,
		client: hostagentpb.NewHostAgentClient(conn),
	}, nil
}

func (m *Module) Name() string {
	return "ipc"
}

func (m *Module) Up(config networkmodule.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRPCTimeout)
	defer cancel()

	resp, err := m.client.Apply(ctx, &hostagentpb.ApplyRequest{
		DesiredState: &hostagentpb.HostDesiredState{
			Network: &hostagentpb.NetworkDesiredState{
				DesiredState: "up",
				Config:       configToProto(config),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("host agent network up via %s: %w", m.socket, err)
	}
	return networkResponseError(m.socket, "up", resp.GetNetwork())
}

func (m *Module) Down() error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRPCTimeout)
	defer cancel()

	resp, err := m.client.Apply(ctx, &hostagentpb.ApplyRequest{
		DesiredState: &hostagentpb.HostDesiredState{
			Network: &hostagentpb.NetworkDesiredState{DesiredState: "down"},
		},
	})
	if err != nil {
		return fmt.Errorf("host agent network down via %s: %w", m.socket, err)
	}
	return networkResponseError(m.socket, "down", resp.GetNetwork())
}

func (m *Module) ConfigurePeers(config networkmodule.Config, peers networkmodule.Peers) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRPCTimeout)
	defer cancel()

	resp, err := m.client.Apply(ctx, &hostagentpb.ApplyRequest{
		DesiredState: &hostagentpb.HostDesiredState{
			Network: &hostagentpb.NetworkDesiredState{
				DesiredState:   "configured",
				Config:         configToProto(config),
				Instances:      instancesToProto(peers.Instances),
				Devices:        devicesToProto(peers.Devices),
				ExitRoutes:     exitRoutesToProto(peers.ExitRoutes),
				ReconcilePeers: true,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("host agent network configure peers via %s: %w", m.socket, err)
	}
	return networkResponseError(m.socket, "configure peers", resp.GetNetwork())
}

func (m *Module) State() (networkmodule.State, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRPCTimeout)
	defer cancel()

	resp, err := m.client.Status(ctx, &hostagentpb.StatusRequest{Network: true})
	if err != nil {
		return networkmodule.State{}, fmt.Errorf("host agent network state via %s: %w", m.socket, err)
	}
	observed := resp.GetNetwork()
	if observed == nil {
		return networkmodule.State{}, fmt.Errorf("host agent network state via %s returned no network state", m.socket)
	}
	if observed.GetMessage() != "" {
		return networkmodule.State{}, fmt.Errorf("host agent network state via %s failed: %s", m.socket, observed.GetMessage())
	}
	return networkStateFromProto(observed.GetState()), nil
}

func (m *Module) CreateNamespacedInterface(config networkmodule.Config, netNSPath string, ip net.IP) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRPCTimeout)
	defer cancel()

	resp, err := m.client.Apply(ctx, &hostagentpb.ApplyRequest{
		DesiredState: &hostagentpb.HostDesiredState{
			Network: &hostagentpb.NetworkDesiredState{
				DesiredState: "configured",
				Config:       configToProto(config),
				NamespacedInterfaces: []*hostagentpb.NamespacedInterface{
					{
						NetnsPath: netNSPath,
						Ip:        ip.String(),
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("host agent network create namespaced interface via %s: %w", m.socket, err)
	}
	return networkResponseError(m.socket, "create namespaced interface", resp.GetNetwork())
}

func networkStateFromProto(state *hostagentpb.NetworkState) networkmodule.State {
	if state == nil {
		return networkmodule.State{}
	}
	out := networkmodule.State{
		Module:        state.GetModule(),
		Up:            state.GetUp(),
		InterfaceName: state.GetInterfaceName(),
		Messages:      append([]string(nil), state.GetMessages()...),
	}
	for _, item := range state.GetInterfaces() {
		out.Interfaces = append(out.Interfaces, networkmodule.InterfaceState{
			Name:       item.GetName(),
			Type:       item.GetType(),
			Index:      int(item.GetIndex()),
			MTU:        int(item.GetMtu()),
			Up:         item.GetUp(),
			Master:     item.GetMaster(),
			MacAddress: item.GetMacAddress(),
			Kind:       item.GetKind(),
		})
	}
	for _, item := range state.GetAddresses() {
		out.Addresses = append(out.Addresses, networkmodule.AddressState{
			InterfaceName: item.GetInterfaceName(),
			CIDR:          item.GetCidr(),
			Scope:         item.GetScope(),
		})
	}
	for _, item := range state.GetRoutes() {
		out.Routes = append(out.Routes, networkmodule.RouteState{
			InterfaceName: item.GetInterfaceName(),
			Destination:   item.GetDestination(),
			Gateway:       item.GetGateway(),
			Source:        item.GetSource(),
			Family:        item.GetFamily(),
			Table:         item.GetTable(),
			Protocol:      item.GetProtocol(),
			Scope:         item.GetScope(),
			Priority:      item.GetPriority(),
			Kind:          item.GetKind(),
		})
	}
	for _, item := range state.GetWireguardPeers() {
		out.WireGuardPeers = append(out.WireGuardPeers, networkmodule.WireGuardPeerState{
			PublicKey:       item.GetPublicKey(),
			Endpoint:        item.GetEndpoint(),
			AllowedIPs:      append([]string(nil), item.GetAllowedIps()...),
			LatestHandshake: item.GetLatestHandshake(),
			RxBytes:         item.GetRxBytes(),
			TxBytes:         item.GetTxBytes(),
		})
	}
	for _, table := range state.GetFirewallTables() {
		tableState := networkmodule.FirewallTableState{
			Family: table.GetFamily(),
			Name:   table.GetName(),
		}
		for _, chain := range table.GetChains() {
			chainState := networkmodule.FirewallChainState{
				Name:     chain.GetName(),
				Type:     chain.GetType(),
				Hook:     chain.GetHook(),
				Priority: chain.GetPriority(),
			}
			for _, rule := range chain.GetRules() {
				chainState.Rules = append(chainState.Rules, networkmodule.FirewallRuleState{
					Expressions: append([]string(nil), rule.GetExpressions()...),
					Packets:     rule.GetPackets(),
					Bytes:       rule.GetBytes(),
				})
			}
			tableState.Chains = append(tableState.Chains, chainState)
		}
		out.FirewallTables = append(out.FirewallTables, tableState)
	}
	for _, item := range state.GetDns() {
		out.DNS = append(out.DNS, networkmodule.DNSState{
			Scope:   item.GetScope(),
			Domain:  item.GetDomain(),
			Servers: append([]string(nil), item.GetServers()...),
			Port:    int(item.GetPort()),
			Active:  item.GetActive(),
			Source:  item.GetSource(),
		})
	}
	return out
}

func (m *Module) Close() error {
	if m.conn == nil {
		return nil
	}
	return m.conn.Close()
}

func configToProto(config networkmodule.Config) *hostagentpb.NetworkConfig {
	return &hostagentpb.NetworkConfig{
		Ipv6Address:         config.IPv6Address.String(),
		Ipv4Address:         addrString(config.IPv4Address),
		LocalPeerId:         config.LocalPeerID,
		WireguardPrivateKey: config.WireGuardPrivateKey,
		Domain:              config.Domain,
	}
}

func instancesToProto(instances []networkmodule.InstancePeer) []*hostagentpb.InstancePeer {
	out := make([]*hostagentpb.InstancePeer, 0, len(instances))
	for _, instance := range instances {
		out = append(out, &hostagentpb.InstancePeer{
			Id:          instance.ID,
			Name:        instance.Name,
			PublicKey:   instance.PublicKey,
			PublicIp:    instance.PublicIP,
			Ipv4Address: addrString(instance.IPv4Address),
			Routes:      instanceRoutesToProto(instance.Routes),
		})
	}
	return out
}

func instanceRoutesToProto(routes []netip.Addr) []string {
	out := make([]string, 0, len(routes))
	for _, route := range routes {
		out = append(out, route.String())
	}
	return out
}

func devicesToProto(devices []networkmodule.DevicePeer) []*hostagentpb.DevicePeer {
	out := make([]*hostagentpb.DevicePeer, 0, len(devices))
	for _, device := range devices {
		out = append(out, &hostagentpb.DevicePeer{
			Id:          device.ID,
			Name:        device.Name,
			PublicKey:   device.PublicKey,
			Ipv4Address: addrString(device.IPv4Address),
		})
	}
	return out
}

func addrString(addr netip.Addr) string {
	if !addr.IsValid() {
		return ""
	}
	return addr.String()
}

func exitRoutesToProto(routes []networkmodule.ExitRoute) []*hostagentpb.ExitRoute {
	out := make([]*hostagentpb.ExitRoute, 0, len(routes))
	for _, route := range routes {
		out = append(out, &hostagentpb.ExitRoute{
			Id:         route.ID,
			DeviceId:   route.DeviceID,
			InstanceId: route.InstanceID,
			Cidrs:      route.CIDRs,
		})
	}
	return out
}

func networkResponseError(socket string, operation string, observed *hostagentpb.NetworkObservedState) error {
	if observed == nil {
		return fmt.Errorf("host agent network %s via %s returned no network state", operation, socket)
	}
	if observed.GetMessage() != "" {
		return fmt.Errorf("host agent network %s via %s failed: %s", operation, socket, observed.GetMessage())
	}
	return nil
}
