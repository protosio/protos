package ipc

import (
	"errors"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	hostagentdaemon "github.com/protosio/protos/internal/hostagent/daemon"
	hostagentipc "github.com/protosio/protos/internal/hostagent/ipc"
	hostagentpb "github.com/protosio/protos/internal/hostagent/proto"
	networkmodule "github.com/protosio/protos/internal/network/module"
	"google.golang.org/grpc"
)

type fakeNetworkModule struct {
	upConfig        networkmodule.Config
	peerConfig      networkmodule.Config
	peers           networkmodule.Peers
	namespacePath   string
	namespaceIP     net.IP
	upCalled        bool
	peersCalled     bool
	namespaceCalled bool
	downCalled      bool
	upErr           error
}

func (m *fakeNetworkModule) Name() string { return "fake" }

func (m *fakeNetworkModule) Up(config networkmodule.Config) error {
	m.upConfig = config
	m.upCalled = true
	return m.upErr
}

func (m *fakeNetworkModule) Down() error {
	m.downCalled = true
	return nil
}

func (m *fakeNetworkModule) State() (networkmodule.State, error) {
	return networkmodule.State{Module: m.Name(), Up: m.upCalled && !m.downCalled}, nil
}

func TestModuleReturnsHostAgentNetworkError(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "protos-hostagent-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(dir)
	socket := filepath.Join(dir, "hostagent.sock")
	t.Setenv(hostagentipc.SocketEnv, socket)

	fake := &fakeNetworkModule{upErr: errors.New("cannot configure host network")}
	grpcServer := grpc.NewServer()
	hostagentpb.RegisterHostAgentServer(grpcServer, hostagentdaemon.NewServer(fake))

	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer os.Remove(socket)
	defer grpcServer.Stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- grpcServer.Serve(listener)
	}()

	module, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer module.Close()

	config := networkmodule.Config{
		IPv6Address:         netip.MustParseAddr("fd7a:115c:a1e0::1"),
		WireGuardPrivateKey: "private",
		Domain:              "protos.internal",
	}
	err = module.Up(config)
	if err == nil {
		t.Fatal("expected host agent network error")
	}
	if !strings.Contains(err.Error(), "cannot configure host network") {
		t.Fatalf("unexpected error: %v", err)
	}

	grpcServer.Stop()
	<-errCh
}

func (m *fakeNetworkModule) ConfigurePeers(config networkmodule.Config, peers networkmodule.Peers) error {
	m.peerConfig = config
	m.peers = peers
	m.peersCalled = true
	return nil
}

func (m *fakeNetworkModule) CreateNamespacedInterface(config networkmodule.Config, netNSPath string, ip net.IP) error {
	m.namespacePath = netNSPath
	m.namespaceIP = ip
	m.namespaceCalled = true
	return nil
}

func (m *fakeNetworkModule) Close() error { return nil }

func TestModuleCallsHostAgent(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "protos-hostagent-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(dir)
	socket := filepath.Join(dir, "hostagent.sock")
	t.Setenv(hostagentipc.SocketEnv, socket)

	fake := &fakeNetworkModule{}
	grpcServer := grpc.NewServer()
	hostagentpb.RegisterHostAgentServer(grpcServer, hostagentdaemon.NewServer(fake))

	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer os.Remove(socket)
	defer grpcServer.Stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- grpcServer.Serve(listener)
	}()

	module, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer module.Close()

	config := networkmodule.Config{
		IPv6Address:         netip.MustParseAddr("fd7a:115c:a1e0::1"),
		WireGuardPrivateKey: "private",
		Domain:              "protos.internal",
	}
	if err := module.Up(config); err != nil {
		t.Fatalf("Up: %v", err)
	}

	peers := networkmodule.Peers{
		Instances: []networkmodule.InstancePeer{{
			ID:        "i1",
			Name:      "instance",
			PublicKey: "pub",
			PublicIP:  "127.0.0.1",
			Routes:    []netip.Addr{netip.MustParseAddr("200:db8::20")},
		}},
		Devices: []networkmodule.DevicePeer{{Name: "device", PublicKey: "devpub"}},
	}
	if err := module.ConfigurePeers(config, peers); err != nil {
		t.Fatalf("ConfigurePeers: %v", err)
	}

	if err := module.CreateNamespacedInterface(config, "/proc/1/ns/net", net.ParseIP("10.0.0.2")); err != nil {
		t.Fatalf("CreateNamespacedInterface: %v", err)
	}

	if err := module.Down(); err != nil {
		t.Fatalf("Down: %v", err)
	}

	grpcServer.Stop()
	<-errCh

	if !fake.upCalled || !fake.peersCalled || !fake.namespaceCalled || !fake.downCalled {
		t.Fatalf("not all daemon methods were called: %#v", fake)
	}
	if fake.upConfig.Domain != config.Domain || fake.peerConfig.Domain != config.Domain {
		t.Fatalf("config did not round-trip: up=%#v peers=%#v", fake.upConfig, fake.peerConfig)
	}
	if len(fake.peers.Instances) != 1 || fake.peers.Instances[0].ID != "i1" {
		t.Fatalf("instances did not round-trip: %#v", fake.peers.Instances)
	}
	if got := fake.peers.Instances[0].Routes; len(got) != 1 || got[0] != netip.MustParseAddr("200:db8::20") {
		t.Fatalf("instance routes did not round-trip: %#v", got)
	}
	if fake.namespacePath != "/proc/1/ns/net" || !fake.namespaceIP.Equal(net.ParseIP("10.0.0.2")) {
		t.Fatalf("namespace request did not round-trip: path=%q ip=%s", fake.namespacePath, fake.namespaceIP)
	}
}
