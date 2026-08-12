// Package testswarmion provides caller-owned transport fixtures for backend
// tests that open a real Swarmion database runtime.
//
// The fixture intentionally uses Protos' production libp2p Link adapter. It
// keeps the physical host alive until test cleanup so closing and reopening a
// database exercises Swarmion's borrowed-transport lifecycle contract.
package testswarmion

import (
	"context"
	"testing"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	libp2phost "github.com/libp2p/go-libp2p/core/host"
	libp2ppeer "github.com/libp2p/go-libp2p/core/peer"

	"github.com/nustiueudinastea/swarmion/transports"
	"github.com/protosio/protos/internal/swarmionlink"
)

const connectTimeout = 10 * time.Second

// Signer is the subset of the backend signer contract required to make the
// application-owned libp2p host use the same identity as Swarmion.
type Signer interface {
	Private() []byte
	GetID() string
}

// Fixture owns the physical test host and lends Link to Swarmion. Tests may
// close and reopen database runtimes with the same Link; only test cleanup
// closes Host.
type Fixture struct {
	Host libp2phost.Host
	Link transports.Link
}

// New creates a caller-owned loopback libp2p host and wraps it with Protos'
// production borrowed-link implementation.
func New(t testing.TB, signer Signer) *Fixture {
	t.Helper()
	if signer == nil {
		t.Fatal("create Swarmion test fixture with nil signer")
	}
	privateKey, err := libp2pcrypto.UnmarshalEd25519PrivateKey(signer.Private())
	if err != nil {
		t.Fatalf("decode Swarmion test fixture identity: %v", err)
	}
	host, err := libp2p.New(
		libp2p.Identity(privateKey),
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	if err != nil {
		t.Fatalf("create caller-owned Swarmion test host: %v", err)
	}
	t.Cleanup(func() {
		if err := host.Close(); err != nil {
			t.Errorf("close caller-owned Swarmion test host: %v", err)
		}
	})
	link, err := swarmionlink.New(host)
	if err != nil {
		_ = host.Close()
		t.Fatalf("wrap caller-owned Swarmion test host: %v", err)
	}
	if got, want := string(link.LocalPeer()), signer.GetID(); got != want {
		_ = host.Close()
		t.Fatalf("caller-owned Swarmion test peer ID = %q, want %q", got, want)
	}
	return &Fixture{Host: host, Link: link}
}

// NewBorrowedLink creates a fixture and returns its borrowed Link. The
// physical host remains owned by the test and is closed by test cleanup.
func NewBorrowedLink(t testing.TB, signer Signer) transports.Link {
	t.Helper()
	return New(t, signer).Link
}

// Connect establishes an application-owned physical route from local to
// remote. The resulting libp2p connection is duplex; Swarmion does not need or
// own a reverse dial.
func Connect(t testing.TB, local, remote *Fixture) {
	t.Helper()
	if local == nil || local.Host == nil || remote == nil || remote.Host == nil {
		t.Fatal("connect nil Swarmion test fixture")
	}
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()
	if err := local.Host.Connect(ctx, libp2ppeer.AddrInfo{
		ID:    remote.Host.ID(),
		Addrs: remote.Host.Addrs(),
	}); err != nil {
		t.Fatalf("connect caller-owned Swarmion test hosts: %v", err)
	}
}
