package tasks

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/pcrypto"
)

type testPayload struct {
	Value string `json:"value"`
}

type testResult struct {
	Done string `json:"done"`
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
	store, err := db.Open(workDir, "protos_tasks_test", key)
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
