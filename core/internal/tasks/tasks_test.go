package tasks

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/testswarmion"
)

type testPayload struct {
	Value string `json:"value"`
}

type testResult struct {
	Done string `json:"done"`
}

type checkpointPayload struct {
	Value   string `json:"value"`
	Receipt string `json:"receipt,omitempty"`
}

type scriptedTaskWritePublisher struct {
	insertConfirmation db.PublishedWriteConfirmation
	insertErr          error
	updateConfirmation db.PublishedWriteConfirmation
	updateErr          error
	insertCalls        int
	updateCalls        int
	insertContext      context.Context
}

type doneObservedContext struct {
	context.Context
	once     sync.Once
	observed chan struct{}
}

func (ctx *doneObservedContext) Done() <-chan struct{} {
	ctx.once.Do(func() { close(ctx.observed) })
	return ctx.Context.Done()
}

func unresolvedTaskWriteForTest() (db.PublishedWriteConfirmation, error) {
	confirmation := taskWriteConfirmationForTest("", false)
	return confirmation, &db.PublishedWriteConfirmationUnresolvedError{
		Confirmation: confirmation,
		Cause:        errors.New("injected exact publication outcome loss"),
	}
}

func (publisher *scriptedTaskWritePublisher) Insert(
	ctx context.Context,
	_ ...db.InsertMapper,
) (db.PublishedWriteConfirmation, error) {
	publisher.insertCalls++
	publisher.insertContext = ctx
	return publisher.insertConfirmation, publisher.insertErr
}

func TestEnqueueContextForwardsCallerContextToAvailabilityPublisher(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)
	publisher := &scriptedTaskWritePublisher{
		insertConfirmation: taskWriteConfirmationForTest(db.PublishedWriteConfirmationOtherPeerAvailable, false),
	}
	manager.taskWrites = publisher

	type contextKey string
	ctx := context.WithValue(context.Background(), contextKey("request"), "apic")
	_, err := EnqueueContext(ctx, manager, EnqueueOptions[testPayload]{
		Stream:      "test.context",
		SubjectType: "test-subject",
		SubjectID:   "subject-context",
		Payload:     testPayload{Value: "input"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if publisher.insertContext == nil || publisher.insertContext.Value(contextKey("request")) != "apic" {
		t.Fatal("enqueue did not forward the caller context to the availability publisher")
	}
}

func TestEnqueueUniqueContextCancellationInterruptsBlockedDedupRead(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)
	publisher := &scriptedTaskWritePublisher{
		insertConfirmation: taskWriteConfirmationForTest(db.PublishedWriteConfirmationOtherPeerAvailable, false),
	}
	manager.taskWrites = publisher

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
	ctx := &doneObservedContext{Context: baseCtx, observed: make(chan struct{})}
	type enqueueResult struct {
		err error
	}
	enqueueDone := make(chan enqueueResult, 1)
	go func() {
		_, _, err := EnqueueUniqueContext(ctx, manager, EnqueueUniqueOptions[testPayload]{
			EnqueueOptions: EnqueueOptions[testPayload]{
				Stream:      "test.unique-context",
				SubjectType: "test-subject",
				SubjectID:   "subject-unique-context",
				Payload:     testPayload{Value: "input"},
			},
		})
		enqueueDone <- enqueueResult{err: err}
	}()

	<-ctx.observed
	cancel()
	result := <-enqueueDone
	if !errors.Is(result.err, context.Canceled) {
		t.Fatalf("unique enqueue error = %v, want context canceled", result.err)
	}
	if publisher.insertCalls != 0 {
		t.Fatalf("canceled unique enqueue publications = %d, want 0", publisher.insertCalls)
	}

	release()
	if err := <-lockHolderDone; err != nil {
		t.Fatal(err)
	}
	if task, found, err := manager.LatestForSubject("test.unique-context", "test-subject", "subject-unique-context"); err != nil {
		t.Fatal(err)
	} else if found {
		t.Fatalf("canceled unique enqueue persisted task %+v", task)
	}
}

func TestEnqueueExactUnresolvedReturnsAndCachesPopulatedRecordWithoutReplay(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)
	confirmation, unresolvedErr := unresolvedTaskWriteForTest()
	publisher := &scriptedTaskWritePublisher{
		insertConfirmation: confirmation,
		insertErr:          unresolvedErr,
	}
	manager.taskWrites = publisher

	record, err := EnqueueContext(context.Background(), manager, EnqueueOptions[testPayload]{
		Stream:      "test.enqueue-unresolved",
		SubjectType: "test-subject",
		SubjectID:   "subject-enqueue-unresolved",
		Payload:     testPayload{Value: "input"},
	})
	if !errors.Is(err, db.ErrPublishedWriteConfirmationUnresolved) {
		t.Fatalf("enqueue error=%v, want exact unresolved classification", err)
	}
	if record.ID == "" || record.Status != StatusPending || record.WriteConfirmation.Stage != "" ||
		record.WriteConfirmation.EventID != confirmation.Receipt.EventID ||
		record.WriteConfirmation.PublishedRootHash != confirmation.Receipt.PublishedRootHash {
		t.Fatalf("enqueue record=%+v, want populated pending record with exact empty-stage confirmation", record)
	}
	if publisher.insertCalls != 1 {
		t.Fatalf("enqueue publications=%d, want exactly one", publisher.insertCalls)
	}
	latest, found := manager.LatestWriteConfirmation(record.ID)
	if !found || !writeConfirmationsEqual(latest, record.WriteConfirmation) {
		t.Fatalf("cached confirmation=%+v found=%t, want %+v", latest, found, record.WriteConfirmation)
	}
	progress, found := manager.LatestProgress(record.ID)
	if !found || progress.Durable || !writeConfirmationsEqual(progress.WriteConfirmation, record.WriteConfirmation) {
		t.Fatalf("unresolved progress=%+v found=%t, want non-durable exact confirmation", progress, found)
	}
}

func (publisher *scriptedTaskWritePublisher) UpdateAndInsert(
	_ context.Context,
	_ []db.UpdateMapper,
	_ []db.InsertMapper,
) (db.PublishedWriteConfirmation, error) {
	publisher.updateCalls++
	return publisher.updateConfirmation, publisher.updateErr
}

func taskWriteConfirmationForTest(stage db.PublishedWriteConfirmationStage, pending bool) db.PublishedWriteConfirmation {
	available := stage == db.PublishedWriteConfirmationOtherPeerAvailable
	confirmed := 0
	if available {
		confirmed = 1
	}
	return db.PublishedWriteConfirmation{
		Receipt: db.PublishedWriteReceipt{
			Committed:         true,
			EventID:           "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			PublishedRootHash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		},
		Stage:               stage,
		Availability:        swarmionOtherPeerRetentionStatusForTaskTest(available, confirmed),
		AvailabilityPending: pending,
	}
}

func swarmionOtherPeerRetentionStatusForTaskTest(available bool, confirmed int) swarmionapp.OtherPeerRetentionStatus {
	status := swarmionapp.OtherPeerRetentionStatus{
		RequiredOtherPeers:  1,
		ConfirmedOtherPeers: confirmed,
		CandidateScope:      swarmionapp.OtherPeerRetentionCandidateScopeCurrentLogicalPeers,
		Available:           available,
	}
	if available {
		status.EligiblePeerIDs = []string{"peer-b"}
		return status
	}
	status.NoCurrentEligiblePeers = true
	status.ReasonCode = swarmionapp.OtherPeerRetentionReasonNoCurrentEligiblePeers
	return status
}

func waitForTaskStatus(t *testing.T, manager *Manager, taskID string, status Status) Record {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		record, err := manager.Get(taskID)
		if err != nil {
			t.Fatal(err)
		}
		if record.Status == status {
			return record
		}
		if time.Now().After(deadline) {
			t.Fatalf("task %s status=%s, want %s", taskID, record.Status, status)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestEnqueueReportsOtherPeerAvailableFromExactReceipt(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)
	publisher := &scriptedTaskWritePublisher{
		insertConfirmation: taskWriteConfirmationForTest(db.PublishedWriteConfirmationOtherPeerAvailable, false),
	}
	manager.taskWrites = publisher

	record, err := EnqueueContext(context.Background(), manager, EnqueueOptions[testPayload]{
		Stream:      "test.available",
		SubjectType: "test-subject",
		SubjectID:   "subject-available",
		Payload:     testPayload{Value: "input"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if publisher.insertCalls != 1 {
		t.Fatalf("task insert calls=%d, want one exact publication", publisher.insertCalls)
	}
	confirmation := record.WriteConfirmation
	if confirmation.Stage != db.PublishedWriteConfirmationOtherPeerAvailable ||
		confirmation.EventID == "" || confirmation.PublishedRootHash == "" ||
		confirmation.RequiredOtherPeers != 1 || confirmation.ConfirmedOtherPeers != 1 ||
		confirmation.CandidateScope != "current_logical_peers" ||
		!slices.Equal(confirmation.EligiblePeerIDs, []string{"peer-b"}) ||
		confirmation.NoCurrentEligiblePeers || confirmation.ReasonCode != "" ||
		confirmation.AvailabilityPending {
		t.Fatalf("enqueue confirmation=%+v, want exact other-peer availability", confirmation)
	}
	latest, found := manager.LatestWriteConfirmation(record.ID)
	if !found || !writeConfirmationsEqual(latest, confirmation) {
		t.Fatalf("latest confirmation=%+v found=%t, want %+v", latest, found, confirmation)
	}
}

func TestEnqueueNoPeerReturnsLocalAcceptanceImmediately(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)
	publisher := &scriptedTaskWritePublisher{
		insertConfirmation: taskWriteConfirmationForTest(db.PublishedWriteConfirmationLocalAccepted, true),
	}
	manager.taskWrites = publisher

	started := time.Now()
	record, err := EnqueueContext(context.Background(), manager, EnqueueOptions[testPayload]{
		Stream:      "test.local-accepted",
		SubjectType: "test-subject",
		SubjectID:   "subject-local-accepted",
		Payload:     testPayload{Value: "input"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("single-peer enqueue took %s, want immediate local acceptance", elapsed)
	}
	if publisher.insertCalls != 1 || record.WriteConfirmation.Stage != db.PublishedWriteConfirmationLocalAccepted ||
		!record.WriteConfirmation.AvailabilityPending || record.WriteConfirmation.ConfirmedOtherPeers != 0 ||
		record.WriteConfirmation.CandidateScope != "current_logical_peers" ||
		!record.WriteConfirmation.NoCurrentEligiblePeers || record.WriteConfirmation.ReasonCode != "no_current_eligible_peers" {
		t.Fatalf("single-peer enqueue calls=%d confirmation=%+v", publisher.insertCalls, record.WriteConfirmation)
	}
}

func TestWriteConfirmationCacheAndWatchDefensivelyCloneEligiblePeers(t *testing.T) {
	manager := NewManager(openTaskTestDB(t))
	watch, cancel, err := manager.Subscribe("task-clone")
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()

	eligible := []string{"peer-b", "peer-c"}
	manager.publishProgress(ProgressUpdate{
		TaskID: "task-clone",
		Status: StatusRunning,
		WriteConfirmation: WriteConfirmation{
			Stage:           db.PublishedWriteConfirmationLocalAccepted,
			CandidateScope:  "current_logical_peers",
			EligiblePeerIDs: eligible,
			ReasonCode:      "insufficient_other_peer_receipts",
		},
	})
	eligible[0] = "mutated-at-ingress"

	observed := <-watch
	observed.WriteConfirmation.EligiblePeerIDs[0] = "mutated-by-watcher"
	latestProgress, found := manager.LatestProgress("task-clone")
	if !found || !slices.Equal(latestProgress.WriteConfirmation.EligiblePeerIDs, []string{"peer-b", "peer-c"}) {
		t.Fatalf("cached progress was aliased: %+v found=%t", latestProgress, found)
	}
	latestProgress.WriteConfirmation.EligiblePeerIDs[0] = "mutated-by-progress-reader"
	latestConfirmation, found := manager.LatestWriteConfirmation("task-clone")
	if !found || !slices.Equal(latestConfirmation.EligiblePeerIDs, []string{"peer-b", "peer-c"}) {
		t.Fatalf("cached confirmation was aliased: %+v found=%t", latestConfirmation, found)
	}
	latestConfirmation.EligiblePeerIDs[0] = "mutated-by-confirmation-reader"
	secondRead, found := manager.LatestWriteConfirmation("task-clone")
	if !found || !slices.Equal(secondRead.EligiblePeerIDs, []string{"peer-b", "peer-c"}) {
		t.Fatalf("confirmation egress was aliased: %+v found=%t", secondRead, found)
	}
}

func TestTaskUpdateAvailabilityTimeoutPreservesAcceptedSuccessWithoutReplay(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)
	timeoutConfirmation := taskWriteConfirmationForTest(db.PublishedWriteConfirmationLocalAccepted, true)
	timeoutConfirmation.AvailabilityError = context.DeadlineExceeded.Error()
	publisher := &scriptedTaskWritePublisher{updateConfirmation: timeoutConfirmation}
	manager.taskWrites = publisher

	record := Record{
		ID:          db.MustNewUUIDv7(),
		Status:      StatusRunning,
		Message:     "running",
		Progress:    35,
		UpdatedAt:   time.Now().UTC(),
		MaxAttempts: 1,
	}
	event := taskEvent(record, nil)
	if err := manager.saveTaskUpdate(record, event); err != nil {
		t.Fatalf("accepted task update became an error after availability timeout: %v", err)
	}
	if publisher.updateCalls != 1 {
		t.Fatalf("task update publications=%d, want exactly one without replay", publisher.updateCalls)
	}
	latest, found := manager.LatestProgress(record.ID)
	if !found || !latest.Durable || latest.WriteConfirmation.Stage != db.PublishedWriteConfirmationLocalAccepted ||
		latest.WriteConfirmation.EventID != timeoutConfirmation.Receipt.EventID ||
		latest.WriteConfirmation.PublishedRootHash != timeoutConfirmation.Receipt.PublishedRootHash ||
		!latest.WriteConfirmation.AvailabilityPending || latest.WriteConfirmation.AvailabilityError != context.DeadlineExceeded.Error() {
		t.Fatalf("timeout progress=%+v found=%t, want accepted local confirmation", latest, found)
	}
}

func TestExactUnresolvedTaskUpdateDefersRunnerUntilPassiveReceiptResolution(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)
	runs := 0
	if err := Register(manager, Stream[testPayload, testResult]{
		Name: "test.update-unresolved",
		Run: func(context.Context, *RunContext[testPayload]) (testResult, error) {
			runs++
			return testResult{Done: "yes"}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	queued, err := EnqueueContext(context.Background(), manager, EnqueueOptions[testPayload]{
		Stream:      "test.update-unresolved",
		SubjectType: "test-subject",
		SubjectID:   "subject-update-unresolved",
		Payload:     testPayload{Value: "input"},
		MaxAttempts: 2,
	})
	if err != nil {
		t.Fatal(err)
	}

	confirmation, unresolvedErr := unresolvedTaskWriteForTest()
	publisher := &scriptedTaskWritePublisher{
		updateConfirmation: confirmation,
		updateErr:          unresolvedErr,
	}
	manager.taskWrites = publisher
	observations := 0
	known := false
	observeErr := errors.New("injected passive receipt observation failure")
	manager.observeWriteReceipt = func(_ context.Context, receipt db.PublishedWriteReceipt) (db.EventReceiptObservation, error) {
		observations++
		if receipt.EventID != confirmation.Receipt.EventID || receipt.PublishedRootHash != confirmation.Receipt.PublishedRootHash {
			t.Fatalf("observed receipt=%+v, want exact unresolved receipt", receipt)
		}
		if observeErr != nil {
			return db.EventReceiptObservation{}, observeErr
		}
		return db.EventReceiptObservation{
			Receipt: receipt,
			Status:  swarmionapp.ReceiptStatus{Known: known},
		}, nil
	}

	if err := manager.RunPending(context.Background()); err == nil || !errors.Is(err, db.ErrPublishedWriteConfirmationUnresolved) {
		t.Fatalf("first runner error=%v, want exact unresolved classification", err)
	}
	if publisher.updateCalls != 1 || runs != 0 {
		t.Fatalf("first runner publications=%d stream_runs=%d, want one/zero", publisher.updateCalls, runs)
	}
	if err := manager.RunPending(context.Background()); err != nil {
		t.Fatalf("receipt observation error should leave task untouched: %v", err)
	}
	if publisher.updateCalls != 1 || runs != 0 || observations != 1 {
		t.Fatalf("second runner publications=%d stream_runs=%d observations=%d, want one/zero/one", publisher.updateCalls, runs, observations)
	}
	latest, found := manager.LatestWriteConfirmation(queued.ID)
	if !found || latest.Stage != "" || latest.EventID != confirmation.Receipt.EventID {
		t.Fatalf("unresolved confirmation=%+v found=%t, want cached exact fence", latest, found)
	}

	observeErr = nil
	if err := manager.RunPending(context.Background()); err != nil {
		t.Fatalf("unknown receipt fence should leave task untouched: %v", err)
	}
	if publisher.updateCalls != 1 || runs != 0 || observations != 2 {
		t.Fatalf("unknown receipt tick publications=%d stream_runs=%d observations=%d, want one/zero/two", publisher.updateCalls, runs, observations)
	}

	known = true
	if err := manager.RunPending(context.Background()); err != nil {
		t.Fatalf("known receipt transition should defer to a fresh next tick: %v", err)
	}
	if publisher.updateCalls != 1 || runs != 0 || observations != 3 {
		t.Fatalf("resolution tick publications=%d stream_runs=%d observations=%d, want one/zero/three", publisher.updateCalls, runs, observations)
	}
	if _, found := manager.LatestWriteConfirmation(queued.ID); found {
		t.Fatal("known exact receipt did not clear the process-local replay fence")
	}
}

func TestStreamAdapterDoesNotRequeueOrFailAfterExactUnresolvedTaskSave(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)
	queued, err := EnqueueContext(context.Background(), manager, EnqueueOptions[testPayload]{
		Stream:      "test.stream-save-unresolved",
		SubjectType: "test-subject",
		SubjectID:   "subject-stream-save-unresolved",
		Payload:     testPayload{Value: "input"},
		MaxAttempts: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	owned := queued
	owned.Status = StatusRunning
	owned.Attempts = 1
	owned.Message = "running"
	owned.StartedAt = time.Now().UTC()
	owned.UpdatedAt = owned.StartedAt

	confirmation, unresolvedErr := unresolvedTaskWriteForTest()
	publisher := &scriptedTaskWritePublisher{
		updateConfirmation: confirmation,
		updateErr:          unresolvedErr,
	}
	manager.taskWrites = publisher
	eventsBefore := taskEventCount(t, store, queued.ID)
	adapter := streamAdapter[testPayload, testResult]{stream: Stream[testPayload, testResult]{
		Name: "test.stream-save-unresolved",
		Run: func(_ context.Context, task *RunContext[testPayload]) (testResult, error) {
			return testResult{}, task.Update(50, "published but unresolved", nil)
		},
	}}

	err = adapter.run(context.Background(), manager, owned)
	if !IsDeferred(err) || !errors.Is(err, db.ErrPublishedWriteConfirmationUnresolved) {
		t.Fatalf("stream error=%v, want deferred exact unresolved", err)
	}
	if publisher.updateCalls != 1 {
		t.Fatalf("stream publications=%d, want exactly one save and no requeue/fail write", publisher.updateCalls)
	}
	if got := taskEventCount(t, store, queued.ID); got != eventsBefore {
		t.Fatalf("stream exact-unresolved path inserted task events: before=%d after=%d", eventsBefore, got)
	}
	stored, getErr := manager.Get(queued.ID)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if stored.Status != StatusPending || stored.Attempts != 0 || stored.WriteConfirmation.Stage != "" ||
		stored.WriteConfirmation.EventID != confirmation.Receipt.EventID {
		t.Fatalf("stream exact-unresolved state=%+v, want unchanged row plus cached exact fence", stored)
	}
}

func TestSaveRecoveredExactUnresolvedPreservesConfirmationAndDefers(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)
	confirmation, unresolvedErr := unresolvedTaskWriteForTest()
	publisher := &scriptedTaskWritePublisher{
		updateConfirmation: confirmation,
		updateErr:          unresolvedErr,
	}
	manager.taskWrites = publisher

	now := time.Now().UTC()
	expected := Record{ID: db.MustNewUUIDv7(), Status: StatusRunning, UpdatedAt: now, Attempts: 1, MaxAttempts: 2}
	recovered := expected
	recovered.Status = StatusPending
	recovered.UpdatedAt = now.Add(time.Second)
	updated, err := manager.saveRecoveredTaskUpdate(expected, recovered, taskEvent(recovered, nil))
	if updated || !IsDeferred(err) || !errors.Is(err, db.ErrPublishedWriteConfirmationUnresolved) {
		t.Fatalf("save recovered updated=%t error=%v, want false/deferred exact unresolved", updated, err)
	}
	if publisher.updateCalls != 1 {
		t.Fatalf("recovery publications=%d, want exactly one", publisher.updateCalls)
	}
	latest, found := manager.LatestWriteConfirmation(recovered.ID)
	if !found || latest.Stage != "" || latest.EventID != confirmation.Receipt.EventID || latest.PublishedRootHash != confirmation.Receipt.PublishedRootHash {
		t.Fatalf("recovery confirmation=%+v found=%t, want cached exact empty-stage confirmation", latest, found)
	}
}

func TestTaskManagerRunsRegisteredStream(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)

	if err := Register(manager, Stream[testPayload, testResult]{
		Name: "test.stream",
		Run: func(ctx context.Context, task *RunContext[testPayload]) (testResult, error) {
			if task.Payload().Value != "input" {
				t.Fatalf("payload = %q, want input", task.Payload().Value)
			}
			if err := task.Update(50, "halfway", map[string]string{"step": "middle"}); err != nil {
				t.Fatal(err)
			}
			return testResult{Done: "yes"}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}

	queued, err := EnqueueContext(context.Background(), manager, EnqueueOptions[testPayload]{
		Stream:      "test.stream",
		SubjectType: "test-subject",
		SubjectID:   "subject-1",
		Title:       "test task",
		Payload:     testPayload{Value: "input"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := manager.RunPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	done, err := manager.Get(queued.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != StatusSucceeded {
		t.Fatalf("status = %q, want succeeded", done.Status)
	}
	if done.Progress != 100 {
		t.Fatalf("progress = %d, want 100", done.Progress)
	}
	if len(done.Result) == 0 {
		t.Fatal("expected result JSON to be stored")
	}

	var eventCount int
	if err := store.ReadRows(context.Background(), "SELECT COUNT(*) FROM task_events WHERE task_id = ?", []any{db.MustUUIDBytes(queued.ID)}, func(rows *sql.Rows) error {
		if !rows.Next() {
			return errors.New("event count query returned no rows")
		}
		return rows.Scan(&eventCount)
	}); err != nil {
		t.Fatal(err)
	}
	if eventCount < 3 {
		t.Fatalf("event count = %d, want at least queued/running/progress/succeeded events", eventCount)
	}
}

func TestTaskRetriesUntilSuccessWithinMaxAttempts(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)

	runs := 0
	if err := Register(manager, Stream[testPayload, testResult]{
		Name: "test.retry",
		Run: func(ctx context.Context, task *RunContext[testPayload]) (testResult, error) {
			runs++
			if runs < 3 {
				return testResult{}, errors.New("transient failure")
			}
			return testResult{Done: "yes"}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}

	queued, err := EnqueueContext(context.Background(), manager, EnqueueOptions[testPayload]{
		Stream:      "test.retry",
		SubjectType: "test-subject",
		SubjectID:   "subject-retry",
		Title:       "retry task",
		Payload:     testPayload{Value: "input"},
		MaxAttempts: 3,
	})
	if err != nil {
		t.Fatal(err)
	}

	// First two runs fail but requeue to pending; the third succeeds.
	for i := 0; i < 2; i++ {
		_ = manager.RunPending(context.Background())
		record, err := manager.Get(queued.ID)
		if err != nil {
			t.Fatal(err)
		}
		if record.Status != StatusPending {
			t.Fatalf("after attempt %d status = %q, want pending (requeued)", i+1, record.Status)
		}
	}
	if err := manager.RunPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	done, err := manager.Get(queued.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != StatusSucceeded {
		t.Fatalf("status = %q, want succeeded after retries", done.Status)
	}
	if done.Attempts != 3 {
		t.Fatalf("attempts = %d, want 3", done.Attempts)
	}
	if runs != 3 {
		t.Fatalf("runs = %d, want 3", runs)
	}
}

func TestTaskReplacePayloadSurvivesRetry(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)

	runs := 0
	if err := Register(manager, Stream[checkpointPayload, testResult]{
		Name: "test.replace-payload",
		Run: func(ctx context.Context, task *RunContext[checkpointPayload]) (testResult, error) {
			runs++
			payload := task.Payload()
			if runs == 1 {
				payload.Receipt = "event-1/root-1"
				if err := task.ReplacePayload(payload); err != nil {
					t.Fatal(err)
				}
				if task.Payload().Receipt != payload.Receipt {
					t.Fatalf("run-context receipt = %q, want %q", task.Payload().Receipt, payload.Receipt)
				}
				return testResult{}, errors.New("transient failure after receipt persistence")
			}
			if payload.Receipt != "event-1/root-1" {
				t.Fatalf("retried payload receipt = %q, want persisted receipt", payload.Receipt)
			}
			return testResult{Done: payload.Receipt}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}

	queued, err := EnqueueContext(context.Background(), manager, EnqueueOptions[checkpointPayload]{
		Stream:      "test.replace-payload",
		SubjectType: "test-subject",
		SubjectID:   "subject-replace-payload",
		Title:       "replace payload task",
		Payload:     checkpointPayload{Value: "input"},
		MaxAttempts: 2,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := manager.RunPending(context.Background()); err == nil {
		t.Fatal("first run should report the transient stream failure")
	}
	retrying, err := manager.Get(queued.ID)
	if err != nil {
		t.Fatal(err)
	}
	if retrying.Status != StatusPending {
		t.Fatalf("retrying status = %q, want pending", retrying.Status)
	}
	var persisted checkpointPayload
	if err := json.Unmarshal(retrying.Payload, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.Receipt != "event-1/root-1" {
		t.Fatalf("persisted receipt = %q, want event-1/root-1", persisted.Receipt)
	}

	if err := manager.RunPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	done, err := manager.Get(queued.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != StatusSucceeded || runs != 2 {
		t.Fatalf("completed task status=%q runs=%d, want succeeded after two runs", done.Status, runs)
	}
}

func TestTaskPayloadSemanticEqualityPreservesLargeIntegerIdentity(t *testing.T) {
	if taskPayloadsSemanticallyEqual(
		json.RawMessage(`{"sequence":9007199254740992}`),
		json.RawMessage(`{"sequence":9007199254740993}`),
	) {
		t.Fatal("distinct integers above the float64 exact range compared equal")
	}
	if !taskPayloadsSemanticallyEqual(
		json.RawMessage(`{"receipt":"event/root","sequence":9007199254740993}`),
		json.RawMessage("{\n  \"sequence\": 9007199254740993, \"receipt\": \"event/root\"\n}"),
	) {
		t.Fatal("semantically equal payloads with reordered fields compared unequal")
	}
	if taskPayloadsSemanticallyEqual(json.RawMessage(`{"value":1} {"value":2}`), json.RawMessage(`{"value":1}`)) {
		t.Fatal("payload with a trailing JSON value compared equal")
	}
}

func TestStreamRunCarriesOwnedRecordAcrossReplacePayloadAndRetry(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)
	queued, err := EnqueueContext(context.Background(), manager, EnqueueOptions[checkpointPayload]{
		Stream:      "test.owned-record",
		SubjectType: "test-subject",
		SubjectID:   "subject-owned-record",
		Title:       "owned record task",
		Payload:     checkpointPayload{Value: "input"},
		MaxAttempts: 2,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Model the exact state just published by markRunning while the live SQL
	// view still exposes the older pending/attempts=0 row. The stream must carry
	// this owned record through its payload replacement and retry instead of
	// re-reading and publishing the stale attempt count.
	owned := queued
	owned.Status = StatusRunning
	owned.Attempts = 1
	owned.Message = "running"
	owned.StartedAt = time.Now().UTC()
	owned.UpdatedAt = owned.StartedAt
	adapter := streamAdapter[checkpointPayload, testResult]{stream: Stream[checkpointPayload, testResult]{
		Name: "test.owned-record",
		Run: func(_ context.Context, task *RunContext[checkpointPayload]) (testResult, error) {
			if task.Task().Attempts != 1 {
				t.Fatalf("stream-owned attempts=%d, want 1", task.Task().Attempts)
			}
			payload := task.Payload()
			payload.Receipt = "event-owned/root-owned"
			if err := task.ReplacePayload(payload); err != nil {
				t.Fatal(err)
			}
			if task.Task().Attempts != 1 {
				t.Fatalf("payload-replaced attempts=%d, want 1", task.Task().Attempts)
			}
			return testResult{}, errors.New("retry after checkpoint")
		},
	}}
	if err := adapter.run(context.Background(), manager, owned); err == nil {
		t.Fatal("stream run should return its retryable error")
	}

	retrying, err := manager.Get(queued.ID)
	if err != nil {
		t.Fatal(err)
	}
	if retrying.Status != StatusPending || retrying.Attempts != 1 {
		t.Fatalf("retrying task status=%s attempts=%d, want pending/1", retrying.Status, retrying.Attempts)
	}
	var persisted checkpointPayload
	if err := json.Unmarshal(retrying.Payload, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.Receipt != "event-owned/root-owned" {
		t.Fatalf("retrying task receipt=%q, want owned checkpoint", persisted.Receipt)
	}
}

func TestStreamCancellationPreservesOwnedRunningReceiptForStartupRecovery(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)
	queued, err := EnqueueContext(context.Background(), manager, EnqueueOptions[checkpointPayload]{
		Stream:      "test.cancel-owned-record",
		SubjectType: "test-subject",
		SubjectID:   "subject-cancel-owned-record",
		Title:       "cancel owned record task",
		Payload:     checkpointPayload{Value: "input"},
		MaxAttempts: 2,
	})
	if err != nil {
		t.Fatal(err)
	}

	owned := queued
	owned.Status = StatusRunning
	owned.Attempts = 1
	owned.Message = "running"
	owned.StartedAt = time.Now().UTC()
	owned.UpdatedAt = owned.StartedAt
	runCtx, cancel := context.WithCancel(context.Background())
	adapter := streamAdapter[checkpointPayload, testResult]{stream: Stream[checkpointPayload, testResult]{
		Name: "test.cancel-owned-record",
		Run: func(_ context.Context, task *RunContext[checkpointPayload]) (testResult, error) {
			payload := task.Payload()
			payload.Receipt = "event-cancel/root-cancel"
			if err := task.ReplacePayload(payload); err != nil {
				t.Fatal(err)
			}
			cancel()
			return testResult{}, context.Canceled
		},
	}}
	if err := adapter.run(runCtx, manager, owned); !errors.Is(err, context.Canceled) {
		t.Fatalf("stream cancellation error=%v, want context canceled", err)
	}

	interrupted, err := manager.Get(queued.ID)
	if err != nil {
		t.Fatal(err)
	}
	if interrupted.Status != StatusRunning || interrupted.Attempts != 1 {
		t.Fatalf("cancelled runner task status=%s attempts=%d, want recoverable running/1", interrupted.Status, interrupted.Attempts)
	}
	var persisted checkpointPayload
	if err := json.Unmarshal(interrupted.Payload, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.Receipt != "event-cancel/root-cancel" {
		t.Fatalf("cancelled runner receipt=%q, want exact persisted receipt", persisted.Receipt)
	}
}

func TestPayloadSaveFailureRequeuesKnownReceiptFromOwnedRecord(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)
	queued, err := EnqueueContext(context.Background(), manager, EnqueueOptions[checkpointPayload]{
		Stream:      "test.receipt-save-failure",
		SubjectType: "test-subject",
		SubjectID:   "subject-receipt-save-failure",
		Title:       "receipt save failure task",
		Payload:     checkpointPayload{Value: "input"},
		MaxAttempts: 2,
	})
	if err != nil {
		t.Fatal(err)
	}

	owned := queued
	owned.Status = StatusRunning
	owned.Attempts = 1
	owned.Message = "running"
	owned.StartedAt = time.Now().UTC()
	owned.UpdatedAt = owned.StartedAt
	injected := errors.New("injected first receipt payload save failure")
	failedReceiptSave := false
	requeuedKnownReceipt := false
	manager.beforeSaveTaskUpdate = func(record Record, _ Event) error {
		var payload checkpointPayload
		if err := json.Unmarshal(record.Payload, &payload); err != nil {
			return err
		}
		if !failedReceiptSave && record.Status == StatusRunning && payload.Receipt == "event-save/root-save" {
			failedReceiptSave = true
			return injected
		}
		if failedReceiptSave && record.Status == StatusPending && payload.Receipt == "event-save/root-save" {
			requeuedKnownReceipt = true
		}
		return nil
	}
	adapter := streamAdapter[checkpointPayload, testResult]{stream: Stream[checkpointPayload, testResult]{
		Name: "test.receipt-save-failure",
		Run: func(_ context.Context, task *RunContext[checkpointPayload]) (testResult, error) {
			payload := task.Payload()
			payload.Receipt = "event-save/root-save"
			return testResult{}, task.ReplacePayload(payload)
		},
	}}
	if err := adapter.run(context.Background(), manager, owned); !errors.Is(err, injected) {
		t.Fatalf("stream error=%v, want injected receipt save failure", err)
	}
	if !failedReceiptSave {
		t.Fatal("receipt payload save failure was not injected")
	}
	if !requeuedKnownReceipt {
		t.Fatal("error path did not requeue the in-memory owned receipt")
	}

	retrying, err := manager.Get(queued.ID)
	if err != nil {
		t.Fatal(err)
	}
	if retrying.Status != StatusPending || retrying.Attempts != 1 {
		t.Fatalf("receipt-save retry status=%s attempts=%d, want pending/1", retrying.Status, retrying.Attempts)
	}
	var persisted checkpointPayload
	if err := json.Unmarshal(retrying.Payload, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.Receipt != "event-save/root-save" {
		t.Fatalf("receipt-save retry payload=%+v, want known exact receipt retained", persisted)
	}
}

func TestRecoverOwnedRunningPreservesPayloadAndAttempt(t *testing.T) {
	store := openTaskTestDB(t)
	ownerA := NewManager(store)
	ownerA.SetExecutorPeerID("peer-a")
	ownerB := NewManager(store)
	ownerB.SetExecutorPeerID("peer-b")

	interrupted, err := EnqueueContext(context.Background(), ownerA, EnqueueOptions[checkpointPayload]{
		Stream:      "test.recover-running",
		SubjectType: "test-subject",
		SubjectID:   "subject-recover-running",
		Title:       "recover running task",
		Payload:     checkpointPayload{Value: "input"},
		MaxAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	otherOwner, err := EnqueueContext(context.Background(), ownerB, EnqueueOptions[checkpointPayload]{
		Stream:      "test.other-owner-running",
		SubjectType: "test-subject",
		SubjectID:   "subject-other-owner-running",
		Title:       "other owner task",
		Payload:     checkpointPayload{Value: "other"},
		MaxAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := ownerA.markRunning(interrupted); err != nil {
		t.Fatal(err)
	}
	if err := ownerB.markRunning(otherOwner); err != nil {
		t.Fatal(err)
	}

	running, err := ownerA.Get(interrupted.ID)
	if err != nil {
		t.Fatal(err)
	}
	runCtx := &RunContext[checkpointPayload]{
		manager: ownerA,
		record:  running,
		payload: checkpointPayload{Value: "input"},
	}
	checkpointed := checkpointPayload{Value: "input", Receipt: "event-restart/root-restart"}
	if err := runCtx.ReplacePayload(checkpointed); err != nil {
		t.Fatal(err)
	}
	if runCtx.Payload().Receipt != checkpointed.Receipt {
		t.Fatalf("run-context receipt = %q, want %q", runCtx.Payload().Receipt, checkpointed.Receipt)
	}

	restarted := NewManager(store)
	restarted.SetExecutorPeerID("peer-a")
	recovered, err := restarted.RecoverOwnedRunning()
	if err != nil {
		t.Fatal(err)
	}
	if recovered != 1 {
		t.Fatalf("recovered tasks = %d, want 1", recovered)
	}

	recoveredRecord, err := restarted.Get(interrupted.ID)
	if err != nil {
		t.Fatal(err)
	}
	if recoveredRecord.Status != StatusPending {
		t.Fatalf("recovered status = %q, want pending", recoveredRecord.Status)
	}
	if recoveredRecord.Attempts != 0 {
		t.Fatalf("recovered attempts = %d, want interrupted attempt rolled back to 0", recoveredRecord.Attempts)
	}
	var recoveredPayload checkpointPayload
	if err := json.Unmarshal(recoveredRecord.Payload, &recoveredPayload); err != nil {
		t.Fatal(err)
	}
	if recoveredPayload != checkpointed {
		t.Fatalf("recovered payload = %+v, want %+v", recoveredPayload, checkpointed)
	}

	otherRecord, err := ownerB.Get(otherOwner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if otherRecord.Status != StatusRunning || otherRecord.Attempts != 1 {
		t.Fatalf("other-owner task status=%q attempts=%d, want running/1", otherRecord.Status, otherRecord.Attempts)
	}

	runs := 0
	if err := Register(restarted, Stream[checkpointPayload, testResult]{
		Name: "test.recover-running",
		Run: func(ctx context.Context, task *RunContext[checkpointPayload]) (testResult, error) {
			runs++
			if task.Payload().Receipt != checkpointed.Receipt {
				t.Fatalf("resumed receipt = %q, want %q", task.Payload().Receipt, checkpointed.Receipt)
			}
			return testResult{Done: task.Payload().Receipt}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := restarted.RunPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	done, err := restarted.Get(interrupted.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != StatusSucceeded || done.Attempts != 1 || runs != 1 {
		t.Fatalf("resumed task status=%q attempts=%d runs=%d, want succeeded/1/1", done.Status, done.Attempts, runs)
	}
}

func TestRecoverOwnedRunningDefersWithoutWriteAndPublishesPreparedPayloadOnlyWhenReady(t *testing.T) {
	store := openTaskTestDB(t)
	original := NewManager(store)
	original.SetExecutorPeerID("peer-recovery-preflight")
	record, err := EnqueueContext(context.Background(), original, EnqueueOptions[checkpointPayload]{
		Stream:      "test.recovery-preflight",
		SubjectType: "test-subject",
		SubjectID:   "subject-recovery-preflight",
		Title:       "recovery preflight task",
		Payload:     checkpointPayload{Value: "input"},
		MaxAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := original.markRunning(record); err != nil {
		t.Fatal(err)
	}

	restarted := NewManager(store)
	restarted.SetExecutorPeerID("peer-recovery-preflight")
	ready := false
	recoveryCalls := 0
	if err := Register(restarted, Stream[checkpointPayload, testResult]{
		Name: "test.recovery-preflight",
		Recover: func(_ context.Context, recovery *RecoveryContext[checkpointPayload]) (StreamRecoveryDisposition, error) {
			recoveryCalls++
			if recovery.Task().Status != StatusRunning || recovery.Payload().Value != "input" {
				t.Fatalf("recovery view=%+v payload=%+v, want interrupted running task", recovery.Task(), recovery.Payload())
			}
			if !ready {
				return StreamRecoveryDeferred, nil
			}
			payload := recovery.Payload()
			payload.Receipt = "event-ready/root-ready"
			recovery.ReplacePayload(payload)
			return StreamRecoveryReady, nil
		},
		Run: func(_ context.Context, task *RunContext[checkpointPayload]) (testResult, error) {
			return testResult{Done: task.Payload().Receipt}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}

	runningBeforeDeferred, err := restarted.Get(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	beforeDeferred := store.TransactionMetrics()
	recovered, err := restarted.RecoverOwnedRunning()
	if err != nil {
		t.Fatal(err)
	}
	if recovered != 0 || recoveryCalls != 1 {
		t.Fatalf("deferred recovery recovered=%d calls=%d, want 0/1", recovered, recoveryCalls)
	}
	if afterDeferred := store.TransactionMetrics(); afterDeferred != beforeDeferred {
		t.Fatalf("deferred recovery wrote a transaction: before=%+v after=%+v", beforeDeferred, afterDeferred)
	}
	deferred, err := restarted.Get(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if deferred.Status != StatusRunning || deferred.Attempts != 1 || string(deferred.Payload) != string(runningBeforeDeferred.Payload) ||
		!deferred.UpdatedAt.Equal(runningBeforeDeferred.UpdatedAt) {
		t.Fatalf("deferred task changed: %+v payload=%s", deferred, deferred.Payload)
	}

	ready = true
	recovered, err = restarted.RecoverOwnedRunning()
	if err != nil {
		t.Fatal(err)
	}
	if recovered != 1 || recoveryCalls != 2 {
		t.Fatalf("ready recovery recovered=%d calls=%d, want 1/2", recovered, recoveryCalls)
	}
	prepared, err := restarted.Get(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	var preparedPayload checkpointPayload
	if err := json.Unmarshal(prepared.Payload, &preparedPayload); err != nil {
		t.Fatal(err)
	}
	if prepared.Status != StatusPending || prepared.Attempts != 0 || preparedPayload.Receipt != "event-ready/root-ready" {
		t.Fatalf("prepared recovery record=%+v payload=%+v, want pending/0 with recovered receipt", prepared, preparedPayload)
	}
}

func TestManagerStartRecoversOwnedRunningBeforeRunningPending(t *testing.T) {
	store := openTaskTestDB(t)
	original := NewManager(store)
	original.SetExecutorPeerID("peer-restart")

	interrupted, err := EnqueueContext(context.Background(), original, EnqueueOptions[checkpointPayload]{
		Stream:      "test.startup-recovery",
		SubjectType: "test-subject",
		SubjectID:   "subject-startup-recovery",
		Title:       "startup recovery task",
		Payload: checkpointPayload{
			Value:   "input",
			Receipt: "event-startup/root-startup",
		},
		MaxAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := original.markRunning(interrupted); err != nil {
		t.Fatal(err)
	}

	restarted := NewManager(store)
	restarted.SetExecutorPeerID("peer-restart")
	runs := make(chan struct{}, 2)
	if err := Register(restarted, Stream[checkpointPayload, testResult]{
		Name: "test.startup-recovery",
		Run: func(ctx context.Context, task *RunContext[checkpointPayload]) (testResult, error) {
			runs <- struct{}{}
			if task.Payload().Receipt != "event-startup/root-startup" {
				t.Fatalf("startup-recovered receipt = %q, want persisted receipt", task.Payload().Receipt)
			}
			return testResult{Done: task.Payload().Receipt}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	stop := restarted.Start(context.Background(), 10*time.Millisecond)
	defer func() {
		if err := stop(); err != nil {
			t.Errorf("stop task manager: %v", err)
		}
	}()

	select {
	case <-runs:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for startup-recovered task to run")
	}

	deadline := time.Now().Add(3 * time.Second)
	for {
		done, err := restarted.Get(interrupted.ID)
		if err != nil {
			t.Fatal(err)
		}
		if done.Status == StatusSucceeded {
			if done.Attempts != 1 {
				t.Fatalf("startup-recovered attempts = %d, want 1", done.Attempts)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("startup-recovered status = %q, want succeeded", done.Status)
		}
		time.Sleep(10 * time.Millisecond)
	}

	select {
	case <-runs:
		t.Fatal("startup recovery ran the interrupted task more than once")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestManagerStartRecoversRunningTaskArrivingAfterEmptyTick(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)
	manager.SetExecutorPeerID("peer-late-recovery")
	barrierRan := make(chan struct{}, 1)
	runs := make(chan struct{}, 2)
	if err := Register(manager, Stream[checkpointPayload, testResult]{
		Name: "test.late-recovery",
		Run: func(_ context.Context, task *RunContext[checkpointPayload]) (testResult, error) {
			if task.Payload().Value == "barrier" {
				barrierRan <- struct{}{}
				return testResult{Done: "barrier"}, nil
			}
			runs <- struct{}{}
			return testResult{Done: task.Payload().Value}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	barrier, err := EnqueueContext(context.Background(), manager, EnqueueOptions[checkpointPayload]{
		Stream:      "test.late-recovery",
		SubjectType: "test-subject",
		SubjectID:   "subject-empty-recovery-barrier",
		Title:       "empty recovery barrier",
		Payload:     checkpointPayload{Value: "barrier"},
		MaxAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	stop := manager.Start(context.Background(), 10*time.Millisecond)
	defer func() {
		if err := stop(); err != nil {
			t.Errorf("stop task manager: %v", err)
		}
	}()

	// Completing this pending barrier proves the runner's first recovery scan
	// observed an empty running set before the late task is introduced.
	select {
	case <-barrierRan:
	case <-time.After(3 * time.Second):
		t.Fatal("initial empty-recovery barrier did not run")
	}
	waitForTaskStatus(t, manager, barrier.ID, StatusSucceeded)
	staging := NewManager(store)
	staging.SetExecutorPeerID("peer-staging")
	record, err := EnqueueContext(context.Background(), staging, EnqueueOptions[checkpointPayload]{
		Stream:      "test.late-recovery",
		SubjectType: "test-subject",
		SubjectID:   "subject-late-recovery",
		Title:       "late recovery task",
		Payload:     checkpointPayload{Value: "late"},
		MaxAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := staging.markRunning(record); err != nil {
		t.Fatal(err)
	}
	running, err := staging.Get(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	running.OwnerPeerID = "peer-late-recovery"
	running.UpdatedAt = time.Now().UTC()
	if err := staging.saveTaskUpdate(running, taskEvent(running, nil)); err != nil {
		t.Fatal(err)
	}

	select {
	case <-runs:
	case <-time.After(3 * time.Second):
		t.Fatal("late replicated running task was not recovered")
	}
	done := waitForTaskStatus(t, manager, record.ID, StatusSucceeded)
	if done.Status != StatusSucceeded || done.Attempts != 1 {
		t.Fatalf("late recovered task status=%s attempts=%d, want succeeded/1", done.Status, done.Attempts)
	}
	select {
	case <-runs:
		t.Fatal("late recovered task ran more than once")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestManagerSameInstanceStopStartRecoversInterruptedRun(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)
	manager.SetExecutorPeerID("peer-same-manager-restart")
	firstStarted := make(chan struct{})
	resumed := make(chan struct{})
	var runMu sync.Mutex
	runs := 0
	if err := Register(manager, Stream[checkpointPayload, testResult]{
		Name: "test.same-manager-restart",
		Run: func(ctx context.Context, task *RunContext[checkpointPayload]) (testResult, error) {
			runMu.Lock()
			runs++
			attempt := runs
			runMu.Unlock()
			if attempt == 1 {
				close(firstStarted)
				<-ctx.Done()
				return testResult{}, ctx.Err()
			}
			close(resumed)
			return testResult{Done: task.Payload().Value}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	record, err := EnqueueContext(context.Background(), manager, EnqueueOptions[checkpointPayload]{
		Stream:      "test.same-manager-restart",
		SubjectType: "test-subject",
		SubjectID:   "subject-same-manager-restart",
		Title:       "same manager restart task",
		Payload:     checkpointPayload{Value: "resume"},
		MaxAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	stopFirst := manager.Start(context.Background(), 10*time.Millisecond)
	select {
	case <-firstStarted:
	case <-time.After(3 * time.Second):
		t.Fatal("first task run did not start")
	}
	if err := stopFirst(); err != nil {
		t.Fatal(err)
	}
	interrupted, err := manager.Get(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if interrupted.Status != StatusRunning || interrupted.Attempts != 1 {
		t.Fatalf("interrupted task status=%s attempts=%d, want running/1", interrupted.Status, interrupted.Attempts)
	}

	stopSecond := manager.Start(context.Background(), 10*time.Millisecond)
	defer func() {
		if err := stopSecond(); err != nil {
			t.Errorf("stop restarted manager: %v", err)
		}
	}()
	select {
	case <-resumed:
	case <-time.After(3 * time.Second):
		t.Fatal("same Manager did not recover its interrupted task after restart")
	}
	done := waitForTaskStatus(t, manager, record.ID, StatusSucceeded)
	if done.Status != StatusSucceeded || done.Attempts != 1 {
		t.Fatalf("resumed task status=%s attempts=%d, want succeeded/1", done.Status, done.Attempts)
	}
}

func TestRecoveryHookErrorDoesNotStarveUnrelatedPendingTask(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)
	manager.SetExecutorPeerID("peer-recovery-error-isolation")
	bad, err := EnqueueContext(context.Background(), manager, EnqueueOptions[testPayload]{
		Stream:      "test.bad-recovery",
		SubjectType: "test-subject",
		SubjectID:   "subject-bad-recovery",
		Title:       "bad recovery task",
		Payload:     testPayload{Value: "bad"},
		MaxAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.markRunning(bad); err != nil {
		t.Fatal(err)
	}
	if err := Register(manager, Stream[testPayload, testResult]{
		Name: "test.bad-recovery",
		Recover: func(context.Context, *RecoveryContext[testPayload]) (StreamRecoveryDisposition, error) {
			return StreamRecoveryReady, errors.New("malformed recovery payload")
		},
		Run: func(context.Context, *RunContext[testPayload]) (testResult, error) {
			t.Fatal("errored recovery task must remain running")
			return testResult{}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	goodRan := make(chan struct{}, 1)
	if err := Register(manager, Stream[testPayload, testResult]{
		Name: "test.good-pending",
		Run: func(_ context.Context, task *RunContext[testPayload]) (testResult, error) {
			goodRan <- struct{}{}
			return testResult{Done: task.Payload().Value}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	good, err := EnqueueContext(context.Background(), manager, EnqueueOptions[testPayload]{
		Stream:      "test.good-pending",
		SubjectType: "test-subject",
		SubjectID:   "subject-good-pending",
		Title:       "good pending task",
		Payload:     testPayload{Value: "good"},
		MaxAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	stop := manager.Start(context.Background(), 10*time.Millisecond)
	defer func() { _ = stop() }()
	select {
	case <-goodRan:
	case <-time.After(3 * time.Second):
		t.Fatal("recovery hook error starved unrelated pending task")
	}
	done := waitForTaskStatus(t, manager, good.ID, StatusSucceeded)
	if done.Status != StatusSucceeded {
		t.Fatalf("unrelated task status=%s, want succeeded", done.Status)
	}
	stillBad, err := manager.Get(bad.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stillBad.Status != StatusRunning {
		t.Fatalf("errored recovery task status=%s, want unchanged running", stillBad.Status)
	}
}

func TestRecoveryCASDoesNotReviveConcurrentCancellation(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)
	manager.SetExecutorPeerID("peer-recovery-cas")
	record, err := EnqueueContext(context.Background(), manager, EnqueueOptions[testPayload]{
		Stream:      "test.recovery-cas",
		SubjectType: "test-subject",
		SubjectID:   "subject-recovery-cas",
		Title:       "recovery CAS task",
		Payload:     testPayload{Value: "input"},
		MaxAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.markRunning(record); err != nil {
		t.Fatal(err)
	}
	recoveryEntered := make(chan struct{})
	releaseRecovery := make(chan struct{})
	runs := 0
	if err := Register(manager, Stream[testPayload, testResult]{
		Name: "test.recovery-cas",
		Recover: func(context.Context, *RecoveryContext[testPayload]) (StreamRecoveryDisposition, error) {
			close(recoveryEntered)
			<-releaseRecovery
			return StreamRecoveryReady, nil
		},
		Run: func(context.Context, *RunContext[testPayload]) (testResult, error) {
			runs++
			return testResult{}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	type recoveryResult struct {
		count int
		err   error
	}
	recoveredResult := make(chan recoveryResult, 1)
	go func() {
		count, recoverErr := manager.RecoverOwnedRunning()
		recoveredResult <- recoveryResult{count: count, err: recoverErr}
	}()
	select {
	case <-recoveryEntered:
	case <-time.After(3 * time.Second):
		t.Fatal("recovery hook did not start")
	}
	if err := manager.Cancel(record.ID, "cancelled during recovery"); err != nil {
		t.Fatal(err)
	}
	eventsAfterCancel := taskEventCount(t, store, record.ID)
	close(releaseRecovery)
	result := <-recoveredResult
	if result.err != nil || result.count != 0 {
		t.Fatalf("stale recovery result count=%d error=%v, want 0/nil", result.count, result.err)
	}
	if err := manager.RunPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	cancelled, err := manager.Get(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.Status != StatusCancelled || runs != 0 {
		t.Fatalf("cancelled task status=%s runs=%d, want cancelled/0", cancelled.Status, runs)
	}
	if got := taskEventCount(t, store, record.ID); got != eventsAfterCancel {
		t.Fatalf("stale recovery inserted an event: after_cancel=%d final=%d", eventsAfterCancel, got)
	}
}

func TestPermanentTaskErrorBypassesConfiguredRetries(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)
	sentinel := errors.New("permanent application conflict")
	runs := 0
	if err := Register(manager, Stream[testPayload, testResult]{
		Name: "test.permanent",
		Run: func(context.Context, *RunContext[testPayload]) (testResult, error) {
			runs++
			return testResult{}, MarkPermanent(sentinel)
		},
	}); err != nil {
		t.Fatal(err)
	}
	record, err := EnqueueContext(context.Background(), manager, EnqueueOptions[testPayload]{
		Stream:      "test.permanent",
		SubjectType: "test-subject",
		SubjectID:   "subject-permanent",
		Title:       "permanent task",
		Payload:     testPayload{Value: "input"},
		MaxAttempts: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		_ = manager.RunPending(context.Background())
	}
	done, err := manager.Get(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != StatusFailed || done.Attempts != 1 || runs != 1 {
		t.Fatalf("permanent task status=%s attempts=%d runs=%d, want failed/1/1", done.Status, done.Attempts, runs)
	}
	wrapped := MarkPermanent(sentinel)
	if again := MarkPermanent(wrapped); !IsPermanent(wrapped) || !errors.Is(wrapped, sentinel) || !IsPermanent(again) || !errors.Is(again, sentinel) {
		t.Fatalf("permanent classification did not preserve cause/idempotence: %v", wrapped)
	}
}

func TestDeferredTaskErrorLeavesOwnedTaskRunningWithoutBookkeeping(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)
	sentinel := errors.New("exact receipt still pending")
	var metricsAtBoundary db.TransactionMetricsSnapshot
	eventsAtBoundary := 0
	runs := 0
	if err := Register(manager, Stream[testPayload, testResult]{
		Name: "test.deferred",
		Run: func(_ context.Context, task *RunContext[testPayload]) (testResult, error) {
			runs++
			metricsAtBoundary = store.TransactionMetrics()
			eventsAtBoundary = taskEventCount(t, store, task.Task().ID)
			return testResult{}, MarkDeferred(sentinel)
		},
	}); err != nil {
		t.Fatal(err)
	}
	record, err := EnqueueContext(context.Background(), manager, EnqueueOptions[testPayload]{
		Stream:      "test.deferred",
		SubjectType: "test-subject",
		SubjectID:   "subject-deferred",
		Title:       "deferred task",
		Payload:     testPayload{Value: "input"},
		MaxAttempts: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = manager.RunPending(context.Background())
	if after := store.TransactionMetrics(); after != metricsAtBoundary {
		t.Fatalf("deferred error path wrote task SQL: boundary=%+v after=%+v", metricsAtBoundary, after)
	}
	if got := taskEventCount(t, store, record.ID); got != eventsAtBoundary {
		t.Fatalf("deferred error path inserted task event: boundary=%d after=%d", eventsAtBoundary, got)
	}
	owned, err := manager.Get(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if owned.Status != StatusRunning || owned.Attempts != 1 || runs != 1 {
		t.Fatalf("deferred task status=%s attempts=%d runs=%d, want running/1/1", owned.Status, owned.Attempts, runs)
	}
	_ = manager.RunPending(context.Background())
	if runs != 1 {
		t.Fatalf("running deferred task was picked up as pending: runs=%d", runs)
	}
	wrapped := MarkDeferred(sentinel)
	if again := MarkDeferred(wrapped); !IsDeferred(wrapped) || !errors.Is(wrapped, sentinel) || !IsDeferred(again) || !errors.Is(again, sentinel) {
		t.Fatalf("deferred classification did not preserve cause/idempotence: %v", wrapped)
	}
}

func TestTaskFailsTerminallyWhenMaxAttemptsIsOne(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)

	runs := 0
	if err := Register(manager, Stream[testPayload, testResult]{
		Name: "test.noretry",
		Run: func(ctx context.Context, task *RunContext[testPayload]) (testResult, error) {
			runs++
			return testResult{}, errors.New("boom")
		},
	}); err != nil {
		t.Fatal(err)
	}

	queued, err := EnqueueContext(context.Background(), manager, EnqueueOptions[testPayload]{
		Stream:      "test.noretry",
		SubjectType: "test-subject",
		SubjectID:   "subject-noretry",
		Title:       "no retry task",
		Payload:     testPayload{Value: "input"},
		MaxAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	_ = manager.RunPending(context.Background())
	// A second RunPending must not pick the failed task back up.
	_ = manager.RunPending(context.Background())

	done, err := manager.Get(queued.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != StatusFailed {
		t.Fatalf("status = %q, want failed", done.Status)
	}
	if runs != 1 {
		t.Fatalf("runs = %d, want 1 (no retry when MaxAttempts=1)", runs)
	}
}

func TestTaskProgressSubscriptionDoesNotPersistEvent(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)

	queued, err := EnqueueContext(context.Background(), manager, EnqueueOptions[testPayload]{
		Stream:      "test.stream",
		SubjectType: "test-subject",
		SubjectID:   "subject-1",
		Title:       "test task",
		Payload:     testPayload{Value: "input"},
	})
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore := taskEventCount(t, store, queued.ID)

	updates, cancel, err := manager.Subscribe(queued.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()

	if err := manager.Progress(queued.ID, StatusRunning, 42, "uploading", map[string]string{"phase": "copy"}); err != nil {
		t.Fatal(err)
	}

	select {
	case update := <-updates:
		if update.TaskID != queued.ID {
			t.Fatalf("update task id = %q, want %q", update.TaskID, queued.ID)
		}
		if update.Durable {
			t.Fatal("live progress update should not be durable")
		}
		if update.Progress != 42 {
			t.Fatalf("update progress = %d, want 42", update.Progress)
		}
		var details map[string]string
		if err := json.Unmarshal(update.Details, &details); err != nil {
			t.Fatalf("decode details: %v", err)
		}
		if details["phase"] != "copy" {
			t.Fatalf("details phase = %q, want copy", details["phase"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for live progress update")
	}

	eventsAfter := taskEventCount(t, store, queued.ID)
	if eventsAfter != eventsBefore {
		t.Fatalf("task events changed from %d to %d after live progress", eventsBefore, eventsAfter)
	}
}

func TestTaskRunnerOnlyClaimsOwnedTasks(t *testing.T) {
	store := openTaskTestDB(t)
	ownerA := NewManager(store)
	ownerA.SetExecutorPeerID("peer-a")
	ownerB := NewManager(store)
	ownerB.SetExecutorPeerID("peer-b")

	runCount := 0
	for _, manager := range []*Manager{ownerA, ownerB} {
		if err := Register(manager, Stream[testPayload, testResult]{
			Name: "test.stream",
			Run: func(ctx context.Context, task *RunContext[testPayload]) (testResult, error) {
				runCount++
				return testResult{Done: task.Payload().Value}, nil
			},
		}); err != nil {
			t.Fatal(err)
		}
	}

	queued, err := EnqueueContext(context.Background(), ownerA, EnqueueOptions[testPayload]{
		Stream:      "test.stream",
		SubjectType: "test-subject",
		SubjectID:   "subject-1",
		Title:       "test task",
		Payload:     testPayload{Value: "peer-a-work"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := ownerB.RunPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	stillPending, err := ownerA.Get(queued.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stillPending.Status != StatusPending {
		t.Fatalf("status after wrong owner = %q, want pending", stillPending.Status)
	}
	if runCount != 0 {
		t.Fatalf("run count after wrong owner = %d, want 0", runCount)
	}

	if err := ownerA.RunPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	done, err := ownerA.Get(queued.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != StatusSucceeded {
		t.Fatalf("status after owner run = %q, want succeeded", done.Status)
	}
	if runCount != 1 {
		t.Fatalf("run count after owner run = %d, want 1", runCount)
	}
}

func TestRegisterIfAbsentKeepsExistingStream(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)

	runCount := 0
	if err := Register(manager, Stream[testPayload, testResult]{
		Name: "test.stream",
		Run: func(ctx context.Context, task *RunContext[testPayload]) (testResult, error) {
			runCount++
			return testResult{Done: "first"}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := RegisterIfAbsent(manager, Stream[testPayload, testResult]{
		Name: "test.stream",
		Run: func(ctx context.Context, task *RunContext[testPayload]) (testResult, error) {
			return testResult{Done: "second"}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}

	queued, err := EnqueueContext(context.Background(), manager, EnqueueOptions[testPayload]{
		Stream:      "test.stream",
		SubjectType: "test-subject",
		SubjectID:   "subject-1",
		Title:       "test task",
		Payload:     testPayload{Value: "input"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.RunPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	done, err := manager.Get(queued.ID)
	if err != nil {
		t.Fatal(err)
	}
	var result testResult
	if err := json.Unmarshal(done.Result, &result); err != nil {
		t.Fatal(err)
	}
	if runCount != 1 || result.Done != "first" {
		t.Fatalf("RegisterIfAbsent replaced existing stream: runCount=%d result=%q", runCount, result.Done)
	}
}

func taskEventCount(t *testing.T, store *db.DB, taskID string) int {
	t.Helper()
	var count int
	if err := store.ReadRows(context.Background(), "SELECT COUNT(*) FROM task_events WHERE task_id = ?", []any{db.MustUUIDBytes(taskID)}, func(rows *sql.Rows) error {
		if !rows.Next() {
			return errors.New("event count query returned no rows")
		}
		return rows.Scan(&count)
	}); err != nil {
		t.Fatal(err)
	}
	return count
}

func openTaskTestDB(t *testing.T) *db.DB {
	t.Helper()
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
	store, err := db.Open(workDir, "protos_tasks_test", key, testswarmion.NewBorrowedLink(t, key))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	if err := store.Init(); err != nil {
		t.Fatal(err)
	}
	return store
}
