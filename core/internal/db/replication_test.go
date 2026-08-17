package db

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	swarmionprotocol "github.com/nustiueudinastea/swarmion/protocol"
	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
)

type peerDrainPreparationRuntime struct {
	status       swarmionapp.Status
	peers        []swarmionapp.PeerInfo
	peerStatuses []swarmionapp.PeerStatus
	compat       []swarmionapp.ManifestCompatibility
	statusErr    error
	peersErr     error
}

func (r *peerDrainPreparationRuntime) Status() (swarmionapp.Status, error) {
	return r.status, r.statusErr
}
func (r *peerDrainPreparationRuntime) Peers() ([]swarmionapp.PeerInfo, error) {
	return append([]swarmionapp.PeerInfo(nil), r.peers...), r.peersErr
}
func (r *peerDrainPreparationRuntime) PeerStatus(context.Context) ([]swarmionapp.PeerStatus, error) {
	return append([]swarmionapp.PeerStatus(nil), r.peerStatuses...), nil
}
func (r *peerDrainPreparationRuntime) ReconcileCheckpoint(context.Context, swarmionapp.CheckpointReconcileRequest) (swarmionapp.CheckpointReconcileResult, error) {
	return swarmionapp.CheckpointReconcileResult{State: swarmionapp.CheckpointReconcileNoTarget}, nil
}
func (r *peerDrainPreparationRuntime) Compatibility(context.Context) ([]swarmionapp.ManifestCompatibility, error) {
	return append([]swarmionapp.ManifestCompatibility(nil), r.compat...), nil
}

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

func TestReplicationPolicyNoticeLogsOnlyWhenCandidatesChange(t *testing.T) {
	db := &DB{}
	first := []prioritizedReplicationCandidate{
		{PeerID: "cloud", DeviceClass: ReplicationDeviceClassCloudVM, Priority: 100},
		{PeerID: "local", DeviceClass: ReplicationDeviceClassLocalUserClient, Priority: 50},
	}
	if !db.shouldLogReplicationPolicyNotice(first) {
		t.Fatal("first notice should be logged")
	}
	if db.shouldLogReplicationPolicyNotice(first) {
		t.Fatal("unchanged notice should be suppressed")
	}
	changed := []prioritizedReplicationCandidate{
		{PeerID: "cloud", DeviceClass: ReplicationDeviceClassCloudVM, Priority: 100},
	}
	if !db.shouldLogReplicationPolicyNotice(changed) {
		t.Fatal("changed notice should be logged")
	}
}

func TestPrepareReplicationPeerDrainRequiresFreshTargetAndExactReplacementCoverage(t *testing.T) {
	openedAt := time.Now().Add(-time.Minute).Truncate(time.Millisecond)
	observedAt := openedAt.Add(time.Second).UnixMilli()
	commit := swarmionprotocol.NewCheckpointCommitID("durable-current")
	root := swarmionprotocol.NewRootHash("durable-root")
	base := func() *peerDrainPreparationRuntime {
		return &peerDrainPreparationRuntime{
			status: swarmionapp.Status{
				PeerID:              "local",
				CheckpointCommitID:  commit,
				CheckpointRootHash:  root,
				DurableMainCommitID: commit,
				DurableMainRootHash: root,
			},
			peers: []swarmionapp.PeerInfo{
				{ID: "target", LastSeenUnixMillis: observedAt},
				{
					ID:                 "remote",
					Participating:      true,
					LastSeenUnixMillis: observedAt,
					CheckpointCommitID: commit,
					CheckpointRootHash: root,
				},
			},
			peerStatuses: []swarmionapp.PeerStatus{
				{PeerID: "target", Routed: true, Participating: true, Compatible: true, LastObservedAt: time.UnixMilli(observedAt)},
				{PeerID: "remote", Routed: true, Participating: true, Compatible: true, LastObservedAt: time.UnixMilli(observedAt)},
			},
		}
	}

	t.Run("no candidates remains pending", func(t *testing.T) {
		runtime := base()
		err := prepareReplicationPeerDrainWithRuntime(context.Background(), runtime, runtime, "target", nil, openedAt)
		if !errors.Is(err, ErrReplicationPeerDrainPending) {
			t.Fatalf("error = %v, want pending", err)
		}
	})

	t.Run("target unobserved remains pending", func(t *testing.T) {
		runtime := base()
		runtime.peers = runtime.peers[1:]
		err := prepareReplicationPeerDrainWithRuntime(context.Background(), runtime, runtime, "target", []ReplicationCandidate{{PeerID: "local", DeviceClass: ReplicationDeviceClassLocalUserClient}}, openedAt)
		if !errors.Is(err, ErrReplicationPeerDrainPending) || !strings.Contains(err.Error(), "freshly observed") {
			t.Fatalf("error = %v, want fresh-target pending", err)
		}
	})

	t.Run("remote behind remains pending", func(t *testing.T) {
		runtime := base()
		runtime.peers[1].CheckpointCommitID = swarmionprotocol.NewCheckpointCommitID("behind")
		runtime.peers[1].CheckpointRootHash = swarmionprotocol.NewRootHash("behind-root")
		err := prepareReplicationPeerDrainWithRuntime(context.Background(), runtime, runtime, "target", []ReplicationCandidate{{PeerID: "remote", DeviceClass: ReplicationDeviceClassCloudVM}}, openedAt)
		if !errors.Is(err, ErrReplicationPeerDrainPending) {
			t.Fatalf("error = %v, want pending", err)
		}
	})

	t.Run("exact current remote succeeds", func(t *testing.T) {
		runtime := base()
		err := prepareReplicationPeerDrainWithRuntime(context.Background(), runtime, runtime, "target", []ReplicationCandidate{{PeerID: "remote", DeviceClass: ReplicationDeviceClassCloudVM}}, openedAt)
		if err != nil {
			t.Fatalf("prepare exact remote: %v", err)
		}
	})

	t.Run("current local candidate succeeds", func(t *testing.T) {
		runtime := base()
		err := prepareReplicationPeerDrainWithRuntime(context.Background(), runtime, runtime, "target", []ReplicationCandidate{{PeerID: "local", DeviceClass: ReplicationDeviceClassLocalUserClient}}, openedAt)
		if err != nil {
			t.Fatalf("prepare current local: %v", err)
		}
	})

	t.Run("restart rejects persisted pre-open target observation", func(t *testing.T) {
		runtime := base()
		runtime.peers[0].LastSeenUnixMillis = openedAt.Add(-time.Second).UnixMilli()
		err := prepareReplicationPeerDrainWithRuntime(context.Background(), runtime, runtime, "target", []ReplicationCandidate{{PeerID: "local", DeviceClass: ReplicationDeviceClassLocalUserClient}}, openedAt)
		if !errors.Is(err, ErrReplicationPeerDrainPending) {
			t.Fatalf("error = %v, want pending", err)
		}
	})
}

func TestCheckpointReconcileOperationalErrorUsesStructuredResult(t *testing.T) {
	if err := checkpointReconcileOperationalError(swarmionapp.CheckpointReconcileResult{
		State:          swarmionapp.CheckpointReconcileNoSnapshot,
		Retryable:      true,
		BlockingReason: "no protocol checkpoint snapshot is available",
	}); err != nil {
		t.Fatalf("no snapshot result should not fail checkpoint reads: %v", err)
	}

	attempted := &swarmionapp.CheckpointTarget{
		CheckpointCommitID: swarmionprotocol.NewCheckpointCommitID("attempted-checkpoint").String(),
		CheckpointRootHash: swarmionprotocol.NewRootHash("attempted-checkpoint-root").String(),
	}
	err := checkpointReconcileOperationalError(swarmionapp.CheckpointReconcileResult{
		State:           swarmionapp.CheckpointReconcileTargetChanged,
		Retryable:       true,
		TargetChanged:   true,
		AttemptedTarget: attempted,
		BlockingReason:  "checkpoint target changed before catch-up",
	})
	if !errors.Is(err, errSwarmionCheckpointCatchUpRetryable) {
		t.Fatalf("target changed error = %v, want retryable checkpoint error", err)
	}
	if !strings.Contains(err.Error(), "checkpoint target changed before catch-up") {
		t.Fatalf("target changed error = %v, want blocking reason", err)
	}

	err = checkpointReconcileOperationalError(swarmionapp.CheckpointReconcileResult{
		State:          swarmionapp.CheckpointReconcileBlockedFatal,
		BlockedByFatal: true,
		BlockingReason: "fatal manifest mismatch",
	})
	if err == nil || errors.Is(err, errSwarmionCheckpointCatchUpRetryable) {
		t.Fatalf("fatal response error = %v, want non-retryable failure", err)
	}
}

func TestPeerDrainStatusSummaryIncludesExactGenerationAndBlockers(t *testing.T) {
	summary := PeerDrainStatusSummary(swarmionapp.PeerDrainSnapshot{
		PeerID:                           "peer-a",
		RouteGeneration:                  "generation-7",
		Active:                           true,
		Finalized:                        false,
		RouteGenerationMatches:           true,
		LocalCheckpointCovered:           false,
		CheckpointCoverageReasonCode:     swarmionapp.PeerDrainBlockingReasonPeerCheckpointNotCovered,
		PreFenceHeartbeatIngressObserved: true,
		PostFenceHeartbeatAccepted:       true,
		HeartbeatIngressFenceSequence:    42,
		BlockingReasonCodes: []swarmionapp.PeerDrainBlockingReason{
			swarmionapp.PeerDrainBlockingReasonPeerCheckpointNotCovered,
			swarmionapp.PeerDrainBlockingReasonPostFenceHeartbeatAccepted,
		},
		BlockingReasons: []string{"checkpoint not covered", "post-fence heartbeat"},
	})
	for _, want := range []string{
		"peer=peer-a",
		"generation=generation-7",
		"finalized=false",
		"checkpoint_covered=false",
		"checkpoint_coverage_reason_code=\"peer_checkpoint_not_covered\"",
		"pre_fence_heartbeat_observed=true",
		"post_fence_heartbeat_accepted=true",
		"ingress_fence_sequence=42",
		"blocking_reason_codes=\"peer_checkpoint_not_covered; post_fence_heartbeat_accepted\"",
		"checkpoint not covered; post-fence heartbeat",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("summary = %q, want %q", summary, want)
		}
	}
}

func TestReplicationPeerDrainFinalizesCoveredUnknownPeerThroughScopedRuntime(t *testing.T) {
	store := openPeerTestDB(t)
	peerID := mustPeerIDFromPublicKeyString(t, testPeerPublicKey(t))
	const generation = "backend-route-generation-1"
	request := swarmionapp.PeerDrainRequest{PeerID: peerID, RouteFenceToken: generation}
	session, err := store.StartReplicationPeerDrain(context.Background(), peerID, generation)
	if err != nil {
		t.Fatalf("start peer drain: %v", err)
	}
	defer session.Close()

	select {
	case event, ok := <-session.Events():
		if !ok {
			t.Fatal("peer drain watch closed before its initial status")
		}
		if event.Err != nil || !event.Initial || event.Kind != swarmionapp.PeerDrainEventReady ||
			!event.Snapshot.ReadyToFinalize || event.Snapshot.Finalized {
			t.Fatalf("initial peer drain event = %+v", event)
		}
		if err := event.ValidateFor(request); err != nil {
			t.Fatalf("validate initial peer drain event: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for initial peer drain event")
	}

	waitCtx, cancelWait := context.WithTimeout(context.Background(), 5*time.Second)
	ready, err := session.WaitReady(waitCtx)
	cancelWait()
	if err != nil {
		t.Fatalf("wait peer drain ready: %v", err)
	}
	if err := ready.ValidateFor(request); err != nil || !ready.ReadyToFinalize || ready.Finalized || !ready.RouteGenerationMatches {
		t.Fatalf("ready peer drain status = %+v", ready)
	}

	response, err := session.Finalize(context.Background())
	if err != nil {
		t.Fatalf("finalize peer drain: %v", err)
	}
	if err := response.ValidateFor(request); err != nil {
		t.Fatalf("validate finalize response: %v", err)
	}
	if !response.Finalized || !response.Snapshot.Finalized || response.Snapshot.Active ||
		!response.Snapshot.RouteGenerationMatches || !response.Snapshot.ReadyToFinalize {
		t.Fatalf("finalize response = %+v", response)
	}

	retried, err := session.Finalize(context.Background())
	if err != nil {
		t.Fatalf("retry finalized peer drain: %v", err)
	}
	if err := retried.ValidateFor(request); err != nil || !retried.Finalized || !retried.Snapshot.Finalized || retried.Snapshot.Active {
		t.Fatalf("idempotent finalize response = %+v", retried)
	}

	finalized, err := session.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("read finalized drain status: %v", err)
	}
	if err := finalized.ValidateFor(request); err != nil || finalized.Active || !finalized.Finalized || !finalized.RouteGenerationMatches || !finalized.ReadyToFinalize ||
		len(finalized.BlockingReasonCodes) != 0 || len(finalized.BlockingReasons) != 0 {
		t.Fatalf("finalized generation tombstone = %+v", finalized)
	}
	session.Close()

	finalizedSession, err := store.StartReplicationPeerDrain(context.Background(), peerID, generation)
	if err != nil {
		t.Fatalf("restart finalized peer drain session: %v", err)
	}
	defer finalizedSession.Close()
	select {
	case event, ok := <-finalizedSession.Events():
		if !ok || event.Err != nil || !event.Initial || event.Kind != swarmionapp.PeerDrainEventFinalized || !event.Snapshot.Finalized || event.Snapshot.Active {
			t.Fatalf("finalized peer drain event = %+v, open=%t", event, ok)
		}
		if err := event.ValidateFor(request); err != nil {
			t.Fatalf("validate finalized peer drain event: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for finalized peer drain event")
	}
	finalizedSession.Close()

	const newerGeneration = "backend-route-generation-2"
	newerRequest := swarmionapp.PeerDrainRequest{PeerID: peerID, RouteFenceToken: newerGeneration}
	newerSession, err := store.StartReplicationPeerDrain(context.Background(), peerID, newerGeneration)
	if err != nil {
		t.Fatalf("start newer peer drain generation: %v", err)
	}
	defer newerSession.Close()
	select {
	case event := <-newerSession.Events():
		if event.Err != nil || event.Kind != swarmionapp.PeerDrainEventReady {
			t.Fatalf("newer initial event = %+v", event)
		}
		if err := event.ValidateFor(newerRequest); err != nil {
			t.Fatalf("validate newer event: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for newer initial event")
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
