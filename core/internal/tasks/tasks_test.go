package tasks

import (
	"context"
	"encoding/json"
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
	rows, err := store.QueryContext(context.Background(), "SELECT COUNT(*) FROM task_events WHERE task_id = ?", db.MustUUIDBytes(queued.ID))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("event count query returned no rows")
	}
	if err := rows.Scan(&eventCount); err != nil {
		t.Fatal(err)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if eventCount < 3 {
		t.Fatalf("event count = %d, want at least queued/running/progress/succeeded events", eventCount)
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

func taskEventCount(t *testing.T, store *db.DB, taskID string) int {
	t.Helper()
	rows, err := store.QueryContext(context.Background(), "SELECT COUNT(*) FROM task_events WHERE task_id = ?", db.MustUUIDBytes(taskID))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("event count query returned no rows")
	}
	var count int
	if err := rows.Scan(&count); err != nil {
		t.Fatal(err)
	}
	if err := rows.Err(); err != nil {
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
