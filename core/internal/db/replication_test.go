package db

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	swarmionadmin "github.com/nustiueudinastea/swarmion/runtime/adminrpc"
	swarmionapp "github.com/nustiueudinastea/swarmion/runtime/app"
)

func TestReplicationPriorityForDeviceClass(t *testing.T) {
	tests := []struct {
		deviceClass string
		want        int
		found       bool
	}{
		{deviceClass: ReplicationDeviceClassPhone, want: 10, found: true},
		{deviceClass: "laptop", want: 50, found: true},
		{deviceClass: ReplicationDeviceClassLocalVM, want: 30, found: true},
		{deviceClass: "vps", want: 100, found: true},
		{deviceClass: "unknown", found: false},
	}
	for _, tt := range tests {
		got, found := ReplicationPriorityForDeviceClass(tt.deviceClass)
		if found != tt.found {
			t.Fatalf("ReplicationPriorityForDeviceClass(%q) found=%v, want %v", tt.deviceClass, found, tt.found)
		}
		if got != tt.want {
			t.Fatalf("ReplicationPriorityForDeviceClass(%q) priority=%d, want %d", tt.deviceClass, got, tt.want)
		}
	}
}

func TestPrioritizedReplicationCandidatesSortsByPriorityAndKeepsHighestPeerPriority(t *testing.T) {
	got := prioritizedReplicationCandidates([]ReplicationCandidate{
		{PeerID: "device", DeviceClass: ReplicationDeviceClassPhone},
		{PeerID: "cloud", DeviceClass: ReplicationDeviceClassCloudVM},
		{PeerID: "device", DeviceClass: ReplicationDeviceClassLocalVM},
		{PeerID: "laptop", DeviceClass: ReplicationDeviceClassLocalUserClient},
		{PeerID: "ignored", DeviceClass: "unknown"},
	})
	want := []prioritizedReplicationCandidate{
		{PeerID: "cloud", DeviceClass: ReplicationDeviceClassCloudVM, Priority: 100},
		{PeerID: "laptop", DeviceClass: ReplicationDeviceClassLocalUserClient, Priority: 50},
		{PeerID: "device", DeviceClass: ReplicationDeviceClassLocalVM, Priority: 30},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("prioritizedReplicationCandidates() = %#v, want %#v", got, want)
	}
}

func TestPrioritizedReplicationCandidatesUsesExplicitPriority(t *testing.T) {
	got := prioritizedReplicationCandidates([]ReplicationCandidate{
		{PeerID: "low-default", DeviceClass: ReplicationDeviceClassPhone, Priority: 90},
		{PeerID: "cloud", DeviceClass: ReplicationDeviceClassCloudVM},
		{PeerID: "unknown", DeviceClass: "unknown", Priority: 60},
	})
	want := []prioritizedReplicationCandidate{
		{PeerID: "cloud", DeviceClass: ReplicationDeviceClassCloudVM, Priority: 100},
		{PeerID: "low-default", DeviceClass: ReplicationDeviceClassPhone, Priority: 90},
		{PeerID: "unknown", DeviceClass: "unknown", Priority: 60},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("prioritizedReplicationCandidates() = %#v, want %#v", got, want)
	}
}

func TestCheckpointCatchUpOperationalErrorUsesStructuredStatus(t *testing.T) {
	if err := checkpointCatchUpOperationalError(swarmionadmin.CheckpointCatchUpResponse{
		Status:    string(swarmionadmin.CheckpointCatchUpStatusNoSnapshot),
		Retryable: true,
	}); err != nil {
		t.Fatalf("no snapshot response should not fail reads: %v", err)
	}

	err := checkpointCatchUpOperationalError(swarmionadmin.CheckpointCatchUpResponse{
		Status:         string(swarmionadmin.CheckpointCatchUpStatusTargetChanged),
		TargetChanged:  true,
		BlockingReason: "checkpoint target changed before catch-up",
	})
	if !errors.Is(err, errSwarmionCheckpointCatchUpRetryable) {
		t.Fatalf("target changed error = %v, want retryable checkpoint error", err)
	}
	if !strings.Contains(err.Error(), "checkpoint target changed before catch-up") {
		t.Fatalf("target changed error = %v, want blocking reason", err)
	}

	err = checkpointCatchUpOperationalError(swarmionadmin.CheckpointCatchUpResponse{
		Status:         string(swarmionadmin.CheckpointCatchUpStatusBlockedFatal),
		BlockingReason: "fatal manifest mismatch",
	})
	if err == nil || errors.Is(err, errSwarmionCheckpointCatchUpRetryable) {
		t.Fatalf("fatal response error = %v, want non-retryable failure", err)
	}
}

func TestPeerRemovalReadinessErrorUsesReadinessContract(t *testing.T) {
	if err := PeerRemovalReadinessError(swarmionapp.PeerRemovalReadinessResponse{
		PeerID:                      "peer-a",
		SafeToRemoveDurableResource: true,
	}); err != nil {
		t.Fatalf("safe peer removal readiness returned error: %v", err)
	}

	err := PeerRemovalReadinessError(swarmionapp.PeerRemovalReadinessResponse{
		PeerID:               "peer-a",
		RemainingObligations: []string{"peer is still a state provider"},
	})
	if !errors.Is(err, errSwarmionPeerRemovalNotReady) {
		t.Fatalf("readiness error = %v, want not-ready error", err)
	}
	if !strings.Contains(err.Error(), "peer is still a state provider") {
		t.Fatalf("readiness error = %v, want obligation in message", err)
	}
}

func TestPeerRemovalReadinessErrorAllowsStaleCheckpointObservation(t *testing.T) {
	err := PeerRemovalReadinessError(swarmionapp.PeerRemovalReadinessResponse{
		PeerID:                   "peer-a",
		StillCheckpointProvider:  true,
		CheckpointProviderReason: "advertises content roots",
		RemainingObligations:     []string{"peer still advertises checkpoint state: advertises content roots"},
	})
	if err != nil {
		t.Fatalf("stale checkpoint-only observation should not block removal: %v", err)
	}

	err = PeerRemovalReadinessError(swarmionapp.PeerRemovalReadinessResponse{
		PeerID:                   "peer-a",
		StillCheckpointProvider:  true,
		CheckpointProviderReason: "advertised checkpoint commit differs from local protocol checkpoint",
		RemainingObligations:     []string{"peer still advertises checkpoint state: advertised checkpoint commit differs from local protocol checkpoint"},
	})
	if err != nil {
		t.Fatalf("stale checkpoint mismatch observation should not block removal: %v", err)
	}

	err = PeerRemovalReadinessError(swarmionapp.PeerRemovalReadinessResponse{
		PeerID:                   "peer-a",
		StillConnected:           true,
		StillCheckpointProvider:  true,
		CheckpointProviderReason: "advertises content roots",
		RemainingObligations:     []string{"peer still advertises checkpoint state: advertises content roots"},
	})
	if !errors.Is(err, errSwarmionPeerRemovalNotReady) {
		t.Fatalf("connected checkpoint provider error = %v, want not-ready error", err)
	}
}

func TestPeerRemovalReadinessErrorAllowsStaleTransportObservation(t *testing.T) {
	err := PeerRemovalReadinessError(swarmionapp.PeerRemovalReadinessResponse{
		PeerID:              "peer-a",
		StillConnected:      true,
		StillActiveViewPeer: true,
		RemainingObligations: []string{
			"peer is still connected",
			"peer is still in the active view",
		},
	})
	if err != nil {
		t.Fatalf("stale transport-only observation should not block removal: %v", err)
	}

	err = PeerRemovalReadinessError(swarmionapp.PeerRemovalReadinessResponse{
		PeerID:             "peer-a",
		StillConnected:     true,
		StillStateProvider: true,
		RemainingObligations: []string{
			"peer is still connected",
			"peer is still a state provider",
		},
	})
	if !errors.Is(err, errSwarmionPeerRemovalNotReady) {
		t.Fatalf("connected state provider error = %v, want not-ready error", err)
	}
}

func TestCompatibilityBoundaryDetailsIncludesInitialBoundaries(t *testing.T) {
	got := compatibilityBoundaryDetails(swarmionapp.ManifestCompatibility{
		LocalInitialRootHash:  "local-root",
		LocalInitialCommitID:  "local-commit",
		RemoteInitialRootHash: "remote-root",
		RemoteInitialCommitID: "remote-commit",
	})
	if !strings.Contains(got, "local initial root=local-root commit=local-commit") ||
		!strings.Contains(got, "remote initial root=remote-root commit=remote-commit") {
		t.Fatalf("compatibilityBoundaryDetails() = %q", got)
	}
}
