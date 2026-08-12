//go:build darwin

package main

import (
	"reflect"
	"testing"

	pbApic "github.com/protosio/protos/apic/proto"
)

func TestRecordRuntimeSnapshotPreservesTransportPlanes(t *testing.T) {
	summary := &mixedCloudRunSummary{}
	summary.recordRuntimeSnapshot("mesh-ready", "local", &pbApic.RuntimeState{
		ConnectedPeers:         []string{"legacy"},
		RoutedPeers:            []string{"routed-b", "routed-a"},
		ParticipatingPeers:     []string{"participating"},
		LogicalPeers:           []string{"logical"},
		LogicalPeerTarget:      8,
		PhysicalConnectedPeers: []string{"physical-b", "physical-a"},
		PeerStatuses: []*pbApic.RuntimePeerStatus{{
			PeerId:               "peer-a",
			Routed:               true,
			Participating:        true,
			Logical:              true,
			PhysicalConnected:    true,
			LastRoutedAtUnixNano: 1234,
		}},
	}, nil)

	if len(summary.RuntimeSnapshots) != 1 {
		t.Fatalf("runtime snapshots = %d, want 1", len(summary.RuntimeSnapshots))
	}
	got := summary.RuntimeSnapshots[0]
	if !reflect.DeepEqual(got.RoutedPeers, []string{"routed-a", "routed-b"}) {
		t.Fatalf("routed peers = %v", got.RoutedPeers)
	}
	if !reflect.DeepEqual(got.ParticipatingPeers, []string{"participating"}) {
		t.Fatalf("participating peers = %v", got.ParticipatingPeers)
	}
	if !reflect.DeepEqual(got.LogicalPeers, []string{"logical"}) || got.LogicalPeerTarget != 8 {
		t.Fatalf("logical view = %v target %d", got.LogicalPeers, got.LogicalPeerTarget)
	}
	if !reflect.DeepEqual(got.PhysicalConnectedPeers, []string{"physical-a", "physical-b"}) {
		t.Fatalf("physical peers = %v", got.PhysicalConnectedPeers)
	}
	if len(got.PeerStatuses) != 1 {
		t.Fatalf("peer statuses = %d, want 1", len(got.PeerStatuses))
	}
	status := got.PeerStatuses[0]
	if !status.Routed || !status.Participating || !status.Logical || !status.PhysicalConnected || status.LastRoutedAtUnixNano != 1234 {
		t.Fatalf("transport peer status = %+v", status)
	}
}
