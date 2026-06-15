package p2p

import (
	"testing"

	"github.com/protosio/protos/internal/p2p/proto"
)

func TestAddKnownRuntimePeerStatusesAddsDbPeersAndSelf(t *testing.T) {
	t.Parallel()

	state := &proto.RuntimeState{
		PeerId:         "local-peer",
		ConnectedPeers: []string{"connected-peer"},
		StateProviders: []string{"provider-peer"},
		PeerStatuses: []*proto.RuntimePeerStatus{
			{PeerId: "connected-peer", Connected: true},
		},
	}

	addKnownRuntimePeerStatuses(state, map[string]struct{}{
		"connected-peer": {},
		"database-peer":  {},
		"provider-peer":  {},
	})

	statuses := map[string]*proto.RuntimePeerStatus{}
	for _, status := range state.GetPeerStatuses() {
		statuses[status.GetPeerId()] = status
	}

	if len(statuses) != 4 {
		t.Fatalf("peer statuses count = %d, want 4: %#v", len(statuses), statuses)
	}
	if self := statuses["local-peer"]; self == nil || !self.GetConnected() || !self.GetDialable() || !self.GetCompatible() || self.GetReason() != "self" {
		t.Fatalf("self status = %#v, want connected dialable compatible self row", self)
	}
	if databasePeer := statuses["database-peer"]; databasePeer == nil || databasePeer.GetConnected() || databasePeer.GetDialable() || databasePeer.GetStateProvider() || databasePeer.GetReason() != "known database peer" {
		t.Fatalf("database peer status = %#v, want inert known database row", databasePeer)
	}
	if provider := statuses["provider-peer"]; provider == nil || !provider.GetStateProvider() {
		t.Fatalf("provider status = %#v, want state_provider=true", provider)
	}
	if connected := statuses["connected-peer"]; connected == nil || !connected.GetConnected() {
		t.Fatalf("connected status = %#v, want existing connected status preserved", connected)
	}
}

func TestFilterRuntimePeerSurfaceRemovesUnknownCachedPeers(t *testing.T) {
	t.Parallel()

	state := &proto.RuntimeState{
		PeerId:         "local-peer",
		StateProviders: []string{"provider-peer", "deleted-peer", "provider-peer"},
		ConnectedPeers: []string{"provider-peer", "deleted-peer", "provider-peer"},
		PeerStatuses: []*proto.RuntimePeerStatus{
			{PeerId: "provider-peer", Connected: true, Dialable: true, StateProvider: true},
			{PeerId: "deleted-peer", Connected: true, Dialable: true, StateProvider: true},
		},
	}

	filterRuntimePeerSurface(state, map[string]struct{}{
		"provider-peer": {},
	})

	if got, want := state.GetStateProviders(), []string{"provider-peer"}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("state providers = %#v, want %#v", got, want)
	}
	if got, want := state.GetConnectedPeers(), []string{"provider-peer"}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("connected peers = %#v, want %#v", got, want)
	}
	if !state.GetPeerStatuses()[0].GetStateProvider() {
		t.Fatal("known provider lost provider flag")
	}
	if len(state.GetPeerStatuses()) != 1 {
		t.Fatalf("peer statuses count = %d, want 1", len(state.GetPeerStatuses()))
	}
}
