package provisioners

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/p2p"
	"github.com/protosio/protos/internal/pcrypto"
)

type fakeReplicationPeerDrainRuntime struct {
	available                bool
	unavailableAfterBegin    bool
	begin                    swarmionapp.PeerDrainSnapshot
	beginStatuses            []swarmionapp.PeerDrainSnapshot
	beginErrors              []error
	watchEvents              [][]swarmionapp.PeerDrainEvent
	watchErrors              []error
	closeWatchAfterEvents    bool
	preserveWatchIdentity    bool
	finalize                 swarmionapp.PeerDrainFinalizeResult
	finalizeResponses        []swarmionapp.PeerDrainFinalizeResult
	finalizeErrors           []error
	preserveFinalizeIdentity bool

	prepareCalls  int
	beginCalls    int
	watchCalls    int
	finalizeCalls int
	generations   []string
	watchContexts []context.Context
}

func (f *fakeReplicationPeerDrainRuntime) Available() bool { return f != nil && f.available }

func (f *fakeReplicationPeerDrainRuntime) Prepare(context.Context, string, []db.ReplicationCandidate) error {
	f.prepareCalls++
	return nil
}

func (f *fakeReplicationPeerDrainRuntime) Start(ctx context.Context, peerID, generation string) (replicationPeerDrainSession, error) {
	f.beginCalls++
	f.generations = append(f.generations, generation)
	initial := f.begin
	if len(f.beginStatuses) > 0 {
		idx := f.beginCalls - 1
		if idx >= len(f.beginStatuses) {
			idx = len(f.beginStatuses) - 1
		}
		initial = f.beginStatuses[idx]
	}
	if len(f.beginErrors) > 0 {
		idx := f.beginCalls - 1
		if idx >= len(f.beginErrors) {
			idx = len(f.beginErrors) - 1
		}
		if f.beginErrors[idx] != nil {
			return nil, f.beginErrors[idx]
		}
	}
	f.watchCalls++
	if len(f.watchErrors) > 0 {
		idx := f.watchCalls - 1
		if idx >= len(f.watchErrors) {
			idx = len(f.watchErrors) - 1
		}
		if f.watchErrors[idx] != nil {
			return nil, f.watchErrors[idx]
		}
	}
	sessionCtx, cancel := context.WithCancel(ctx)
	f.watchContexts = append(f.watchContexts, sessionCtx)
	var events []swarmionapp.PeerDrainEvent
	if len(f.watchEvents) > 0 {
		idx := f.watchCalls - 1
		if idx >= len(f.watchEvents) {
			idx = len(f.watchEvents) - 1
		}
		events = f.watchEvents[idx]
	}
	watch := make(chan swarmionapp.PeerDrainEvent, len(events)+1)
	initial = correlatePeerDrainSnapshotForTest(initial, peerID, generation)
	watch <- swarmionapp.PeerDrainEvent{
		Kind:     peerDrainEventKindForTest(initial),
		Snapshot: initial,
		Initial:  true,
	}
	for _, event := range events {
		if !f.preserveWatchIdentity {
			event.Snapshot = correlatePeerDrainSnapshotForTest(event.Snapshot, peerID, generation)
		}
		if event.Kind == "" {
			event.Kind = peerDrainEventKindForTest(event.Snapshot)
		}
		watch <- event
	}
	done := make(chan struct{})
	if f.closeWatchAfterEvents {
		close(watch)
		close(done)
	} else {
		go func() {
			<-sessionCtx.Done()
			close(watch)
			close(done)
		}()
	}
	if f.unavailableAfterBegin {
		f.available = false
	}
	return &fakeReplicationPeerDrainSession{
		runtime:    f,
		peerID:     peerID,
		generation: generation,
		events:     watch,
		cancel:     cancel,
		done:       done,
	}, nil
}

type fakeReplicationPeerDrainSession struct {
	runtime    *fakeReplicationPeerDrainRuntime
	peerID     string
	generation string
	events     <-chan swarmionapp.PeerDrainEvent
	cancel     context.CancelFunc
	done       <-chan struct{}
	closeOnce  sync.Once
}

func (s *fakeReplicationPeerDrainSession) Events() <-chan swarmionapp.PeerDrainEvent {
	if s == nil {
		return nil
	}
	return s.events
}

func (s *fakeReplicationPeerDrainSession) Finalize(context.Context) (swarmionapp.PeerDrainFinalizeResult, error) {
	if s == nil || s.runtime == nil {
		return swarmionapp.PeerDrainFinalizeResult{}, fmt.Errorf("fake peer-drain session is unavailable")
	}
	f := s.runtime
	f.finalizeCalls++
	result := f.finalize
	if len(f.finalizeResponses) > 0 {
		idx := f.finalizeCalls - 1
		if idx >= len(f.finalizeResponses) {
			idx = len(f.finalizeResponses) - 1
		}
		result = f.finalizeResponses[idx]
	}
	if !f.preserveFinalizeIdentity {
		if result.Finalized && result.Snapshot.PeerID == "" {
			result.Snapshot = finalizedPeerDrainSnapshotForTest(s.peerID, s.generation)
		} else {
			result.Snapshot = correlatePeerDrainSnapshotForTest(result.Snapshot, s.peerID, s.generation)
		}
	}
	var err error
	if len(f.finalizeErrors) > 0 {
		idx := f.finalizeCalls - 1
		if idx >= len(f.finalizeErrors) {
			idx = len(f.finalizeErrors) - 1
		}
		err = f.finalizeErrors[idx]
	}
	if !f.preserveFinalizeIdentity {
		var invalidated *swarmionapp.PeerDrainGenerationInvalidatedError
		if errors.As(err, &invalidated) && invalidated != nil {
			invalidated.Snapshot = correlatePeerDrainSnapshotForTest(invalidated.Snapshot, s.peerID, s.generation)
		}
		var retryable *swarmionapp.PeerDrainFinalizationRetryableError
		if errors.As(err, &retryable) && retryable != nil {
			retryable.Snapshot = correlatePeerDrainSnapshotForTest(retryable.Snapshot, s.peerID, s.generation)
		}
	}
	return result, err
}

func (s *fakeReplicationPeerDrainSession) Close() {
	if s == nil {
		return
	}
	s.closeOnce.Do(func() {
		if s.cancel != nil {
			s.cancel()
		}
	})
	if s.done != nil {
		<-s.done
	}
}

func correlatePeerDrainSnapshotForTest(snapshot swarmionapp.PeerDrainSnapshot, peerID, generation string) swarmionapp.PeerDrainSnapshot {
	snapshot.PeerID = peerID
	snapshot.RouteGeneration = generation
	if snapshot.Finalized {
		return snapshot
	}
	if len(snapshot.BlockingReasonCodes) == 0 {
		switch {
		case !snapshot.Active:
			snapshot.BlockingReasonCodes = []swarmionapp.PeerDrainBlockingReason{swarmionapp.PeerDrainBlockingReasonNoActiveGeneration}
		case !snapshot.RouteGenerationMatches:
			snapshot.BlockingReasonCodes = []swarmionapp.PeerDrainBlockingReason{swarmionapp.PeerDrainBlockingReasonNewerRouteGenerationActive}
		default:
			if !snapshot.PreFenceHeartbeatIngressObserved {
				snapshot.BlockingReasonCodes = append(snapshot.BlockingReasonCodes, swarmionapp.PeerDrainBlockingReasonPreFenceHeartbeatPending)
			}
			if snapshot.PostFenceHeartbeatAccepted {
				snapshot.BlockingReasonCodes = append(snapshot.BlockingReasonCodes, swarmionapp.PeerDrainBlockingReasonPostFenceHeartbeatAccepted)
			}
			if !snapshot.LocalCheckpointCovered && snapshot.CheckpointCoverageReasonCode != "" {
				snapshot.BlockingReasonCodes = append(snapshot.BlockingReasonCodes, snapshot.CheckpointCoverageReasonCode)
			}
		}
	}
	if snapshot.CheckpointCoverageReasonCode != "" && snapshot.CheckpointCoverageReason == "" {
		snapshot.CheckpointCoverageReason = peerDrainReasonTextForTest(snapshot.CheckpointCoverageReasonCode)
	}
	if len(snapshot.BlockingReasons) == 0 && len(snapshot.BlockingReasonCodes) > 0 {
		snapshot.BlockingReasons = make([]string, len(snapshot.BlockingReasonCodes))
		for index, reason := range snapshot.BlockingReasonCodes {
			snapshot.BlockingReasons[index] = peerDrainReasonTextForTest(reason)
		}
	}
	return snapshot
}

func finalizedPeerDrainSnapshotForTest(peerID, generation string) swarmionapp.PeerDrainSnapshot {
	return swarmionapp.PeerDrainSnapshot{
		PeerID:                           peerID,
		RouteGeneration:                  generation,
		Finalized:                        true,
		RouteGenerationMatches:           true,
		LocalCheckpointCovered:           true,
		PreFenceHeartbeatIngressObserved: true,
		ReadyToFinalize:                  true,
	}
}

func peerDrainEventKindForTest(snapshot swarmionapp.PeerDrainSnapshot) swarmionapp.PeerDrainEventKind {
	if snapshot.Finalized {
		return swarmionapp.PeerDrainEventFinalized
	}
	for _, reason := range snapshot.BlockingReasonCodes {
		if reason == swarmionapp.PeerDrainBlockingReasonPostFenceHeartbeatAccepted ||
			reason == swarmionapp.PeerDrainBlockingReasonNewerRouteGenerationActive ||
			reason == swarmionapp.PeerDrainBlockingReasonNoActiveGeneration {
			return swarmionapp.PeerDrainEventGenerationInvalidated
		}
	}
	if snapshot.ReadyToFinalize {
		return swarmionapp.PeerDrainEventReady
	}
	return swarmionapp.PeerDrainEventBlocked
}

func peerDrainReasonTextForTest(reason swarmionapp.PeerDrainBlockingReason) string {
	switch reason {
	case swarmionapp.PeerDrainBlockingReasonAppNotInitialized:
		return "app is not initialized"
	case swarmionapp.PeerDrainBlockingReasonNewerRouteGenerationActive:
		return "a newer route generation is active"
	case swarmionapp.PeerDrainBlockingReasonNoActiveGeneration:
		return "no active peer drain for this route generation"
	case swarmionapp.PeerDrainBlockingReasonIngressFenceUnavailable:
		return "heartbeat ingress fencing is unavailable"
	case swarmionapp.PeerDrainBlockingReasonPreFenceHeartbeatPending:
		return "an acknowledged pre-fence heartbeat has not reached local observation state"
	case swarmionapp.PeerDrainBlockingReasonPostFenceHeartbeatAccepted:
		return "a heartbeat was accepted after the application route fence"
	case swarmionapp.PeerDrainBlockingReasonLocalLineageUnavailable:
		return "local checkpoint lineage is unavailable"
	case swarmionapp.PeerDrainBlockingReasonPeerCheckpointMissingRoot:
		return "peer advertises a checkpoint commit without a checkpoint root"
	case swarmionapp.PeerDrainBlockingReasonPeerCheckpointNotValidated:
		return "peer checkpoint is not present in local validated lineage"
	case swarmionapp.PeerDrainBlockingReasonCheckpointMetadataIncomplete:
		return "local checkpoint lineage metadata is incomplete"
	case swarmionapp.PeerDrainBlockingReasonPeerCheckpointNotCovered:
		return "local checkpoint lineage does not cover the peer checkpoint"
	default:
		return ""
	}
}

type fakeReplicationPeerRouteFence struct {
	prefix      string
	next        int
	currentPeer string
	current     string
	beforeFence func(int) error
	beforeGuard func(string, string) error
	guardCalls  int
}

func (f *fakeReplicationPeerRouteFence) FencePeer(machine p2p.Machine) (string, string, error) {
	peerID, err := db.PeerIDFromPublicKeyString(machine.GetPublicKey())
	if err != nil {
		return "", "", err
	}
	next := f.next + 1
	if f.beforeFence != nil {
		if err := f.beforeFence(next); err != nil {
			return "", "", err
		}
	}
	f.next = next
	f.currentPeer = peerID
	f.current = fmt.Sprintf("%s-%d", f.prefix, f.next)
	return peerID, f.current, nil
}

func (f *fakeReplicationPeerRouteFence) WithPeerFenceGeneration(
	ctx context.Context,
	peerID, generation string,
	fn func() error,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.guardCalls++
	if f.beforeGuard != nil {
		if err := f.beforeGuard(peerID, generation); err != nil {
			return err
		}
	}
	if peerID != f.currentPeer || generation != f.current {
		return fmt.Errorf("route-fence generation changed")
	}
	return fn()
}

func peerDrainTestInstance(t *testing.T) InstanceInfo {
	t.Helper()
	key, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return InstanceInfo{
		ID:            db.MustNewUUIDv7(),
		Name:          "peer-drain-test",
		PublicKey:     key.PublicString(),
		Kind:          KindCloudVM,
		KindID:        "peer-drain-provider",
		DesiredStatus: ServerStateDeleting,
		Location:      "test",
	}
}

func TestInstancePeerDrainPreFenceIngressFinalizesExactGeneration(t *testing.T) {
	instance := peerDrainTestInstance(t)
	runtime := &fakeReplicationPeerDrainRuntime{
		available: true,
		begin: swarmionapp.PeerDrainSnapshot{
			Active:                           true,
			RouteGenerationMatches:           true,
			LocalCheckpointCovered:           true,
			PreFenceHeartbeatIngressObserved: true,
			ReadyToFinalize:                  true,
		},
		finalize: swarmionapp.PeerDrainFinalizeResult{Finalized: true},
	}
	fence := &fakeReplicationPeerRouteFence{prefix: "boot-a"}
	manager := &Manager{peerDrainRuntime: runtime, peerRouteFence: fence}
	if err := manager.waitForInstancePeerDurableRemovalReady(context.Background(), instance); err != nil {
		t.Fatalf("ready peer drain: %v", err)
	}
	if runtime.beginCalls != 1 || runtime.watchCalls != 1 || runtime.finalizeCalls != 1 {
		t.Fatalf("drain calls begin=%d watch=%d finalize=%d", runtime.beginCalls, runtime.watchCalls, runtime.finalizeCalls)
	}
	if len(runtime.generations) != 1 || runtime.generations[0] != fence.current {
		t.Fatalf("drain generation = %v, fence = %q", runtime.generations, fence.current)
	}
}

func TestInstancePeerDrainRejectsWrongSuccessfulFinalizeIdentity(t *testing.T) {
	instance := peerDrainTestInstance(t)
	runtime := &fakeReplicationPeerDrainRuntime{
		available: true,
		begin: swarmionapp.PeerDrainSnapshot{
			Active:                           true,
			RouteGenerationMatches:           true,
			LocalCheckpointCovered:           true,
			PreFenceHeartbeatIngressObserved: true,
			ReadyToFinalize:                  true,
		},
		finalize: swarmionapp.PeerDrainFinalizeResult{
			Finalized: true,
			Snapshot:  finalizedPeerDrainSnapshotForTest("wrong-peer", "wrong-generation"),
		},
		preserveFinalizeIdentity: true,
	}
	manager := &Manager{
		peerDrainRuntime: runtime,
		peerRouteFence:   &fakeReplicationPeerRouteFence{prefix: "identity"},
	}
	continued := 0
	err := manager.withInstancePeerDurableRemovalReady(context.Background(), instance, func() error {
		continued++
		return nil
	})
	if !errors.Is(err, db.ErrReplicationPeerDrainPending) || !strings.Contains(err.Error(), "validate generation") {
		t.Fatalf("wrong finalize identity error = %v", err)
	}
	if continued != 0 {
		t.Fatalf("phase continuation ran %d times after wrong finalize identity", continued)
	}
}

func TestInstancePeerDrainInactiveGenerationRefencesBeforeContinuation(t *testing.T) {
	instance := peerDrainTestInstance(t)
	runtime := &fakeReplicationPeerDrainRuntime{
		available: true,
		beginStatuses: []swarmionapp.PeerDrainSnapshot{
			{Active: false, RouteGenerationMatches: false},
			{
				Active:                           true,
				RouteGenerationMatches:           true,
				LocalCheckpointCovered:           true,
				PreFenceHeartbeatIngressObserved: true,
				ReadyToFinalize:                  true,
			},
		},
		finalize: swarmionapp.PeerDrainFinalizeResult{Finalized: true},
	}
	fence := &fakeReplicationPeerRouteFence{prefix: "inactive"}
	manager := &Manager{peerDrainRuntime: runtime, peerRouteFence: fence}
	continued := 0
	if err := manager.withInstancePeerDurableRemovalReady(context.Background(), instance, func() error {
		continued++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if runtime.beginCalls != 2 || runtime.finalizeCalls != 1 || fence.next != 2 || continued != 1 {
		t.Fatalf("begin=%d finalize=%d fences=%d continued=%d, want 2/1/2/1", runtime.beginCalls, runtime.finalizeCalls, fence.next, continued)
	}
	if len(runtime.generations) != 2 || runtime.generations[0] == runtime.generations[1] {
		t.Fatalf("refence reused generation: %v", runtime.generations)
	}
}

func TestInstancePeerDrainRouteFenceReplacementBeforeFinalizeStartsFreshSession(t *testing.T) {
	instance := peerDrainTestInstance(t)
	ready := swarmionapp.PeerDrainSnapshot{
		Active:                           true,
		RouteGenerationMatches:           true,
		LocalCheckpointCovered:           true,
		PreFenceHeartbeatIngressObserved: true,
		ReadyToFinalize:                  true,
	}
	runtime := &fakeReplicationPeerDrainRuntime{
		available:         true,
		beginStatuses:     []swarmionapp.PeerDrainSnapshot{ready, ready},
		finalizeResponses: []swarmionapp.PeerDrainFinalizeResult{{Finalized: true}},
	}
	fence := &fakeReplicationPeerRouteFence{prefix: "guard-race"}
	fence.beforeGuard = func(_, generation string) error {
		if fence.guardCalls == 1 {
			fence.current = generation + "-replaced"
		}
		return nil
	}
	fence.beforeFence = func(next int) error {
		if next > 1 && (len(runtime.watchContexts) == 0 || runtime.watchContexts[0].Err() == nil) {
			return fmt.Errorf("prior generation session was not closed before re-fence")
		}
		return nil
	}
	manager := &Manager{peerDrainRuntime: runtime, peerRouteFence: fence}
	continued := 0
	if err := manager.withInstancePeerDurableRemovalReady(context.Background(), instance, func() error {
		continued++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if runtime.beginCalls != 2 || runtime.finalizeCalls != 1 || fence.next != 2 || fence.guardCalls != 2 || continued != 1 {
		t.Fatalf(
			"starts=%d finalizes=%d fences=%d guards=%d continued=%d, want 2/1/2/2/1",
			runtime.beginCalls,
			runtime.finalizeCalls,
			fence.next,
			fence.guardCalls,
			continued,
		)
	}
	if runtime.generations[0] == runtime.generations[1] {
		t.Fatalf("route guard race reused generation %q", runtime.generations[0])
	}
}

func TestInstancePeerDrainWaitsForPreFenceIngressObservation(t *testing.T) {
	instance := peerDrainTestInstance(t)
	runtime := &fakeReplicationPeerDrainRuntime{
		available: true,
		begin: swarmionapp.PeerDrainSnapshot{
			Active:                 true,
			RouteGenerationMatches: true,
			LocalCheckpointCovered: true,
		},
		watchEvents: [][]swarmionapp.PeerDrainEvent{{{
			Snapshot: swarmionapp.PeerDrainSnapshot{
				Active:                           true,
				RouteGenerationMatches:           true,
				LocalCheckpointCovered:           true,
				PreFenceHeartbeatIngressObserved: true,
				ReadyToFinalize:                  true,
			},
		}}},
		finalize: swarmionapp.PeerDrainFinalizeResult{Finalized: true},
	}
	fence := &fakeReplicationPeerRouteFence{prefix: "boot-wait"}
	manager := &Manager{peerDrainRuntime: runtime, peerRouteFence: fence}
	if err := manager.waitForInstancePeerDurableRemovalReady(context.Background(), instance); err != nil {
		t.Fatalf("wait for pre-fence heartbeat ingress: %v", err)
	}
	if runtime.beginCalls != 1 || runtime.watchCalls != 1 || runtime.finalizeCalls != 1 {
		t.Fatalf("drain calls begin=%d watch=%d finalize=%d", runtime.beginCalls, runtime.watchCalls, runtime.finalizeCalls)
	}
}

func TestInstancePeerDrainUncoveredCheckpointRemainsPending(t *testing.T) {
	instance := peerDrainTestInstance(t)
	runtime := &fakeReplicationPeerDrainRuntime{
		available: true,
		begin: swarmionapp.PeerDrainSnapshot{
			Active:                         true,
			RouteGenerationMatches:         true,
			LocalCheckpointCovered:         false,
			CheckpointCoverageReasonCode:   swarmionapp.PeerDrainBlockingReasonPeerCheckpointNotValidated,
			CachedContentProviderHintCount: 1,
		},
	}
	manager := &Manager{
		peerDrainRuntime: runtime,
		peerRouteFence:   &fakeReplicationPeerRouteFence{prefix: "boot-a"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := manager.waitForInstancePeerDurableRemovalReady(ctx, instance)
	if !errors.Is(err, db.ErrReplicationPeerDrainPending) {
		t.Fatalf("uncovered drain error = %v, want pending", err)
	}
	if runtime.finalizeCalls != 0 {
		t.Fatalf("uncovered drain finalized %d times", runtime.finalizeCalls)
	}
}

func TestInstancePeerDrainBeginErrorStaysDeferred(t *testing.T) {
	instance := peerDrainTestInstance(t)
	runtime := &fakeReplicationPeerDrainRuntime{
		available:   true,
		beginErrors: []error{errors.New("heartbeat ingress boundary temporarily unavailable")},
	}
	manager := &Manager{
		peerDrainRuntime: runtime,
		peerRouteFence:   &fakeReplicationPeerRouteFence{prefix: "boot-begin-error"},
	}
	err := manager.waitForInstancePeerDurableRemovalReady(context.Background(), instance)
	if !errors.Is(err, db.ErrReplicationPeerDrainPending) ||
		!strings.Contains(err.Error(), "heartbeat ingress boundary temporarily unavailable") {
		t.Fatalf("begin drain error = %v, want deferred typed error", err)
	}
	if runtime.beginCalls != 1 || runtime.finalizeCalls != 0 {
		t.Fatalf("drain calls begin=%d finalize=%d", runtime.beginCalls, runtime.finalizeCalls)
	}
}

func TestInstancePeerDrainPostFenceHeartbeatStartsNewGeneration(t *testing.T) {
	instance := peerDrainTestInstance(t)
	runtime := &fakeReplicationPeerDrainRuntime{
		available: true,
		beginStatuses: []swarmionapp.PeerDrainSnapshot{
			{
				Active:                 true,
				RouteGenerationMatches: true,
				LocalCheckpointCovered: true,
			},
			{
				Active:                           true,
				RouteGenerationMatches:           true,
				LocalCheckpointCovered:           true,
				PreFenceHeartbeatIngressObserved: true,
				ReadyToFinalize:                  true,
			},
		},
		watchEvents: [][]swarmionapp.PeerDrainEvent{{{
			Snapshot: swarmionapp.PeerDrainSnapshot{
				Active:                           true,
				RouteGenerationMatches:           true,
				LocalCheckpointCovered:           true,
				PreFenceHeartbeatIngressObserved: true,
				PostFenceHeartbeatAccepted:       true,
				BlockingReasonCodes:              []swarmionapp.PeerDrainBlockingReason{swarmionapp.PeerDrainBlockingReasonPostFenceHeartbeatAccepted},
			},
		}}, nil},
		finalize: swarmionapp.PeerDrainFinalizeResult{Finalized: true},
	}
	fence := &fakeReplicationPeerRouteFence{prefix: "boot-a"}
	fence.beforeFence = func(next int) error {
		if next > 1 && (len(runtime.watchContexts) == 0 || runtime.watchContexts[0].Err() == nil) {
			return fmt.Errorf("prior generation watch was not canceled before re-fence")
		}
		return nil
	}
	manager := &Manager{peerDrainRuntime: runtime, peerRouteFence: fence}
	if err := manager.waitForInstancePeerDurableRemovalReady(context.Background(), instance); err != nil {
		t.Fatalf("restart drain after post-fence heartbeat: %v", err)
	}
	if runtime.beginCalls != 2 || runtime.watchCalls != 2 || runtime.finalizeCalls != 1 || fence.next != 2 {
		t.Fatalf("drain calls begin=%d watch=%d finalize=%d fences=%d", runtime.beginCalls, runtime.watchCalls, runtime.finalizeCalls, fence.next)
	}
	if len(runtime.generations) != 2 || runtime.generations[0] == runtime.generations[1] {
		t.Fatalf("post-fence heartbeat reused generation: %v", runtime.generations)
	}
}

func TestInstancePeerDrainFinalizeRaceStartsNewGeneration(t *testing.T) {
	instance := peerDrainTestInstance(t)
	ready := swarmionapp.PeerDrainSnapshot{
		Active:                           true,
		RouteGenerationMatches:           true,
		LocalCheckpointCovered:           true,
		PreFenceHeartbeatIngressObserved: true,
		ReadyToFinalize:                  true,
	}
	runtime := &fakeReplicationPeerDrainRuntime{
		available:     true,
		beginStatuses: []swarmionapp.PeerDrainSnapshot{ready, ready},
		finalizeResponses: []swarmionapp.PeerDrainFinalizeResult{
			{Snapshot: swarmionapp.PeerDrainSnapshot{
				Active:                           true,
				RouteGenerationMatches:           true,
				LocalCheckpointCovered:           true,
				PreFenceHeartbeatIngressObserved: true,
				PostFenceHeartbeatAccepted:       true,
				BlockingReasonCodes:              []swarmionapp.PeerDrainBlockingReason{swarmionapp.PeerDrainBlockingReasonPostFenceHeartbeatAccepted},
			}},
			{Finalized: true},
		},
		finalizeErrors: []error{
			&swarmionapp.PeerDrainGenerationInvalidatedError{Snapshot: swarmionapp.PeerDrainSnapshot{
				Active:                           true,
				RouteGenerationMatches:           true,
				LocalCheckpointCovered:           true,
				PreFenceHeartbeatIngressObserved: true,
				PostFenceHeartbeatAccepted:       true,
				BlockingReasonCodes:              []swarmionapp.PeerDrainBlockingReason{swarmionapp.PeerDrainBlockingReasonPostFenceHeartbeatAccepted},
			}},
			nil,
		},
	}
	fence := &fakeReplicationPeerRouteFence{prefix: "boot-race"}
	manager := &Manager{peerDrainRuntime: runtime, peerRouteFence: fence}
	if err := manager.waitForInstancePeerDurableRemovalReady(context.Background(), instance); err != nil {
		t.Fatalf("restart drain after finalize race: %v", err)
	}
	if runtime.beginCalls != 2 || runtime.finalizeCalls != 2 || fence.next != 2 {
		t.Fatalf("drain calls begin=%d finalize=%d fences=%d", runtime.beginCalls, runtime.finalizeCalls, fence.next)
	}
	if len(runtime.generations) != 2 || runtime.generations[0] == runtime.generations[1] {
		t.Fatalf("finalize race reused generation: %v", runtime.generations)
	}
}

func TestInstancePeerDrainNewerGenerationCodeCancelsWatchBeforeRefence(t *testing.T) {
	instance := peerDrainTestInstance(t)
	runtime := &fakeReplicationPeerDrainRuntime{
		available: true,
		beginStatuses: []swarmionapp.PeerDrainSnapshot{
			{Active: true, RouteGenerationMatches: true, LocalCheckpointCovered: true},
			{
				Active:                           true,
				RouteGenerationMatches:           true,
				LocalCheckpointCovered:           true,
				PreFenceHeartbeatIngressObserved: true,
				ReadyToFinalize:                  true,
			},
		},
		watchEvents: [][]swarmionapp.PeerDrainEvent{{{
			Snapshot: swarmionapp.PeerDrainSnapshot{
				Active:                 true,
				RouteGenerationMatches: false,
				BlockingReasonCodes: []swarmionapp.PeerDrainBlockingReason{
					swarmionapp.PeerDrainBlockingReasonNewerRouteGenerationActive,
				},
			},
		}}, nil},
		finalize: swarmionapp.PeerDrainFinalizeResult{Finalized: true},
	}
	fence := &fakeReplicationPeerRouteFence{prefix: "newer-generation"}
	fence.beforeFence = func(next int) error {
		if next > 1 && (len(runtime.watchContexts) == 0 || runtime.watchContexts[0].Err() == nil) {
			return fmt.Errorf("prior generation watch was not canceled before re-fence")
		}
		return nil
	}
	manager := &Manager{peerDrainRuntime: runtime, peerRouteFence: fence}
	if err := manager.waitForInstancePeerDurableRemovalReady(context.Background(), instance); err != nil {
		t.Fatalf("refence newer generation blocker: %v", err)
	}
	if runtime.beginCalls != 2 || runtime.watchCalls != 2 || runtime.finalizeCalls != 1 || fence.next != 2 {
		t.Fatalf("begin=%d watch=%d finalize=%d fences=%d, want 2/2/1/2", runtime.beginCalls, runtime.watchCalls, runtime.finalizeCalls, fence.next)
	}
}

func TestInstancePeerDrainTypedInactiveFinalizeRefences(t *testing.T) {
	instance := peerDrainTestInstance(t)
	ready := swarmionapp.PeerDrainSnapshot{
		Active:                           true,
		RouteGenerationMatches:           true,
		LocalCheckpointCovered:           true,
		PreFenceHeartbeatIngressObserved: true,
		ReadyToFinalize:                  true,
	}
	runtime := &fakeReplicationPeerDrainRuntime{
		available:     true,
		beginStatuses: []swarmionapp.PeerDrainSnapshot{ready, ready},
		finalizeResponses: []swarmionapp.PeerDrainFinalizeResult{
			{Snapshot: swarmionapp.PeerDrainSnapshot{Active: true, BlockingReasonCodes: []swarmionapp.PeerDrainBlockingReason{swarmionapp.PeerDrainBlockingReasonNewerRouteGenerationActive}}},
			{Finalized: true},
		},
		finalizeErrors: []error{
			&swarmionapp.PeerDrainGenerationInvalidatedError{Snapshot: swarmionapp.PeerDrainSnapshot{Active: true, BlockingReasonCodes: []swarmionapp.PeerDrainBlockingReason{swarmionapp.PeerDrainBlockingReasonNewerRouteGenerationActive}}},
			nil,
		},
	}
	fence := &fakeReplicationPeerRouteFence{prefix: "typed-inactive"}
	manager := &Manager{peerDrainRuntime: runtime, peerRouteFence: fence}
	if err := manager.waitForInstancePeerDurableRemovalReady(context.Background(), instance); err != nil {
		t.Fatalf("typed inactive finalize: %v", err)
	}
	if runtime.beginCalls != 2 || runtime.watchCalls != 2 || runtime.finalizeCalls != 2 || fence.next != 2 {
		t.Fatalf("begin=%d watch=%d finalize=%d fences=%d, want 2/2/2/2", runtime.beginCalls, runtime.watchCalls, runtime.finalizeCalls, fence.next)
	}
	if runtime.generations[0] == runtime.generations[1] {
		t.Fatalf("inactive finalize reused generation %q", runtime.generations[0])
	}
}

func TestInstancePeerDrainTypedNotReadyContinuesWatch(t *testing.T) {
	instance := peerDrainTestInstance(t)
	ready := swarmionapp.PeerDrainSnapshot{
		Active:                           true,
		RouteGenerationMatches:           true,
		LocalCheckpointCovered:           true,
		PreFenceHeartbeatIngressObserved: true,
		ReadyToFinalize:                  true,
	}
	waiting := swarmionapp.PeerDrainSnapshot{
		Active:                           true,
		RouteGenerationMatches:           true,
		LocalCheckpointCovered:           false,
		PreFenceHeartbeatIngressObserved: true,
		CheckpointCoverageReasonCode:     swarmionapp.PeerDrainBlockingReasonPeerCheckpointNotCovered,
		BlockingReasonCodes: []swarmionapp.PeerDrainBlockingReason{
			swarmionapp.PeerDrainBlockingReasonPeerCheckpointNotCovered,
		},
	}
	runtime := &fakeReplicationPeerDrainRuntime{
		available: true,
		begin:     ready,
		watchEvents: [][]swarmionapp.PeerDrainEvent{{{
			Snapshot: ready,
		}}},
		finalizeResponses: []swarmionapp.PeerDrainFinalizeResult{{Snapshot: waiting}, {Finalized: true}},
		finalizeErrors: []error{
			swarmionapp.ErrPeerDrainNotReady,
			nil,
		},
	}
	fence := &fakeReplicationPeerRouteFence{prefix: "typed-wait"}
	manager := &Manager{peerDrainRuntime: runtime, peerRouteFence: fence}
	if err := manager.waitForInstancePeerDurableRemovalReady(context.Background(), instance); err != nil {
		t.Fatalf("typed not-ready watch continuation: %v", err)
	}
	if runtime.beginCalls != 1 || runtime.watchCalls != 1 || runtime.finalizeCalls != 2 || fence.next != 1 {
		t.Fatalf("begin=%d watch=%d finalize=%d fences=%d, want 1/1/2/1", runtime.beginCalls, runtime.watchCalls, runtime.finalizeCalls, fence.next)
	}
}

func TestInstancePeerDrainCacheClearErrorRetriesFinalizeSameGeneration(t *testing.T) {
	instance := peerDrainTestInstance(t)
	ready := swarmionapp.PeerDrainSnapshot{
		Active:                           true,
		RouteGenerationMatches:           true,
		LocalCheckpointCovered:           true,
		PreFenceHeartbeatIngressObserved: true,
		ReadyToFinalize:                  true,
	}
	runtime := &fakeReplicationPeerDrainRuntime{
		available:         true,
		begin:             ready,
		finalizeResponses: []swarmionapp.PeerDrainFinalizeResult{{Snapshot: ready}, {Finalized: true}},
		finalizeErrors: []error{
			&swarmionapp.PeerDrainFinalizationRetryableError{Snapshot: ready, Cause: errors.New("clear local provider cache: transient failure")},
			nil,
		},
	}
	fence := &fakeReplicationPeerRouteFence{prefix: "cache-retry"}
	manager := &Manager{peerDrainRuntime: runtime, peerRouteFence: fence}
	if err := manager.waitForInstancePeerDurableRemovalReady(context.Background(), instance); err != nil {
		t.Fatalf("retry cache-clear finalization: %v", err)
	}
	if runtime.beginCalls != 1 || runtime.watchCalls != 1 || runtime.finalizeCalls != 2 || fence.next != 1 {
		t.Fatalf("begin=%d watch=%d finalize=%d fences=%d, want 1/1/2/1", runtime.beginCalls, runtime.watchCalls, runtime.finalizeCalls, fence.next)
	}
}

func TestInstancePeerDrainUntypedFinalizeErrorCannotGrantRetryAuthority(t *testing.T) {
	instance := peerDrainTestInstance(t)
	ready := swarmionapp.PeerDrainSnapshot{
		Active:                           true,
		RouteGenerationMatches:           true,
		LocalCheckpointCovered:           true,
		PreFenceHeartbeatIngressObserved: true,
		ReadyToFinalize:                  true,
	}
	runtime := &fakeReplicationPeerDrainRuntime{
		available:         true,
		begin:             ready,
		finalizeResponses: []swarmionapp.PeerDrainFinalizeResult{{Snapshot: ready}, {Finalized: true}},
		finalizeErrors:    []error{errors.New("untyped cache failure"), nil},
	}
	fence := &fakeReplicationPeerRouteFence{prefix: "untyped-cache-error"}
	manager := &Manager{peerDrainRuntime: runtime, peerRouteFence: fence}
	continued := 0
	err := manager.withInstancePeerDurableRemovalReady(context.Background(), instance, func() error {
		continued++
		return nil
	})
	if !errors.Is(err, db.ErrReplicationPeerDrainPending) || !strings.Contains(err.Error(), "untyped cache failure") {
		t.Fatalf("untyped finalization error = %v, want pending", err)
	}
	if runtime.finalizeCalls != 1 || fence.next != 1 || continued != 0 {
		t.Fatalf("untyped finalization retried or continued: finalizes=%d fences=%d continued=%d", runtime.finalizeCalls, fence.next, continued)
	}
}

func TestInstancePeerDrainFinalizedStatusUsesIdempotentFinalize(t *testing.T) {
	instance := peerDrainTestInstance(t)
	runtime := &fakeReplicationPeerDrainRuntime{
		available: true,
		begin: swarmionapp.PeerDrainSnapshot{
			Active:                 true,
			RouteGenerationMatches: true,
			LocalCheckpointCovered: true,
		},
		watchEvents: [][]swarmionapp.PeerDrainEvent{{{
			Snapshot: swarmionapp.PeerDrainSnapshot{
				Finalized:                        true,
				RouteGenerationMatches:           true,
				LocalCheckpointCovered:           true,
				PreFenceHeartbeatIngressObserved: true,
				ReadyToFinalize:                  true,
			},
		}}},
		finalize: swarmionapp.PeerDrainFinalizeResult{Finalized: true},
	}
	fence := &fakeReplicationPeerRouteFence{prefix: "finalized"}
	manager := &Manager{peerDrainRuntime: runtime, peerRouteFence: fence}
	if err := manager.waitForInstancePeerDurableRemovalReady(context.Background(), instance); err != nil {
		t.Fatalf("idempotent finalized generation: %v", err)
	}
	if runtime.beginCalls != 1 || runtime.watchCalls != 1 || runtime.finalizeCalls != 1 || fence.next != 1 {
		t.Fatalf("begin=%d watch=%d finalize=%d fences=%d, want 1/1/1/1", runtime.beginCalls, runtime.watchCalls, runtime.finalizeCalls, fence.next)
	}
	if runtime.watchContexts[0].Err() == nil {
		t.Fatal("successful finalization did not cancel its generation watch")
	}
}

func TestInstancePeerDrainWatchFailuresRemainPending(t *testing.T) {
	instance := peerDrainTestInstance(t)
	waiting := swarmionapp.PeerDrainSnapshot{Active: true, RouteGenerationMatches: true, LocalCheckpointCovered: true}
	tests := []struct {
		name    string
		runtime *fakeReplicationPeerDrainRuntime
		want    string
	}{
		{
			name: "construction error",
			runtime: &fakeReplicationPeerDrainRuntime{
				available:   true,
				begin:       waiting,
				watchErrors: []error{errors.New("watch unavailable")},
			},
			want: "watch unavailable",
		},
		{
			name: "event error",
			runtime: &fakeReplicationPeerDrainRuntime{
				available: true,
				begin:     waiting,
				watchEvents: [][]swarmionapp.PeerDrainEvent{{{
					Err: errors.New("status read failed"),
				}}},
			},
			want: "status read failed",
		},
		{
			name: "early close",
			runtime: &fakeReplicationPeerDrainRuntime{
				available:             true,
				begin:                 waiting,
				closeWatchAfterEvents: true,
			},
			want: "closed before a terminal status",
		},
		{
			name: "wrong scoped identity",
			runtime: &fakeReplicationPeerDrainRuntime{
				available:             true,
				begin:                 waiting,
				preserveWatchIdentity: true,
				watchEvents: [][]swarmionapp.PeerDrainEvent{{{
					Snapshot: swarmionapp.PeerDrainSnapshot{
						PeerID:                 "wrong-peer",
						RouteGeneration:        "wrong-generation",
						Active:                 true,
						RouteGenerationMatches: true,
					},
				}}},
			},
			want: "does not match request",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fence := &fakeReplicationPeerRouteFence{prefix: "watch-failure"}
			manager := &Manager{peerDrainRuntime: tt.runtime, peerRouteFence: fence}
			err := manager.waitForInstancePeerDurableRemovalReady(context.Background(), instance)
			if !errors.Is(err, db.ErrReplicationPeerDrainPending) || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("watch failure error = %v, want pending containing %q", err, tt.want)
			}
			if tt.runtime.finalizeCalls != 0 || fence.next != 1 {
				t.Fatalf("watch failure finalized=%d fences=%d, want 0/1", tt.runtime.finalizeCalls, fence.next)
			}
		})
	}
}

func TestInstancePeerDrainRestartUsesFreshGeneration(t *testing.T) {
	instance := peerDrainTestInstance(t)
	firstRuntime := &fakeReplicationPeerDrainRuntime{
		available: true,
		begin:     swarmionapp.PeerDrainSnapshot{Active: true, RouteGenerationMatches: true},
	}
	firstFence := &fakeReplicationPeerRouteFence{prefix: "boot-before-restart"}
	first := &Manager{peerDrainRuntime: firstRuntime, peerRouteFence: firstFence}
	firstCtx, firstCancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	firstErr := first.waitForInstancePeerDurableRemovalReady(firstCtx, instance)
	firstCancel()
	if !errors.Is(firstErr, db.ErrReplicationPeerDrainPending) {
		t.Fatalf("first drain error = %v, want pending", firstErr)
	}

	secondRuntime := &fakeReplicationPeerDrainRuntime{
		available: true,
		begin: swarmionapp.PeerDrainSnapshot{
			Active:                           true,
			RouteGenerationMatches:           true,
			LocalCheckpointCovered:           true,
			PreFenceHeartbeatIngressObserved: true,
			ReadyToFinalize:                  true,
		},
		finalize: swarmionapp.PeerDrainFinalizeResult{Finalized: true},
	}
	secondFence := &fakeReplicationPeerRouteFence{prefix: "boot-after-restart"}
	second := &Manager{peerDrainRuntime: secondRuntime, peerRouteFence: secondFence}
	if err := second.waitForInstancePeerDurableRemovalReady(context.Background(), instance); err != nil {
		t.Fatalf("restarted drain: %v", err)
	}
	if firstFence.current == secondFence.current {
		t.Fatalf("restart reused route-fence generation %q", firstFence.current)
	}
	if secondRuntime.finalizeCalls != 1 {
		t.Fatalf("restarted drain finalize calls = %d, want 1", secondRuntime.finalizeCalls)
	}
}

func TestProviderDeletionCannotRunBeforePeerDrainFinalizes(t *testing.T) {
	store := openProvisionerTestDB(t)
	provider := &fakeStopFailDeleteProvider{}
	manager := newLifecycleTestManager(t, store, newProvisionerRegistry(fakeStopFailDeleteFactory{provider: provider}))
	manager.peerDrainRuntime = &fakeReplicationPeerDrainRuntime{
		available: true,
		beginStatuses: []swarmionapp.PeerDrainSnapshot{
			{
				Active:                           true,
				RouteGenerationMatches:           true,
				LocalCheckpointCovered:           true,
				PreFenceHeartbeatIngressObserved: true,
				PostFenceHeartbeatAccepted:       true,
			},
			{
				Active:                 true,
				RouteGenerationMatches: true,
				LocalCheckpointCovered: true,
				ReadyToFinalize:        true,
			},
		},
	}
	manager.peerRouteFence = &fakeReplicationPeerRouteFence{prefix: "boot-a"}
	if err := manager.AddProvisioner("peer-drain-provider", fakeStopFailDeleteType.String(), nil); err != nil {
		t.Fatal(err)
	}
	instance := peerDrainTestInstance(t)
	instance.DesiredStatus = ServerStateRunning
	instance.ProviderResourceID = "provider-vm-id"
	insertInstanceForDeleteReceiptTest(t, store, &instance)
	operationID := db.MustNewUUIDv7()
	identity := instanceDeleteOperationIdentityForTest(t, store, operationID, instance, false)
	err := manager.deleteInstanceImperative(
		context.Background(),
		nil,
		instance.ID,
		false,
		operationID,
		identity,
		nil,
		func(instanceDeleteOperationReceipt, int, string) error { return nil },
	)
	if !errors.Is(err, db.ErrReplicationPeerDrainPending) {
		t.Fatalf("delete error = %v, want peer drain pending", err)
	}
	if provider.deleteCalls != 0 || provider.stopCalls != 0 || provider.volumeDeleteCalls != 0 {
		t.Fatalf("provider mutated before drain finalize: stop=%d delete=%d volume_delete=%d", provider.stopCalls, provider.deleteCalls, provider.volumeDeleteCalls)
	}
}

func TestProviderBackedBlankIdentityDeletionFailsClosed(t *testing.T) {
	for _, localOnly := range []bool{false, true} {
		localOnly := localOnly
		t.Run(fmt.Sprintf("local_only_%t", localOnly), func(t *testing.T) {
			store := openProvisionerTestDB(t)
			provider := &fakeStopFailDeleteProvider{}
			manager := newLifecycleTestManager(t, store, newProvisionerRegistry(fakeStopFailDeleteFactory{provider: provider}))
			instance := InstanceInfo{
				ID:                 db.MustNewUUIDv7(),
				Name:               "blank-identity",
				Kind:               KindCloudVM,
				KindID:             "provider",
				ProviderResourceID: "provider-vm-id",
				DesiredStatus:      ServerStateRunning,
				Location:           "test",
			}
			insertInstanceForDeleteReceiptTest(t, store, &instance)
			operationID := db.MustNewUUIDv7()
			identity := instanceDeleteOperationIdentityForTest(t, store, operationID, instance, localOnly)
			publishCalls := 0
			manager.publishDeleteOperation = func(context.Context, db.PublishedWriteOperation, InstanceInfo) (db.PublishedWriteReceipt, error) {
				publishCalls++
				return db.PublishedWriteReceipt{}, fmt.Errorf("unexpected delete publication")
			}

			err := manager.deleteInstanceImperative(
				context.Background(), nil, instance.ID, localOnly, operationID, identity, nil,
				func(instanceDeleteOperationReceipt, int, string) error { return nil },
			)
			if !errors.Is(err, db.ErrReplicationPeerDrainPending) || !strings.Contains(err.Error(), "no replicated peer identity") {
				t.Fatalf("blank-identity delete error = %v, want peer drain pending", err)
			}
			if provider.deleteCalls != 0 || provider.stopCalls != 0 || provider.volumeDeleteCalls != 0 || publishCalls != 0 {
				t.Fatalf("blank-identity delete mutated state: stop=%d delete=%d volume_delete=%d publish=%d", provider.stopCalls, provider.deleteCalls, provider.volumeDeleteCalls, publishCalls)
			}
			stored, lookupErr := manager.getInstanceRecord(instance.ID)
			if lookupErr != nil {
				t.Fatalf("blank-identity delete removed recovery record: %v", lookupErr)
			}
			if stored.DesiredStatus != ServerStateRunning || stored.ProviderResourceID != instance.ProviderResourceID {
				t.Fatalf("blank-identity recovery record changed: %#v", stored)
			}
		})
	}
}

func TestInconsistentReadyPeerDrainWithoutPreFenceIngressFailsClosed(t *testing.T) {
	store := openProvisionerTestDB(t)
	provider := &fakeStopFailDeleteProvider{}
	manager := newLifecycleTestManager(t, store, newProvisionerRegistry(fakeStopFailDeleteFactory{provider: provider}))
	runtime := &fakeReplicationPeerDrainRuntime{
		available: true,
		begin: swarmionapp.PeerDrainSnapshot{
			Active:                 true,
			RouteGenerationMatches: true,
			LocalCheckpointCovered: true,
			ReadyToFinalize:        true,
		},
		finalize: swarmionapp.PeerDrainFinalizeResult{Finalized: true},
	}
	manager.peerDrainRuntime = runtime
	manager.peerRouteFence = &fakeReplicationPeerRouteFence{prefix: "boot-ready"}
	if err := manager.AddProvisioner("peer-drain-provider", fakeStopFailDeleteType.String(), nil); err != nil {
		t.Fatal(err)
	}
	instance := peerDrainTestInstance(t)
	instance.DesiredStatus = ServerStateRunning
	instance.ProviderResourceID = "provider-vm-id"
	insertInstanceForDeleteReceiptTest(t, store, &instance)
	operationID := db.MustNewUUIDv7()
	identity := instanceDeleteOperationIdentityForTest(t, store, operationID, instance, false)
	publishCalls := 0
	manager.publishDeleteOperation = func(context.Context, db.PublishedWriteOperation, InstanceInfo) (db.PublishedWriteReceipt, error) {
		publishCalls++
		return db.PublishedWriteReceipt{}, fmt.Errorf("unexpected delete publication")
	}
	err := manager.deleteInstanceImperative(
		context.Background(), nil, instance.ID, false, operationID, identity, nil,
		func(instanceDeleteOperationReceipt, int, string) error { return nil },
	)
	if !errors.Is(err, db.ErrReplicationPeerDrainPending) || !strings.Contains(err.Error(), "invalid event") {
		t.Fatalf("delete error = %v, want inconsistent ready status pending", err)
	}
	if runtime.finalizeCalls != 0 {
		t.Fatalf("inconsistent ready status finalized without pre-fence ingress proof %d times", runtime.finalizeCalls)
	}
	if provider.deleteCalls != 0 || provider.stopCalls != 0 || provider.volumeDeleteCalls != 0 || publishCalls != 0 {
		t.Fatalf("ready local status mutated state: stop=%d delete=%d volume_delete=%d publish=%d", provider.stopCalls, provider.deleteCalls, provider.volumeDeleteCalls, publishCalls)
	}
}

func TestPeerDrainRuntimeUnavailableBeforePrepareCannotMutateProviderOrPublishDelete(t *testing.T) {
	store := openProvisionerTestDB(t)
	provider := &fakeStopFailDeleteProvider{}
	manager := newLifecycleTestManager(t, store, newProvisionerRegistry(fakeStopFailDeleteFactory{provider: provider}))
	manager.peerDrainRuntime = &fakeReplicationPeerDrainRuntime{available: false}
	manager.peerRouteFence = &fakeReplicationPeerRouteFence{prefix: "boot-unavailable"}
	instance := peerDrainTestInstance(t)
	instance.DesiredStatus = ServerStateRunning
	instance.ProviderResourceID = "provider-vm-id"
	insertInstanceForDeleteReceiptTest(t, store, &instance)
	operationID := db.MustNewUUIDv7()
	identity := instanceDeleteOperationIdentityForTest(t, store, operationID, instance, false)
	publishCalls := 0
	manager.publishDeleteOperation = func(context.Context, db.PublishedWriteOperation, InstanceInfo) (db.PublishedWriteReceipt, error) {
		publishCalls++
		return db.PublishedWriteReceipt{}, fmt.Errorf("unexpected delete publication")
	}
	err := manager.deleteInstanceImperative(
		context.Background(), nil, instance.ID, false, operationID, identity, nil,
		func(instanceDeleteOperationReceipt, int, string) error { return nil },
	)
	if !errors.Is(err, db.ErrReplicationPeerDrainUnavailable) {
		t.Fatalf("delete error = %v, want drain unavailable", err)
	}
	if provider.deleteCalls != 0 || provider.stopCalls != 0 || provider.volumeDeleteCalls != 0 || publishCalls != 0 {
		t.Fatalf("unavailable prepare mutated state: stop=%d delete=%d volume_delete=%d publish=%d", provider.stopCalls, provider.deleteCalls, provider.volumeDeleteCalls, publishCalls)
	}
}

func TestPeerDrainRuntimeDisappearsAfterBeginCannotMutateProviderOrPublishDelete(t *testing.T) {
	store := openProvisionerTestDB(t)
	provider := &fakeStopFailDeleteProvider{}
	manager := newLifecycleTestManager(t, store, newProvisionerRegistry(fakeStopFailDeleteFactory{provider: provider}))
	runtime := &fakeReplicationPeerDrainRuntime{
		available:             true,
		unavailableAfterBegin: true,
		begin: swarmionapp.PeerDrainSnapshot{
			Active:                 true,
			RouteGenerationMatches: true,
			LocalCheckpointCovered: true,
			ReadyToFinalize:        true,
		},
	}
	manager.peerDrainRuntime = runtime
	manager.peerRouteFence = &fakeReplicationPeerRouteFence{prefix: "boot-disappears"}
	instance := peerDrainTestInstance(t)
	instance.DesiredStatus = ServerStateRunning
	instance.ProviderResourceID = "provider-vm-id"
	insertInstanceForDeleteReceiptTest(t, store, &instance)
	operationID := db.MustNewUUIDv7()
	identity := instanceDeleteOperationIdentityForTest(t, store, operationID, instance, false)
	publishCalls := 0
	manager.publishDeleteOperation = func(context.Context, db.PublishedWriteOperation, InstanceInfo) (db.PublishedWriteReceipt, error) {
		publishCalls++
		return db.PublishedWriteReceipt{}, fmt.Errorf("unexpected delete publication")
	}
	err := manager.deleteInstanceImperative(
		context.Background(), nil, instance.ID, false, operationID, identity, nil,
		func(instanceDeleteOperationReceipt, int, string) error { return nil },
	)
	if !errors.Is(err, db.ErrReplicationPeerDrainUnavailable) {
		t.Fatalf("delete error = %v, want drain unavailable", err)
	}
	if runtime.beginCalls != 1 || runtime.finalizeCalls != 0 {
		t.Fatalf("runtime calls begin=%d finalize=%d", runtime.beginCalls, runtime.finalizeCalls)
	}
	if provider.deleteCalls != 0 || provider.stopCalls != 0 || provider.volumeDeleteCalls != 0 || publishCalls != 0 {
		t.Fatalf("runtime disappearance mutated state: stop=%d delete=%d volume_delete=%d publish=%d", provider.stopCalls, provider.deleteCalls, provider.volumeDeleteCalls, publishCalls)
	}
}

func TestQueueStartInstanceRejectsReplicatedOrTaskOwnedDelete(t *testing.T) {
	t.Run("replicated deleting status", func(t *testing.T) {
		store := openProvisionerTestDB(t)
		manager := newLifecycleTestManager(t, store, newProvisionerRegistry())
		instance := peerDrainTestInstance(t)
		instance.DesiredStatus = ServerStateDeleting
		insertInstanceForDeleteReceiptTest(t, store, &instance)

		if _, err := manager.QueueStartInstance(context.Background(), instance.ID); !errors.Is(err, ErrInstanceLifecycleConflict) {
			t.Fatalf("QueueStartInstance error = %v, want lifecycle conflict", err)
		}
		stored, err := manager.getInstanceRecord(instance.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !IsDeletingInstance(stored) {
			t.Fatalf("start changed deleting desired status to %q", stored.DesiredStatus)
		}
	})

	t.Run("active delete task", func(t *testing.T) {
		store := openProvisionerTestDB(t)
		manager := newLifecycleTestManager(t, store, newProvisionerRegistry())
		instance := peerDrainTestInstance(t)
		instance.DesiredStatus = ServerStateRunning
		insertInstanceForDeleteReceiptTest(t, store, &instance)
		if _, err := manager.QueueDeleteInstance(context.Background(), instance.ID); err != nil {
			t.Fatalf("queue delete: %v", err)
		}
		if _, err := manager.QueueStartInstance(context.Background(), instance.ID); !errors.Is(err, ErrInstanceLifecycleConflict) {
			t.Fatalf("QueueStartInstance error = %v, want lifecycle conflict", err)
		}
		stored, err := manager.getInstanceRecord(instance.ID)
		if err != nil {
			t.Fatal(err)
		}
		if stored.DesiredStatus != ServerStateRunning {
			t.Fatalf("conflicting start changed desired status to %q", stored.DesiredStatus)
		}
	})
}

func TestCanceledLifecycleQueueContextDoesNotPersistTask(t *testing.T) {
	store := openProvisionerTestDB(t)
	manager := newLifecycleTestManager(t, store, newProvisionerRegistry())
	instance := peerDrainTestInstance(t)
	instance.DesiredStatus = ServerStateStopped
	insertInstanceForDeleteReceiptTest(t, store, &instance)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := manager.QueueDeleteInstance(ctx, instance.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("QueueDeleteInstance error = %v, want context canceled", err)
	}
	if task, found, err := manager.tasks.LatestForSubject(
		InstanceLifecycleTaskStream,
		taskSubjectInstance,
		instanceLifecycleSubjectID(instance.ID, instanceLifecycleOperationDelete),
	); err != nil {
		t.Fatal(err)
	} else if found {
		t.Fatalf("canceled delete persisted task %+v", task)
	}

	if _, err := manager.QueueStartInstance(ctx, instance.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("QueueStartInstance error = %v, want context canceled", err)
	}
	if task, found, err := manager.tasks.LatestForSubject(
		InstanceLifecycleTaskStream,
		taskSubjectInstance,
		instanceLifecycleSubjectID(instance.ID, instanceLifecycleOperationReconcile),
	); err != nil {
		t.Fatal(err)
	} else if found {
		t.Fatalf("canceled start persisted task %+v", task)
	}

	stored, err := manager.getInstanceRecord(instance.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.DesiredStatus != ServerStateStopped {
		t.Fatalf("canceled lifecycle request changed desired status to %q", stored.DesiredStatus)
	}
}
