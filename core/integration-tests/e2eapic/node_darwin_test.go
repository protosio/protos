//go:build darwin

package e2eapic

import (
	"strings"
	"testing"

	pbApic "github.com/protosio/protos/apic/proto"
	"google.golang.org/protobuf/proto"
)

func TestTerminalStatusIgnoresRunningProgressDetails(t *testing.T) {
	if terminalStatus("running: initializing VM: failed to retrieve provisioner") {
		t.Fatal("running progress detail was treated as terminal")
	}
	if !terminalStatus("failed: provisioner error") {
		t.Fatal("failed status prefix was not treated as terminal")
	}
	if !terminalStatus("cancelled") {
		t.Fatal("cancelled status was not treated as terminal")
	}
}

func TestCheckpointedRuntimeRootRequiresConvergedRoots(t *testing.T) {
	state := &pbApic.RuntimeState{
		CheckpointRootHash:  "root",
		TentativeRootHash:   "root",
		DurableMainRootHash: "root",
		FatalState:          "none",
	}
	root, err := checkpointedRuntimeRoot(state)
	if err != nil {
		t.Fatalf("checkpointedRuntimeRoot() error = %v", err)
	}
	if root != "root" {
		t.Fatalf("checkpointedRuntimeRoot() = %q, want root", root)
	}

	state.DurableMainRootHash = "other"
	_, err = checkpointedRuntimeRoot(state)
	if err == nil || !strings.Contains(err.Error(), "durable root") {
		t.Fatalf("checkpointedRuntimeRoot() error = %v, want durable root mismatch", err)
	}
}

func TestRuntimePeerTransportReadyRequiresExplicitPlanes(t *testing.T) {
	assertReady := func(t *testing.T, state *pbApic.RuntimeState, want bool, wantMissing []string) {
		t.Helper()
		got, missing := runtimePeerTransportReady(state, "peer-a")
		if got != want || strings.Join(missing, "\x00") != strings.Join(wantMissing, "\x00") {
			t.Fatalf("runtimePeerTransportReady() = (%t, %v), want (%t, %v)", got, missing, want, wantMissing)
		}
	}

	assertReady(t, nil, false, []string{"physical", "routed", "participating"})
	assertReady(t, &pbApic.RuntimeState{
		PhysicalConnectedPeers: []string{"peer-a"},
		RoutedPeers:            []string{"peer-a"},
		ParticipatingPeers:     []string{"peer-a"},
	}, true, nil)
	assertReady(t, &pbApic.RuntimeState{
		PhysicalConnectedPeers: []string{"peer-a"},
		RoutedPeers:            []string{"peer-a"},
	}, false, []string{"participating"})
	assertReady(t, &pbApic.RuntimeState{
		LogicalPeers: []string{"peer-a"},
		PeerStatuses: []*pbApic.RuntimePeerStatus{{
			PeerId:  "peer-a",
			Logical: true,
		}},
	}, false, []string{"physical", "routed", "participating"})
	assertReady(t, &pbApic.RuntimeState{PeerStatuses: []*pbApic.RuntimePeerStatus{{
		PeerId:            "peer-a",
		PhysicalConnected: true,
		Routed:            true,
		Participating:     true,
	}}}, true, nil)
}

func TestRuntimeTransportPlanesProtoRoundTrip(t *testing.T) {
	want := &pbApic.RuntimeState{
		RoutedPeers:            []string{"peer-a"},
		ParticipatingPeers:     []string{"peer-a"},
		LogicalPeers:           []string{"peer-a"},
		LogicalPeerTarget:      8,
		PhysicalConnectedPeers: []string{"peer-a", "peer-control"},
		PeerStatuses: []*pbApic.RuntimePeerStatus{{
			PeerId:               "peer-a",
			Routed:               true,
			Participating:        true,
			Logical:              true,
			PhysicalConnected:    true,
			LastRoutedAtUnixNano: 1234,
		}},
	}

	wire, err := proto.Marshal(want)
	if err != nil {
		t.Fatalf("marshal runtime state: %v", err)
	}
	got := new(pbApic.RuntimeState)
	if err := proto.Unmarshal(wire, got); err != nil {
		t.Fatalf("unmarshal runtime state: %v", err)
	}
	if !proto.Equal(got, want) {
		t.Fatalf("round trip = %v, want %v", got, want)
	}
}

func TestRuntimeStateSummaryIncludesEventReceiptContentDissentObservations(t *testing.T) {
	summary := RuntimeStateSummary(&pbApic.RuntimeState{
		EventReceiptContentDissentObservations: 5,
		PhysicalConnectedPeers:                 []string{"physical"},
		RoutedPeers:                            []string{"routed"},
		ParticipatingPeers:                     []string{"participating"},
		LogicalPeers:                           []string{"logical"},
		LogicalPeerTarget:                      8,
	})
	if !strings.Contains(summary, "event_receipt_content_dissent_observations=5") {
		t.Fatalf("runtime state summary = %q, want content dissent observations", summary)
	}
	for _, want := range []string{"physical=[physical]", "routed=[routed]", "participating=[participating]", "logical=[logical]", "logical_target=8"} {
		if !strings.Contains(summary, want) {
			t.Fatalf("runtime state summary = %q, want %q", summary, want)
		}
	}
}
