package daemon

import (
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
	lastConfig      networkmodule.Config
	lastPeers       networkmodule.Peers
}

func (m *fakeNetworkModule) Name() string { return "fake" }

func (m *fakeNetworkModule) Up(config networkmodule.Config) error {
	m.upCalled++
	m.lastConfig = config
	return nil
}

func (m *fakeNetworkModule) Down() error {
	m.downCalled++
	return nil
}

func (m *fakeNetworkModule) ConfigurePeers(config networkmodule.Config, peers networkmodule.Peers) error {
	m.configureCalled++
	m.lastConfig = config
	m.lastPeers = peers
	return nil
}

func (m *fakeNetworkModule) Close() error { return nil }

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
