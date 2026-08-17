package provisioners

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
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
	publishPCalls int
	publishDCalls int
}

type lifecycleGateObservedContext struct {
	context.Context
	once     sync.Once
	observed chan struct{}
	target   int32
	calls    atomic.Int32
}

func (ctx *lifecycleGateObservedContext) Done() <-chan struct{} {
	target := ctx.target
	if target <= 0 {
		target = 1
	}
	if ctx.calls.Add(1) >= target {
		ctx.once.Do(func() { close(ctx.observed) })
	}
	return ctx.Context.Done()
}

func TestLifecycleQueueCancellationInterruptsHeldInstanceGate(t *testing.T) {
	store := openProvisionerTestDB(t)
	instance := instanceWithTestLifecycleOwner(t, store, peerDrainAuthorizationTestInstance(t))
	instance.DesiredStatus = ServerStateStopped
	insertInstanceForDeleteReceiptTest(t, store, &instance)
	manager := newLifecycleTestManager(t, store, newProvisionerRegistry())

	unlock := manager.lockInstanceLifecycle(instance.ID)
	defer unlock()
	baseCtx, cancel := context.WithCancel(context.Background())
	ctx := &lifecycleGateObservedContext{Context: baseCtx, observed: make(chan struct{})}
	queueDone := make(chan error, 1)
	go func() {
		_, err := manager.QueueStartInstance(ctx, instance.ID)
		queueDone <- err
	}()

	<-ctx.observed
	cancel()
	select {
	case err := <-queueDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("QueueStartInstance error = %v, want context canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("QueueStartInstance did not honor cancellation while the lifecycle gate was held")
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
		t.Fatalf("canceled start changed desired status to %q", stored.DesiredStatus)
	}
}

func TestLifecycleQueueCancellationInterruptsBlockedInstanceRead(t *testing.T) {
	store := openProvisionerTestDB(t)
	instance := instanceWithTestLifecycleOwner(t, store, peerDrainAuthorizationTestInstance(t))
	instance.DesiredStatus = ServerStateStopped
	insertInstanceForDeleteReceiptTest(t, store, &instance)
	manager := newLifecycleTestManager(t, store, newProvisionerRegistry())

	lockHeld := make(chan struct{})
	releaseLock := make(chan struct{})
	var releaseLockOnce sync.Once
	release := func() { releaseLockOnce.Do(func() { close(releaseLock) }) }
	defer release()
	lockHolderDone := make(chan error, 1)
	go func() {
		lockHolderDone <- store.ReadRows(context.Background(), "SELECT 1", nil, func(rows *sql.Rows) error {
			if !rows.Next() {
				return fmt.Errorf("lock-holder query returned no rows")
			}
			close(lockHeld)
			<-releaseLock
			return rows.Err()
		})
	}()
	<-lockHeld

	baseCtx, cancel := context.WithCancel(context.Background())
	ctx := &lifecycleGateObservedContext{
		Context:  baseCtx,
		observed: make(chan struct{}),
		target:   2, // lifecycle-gate admission, then the blocked database read
	}
	queueDone := make(chan error, 1)
	go func() {
		_, err := manager.QueueStartInstance(ctx, instance.ID)
		queueDone <- err
	}()

	<-ctx.observed
	cancel()
	select {
	case err := <-queueDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("QueueStartInstance error = %v, want context canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("QueueStartInstance did not honor cancellation while the instance read was blocked")
	}

	release()
	if err := <-lockHolderDone; err != nil {
		t.Fatal(err)
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
		t.Fatalf("canceled start changed desired status to %q", stored.DesiredStatus)
	}
}

func newPeerDrainAuthorizationCrashFixture(t *testing.T) *peerDrainAuthorizationCrashFixture {
	t.Helper()
	store := openProvisionerTestDB(t)
	instance := instanceWithTestLifecycleOwner(t, store, peerDrainAuthorizationTestInstance(t))
	insertInstanceForDeleteReceiptTest(t, store, &instance)
	operationID := db.MustNewUUIDv7()
	deleteOperation := instanceDeleteOperationIdentityForTest(t, store, operationID, instance, false)
	authorization, err := newInstancePeerDrainAuthorization(store, operationID, deleteOperation, instance, false)
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
		begin: swarmionapp.PeerDrainSnapshot{
			Active:                           true,
			RouteGenerationMatches:           true,
			LocalCheckpointCovered:           true,
			PreFenceHeartbeatIngressObserved: true,
			ReadyToFinalize:                  true,
		},
		finalize: swarmionapp.PeerDrainFinalizeResult{Finalized: true},
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
	manager.publishPeerDrainAuthorization = func(ctx context.Context, operation db.PublishedWriteOperation, expected InstanceInfo, fact tasks.OperationFact) (db.PublishedWriteReceipt, error) {
		fixture.publishPCalls++
		if !operation.Equal(authorization.publishedWriteOperation()) || !persistentInstanceEqual(expected, authorization.expectedInstance()) || fact.Kind != instancePeerDrainAuthorizationFact {
			return db.PublishedWriteReceipt{}, fmt.Errorf("unexpected P publication body")
		}
		return db.InsertAndUpdateWithOperationReceiptContext(
			ctx,
			store,
			operation,
			[]db.InsertMapper{createInstancePeerDrainAuthorizationFactCASMapper(fact, expected)},
			[]db.UpdateMapper{createInstancePeerDrainAuthorizationUpdateMapper(expected, fact)},
		)
	}
	manager.waitPeerDrainAuthorization = func(_ context.Context, receipt db.PublishedWriteReceipt, _ string) (db.EventReceiptObservation, error) {
		return db.EventReceiptObservation{
			Receipt: receipt,
			State:   db.EventReceiptStateAppliedDurably,
			Status: swarmionapp.ReceiptStatus{
				AppliedDurably:            true,
				Checkpointed:              true,
				CheckpointCommitID:        "event-P-checkpoint",
				DurableCheckpointCommitID: "later-durable-P-head",
			},
		}, nil
	}
	manager.verifyPeerDrainAuthorization = func(_ context.Context, checkpoint string, got instancePeerDrainAuthorization, fact tasks.OperationFact) error {
		if checkpoint != "event-P-checkpoint" || !sameInstancePeerDrainAuthorization(got, authorization) || fact.Kind != instancePeerDrainAuthorizationFact {
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
		func(instanceDeleteOperationReceipt, int, string) error { return nil },
	)
}

func (fixture *peerDrainAuthorizationCrashFixture) restartPeerDrainLifecycle(prefix string) (*fakeReplicationPeerDrainRuntime, *fakeReplicationPeerRouteFence) {
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
	authorization, err := newInstancePeerDrainAuthorization(store, operationID, deleteOperation, instance, false)
	if err != nil {
		t.Fatal(err)
	}
	receipt := db.PublishedWriteReceipt{
		EventID:               strings.Repeat("a", 64),
		PublishedRootHash:     strings.Repeat("b", 32),
		AuthorPeerID:          "wrong-author",
		OperationIntentDigest: authorization.Operation.IntentDigest(),
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
	authorization, err := newInstancePeerDrainAuthorization(store, operationID, deleteOperation, instance, false)
	if err != nil {
		t.Fatal(err)
	}
	if authorization.Operation.Key() == deleteOperation.Operation.Key() {
		t.Fatal("authorization P key was not domain-separated from delete D")
	}
	repeated, err := newInstancePeerDrainAuthorization(store, operationID, deleteOperation, instance, false)
	if err != nil {
		t.Fatal(err)
	}
	if repeated.Operation.Key() == authorization.Operation.Key() {
		t.Fatal("independent authorization operations reused a random key")
	}
	if repeated.Operation.IntentDigest() != authorization.Operation.IntentDigest() ||
		repeated.Operation.AuthorPeerID() != authorization.Operation.AuthorPeerID() {
		t.Fatalf("authorization immutable intent changed across random identities: first=%s repeated=%s", authorization.Operation.Identity, repeated.Operation.Identity)
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
		_, err := manager.QueueStartInstance(context.Background(), instance.ID)
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

func TestCurrentPeerDrainAuthorizationFactRejectsRemovedVersionField(t *testing.T) {
	store := openProvisionerTestDB(t)
	instance := instanceWithTestLifecycleOwner(t, store, peerDrainAuthorizationTestInstance(t))
	operationID := db.MustNewUUIDv7()
	deleteOperation := instanceDeleteOperationIdentityForTest(t, store, operationID, instance, false)
	authorization, err := newInstancePeerDrainAuthorization(store, operationID, deleteOperation, instance, false)
	if err != nil {
		t.Fatal(err)
	}
	fact, err := newInstancePeerDrainAuthorizationFact(authorization)
	if err != nil {
		t.Fatal(err)
	}
	fact.Payload = []byte(strings.TrimSuffix(string(fact.Payload), "}") + `,"version":"retired"}`)
	if _, err := instancePeerDrainAuthorizationFromFact(fact); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("decode authorization fact error = %v, want unknown-field rejection", err)
	}
}

func TestPeerDrainAuthorizationStaleSnapshotConsumesReceiptButCannotAuthorize(t *testing.T) {
	store := openProvisionerTestDB(t)
	instance := instanceWithTestLifecycleOwner(t, store, peerDrainAuthorizationTestInstance(t))
	insertInstanceForDeleteReceiptTest(t, store, &instance)
	operationID := db.MustNewUUIDv7()
	deleteOperation := instanceDeleteOperationIdentityForTest(t, store, operationID, instance, false)
	authorization, err := newInstancePeerDrainAuthorization(store, operationID, deleteOperation, instance, false)
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
	_, err = manager.completeInstancePeerDrainAuthorization(ctx, authorization, fact, db.PublishedWriteReceipt{}, false, 1)
	if !errors.Is(err, ErrInstanceDeleteInvariantConflict) && !errors.Is(err, db.ErrEventReceiptPending) {
		t.Fatalf("stale authorization error = %v, want fail-closed invariant/pending", err)
	}
	resolved, err := store.LookupPublishedWriteOperation(ctx, authorization.publishedWriteOperation())
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Disposition() != swarmionapp.OperationAccepted {
		t.Fatalf("stale no-change P disposition=%s diagnostic=%v, want accepted", resolved.Disposition(), resolved.Diagnostic())
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
	restartedRuntime.begin = swarmionapp.PeerDrainSnapshot{
		Active:                       true,
		RouteGenerationMatches:       true,
		LocalCheckpointCovered:       false,
		CheckpointCoverageReasonCode: swarmionapp.PeerDrainBlockingReasonPeerCheckpointNotCovered,
		BlockingReasonCodes: []swarmionapp.PeerDrainBlockingReason{
			swarmionapp.PeerDrainBlockingReasonPeerCheckpointNotCovered,
		},
	}
	started := make(chan struct{})
	restartedRuntime.afterStart = func() { close(started) }
	fixture.manager.afterPeerDrainAuthorized = nil
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { result <- fixture.runDeleteContext(ctx) }()
	select {
	case <-started:
	case <-time.After(30 * time.Second):
		cancel()
		t.Fatal("recovered delete did not reach the blocked replacement drain")
	}
	cancel()
	var err error
	select {
	case err = <-result:
	case <-time.After(30 * time.Second):
		t.Fatal("recovered delete did not stop after its blocked drain was canceled")
	}
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
