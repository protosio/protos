package daemon

import (
	"errors"
	"net"
	"net/netip"
	"testing"

	hostagentpb "github.com/protosio/protos/internal/hostagent/proto"
	networkmodule "github.com/protosio/protos/internal/network/module"
)

type fakeNetworkModule struct {
	upCalled        int
	configureCalled int
	downCalled      int
	closeCalled     int
	lastConfig      networkmodule.Config
	lastPeers       networkmodule.Peers
	downErr         error
	closeErr        error
}

func (m *fakeNetworkModule) Name() string { return "fake" }

func (m *fakeNetworkModule) Up(config networkmodule.Config) error {
	m.upCalled++
	m.lastConfig = config
	return nil
}

func (m *fakeNetworkModule) Down() error {
	m.downCalled++
	return m.downErr
}

func (m *fakeNetworkModule) ConfigurePeers(config networkmodule.Config, peers networkmodule.Peers) error {
	m.configureCalled++
	m.lastConfig = config
	m.lastPeers = peers
	return nil
}

func (m *fakeNetworkModule) State() (networkmodule.State, error) {
	return networkmodule.State{Module: m.Name(), Up: m.upCalled > m.downCalled}, nil
}

func (m *fakeNetworkModule) Close() error {
	m.closeCalled++
	return m.closeErr
}

func (m *fakeNetworkModule) CreateNamespacedInterface(networkmodule.Config, string, net.IP) error {
	return nil
}

func TestApplyNetworkConfiguredReconcilesEmptyPeerSet(t *testing.T) {
	networkModule := &fakeNetworkModule{}
	server := NewServer(networkModule)
	config := &hostagentpb.NetworkConfig{
		Ipv6Address:         "200:db8::1",
		WireguardPrivateKey: "private",
		Domain:              "protos.internal",
	}

	resp := server.applyNetwork(&hostagentpb.NetworkDesiredState{
		DesiredState:   "configured",
		Config:         config,
		ReconcilePeers: true,
		Instances: []*hostagentpb.InstancePeer{
			{Id: "vm-1", Name: "vm-1", PublicKey: "pub", PublicIp: "192.0.2.10", Routes: []string{"200:db8::20"}},
		},
	})
	if resp.GetMessage() != "" {
		t.Fatalf("applyNetwork returned message %q", resp.GetMessage())
	}
	if networkModule.configureCalled != 1 || len(networkModule.lastPeers.Instances) != 1 {
		t.Fatalf("first reconcile did not apply peer set: called=%d peers=%#v", networkModule.configureCalled, networkModule.lastPeers)
	}
	if got := networkModule.lastPeers.Instances[0].Routes; len(got) != 1 || got[0] != netip.MustParseAddr("200:db8::20") {
		t.Fatalf("instance routes did not round-trip: %#v", got)
	}

	resp = server.applyNetwork(&hostagentpb.NetworkDesiredState{
		DesiredState:   "configured",
		Config:         config,
		ReconcilePeers: true,
	})
	if resp.GetMessage() != "" {
		t.Fatalf("empty applyNetwork returned message %q", resp.GetMessage())
	}
	if networkModule.configureCalled != 2 {
		t.Fatalf("empty peer reconcile was not applied, configure calls=%d", networkModule.configureCalled)
	}
	if len(networkModule.lastPeers.Instances) != 0 || len(networkModule.lastPeers.Devices) != 0 {
		t.Fatalf("empty peer reconcile did not clear peers: %#v", networkModule.lastPeers)
	}
	if networkModule.lastConfig.IPv6Address != netip.MustParseAddr("200:db8::1") {
		t.Fatalf("config did not round-trip: %#v", networkModule.lastConfig)
	}
}

func TestServerCloseTearsDownNetwork(t *testing.T) {
	networkModule := &fakeNetworkModule{}
	server := NewServer(networkModule)

	resp := server.applyNetwork(&hostagentpb.NetworkDesiredState{
		DesiredState: "up",
		Config: &hostagentpb.NetworkConfig{
			Ipv6Address:         "200:db8::1",
			WireguardPrivateKey: "private",
		},
	})
	if resp.GetMessage() != "" {
		t.Fatalf("applyNetwork returned message %q", resp.GetMessage())
	}

	if err := server.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if networkModule.downCalled != 1 {
		t.Fatalf("Down called %d times, want 1", networkModule.downCalled)
	}
	if networkModule.closeCalled != 1 {
		t.Fatalf("Close called %d times, want 1", networkModule.closeCalled)
	}

	if err := server.Close(); err != nil {
		t.Fatalf("second Close returned error: %v", err)
	}
	if networkModule.downCalled != 1 || networkModule.closeCalled != 1 {
		t.Fatalf("Close was not idempotent: down=%d close=%d", networkModule.downCalled, networkModule.closeCalled)
	}
}

func TestServerCloseStillClosesModuleWhenDownFails(t *testing.T) {
	networkModule := &fakeNetworkModule{downErr: errors.New("down failed")}
	server := NewServer(networkModule)

	resp := server.applyNetwork(&hostagentpb.NetworkDesiredState{
		DesiredState: "up",
		Config: &hostagentpb.NetworkConfig{
			Ipv6Address:         "200:db8::1",
			WireguardPrivateKey: "private",
		},
	})
	if resp.GetMessage() != "" {
		t.Fatalf("applyNetwork returned message %q", resp.GetMessage())
	}

	if err := server.Close(); err == nil {
		t.Fatalf("Close returned nil error")
	}
	if networkModule.downCalled != 1 {
		t.Fatalf("Down called %d times, want 1", networkModule.downCalled)
	}
	if networkModule.closeCalled != 1 {
		t.Fatalf("Close called %d times, want 1", networkModule.closeCalled)
	}
}
