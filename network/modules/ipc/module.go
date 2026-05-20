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
				ReconcilePeers: true,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("host agent network configure peers via %s: %w", m.socket, err)
	}
	return networkResponseError(m.socket, "configure peers", resp.GetNetwork())
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

func (m *Module) Close() error {
	if m.conn == nil {
		return nil
	}
	return m.conn.Close()
}

func configToProto(config networkmodule.Config) *hostagentpb.NetworkConfig {
	return &hostagentpb.NetworkConfig{
		Ipv6Address:         config.IPv6Address.String(),
		WireguardPrivateKey: config.WireGuardPrivateKey,
		Domain:              config.Domain,
	}
}

func instancesToProto(instances []networkmodule.InstancePeer) []*hostagentpb.InstancePeer {
	out := make([]*hostagentpb.InstancePeer, 0, len(instances))
	for _, instance := range instances {
		out = append(out, &hostagentpb.InstancePeer{
			Id:        instance.ID,
			Name:      instance.Name,
			PublicKey: instance.PublicKey,
			PublicIp:  instance.PublicIP,
			Routes:    instanceRoutesToProto(instance.Routes),
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
			Name:      device.Name,
			PublicKey: device.PublicKey,
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
