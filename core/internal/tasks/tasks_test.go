package tasks

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	swarmionprotocol "github.com/nustiueudinastea/swarmion/protocol"
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

func TestTaskCheckpointOperationCanonicalizesReplicatedPayload(t *testing.T) {
	baseKey, err := db.NewPublishedWriteOperationKey()
	if err != nil {
		t.Fatal(err)
	}
	left := Record{
		ID:          db.MustNewUUIDv7(),
		OwnerPeerID: "peer-a",
		Payload:     json.RawMessage(`{"receipt":{"root":"root-1","event":"event-1"},"sequence":9007199254740993}`),
	}
	right := left
	right.UpdatedAt = time.Now().UTC().Add(time.Hour)
	right.Payload = json.RawMessage("{\n  \"sequence\": 9007199254740993, \"receipt\": {\"event\": \"event-1\", \"root\": \"root-1\"}\n}")
	leftEvent := Event{
		ID:        db.MustNewUUIDv7(),
		TaskID:    left.ID,
		Status:    StatusRunning,
		Message:   "receipt persisted",
		Progress:  55,
		Details:   json.RawMessage(`{"sequence":9007199254740993,"step":"receipt"}`),
		CreatedAt: time.Now().UTC(),
	}
	rightEvent := leftEvent
	rightEvent.ID = db.MustNewUUIDv7()
	rightEvent.Details = json.RawMessage(`{"step":"receipt","sequence":9007199254740993}`)
	rightEvent.CreatedAt = leftEvent.CreatedAt.Add(time.Hour)
	leftOperation, err := taskCheckpointPublishedWriteOperation(left, leftEvent, baseKey)
	if err != nil {
		t.Fatal(err)
	}
	rightOperation, err := taskCheckpointPublishedWriteOperation(right, rightEvent, baseKey)
	if err != nil {
		t.Fatal(err)
	}
	if leftOperation != rightOperation {
		t.Fatalf("semantically equal payload operation mismatch: left=%+v right=%+v", leftOperation, rightOperation)
	}

	changed := left
	changed.Payload = json.RawMessage(`{"receipt":{"root":"root-2","event":"event-1"},"sequence":9007199254740993}`)
	changedOperation, err := taskCheckpointPublishedWriteOperation(changed, leftEvent, baseKey)
	if err != nil {
		t.Fatal(err)
	}
	if changedOperation.Key == leftOperation.Key || changedOperation.IntentDigest == leftOperation.IntentDigest {
		t.Fatalf("changed payload reused checkpoint operation: original=%+v changed=%+v", leftOperation, changedOperation)
	}

	variants := []struct {
		name   string
		record Record
		event  Event
	}{
		{name: "progress", record: func() Record { value := left; value.Progress++; return value }(), event: leftEvent},
		{name: "message", record: func() Record { value := left; value.Message = "next checkpoint"; return value }(), event: leftEvent},
		{name: "status", record: func() Record { value := left; value.Status = StatusPending; return value }(), event: leftEvent},
		{name: "attempt", record: func() Record { value := left; value.Attempts++; return value }(), event: leftEvent},
		{name: "event details", record: left, event: func() Event { value := leftEvent; value.Details = json.RawMessage(`{"step":"next"}`); return value }()},
	}
	for _, variant := range variants {
		t.Run(variant.name, func(t *testing.T) {
			operation, err := taskCheckpointPublishedWriteOperation(variant.record, variant.event, baseKey)
			if err != nil {
				t.Fatal(err)
			}
			if operation.Key == leftOperation.Key || operation.IntentDigest == leftOperation.IntentDigest {
				t.Fatalf("changed logical checkpoint reused operation: original=%+v changed=%+v", leftOperation, operation)
			}
		})
	}
}

type fakeTaskCheckpointTracker struct {
	observation       db.EventReceiptObservation
	waitErr           error
	record            Record
	found             bool
	invariantErr      error
	waitCalls         int
	invariantCalls    int
	waitReceipt       db.PublishedWriteReceipt
	invariantCommitID string
	invariantTaskID   string
}

type blockingTaskCheckpointTracker struct {
	delegate taskCheckpointTracker
	entered  chan struct{}
	release  chan struct{}
	once     sync.Once
}

func (tracker *blockingTaskCheckpointTracker) WaitForPublishedWriteApplied(
	ctx context.Context,
	receipt db.PublishedWriteReceipt,
	reason string,
) (db.EventReceiptObservation, error) {
	tracker.once.Do(func() { close(tracker.entered) })
	select {
	case <-tracker.release:
	case <-ctx.Done():
		return db.EventReceiptObservation{}, ctx.Err()
	}
	return tracker.delegate.WaitForPublishedWriteApplied(ctx, receipt, reason)
}

func (tracker *blockingTaskCheckpointTracker) RecordAtCheckpoint(
	ctx context.Context,
	checkpointCommitID string,
	taskID string,
) (Record, bool, error) {
	return tracker.delegate.RecordAtCheckpoint(ctx, checkpointCommitID, taskID)
}

func (tracker *fakeTaskCheckpointTracker) WaitForPublishedWriteApplied(
	_ context.Context,
	receipt db.PublishedWriteReceipt,
	_ string,
) (db.EventReceiptObservation, error) {
	tracker.waitCalls++
	tracker.waitReceipt = receipt
	observation := tracker.observation
	observation.Receipt = receipt
	return observation, tracker.waitErr
}

func (tracker *fakeTaskCheckpointTracker) RecordAtCheckpoint(
	_ context.Context,
	checkpointCommitID string,
	taskID string,
) (Record, bool, error) {
	tracker.invariantCalls++
	tracker.invariantCommitID = checkpointCommitID
	tracker.invariantTaskID = taskID
	record := tracker.record
	record.Payload = append(json.RawMessage(nil), record.Payload...)
	record.Result = append(json.RawMessage(nil), record.Result...)
	return record, tracker.found, tracker.invariantErr
}

func newCheckpointPublicationTestContext(
	t *testing.T,
	suffix string,
) (*Manager, *RunContext[checkpointPayload], *fakeTaskCheckpointTracker) {
	t.Helper()
	store := openTaskTestDB(t)
	manager := NewManager(store)
	queued, err := Enqueue(manager, EnqueueOptions[checkpointPayload]{
		Stream:      "test.checkpoint-publication-" + suffix,
		SubjectType: "test-subject",
		SubjectID:   "subject-checkpoint-publication-" + suffix,
		Title:       "checkpoint publication task",
		Payload:     checkpointPayload{Value: "input"},
		MaxAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	tracker := &fakeTaskCheckpointTracker{
		observation: db.EventReceiptObservation{
			State: db.EventReceiptStateAppliedDurably,
			Status: swarmionapp.BranchEventReceiptStatus{
				AppliedDurably:            true,
				ContentCoverage:           swarmionapp.BranchEventContentCoverageCovered,
				CheckpointCommitID:        "event-task-checkpoint",
				DurableCheckpointCommitID: "durable-task-head",
			},
		},
		found: true,
	}
	manager.checkpointTracker = tracker
	running := queued
	running.Status = StatusRunning
	running.Message = "running"
	running.Attempts = 1
	running.StartedAt = time.Now().UTC()
	running.UpdatedAt = running.StartedAt
	return manager, &RunContext[checkpointPayload]{
		manager: manager,
		record:  running,
		payload: checkpointPayload{Value: "input"},
	}, tracker
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

	queued, err := Enqueue(manager, EnqueueOptions[testPayload]{
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

	queued, err := Enqueue(manager, EnqueueOptions[testPayload]{
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

func TestTaskPayloadCheckpointSurvivesRetry(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)

	runs := 0
	if err := Register(manager, Stream[checkpointPayload, testResult]{
		Name: "test.payload-checkpoint",
		Run: func(ctx context.Context, task *RunContext[checkpointPayload]) (testResult, error) {
			runs++
			payload := task.Payload()
			if runs == 1 {
				payload.Receipt = "event-1/root-1"
				if err := task.CheckpointPayload(payload, 55, "receipt persisted", map[string]string{"event_id": "event-1"}); err != nil {
					t.Fatal(err)
				}
				if task.Payload().Receipt != payload.Receipt {
					t.Fatalf("run-context receipt = %q, want %q", task.Payload().Receipt, payload.Receipt)
				}
				if task.Task().Progress != 55 || task.Task().Message != "receipt persisted" {
					t.Fatalf("run-context task = %+v, want checkpointed progress/message", task.Task())
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

	queued, err := Enqueue(manager, EnqueueOptions[checkpointPayload]{
		Stream:      "test.payload-checkpoint",
		SubjectType: "test-subject",
		SubjectID:   "subject-payload-checkpoint",
		Title:       "payload checkpoint task",
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
	if retrying.Progress != 55 {
		t.Fatalf("retrying progress = %d, want checkpointed progress 55", retrying.Progress)
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

func TestCheckpointPayloadSamePayloadPersistsEachDistinctLogicalState(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)
	queued, err := Enqueue(manager, EnqueueOptions[checkpointPayload]{
		Stream:      "test.distinct-checkpoint-state",
		SubjectType: "test-subject",
		SubjectID:   "subject-distinct-checkpoint-state",
		Title:       "distinct checkpoint state task",
		Payload:     checkpointPayload{Value: "input", Receipt: "event/root"},
		MaxAttempts: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	operationKey, err := db.NewPublishedWriteOperationKey()
	if err != nil {
		t.Fatal(err)
	}
	running := queued
	running.Status = StatusRunning
	running.Message = "running"
	running.Attempts = 1
	running.StartedAt = time.Now().UTC()
	running.UpdatedAt = running.StartedAt
	payload := checkpointPayload{Value: "input", Receipt: "event/root"}
	runCtx := &RunContext[checkpointPayload]{manager: manager, record: running, payload: payload}

	if err := runCtx.CheckpointPayloadWithOperationKey(
		operationKey,
		payload,
		25,
		"first receipt checkpoint",
		map[string]string{"phase": "first"},
	); err != nil {
		t.Fatalf("persist first logical checkpoint: %v", err)
	}
	if err := runCtx.CheckpointPayloadWithOperationKey(
		operationKey,
		payload,
		75,
		"second receipt checkpoint",
		map[string]string{"phase": "second"},
	); err != nil {
		t.Fatalf("persist second logical checkpoint: %v", err)
	}

	persisted, err := manager.Get(queued.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Status != StatusRunning || persisted.Progress != 75 || persisted.Message != "second receipt checkpoint" {
		t.Fatalf(
			"persisted task state=%s/%d/%q, want running/75/second receipt checkpoint",
			persisted.Status,
			persisted.Progress,
			persisted.Message,
		)
	}
	latest, ok := manager.LatestProgress(queued.ID)
	if !ok || latest.Progress != 75 || latest.Message != "second receipt checkpoint" {
		t.Fatalf("latest progress=%+v found=%t, want the second persisted checkpoint", latest, ok)
	}
	events, err := manager.Events(queued.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 3 {
		t.Fatalf("task events=%d, want queued plus two distinct checkpoints", len(events))
	}
	last := events[len(events)-1]
	if last.Progress != 75 || last.Message != "second receipt checkpoint" || !taskPayloadsSemanticallyEqual(last.Details, json.RawMessage(`{"phase":"second"}`)) {
		t.Fatalf("last checkpoint event=%+v, want the second logical event", last)
	}
}

func TestCheckpointPayloadRequiresDurableCheckpointPayloadInvariant(t *testing.T) {
	tests := []struct {
		name             string
		coverage         swarmionapp.BranchEventContentCoverage
		checkpointJSON   json.RawMessage
		mutateCheckpoint func(*Record)
		found            bool
		wantConflict     bool
		wantProgress     bool
	}{
		{
			name:           "content covered matching payload",
			coverage:       swarmionapp.BranchEventContentCoverageCovered,
			checkpointJSON: json.RawMessage(`{"receipt":"event/root","value":"input"}`),
			found:          true,
			wantProgress:   true,
		},
		{
			name:           "content dissent matching payload",
			coverage:       swarmionapp.BranchEventContentCoverageDissent,
			checkpointJSON: json.RawMessage(`{"receipt":"event/root","value":"input"}`),
			found:          true,
			wantProgress:   true,
		},
		{
			name:           "content dissent competing payload",
			coverage:       swarmionapp.BranchEventContentCoverageDissent,
			checkpointJSON: json.RawMessage(`{"receipt":"competing/root","value":"input"}`),
			found:          true,
			wantConflict:   true,
		},
		{
			name:           "content dissent competing logical record",
			coverage:       swarmionapp.BranchEventContentCoverageDissent,
			checkpointJSON: json.RawMessage(`{"receipt":"event/root","value":"input"}`),
			mutateCheckpoint: func(record *Record) {
				record.Progress = 54
				record.Message = "competing checkpoint"
			},
			found:        true,
			wantConflict: true,
		},
		{
			name:         "content dissent absent task",
			coverage:     swarmionapp.BranchEventContentCoverageDissent,
			found:        false,
			wantConflict: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := openTaskTestDB(t)
			manager := NewManager(store)
			queued, err := Enqueue(manager, EnqueueOptions[checkpointPayload]{
				Stream:      "test.checkpoint-invariant",
				SubjectType: "test-subject",
				SubjectID:   "subject-checkpoint-invariant",
				Title:       "checkpoint invariant task",
				Payload:     checkpointPayload{Value: "input"},
				MaxAttempts: 2,
			})
			if err != nil {
				t.Fatal(err)
			}

			tracker := &fakeTaskCheckpointTracker{
				observation: db.EventReceiptObservation{
					State: db.EventReceiptStateAppliedDurably,
					Status: swarmionapp.BranchEventReceiptStatus{
						AppliedDurably:            true,
						ContentCoverage:           tt.coverage,
						CheckpointCommitID:        "event-task-checkpoint",
						DurableCheckpointCommitID: "later-durable-task-head",
					},
				},
				found: tt.found,
			}
			manager.checkpointTracker = tracker
			publicationCalls := 0
			manager.beforeSaveTaskUpdate = func(_ Record, _ Event) error {
				publicationCalls++
				return nil
			}

			running := queued
			running.Status = StatusRunning
			running.Message = "running"
			running.Attempts = 1
			running.StartedAt = time.Now().UTC()
			running.UpdatedAt = running.StartedAt
			tracker.record = running
			tracker.record.Status = StatusRunning
			tracker.record.Progress = 55
			tracker.record.Message = "receipt persisted"
			tracker.record.Payload = append(json.RawMessage(nil), tt.checkpointJSON...)
			if tt.mutateCheckpoint != nil {
				tt.mutateCheckpoint(&tracker.record)
			}
			runCtx := &RunContext[checkpointPayload]{
				manager: manager,
				record:  running,
				payload: checkpointPayload{Value: "input"},
			}
			checkpointed := checkpointPayload{Value: "input", Receipt: "event/root"}
			err = runCtx.CheckpointPayload(checkpointed, 55, "receipt persisted", nil)
			if tt.wantConflict {
				if !errors.Is(err, ErrCheckpointInvariantConflict) {
					t.Fatalf("checkpoint error=%v, want task checkpoint invariant conflict", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}

			if publicationCalls != 1 {
				t.Fatalf("task checkpoint publications=%d, want exactly one", publicationCalls)
			}
			if tracker.waitCalls != 1 || tracker.invariantCalls != 1 {
				t.Fatalf("checkpoint wait calls=%d invariant calls=%d, want 1/1", tracker.waitCalls, tracker.invariantCalls)
			}
			if !tracker.waitReceipt.Committed || tracker.waitReceipt.EventID == "" || tracker.waitReceipt.PublishedRootHash == "" {
				t.Fatalf("checkpoint wait lost exact publication receipt: %+v", tracker.waitReceipt)
			}
			if tracker.invariantCommitID != "event-task-checkpoint" || tracker.invariantTaskID != queued.ID {
				t.Fatalf("invariant query checkpoint=%q task=%q, want exact event checkpoint and task %q", tracker.invariantCommitID, tracker.invariantTaskID, queued.ID)
			}
			progress, ok := manager.LatestProgress(queued.ID)
			if tt.wantProgress {
				if !ok || progress.Progress != 55 || progress.Message != "receipt persisted" {
					t.Fatalf("successful checkpoint progress=%+v found=%t, want persisted progress", progress, ok)
				}
			} else if ok && progress.Progress == 55 {
				t.Fatalf("conflicting checkpoint was reported as persisted: %+v", progress)
			}
		})
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

func TestCheckpointPayloadRetriesTypedNotAcceptedOutcome(t *testing.T) {
	manager, runCtx, tracker := newCheckpointPublicationTestContext(t, "retry-not-accepted")
	manager.waitTaskCheckpointPublicationRetry = func(context.Context, int) error { return nil }
	receipt := db.PublishedWriteReceipt{
		Committed:         true,
		EventID:           swarmionprotocol.NewEventID("task checkpoint retry event").String(),
		PublishedRootHash: swarmionprotocol.NewRootHash("task checkpoint retry root").String(),
	}
	notAcceptedErr := &swarmionapp.CommitNotAcceptedError{Cause: errors.New("event admission did not persist")}
	var (
		publishCalls int
		firstRecord  Record
		firstEvent   Event
	)
	manager.publishTaskCheckpoint = func(ctx context.Context, record Record, event Event) (db.PublishedWriteReceipt, error) {
		publishCalls++
		if _, hasDeadline := ctx.Deadline(); !hasDeadline || ctx.Err() != nil {
			t.Fatalf("checkpoint publication retry did not use a live bounded context")
		}
		if publishCalls == 1 {
			firstRecord = record
			firstRecord.Payload = append(json.RawMessage(nil), record.Payload...)
			firstEvent = event
			firstEvent.Details = append(json.RawMessage(nil), event.Details...)
			return db.PublishedWriteReceipt{}, notAcceptedErr
		}
		if !reflect.DeepEqual(record, firstRecord) || !reflect.DeepEqual(event, firstEvent) {
			t.Fatalf("checkpoint retry changed logical write: first=%+v/%+v retry=%+v/%+v", firstRecord, firstEvent, record, event)
		}
		tracker.record = record
		return receipt, nil
	}

	err := runCtx.CheckpointPayload(
		checkpointPayload{Value: "input", Receipt: "event/root"},
		55,
		"receipt persisted",
		map[string]string{"phase": "receipt"},
	)
	if err != nil {
		t.Fatalf("retry task checkpoint publication: %v", err)
	}
	if publishCalls != 2 {
		t.Fatalf("checkpoint publication calls=%d, want 2", publishCalls)
	}
	if tracker.waitCalls != 1 || tracker.invariantCalls != 1 {
		t.Fatalf("checkpoint wait calls=%d invariant calls=%d, want 1/1", tracker.waitCalls, tracker.invariantCalls)
	}
	if tracker.waitReceipt.EventID != receipt.EventID || tracker.waitReceipt.PublishedRootHash != receipt.PublishedRootHash {
		t.Fatalf("checkpoint tracked receipt=%+v, want %+v", tracker.waitReceipt, receipt)
	}
}

func TestCheckpointPayloadBoundsTypedNotAcceptedOutcomes(t *testing.T) {
	manager, runCtx, tracker := newCheckpointPublicationTestContext(t, "bounded-not-accepted")
	manager.waitTaskCheckpointPublicationRetry = func(context.Context, int) error { return nil }
	publishCalls := 0
	manager.publishTaskCheckpoint = func(context.Context, Record, Event) (db.PublishedWriteReceipt, error) {
		publishCalls++
		return db.PublishedWriteReceipt{}, &swarmionapp.CommitNotAcceptedError{Cause: errors.New("event admission did not persist")}
	}

	err := runCtx.CheckpointPayload(checkpointPayload{Value: "input", Receipt: "event/root"}, 55, "receipt persisted", nil)
	if err == nil || !strings.Contains(err.Error(), "remained unresolved after 20 retryable attempts") {
		t.Fatalf("bounded checkpoint publication error=%v", err)
	}
	if publishCalls != taskCheckpointPublicationMaxAttempts {
		t.Fatalf("checkpoint publication calls=%d, want %d", publishCalls, taskCheckpointPublicationMaxAttempts)
	}
	if tracker.waitCalls != 0 || tracker.invariantCalls != 0 {
		t.Fatalf("unpublished checkpoint wait calls=%d invariant calls=%d, want 0/0", tracker.waitCalls, tracker.invariantCalls)
	}
	if IsPermanent(err) {
		t.Fatalf("explicit not-accepted-safe-to-retry outcome was marked permanent: %v", err)
	}
}

func TestCheckpointPayloadDoesNotRetryUnsafePublicationErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "receipt unavailable",
			err:  fmt.Errorf("bootstrap incomplete: %w", db.ErrOperationReceiptUnavailable),
			want: db.ErrOperationReceiptUnavailable,
		},
		{
			name: "operation key conflict",
			err:  fmt.Errorf("different immutable intent: %w", swarmionprotocol.ErrOperationKeyConflict),
			want: swarmionprotocol.ErrOperationKeyConflict,
		},
		{
			name: "typed content conflict",
			err:  &swarmionapp.ContentConflictError{CandidateRootHash: "abc", ProtocolRootHash: "def"},
			want: swarmionapp.ErrContentConflict,
		},
		{
			name: "dirty operation workspace",
			err:  fmt.Errorf("ordinary draft remained: %w", swarmionapp.ErrOperationWorkspaceDirty),
			want: swarmionapp.ErrOperationWorkspaceDirty,
		},
		{
			name: "ambiguous apply rollback failure",
			err:  errors.New("ambiguous task checkpoint apply/rollback failure"),
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, runCtx, tracker := newCheckpointPublicationTestContext(t, strings.ReplaceAll(tt.name, " ", "-"))
			waitCalls := 0
			manager.waitTaskCheckpointPublicationRetry = func(context.Context, int) error {
				waitCalls++
				return nil
			}
			publishCalls := 0
			manager.publishTaskCheckpoint = func(context.Context, Record, Event) (db.PublishedWriteReceipt, error) {
				publishCalls++
				return db.PublishedWriteReceipt{}, tt.err
			}

			err := runCtx.CheckpointPayload(checkpointPayload{Value: "input", Receipt: "event/root"}, 55, "receipt persisted", nil)
			if tt.want != nil && !errors.Is(err, tt.want) {
				t.Fatalf("checkpoint publication error=%v, want %v", err, tt.want)
			}
			if err == nil || !IsPermanent(err) {
				t.Fatalf("unsafe no-receipt checkpoint error=%v permanent=%t, want permanent", err, IsPermanent(err))
			}
			if publishCalls != 1 || waitCalls != 0 {
				t.Fatalf("checkpoint publication calls=%d retry waits=%d, want 1/0", publishCalls, waitCalls)
			}
			if tracker.waitCalls != 0 || tracker.invariantCalls != 0 {
				t.Fatalf("unsafe publication wait calls=%d invariant calls=%d, want 0/0", tracker.waitCalls, tracker.invariantCalls)
			}
		})
	}
}

func TestUnsafeTaskCheckpointPublicationDoesNotReplayStream(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "dirty operation workspace",
			err:  fmt.Errorf("task checkpoint workspace: %w", swarmionapp.ErrOperationWorkspaceDirty),
		},
		{
			name: "ambiguous apply rollback",
			err:  errors.New("ambiguous task checkpoint apply/rollback response"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := openTaskTestDB(t)
			manager := NewManager(store)
			operationKey, err := db.NewPublishedWriteOperationKey()
			if err != nil {
				t.Fatal(err)
			}
			publishCalls := 0
			manager.publishTaskCheckpoint = func(context.Context, Record, Event) (db.PublishedWriteReceipt, error) {
				publishCalls++
				return db.PublishedWriteReceipt{}, tt.err
			}
			runs := 0
			if err := Register(manager, Stream[checkpointPayload, testResult]{
				Name: "test.checkpoint-no-replay." + strings.ReplaceAll(tt.name, " ", "-"),
				Run: func(_ context.Context, task *RunContext[checkpointPayload]) (testResult, error) {
					runs++
					payload := task.Payload()
					payload.Receipt = "domain-event/domain-root"
					if checkpointErr := task.CheckpointPayloadWithOperationKey(
						operationKey,
						payload,
						55,
						"persist exact domain receipt",
						nil,
					); checkpointErr != nil {
						return testResult{}, checkpointErr
					}
					return testResult{Done: "unexpected"}, nil
				},
			}); err != nil {
				t.Fatal(err)
			}
			streamName := "test.checkpoint-no-replay." + strings.ReplaceAll(tt.name, " ", "-")
			record, err := Enqueue(manager, EnqueueOptions[checkpointPayload]{
				Stream:      streamName,
				SubjectType: "test-subject",
				SubjectID:   "subject-" + strings.ReplaceAll(tt.name, " ", "-"),
				Title:       "checkpoint no replay",
				Payload:     checkpointPayload{Value: "input"},
				MaxAttempts: 3,
			})
			if err != nil {
				t.Fatal(err)
			}
			for attempt := 0; attempt < 3; attempt++ {
				_ = manager.RunPending(context.Background())
			}
			done, err := manager.Get(record.ID)
			if err != nil {
				t.Fatal(err)
			}
			if done.Status != StatusFailed || done.Attempts != 1 || runs != 1 || publishCalls != 1 {
				t.Fatalf(
					"unsafe checkpoint task status=%s attempts=%d runs=%d publications=%d, want failed/1/1/1",
					done.Status,
					done.Attempts,
					runs,
					publishCalls,
				)
			}
		})
	}
}

func TestCheckpointPayloadFailsClosedOnReceiptIdentityConflict(t *testing.T) {
	manager, runCtx, tracker := newCheckpointPublicationTestContext(t, "receipt-identity-conflict")
	receipt := db.PublishedWriteReceipt{
		OutcomeUncertain:  true,
		EventID:           swarmionprotocol.NewEventID("task checkpoint uncertain identity").String(),
		PublishedRootHash: swarmionprotocol.NewRootHash("task checkpoint uncertain root").String(),
	}
	conflict := &db.PublishedWriteReceiptIdentityConflictError{
		Receipt:                   receipt,
		ResolvedEventID:           swarmionprotocol.NewEventID("task checkpoint mismatched identity").String(),
		ResolvedPublishedRootHash: swarmionprotocol.NewRootHash("task checkpoint mismatched root").String(),
		Cause:                     errors.New("injected receipt identity mismatch"),
	}
	publishCalls := 0
	manager.publishTaskCheckpoint = func(context.Context, Record, Event) (db.PublishedWriteReceipt, error) {
		publishCalls++
		return receipt, conflict
	}

	err := runCtx.CheckpointPayload(checkpointPayload{Value: "input", Receipt: "event/root"}, 55, "receipt persisted", nil)
	if !errors.Is(err, db.ErrPublishedWriteReceiptIdentityConflict) {
		t.Fatalf("checkpoint receipt identity conflict error=%v", err)
	}
	if publishCalls != 1 {
		t.Fatalf("checkpoint identity-conflict publication calls=%d, want 1", publishCalls)
	}
	if tracker.waitCalls != 0 || tracker.invariantCalls != 0 {
		t.Fatalf("identity conflict was suppressed into receipt tracking: waits=%d invariants=%d", tracker.waitCalls, tracker.invariantCalls)
	}
}

func TestCheckpointPayloadTracksPublishedReceiptDespitePublicationError(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)
	queued, err := Enqueue(manager, EnqueueOptions[checkpointPayload]{
		Stream:      "test.checkpoint-partial-publication",
		SubjectType: "test-subject",
		SubjectID:   "subject-checkpoint-partial-publication",
		Title:       "checkpoint partial publication task",
		Payload:     checkpointPayload{Value: "input"},
		MaxAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	receipt := db.PublishedWriteReceipt{
		Committed:         true,
		EventID:           swarmionprotocol.NewEventID("task checkpoint partial publication event").String(),
		PublishedRootHash: swarmionprotocol.NewRootHash("task checkpoint partial publication root").String(),
	}
	publishCalls := 0
	publishErr := errors.New("ambiguous transport error after event publication")
	manager.publishTaskCheckpoint = func(ctx context.Context, _ Record, _ Event) (db.PublishedWriteReceipt, error) {
		publishCalls++
		if _, hasDeadline := ctx.Deadline(); !hasDeadline || ctx.Err() != nil {
			t.Fatalf("checkpoint publication did not use a fresh bounded context")
		}
		return receipt, publishErr
	}
	tracker := &fakeTaskCheckpointTracker{
		observation: db.EventReceiptObservation{
			State: db.EventReceiptStateAppliedDurably,
			Status: swarmionapp.BranchEventReceiptStatus{
				AppliedDurably:            true,
				ContentCoverage:           swarmionapp.BranchEventContentCoverageDissent,
				CheckpointCommitID:        "event-task-checkpoint",
				DurableCheckpointCommitID: "later-durable-task-head",
			},
		},
		found: true,
	}
	manager.checkpointTracker = tracker
	running := queued
	running.Status = StatusRunning
	running.Message = "running"
	running.Attempts = 1
	running.StartedAt = time.Now().UTC()
	running.UpdatedAt = running.StartedAt
	tracker.record = running
	tracker.record.Status = StatusRunning
	tracker.record.Progress = 55
	tracker.record.Message = "receipt persisted"
	tracker.record.Payload = json.RawMessage(`{"receipt":"event/root","value":"input"}`)
	runCtx := &RunContext[checkpointPayload]{
		manager: manager,
		record:  running,
		payload: checkpointPayload{Value: "input"},
	}
	if err := runCtx.CheckpointPayload(
		checkpointPayload{Value: "input", Receipt: "event/root"},
		55,
		"receipt persisted",
		nil,
	); err != nil {
		t.Fatalf("checkpoint should resolve the exact published event despite its accompanying error: %v", err)
	}
	if publishCalls != 1 || tracker.waitCalls != 1 || tracker.waitReceipt.EventID != receipt.EventID || tracker.waitReceipt.PublishedRootHash != receipt.PublishedRootHash {
		t.Fatalf("partial publication calls=%d wait_calls=%d waited_receipt=%+v, want one publication and exact receipt %+v", publishCalls, tracker.waitCalls, tracker.waitReceipt, receipt)
	}
	if tracker.invariantCommitID != "event-task-checkpoint" {
		t.Fatalf("partial publication invariant checkpoint=%q, want exact event checkpoint", tracker.invariantCommitID)
	}
}

func TestStreamRunCarriesOwnedRecordAcrossPayloadCheckpointAndRetry(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)
	queued, err := Enqueue(manager, EnqueueOptions[checkpointPayload]{
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
	// this owned record through its payload checkpoint and retry instead of
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
			if err := task.CheckpointPayload(payload, 55, "receipt persisted", nil); err != nil {
				t.Fatal(err)
			}
			if task.Task().Attempts != 1 {
				t.Fatalf("checkpointed attempts=%d, want 1", task.Task().Attempts)
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
	queued, err := Enqueue(manager, EnqueueOptions[checkpointPayload]{
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
			if err := task.CheckpointPayload(payload, 55, "receipt persisted", nil); err != nil {
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
	queued, err := Enqueue(manager, EnqueueOptions[checkpointPayload]{
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
			return testResult{}, task.CheckpointPayload(payload, 55, "receipt persisted", nil)
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

	interrupted, err := Enqueue(ownerA, EnqueueOptions[checkpointPayload]{
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
	otherOwner, err := Enqueue(ownerB, EnqueueOptions[checkpointPayload]{
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
	record, err := Enqueue(original, EnqueueOptions[checkpointPayload]{
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

	interrupted, err := Enqueue(original, EnqueueOptions[checkpointPayload]{
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
	barrier, err := Enqueue(manager, EnqueueOptions[checkpointPayload]{
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
	record, err := Enqueue(staging, EnqueueOptions[checkpointPayload]{
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
	record, err := Enqueue(manager, EnqueueOptions[checkpointPayload]{
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
	bad, err := Enqueue(manager, EnqueueOptions[testPayload]{
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
	good, err := Enqueue(manager, EnqueueOptions[testPayload]{
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
	record, err := Enqueue(manager, EnqueueOptions[testPayload]{
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
	record, err := Enqueue(manager, EnqueueOptions[testPayload]{
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
	record, err := Enqueue(manager, EnqueueOptions[testPayload]{
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

func TestForeignUnavailableTaskCheckpointDefersWithoutSQLOrLocalPublication(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)
	operationKey, err := db.NewPublishedWriteOperationKey()
	if err != nil {
		t.Fatal(err)
	}
	publishCalls := 0
	manager.publishTaskCheckpoint = func(context.Context, Record, Event) (db.PublishedWriteReceipt, error) {
		publishCalls++
		return db.PublishedWriteReceipt{}, errors.New("publisher must not run for a foreign unavailable operation")
	}
	var metricsAtBoundary db.TransactionMetricsSnapshot
	eventsAtBoundary := 0
	if err := Register(manager, Stream[checkpointPayload, testResult]{
		Name: "test.foreign-checkpoint-unavailable",
		Run: func(_ context.Context, task *RunContext[checkpointPayload]) (testResult, error) {
			metricsAtBoundary = store.TransactionMetrics()
			eventsAtBoundary = taskEventCount(t, store, task.Task().ID)
			payload := task.Payload()
			payload.Receipt = "foreign-event/foreign-root"
			return testResult{}, task.CheckpointPayloadWithOperationAuthor(
				operationKey,
				"unknown-foreign-checkpoint-author",
				payload,
				55,
				"persist foreign receipt",
				nil,
			)
		},
	}); err != nil {
		t.Fatal(err)
	}
	record, err := Enqueue(manager, EnqueueOptions[checkpointPayload]{
		Stream:      "test.foreign-checkpoint-unavailable",
		SubjectType: "test-subject",
		SubjectID:   "subject-foreign-checkpoint-unavailable",
		Title:       "foreign checkpoint unavailable",
		Payload:     checkpointPayload{Value: "input"},
		MaxAttempts: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = manager.RunPending(context.Background())
	if publishCalls != 0 {
		t.Fatalf("foreign unavailable checkpoint publisher calls=%d, want 0", publishCalls)
	}
	if after := store.TransactionMetrics(); after != metricsAtBoundary {
		t.Fatalf("foreign unavailable checkpoint wrote task SQL: boundary=%+v after=%+v", metricsAtBoundary, after)
	}
	if got := taskEventCount(t, store, record.ID); got != eventsAtBoundary {
		t.Fatalf("foreign unavailable checkpoint inserted task event: boundary=%d after=%d", eventsAtBoundary, got)
	}
	owned, err := manager.Get(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	var payload checkpointPayload
	if err := json.Unmarshal(owned.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if owned.Status != StatusRunning || owned.Attempts != 1 || payload.Receipt != "" {
		t.Fatalf("foreign unavailable checkpoint changed task: record=%+v payload=%+v", owned, payload)
	}
}

func TestExactPendingTaskCheckpointDefersWithoutErrorPathWrite(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)
	operationKey, err := db.NewPublishedWriteOperationKey()
	if err != nil {
		t.Fatal(err)
	}
	receipt := db.PublishedWriteReceipt{
		Committed:         true,
		EventID:           swarmionprotocol.NewEventID("deferred task checkpoint event").String(),
		PublishedRootHash: swarmionprotocol.NewRootHash("deferred task checkpoint root").String(),
	}
	publishCalls := 0
	manager.publishTaskCheckpoint = func(context.Context, Record, Event) (db.PublishedWriteReceipt, error) {
		publishCalls++
		return receipt, nil
	}
	pendingObservation := db.EventReceiptObservation{
		Receipt: receipt,
		State:   db.EventReceiptStatePending,
		Status: swarmionapp.BranchEventReceiptStatus{
			EventID:                   receipt.EventID,
			ExpectedPublishedRootHash: receipt.PublishedRootHash,
			Known:                     true,
			ContentCoverage:           swarmionapp.BranchEventContentCoveragePending,
		},
	}
	manager.checkpointTracker = &fakeTaskCheckpointTracker{
		observation: pendingObservation,
		waitErr: &db.EventReceiptPendingError{
			Observation: pendingObservation,
			Reason:      "test exact checkpoint remains pending",
			Cause:       context.DeadlineExceeded,
		},
	}
	var metricsAtBoundary db.TransactionMetricsSnapshot
	eventsAtBoundary := 0
	if err := Register(manager, Stream[checkpointPayload, testResult]{
		Name: "test.exact-checkpoint-pending",
		Run: func(_ context.Context, task *RunContext[checkpointPayload]) (testResult, error) {
			metricsAtBoundary = store.TransactionMetrics()
			eventsAtBoundary = taskEventCount(t, store, task.Task().ID)
			payload := task.Payload()
			payload.Receipt = "exact-event/exact-root"
			return testResult{}, task.CheckpointPayloadWithOperationKey(
				operationKey,
				payload,
				55,
				"persist exact receipt",
				nil,
			)
		},
	}); err != nil {
		t.Fatal(err)
	}
	record, err := Enqueue(manager, EnqueueOptions[checkpointPayload]{
		Stream:      "test.exact-checkpoint-pending",
		SubjectType: "test-subject",
		SubjectID:   "subject-exact-checkpoint-pending",
		Title:       "exact checkpoint pending",
		Payload:     checkpointPayload{Value: "input"},
		MaxAttempts: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = manager.RunPending(context.Background())
	if publishCalls != 1 {
		t.Fatalf("exact pending checkpoint publications=%d, want 1", publishCalls)
	}
	if after := store.TransactionMetrics(); after != metricsAtBoundary {
		t.Fatalf("exact pending checkpoint error path wrote task SQL: boundary=%+v after=%+v", metricsAtBoundary, after)
	}
	if got := taskEventCount(t, store, record.ID); got != eventsAtBoundary {
		t.Fatalf("exact pending checkpoint error path inserted task event: boundary=%d after=%d", eventsAtBoundary, got)
	}
	owned, err := manager.Get(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if owned.Status != StatusRunning || owned.Attempts != 1 {
		t.Fatalf("exact pending checkpoint status=%s attempts=%d, want running/1", owned.Status, owned.Attempts)
	}
}

func TestManagerStopJoinsInFlightCheckpointBeforeDatabaseCloseAndRestartRecovery(t *testing.T) {
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
	const databaseName = "protos_task_runner_stop_restart_test"
	transport := testswarmion.New(t, key)
	store, err := db.Open(workDir, databaseName, key, transport.Link)
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

	manager := NewManager(store)
	blockingTracker := &blockingTaskCheckpointTracker{
		delegate: manager.checkpointTracker,
		entered:  make(chan struct{}),
		release:  make(chan struct{}),
	}
	manager.checkpointTracker = blockingTracker
	const persistedReceipt = "domain-event-id/domain-published-root"
	if err := Register(manager, Stream[checkpointPayload, testResult]{
		Name: "test.stop-inflight-checkpoint",
		Run: func(ctx context.Context, task *RunContext[checkpointPayload]) (testResult, error) {
			payload := task.Payload()
			payload.Receipt = persistedReceipt
			if err := task.CheckpointPayload(payload, 55, "receipt persisted", nil); err != nil {
				return testResult{}, err
			}
			<-ctx.Done()
			return testResult{}, ctx.Err()
		},
	}); err != nil {
		t.Fatal(err)
	}
	queued, err := Enqueue(manager, EnqueueOptions[checkpointPayload]{
		Stream:      "test.stop-inflight-checkpoint",
		SubjectType: "test-subject",
		SubjectID:   "subject-stop-inflight-checkpoint",
		Title:       "stop in-flight checkpoint task",
		Payload:     checkpointPayload{Value: "input"},
		MaxAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	stop := manager.Start(context.Background(), 10*time.Millisecond)
	var releaseOnce sync.Once
	releaseCheckpoint := func() {
		releaseOnce.Do(func() { close(blockingTracker.release) })
	}
	defer func() {
		releaseCheckpoint()
		if err := stop(); err != nil {
			t.Errorf("stop task runner cleanup: %v", err)
		}
	}()
	select {
	case <-blockingTracker.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for task checkpoint status wait")
	}

	stopResult := make(chan error, 1)
	go func() { stopResult <- stop() }()
	select {
	case err := <-stopResult:
		t.Fatalf("task runner stopper returned before in-flight checkpoint completed: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	releaseCheckpoint()
	select {
	case err := <-stopResult:
		if err != nil {
			t.Fatalf("join task runner after checkpoint: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out joining task runner after checkpoint completed")
	}

	interrupted, err := manager.Get(queued.ID)
	if err != nil {
		t.Fatal(err)
	}
	if interrupted.Status != StatusRunning || interrupted.Attempts != 1 {
		t.Fatalf("joined task status=%s attempts=%d, want recoverable running/1", interrupted.Status, interrupted.Attempts)
	}
	var interruptedPayload checkpointPayload
	if err := json.Unmarshal(interrupted.Payload, &interruptedPayload); err != nil {
		t.Fatal(err)
	}
	if interruptedPayload.Receipt != persistedReceipt {
		t.Fatalf("joined task receipt=%q, want %q", interruptedPayload.Receipt, persistedReceipt)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("close database after joined task runner: %v", err)
	}
	activeStore = nil
	store, err = db.Open(workDir, databaseName, key, transport.Link)
	if err != nil {
		t.Fatalf("reopen database after joined task runner: %v", err)
	}
	activeStore = store

	restarted := NewManager(store)
	resumedRuns := make(chan struct{}, 1)
	if err := Register(restarted, Stream[checkpointPayload, testResult]{
		Name: "test.stop-inflight-checkpoint",
		Run: func(_ context.Context, task *RunContext[checkpointPayload]) (testResult, error) {
			if task.Payload().Receipt != persistedReceipt {
				return testResult{}, fmt.Errorf("restarted receipt=%q, want %q", task.Payload().Receipt, persistedReceipt)
			}
			resumedRuns <- struct{}{}
			return testResult{Done: task.Payload().Receipt}, nil
		},
	}); err != nil {
		t.Fatal(err)
	}
	restartedStop := restarted.Start(context.Background(), 10*time.Millisecond)
	defer func() {
		if err := restartedStop(); err != nil {
			t.Errorf("stop restarted task runner: %v", err)
		}
	}()
	select {
	case <-resumedRuns:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for restarted task to resume")
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		done, err := restarted.Get(queued.ID)
		if err != nil {
			t.Fatal(err)
		}
		if done.Status == StatusSucceeded {
			if done.Attempts != 1 {
				t.Fatalf("restarted task attempts=%d, want resumed logical attempt 1", done.Attempts)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("restarted task status=%s attempts=%d, want succeeded/1", done.Status, done.Attempts)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := restartedStop(); err != nil {
		t.Fatalf("join restarted task runner: %v", err)
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

	queued, err := Enqueue(manager, EnqueueOptions[testPayload]{
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

	queued, err := Enqueue(manager, EnqueueOptions[testPayload]{
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

	queued, err := Enqueue(ownerA, EnqueueOptions[testPayload]{
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

	queued, err := Enqueue(manager, EnqueueOptions[testPayload]{
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
