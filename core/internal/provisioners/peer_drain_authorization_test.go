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
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/tasks"
)

type peerDrainAuthorizationCrashFixture struct {
	manager       *Manager
	instance      InstanceInfo
	delete        instanceDeleteOperationIdentity
	authorization instancePeerDrainAuthorization
	runtime       *fakeReplicationPeerDrainRuntime
	fence         *fakeReplicationPeerRouteFence
	provider      *fakeStopFailDeleteProvider
	accepted      bool
	publishPCalls int
	publishDCalls int
}

func newPeerDrainAuthorizationCrashFixture(t *testing.T) *peerDrainAuthorizationCrashFixture {
	t.Helper()
	store := openProvisionerTestDB(t)
	instance := instanceWithTestLifecycleOwner(t, store, peerDrainAuthorizationTestInstance(t))
	insertInstanceForDeleteReceiptTest(t, store, &instance)
	operationID := db.MustNewUUIDv7()
	deleteOperation := instanceDeleteOperationIdentityForTest(t, store, operationID, instance, false)
	authorization, err := newInstancePeerDrainAuthorization(operationID, deleteOperation, instance, false)
	if err != nil {
		t.Fatal(err)
	}
	provider := &fakeStopFailDeleteProvider{instances: map[string]InstanceInfo{
		instance.ProviderResourceID: {
			ID:                 instance.ProviderResourceID,
			ProviderResourceID: instance.ProviderResourceID,
			Status:             ServerStateStopped,
		},
	}}
	manager := newLifecycleTestManager(t, store, newProvisionerRegistry(fakeStopFailDeleteFactory{provider: provider}))
	if err := manager.AddProvisioner(instance.KindID, fakeStopFailDeleteType.String(), nil); err != nil {
		t.Fatal(err)
	}
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
	fixture := &peerDrainAuthorizationCrashFixture{
		manager:       manager,
		instance:      instance,
		delete:        deleteOperation,
		authorization: authorization,
		runtime:       runtime,
		fence:         &fakeReplicationPeerRouteFence{prefix: "crash"},
		provider:      provider,
	}
	manager.peerDrainRuntime = runtime
	manager.peerRouteFence = fixture.fence
	branchReceipt := func() swarmionapp.BranchOperationReceipt {
		return swarmionapp.BranchOperationReceipt{
			Resolution:        swarmionapp.BranchOperationReceiptFound,
			EventID:           strings.Repeat("a", 64),
			PublishedRootHash: strings.Repeat("b", 32),
			EventDigest:       strings.Repeat("c", 64),
			AuthorPeerID:      authorization.AuthorPeerID,
			AuthorSeq:         7,
			IntentDigest:      authorization.IntentDigest,
		}
	}
	manager.lookupPeerDrainAuthorization = func(_ context.Context, operation db.PublishedWriteOperation) (swarmionapp.BranchOperationReceipt, error) {
		if operation != authorization.publishedWriteOperation() {
			return swarmionapp.BranchOperationReceipt{}, fmt.Errorf("unexpected P operation: %+v", operation)
		}
		if !fixture.accepted {
			return swarmionapp.BranchOperationReceipt{Resolution: swarmionapp.BranchOperationReceiptAbsent, SafeToPublish: true}, nil
		}
		return branchReceipt(), nil
	}
	manager.publishPeerDrainAuthorization = func(_ context.Context, operation db.PublishedWriteOperation, expected InstanceInfo, fact tasks.OperationFact) (db.PublishedWriteReceipt, error) {
		fixture.publishPCalls++
		if operation != authorization.publishedWriteOperation() || !persistentInstanceEqual(expected, authorization.expectedInstance()) || fact.Kind != instancePeerDrainAuthorizedV1 {
			return db.PublishedWriteReceipt{}, fmt.Errorf("unexpected P publication body")
		}
		fixture.accepted = true
		return db.PublishedWriteReceiptFromOperation(branchReceipt())
	}
	manager.waitPeerDrainAuthorization = func(_ context.Context, receipt db.PublishedWriteReceipt, _ string) (db.EventReceiptObservation, error) {
		return db.EventReceiptObservation{
			Receipt: receipt,
			State:   db.EventReceiptStateAppliedDurably,
			Status: swarmionapp.BranchEventReceiptStatus{
				AppliedDurably:            true,
				Checkpointed:              true,
				CheckpointCommitID:        "event-P-checkpoint",
				DurableCheckpointCommitID: "later-durable-P-head",
			},
		}, nil
	}
	manager.verifyPeerDrainAuthorization = func(_ context.Context, checkpoint string, got instancePeerDrainAuthorization, fact tasks.OperationFact) error {
		if checkpoint != "event-P-checkpoint" || got != authorization || fact.Kind != instancePeerDrainAuthorizedV1 {
			return fmt.Errorf("unexpected P checkpoint verification")
		}
		return nil
	}
	manager.publishDeleteOperation = func(context.Context, db.PublishedWriteOperation, InstanceInfo) (db.PublishedWriteReceipt, error) {
		fixture.publishDCalls++
		return db.PublishedWriteReceipt{}, fmt.Errorf("stop after provider boundary")
	}
	return fixture
}

func (fixture *peerDrainAuthorizationCrashFixture) runDelete() error {
	return fixture.runDeleteContext(context.Background())
}

func (fixture *peerDrainAuthorizationCrashFixture) runDeleteContext(ctx context.Context) error {
	return fixture.manager.deleteInstanceImperative(
		ctx,
		nil,
		fixture.instance.ID,
		false,
		fixture.authorization.TaskID,
		fixture.delete,
		&fixture.authorization,
		nil,
		func(instanceDeleteOperationReceipt, int, string) error { return nil },
	)
}

func (fixture *peerDrainAuthorizationCrashFixture) restartPeerDrainLifecycle(prefix string) (*fakeReplicationPeerDrainRuntime, *fakeReplicationPeerRouteFence) {
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
	fence := &fakeReplicationPeerRouteFence{prefix: prefix}
	fixture.runtime = runtime
	fixture.fence = fence
	fixture.manager.peerDrainRuntime = runtime
	fixture.manager.peerRouteFence = fence
	return runtime, fence
}

func runWithCrashCapture(fn func() error) (err error, panicked bool) {
	defer func() {
		if recover() != nil {
			panicked = true
		}
	}()
	err = fn()
	return err, false
}

func peerDrainAuthorizationTestInstance(t *testing.T) InstanceInfo {
	t.Helper()
	key, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return InstanceInfo{
		ID:                  db.MustNewUUIDv7(),
		Name:                "authorization-test",
		PublicKey:           key.PublicString(),
		PublicIP:            "203.0.113.42",
		Kind:                KindCloudVM,
		KindID:              "authorization-provider",
		ProviderResourceID:  "provider-resource-42",
		DesiredStatus:       ServerStateRunning,
		ReplicationPriority: 17,
		Location:            "test-region",
		Architecture:        "x86_64",
	}
}

func TestPeerDrainAuthorizationRejectsMismatchedReceiptIdentity(t *testing.T) {
	store := openProvisionerTestDB(t)
	instance := instanceWithTestLifecycleOwner(t, store, peerDrainAuthorizationTestInstance(t))
	operationID := db.MustNewUUIDv7()
	deleteOperation := instanceDeleteOperationIdentityForTest(t, store, operationID, instance, false)
	authorization, err := newInstancePeerDrainAuthorization(operationID, deleteOperation, instance, false)
	if err != nil {
		t.Fatal(err)
	}
	receipt := db.PublishedWriteReceipt{
		EventID:               strings.Repeat("a", 64),
		PublishedRootHash:     strings.Repeat("b", 32),
		AuthorPeerID:          "wrong-author",
		AuthorSeq:             1,
		OperationIntentDigest: authorization.IntentDigest,
	}
	if err := validatePeerDrainAuthorizationReceipt(receipt, authorization); !errors.Is(err, db.ErrPublishedWriteReceiptIdentityConflict) {
		t.Fatalf("receipt identity error = %v, want conflict", err)
	}
}

func TestPeerDrainAuthorizationPublishesAtomicExactFactAndDeletingCAS(t *testing.T) {
	store := openProvisionerTestDB(t)
	instance := instanceWithTestLifecycleOwner(t, store, peerDrainAuthorizationTestInstance(t))
	insertInstanceForDeleteReceiptTest(t, store, &instance)
	operationID := db.MustNewUUIDv7()
	deleteOperation := instanceDeleteOperationIdentityForTest(t, store, operationID, instance, false)
	authorization, err := newInstancePeerDrainAuthorization(operationID, deleteOperation, instance, false)
	if err != nil {
		t.Fatal(err)
	}
	if authorization.Key == deleteOperation.Key {
		t.Fatal("authorization P key was not domain-separated from delete D")
	}
	repeated, err := newInstancePeerDrainAuthorization(operationID, deleteOperation, instance, false)
	if err != nil || repeated != authorization {
		t.Fatalf("authorization is not deterministic: repeated=%+v err=%v", repeated, err)
	}
	fact, err := newInstancePeerDrainAuthorizationFact(authorization)
	if err != nil {
		t.Fatal(err)
	}
	manager := newLifecycleTestManager(t, store, newProvisionerRegistry())
	unlock := manager.lockInstanceLifecycle(instance.ID)
	type queueResult struct {
		err error
	}
	queued := make(chan queueResult, 1)
	go func() {
		_, err := manager.QueueStartInstance(instance.ID)
		queued <- queueResult{err: err}
	}()
	select {
	case result := <-queued:
		unlock()
		t.Fatalf("queue start escaped lifecycle lock before P: %v", result.err)
	case <-time.After(50 * time.Millisecond):
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	receipt, err := db.InsertAndUpdateWithOperationReceiptContext(
		ctx,
		store,
		authorization.publishedWriteOperation(),
		[]db.InsertMapper{createInstancePeerDrainAuthorizationFactCASMapper(fact, instance)},
		[]db.UpdateMapper{createInstancePeerDrainAuthorizationUpdateMapper(instance, fact)},
	)
	if err != nil {
		t.Fatalf("publish peer-drain authorization P: %v", err)
	}
	if err := validatePeerDrainAuthorizationReceipt(receipt, authorization); err != nil {
		t.Fatal(err)
	}
	observation, err := store.WaitForPublishedWriteApplied(ctx, receipt, "peer-drain authorization atomic test")
	if err != nil {
		t.Fatal(err)
	}
	if !observation.Status.Checkpointed || !observation.Status.AppliedDurably ||
		observation.Status.CheckpointCommitID == "" || observation.Status.DurableCheckpointCommitID == "" {
		t.Fatalf("authorization receipt status = %+v", observation.Status)
	}
	if err := manager.verifyInstancePeerDrainAuthorizationAtCheckpoint(
		ctx,
		observation.Status.CheckpointCommitID,
		authorization,
		fact,
	); err != nil {
		t.Fatalf("verify exact authorization checkpoint: %v", err)
	}
	stored, err := db.SelectOne(store, createInstanceQueryMapper(instance.ID))
	if err != nil {
		t.Fatal(err)
	}
	if stored.DesiredStatus != ServerStateDeleting {
		t.Fatalf("desired status = %q, want deleting", stored.DesiredStatus)
	}
	unlock()
	select {
	case result := <-queued:
		if !errors.Is(result.err, ErrInstanceLifecycleConflict) {
			t.Fatalf("paused queue start after P error = %v, want lifecycle conflict", result.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("paused queue start did not resume")
	}
	stale := instance
	stale.DesiredStatus = ServerStateRunning
	staleMachine, staleMetadata := createInstanceLifecycleUpdateMapper(stale)
	if _, err := db.UpdateWithReceiptContext(context.Background(), store, staleMachine, staleMetadata); err != nil {
		t.Fatalf("attempt stale lifecycle update: %v", err)
	}
	stored, err = db.SelectOne(store, createInstanceQueryMapper(instance.ID))
	if err != nil {
		t.Fatal(err)
	}
	if stored.DesiredStatus != ServerStateDeleting {
		t.Fatalf("cross-peer guard allowed stale desired status %q", stored.DesiredStatus)
	}
}

func TestPeerDrainAuthorizationStaleSnapshotConsumesReceiptButCannotAuthorize(t *testing.T) {
	store := openProvisionerTestDB(t)
	instance := instanceWithTestLifecycleOwner(t, store, peerDrainAuthorizationTestInstance(t))
	insertInstanceForDeleteReceiptTest(t, store, &instance)
	operationID := db.MustNewUUIDv7()
	deleteOperation := instanceDeleteOperationIdentityForTest(t, store, operationID, instance, false)
	authorization, err := newInstancePeerDrainAuthorization(operationID, deleteOperation, instance, false)
	if err != nil {
		t.Fatal(err)
	}
	fact, err := newInstancePeerDrainAuthorizationFact(authorization)
	if err != nil {
		t.Fatal(err)
	}

	changed := instance
	changed.ReplicationPriority++
	machine, _ := createInstanceUpdateMapper(changed)
	changedReceipt, err := db.UpdateWithReceiptContext(context.Background(), store, machine)
	if err != nil {
		t.Fatal(err)
	}
	waitForTestPublishedEvent(t, store, changedReceipt, "prepare stale peer-drain authorization snapshot")

	manager := newLifecycleTestManager(t, store, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_, err = manager.completeInstancePeerDrainAuthorization(ctx, authorization, fact, db.PublishedWriteReceipt{}, false)
	if !errors.Is(err, ErrInstanceDeleteInvariantConflict) && !errors.Is(err, db.ErrEventReceiptPending) {
		t.Fatalf("stale authorization error = %v, want fail-closed invariant/pending", err)
	}
	resolved, err := store.LookupPublishedWriteOperation(ctx, authorization.publishedWriteOperation())
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Resolution != swarmionapp.BranchOperationReceiptFound {
		t.Fatalf("stale no-change P resolution = %+v, want found", resolved)
	}
	stored, err := db.SelectOne(store, createInstanceQueryMapper(instance.ID))
	if err != nil {
		t.Fatal(err)
	}
	if stored.DesiredStatus == ServerStateDeleting {
		t.Fatal("stale authorization changed instance to deleting")
	}
}

func TestPeerDrainAuthorizationCrashAfterFinalizeAndPDurableRedrainsBeforeProviderContinuation(t *testing.T) {
	fixture := newPeerDrainAuthorizationCrashFixture(t)
	firstRuntime := fixture.runtime
	fixture.manager.afterPeerDrainAuthorized = func(db.PublishedWriteReceipt) {
		panic("crash after finalized durable P")
	}
	if err, panicked := runWithCrashCapture(fixture.runDelete); err != nil || !panicked {
		t.Fatalf("first delete err=%v panicked=%t, want injected crash", err, panicked)
	}
	if firstRuntime.beginCalls != 1 || firstRuntime.finalizeCalls != 1 || fixture.publishPCalls != 1 || fixture.provider.deleteCalls != 0 {
		t.Fatalf("first phase begin=%d finalize=%d P=%d provider_delete=%d", firstRuntime.beginCalls, firstRuntime.finalizeCalls, fixture.publishPCalls, fixture.provider.deleteCalls)
	}

	restartedRuntime, restartedFence := fixture.restartPeerDrainLifecycle("restarted-after-P")
	fixture.manager.afterPeerDrainAuthorized = nil
	err := fixture.runDelete()
	if err == nil || !strings.Contains(err.Error(), "stop after provider boundary") {
		t.Fatalf("recovery error = %v, want injected D boundary", err)
	}
	if firstRuntime.beginCalls != 1 || firstRuntime.finalizeCalls != 1 {
		t.Fatalf("recovery reused pre-crash runtime begin=%d finalize=%d", firstRuntime.beginCalls, firstRuntime.finalizeCalls)
	}
	if restartedRuntime.beginCalls != 1 || restartedRuntime.watchCalls != 1 || restartedRuntime.finalizeCalls != 1 || restartedFence.next != 1 {
		t.Fatalf(
			"recovery fresh drain begin=%d watch=%d finalize=%d fences=%d, want 1/1/1/1",
			restartedRuntime.beginCalls,
			restartedRuntime.watchCalls,
			restartedRuntime.finalizeCalls,
			restartedFence.next,
		)
	}
	if fixture.publishPCalls != 1 || fixture.provider.deleteCalls != 1 {
		t.Fatalf("recovery P=%d provider_delete=%d, want 1/1", fixture.publishPCalls, fixture.provider.deleteCalls)
	}
}

func TestPeerDrainAuthorizationRecoveredDurablePBlocksProviderUntilFreshDrainFinalizes(t *testing.T) {
	fixture := newPeerDrainAuthorizationCrashFixture(t)
	fixture.manager.afterPeerDrainAuthorized = func(db.PublishedWriteReceipt) {
		panic("crash after finalized durable P")
	}
	if err, panicked := runWithCrashCapture(fixture.runDelete); err != nil || !panicked {
		t.Fatalf("first delete err=%v panicked=%t, want injected crash", err, panicked)
	}

	restartedRuntime, restartedFence := fixture.restartPeerDrainLifecycle("blocked-restart")
	restartedRuntime.begin = swarmionapp.PeerDrainStatus{
		Active:                       true,
		RouteGenerationMatches:       true,
		LocalCheckpointCovered:       false,
		CheckpointCoverageReasonCode: swarmionapp.PeerDrainBlockingReasonPeerCheckpointNotCovered,
		BlockingReasonCodes: []swarmionapp.PeerDrainBlockingReason{
			swarmionapp.PeerDrainBlockingReasonPeerCheckpointNotCovered,
		},
	}
	fixture.manager.afterPeerDrainAuthorized = nil
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := fixture.runDeleteContext(ctx)
	if !errors.Is(err, db.ErrReplicationPeerDrainPending) {
		t.Fatalf("blocked recovery error = %v, want peer-drain pending", err)
	}
	if restartedRuntime.beginCalls != 1 || restartedRuntime.watchCalls != 1 || restartedRuntime.finalizeCalls != 0 || restartedFence.next != 1 {
		t.Fatalf(
			"blocked recovery drain begin=%d watch=%d finalize=%d fences=%d, want 1/1/0/1",
			restartedRuntime.beginCalls,
			restartedRuntime.watchCalls,
			restartedRuntime.finalizeCalls,
			restartedFence.next,
		)
	}
	if fixture.publishPCalls != 1 || fixture.provider.deleteCalls != 0 || fixture.publishDCalls != 0 {
		t.Fatalf(
			"blocked recovery crossed P/provider/D boundary P=%d provider=%d D=%d",
			fixture.publishPCalls,
			fixture.provider.deleteCalls,
			fixture.publishDCalls,
		)
	}
}

func TestPeerDrainAuthorizationRetriesTypedNotAcceptedPUnderFinalizedFence(t *testing.T) {
	fixture := newPeerDrainAuthorizationCrashFixture(t)
	publishAccepted := fixture.manager.publishPeerDrainAuthorization
	attempts := 0
	fixture.manager.publishPeerDrainAuthorization = func(
		ctx context.Context,
		operation db.PublishedWriteOperation,
		expected InstanceInfo,
		fact tasks.OperationFact,
	) (db.PublishedWriteReceipt, error) {
		attempts++
		if attempts == 1 {
			return db.PublishedWriteReceipt{}, fmt.Errorf("transient P rejection: %w", swarmionapp.ErrCommitNotAcceptedSafeToRetry)
		}
		return publishAccepted(ctx, operation, expected, fact)
	}

	err := fixture.runDelete()
	if err == nil || !strings.Contains(err.Error(), "stop after provider boundary") {
		t.Fatalf("delete error = %v, want injected D boundary", err)
	}
	if attempts != 2 || fixture.publishPCalls != 1 {
		t.Fatalf("P attempts=%d accepted publications=%d, want 2/1", attempts, fixture.publishPCalls)
	}
	if fixture.runtime.beginCalls != 1 || fixture.runtime.finalizeCalls != 1 || fixture.fence.next != 1 {
		t.Fatalf(
			"typed not-accepted P escaped finalized fence: begin=%d finalize=%d fences=%d",
			fixture.runtime.beginCalls,
			fixture.runtime.finalizeCalls,
			fixture.fence.next,
		)
	}
	if fixture.provider.deleteCalls != 1 {
		t.Fatalf("provider delete calls=%d, want 1 after durable P", fixture.provider.deleteCalls)
	}
}

func TestPeerDrainAuthorizationNotAcceptedPRemainsRecoverableAfterLifecycleLoss(t *testing.T) {
	fixture := newPeerDrainAuthorizationCrashFixture(t)
	publishAccepted := fixture.manager.publishPeerDrainAuthorization
	transientAttempts := 0
	fixture.manager.publishPeerDrainAuthorization = func(
		context.Context,
		db.PublishedWriteOperation,
		InstanceInfo,
		tasks.OperationFact,
	) (db.PublishedWriteReceipt, error) {
		transientAttempts++
		return db.PublishedWriteReceipt{}, fmt.Errorf("transient P rejection: %w", swarmionapp.ErrCommitNotAcceptedSafeToRetry)
	}

	firstCtx, cancelFirst := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancelFirst()
	err := fixture.runDeleteContext(firstCtx)
	if !errors.Is(err, db.ErrReplicationPeerDrainPending) || !db.IsRetryablePublishedWriteError(err) {
		t.Fatalf("first delete error = %v, want deferred typed-not-accepted P", err)
	}
	if transientAttempts != 1 || fixture.runtime.beginCalls != 1 || fixture.runtime.finalizeCalls != 1 || fixture.provider.deleteCalls != 0 {
		t.Fatalf(
			"first phase P_attempts=%d begin=%d finalize=%d provider_delete=%d, want 1/1/1/0",
			transientAttempts,
			fixture.runtime.beginCalls,
			fixture.runtime.finalizeCalls,
			fixture.provider.deleteCalls,
		)
	}
	stored, getErr := fixture.manager.getInstanceRecord(fixture.instance.ID)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if stored.DesiredStatus != fixture.instance.DesiredStatus {
		t.Fatalf("P-not-accepted instance status=%q, want unchanged %q", stored.DesiredStatus, fixture.instance.DesiredStatus)
	}
	if _, found, factErr := fixture.manager.tasks.OperationFact(context.Background(), fixture.authorization.TaskID, instancePeerDrainAuthorizedV1); factErr != nil {
		t.Fatal(factErr)
	} else if found {
		t.Fatal("typed not-accepted P unexpectedly persisted its authorization fact")
	}

	// Model loss of all lifecycle-local Swarmion/fence state. Because P was
	// proven not accepted, the replicated instance stayed in its pre-delete
	// state; recovery must establish a wholly new fence and drain generation.
	restartedRuntime := &fakeReplicationPeerDrainRuntime{
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
	restartedFence := &fakeReplicationPeerRouteFence{prefix: "restarted"}
	fixture.manager.peerDrainRuntime = restartedRuntime
	fixture.manager.peerRouteFence = restartedFence
	fixture.manager.publishPeerDrainAuthorization = publishAccepted

	err = fixture.runDelete()
	if err == nil || !strings.Contains(err.Error(), "stop after provider boundary") {
		t.Fatalf("recovery error = %v, want injected D boundary", err)
	}
	if restartedRuntime.beginCalls != 1 || restartedRuntime.finalizeCalls != 1 || restartedFence.next != 1 {
		t.Fatalf(
			"recovery did not create one fresh drain: begin=%d finalize=%d fences=%d",
			restartedRuntime.beginCalls,
			restartedRuntime.finalizeCalls,
			restartedFence.next,
		)
	}
	if fixture.publishPCalls != 1 || fixture.provider.deleteCalls != 1 {
		t.Fatalf("recovery accepted P=%d provider_delete=%d, want 1/1", fixture.publishPCalls, fixture.provider.deleteCalls)
	}
}

func TestPeerDrainAuthorizationCrashAfterProviderDeleteDoesNotRepeatDestruction(t *testing.T) {
	fixture := newPeerDrainAuthorizationCrashFixture(t)
	firstRuntime := fixture.runtime
	fixture.manager.afterProviderDelete = func(resourceID string) {
		delete(fixture.provider.instances, resourceID)
		panic("crash after provider delete")
	}
	if err, panicked := runWithCrashCapture(fixture.runDelete); err != nil || !panicked {
		t.Fatalf("first delete err=%v panicked=%t, want injected crash", err, panicked)
	}
	if fixture.provider.deleteCalls != 1 || fixture.publishDCalls != 0 {
		t.Fatalf("first phase provider_delete=%d D=%d, want 1/0", fixture.provider.deleteCalls, fixture.publishDCalls)
	}

	restartedRuntime, restartedFence := fixture.restartPeerDrainLifecycle("restarted-after-provider")
	fixture.manager.afterProviderDelete = nil
	err := fixture.runDelete()
	if err == nil || !strings.Contains(err.Error(), "stop after provider boundary") {
		t.Fatalf("recovery error = %v, want injected D boundary", err)
	}
	if firstRuntime.beginCalls != 1 || firstRuntime.finalizeCalls != 1 {
		t.Fatalf("recovery reused pre-crash runtime begin=%d finalize=%d", firstRuntime.beginCalls, firstRuntime.finalizeCalls)
	}
	if restartedRuntime.beginCalls != 1 || restartedRuntime.watchCalls != 1 || restartedRuntime.finalizeCalls != 1 || restartedFence.next != 1 || fixture.publishPCalls != 1 {
		t.Fatalf(
			"recovery fresh drain begin=%d watch=%d finalize=%d fences=%d P=%d, want 1/1/1/1/1",
			restartedRuntime.beginCalls,
			restartedRuntime.watchCalls,
			restartedRuntime.finalizeCalls,
			restartedFence.next,
			fixture.publishPCalls,
		)
	}
	if fixture.provider.deleteCalls != 1 || fixture.publishDCalls != 1 {
		t.Fatalf("recovery provider_delete=%d D=%d, want 1/1", fixture.provider.deleteCalls, fixture.publishDCalls)
	}
}

func TestPeerDrainAuthorizationOrdersDurablePBeforeProviderAndD(t *testing.T) {
	fixture := newPeerDrainAuthorizationCrashFixture(t)
	var order []string
	fixture.manager.afterPeerDrainAuthorized = func(db.PublishedWriteReceipt) {
		order = append(order, "P-applied-and-verified")
	}
	fixture.manager.afterProviderDelete = func(string) {
		order = append(order, "provider-delete")
	}
	fixture.manager.publishDeleteOperation = func(context.Context, db.PublishedWriteOperation, InstanceInfo) (db.PublishedWriteReceipt, error) {
		fixture.publishDCalls++
		order = append(order, "D-publish")
		return db.PublishedWriteReceipt{}, fmt.Errorf("stop after provider boundary")
	}
	err := fixture.runDelete()
	if err == nil || !strings.Contains(err.Error(), "stop after provider boundary") {
		t.Fatalf("delete error = %v, want injected D boundary", err)
	}
	want := []string{"P-applied-and-verified", "provider-delete", "D-publish"}
	if len(order) != len(want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
	for index := range want {
		if order[index] != want[index] {
			t.Fatalf("order = %v, want %v", order, want)
		}
	}
	if fixture.runtime.finalizeCalls != 1 || fixture.provider.deleteCalls != 1 {
		t.Fatalf("finalize=%d provider_delete=%d, want 1/1", fixture.runtime.finalizeCalls, fixture.provider.deleteCalls)
	}
}
