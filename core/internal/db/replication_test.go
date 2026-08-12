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
	swarmionadmin "github.com/nustiueudinastea/swarmion/runtime/adminrpc"
)

type peerDrainPreparationRuntime struct {
	status       swarmionapp.Status
	peers        []swarmionapp.PeerInfo
	peerStatuses []swarmionapp.PeerStatus
	compat       []swarmionapp.ManifestCompatibility
}

func (r *peerDrainPreparationRuntime) Status() swarmionapp.Status { return r.status }
func (r *peerDrainPreparationRuntime) Peers() []swarmionapp.PeerInfo {
	return append([]swarmionapp.PeerInfo(nil), r.peers...)
}
func (r *peerDrainPreparationRuntime) PeerStatus(context.Context) ([]swarmionapp.PeerStatus, error) {
	return append([]swarmionapp.PeerStatus(nil), r.peerStatuses...), nil
}
func (r *peerDrainPreparationRuntime) CatchUpCheckpoint(context.Context, swarmionadmin.CheckpointCatchUpRequest) (swarmionadmin.CheckpointCatchUpResponse, error) {
	return swarmionadmin.CheckpointCatchUpResponse{Status: string(swarmionadmin.CheckpointCatchUpStatusAlreadyCurrent)}, nil
}
func (r *peerDrainPreparationRuntime) Compatibility(context.Context) ([]swarmionapp.ManifestCompatibility, error) {
	return append([]swarmionapp.ManifestCompatibility(nil), r.compat...), nil
}
func (r *peerDrainPreparationRuntime) BeginPeerDrain(context.Context, swarmionapp.PeerDrainRequest) (swarmionapp.PeerDrainStatus, error) {
	return swarmionapp.PeerDrainStatus{}, nil
}
func (r *peerDrainPreparationRuntime) PeerDrainStatus(context.Context, swarmionapp.PeerDrainRequest) (swarmionapp.PeerDrainStatus, error) {
	return swarmionapp.PeerDrainStatus{}, nil
}
func (r *peerDrainPreparationRuntime) WatchPeerDrain(context.Context, swarmionapp.PeerDrainRequest) (<-chan swarmionapp.PeerDrainEvent, error) {
	events := make(chan swarmionapp.PeerDrainEvent)
	close(events)
	return events, nil
}
func (r *peerDrainPreparationRuntime) WaitPeerDrainReady(context.Context, swarmionapp.PeerDrainRequest) (swarmionapp.PeerDrainStatus, error) {
	return swarmionapp.PeerDrainStatus{}, nil
}
func (r *peerDrainPreparationRuntime) FinalizePeerDrain(context.Context, swarmionapp.PeerDrainRequest) (swarmionapp.PeerDrainFinalizeResponse, error) {
	return swarmionapp.PeerDrainFinalizeResponse{}, nil
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
		err := prepareReplicationPeerDrainWithRuntime(context.Background(), base(), "target", nil, openedAt)
		if !errors.Is(err, ErrReplicationPeerDrainPending) {
			t.Fatalf("error = %v, want pending", err)
		}
	})

	t.Run("target unobserved remains pending", func(t *testing.T) {
		runtime := base()
		runtime.peers = runtime.peers[1:]
		err := prepareReplicationPeerDrainWithRuntime(context.Background(), runtime, "target", []ReplicationCandidate{{PeerID: "local", DeviceClass: ReplicationDeviceClassLocalUserClient}}, openedAt)
		if !errors.Is(err, ErrReplicationPeerDrainPending) || !strings.Contains(err.Error(), "freshly observed") {
			t.Fatalf("error = %v, want fresh-target pending", err)
		}
	})

	t.Run("remote behind remains pending", func(t *testing.T) {
		runtime := base()
		runtime.peers[1].CheckpointCommitID = swarmionprotocol.NewCheckpointCommitID("behind")
		runtime.peers[1].CheckpointRootHash = swarmionprotocol.NewRootHash("behind-root")
		err := prepareReplicationPeerDrainWithRuntime(context.Background(), runtime, "target", []ReplicationCandidate{{PeerID: "remote", DeviceClass: ReplicationDeviceClassCloudVM}}, openedAt)
		if !errors.Is(err, ErrReplicationPeerDrainPending) {
			t.Fatalf("error = %v, want pending", err)
		}
	})

	t.Run("exact current remote succeeds", func(t *testing.T) {
		err := prepareReplicationPeerDrainWithRuntime(context.Background(), base(), "target", []ReplicationCandidate{{PeerID: "remote", DeviceClass: ReplicationDeviceClassCloudVM}}, openedAt)
		if err != nil {
			t.Fatalf("prepare exact remote: %v", err)
		}
	})

	t.Run("current local candidate succeeds", func(t *testing.T) {
		err := prepareReplicationPeerDrainWithRuntime(context.Background(), base(), "target", []ReplicationCandidate{{PeerID: "local", DeviceClass: ReplicationDeviceClassLocalUserClient}}, openedAt)
		if err != nil {
			t.Fatalf("prepare current local: %v", err)
		}
	})

	t.Run("restart rejects persisted pre-open target observation", func(t *testing.T) {
		runtime := base()
		runtime.peers[0].LastSeenUnixMillis = openedAt.Add(-time.Second).UnixMilli()
		err := prepareReplicationPeerDrainWithRuntime(context.Background(), runtime, "target", []ReplicationCandidate{{PeerID: "local", DeviceClass: ReplicationDeviceClassLocalUserClient}}, openedAt)
		if !errors.Is(err, ErrReplicationPeerDrainPending) {
			t.Fatalf("error = %v, want pending", err)
		}
	})
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

func TestPeerDrainStatusSummaryIncludesExactGenerationAndBlockers(t *testing.T) {
	summary := PeerDrainStatusSummary(swarmionapp.PeerDrainStatus{
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
	status, err := store.BeginReplicationPeerDrain(context.Background(), peerID, generation)
	if err != nil {
		t.Fatalf("begin peer drain: %v", err)
	}
	if !status.Active || !status.RouteGenerationMatches || !status.LocalCheckpointCovered ||
		!status.PreFenceHeartbeatIngressObserved || status.PostFenceHeartbeatAccepted || !status.ReadyToFinalize {
		t.Fatalf("covered unknown peer drain status = %+v", status)
	}

	watchCtx, cancelWatch := context.WithCancel(context.Background())
	watch, err := store.WatchReplicationPeerDrain(watchCtx, peerID, generation)
	if err != nil {
		cancelWatch()
		t.Fatalf("watch peer drain: %v", err)
	}
	select {
	case event, ok := <-watch:
		if !ok {
			cancelWatch()
			t.Fatal("peer drain watch closed before its initial status")
		}
		if event.Err != nil || !event.Initial || !event.Status.ReadyToFinalize || event.Status.Finalized {
			cancelWatch()
			t.Fatalf("initial peer drain event = %+v", event)
		}
	case <-time.After(5 * time.Second):
		cancelWatch()
		t.Fatal("timed out waiting for initial peer drain event")
	}
	cancelWatch()

	waitCtx, cancelWait := context.WithTimeout(context.Background(), 5*time.Second)
	ready, err := store.WaitReplicationPeerDrainReady(waitCtx, peerID, generation)
	cancelWait()
	if err != nil {
		t.Fatalf("wait peer drain ready: %v", err)
	}
	if !ready.ReadyToFinalize || ready.Finalized || !ready.RouteGenerationMatches {
		t.Fatalf("ready peer drain status = %+v", ready)
	}

	response, err := store.FinalizeReplicationPeerDrain(context.Background(), peerID, generation)
	if err != nil {
		t.Fatalf("finalize peer drain: %v", err)
	}
	if !response.Finalized || response.RouteGeneration != generation || !response.Status.Finalized ||
		response.Status.Active || !response.Status.RouteGenerationMatches || !response.Status.ReadyToFinalize {
		t.Fatalf("finalize response = %+v", response)
	}

	retried, err := store.FinalizeReplicationPeerDrain(context.Background(), peerID, generation)
	if err != nil {
		t.Fatalf("retry finalized peer drain: %v", err)
	}
	if !retried.Finalized || !retried.Status.Finalized || retried.Status.Active {
		t.Fatalf("idempotent finalize response = %+v", retried)
	}

	finalized, err := store.ReplicationPeerDrainStatus(context.Background(), peerID, generation)
	if err != nil {
		t.Fatalf("read finalized drain status: %v", err)
	}
	if finalized.Active || !finalized.Finalized || !finalized.RouteGenerationMatches || !finalized.ReadyToFinalize ||
		len(finalized.BlockingReasonCodes) != 0 || len(finalized.BlockingReasons) != 0 {
		t.Fatalf("finalized generation tombstone = %+v", finalized)
	}

	finalizedWatchCtx, cancelFinalizedWatch := context.WithCancel(context.Background())
	finalizedWatch, err := store.WatchReplicationPeerDrain(finalizedWatchCtx, peerID, generation)
	if err != nil {
		cancelFinalizedWatch()
		t.Fatalf("watch finalized peer drain: %v", err)
	}
	select {
	case event, ok := <-finalizedWatch:
		if !ok || event.Err != nil || !event.Initial || !event.Status.Finalized || event.Status.Active {
			cancelFinalizedWatch()
			t.Fatalf("finalized peer drain event = %+v, open=%t", event, ok)
		}
	case <-time.After(5 * time.Second):
		cancelFinalizedWatch()
		t.Fatal("timed out waiting for finalized peer drain event")
	}
	cancelFinalizedWatch()

	finalizedWaitCtx, cancelFinalizedWait := context.WithTimeout(context.Background(), 5*time.Second)
	finalizedReady, err := store.WaitReplicationPeerDrainReady(finalizedWaitCtx, peerID, generation)
	cancelFinalizedWait()
	if err != nil {
		t.Fatalf("wait finalized peer drain: %v", err)
	}
	if !finalizedReady.Finalized || finalizedReady.Active || !finalizedReady.ReadyToFinalize {
		t.Fatalf("finalized wait status = %+v", finalizedReady)
	}

	const newerGeneration = "backend-route-generation-2"
	newer, err := store.BeginReplicationPeerDrain(context.Background(), peerID, newerGeneration)
	if err != nil {
		t.Fatalf("begin newer peer drain generation: %v", err)
	}
	if !newer.Active || !newer.RouteGenerationMatches || !newer.ReadyToFinalize {
		t.Fatalf("newer peer drain status = %+v", newer)
	}

	inactiveWaitCtx, cancelInactiveWait := context.WithTimeout(context.Background(), 5*time.Second)
	inactiveWait, err := store.WaitReplicationPeerDrainReady(inactiveWaitCtx, peerID, generation)
	cancelInactiveWait()
	if !errors.Is(err, swarmionapp.ErrPeerDrainGenerationInactive) {
		t.Fatalf("superseded wait error = %v, want generation-inactive sentinel", err)
	}
	var inactiveWaitError *swarmionapp.PeerDrainNotReadyError
	if !errors.As(err, &inactiveWaitError) || inactiveWaitError == nil {
		t.Fatalf("superseded wait error = %v, want typed peer-drain status", err)
	}
	if inactiveWait.PeerID != peerID || inactiveWait.RouteGeneration != generation ||
		inactiveWait.RouteGenerationMatches ||
		inactiveWaitError.Status.PeerID != inactiveWait.PeerID ||
		inactiveWaitError.Status.RouteGeneration != inactiveWait.RouteGeneration ||
		inactiveWaitError.Status.RouteGenerationMatches != inactiveWait.RouteGenerationMatches ||
		inactiveWaitError.Status.Active != inactiveWait.Active {
		t.Fatalf("superseded wait status = %+v, typed = %+v", inactiveWait, inactiveWaitError.Status)
	}
	if !reflect.DeepEqual(inactiveWait.BlockingReasonCodes, []swarmionapp.PeerDrainBlockingReason{
		swarmionapp.PeerDrainBlockingReasonNewerRouteGenerationActive,
	}) {
		t.Fatalf("superseded wait blocking codes = %v", inactiveWait.BlockingReasonCodes)
	}

	const inactiveGeneration = "backend-route-generation-inactive"
	_, err = store.FinalizeReplicationPeerDrain(context.Background(), peerID, inactiveGeneration)
	if !errors.Is(err, swarmionapp.ErrPeerDrainGenerationInactive) {
		t.Fatalf("inactive finalize error = %v, want generation-inactive sentinel", err)
	}
	var notReady *swarmionapp.PeerDrainNotReadyError
	if !errors.As(err, &notReady) || notReady == nil {
		t.Fatalf("inactive finalize error = %v, want typed peer-drain status", err)
	}
	if notReady.Status.PeerID != peerID || notReady.Status.RouteGeneration != inactiveGeneration ||
		notReady.Status.RouteGenerationMatches {
		t.Fatalf("inactive finalize typed status = %+v", notReady.Status)
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
