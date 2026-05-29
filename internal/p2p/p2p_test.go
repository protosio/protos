package p2p

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/util"
	"google.golang.org/grpc"
)

type testMachine struct {
	id         string
	publicKey  string
	publicIP   string
	internalIP string
}

func (m testMachine) GetID() string        { return m.id }
func (m testMachine) GetPublicKey() string { return m.publicKey }
func (m testMachine) GetPublicIP() string  { return m.publicIP }
func (m testMachine) GetName() string      { return m.id }
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
		machines: util.NewMap[string, Machine](),
		clients:  util.NewMap[string, *Client](),
	}
}

func TestDestinationStringsIncludeWireGuardIPv6WithoutPublicIP(t *testing.T) {
	machine, peerID, internalIP := newTestMachine(t, "")
	p2p := newTestP2P()

	got := p2p.destinationStrings(peerID, machine)
	want := fmt.Sprintf(destinationIPv6Template, internalIP, config.Get().P2PPort, peerID)
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
		fmt.Sprintf(destinationIPv6Template, machine.internalIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationIPv6Template, derivedInternalIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationStringTemplate, machine.publicIP, config.Get().P2PPort, peerID),
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
		fmt.Sprintf(destinationIPv6Template, machine.internalIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationIPv6Template, derivedInternalIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationStringTemplate, machine.publicIP, config.Get().P2PPort, peerID),
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
	defer localStop()
	remoteStop, err := remote.StartServer()
	if err != nil {
		t.Fatalf("start remote server: %v", err)
	}
	defer remoteStop()

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

type testExternalDB struct {
	initialized bool
}

func (f testExternalDB) AddPeer(string, *grpc.ClientConn) error { return nil }
func (f testExternalDB) RemovePeer(string) error                { return nil }
func (f testExternalDB) GetAllCommits() ([]db.Commit, error)    { return nil, nil }
func (f testExternalDB) ExecSQLAndCommit(string, string) (string, error) {
	return "", nil
}
func (f testExternalDB) GetLastCommit(string) (db.Commit, error) { return db.Commit{}, nil }
func (f testExternalDB) CatchUpFinalized(context.Context, string) error {
	return nil
}
func (f testExternalDB) InitFromPeer(string, []string) error { return nil }
func (f testExternalDB) EnableGRPCServers(*grpc.Server) error {
	return nil
}
func (f testExternalDB) Initialized() bool { return f.initialized }
