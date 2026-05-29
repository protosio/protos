package apic

import (
	"testing"
	"time"

	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/protosio/protos/internal/provisioners"
)

func TestBaseInstanceDeployFieldsFollowDeployRequestDescriptor(t *testing.T) {
	t.Parallel()

	fields := baseInstanceDeployFields()
	descriptor := (&pbApic.DeployInstanceRequest{}).ProtoReflect().Descriptor()
	if len(fields) != descriptor.Fields().Len() {
		t.Fatalf("field count = %d, want %d", len(fields), descriptor.Fields().Len())
	}
	for i, field := range fields {
		want := string(descriptor.Fields().Get(i).Name())
		if field.GetName() != want {
			t.Fatalf("field[%d] = %q, want %q", i, field.GetName(), want)
		}
	}
}

func TestInstanceDeployImageLessPrefersNewestUpdatedAt(t *testing.T) {
	t.Parallel()

	oldImage := provisioners.ImageInfo{Name: "z-old", UpdatedAt: time.Unix(100, 0)}
	newImage := provisioners.ImageInfo{Name: "a-new", UpdatedAt: time.Unix(200, 0)}
	if !instanceDeployImageLess(newImage, oldImage) {
		t.Fatal("newer image should sort before older image")
	}
	if instanceDeployImageLess(oldImage, newImage) {
		t.Fatal("older image should not sort before newer image")
	}
	if !instanceDeployImageLess(provisioners.ImageInfo{Name: "a"}, provisioners.ImageInfo{Name: "b"}) {
		t.Fatal("images without update times should sort by name")
	}
}

func TestIsPublicExitIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{name: "public IPv4", ip: "5.161.52.86", want: true},
		{name: "public IPv6", ip: "2606:4700:4700::1111", want: true},
		{name: "private IPv4", ip: "10.0.0.1", want: false},
		{name: "carrier NAT IPv4", ip: "100.64.0.1", want: false},
		{name: "documentation IPv4", ip: "203.0.113.10", want: false},
		{name: "documentation IPv6", ip: "2001:db8::1", want: false},
		{name: "loopback", ip: "127.0.0.1", want: false},
		{name: "invalid", ip: "not-an-ip", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isPublicExitIP(tt.ip); got != tt.want {
				t.Fatalf("isPublicExitIP(%q) = %t, want %t", tt.ip, got, tt.want)
			}
		})
	}
}

func TestAddKnownRuntimePeerStatusesAddsDbPeersAndSelf(t *testing.T) {
	t.Parallel()

	state := &pbApic.RuntimeState{
		PeerId:             "local-peer",
		ConnectedPeers:     []string{"connected-peer"},
		ActiveWitnessIds:   []string{"witness-peer"},
		EligibleWitnessIds: []string{"eligible-peer"},
		StateProviders:     []string{"provider-peer"},
		PeerStatuses: []*pbApic.RuntimePeerStatus{
			{PeerId: "connected-peer", Connected: true},
		},
	}

	addKnownRuntimePeerStatuses(state, map[string]struct{}{
		"connected-peer": {},
		"witness-peer":   {},
		"eligible-peer":  {},
		"provider-peer":  {},
	})

	statuses := map[string]*pbApic.RuntimePeerStatus{}
	for _, status := range state.GetPeerStatuses() {
		statuses[status.GetPeerId()] = status
	}

	if len(statuses) != 5 {
		t.Fatalf("peer statuses count = %d, want 5: %#v", len(statuses), statuses)
	}
	if self := statuses["local-peer"]; self == nil || !self.GetConnected() || !self.GetDialable() || !self.GetCompatible() || self.GetReason() != "self" {
		t.Fatalf("self status = %#v, want connected dialable compatible self row", self)
	}
	if witness := statuses["witness-peer"]; witness == nil || !witness.GetWitness() {
		t.Fatalf("witness status = %#v, want witness=true", witness)
	}
	if eligible := statuses["eligible-peer"]; eligible == nil || !eligible.GetEligibleWitness() {
		t.Fatalf("eligible status = %#v, want eligible_witness=true", eligible)
	}
	if provider := statuses["provider-peer"]; provider == nil || !provider.GetStateProvider() {
		t.Fatalf("provider status = %#v, want state_provider=true", provider)
	}
	if connected := statuses["connected-peer"]; connected == nil || !connected.GetConnected() {
		t.Fatalf("connected status = %#v, want existing connected status preserved", connected)
	}
}
