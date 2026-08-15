package provisioners

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/tasks"
	"github.com/protosio/protos/internal/testswarmion"
)

const simulatedDeleteReceiptProcessExit = "process exited after persisting delete receipt"

type deleteReceiptInterruption string

const (
	deleteReceiptInterruptionProcessExit  deleteReceiptInterruption = "process_exit"
	deleteReceiptInterruptionRunnerCancel deleteReceiptInterruption = "runner_cancel"
)

// interruptAfterPersistedDeleteReceiptTracker models an interruption at the
// narrow recovery boundary under test: deleteInstanceRecords has published the
// delete and the immutable exact-receipt fact, but the task has not made its
// first event-status query or recorded protocol completion. The mutable task
// payload deliberately has no receipt at this point; startup recovery must
// rebuild that projection from the fact after reopening.
type interruptAfterPersistedDeleteReceiptTracker struct {
	delegate     instanceDeleteReceiptTracker
	interruption deleteReceiptInterruption
	cancel       context.CancelFunc
	receipt      db.PublishedWriteReceipt
	waitCalls    int
}

func (tracker *interruptAfterPersistedDeleteReceiptTracker) WaitForPublishedWriteApplied(
	_ context.Context,
	receipt db.PublishedWriteReceipt,
	_ string,
) (db.EventReceiptObservation, error) {
	tracker.waitCalls++
	tracker.receipt = receipt
	if tracker.interruption == deleteReceiptInterruptionProcessExit {
		panic(simulatedDeleteReceiptProcessExit)
	}
	if tracker.cancel != nil {
		tracker.cancel()
	}
	return db.EventReceiptObservation{}, context.Canceled
}

func (tracker *interruptAfterPersistedDeleteReceiptTracker) InstanceExistsAtCheckpoint(
	ctx context.Context,
	checkpointCommitID string,
	instanceID string,
) (bool, error) {
	return tracker.delegate.InstanceExistsAtCheckpoint(ctx, checkpointCommitID, instanceID)
}

func TestInstanceDeleteTaskRestartResumesExactReceiptWithoutRepublishOrSequenceGap(t *testing.T) {
	testInstanceDeleteTaskRestartResumesExactReceipt(t, deleteReceiptInterruptionProcessExit)
}

func TestInstanceDeleteTaskRunnerCancellationPreservesExactReceiptForStartupRecovery(t *testing.T) {
	testInstanceDeleteTaskRestartResumesExactReceipt(t, deleteReceiptInterruptionRunnerCancel)
}

func TestInstanceDeleteTaskRestartRecoversOperationBeforeReceiptCheckpoint(t *testing.T) {
	t.Setenv("SWARMION_CHECKPOINT_SCHEDULER_PROFILE", "single_node_development")

	cfg := config.Get()
	previousP2PPort := cfg.P2PPort
	cfg.P2PPort = 0
	t.Cleanup(func() { cfg.P2PPort = previousP2PPort })

	workDir := t.TempDir()
	key, err := pcrypto.GetLocalKey(workDir)
	if err != nil {
		t.Fatal(err)
	}
	networkFixture := testswarmion.New(t, key)
	const databaseName = "protos_delete_operation_restart_test"
	store, err := db.Open(workDir, databaseName, key, networkFixture.Link)
	if err != nil {
		t.Fatal(err)
	}
	activeStore := store
	t.Cleanup(func() {
		if activeStore != nil {
			_ = activeStore.Close()
		}
	})
	if err := store.Init(); err != nil {
		t.Fatal(err)
	}

	instance := InstanceInfo{
		ID:                  db.MustNewUUIDv7(),
		Name:                "restart-before-receipt-checkpoint",
		Kind:                "receipt_test_record",
		KindID:              "restart-delete-operation-test",
		ProviderResourceID:  "restart-delete-operation-provider-id",
		DesiredStatus:       ServerStateRunning,
		ReplicationPriority: 1,
		Location:            "test",
	}
	insertInstanceForDeleteReceiptTest(t, store, &instance)

	firstManager := newLifecycleTestManager(t, store, newProvisionerRegistry())
	record, err := firstManager.QueueDeleteInstanceLocal(context.Background(), instance.ID)
	if err != nil {
		t.Fatal(err)
	}
	queuedPayload := decodeDeleteRestartPayload(t, record)
	if queuedPayload.DeleteOperation == nil {
		t.Fatal("queued delete task did not replicate its operation identity")
	}
	operationIdentity := *queuedPayload.DeleteOperation
	if queuedPayload.OperationStateModel != instanceDeleteOperationFactsV1 {
		t.Fatalf("queued operation state model=%q, want %q", queuedPayload.OperationStateModel, instanceDeleteOperationFactsV1)
	}
	var accepted db.PublishedWriteReceipt
	firstManager.afterInstanceDeletePublished = func(receipt db.PublishedWriteReceipt) {
		accepted = receipt
		panic("process exited before delete receipt fact")
	}
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		_ = firstManager.tasks.RunPending(context.Background())
	}()
	if recovered != "process exited before delete receipt fact" {
		interrupted, getErr := firstManager.tasks.Get(record.ID)
		t.Fatalf(
			"delete task interruption=%v, want post-publication process exit; task status=%q attempts=%d error=%q get_error=%v",
			recovered,
			interrupted.Status,
			interrupted.Attempts,
			interrupted.ErrorMessage,
			getErr,
		)
	}
	if accepted.EventID == "" || accepted.PublishedRootHash == "" {
		t.Fatalf("post-publication interruption lost accepted receipt: %+v", accepted)
	}
	interrupted, err := firstManager.tasks.Get(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	interruptedPayload := decodeDeleteRestartPayload(t, interrupted)
	if interruptedPayload.DeleteOperation == nil || interruptedPayload.DeleteReceipt != nil ||
		interruptedPayload.OperationStateModel != instanceDeleteOperationFactsV1 {
		t.Fatalf("interrupted task payload=%+v, want operation identity without returned receipt", interruptedPayload)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close database for operation-receipt restart: %v", err)
	}
	activeStore = nil
	store, err = db.Open(workDir, databaseName, key, networkFixture.Link)
	if err != nil {
		t.Fatalf("reopen database after operation-receipt restart: %v", err)
	}
	activeStore = store

	resolved, err := store.LookupPublishedWriteOperation(context.Background(), operationIdentity.publishedWriteOperation())
	if err != nil {
		t.Fatalf("resolve delete operation after restart: %v", err)
	}
	recoveredReceipt, err := db.PublishedWriteReceiptFromResolution(resolved)
	if err != nil {
		t.Fatalf("convert recovered delete operation receipt: %v", err)
	}
	if recoveredReceipt.EventID != accepted.EventID || recoveredReceipt.PublishedRootHash != accepted.PublishedRootHash {
		t.Fatalf("recovered receipt=%+v, want accepted %+v", recoveredReceipt, accepted)
	}
	var immediateStatus string
	if err := store.GetSqlDB().QueryRowContext(
		context.Background(),
		"SELECT status FROM tasks WHERE id = ?",
		db.MustUUIDBytes(record.ID),
	).Scan(&immediateStatus); err != nil {
		t.Fatalf("read interrupted task immediately after Open without readiness polling: %v", err)
	}
	if immediateStatus != string(tasks.StatusRunning) {
		t.Fatalf("task status immediately after Open=%q, want recovered %q", immediateStatus, tasks.StatusRunning)
	}

	restartedManager := newLifecycleTestManager(t, store, newProvisionerRegistry())
	effectFact := waitForDeleteRestartOperationFact(t, restartedManager.tasks, record.ID, tasks.OperationFactKindEffect)
	assertDeleteRestartEffectFact(t, effectFact, operationIdentity)
	if receiptFact, found, err := restartedManager.tasks.OperationFact(context.Background(), record.ID, tasks.OperationFactKindReceiptV2); err != nil {
		t.Fatalf("read pre-recovery delete receipt fact: %v", err)
	} else if found {
		t.Fatalf("delete receipt fact unexpectedly existed before startup recovery: %+v", receiptFact)
	}
	var duplicateDeleteBodies atomic.Int32
	restartedManager.afterInstanceDeletePublished = func(db.PublishedWriteReceipt) {
		duplicateDeleteBodies.Add(1)
	}
	beforeRestartWork := store.TransactionMetrics()
	stop := restartedManager.tasks.Start(context.Background(), 10*time.Millisecond)
	done := waitForDeleteRestartTaskSucceeded(t, restartedManager.tasks, record.ID)
	if err := stop(); err != nil {
		t.Fatalf("stop restarted task runner: %v", err)
	}
	if duplicateDeleteBodies.Load() != 0 {
		t.Fatalf("restart executed %d duplicate delete SQL bodies", duplicateDeleteBodies.Load())
	}
	restartMetrics := transactionMetricsDeltaForDeleteRestart(store.TransactionMetrics(), beforeRestartWork)
	if restartMetrics.TypedConflicts != 0 || restartMetrics.CommitsFailed != 0 ||
		restartMetrics.SQLViewNotReadyOutcomes != 0 || restartMetrics.RollbacksAttempted != 0 ||
		restartMetrics.RollbacksSucceeded != 0 || restartMetrics.RollbacksFailed != 0 {
		t.Fatalf("restart transaction metrics=%+v, want immediate readiness with no conflicts or rollbacks", restartMetrics)
	}
	t.Logf(
		"restart readiness metrics conflicts=%d readiness=%d rollbacks=%d/%d metrics=%+v",
		restartMetrics.TypedConflicts,
		restartMetrics.SQLViewNotReadyOutcomes,
		restartMetrics.RollbacksAttempted,
		restartMetrics.TransactionsStarted,
		restartMetrics,
	)
	finalPayload := decodeDeleteRestartPayload(t, done)
	if finalPayload.DeleteReceipt != nil {
		t.Fatalf("v2-only recovery projected a downgrade-unsafe receipt into the task payload: %+v", finalPayload.DeleteReceipt)
	}
	if done.Attempts != 1 {
		t.Fatalf("recovered task attempts=%d, want one resumed logical attempt", done.Attempts)
	}
	finalPayload = decodeDeleteRestartPayload(t, done)
	if finalPayload.OperationStateModel != instanceDeleteOperationFactsV1 {
		t.Fatalf("recovered task changed its immutable operation state model: %+v", finalPayload)
	}
	receiptFact := waitForDeleteRestartOperationFact(t, restartedManager.tasks, record.ID, tasks.OperationFactKindReceiptV2)
	assertDeleteRestartReceiptFact(
		t,
		receiptFact,
		operationIdentity,
		instanceDeleteReceiptFromPublished(record.ID, operationIdentity, accepted),
	)
}

func transactionMetricsDeltaForDeleteRestart(after, before db.TransactionMetricsSnapshot) db.TransactionMetricsSnapshot {
	return db.TransactionMetricsSnapshot{
		TransactionsStarted:                  after.TransactionsStarted - before.TransactionsStarted,
		CommitsAttempted:                     after.CommitsAttempted - before.CommitsAttempted,
		CommitsSucceeded:                     after.CommitsSucceeded - before.CommitsSucceeded,
		CommitsFailed:                        after.CommitsFailed - before.CommitsFailed,
		NoopCommitOutcomes:                   after.NoopCommitOutcomes - before.NoopCommitOutcomes,
		RollbacksAttempted:                   after.RollbacksAttempted - before.RollbacksAttempted,
		RollbacksSucceeded:                   after.RollbacksSucceeded - before.RollbacksSucceeded,
		RollbacksFailed:                      after.RollbacksFailed - before.RollbacksFailed,
		RollbacksApplyPhase:                  after.RollbacksApplyPhase - before.RollbacksApplyPhase,
		RollbacksBeforeCommitPhase:           after.RollbacksBeforeCommitPhase - before.RollbacksBeforeCommitPhase,
		RollbacksPanicPhase:                  after.RollbacksPanicPhase - before.RollbacksPanicPhase,
		RollbacksApplyFailure:                after.RollbacksApplyFailure - before.RollbacksApplyFailure,
		RollbacksContextCanceled:             after.RollbacksContextCanceled - before.RollbacksContextCanceled,
		RollbacksContextDeadline:             after.RollbacksContextDeadline - before.RollbacksContextDeadline,
		RollbacksSQLViewNotReady:             after.RollbacksSQLViewNotReady - before.RollbacksSQLViewNotReady,
		RollbacksPanic:                       after.RollbacksPanic - before.RollbacksPanic,
		TypedConflicts:                       after.TypedConflicts - before.TypedConflicts,
		OperationReceiptsFoundAfterCommitErr: after.OperationReceiptsFoundAfterCommitErr - before.OperationReceiptsFoundAfterCommitErr,
		UncertainEventReceiptsAfterCommitErr: after.UncertainEventReceiptsAfterCommitErr - before.UncertainEventReceiptsAfterCommitErr,
		SQLViewNotReadyOutcomes:              after.SQLViewNotReadyOutcomes - before.SQLViewNotReadyOutcomes,
		OperationTransactionsAttempted:       after.OperationTransactionsAttempted - before.OperationTransactionsAttempted,
		OperationTransactionsExecuted:        after.OperationTransactionsExecuted - before.OperationTransactionsExecuted,
		OperationTransactionsAlreadyAccepted: after.OperationTransactionsAlreadyAccepted - before.OperationTransactionsAlreadyAccepted,
		OperationTransactionsNoChange:        after.OperationTransactionsNoChange - before.OperationTransactionsNoChange,
		OperationTransactionsFailed:          after.OperationTransactionsFailed - before.OperationTransactionsFailed,
		OperationWorkspaceDirtyOutcomes:      after.OperationWorkspaceDirtyOutcomes - before.OperationWorkspaceDirtyOutcomes,
	}
}

func testInstanceDeleteTaskRestartResumesExactReceipt(t *testing.T, interruption deleteReceiptInterruption) {
	t.Helper()
	t.Setenv("SWARMION_CHECKPOINT_SCHEDULER_PROFILE", "single_node_development")

	cfg := config.Get()
	previousP2PPort := cfg.P2PPort
	cfg.P2PPort = 0
	t.Cleanup(func() {
		cfg.P2PPort = previousP2PPort
	})

	workDir := t.TempDir()
	key, err := pcrypto.GetLocalKey(workDir)
	if err != nil {
		t.Fatal(err)
	}
	networkFixture := testswarmion.New(t, key)
	const databaseName = "protos_delete_receipt_restart_test"

	store, err := db.Open(workDir, databaseName, key, networkFixture.Link)
	if err != nil {
		t.Fatal(err)
	}
	activeStore := store
	t.Cleanup(func() {
		if activeStore != nil {
			_ = activeStore.Close()
		}
	})
	if err := store.Init(); err != nil {
		t.Fatal(err)
	}

	instance := InstanceInfo{
		ID:                  db.MustNewUUIDv7(),
		Name:                "restart-delete-receipt",
		Kind:                "receipt_test_record",
		KindID:              "restart-delete-receipt-test",
		ProviderResourceID:  "restart-delete-provider-id",
		DesiredStatus:       ServerStateRunning,
		ReplicationPriority: 1,
		Location:            "test",
	}
	insertInstanceForDeleteReceiptTest(t, store, &instance)

	firstManager := newLifecycleTestManager(t, store, newProvisionerRegistry())
	record, err := firstManager.QueueDeleteInstanceLocal(context.Background(), instance.ID)
	if err != nil {
		t.Fatal(err)
	}
	queuedPayload := decodeDeleteRestartPayload(t, record)
	if queuedPayload.DeleteOperation == nil {
		t.Fatal("queued delete task did not replicate its operation identity")
	}
	operationIdentity := *queuedPayload.DeleteOperation
	if queuedPayload.OperationStateModel != instanceDeleteOperationFactsV1 {
		t.Fatalf("queued operation state model=%q, want %q", queuedPayload.OperationStateModel, instanceDeleteOperationFactsV1)
	}
	if queuedPayload.DeleteReceipt != nil {
		t.Fatalf("queued immutable-fact delete unexpectedly has a receipt: %+v", queuedPayload)
	}
	realTracker := swarmionInstanceDeleteReceiptTracker{database: store}
	interruptTracker := &interruptAfterPersistedDeleteReceiptTracker{
		delegate:     realTracker,
		interruption: interruption,
	}
	firstManager.deleteReceiptTracker = interruptTracker

	var recoveredPanic any
	var interruptedRunErr error
	lastPersistedAttempts := 0
	interrupted := false
	for schedulerRun := 1; schedulerRun <= instanceDeleteMaxAttempts*2; schedulerRun++ {
		recoveredPanic = nil
		interruptedRunErr = nil
		runCtx := context.Background()
		cancelRun := func() {}
		if interruption == deleteReceiptInterruptionRunnerCancel {
			var cancel context.CancelFunc
			runCtx, cancel = context.WithCancel(context.Background())
			cancelRun = cancel
			interruptTracker.cancel = cancel
		}
		func() {
			defer func() {
				recoveredPanic = recover()
			}()
			interruptedRunErr = firstManager.tasks.RunPending(runCtx)
		}()
		cancelRun()
		if recoveredPanic != nil {
			interrupted = true
			break
		}
		if interruption == deleteReceiptInterruptionRunnerCancel && interruptTracker.waitCalls > 0 {
			interrupted = true
			break
		}
		if interruptedRunErr == nil {
			t.Fatal("delete task completed without reaching simulated process exit")
		}

		// A pre-publication staged-root conflict is handled by the lifecycle
		// task's existing bounded retry policy. Receipt authority never moves into
		// this mutable row: once accepted, it lives in the immutable fact table.
		retrying, getErr := firstManager.tasks.Get(record.ID)
		if getErr != nil {
			t.Fatal(getErr)
		}
		retryingPayload := decodeDeleteRestartPayload(t, retrying)
		if retryingPayload.OperationStateModel != instanceDeleteOperationFactsV1 || retryingPayload.DeleteReceipt != nil {
			t.Fatalf("pre-publication retry introduced mutable receipt authority: %+v", retryingPayload)
		}
		if retrying.Status != tasks.StatusPending {
			t.Fatalf("pre-publication retry status=%s attempts=%d error=%v, want pending", retrying.Status, retrying.Attempts, interruptedRunErr)
		}
		if retrying.Attempts < lastPersistedAttempts {
			t.Fatalf("pre-publication attempts rolled back %d->%d: %v", lastPersistedAttempts, retrying.Attempts, interruptedRunErr)
		}
		lastPersistedAttempts = retrying.Attempts
	}
	if !interrupted {
		t.Fatalf("delete task did not reach %s interruption: last error=%v", interruption, interruptedRunErr)
	}
	switch interruption {
	case deleteReceiptInterruptionProcessExit:
		if recoveredPanic != simulatedDeleteReceiptProcessExit {
			t.Fatalf("delete run panic=%v error=%v, want simulated process exit", recoveredPanic, interruptedRunErr)
		}
	case deleteReceiptInterruptionRunnerCancel:
		if recoveredPanic != nil || interruptedRunErr == nil || !strings.Contains(interruptedRunErr.Error(), context.Canceled.Error()) {
			t.Fatalf("cancelled delete run panic=%v error=%v, want runner cancellation", recoveredPanic, interruptedRunErr)
		}
	}
	if interruptTracker.waitCalls != 1 || interruptTracker.receipt.EventID == "" || interruptTracker.receipt.PublishedRootHash == "" {
		t.Fatalf("pre-restart delete receipt tracker calls=%d receipt=%+v", interruptTracker.waitCalls, interruptTracker.receipt)
	}
	interruptedRecord, err := firstManager.tasks.Get(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	interruptedPayload := decodeDeleteRestartPayload(t, interruptedRecord)
	if interruptedPayload.DeleteReceipt != nil || interruptedPayload.OperationStateModel != instanceDeleteOperationFactsV1 ||
		interruptedPayload.DeleteOperation == nil || *interruptedPayload.DeleteOperation != operationIdentity {
		t.Fatalf("interrupted immutable-fact task retained mutable receipt state: record=%+v payload=%+v", interruptedRecord, interruptedPayload)
	}
	originalReceipt := instanceDeleteReceiptFromPublished(record.ID, operationIdentity, interruptTracker.receipt)

	// Close and reopen the actual Swarmion-backed database. In particular, this
	// exercises restoration of authored event sequence state and the immutable
	// operation facts; merely constructing another task manager on the same live
	// DB would not cover that boundary.
	if err := store.Close(); err != nil {
		t.Fatalf("close database for restart: %v", err)
	}
	activeStore = nil
	store, err = db.Open(workDir, databaseName, key, networkFixture.Link)
	if err != nil {
		t.Fatalf("reopen database after restart: %v", err)
	}
	activeStore = store

	restartedManager := newLifecycleTestManager(t, store, newProvisionerRegistry())
	effectFact := waitForDeleteRestartOperationFact(t, restartedManager.tasks, record.ID, tasks.OperationFactKindEffect)
	assertDeleteRestartEffectFact(t, effectFact, operationIdentity)
	receiptFact := waitForDeleteRestartOperationFact(t, restartedManager.tasks, record.ID, tasks.OperationFactKindReceiptV2)
	assertDeleteRestartReceiptFact(t, receiptFact, operationIdentity, originalReceipt)

	restored, err := restartedManager.tasks.Get(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Status != tasks.StatusRunning || restored.Attempts < 1 || restored.Attempts > instanceDeleteMaxAttempts {
		t.Fatalf("restored task status=%s attempts=%d, want running with a bounded positive attempt", restored.Status, restored.Attempts)
	}
	interruptedAttempt := restored.Attempts
	restoredPayload := decodeDeleteRestartPayload(t, restored)
	if restoredPayload.DeleteReceipt != nil || restoredPayload.OperationStateModel != instanceDeleteOperationFactsV1 ||
		restoredPayload.DeleteOperation == nil || *restoredPayload.DeleteOperation != operationIdentity {
		t.Fatalf("restored mutable task row became receipt authority before startup recovery: %+v", restoredPayload)
	}

	// The receipt fact exists before startup recovery begins. Settle the restored
	// outbox only to establish an exact event-count baseline; the task row must
	// remain unchanged until the recovery hook projects the fact into it.
	baseline := waitForDeleteRestartRuntimeQuiescent(t, store, "settle restored delete outbox")
	baselineTaskEvents := deleteRestartTaskEventCount(t, store, record.ID)
	baselineAuthoredEventIDs := deleteRestartAuthoredEventIDs(baseline)

	stop := restartedManager.tasks.Start(context.Background(), 10*time.Millisecond)
	defer func() {
		if err := stop(); err != nil {
			t.Errorf("stop restarted task runner: %v", err)
		}
	}()
	done := waitForDeleteRestartTaskSucceeded(t, restartedManager.tasks, record.ID)
	if err := stop(); err != nil {
		t.Fatalf("stop restarted task runner: %v", err)
	}
	stop = func() error { return nil }
	if done.Attempts != interruptedAttempt {
		t.Fatalf("restarted task attempts=%d, want resumed logical attempt %d", done.Attempts, interruptedAttempt)
	}

	finalPayload := decodeDeleteRestartPayload(t, done)
	if finalPayload.DeleteReceipt != nil || finalPayload.OperationStateModel != instanceDeleteOperationFactsV1 {
		t.Fatalf("completed v2-only task retained a downgrade-unsafe payload receipt: %+v", finalPayload)
	}
	var result instanceLifecycleTaskResult
	if err := json.Unmarshal(done.Result, &result); err != nil {
		t.Fatal(err)
	}
	if !result.Deleted || result.DeleteReceipt == nil ||
		result.DeleteReceipt.EventID != originalReceipt.EventID ||
		result.DeleteReceipt.PublishedRootHash != originalReceipt.PublishedRootHash {
		t.Fatalf("completed result did not retain exact delete receipt: %+v", result)
	}
	finalReceiptFact := waitForDeleteRestartOperationFact(t, restartedManager.tasks, record.ID, tasks.OperationFactKindReceiptV2)
	assertDeleteRestartReceiptFact(t, finalReceiptFact, operationIdentity, originalReceipt)

	// Fact-first startup recovery performs exactly four task publications:
	// running->pending recovery (which deliberately keeps a v2-only receipt out
	// of the payload), mark-running, the ordinary receipt-free "deleting"
	// progress update, and task success. Reading or re-recording identical
	// immutable facts publishes nothing. Compare task rows, checkpoint events,
	// and authored event IDs one-for-one: any extra event would be a duplicate
	// domain or receipt-fact
	// publication. Requiring the clock to drain also proves that the restored
	// author sequence continued without an unfillable gap.
	afterRecovery := waitForDeleteRestartRuntimeQuiescent(t, store, "settle restarted delete task")
	recoveryTaskEventDelta := deleteRestartTaskEventCount(t, store, record.ID) - baselineTaskEvents
	recoveryCheckpointDelta := afterRecovery.CheckpointEventCount - baseline.CheckpointEventCount
	recoveryAuthoredEventIDs := deleteRestartNewAuthoredEventIDs(baselineAuthoredEventIDs, afterRecovery)
	if recoveryTaskEventDelta != 4 ||
		recoveryCheckpointDelta != recoveryTaskEventDelta ||
		len(recoveryAuthoredEventIDs) != recoveryTaskEventDelta {
		t.Fatalf(
			"restart task/checkpoint/authored deltas=%d/%d/%d ids=%v, want exactly 4 one-for-one task-only events",
			recoveryTaskEventDelta,
			recoveryCheckpointDelta,
			len(recoveryAuthoredEventIDs),
			recoveryAuthoredEventIDs,
		)
	}
	if _, err := restartedManager.GetInstance(instance.ID); err == nil {
		t.Fatalf("deleted instance %s is present after receipt recovery", instance.ID)
	}

	// Publish one more real application event after recovery. Its exact receipt
	// must reach applied_durably as the immediately next checkpoint event; this
	// is the behavioral proof that restart did not leave an author-sequence gap.
	continuation := InstanceInfo{
		ID:                  db.MustNewUUIDv7(),
		Name:                "restart-sequence-continuation",
		Kind:                KindLocalVM,
		KindID:              "restart-delete-receipt-test",
		ProviderResourceID:  "restart-sequence-provider-id",
		DesiredStatus:       ServerStateRunning,
		ReplicationPriority: 1,
		Location:            "test",
	}
	continuationReceipt := insertInstanceForDeleteReceiptTest(t, store, &continuation)
	if continuationReceipt.EventID == originalReceipt.EventID {
		t.Fatalf("post-restart event reused delete event ID %s", originalReceipt.EventID)
	}
	afterContinuation := waitForDeleteRestartRuntimeQuiescent(t, store, "settle post-restart sequence continuation")
	if delta := afterContinuation.CheckpointEventCount - afterRecovery.CheckpointEventCount; delta != 1 {
		t.Fatalf("post-restart checkpoint event delta=%d, want exactly one contiguous continuation event", delta)
	}
}

func decodeDeleteRestartPayload(t *testing.T, record tasks.Record) instanceLifecycleTaskPayload {
	t.Helper()
	var payload instanceLifecycleTaskPayload
	if err := json.Unmarshal(record.Payload, &payload); err != nil {
		t.Fatalf("decode task %s payload: %v", record.ID, err)
	}
	return payload
}

func deleteRestartTaskEventCount(t *testing.T, store *db.DB, taskID string) int {
	t.Helper()
	var count int
	if err := store.ReadRows(
		context.Background(),
		"SELECT COUNT(*) FROM task_events WHERE task_id = ?",
		[]any{db.MustUUIDBytes(taskID)},
		func(rows *sql.Rows) error {
			if !rows.Next() {
				return fmt.Errorf("task event count query returned no rows")
			}
			return rows.Scan(&count)
		},
	); err != nil {
		t.Fatalf("count task events for %s: %v", taskID, err)
	}
	return count
}

func waitForDeleteRestartOperationFact(
	t *testing.T,
	manager *tasks.Manager,
	taskID string,
	kind string,
) tasks.OperationFact {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	var last string
	for {
		fact, found, err := manager.OperationFact(context.Background(), taskID, kind)
		if err == nil && found {
			return fact
		}
		if err != nil {
			last = err.Error()
		} else {
			last = "not visible"
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for immutable delete operation fact %s/%s: %s", taskID, kind, last)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func assertDeleteRestartEffectFact(
	t *testing.T,
	fact tasks.OperationFact,
	expected instanceDeleteOperationIdentity,
) {
	t.Helper()
	actual, err := instanceDeleteIdentityFromEffectFact(fact)
	if err != nil {
		t.Fatalf("decode immutable delete effect fact: %v", err)
	}
	if actual != expected {
		t.Fatalf("immutable delete effect fact identity=%+v, want %+v", actual, expected)
	}
}

func assertDeleteRestartReceiptFact(
	t *testing.T,
	fact tasks.OperationFact,
	identity instanceDeleteOperationIdentity,
	expected instanceDeleteOperationReceipt,
) instanceDeleteOperationReceipt {
	t.Helper()
	actual, err := instanceDeleteReceiptFromFact(fact, identity)
	if err != nil {
		t.Fatalf("decode immutable delete receipt fact: %v", err)
	}
	if err := validateInstanceDeleteOperationReceipt(
		actual,
		identity,
		expected.OperationID,
		expected.ExpectedInvariant.InstanceID,
	); err != nil {
		t.Fatalf("validate immutable delete receipt fact: %v", err)
	}
	if actual.OperationID != expected.OperationID ||
		actual.Operation != expected.Operation ||
		actual.ExpectedInvariant != expected.ExpectedInvariant ||
		actual.EventID != expected.EventID ||
		actual.PublishedRootHash != expected.PublishedRootHash ||
		actual.OperationIntentDigest != expected.OperationIntentDigest ||
		actual.OperationAuthorPeerID != expected.OperationAuthorPeerID {
		t.Fatalf("immutable delete receipt fact=%+v, want exact accepted identity %+v", actual, expected)
	}
	switch fact.Kind {
	case tasks.OperationFactKindReceipt:
		if actual.EventDigest != expected.EventDigest || actual.AuthorSeq != expected.AuthorSeq {
			t.Fatalf("legacy delete receipt metadata=%q/%d, want %q/%d", actual.EventDigest, actual.AuthorSeq, expected.EventDigest, expected.AuthorSeq)
		}
	case tasks.OperationFactKindReceiptV2:
		if actual.EventDigest != "" || actual.AuthorSeq != 0 {
			t.Fatalf("v2 delete receipt retained legacy metadata: %+v", actual)
		}
	default:
		t.Fatalf("unexpected delete receipt fact kind %q", fact.Kind)
	}
	return actual
}

func waitForDeleteRestartTaskSucceeded(t *testing.T, manager *tasks.Manager, taskID string) tasks.Record {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for {
		record, err := manager.Get(taskID)
		if err != nil {
			t.Fatal(err)
		}
		switch record.Status {
		case tasks.StatusSucceeded:
			return record
		case tasks.StatusFailed, tasks.StatusCancelled:
			t.Fatalf("restarted delete task ended as %s: %s", record.Status, record.ErrorMessage)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for restarted delete task: status=%s message=%s error=%s", record.Status, record.Message, record.ErrorMessage)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func deleteRestartAuthoredEventIDs(status swarmionapp.Status) map[string]struct{} {
	result := make(map[string]struct{}, len(status.RecentAuthoredWrites))
	for _, write := range status.RecentAuthoredWrites {
		if eventID := strings.TrimSpace(write.EventID); eventID != "" {
			result[eventID] = struct{}{}
		}
	}
	return result
}

func deleteRestartNewAuthoredEventIDs(before map[string]struct{}, after swarmionapp.Status) []string {
	var result []string
	for eventID := range deleteRestartAuthoredEventIDs(after) {
		if _, found := before[eventID]; !found {
			result = append(result, eventID)
		}
	}
	sort.Strings(result)
	return result
}

func waitForDeleteRestartRuntimeQuiescent(t *testing.T, store *db.DB, reason string) swarmionapp.Status {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	var last swarmionapp.Status
	for {
		status, ok := store.SwarmionStatus()
		if ok {
			last = status
			if status.ClockEvents == 0 &&
				status.EventQueuePendingCount == 0 &&
				status.EventClosurePendingCount == 0 &&
				status.EventClosureActiveCount == 0 {
				return status
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf(
				"timed out waiting for Swarmion quiescence (%s): clock=%d queue=%d closure_pending=%d closure_active=%d checkpoints=%d",
				reason,
				last.ClockEvents,
				last.EventQueuePendingCount,
				last.EventClosurePendingCount,
				last.EventClosureActiveCount,
				last.CheckpointEventCount,
			)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
