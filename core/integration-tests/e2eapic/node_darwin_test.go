//go:build darwin

package e2eapic

import (
	"strings"
	"testing"

	pbApic "github.com/protosio/protos/apic/proto"
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

func TestRuntimePeerConnectedUsesCanonicalRuntimeState(t *testing.T) {
	if runtimePeerConnected(nil, "peer-a") {
		t.Fatal("nil runtime state reported connected")
	}
	if runtimePeerConnected(&pbApic.RuntimeState{}, "") {
		t.Fatal("empty peer id reported connected")
	}
	if !runtimePeerConnected(&pbApic.RuntimeState{ConnectedPeers: []string{"peer-a"}}, "peer-a") {
		t.Fatal("peer in connected_peers was not reported connected")
	}
	if !runtimePeerConnected(&pbApic.RuntimeState{PeerStatuses: []*pbApic.RuntimePeerStatus{{PeerId: "peer-a", Connected: true}}}, "peer-a") {
		t.Fatal("peer_statuses.connected was not reported connected")
	}
	if runtimePeerConnected(&pbApic.RuntimeState{PeerStatuses: []*pbApic.RuntimePeerStatus{{PeerId: "peer-a", Connected: false}}}, "peer-a") {
		t.Fatal("disconnected peer status reported connected")
	}
}
