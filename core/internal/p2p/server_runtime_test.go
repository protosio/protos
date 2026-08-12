package p2p

import (
	"testing"
	"time"

	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
	"github.com/protosio/protos/internal/p2p/proto"
)

func TestAddKnownRuntimePeerStatusesAddsDbPeersAndSelf(t *testing.T) {
	t.Parallel()

	state := &proto.RuntimeState{
		PeerId:                 "local-peer",
		ConnectedPeers:         []string{"connected-peer"},
		RoutedPeers:            []string{"connected-peer"},
		ParticipatingPeers:     []string{"provider-peer"},
		LogicalPeers:           []string{"database-peer"},
		PhysicalConnectedPeers: []string{"provider-peer"},
		StateProviders:         []string{"provider-peer"},
		PeerStatuses: []*proto.RuntimePeerStatus{
			{PeerId: "connected-peer", Connected: true, Dialable: true, Routed: true},
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
	if self := statuses["local-peer"]; self == nil || self.GetConnected() || self.GetDialable() || self.GetRouted() || self.GetPhysicalConnected() || !self.GetCompatible() || self.GetReason() != "self" {
		t.Fatalf("self status = %#v, want compatible self row without inferred reachability", self)
	}
	if databasePeer := statuses["database-peer"]; databasePeer == nil || databasePeer.GetConnected() || databasePeer.GetDialable() || databasePeer.GetStateProvider() || databasePeer.GetReason() != "known database peer" {
		t.Fatalf("database peer status = %#v, want inert known database row", databasePeer)
	}
	if provider := statuses["provider-peer"]; provider == nil || !provider.GetStateProvider() {
		t.Fatalf("provider status = %#v, want state_provider=true", provider)
	} else if !provider.GetParticipating() || !provider.GetPhysicalConnected() {
		t.Fatalf("provider transport planes = %#v, want participating and physical", provider)
	}
	if connected := statuses["connected-peer"]; connected == nil || !connected.GetRouted() || !connected.GetConnected() || !connected.GetDialable() {
		t.Fatalf("connected status = %#v, want routed and legacy aliases", connected)
	}
	if databasePeer := statuses["database-peer"]; databasePeer == nil || !databasePeer.GetLogical() {
		t.Fatalf("database peer status = %#v, want logical plane", databasePeer)
	}
}

func TestFilterRuntimePeerSurfaceRemovesUnknownCachedPeers(t *testing.T) {
	t.Parallel()

	state := &proto.RuntimeState{
		PeerId:                 "local-peer",
		StateProviders:         []string{"provider-peer", "deleted-peer", "provider-peer"},
		ConnectedPeers:         []string{"provider-peer", "deleted-peer", "provider-peer"},
		RoutedPeers:            []string{"provider-peer", "deleted-peer", "provider-peer"},
		ParticipatingPeers:     []string{"provider-peer", "deleted-peer", "provider-peer"},
		LogicalPeers:           []string{"provider-peer", "deleted-peer", "provider-peer"},
		PhysicalConnectedPeers: []string{"provider-peer", "deleted-peer", "provider-peer"},
		PeerStatuses: []*proto.RuntimePeerStatus{
			{PeerId: "provider-peer", Connected: true, Dialable: true, StateProvider: true, Routed: true, Participating: true, Logical: true, PhysicalConnected: true, LastRoutedAtUnixNano: 1234},
			{PeerId: "provider-peer", Connected: true, Dialable: true, StateProvider: true, Routed: true, Participating: true, Logical: true, PhysicalConnected: true},
			{PeerId: "deleted-peer", Connected: true, Dialable: true, StateProvider: true, Routed: true, Participating: true, Logical: true, PhysicalConnected: true},
		},
		Compatibility: []*proto.RuntimeCompatibility{
			{PeerId: "provider-peer", Compatible: true},
			{PeerId: "provider-peer", Compatible: true},
			{PeerId: "deleted-peer", Blocking: true},
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
	for name, got := range map[string][]string{
		"routed":        state.GetRoutedPeers(),
		"participating": state.GetParticipatingPeers(),
		"logical":       state.GetLogicalPeers(),
		"physical":      state.GetPhysicalConnectedPeers(),
	} {
		if len(got) != 1 || got[0] != "provider-peer" {
			t.Fatalf("%s peers = %#v, want provider-peer", name, got)
		}
	}
	if !state.GetPeerStatuses()[0].GetStateProvider() {
		t.Fatal("known provider lost provider flag")
	}
	if len(state.GetPeerStatuses()) != 1 {
		t.Fatalf("peer statuses count = %d, want 1", len(state.GetPeerStatuses()))
	}
	status := state.GetPeerStatuses()[0]
	if !status.GetRouted() || !status.GetParticipating() || !status.GetLogical() || !status.GetPhysicalConnected() || !status.GetConnected() || !status.GetDialable() || status.GetLastRoutedAtUnixNano() != 1234 {
		t.Fatalf("filtered transport status = %#v, want every explicit plane and timestamp retained", status)
	}
	if got := state.GetCompatibility(); len(got) != 1 || got[0].GetPeerId() != "provider-peer" {
		t.Fatalf("compatibility = %#v, want only provider peer", got)
	}
}

func TestRuntimePeerStatusToP2PProtoPreservesExplicitPlanes(t *testing.T) {
	t.Parallel()
	lastRoutedAt := time.Unix(123, 456)
	status := runtimePeerStatusToP2PProto(swarmionapp.PeerStatus{
		PeerID:        "peer-a",
		Routed:        true,
		Participating: true,
		Logical:       true,
		StateProvider: true,
		Compatible:    true,
		Addresses:     []string{"/ip4/192.0.2.10/tcp/1111"},
		LastRoutedAt:  lastRoutedAt,
	})
	if !status.GetRouted() || !status.GetParticipating() || !status.GetLogical() || !status.GetConnected() || !status.GetDialable() {
		t.Fatalf("Swarmion transport planes or legacy aliases were not preserved: %#v", status)
	}
	if status.GetPhysicalConnected() {
		t.Fatalf("Swarmion status inferred physical connectivity: %#v", status)
	}
	if got := status.GetLastRoutedAtUnixNano(); got != lastRoutedAt.UnixNano() {
		t.Fatalf("last routed timestamp = %d, want %d", got, lastRoutedAt.UnixNano())
	}
}

func TestSynchronizeRuntimePeerStatusPlanesUsesOnlyPhysicalHostSurface(t *testing.T) {
	t.Parallel()
	state := &proto.RuntimeState{
		PhysicalConnectedPeers: []string{"peer-a"},
		PeerStatuses: []*proto.RuntimePeerStatus{{
			PeerId:        "peer-a",
			Participating: true,
			Logical:       true,
			Connected:     true,
			Dialable:      true,
		}},
	}
	synchronizeRuntimePeerStatusPlanes(state)
	status := state.GetPeerStatuses()[0]
	if !status.GetPhysicalConnected() || status.GetRouted() || status.GetConnected() || status.GetDialable() || !status.GetParticipating() || !status.GetLogical() {
		t.Fatalf("synchronized status = %#v, want physical-only host mapping with Swarmion planes unchanged", status)
	}
}
