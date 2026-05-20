package p2p

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"slices"
	"testing"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/util"
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
