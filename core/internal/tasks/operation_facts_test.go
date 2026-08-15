package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/protosio/protos/internal/db"
)

func TestOperationFactIsDeterministicAndImmutable(t *testing.T) {
	store := openTaskTestDB(t)
	manager := NewManager(store)
	taskID := db.MustNewUUIDv7()
	operation, err := db.NewPublishedWriteOperation(
		"0123456789abcdef0123456789abcdef",
		"protos:test:immutable-operation-fact",
		taskID,
	)
	if err != nil {
		t.Fatal(err)
	}
	operation.AuthorPeerID = "peer-a"

	left, err := NewOperationFact(
		taskID,
		OperationFactKindEffect,
		operation,
		"instance",
		"instance-1",
		json.RawMessage(`{"operation":"delete","invariant":{"kind":"absent","id":"instance-1"}}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	right, err := NewOperationFact(
		taskID,
		OperationFactKindEffect,
		operation,
		"instance",
		"instance-1",
		json.RawMessage(`{
          "invariant": {"id":"instance-1", "kind":"absent"},
          "operation": "delete"
        }`),
	)
	if err != nil {
		t.Fatal(err)
	}
	if left.ID != right.ID || string(left.Payload) != string(right.Payload) {
		t.Fatalf("semantic equivalents produced different facts: left=%+v right=%+v", left, right)
	}

	if err := manager.EnsureOperationFact(context.Background(), left); err != nil {
		t.Fatal(err)
	}
	if err := manager.EnsureOperationFact(context.Background(), right); err != nil {
		t.Fatalf("identical fact was not idempotent: %v", err)
	}
	stored, found, err := manager.OperationFact(context.Background(), taskID, OperationFactKindEffect)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("operation fact was not found after publication")
	}
	if err := compareOperationFacts(stored, left); err != nil {
		t.Fatalf("stored fact changed: %v", err)
	}

	conflicting := left
	conflicting.Payload = json.RawMessage(`{"operation":"recreate"}`)
	if err := manager.EnsureOperationFact(context.Background(), conflicting); !errors.Is(err, ErrOperationFactConflict) {
		t.Fatalf("conflicting immutable fact error=%v, want ErrOperationFactConflict", err)
	}
}

func TestOperationFactIdentityRejectsMutableOrMalformedInputs(t *testing.T) {
	operation, err := db.NewPublishedWriteOperation(
		"0123456789abcdef0123456789abcdef",
		"protos:test:immutable-operation-fact",
	)
	if err != nil {
		t.Fatal(err)
	}
	operation.AuthorPeerID = "peer-a"
	if _, err := NewOperationFact("not-a-uuid", OperationFactKindReceipt, operation, "instance", "instance-1", struct{}{}); err == nil {
		t.Fatal("malformed task ID was accepted")
	}
	operation.IntentDigest = "not-a-digest"
	if _, err := NewOperationFact(db.MustNewUUIDv7(), OperationFactKindReceipt, operation, "instance", "instance-1", struct{}{}); err == nil {
		t.Fatal("malformed operation digest was accepted")
	}
}

func TestOperationFactMatchesAtCheckpoint(t *testing.T) {
	t.Setenv("SWARMION_CHECKPOINT_SCHEDULER_PROFILE", "single_node_development")
	store := openTaskTestDB(t)
	manager := NewManager(store)

	fact := newCheckpointTestOperationFact(t, db.MustNewUUIDv7(), "instance-1")
	checkpointCommitID := publishOperationFactAndWaitDurable(t, store, fact)

	equivalent := fact
	equivalent.Payload = json.RawMessage(`{
      "invariant": {"id": "instance-1", "kind": "absent"},
      "operation": "delete"
    }`)
	matches, err := manager.OperationFactMatchesAtCheckpoint(
		context.Background(),
		checkpointCommitID,
		equivalent,
	)
	if err != nil {
		t.Fatalf("match identical operation fact at checkpoint: %v", err)
	}
	if !matches {
		t.Fatal("identical operation fact was reported absent")
	}

	absent := newCheckpointTestOperationFact(t, db.MustNewUUIDv7(), "instance-absent")
	matches, err = manager.OperationFactMatchesAtCheckpoint(
		context.Background(),
		checkpointCommitID,
		absent,
	)
	if err != nil {
		t.Fatalf("read absent operation fact at checkpoint: %v", err)
	}
	if matches {
		t.Fatal("absent operation fact was reported present")
	}

	conflicting := fact
	conflicting.Payload = json.RawMessage(`{"operation":"recreate"}`)
	matches, err = manager.OperationFactMatchesAtCheckpoint(
		context.Background(),
		checkpointCommitID,
		conflicting,
	)
	if matches || !errors.Is(err, ErrOperationFactConflict) {
		t.Fatalf("conflicting operation fact match=%t error=%v, want false and ErrOperationFactConflict", matches, err)
	}

	malformedExpected := fact
	malformedExpected.ID = "not-the-derived-identity"
	if matches, err = manager.OperationFactMatchesAtCheckpoint(
		context.Background(),
		checkpointCommitID,
		malformedExpected,
	); matches || err == nil {
		t.Fatalf("malformed expected operation fact match=%t error=%v, want validation error", matches, err)
	}
}

func TestOperationFactMatchesAtCheckpointRejectsMalformedStoredIdentity(t *testing.T) {
	t.Setenv("SWARMION_CHECKPOINT_SCHEDULER_PROFILE", "single_node_development")
	store := openTaskTestDB(t)
	manager := NewManager(store)
	expected := newCheckpointTestOperationFact(t, db.MustNewUUIDv7(), "instance-malformed")

	malformed := expected
	malformed.Kind = OperationFactKindReceipt
	checkpointCommitID := publishOperationFactAndWaitDurable(t, store, malformed)

	matches, err := manager.OperationFactMatchesAtCheckpoint(
		context.Background(),
		checkpointCommitID,
		expected,
	)
	if matches || !errors.Is(err, ErrOperationFactConflict) {
		t.Fatalf("malformed stored operation fact match=%t error=%v, want false and ErrOperationFactConflict", matches, err)
	}
}

func newCheckpointTestOperationFact(t *testing.T, taskID, subjectID string) OperationFact {
	t.Helper()
	operation, err := db.NewPublishedWriteOperation(
		"fedcba9876543210fedcba9876543210",
		"protos:test:checkpoint-operation-fact",
		taskID,
	)
	if err != nil {
		t.Fatal(err)
	}
	operation.AuthorPeerID = "peer-checkpoint-author"
	fact, err := NewOperationFact(
		taskID,
		OperationFactKindEffect,
		operation,
		"instance",
		subjectID,
		json.RawMessage(`{"operation":"delete","invariant":{"kind":"absent","id":"`+subjectID+`"}}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	return fact
}

func publishOperationFactAndWaitDurable(t *testing.T, store *db.DB, fact OperationFact) string {
	t.Helper()
	receipt, err := db.InsertWithReceiptContext(
		context.Background(),
		store,
		InsertOperationFactMapper(fact),
	)
	if err != nil {
		t.Fatalf("publish operation fact: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	observation, err := store.WaitForPublishedWriteApplied(ctx, receipt, "operation fact checkpoint test")
	if err != nil {
		t.Fatalf("wait for operation fact durable checkpoint: %v", err)
	}
	if !observation.Status.AppliedDurably || observation.Status.DurableCheckpointCommitID == "" {
		t.Fatalf("operation fact did not reach an exact durable checkpoint: %+v", observation.Status)
	}
	return observation.Status.DurableCheckpointCommitID
}
