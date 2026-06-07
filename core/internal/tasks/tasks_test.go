package tasks

import (
	"context"
	"testing"

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
