package network

import (
	"context"
	"testing"
	"time"

	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/testswarmion"
)

func TestExitRouteWritesReturnSinglePeerAvailabilityConfirmation(t *testing.T) {
	store := newTestExitRouteDB(t)
	deviceID := db.MustNewUUIDv7()
	instanceID := db.MustNewUUIDv7()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	route, created, err := SetExitRouteWithConfirmationContext(ctx, store, deviceID, instanceID, "1.1.1.1", nil)
	if err != nil {
		t.Fatalf("create exit route: %v", err)
	}
	assertSinglePeerWriteConfirmation(t, created)

	updatedRoute, updated, err := SetExitRouteWithConfirmationContext(ctx, store, deviceID, instanceID, "8.8.8.8", []string{"10.0.0.0/8"})
	if err != nil {
		t.Fatalf("update exit route: %v", err)
	}
	assertSinglePeerWriteConfirmation(t, updated)
	if updatedRoute.ID != route.ID {
		t.Fatalf("updated route ID = %s, want %s", updatedRoute.ID, route.ID)
	}

	removed, err := ClearExitRouteWithConfirmationContext(ctx, store, deviceID)
	if err != nil {
		t.Fatalf("clear exit route: %v", err)
	}
	assertSinglePeerWriteConfirmation(t, removed)
	if err := ctx.Err(); err != nil {
		t.Fatalf("single-peer route writes did not return promptly: %v", err)
	}

	noChange, err := ClearExitRouteWithConfirmationContext(ctx, store, deviceID)
	if err != nil {
		t.Fatalf("clear missing exit route: %v", err)
	}
	if noChange.Stage != db.PublishedWriteConfirmationNoChange {
		t.Fatalf("missing-route confirmation stage = %q, want %q", noChange.Stage, db.PublishedWriteConfirmationNoChange)
	}
}

func assertSinglePeerWriteConfirmation(t *testing.T, confirmation db.PublishedWriteConfirmation) {
	t.Helper()
	if confirmation.Stage != db.PublishedWriteConfirmationLocalAccepted {
		t.Fatalf("confirmation stage = %q, want %q", confirmation.Stage, db.PublishedWriteConfirmationLocalAccepted)
	}
	if !confirmation.AvailabilityPending {
		t.Fatal("single-peer write should preserve pending other-peer availability")
	}
	if !confirmation.Receipt.HasExactEventIdentity() {
		t.Fatalf("confirmation did not preserve exact receipt: %+v", confirmation.Receipt)
	}
}

func newTestExitRouteDB(t *testing.T) *db.DB {
	t.Helper()
	cfg := config.Get()
	previousP2PPort := cfg.P2PPort
	cfg.P2PPort = 0
	t.Cleanup(func() {
		cfg.P2PPort = previousP2PPort
	})

	workDir := t.TempDir()
	key, err := pcrypto.GetLocalKey(workDir)
	if err != nil {
		t.Fatalf("get local key: %v", err)
	}
	store, err := db.Open(workDir, "protos_test", key, testswarmion.NewBorrowedLink(t, key))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	if err := store.Init(); err != nil {
		t.Fatalf("init db: %v", err)
	}
	return store
}

func TestNormalizeDNSServer(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "empty", input: "", want: ""},
		{name: "ipv4 default port", input: "8.8.8.8", want: "8.8.8.8:53"},
		{name: "ipv4 explicit port", input: "1.1.1.1:5353", want: "1.1.1.1:5353"},
		{name: "ipv6 default port", input: "2001:4860:4860::8888", want: "[2001:4860:4860::8888]:53"},
		{name: "ipv6 explicit port", input: "[2001:4860:4860::8888]:5353", want: "[2001:4860:4860::8888]:5353"},
		{name: "hostname rejected", input: "dns.google:53", wantErr: true},
		{name: "bad port rejected", input: "8.8.8.8:99999", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NormalizeDNSServer(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("NormalizeDNSServer(%q) succeeded, want error", test.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeDNSServer(%q): %v", test.input, err)
			}
			if got != test.want {
				t.Fatalf("NormalizeDNSServer(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestNormalizeExitRouteCIDRs(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    []string
		wantErr bool
	}{
		{name: "default full tunnel", input: nil, want: []string{"0.0.0.0/0", "::/0"}},
		{name: "single host", input: []string{"1.2.3.4/32"}, want: []string{"1.2.3.4/32"}},
		{name: "masked network", input: []string{"1.2.3.4/24"}, want: []string{"1.2.3.0/24"}},
		{name: "dedupe", input: []string{"1.2.3.4/32", "1.2.3.4/32"}, want: []string{"1.2.3.4/32"}},
		{name: "reject invalid", input: []string{"1.2.3.4"}, wantErr: true},
		{name: "reject empty", input: []string{""}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NormalizeExitRouteCIDRs(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("NormalizeExitRouteCIDRs(%v) succeeded, want error", test.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeExitRouteCIDRs(%v): %v", test.input, err)
			}
			if len(got) != len(test.want) {
				t.Fatalf("NormalizeExitRouteCIDRs(%v) = %v, want %v", test.input, got, test.want)
			}
			for i := range test.want {
				if got[i] != test.want[i] {
					t.Fatalf("NormalizeExitRouteCIDRs(%v) = %v, want %v", test.input, got, test.want)
				}
			}
		})
	}
}

func TestRouteCIDRsUseFullTunnel(t *testing.T) {
	if !RouteCIDRsUseFullTunnel(nil) {
		t.Fatalf("nil CIDRs should default to full tunnel")
	}
	if !RouteCIDRsUseFullTunnel([]string{"0.0.0.0/0"}) {
		t.Fatalf("0.0.0.0/0 should use full tunnel")
	}
	if RouteCIDRsUseFullTunnel([]string{"1.2.3.4/32"}) {
		t.Fatalf("single host route should not use full tunnel")
	}
}
