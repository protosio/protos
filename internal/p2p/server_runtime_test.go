package p2p

import (
	"testing"

	"github.com/protosio/protos/internal/p2p/proto"
)

func TestAddKnownRuntimePeerStatusesAddsDbPeersAndSelf(t *testing.T) {
	t.Parallel()

	state := &proto.RuntimeState{
		PeerId:             "local-peer",
		ConnectedPeers:     []string{"connected-peer"},
		ActiveWitnessIds:   []string{"witness-peer"},
		EligibleWitnessIds: []string{"eligible-peer"},
		StateProviders:     []string{"provider-peer"},
		PeerStatuses: []*proto.RuntimePeerStatus{
			{PeerId: "connected-peer", Connected: true},
		},
	}

	addKnownRuntimePeerStatuses(state, map[string]struct{}{
		"connected-peer": {},
		"witness-peer":   {},
		"eligible-peer":  {},
		"provider-peer":  {},
	})

	statuses := map[string]*proto.RuntimePeerStatus{}
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
