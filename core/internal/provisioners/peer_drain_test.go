package provisioners

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	begin                    swarmionapp.PeerDrainStatus
	beginStatuses            []swarmionapp.PeerDrainStatus
	beginErrors              []error
	watchEvents              [][]swarmionapp.PeerDrainEvent
	watchErrors              []error
	closeWatchAfterEvents    bool
	preserveWatchIdentity    bool
	finalize                 swarmionapp.PeerDrainFinalizeResponse
	finalizeResponses        []swarmionapp.PeerDrainFinalizeResponse
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

func (f *fakeReplicationPeerDrainRuntime) Begin(_ context.Context, peerID, generation string) (swarmionapp.PeerDrainStatus, error) {
	f.beginCalls++
	f.generations = append(f.generations, generation)
	status := f.begin
	if len(f.beginStatuses) > 0 {
		idx := f.beginCalls - 1
		if idx >= len(f.beginStatuses) {
			idx = len(f.beginStatuses) - 1
		}
		status = f.beginStatuses[idx]
	}
	status.PeerID = peerID
	status.RouteGeneration = generation
	if f.unavailableAfterBegin {
		f.available = false
	}
	var err error
	if len(f.beginErrors) > 0 {
		idx := f.beginCalls - 1
		if idx >= len(f.beginErrors) {
			idx = len(f.beginErrors) - 1
		}
		err = f.beginErrors[idx]
	}
	return status, err
}

func (f *fakeReplicationPeerDrainRuntime) Watch(ctx context.Context, peerID, generation string) (<-chan swarmionapp.PeerDrainEvent, error) {
	f.watchCalls++
	f.watchContexts = append(f.watchContexts, ctx)
	if len(f.watchErrors) > 0 {
		idx := f.watchCalls - 1
		if idx >= len(f.watchErrors) {
			idx = len(f.watchErrors) - 1
		}
		if f.watchErrors[idx] != nil {
			return nil, f.watchErrors[idx]
		}
	}
	var events []swarmionapp.PeerDrainEvent
	if len(f.watchEvents) > 0 {
		idx := f.watchCalls - 1
		if idx >= len(f.watchEvents) {
			idx = len(f.watchEvents) - 1
		}
		events = f.watchEvents[idx]
	}
	watch := make(chan swarmionapp.PeerDrainEvent, len(events))
	for _, event := range events {
		if !f.preserveWatchIdentity {
			event.Status.PeerID = peerID
			event.Status.RouteGeneration = generation
		}
		watch <- event
	}
	if f.closeWatchAfterEvents {
		close(watch)
	}
	return watch, nil
}

func (f *fakeReplicationPeerDrainRuntime) Finalize(_ context.Context, peerID, generation string) (swarmionapp.PeerDrainFinalizeResponse, error) {
	f.finalizeCalls++
	response := f.finalize
	if len(f.finalizeResponses) > 0 {
		idx := f.finalizeCalls - 1
		if idx >= len(f.finalizeResponses) {
			idx = len(f.finalizeResponses) - 1
		}
		response = f.finalizeResponses[idx]
	}
	if !f.preserveFinalizeIdentity {
		response.PeerID = peerID
		response.RouteGeneration = generation
		response.Status.PeerID = peerID
		response.Status.RouteGeneration = generation
	}
	var err error
	if len(f.finalizeErrors) > 0 {
		idx := f.finalizeCalls - 1
		if idx >= len(f.finalizeErrors) {
			idx = len(f.finalizeErrors) - 1
		}
		err = f.finalizeErrors[idx]
	}
	var notReady *swarmionapp.PeerDrainNotReadyError
	if !f.preserveFinalizeIdentity && errors.As(err, &notReady) && notReady != nil {
		notReady.Status.PeerID = peerID
		notReady.Status.RouteGeneration = generation
	}
	return response, err
}

type fakeReplicationPeerRouteFence struct {
	prefix      string
	next        int
	currentPeer string
	current     string
	beforeFence func(int) error
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
		begin: swarmionapp.PeerDrainStatus{
			Active:                           true,
			RouteGenerationMatches:           true,
			LocalCheckpointCovered:           true,
			PreFenceHeartbeatIngressObserved: true,
			ReadyToFinalize:                  true,
		},
		finalize: swarmionapp.PeerDrainFinalizeResponse{Finalized: true},
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
		begin: swarmionapp.PeerDrainStatus{
			Active:                           true,
			RouteGenerationMatches:           true,
			LocalCheckpointCovered:           true,
			PreFenceHeartbeatIngressObserved: true,
			ReadyToFinalize:                  true,
		},
		finalize: swarmionapp.PeerDrainFinalizeResponse{
			Finalized:       true,
			PeerID:          "wrong-peer",
			RouteGeneration: "wrong-generation",
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
	if !errors.Is(err, db.ErrReplicationPeerDrainPending) || !strings.Contains(err.Error(), "finalize response identity") {
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
		beginStatuses: []swarmionapp.PeerDrainStatus{
			{Active: false, RouteGenerationMatches: false},
			{
				Active:                           true,
				RouteGenerationMatches:           true,
				LocalCheckpointCovered:           true,
				PreFenceHeartbeatIngressObserved: true,
				ReadyToFinalize:                  true,
			},
		},
		finalize: swarmionapp.PeerDrainFinalizeResponse{Finalized: true},
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

func TestInstancePeerDrainWaitsForPreFenceIngressObservation(t *testing.T) {
	instance := peerDrainTestInstance(t)
	runtime := &fakeReplicationPeerDrainRuntime{
		available: true,
		begin: swarmionapp.PeerDrainStatus{
			Active:                 true,
			RouteGenerationMatches: true,
			LocalCheckpointCovered: true,
		},
		watchEvents: [][]swarmionapp.PeerDrainEvent{{{
			Status: swarmionapp.PeerDrainStatus{
				Active:                           true,
				RouteGenerationMatches:           true,
				LocalCheckpointCovered:           true,
				PreFenceHeartbeatIngressObserved: true,
				ReadyToFinalize:                  true,
			},
		}}},
		finalize: swarmionapp.PeerDrainFinalizeResponse{Finalized: true},
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
		begin: swarmionapp.PeerDrainStatus{
			Active:                         true,
			RouteGenerationMatches:         true,
			LocalCheckpointCovered:         false,
			CheckpointCoverageReason:       "peer checkpoint is not present in local validated lineage",
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
		beginStatuses: []swarmionapp.PeerDrainStatus{
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
			Status: swarmionapp.PeerDrainStatus{
				Active:                           true,
				RouteGenerationMatches:           true,
				LocalCheckpointCovered:           true,
				PreFenceHeartbeatIngressObserved: true,
				PostFenceHeartbeatAccepted:       true,
				BlockingReasonCodes:              []swarmionapp.PeerDrainBlockingReason{swarmionapp.PeerDrainBlockingReasonPostFenceHeartbeatAccepted},
			},
		}}, nil},
		finalize: swarmionapp.PeerDrainFinalizeResponse{Finalized: true},
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
	ready := swarmionapp.PeerDrainStatus{
		Active:                           true,
		RouteGenerationMatches:           true,
		LocalCheckpointCovered:           true,
		PreFenceHeartbeatIngressObserved: true,
		ReadyToFinalize:                  true,
	}
	runtime := &fakeReplicationPeerDrainRuntime{
		available:     true,
		beginStatuses: []swarmionapp.PeerDrainStatus{ready, ready},
		finalizeResponses: []swarmionapp.PeerDrainFinalizeResponse{
			{Status: swarmionapp.PeerDrainStatus{
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
			&swarmionapp.PeerDrainNotReadyError{Status: swarmionapp.PeerDrainStatus{
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
		beginStatuses: []swarmionapp.PeerDrainStatus{
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
			Status: swarmionapp.PeerDrainStatus{
				Active:                 true,
				RouteGenerationMatches: true,
				LocalCheckpointCovered: true,
				BlockingReasonCodes: []swarmionapp.PeerDrainBlockingReason{
					swarmionapp.PeerDrainBlockingReasonNewerRouteGenerationActive,
				},
			},
		}}, nil},
		finalize: swarmionapp.PeerDrainFinalizeResponse{Finalized: true},
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
	ready := swarmionapp.PeerDrainStatus{
		Active:                           true,
		RouteGenerationMatches:           true,
		LocalCheckpointCovered:           true,
		PreFenceHeartbeatIngressObserved: true,
		ReadyToFinalize:                  true,
	}
	runtime := &fakeReplicationPeerDrainRuntime{
		available:         true,
		beginStatuses:     []swarmionapp.PeerDrainStatus{ready, ready},
		finalizeResponses: []swarmionapp.PeerDrainFinalizeResponse{{Status: ready}, {Finalized: true}},
		finalizeErrors:    []error{swarmionapp.ErrPeerDrainGenerationInactive, nil},
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
	ready := swarmionapp.PeerDrainStatus{
		Active:                           true,
		RouteGenerationMatches:           true,
		LocalCheckpointCovered:           true,
		PreFenceHeartbeatIngressObserved: true,
		ReadyToFinalize:                  true,
	}
	waiting := swarmionapp.PeerDrainStatus{
		Active:                       true,
		RouteGenerationMatches:       true,
		LocalCheckpointCovered:       false,
		CheckpointCoverageReasonCode: swarmionapp.PeerDrainBlockingReasonPeerCheckpointNotCovered,
		BlockingReasonCodes: []swarmionapp.PeerDrainBlockingReason{
			swarmionapp.PeerDrainBlockingReasonPeerCheckpointNotCovered,
		},
	}
	runtime := &fakeReplicationPeerDrainRuntime{
		available: true,
		begin:     ready,
		watchEvents: [][]swarmionapp.PeerDrainEvent{{{
			Status: ready,
		}}},
		finalizeResponses: []swarmionapp.PeerDrainFinalizeResponse{{Status: waiting}, {Finalized: true}},
		finalizeErrors: []error{
			&swarmionapp.PeerDrainNotReadyError{Status: waiting},
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
	ready := swarmionapp.PeerDrainStatus{
		Active:                           true,
		RouteGenerationMatches:           true,
		LocalCheckpointCovered:           true,
		PreFenceHeartbeatIngressObserved: true,
		ReadyToFinalize:                  true,
	}
	runtime := &fakeReplicationPeerDrainRuntime{
		available:         true,
		begin:             ready,
		finalizeResponses: []swarmionapp.PeerDrainFinalizeResponse{{Status: ready}, {Finalized: true}},
		finalizeErrors:    []error{errors.New("clear local provider cache: transient failure"), nil},
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

func TestInstancePeerDrainFinalizedStatusUsesIdempotentFinalize(t *testing.T) {
	instance := peerDrainTestInstance(t)
	runtime := &fakeReplicationPeerDrainRuntime{
		available: true,
		begin: swarmionapp.PeerDrainStatus{
			Active:                 true,
			RouteGenerationMatches: true,
			LocalCheckpointCovered: true,
		},
		watchEvents: [][]swarmionapp.PeerDrainEvent{{{
			Status: swarmionapp.PeerDrainStatus{
				Finalized:                        true,
				RouteGenerationMatches:           true,
				LocalCheckpointCovered:           true,
				PreFenceHeartbeatIngressObserved: true,
				ReadyToFinalize:                  true,
			},
		}}},
		finalize: swarmionapp.PeerDrainFinalizeResponse{Finalized: true},
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
	waiting := swarmionapp.PeerDrainStatus{Active: true, RouteGenerationMatches: true, LocalCheckpointCovered: true}
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
					Status: swarmionapp.PeerDrainStatus{
						PeerID:                 "wrong-peer",
						RouteGeneration:        "wrong-generation",
						Active:                 true,
						RouteGenerationMatches: true,
					},
				}}},
			},
			want: "peer-drain status identity",
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
		begin:     swarmionapp.PeerDrainStatus{Active: true, RouteGenerationMatches: true},
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
		begin: swarmionapp.PeerDrainStatus{
			Active:                           true,
			RouteGenerationMatches:           true,
			LocalCheckpointCovered:           true,
			PreFenceHeartbeatIngressObserved: true,
			ReadyToFinalize:                  true,
		},
		finalize: swarmionapp.PeerDrainFinalizeResponse{Finalized: true},
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
		beginStatuses: []swarmionapp.PeerDrainStatus{
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
				context.Background(), nil, instance.ID, localOnly, operationID, identity, nil, nil,
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
		begin: swarmionapp.PeerDrainStatus{
			Active:                 true,
			RouteGenerationMatches: true,
			LocalCheckpointCovered: true,
			ReadyToFinalize:        true,
		},
		finalize: swarmionapp.PeerDrainFinalizeResponse{Finalized: true},
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
		context.Background(), nil, instance.ID, false, operationID, identity, nil, nil,
		func(instanceDeleteOperationReceipt, int, string) error { return nil },
	)
	if !errors.Is(err, db.ErrReplicationPeerDrainPending) || !strings.Contains(err.Error(), "inconsistent ready peer-drain status") {
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
		context.Background(), nil, instance.ID, false, operationID, identity, nil, nil,
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
		begin: swarmionapp.PeerDrainStatus{
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
		context.Background(), nil, instance.ID, false, operationID, identity, nil, nil,
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

		if _, err := manager.QueueStartInstance(instance.ID); !errors.Is(err, ErrInstanceLifecycleConflict) {
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
		if _, err := manager.QueueStartInstance(instance.ID); !errors.Is(err, ErrInstanceLifecycleConflict) {
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
