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
		RoutedPeers:            []string{"routed-peer"},
		ParticipatingPeers:     []string{"provider-peer"},
		LogicalPeers:           []string{"database-peer"},
		PhysicalConnectedPeers: []string{"provider-peer"},
		StateProviders:         []string{"provider-peer"},
		PeerStatuses: []*proto.RuntimePeerStatus{
			{PeerId: "routed-peer", Routed: true},
		},
	}

	addKnownRuntimePeerStatuses(state, map[string]struct{}{
		"routed-peer":   {},
		"database-peer": {},
		"provider-peer": {},
	})

	statuses := map[string]*proto.RuntimePeerStatus{}
	for _, status := range state.GetPeerStatuses() {
		statuses[status.GetPeerId()] = status
	}

	if len(statuses) != 4 {
		t.Fatalf("peer statuses count = %d, want 4: %#v", len(statuses), statuses)
	}
	if self := statuses["local-peer"]; self == nil || self.GetRouted() || self.GetPhysicalConnected() || !self.GetCompatible() || self.GetReason() != "self" {
		t.Fatalf("self status = %#v, want compatible self row without inferred reachability", self)
	}
	if databasePeer := statuses["database-peer"]; databasePeer == nil || databasePeer.GetRouted() || databasePeer.GetStateProvider() || databasePeer.GetReason() != "known database peer" {
		t.Fatalf("database peer status = %#v, want inert known database row", databasePeer)
	}
	if provider := statuses["provider-peer"]; provider == nil || !provider.GetStateProvider() {
		t.Fatalf("provider status = %#v, want state_provider=true", provider)
	} else if !provider.GetParticipating() || !provider.GetPhysicalConnected() {
		t.Fatalf("provider transport planes = %#v, want participating and physical", provider)
	}
	if routed := statuses["routed-peer"]; routed == nil || !routed.GetRouted() {
		t.Fatalf("routed status = %#v, want routed", routed)
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
		RoutedPeers:            []string{"provider-peer", "deleted-peer", "provider-peer"},
		ParticipatingPeers:     []string{"provider-peer", "deleted-peer", "provider-peer"},
		LogicalPeers:           []string{"provider-peer", "deleted-peer", "provider-peer"},
		PhysicalConnectedPeers: []string{"provider-peer", "deleted-peer", "provider-peer"},
		PeerStatuses: []*proto.RuntimePeerStatus{
			{PeerId: "provider-peer", StateProvider: true, Routed: true, Participating: true, Logical: true, PhysicalConnected: true, LastRoutedAtUnixNano: 1234},
			{PeerId: "provider-peer", StateProvider: true, Routed: true, Participating: true, Logical: true, PhysicalConnected: true},
			{PeerId: "deleted-peer", StateProvider: true, Routed: true, Participating: true, Logical: true, PhysicalConnected: true},
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
	if !status.GetRouted() || !status.GetParticipating() || !status.GetLogical() || !status.GetPhysicalConnected() || status.GetLastRoutedAtUnixNano() != 1234 {
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
	if !status.GetRouted() || !status.GetParticipating() || !status.GetLogical() {
		t.Fatalf("Swarmion transport planes were not preserved: %#v", status)
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
		}},
	}
	synchronizeRuntimePeerStatusPlanes(state)
	status := state.GetPeerStatuses()[0]
	if !status.GetPhysicalConnected() || status.GetRouted() || !status.GetParticipating() || !status.GetLogical() {
		t.Fatalf("synchronized status = %#v, want physical-only host mapping with Swarmion planes unchanged", status)
	}
}
