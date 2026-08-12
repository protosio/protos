package p2p

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"
	"testing"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
	swarmiontransport "github.com/nustiueudinastea/swarmion/transports"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	p2pproto "github.com/protosio/protos/internal/p2p/proto"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/swarmionlink"
	"github.com/protosio/protos/internal/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

type testMachine struct {
	id         string
	name       string
	publicKey  string
	publicIP   string
	internalIP string
}

func (m testMachine) GetID() string        { return m.id }
func (m testMachine) GetPublicKey() string { return m.publicKey }
func (m testMachine) GetPublicIP() string  { return m.publicIP }
func (m testMachine) GetName() string {
	if m.name != "" {
		return m.name
	}
	return m.id
}
func (m testMachine) GetInternalIP() string {
	return m.internalIP
}

func newTestMachine(t *testing.T, publicIP string) (testMachine, string, string) {
	t.Helper()

	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	libp2pPub, err := crypto.UnmarshalEd25519PublicKey(pub)
	if err != nil {
		t.Fatalf("unmarshal libp2p key: %v", err)
	}
	peerID, err := peer.IDFromPublicKey(libp2pPub)
	if err != nil {
		t.Fatalf("peer id: %v", err)
	}

	key := pcrypto.Key{Pub: pub}
	internalIP := key.IPv6Address().String()
	machine := testMachine{
		id:        "test-peer",
		publicKey: base64.StdEncoding.EncodeToString(pub),
		publicIP:  publicIP,
	}
	return machine, peerID.String(), internalIP
}

func newTestP2P() *P2P {
	return &P2P{
		machines:     util.NewMap[string, Machine](),
		clients:      util.NewMap[string, *Client](),
		peerStates:   util.NewMap[string, PeerState](),
		pendingPeers: util.NewMap[string, bool](),
	}
}

func TestGetPeersKeysByPeerID(t *testing.T) {
	p2p := newTestP2P()
	p2p.machines.Set("peer-id-1", testMachine{id: "machine-id-1", name: "display-name-1"})
	p2p.peerStates.Set("peer-id-1", PeerState{Status: PeerStatusConnected})

	server := &Server{p2p: p2p}
	resp, err := server.GetPeers(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetPeers() error = %v", err)
	}

	if got := resp.GetPeers()["peer-id-1"]; got != string(PeerStatusConnected) {
		t.Fatalf("GetPeers()[peer-id-1] = %q, want %q", got, PeerStatusConnected)
	}
	if _, found := resp.GetPeers()["display-name-1"]; found {
		t.Fatalf("GetPeers() used machine name as key: %v", resp.GetPeers())
	}
}

func TestConfigurePeersIgnoresLocalPeer(t *testing.T) {
	localKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("local key: %v", err)
	}
	p2p, err := NewManager(localKey, nil, testExternalDB{initialized: true}, 0)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	defer p2p.host.Close()

	localMachine := testMachine{
		id:        "local",
		name:      "local-device",
		publicKey: base64.StdEncoding.EncodeToString(localKey.Public()),
		publicIP:  "127.0.0.1",
	}
	remoteMachine, remotePeerID, _ := newTestMachine(t, "203.0.113.10")

	if err := p2p.ConfigurePeers([]Machine{localMachine, remoteMachine}); err != nil {
		t.Fatalf("configure peers: %v", err)
	}

	if _, found := p2p.machines.Get(p2p.PeerID()); found {
		t.Fatalf("local peer %s was tracked as a remote machine", p2p.PeerID())
	}
	if _, found := p2p.peerStates.Get(p2p.PeerID()); found {
		t.Fatalf("local peer %s has remote peer state", p2p.PeerID())
	}
	if _, found := p2p.machines.Get(remotePeerID); !found {
		t.Fatalf("remote peer %s was not tracked", remotePeerID)
	}
}

func TestDisconnectedKnownPeerCountIgnoresExtraPendingClients(t *testing.T) {
	known := map[string]Machine{
		"peer-a": testMachine{id: "peer-a"},
		"peer-b": testMachine{id: "peer-b"},
	}
	connected := map[string]*Client{
		"peer-a":       {},
		"pending-peer": {},
	}
	if got := disconnectedKnownPeerCount(known, connected); got != 1 {
		t.Fatalf("disconnectedKnownPeerCount() = %d, want 1", got)
	}
}

func TestUsablePeerConnectednessAcceptsLimitedConnections(t *testing.T) {
	for _, connectedness := range []network.Connectedness{network.Connected, network.Limited} {
		if !usablePeerConnectedness(connectedness) {
			t.Fatalf("usablePeerConnectedness(%s) = false, want true", connectedness)
		}
	}
	if usablePeerConnectedness(network.NotConnected) {
		t.Fatalf("usablePeerConnectedness(%s) = true, want false", network.NotConnected)
	}
}

func TestDestinationStringsIncludeWireGuardIPv6WithoutPublicIP(t *testing.T) {
	machine, peerID, internalIP := newTestMachine(t, "")
	p2p := newTestP2P()

	got := p2p.destinationStrings(peerID, machine)
	want := fmt.Sprintf(destinationTCPIPv6Template, internalIP, config.Get().P2PPort, peerID)
	if !slices.Contains(got, want) {
		t.Fatalf("destinationStrings() = %v, want %s", got, want)
	}
}

func TestDestinationStringsIncludeMultipleKnownIPs(t *testing.T) {
	machine, peerID, derivedInternalIP := newTestMachine(t, "203.0.113.10")
	machine.internalIP = "fd00::10"
	p2p := newTestP2P()

	got := p2p.destinationStrings(peerID, machine)
	wants := []string{
		fmt.Sprintf(destinationTCPIPv6Template, machine.internalIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationQUICIPv6Template, machine.internalIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationTCPIPv6Template, derivedInternalIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationQUICIPv6Template, derivedInternalIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationTCPIPv4Template, machine.publicIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationQUICIPv4Template, machine.publicIP, config.Get().P2PPort, peerID),
	}
	for _, want := range wants {
		if !slices.Contains(got, want) {
			t.Fatalf("destinationStrings() = %v, want %s", got, want)
		}
	}
}

func TestDestinationStringsPreferWireGuardIPv6BeforePublicIP(t *testing.T) {
	machine, peerID, derivedInternalIP := newTestMachine(t, "203.0.113.10")
	machine.internalIP = "fd00::10"
	p2p := newTestP2P()

	got := p2p.destinationStrings(peerID, machine)
	want := []string{
		fmt.Sprintf(destinationTCPIPv6Template, machine.internalIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationQUICIPv6Template, machine.internalIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationTCPIPv6Template, derivedInternalIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationQUICIPv6Template, derivedInternalIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationTCPIPv4Template, machine.publicIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationQUICIPv4Template, machine.publicIP, config.Get().P2PPort, peerID),
	}
	if len(got) < len(want) {
		t.Fatalf("destinationStrings() = %v, want at least %d destinations", got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("destinationStrings()[%d] = %s, want %s; all destinations: %v", i, got[i], want[i], got)
		}
	}
}

func TestDestinationStringsDoNotPromotePrivatePublicIPBeforeWireGuardIPv6(t *testing.T) {
	machine, peerID, derivedInternalIP := newTestMachine(t, "192.168.64.10")
	p2p := newTestP2P()

	got := p2p.destinationStrings(peerID, machine)
	want := []string{
		fmt.Sprintf(destinationTCPIPv6Template, derivedInternalIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationQUICIPv6Template, derivedInternalIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationTCPIPv4Template, machine.publicIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationQUICIPv4Template, machine.publicIP, config.Get().P2PPort, peerID),
	}
	if len(got) < len(want) {
		t.Fatalf("destinationStrings() = %v, want at least %d destinations", got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("destinationStrings()[%d] = %s, want %s; all destinations: %v", i, got[i], want[i], got)
		}
	}
}

func TestDestinationStringsUseOnlyPublicRelayCandidates(t *testing.T) {
	peerHost, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("new host: %v", err)
	}
	t.Cleanup(func() { _ = peerHost.Close() })

	target, targetID, _ := newTestMachine(t, "")
	publicRelay, publicRelayID, _ := newTestMachine(t, "203.0.113.30")
	privatePeer, privatePeerID, _ := newTestMachine(t, "192.168.64.30")
	manager := newTestP2P()
	manager.host = peerHost
	manager.machines.Set(publicRelayID, publicRelay)
	manager.machines.Set(privatePeerID, privatePeer)
	publicRelayClient := &Client{}
	manager.clients.Set(publicRelayID, publicRelayClient)
	privateRelayClient := &Client{}
	privateRelayClient.setCapabilities([]string{peerCapabilityRelayService})
	manager.clients.Set(privatePeerID, privateRelayClient)

	publicCircuit := fmt.Sprintf("/p2p/%s/p2p-circuit/p2p/%s", publicRelayID, targetID)
	privateCircuit := fmt.Sprintf("/p2p/%s/p2p-circuit/p2p/%s", privatePeerID, targetID)
	if got := manager.destinationStrings(targetID, target); slices.Contains(got, publicCircuit) {
		t.Fatalf("destinationStrings() = %v, used relay without advertised capability", got)
	}
	publicRelayClient.setCapabilities([]string{peerCapabilityRelayService})
	got := manager.destinationStrings(targetID, target)
	if !slices.Contains(got, publicCircuit) {
		t.Fatalf("destinationStrings() = %v, want public relay circuit %s", got, publicCircuit)
	}
	if slices.Contains(got, privateCircuit) {
		t.Fatalf("destinationStrings() = %v, unexpectedly used private peer as relay", got)
	}
}

func TestPeerAddrInfoFromDestinationsCombinesPeerAddresses(t *testing.T) {
	_, peerID, _ := newTestMachine(t, "")
	destinations := []string{
		fmt.Sprintf(destinationTCPIPv4Template, "192.0.2.10", 10500, peerID),
		fmt.Sprintf(destinationTCPIPv4Template, "192.0.2.10", 10500, peerID),
		fmt.Sprintf(destinationQUICIPv4Template, "192.0.2.11", 10500, peerID),
	}

	info, err := peerAddrInfoFromDestinations(peerID, destinations)
	if err != nil {
		t.Fatalf("peerAddrInfoFromDestinations() error = %v", err)
	}

	if info.ID.String() != peerID {
		t.Fatalf("AddrInfo ID = %s, want %s", info.ID, peerID)
	}
	var addrs []string
	for _, addr := range info.Addrs {
		addrs = append(addrs, addr.String())
	}
	want := []string{
		"/ip4/192.0.2.10/tcp/10500",
		"/ip4/192.0.2.11/udp/10500/quic-v1",
	}
	if !slices.Equal(addrs, want) {
		t.Fatalf("AddrInfo addrs = %v, want %v", addrs, want)
	}
}

func TestPeerAddrInfoFromDestinationsRejectsWrongPeerID(t *testing.T) {
	_, peerID, _ := newTestMachine(t, "")
	_, otherPeerID, _ := newTestMachine(t, "")
	destinations := []string{
		fmt.Sprintf(destinationTCPIPv4Template, "192.0.2.10", 10500, otherPeerID),
	}

	if _, err := peerAddrInfoFromDestinations(peerID, destinations); err == nil {
		t.Fatal("peerAddrInfoFromDestinations() returned nil error for wrong destination peer")
	}
}

func TestAddPeerEndpointHintsPreservesSharedPeerstoreAddresses(t *testing.T) {
	peerHost, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("new host: %v", err)
	}
	t.Cleanup(func() { _ = peerHost.Close() })
	_, remotePeerID, _ := newTestMachine(t, "")
	remoteID, err := peer.Decode(remotePeerID)
	if err != nil {
		t.Fatalf("decode remote peer: %v", err)
	}
	existing := ma.StringCast("/ip4/198.51.100.7/tcp/7777")
	hint := ma.StringCast("/ip4/203.0.113.8/udp/10500/quic-v1")
	peerHost.Peerstore().AddAddr(remoteID, existing, time.Hour)

	manager := &P2P{host: peerHost}
	manager.addPeerEndpointHints(peer.AddrInfo{ID: remoteID, Addrs: []ma.Multiaddr{hint}})
	got := peerHost.Peerstore().Addrs(remoteID)
	if !slices.ContainsFunc(got, func(addr ma.Multiaddr) bool { return addr.Equal(existing) }) {
		t.Fatalf("shared peerstore addresses = %v, lost pre-existing address %s", got, existing)
	}
	if !slices.ContainsFunc(got, func(addr ma.Multiaddr) bool { return addr.Equal(hint) }) {
		t.Fatalf("shared peerstore addresses = %v, missing new hint %s", got, hint)
	}
}

func TestDiscoverPeerAtLearnsRemoteIdentityWithFakePeerID(t *testing.T) {
	localKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("local key: %v", err)
	}
	remoteKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("remote key: %v", err)
	}
	local, err := NewManager(localKey, nil, testExternalDB{initialized: true}, 0)
	if err != nil {
		t.Fatalf("local manager: %v", err)
	}
	defer local.host.Close()
	remote, err := NewManager(remoteKey, nil, testExternalDB{}, 0)
	if err != nil {
		t.Fatalf("remote manager: %v", err)
	}
	defer remote.host.Close()

	localStop, err := local.StartServer()
	if err != nil {
		t.Fatalf("start local server: %v", err)
	}
	defer func() {
		if err := localStop(); err != nil {
			t.Errorf("stop local server: %v", err)
		}
	}()
	remoteStop, err := remote.StartServer()
	if err != nil {
		t.Fatalf("start remote server: %v", err)
	}
	defer func() {
		if err := remoteStop(); err != nil {
			t.Errorf("stop remote server: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	discovered, err := local.DiscoverPeerAt(ctx, loopbackQUICAddr(t, remote))
	if err != nil {
		t.Fatalf("discover peer: %v", err)
	}
	if discovered.ID != remoteKey.GetID() {
		t.Fatalf("discovered peer ID = %s, want %s", discovered.ID, remoteKey.GetID())
	}
	if discovered.PublicKey != remoteKey.PublicKey() {
		t.Fatalf("discovered public key = %s, want %s", discovered.PublicKey, remoteKey.PublicKey())
	}
	if discovered.Fingerprint == "" {
		t.Fatal("expected non-empty fingerprint")
	}
}

func TestDiscoverPeerAtLearnsUnknownIdentityAfterMembershipReconciliation(t *testing.T) {
	transports := []struct {
		name string
		addr func(*testing.T, *P2P) string
	}{
		{name: "tcp", addr: loopbackTCPAddr},
		{name: "quic", addr: loopbackQUICAddr},
	}
	for _, transport := range transports {
		t.Run(transport.name, func(t *testing.T) {
			localKey, err := pcrypto.GetLocalKey(t.TempDir())
			if err != nil {
				t.Fatalf("local key: %v", err)
			}
			remoteKey, err := pcrypto.GetLocalKey(t.TempDir())
			if err != nil {
				t.Fatalf("remote key: %v", err)
			}
			local, err := NewManager(localKey, nil, testExternalDB{initialized: true}, 0)
			if err != nil {
				t.Fatalf("local manager: %v", err)
			}
			t.Cleanup(func() { _ = local.host.Close() })
			remote, err := NewManager(remoteKey, nil, testExternalDB{}, 0)
			if err != nil {
				t.Fatalf("remote manager: %v", err)
			}
			t.Cleanup(func() { _ = remote.host.Close() })

			localStop, err := local.StartServer()
			if err != nil {
				t.Fatalf("start local server: %v", err)
			}
			t.Cleanup(func() {
				if err := localStop(); err != nil {
					t.Errorf("stop local server: %v", err)
				}
			})
			remoteStop, err := remote.StartServer()
			if err != nil {
				t.Fatalf("start remote server: %v", err)
			}
			t.Cleanup(func() {
				if err := remoteStop(); err != nil {
					t.Errorf("stop remote server: %v", err)
				}
			})

			// A normal running repository has already reconciled the replicated
			// peer set and therefore fences unknown identities. Provisioning still
			// needs a narrowly scoped identity probe before it can explicitly admit
			// the real ID.
			if err := local.ConfigurePeers(nil); err != nil {
				t.Fatalf("reconcile empty membership: %v", err)
			}
			placeholderID, err := randomPeerID()
			if err != nil {
				t.Fatalf("placeholder peer ID: %v", err)
			}
			local.newIdentityProbePeerID = func() (peer.ID, error) { return placeholderID, nil }

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			discovered, err := local.DiscoverPeerAt(ctx, transport.addr(t, remote))
			if err != nil {
				t.Fatalf("discover peer after membership reconciliation: %v", err)
			}
			if discovered.ID != remoteKey.GetID() {
				t.Fatalf("discovered peer ID = %s, want %s", discovered.ID, remoteKey.GetID())
			}
			remoteID, err := peer.Decode(remoteKey.GetID())
			if err != nil {
				t.Fatalf("decode remote peer: %v", err)
			}
			if local.routeFence.IsPeerFenced(remoteID) {
				t.Fatal("successfully discovered peer remained fenced")
			}
			if local.routeFence.InterceptPeerDial(placeholderID) {
				t.Fatal("successful discovery retained its placeholder identity probe")
			}
		})
	}
}

func TestDiscoverPeerAtReleasesIdentityProbeAfterDialFailure(t *testing.T) {
	localKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("local key: %v", err)
	}
	local, err := NewManager(localKey, nil, testExternalDB{initialized: true}, 0)
	if err != nil {
		t.Fatalf("local manager: %v", err)
	}
	t.Cleanup(func() { _ = local.host.Close() })
	if err := local.ConfigurePeers(nil); err != nil {
		t.Fatalf("reconcile empty membership: %v", err)
	}
	placeholderID, err := randomPeerID()
	if err != nil {
		t.Fatalf("placeholder peer ID: %v", err)
	}
	local.newIdentityProbePeerID = func() (peer.ID, error) { return placeholderID, nil }

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := local.DiscoverPeerAt(ctx, "/ip4/127.0.0.1/tcp/1"); err == nil {
		t.Fatal("identity discovery unexpectedly succeeded against a closed endpoint")
	}
	if local.routeFence.InterceptPeerDial(placeholderID) {
		t.Fatal("failed discovery retained its placeholder identity probe")
	}
}

func TestDiscoverPeerAtCannotReopenDeletedRealIdentity(t *testing.T) {
	localKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("local key: %v", err)
	}
	remoteKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("remote key: %v", err)
	}
	local, err := NewManager(localKey, nil, testExternalDB{initialized: true}, 0)
	if err != nil {
		t.Fatalf("local manager: %v", err)
	}
	t.Cleanup(func() { _ = local.host.Close() })
	remote, err := NewManager(remoteKey, nil, testExternalDB{}, 0)
	if err != nil {
		t.Fatalf("remote manager: %v", err)
	}
	t.Cleanup(func() { _ = remote.host.Close() })
	if err := local.ConfigurePeers(nil); err != nil {
		t.Fatalf("reconcile empty membership: %v", err)
	}
	remoteID, err := peer.Decode(remoteKey.GetID())
	if err != nil {
		t.Fatalf("decode remote peer: %v", err)
	}
	if _, err := local.routeFence.FencePeer(remoteID); err != nil {
		t.Fatalf("fence deleted peer: %v", err)
	}
	placeholderID, err := randomPeerID()
	if err != nil {
		t.Fatalf("placeholder peer ID: %v", err)
	}
	local.newIdentityProbePeerID = func() (peer.ID, error) { return placeholderID, nil }

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := local.DiscoverPeerAt(ctx, loopbackTCPAddr(t, remote)); !errors.Is(err, swarmionlink.ErrPeerFenced) {
		t.Fatalf("discovery across deletion fence error = %v, want ErrPeerFenced", err)
	}
	if !local.routeFence.IsPeerFenced(remoteID) {
		t.Fatal("discovery reopened the deleted real identity")
	}
	if local.routeFence.InterceptPeerDial(placeholderID) {
		t.Fatal("deletion-fence rejection retained its placeholder identity probe")
	}
	if local.host.Network().Connectedness(remoteID) != network.NotConnected {
		t.Fatal("discovery across deletion fence established a physical route")
	}
}

func TestDiscoverPeerAtReleasesLearnedIDAfterFollowupDialFailure(t *testing.T) {
	localKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("local key: %v", err)
	}
	remoteKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("remote key: %v", err)
	}
	local, err := NewManager(localKey, nil, testExternalDB{initialized: true}, 0)
	if err != nil {
		t.Fatalf("local manager: %v", err)
	}
	t.Cleanup(func() { _ = local.host.Close() })
	remote, err := NewManager(remoteKey, nil, testExternalDB{}, 0)
	if err != nil {
		t.Fatalf("remote manager: %v", err)
	}
	t.Cleanup(func() { _ = remote.host.Close() })
	if err := local.ConfigurePeers(nil); err != nil {
		t.Fatalf("reconcile empty membership: %v", err)
	}
	placeholderID, err := randomPeerID()
	if err != nil {
		t.Fatal(err)
	}
	local.newIdentityProbePeerID = func() (peer.ID, error) { return placeholderID, nil }
	local.afterIdentityLearnedForTest = func(actual peer.ID) {
		if actual != remote.host.ID() {
			t.Fatalf("learned peer = %s, want %s", actual, remote.host.ID())
		}
		if err := remote.host.Close(); err != nil {
			t.Fatalf("close remote before follow-up dial: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = local.DiscoverPeerAt(ctx, loopbackTCPAddr(t, remote))
	var dialErr *discoveredPeerDialError
	if !errors.As(err, &dialErr) {
		t.Fatalf("follow-up dial error = %v, want discoveredPeerDialError", err)
	}
	assertRejectedDiscoveredPeer(t, local, remote.host.ID(), placeholderID)
}

func TestDiscoverPeerAtRejectsUnsupportedAuthenticatedKeyWithoutAdmissionLeak(t *testing.T) {
	localKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("local key: %v", err)
	}
	local, err := NewManager(localKey, nil, testExternalDB{initialized: true}, 0)
	if err != nil {
		t.Fatalf("local manager: %v", err)
	}
	t.Cleanup(func() { _ = local.host.Close() })
	if err := local.ConfigurePeers(nil); err != nil {
		t.Fatalf("reconcile empty membership: %v", err)
	}

	privateKey, _, err := crypto.GenerateKeyPair(crypto.RSA, 2048)
	if err != nil {
		t.Fatalf("generate unsupported remote key: %v", err)
	}
	remote, err := libp2p.New(
		libp2p.Identity(privateKey),
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	if err != nil {
		t.Fatalf("new unsupported-key remote: %v", err)
	}
	t.Cleanup(func() { _ = remote.Close() })
	placeholderID, err := randomPeerID()
	if err != nil {
		t.Fatal(err)
	}
	local.newIdentityProbePeerID = func() (peer.ID, error) { return placeholderID, nil }

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = local.DiscoverPeerAt(ctx, loopbackTCPHostAddr(t, remote))
	if err == nil || !strings.Contains(err.Error(), "unsupported peer public key type RSA") {
		t.Fatalf("unsupported-key discovery error = %v", err)
	}
	assertRejectedDiscoveredPeer(t, local, remote.ID(), placeholderID)
}

func TestConnectKnownPeerAtFailureDoesNotDeleteAnotherScopePendingAdmission(t *testing.T) {
	localKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	local, err := NewManager(localKey, nil, testExternalDB{initialized: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = local.host.Close() })
	if err := local.ConfigurePeers(nil); err != nil {
		t.Fatal(err)
	}
	actualID, err := randomPeerID()
	if err != nil {
		t.Fatal(err)
	}
	otherScope, err := local.routeFence.BeginTemporaryPeerAdmission(actualID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(otherScope.Close)
	local.pendingPeers.Set(actualID.String(), true)
	t.Cleanup(func() { local.pendingPeers.Delete(actualID.String()) })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := local.connectKnownPeerAt(ctx, "/ip4/127.0.0.1/tcp/1", actualID); err == nil {
		t.Fatal("follow-up dial unexpectedly succeeded")
	}
	if pending, found := local.pendingPeers.Get(actualID.String()); !found || !pending {
		t.Fatal("failed discovery deleted another scope's pending admission")
	}
	if !local.routeFence.IsPeerConnectionAllowed(actualID) {
		t.Fatal("failed discovery revoked another scope's temporary admission")
	}
}

func assertRejectedDiscoveredPeer(t *testing.T, local *P2P, actualID, placeholderID peer.ID) {
	t.Helper()
	if !local.routeFence.IsPeerFenced(actualID) || local.routeFence.IsPeerConnectionAllowed(actualID) {
		t.Fatal("rejected real identity retained route-fence admission")
	}
	if local.routeFence.InterceptPeerDial(actualID) ||
		local.routeFence.InterceptSecured(network.DirOutbound, actualID, nil) {
		t.Fatal("rejected real identity remained dialable or secured-admitted")
	}
	if local.controlPeerAllowed(actualID) {
		t.Fatal("rejected real identity retained Protos control admission")
	}
	if pending, found := local.pendingPeers.Get(actualID.String()); found || pending {
		t.Fatal("rejected real identity retained pending-peer admission")
	}
	if _, found := local.machines.Get(actualID.String()); found {
		t.Fatal("rejected real identity retained machine admission")
	}
	if connectedness := local.host.Network().Connectedness(actualID); connectedness != network.NotConnected {
		t.Fatalf("rejected real identity retained physical connectedness %s", connectedness)
	}
	if connections := local.host.Network().ConnsToPeer(actualID); len(connections) != 0 {
		t.Fatalf("rejected real identity retained %d physical connections", len(connections))
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	snapshot, subscription, err := local.registry.Link().SubscribeRoutes(ctx)
	if err != nil {
		t.Fatalf("subscribe Swarmion Link routes: %v", err)
	}
	t.Cleanup(func() { _ = subscription.Close(context.Background()) })
	if _, found := snapshot.Routes[swarmiontransport.PeerID(actualID.String())]; found {
		t.Fatal("rejected real identity survived in the Swarmion Link route snapshot")
	}
	if local.routeFence.InterceptPeerDial(placeholderID) {
		t.Fatal("rejected discovery retained its placeholder identity probe")
	}
	if addrs := local.host.Peerstore().Addrs(placeholderID); len(addrs) != 0 {
		t.Fatalf("placeholder retained call-owned peerstore addresses: %v", addrs)
	}
}

func TestTofuTransportAddrsPreferTCPBeforeQUIC(t *testing.T) {
	p2p := &P2P{p2pPort: 1234}

	got, err := p2p.tofuTransportAddrs("203.0.113.10")
	if err != nil {
		t.Fatalf("tofuTransportAddrs() error = %v", err)
	}
	want := []string{
		"/ip4/203.0.113.10/tcp/1234",
		"/ip4/203.0.113.10/udp/1234/quic-v1",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("tofuTransportAddrs() = %v, want %v", got, want)
	}
}

func TestGetRuntimeStateRequiresStrictCatchUp(t *testing.T) {
	wantErr := fmt.Errorf("strict catch-up failed")
	server := &Server{DB: runtimeStateExternalDB{strictErr: wantErr}}

	_, err := server.GetRuntimeState(context.Background(), &p2pproto.GetRuntimeStateRequest{})
	if err == nil {
		t.Fatal("GetRuntimeState returned nil error, want strict catch-up error")
	}
	if err.Error() != wantErr.Error() {
		t.Fatalf("GetRuntimeState error = %v, want %v", err, wantErr)
	}
}

func TestGetRuntimeStateExposesEventReceiptContentDissentObservations(t *testing.T) {
	t.Parallel()

	server := &Server{DB: runtimeStateExternalDB{
		eventReceiptMetrics: db.EventReceiptMetrics{ContentDissentObservations: 13},
	}}
	response, err := server.GetRuntimeState(context.Background(), &p2pproto.GetRuntimeStateRequest{})
	if err != nil {
		t.Fatalf("GetRuntimeState() error = %v", err)
	}
	if got := response.GetState().GetEventReceiptContentDissentObservations(); got != 13 {
		t.Fatalf("event receipt content dissent observations = %d, want 13", got)
	}
}

func TestImageCapableClientsRequireAdvertisedCapability(t *testing.T) {
	withCapability := &Client{ImagesClient: fakeImagesClient{}}
	withCapability.setCapabilities([]string{peerCapabilityImageContent})
	withoutCapability := &Client{ImagesClient: fakeImagesClient{}}

	got := imageCapableClients(map[string]*Client{
		"with":    withCapability,
		"without": withoutCapability,
		"nil":     nil,
	})

	if _, found := got["with"]; !found {
		t.Fatal("client advertising image content capability was not selected")
	}
	if _, found := got["without"]; found {
		t.Fatal("client without image content capability was selected")
	}
	if _, found := got["nil"]; found {
		t.Fatal("nil client was selected")
	}
}

func TestPeerCapabilitiesAdvertiseSwarmionTransportWhenDBInitialized(t *testing.T) {
	uninitialized := (&Server{
		DB:  testExternalDB{initialized: false},
		p2p: &P2P{},
	}).peerCapabilities()
	if slices.Contains(uninitialized, peerCapabilitySwarmionTransport) {
		t.Fatalf("uninitialized DB capabilities = %v, should not include %s", uninitialized, peerCapabilitySwarmionTransport)
	}

	initialized := (&Server{
		DB:  testExternalDB{initialized: true},
		p2p: &P2P{},
	}).peerCapabilities()
	if !slices.Contains(initialized, peerCapabilitySwarmionTransport) {
		t.Fatalf("initialized DB capabilities = %v, want %s", initialized, peerCapabilitySwarmionTransport)
	}
}

func TestExistingRepositoryBootstrapPendingRejectsControlAndInit(t *testing.T) {
	remoteKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	remotePeer, err := peer.Decode(remoteKey.GetID())
	if err != nil {
		t.Fatal(err)
	}
	pendingDB := testExternalDB{readiness: db.RepositoryReadiness{
		ExistingRepository: true,
		BootstrapPending:   true,
	}}
	manager := &P2P{
		externalDB:   pendingDB,
		machines:     util.NewMap[string, Machine](),
		pendingPeers: util.NewMap[string, bool](),
	}
	if manager.controlPeerAllowed(remotePeer) {
		t.Fatal("unknown control peer was admitted while existing-repository bootstrap was pending")
	}
	server := &Server{DB: pendingDB, p2p: manager}
	if _, err := server.Init(context.Background(), &p2pproto.InitRequest{}); err == nil || !strings.Contains(err.Error(), "awaiting bootstrap recovery") {
		t.Fatalf("Init error = %v, want existing-repository bootstrap rejection", err)
	}

	permanentBootstrapErr := errors.New("permanent bootstrap lineage failure")
	failedDB := testExternalDB{readiness: db.RepositoryReadiness{
		ExistingRepository: true,
		BootstrapError:     permanentBootstrapErr,
	}}
	manager.externalDB = failedDB
	if manager.controlPeerAllowed(remotePeer) {
		t.Fatal("unknown control peer was admitted after permanent existing-repository bootstrap failure")
	}
	server.DB = failedDB
	if _, err := server.Init(context.Background(), &p2pproto.InitRequest{}); !errors.Is(err, permanentBootstrapErr) {
		t.Fatalf("Init error = %v, want permanent bootstrap failure %v", err, permanentBootstrapErr)
	}

	manager.externalDB = testExternalDB{readiness: db.RepositoryReadiness{
		Initialized:        true,
		ExistingRepository: true,
	}}
	manager.machines.Set(remotePeer.String(), testMachine{id: "recovered", publicKey: remoteKey.PublicString()})
	if !manager.controlPeerAllowed(remotePeer) {
		t.Fatal("known control peer remained rejected after repository readiness recovered")
	}

	manager.externalDB = testExternalDB{}
	if !manager.controlPeerAllowed(remotePeer) {
		t.Fatal("true fresh database did not retain initialization admission")
	}
}

func TestPeerCapabilitiesAdvertiseRelayServiceOnlyForPublicHost(t *testing.T) {
	newHost := func(t *testing.T, advertised ma.Multiaddr) host.Host {
		t.Helper()
		options := []libp2p.Option{libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0")}
		if advertised != nil {
			options = append(options, libp2p.AddrsFactory(func([]ma.Multiaddr) []ma.Multiaddr {
				return []ma.Multiaddr{advertised}
			}))
		}
		peerHost, err := libp2p.New(options...)
		if err != nil {
			t.Fatalf("new host: %v", err)
		}
		t.Cleanup(func() { _ = peerHost.Close() })
		return peerHost
	}

	privateHost := newHost(t, nil)
	privateCapabilities := (&Server{DB: testExternalDB{initialized: true}, p2p: &P2P{host: privateHost}}).peerCapabilities()
	if slices.Contains(privateCapabilities, peerCapabilityRelayService) {
		t.Fatalf("private host capabilities = %v, unexpectedly advertises relay service", privateCapabilities)
	}

	publicHost := newHost(t, ma.StringCast("/ip4/203.0.113.44/tcp/10500"))
	publicCapabilities := (&Server{DB: testExternalDB{initialized: true}, p2p: &P2P{host: publicHost}}).peerCapabilities()
	if !slices.Contains(publicCapabilities, peerCapabilityRelayService) {
		t.Fatalf("public host capabilities = %v, want %s", publicCapabilities, peerCapabilityRelayService)
	}
}

func TestBorrowedManagerAndSwarmionLinkReuseApplicationHost(t *testing.T) {
	localKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("local key: %v", err)
	}
	peerHost, err := NewHost(localKey, 0)
	if err != nil {
		t.Fatalf("new host: %v", err)
	}
	t.Cleanup(func() { _ = peerHost.Close() })

	registry, err := swarmionlink.NewRegistry(peerHost)
	if err != nil {
		t.Fatalf("new shared registry: %v", err)
	}
	manager, err := NewManagerWithRegistry(peerHost, registry, nil, testExternalDB{initialized: true}, 0)
	if err != nil {
		t.Fatalf("new borrowed manager: %v", err)
	}
	link := registry.Link()
	if got, want := string(link.LocalPeer()), manager.PeerID(); got != want {
		t.Fatalf("shared host peer ID = %s, want %s", got, want)
	}

	if err := manager.Close(); err != nil {
		t.Fatalf("close borrowed manager: %v", err)
	}
	if len(peerHost.Addrs()) == 0 {
		t.Fatal("borrowed manager closed the application host")
	}

	remote, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("new remote host: %v", err)
	}
	t.Cleanup(func() { _ = remote.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := remote.Connect(ctx, peer.AddrInfo{ID: peerHost.ID(), Addrs: peerHost.Addrs()}); err != nil {
		t.Fatalf("connect after borrowed manager close: %v", err)
	}
	if got, want := manager.PhysicalConnectedPeerIDs(), []string{remote.ID().String()}; !slices.Equal(got, want) {
		t.Fatalf("physical connected peer IDs = %v, want %v", got, want)
	}
}

func TestPhysicalConnectedPeerIDsHandlesMissingHost(t *testing.T) {
	t.Parallel()
	if got := (*P2P)(nil).PhysicalConnectedPeerIDs(); got != nil {
		t.Fatalf("nil manager physical peer IDs = %v, want nil", got)
	}
	if got := (&P2P{}).PhysicalConnectedPeerIDs(); got != nil {
		t.Fatalf("missing host physical peer IDs = %v, want nil", got)
	}
}

func TestBoundedGRPCStopForcesBlockedRPC(t *testing.T) {
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	started := make(chan struct{})
	server.RegisterService(&blockingGRPCServiceDesc, &blockingGRPCService{started: started})
	go func() { _ = server.Serve(listener) }()

	conn, err := grpc.NewClient(
		"passthrough:///p2p-stop-test",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
	)
	if err != nil {
		t.Fatalf("new grpc client: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	callDone := make(chan error, 1)
	go func() {
		callDone <- conn.Invoke(context.Background(), "/protos.test.Blocker/Block", &emptypb.Empty{}, &emptypb.Empty{})
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("blocked RPC did not start")
	}

	startedAt := time.Now()
	if forced := boundedGRPCStop(server, 20*time.Millisecond); !forced {
		t.Fatal("bounded gRPC stop reported graceful completion for blocked RPC")
	}
	if elapsed := time.Since(startedAt); elapsed > time.Second {
		t.Fatalf("bounded gRPC stop took %s", elapsed)
	}
	select {
	case <-callDone:
	case <-time.After(time.Second):
		t.Fatal("forced gRPC stop did not release blocked RPC")
	}
}

func TestNewManagerWithHostRejectsApplicationProtocolCollision(t *testing.T) {
	localKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("local key: %v", err)
	}
	peerHost, err := NewHost(localKey, 0)
	if err != nil {
		t.Fatalf("new host: %v", err)
	}
	t.Cleanup(func() { _ = peerHost.Close() })
	peerHost.SetStreamHandler(protosRPCProtocol, func(network.Stream) {})

	_, err = NewManagerWithHost(peerHost, nil, testExternalDB{initialized: true}, 0)
	if err == nil || !errors.Is(err, swarmiontransport.ErrProtocolInUse) {
		t.Fatalf("protocol collision error = %v, want typed collision rejection", err)
	}
}

func TestUnknownPeerControlStreamRejectedWithoutClosingSharedRoute(t *testing.T) {
	localKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("local key: %v", err)
	}
	manager, err := NewManager(localKey, nil, testExternalDB{initialized: true}, 0)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	stop, err := manager.StartServer()
	if err != nil {
		t.Fatalf("start manager: %v", err)
	}
	t.Cleanup(func() {
		_ = stop()
		_ = manager.Close()
	})

	remote, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("new remote host: %v", err)
	}
	t.Cleanup(func() { _ = remote.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := remote.Connect(ctx, peer.AddrInfo{ID: manager.host.ID(), Addrs: manager.host.Addrs()}); err != nil {
		t.Fatalf("connect physical route: %v", err)
	}

	stream, streamErr := remote.NewStream(ctx, manager.host.ID(), protosRPCProtocol)
	if streamErr == nil {
		_ = stream.SetDeadline(time.Now().Add(time.Second))
		_, _ = stream.Write([]byte("not grpc"))
		buffer := make([]byte, 1)
		if _, err := stream.Read(buffer); err == nil {
			t.Fatal("unknown peer control stream remained open")
		}
		_ = stream.Close()
	}

	if got := remote.Network().Connectedness(manager.host.ID()); !usablePeerConnectedness(got) {
		t.Fatalf("control rejection closed shared physical route: %s", got)
	}
}

func TestAddPeerAndReconcilerBoundControlClientReadiness(t *testing.T) {
	localKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("local key: %v", err)
	}
	manager, err := NewManager(localKey, nil, testExternalDB{initialized: true}, 0)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	manager.controlClientTimeout = 50 * time.Millisecond
	// Isolate the synchronous AddPeer/reconciler path from the independent
	// connection notification path. Both paths use the same bounded readiness
	// helper, but this regression specifically proves neither synchronous caller
	// can be held forever by a physically connected peer without Protos control.
	manager.host.Network().StopNotify(manager.notify)
	manager.controlClientTimeout = 40 * time.Millisecond

	remoteKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("remote key: %v", err)
	}
	remote, err := NewHost(remoteKey, 0)
	if err != nil {
		t.Fatalf("new remote host: %v", err)
	}
	t.Cleanup(func() { _ = remote.Close() })

	for _, address := range remote.Addrs() {
		value, valueErr := address.ValueForProtocol(ma.P_TCP)
		if valueErr != nil {
			continue
		}
		if _, scanErr := fmt.Sscanf(value, "%d", &manager.p2pPort); scanErr == nil && manager.p2pPort > 0 {
			break
		}
	}
	if manager.p2pPort == 0 {
		t.Fatal("remote host did not expose a TCP listen port")
	}

	machine := testMachine{
		id:        "remote-without-control",
		publicKey: remoteKey.PublicString(),
		publicIP:  "127.0.0.1",
	}
	addResult := make(chan error, 1)
	go func() {
		_, addErr := manager.AddPeer(machine)
		addResult <- addErr
	}()
	select {
	case addErr := <-addResult:
		if addErr == nil {
			t.Fatal("AddPeer succeeded without a remote Protos control handler")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("AddPeer blocked waiting for remote Protos control readiness")
	}
	if got := manager.host.Network().Connectedness(remote.ID()); !usablePeerConnectedness(got) {
		t.Fatalf("failed control readiness closed shared physical route: %s", got)
	}

	manager.updatePeerState(remote.ID().String(), func(state *PeerState) {
		state.Status = PeerStatusDesired
		state.LastAttempt = time.Time{}
		state.LastError = ""
		state.Attempts = 0
	})
	reconciled := make(chan struct{})
	go func() {
		manager.reconcilePeers()
		close(reconciled)
	}()
	select {
	case <-reconciled:
	case <-time.After(2 * time.Second):
		t.Fatal("peer reconciler blocked waiting for remote Protos control readiness")
	}
	state, found := manager.peerStates.Get(remote.ID().String())
	if !found {
		t.Fatal("peer reconciler removed desired peer state")
	}
	if state.Status != PeerStatusUnreachable || state.Attempts != 1 || state.LastError == "" {
		t.Fatalf("peer reconciler state = %+v, want one failed bounded attempt", state)
	}
	if got := manager.host.Network().Connectedness(remote.ID()); !usablePeerConnectedness(got) {
		t.Fatalf("reconciler control failure closed shared physical route: %s", got)
	}
}

func TestRefreshPeerControlAfterRemoteRestartKeepsPhysicalRoute(t *testing.T) {
	localKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("local key: %v", err)
	}
	manager, err := NewManager(localKey, nil, testExternalDB{initialized: true}, 0)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = manager.Close() })

	remoteKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("remote key: %v", err)
	}
	remote, err := NewHost(remoteKey, 0)
	if err != nil {
		t.Fatalf("new remote host: %v", err)
	}
	t.Cleanup(func() { _ = remote.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := manager.host.Connect(ctx, peer.AddrInfo{ID: remote.ID(), Addrs: remote.Addrs()}); err != nil {
		t.Fatalf("connect physical route: %v", err)
	}
	machine := testMachine{id: "remote", publicKey: remoteKey.PublicString()}
	peerID, err := manager.trackPeer(machine)
	if err != nil {
		t.Fatalf("track peer: %v", err)
	}

	if err := manager.RefreshPeerControlAfterRemoteRestart(machine); err != nil {
		t.Fatalf("refresh control: %v", err)
	}
	if _, found := manager.machines.Get(peerID); !found {
		t.Fatalf("remote restart discarded desired machine %s", peerID)
	}
	if got := manager.host.Network().Connectedness(remote.ID()); !usablePeerConnectedness(got) {
		t.Fatalf("control refresh closed shared physical route: %s", got)
	}
}

func TestSiblingConnectionCallbacksReuseClientAndWaitForPeerRouteLoss(t *testing.T) {
	localKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("local key: %v", err)
	}
	manager, err := NewManager(localKey, nil, testExternalDB{initialized: true}, 0)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = manager.Close() })

	remoteKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("remote key: %v", err)
	}
	remote, err := NewHost(remoteKey, 0)
	if err != nil {
		t.Fatalf("new remote host: %v", err)
	}
	t.Cleanup(func() { _ = remote.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := manager.host.Connect(ctx, peer.AddrInfo{ID: remote.ID(), Addrs: remote.Addrs()}); err != nil {
		t.Fatalf("connect physical route: %v", err)
	}
	machine := testMachine{id: "remote", publicKey: remoteKey.PublicString()}
	if _, err := manager.trackPeer(machine); err != nil {
		t.Fatalf("track peer: %v", err)
	}

	existing := &Client{}
	manager.clients.Set(remote.ID().String(), existing)
	connections := manager.host.Network().ConnsToPeer(remote.ID())
	if len(connections) == 0 {
		t.Fatal("missing physical connection")
	}
	// Simulate a sibling route arrival. It must reuse the existing logical
	// client rather than overwrite and leak it.
	manager.newConnectionHandler(manager.host.Network(), connections[0])
	time.Sleep(50 * time.Millisecond)
	if got, found := manager.clients.Get(remote.ID().String()); !found || got != existing {
		t.Fatalf("sibling connection replaced logical client: got %p, want %p", got, existing)
	}

	// Track a second route, then simulate one sibling disappearing. Membership,
	// not a timer or callback ordering, keeps the logical client alive.
	manager.addPhysicalRoute(remote.ID(), "synthetic-quic-sibling")
	manager.closeConnectionHandler(manager.host.Network(), connections[0])
	if got, found := manager.clients.Get(remote.ID().String()); !found || got != existing {
		t.Fatalf("sibling route loss removed logical client: got %p, want %p", got, existing)
	}
	manager.relayReservations.Set(remote.ID().String(), time.Now().Add(time.Hour))

	if err := manager.host.Network().ClosePeer(remote.ID()); err != nil {
		t.Fatalf("close all peer routes: %v", err)
	}
	// ClosePeer returns before every libp2p Disconnected callback has
	// necessarily drained. Wait until the physical network and the callback-
	// maintained membership agree that only the synthetic sibling remains;
	// otherwise this test races the real connection's asynchronous removal.
	physicalDeadline := time.Now().Add(time.Second)
	for time.Now().Before(physicalDeadline) {
		if manager.host.Network().Connectedness(remote.ID()) == network.NotConnected &&
			manager.physicalRouteCount(remote.ID()) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if connectedness, routes := manager.host.Network().Connectedness(remote.ID()), manager.physicalRouteCount(remote.ID()); connectedness != network.NotConnected || routes != 1 {
		t.Fatalf("physical route callbacks did not drain: connectedness=%s routes=%d", connectedness, routes)
	}
	if remaining := manager.removePhysicalRoute(remote.ID(), "synthetic-quic-sibling"); remaining != 0 {
		t.Fatalf("synthetic sibling removal left %d routes", remaining)
	}
	manager.handleLastPhysicalRouteLost(manager.host.Network(), remote.ID())
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if _, found := manager.clients.Get(remote.ID().String()); !found {
			if _, retained := manager.relayReservations.Get(remote.ID().String()); retained {
				t.Fatal("final route loss retained stale relay reservation")
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("logical client remained after all physical routes were lost")
}

func TestRemovePeerRetainsPhysicalRemovalAuthority(t *testing.T) {
	localKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("local key: %v", err)
	}
	manager, err := NewManager(localKey, nil, testExternalDB{initialized: true}, 0)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = manager.Close() })

	remoteKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("remote key: %v", err)
	}
	remote, err := NewHost(remoteKey, 0)
	if err != nil {
		t.Fatalf("new remote host: %v", err)
	}
	t.Cleanup(func() { _ = remote.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := manager.host.Connect(ctx, peer.AddrInfo{ID: remote.ID(), Addrs: remote.Addrs()}); err != nil {
		t.Fatalf("connect physical route: %v", err)
	}
	machine := testMachine{id: "remote", publicKey: remoteKey.PublicString()}
	if _, err := manager.trackPeer(machine); err != nil {
		t.Fatalf("track peer: %v", err)
	}
	manager.relayReservations.Set(remote.ID().String(), time.Now().Add(time.Hour))

	if err := manager.RemovePeer(machine); err != nil {
		t.Fatalf("remove peer: %v", err)
	}
	if got := manager.host.Network().Connectedness(remote.ID()); got != network.NotConnected {
		t.Fatalf("canonical machine removal left physical route %s", got)
	}
	if _, retained := manager.relayReservations.Get(remote.ID().String()); retained {
		t.Fatal("canonical machine removal retained stale relay reservation")
	}
}

func TestFencePeerRejectsDynamicRedialAndAdmitReopensFreshGeneration(t *testing.T) {
	localKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("local key: %v", err)
	}
	manager, err := NewManager(localKey, nil, testExternalDB{initialized: true}, 0)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	manager.controlClientTimeout = 50 * time.Millisecond
	remoteKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("remote key: %v", err)
	}
	remote, err := NewHost(remoteKey, 0)
	if err != nil {
		t.Fatalf("new remote host: %v", err)
	}
	t.Cleanup(func() { _ = remote.Close() })
	machine := testMachine{id: "remote", publicKey: remoteKey.PublicString()}
	if _, err := manager.trackPeer(machine); err != nil {
		t.Fatalf("track peer: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := remote.Connect(ctx, peer.AddrInfo{ID: manager.host.ID(), Addrs: manager.host.Addrs()}); err != nil {
		t.Fatalf("connect admitted peer: %v", err)
	}
	peerID, first, err := manager.FencePeer(machine)
	if err != nil {
		t.Fatalf("fence peer: %v", err)
	}
	if peerID != remote.ID().String() || first == "" {
		t.Fatalf("fence result = (%q, %q), want remote peer and generation", peerID, first)
	}
	redialCtx, redialCancel := context.WithTimeout(context.Background(), time.Second)
	_ = remote.Connect(redialCtx, peer.AddrInfo{ID: manager.host.ID(), Addrs: manager.host.Addrs()})
	redialCancel()
	time.Sleep(100 * time.Millisecond)
	if remote.Network().Connectedness(manager.host.ID()) != network.NotConnected ||
		manager.host.Network().Connectedness(remote.ID()) != network.NotConnected {
		t.Fatal("fenced peer redial established a physical connection")
	}
	if err := manager.routeFence.ReopenPeer(remote.ID()); err != nil {
		t.Fatalf("readmit peer: %v", err)
	}
	if err := remote.Connect(ctx, peer.AddrInfo{ID: manager.host.ID(), Addrs: manager.host.Addrs()}); err != nil {
		t.Fatalf("readmitted peer connect: %v", err)
	}
	_, second, err := manager.FencePeer(machine)
	if err != nil {
		t.Fatalf("refence peer: %v", err)
	}
	if first == second {
		t.Fatalf("fence generation reused: %s", first)
	}
	if err := manager.WithPeerFenceGeneration(ctx, peerID, first, func() error { return nil }); err == nil {
		t.Fatal("old fence generation remained valid")
	}
	if err := manager.WithPeerFenceGeneration(ctx, peerID, second, func() error { return nil }); err != nil {
		t.Fatalf("current fence generation invalid: %v", err)
	}
}

func TestFencePeerFailsClosedOnPhysicalCloseFailureOrSurvivingConnection(t *testing.T) {
	tests := []struct {
		name            string
		closeErr        error
		connectionCount int
	}{
		{name: "close failure", closeErr: fmt.Errorf("injected close failure")},
		{name: "surviving sibling connection", connectionCount: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localKey, err := pcrypto.GetLocalKey(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			manager, err := NewManager(localKey, nil, testExternalDB{initialized: true}, 0)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = manager.Close() })
			remoteKey, err := pcrypto.GetLocalKey(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			remoteID, err := manager.pubKeyToPeerID(remoteKey.PublicString())
			if err != nil {
				t.Fatal(err)
			}
			manager.closePeerForDrain = func(peer.ID) error { return tt.closeErr }
			manager.peerConnectionsForDrain = func(peer.ID) int { return tt.connectionCount }
			machine := testMachine{id: "remote", publicKey: remoteKey.PublicString()}

			if _, _, err := manager.FencePeer(machine); err == nil {
				t.Fatal("FencePeer succeeded without verified physical quiescence")
			}
			if !manager.routeFence.IsPeerFenced(remoteID) {
				t.Fatal("failed FencePeer reopened its explicit fence")
			}
		})
	}
}

func TestConfigurePeersCannotReopenExplicitDeletionFence(t *testing.T) {
	localKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager(localKey, nil, testExternalDB{initialized: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	remoteKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	remoteID, err := manager.pubKeyToPeerID(remoteKey.PublicString())
	if err != nil {
		t.Fatal(err)
	}
	machine := testMachine{id: "remote", publicKey: remoteKey.PublicString()}
	if _, _, err := manager.FencePeer(machine); err != nil {
		t.Fatal(err)
	}
	if err := manager.ConfigurePeers([]Machine{machine}); err != nil {
		t.Fatalf("stale ConfigurePeers should be ignored, got: %v", err)
	}
	if !manager.routeFence.IsPeerFenced(remoteID) {
		t.Fatal("stale ConfigurePeers reopened explicit deletion fence")
	}
	if _, tracked := manager.machines.Get(remoteID.String()); tracked {
		t.Fatal("stale ConfigurePeers restored peer tracking")
	}
}

func TestConfigurePeersKeepsPendingBootstrapAdmissionAndFencesItWhenReleased(t *testing.T) {
	localKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager(localKey, nil, testExternalDB{initialized: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = manager.Close() })
	remoteKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	remotePeerID, err := manager.pubKeyToPeerID(remoteKey.PublicString())
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.routeFence.AdmitPeer(remotePeerID); err != nil {
		t.Fatal(err)
	}
	manager.pendingPeers.Set(remotePeerID.String(), true)
	if err := manager.ConfigurePeers(nil); err != nil {
		t.Fatal(err)
	}
	if manager.routeFence.IsPeerFenced(remotePeerID) {
		t.Fatal("pending/bootstrap peer was fenced by replicated-membership reconciliation")
	}
	manager.pendingPeers.Delete(remotePeerID.String())
	if err := manager.ConfigurePeers(nil); err != nil {
		t.Fatal(err)
	}
	if !manager.routeFence.IsPeerFenced(remotePeerID) {
		t.Fatal("released unknown peer remained admitted")
	}
}

func TestDialableListenMultiaddrsUseSharedPort(t *testing.T) {
	peerHost, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("new host: %v", err)
	}
	t.Cleanup(func() { _ = peerHost.Close() })
	manager := &P2P{host: peerHost, p2pPort: 10500}
	got := manager.DialableListenMultiaddrs([]string{"203.0.113.10"})
	wants := []string{
		fmt.Sprintf(destinationTCPIPv4Template, "203.0.113.10", 10500, peerHost.ID()),
		fmt.Sprintf(destinationQUICIPv4Template, "203.0.113.10", 10500, peerHost.ID()),
	}
	for _, want := range wants {
		if !slices.Contains(got, want) {
			t.Fatalf("DialableListenMultiaddrs() = %v, want %s", got, want)
		}
	}
	for _, addr := range got {
		if strings.Contains(addr, "/10501/") {
			t.Fatalf("found removed P2PPort+1 Swarmion endpoint: %s", addr)
		}
	}
}

func loopbackQUICAddr(t *testing.T, p2p *P2P) string {
	t.Helper()
	for _, addr := range p2p.host.Addrs() {
		port, err := addr.ValueForProtocol(ma.P_UDP)
		if err != nil {
			continue
		}
		return fmt.Sprintf("/ip4/127.0.0.1/udp/%s/quic-v1", port)
	}
	t.Fatalf("no QUIC listen address found: %v", p2p.host.Addrs())
	return ""
}

func loopbackTCPAddr(t *testing.T, p2p *P2P) string {
	t.Helper()
	return loopbackTCPHostAddr(t, p2p.host)
}

func loopbackTCPHostAddr(t *testing.T, peerHost host.Host) string {
	t.Helper()
	for _, addr := range peerHost.Addrs() {
		port, err := addr.ValueForProtocol(ma.P_TCP)
		if err != nil {
			continue
		}
		return fmt.Sprintf("/ip4/127.0.0.1/tcp/%s", port)
	}
	t.Fatalf("no TCP listen address found: %v", peerHost.Addrs())
	return ""
}

type testExternalDB struct {
	initialized bool
	readiness   db.RepositoryReadiness
}

type blockingGRPCServiceServer interface {
	Block(context.Context, *emptypb.Empty) (*emptypb.Empty, error)
}

type blockingGRPCService struct {
	started chan struct{}
}

func (service *blockingGRPCService) Block(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	close(service.started)
	<-ctx.Done()
	return nil, ctx.Err()
}

var blockingGRPCServiceDesc = grpc.ServiceDesc{
	ServiceName: "protos.test.Blocker",
	HandlerType: (*blockingGRPCServiceServer)(nil),
	Methods: []grpc.MethodDesc{{
		MethodName: "Block",
		Handler: func(server interface{}, ctx context.Context, decode func(interface{}) error, _ grpc.UnaryServerInterceptor) (interface{}, error) {
			request := &emptypb.Empty{}
			if err := decode(request); err != nil {
				return nil, err
			}
			return server.(blockingGRPCServiceServer).Block(ctx, request)
		},
	}},
}

type runtimeStateExternalDB struct {
	testExternalDB
	strictErr           error
	eventReceiptMetrics db.EventReceiptMetrics
}

type fakeImagesClient struct{}

func (fakeImagesClient) DescribeImage(context.Context, *p2pproto.DescribeImageRequest, ...grpc.CallOption) (*p2pproto.DescribeImageResponse, error) {
	return nil, nil
}

func (fakeImagesClient) GetImageContent(context.Context, *p2pproto.GetImageContentRequest, ...grpc.CallOption) (*p2pproto.GetImageContentResponse, error) {
	return nil, nil
}

func (fakeImagesClient) GetImageBlob(context.Context, *p2pproto.GetImageBlobRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[p2pproto.GetImageBlobResponse], error) {
	return nil, nil
}

func (f testExternalDB) AddPeer(string, *grpc.ClientConn) error { return nil }
func (f testExternalDB) RemovePeer(string) error                { return nil }
func (f testExternalDB) GetAllCommits() ([]db.Commit, error)    { return nil, nil }
func (f testExternalDB) GetCommitDiff(context.Context, string, string) (db.CommitDiff, error) {
	return db.CommitDiff{}, nil
}
func (f testExternalDB) ExecuteSQL(context.Context, string, int) (db.SQLResult, error) {
	return db.SQLResult{}, nil
}
func (f testExternalDB) GetLastCommit(string) (db.Commit, error) { return db.Commit{}, nil }
func (f testExternalDB) CatchUpCheckpoint(context.Context, string) error {
	return nil
}
func (f testExternalDB) CatchUpCheckpointStrict(context.Context, string) error {
	return nil
}
func (f testExternalDB) InitFromPeerContext(context.Context, string, []string) error { return nil }
func (f testExternalDB) EnableGRPCServers(*grpc.Server) error {
	return nil
}
func (f testExternalDB) Initialized() bool { return f.initialized }
func (f testExternalDB) RepositoryReadiness() db.RepositoryReadiness {
	readiness := f.readiness
	if f.initialized {
		readiness.Initialized = true
	}
	return readiness
}

func (f runtimeStateExternalDB) CatchUpCheckpointStrict(context.Context, string) error {
	return f.strictErr
}

func (f runtimeStateExternalDB) SwarmionStatus() (swarmionapp.Status, bool) {
	return swarmionapp.Status{}, true
}

func (f runtimeStateExternalDB) SwarmionCompatibility(context.Context) ([]swarmionapp.ManifestCompatibility, error) {
	return nil, nil
}

func (f runtimeStateExternalDB) SwarmionPeerStatus(context.Context) ([]swarmionapp.PeerStatus, error) {
	return nil, nil
}

func (f runtimeStateExternalDB) SwarmionContentSyncTrace() ([]string, bool) {
	return nil, false
}

func (f runtimeStateExternalDB) EventReceiptMetrics() db.EventReceiptMetrics {
	return f.eventReceiptMetrics
}
