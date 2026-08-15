package provisioners

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/tasks"
)

func TestInstanceDeleteReceiptFactVersionCompatibility(t *testing.T) {
	taskID, identity, receipt := instanceDeleteReceiptFactCompatibilityFixture(t)
	legacy := legacyInstanceDeleteReceiptFactForTest(t, taskID, identity, receipt)
	current, err := newInstanceDeleteReceiptFact(receipt, identity)
	if err != nil {
		t.Fatal(err)
	}
	conflictingReceipt := receipt
	conflictingReceipt.EventID = strings.Repeat("c", 64)
	conflicting, err := newInstanceDeleteReceiptFact(conflictingReceipt, identity)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		legacy         *tasks.OperationFact
		current        *tasks.OperationFact
		wantFound      bool
		wantLegacyData bool
		wantConflict   bool
	}{
		{name: "old only", legacy: &legacy, wantFound: true, wantLegacyData: true},
		{name: "new only", current: &current, wantFound: true},
		{name: "both matching", legacy: &legacy, current: &current, wantFound: true, wantLegacyData: true},
		{name: "both conflicting", legacy: &legacy, current: &conflicting, wantConflict: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found, err := reconcileInstanceDeleteReceiptFacts(taskID, identity, tt.legacy, tt.current)
			if tt.wantConflict {
				if found || !errors.Is(err, tasks.ErrOperationFactConflict) {
					t.Fatalf("found=%t error=%v, want fail-closed fact conflict", found, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if found != tt.wantFound {
				t.Fatalf("found=%t, want %t", found, tt.wantFound)
			}
			if got.EventID != receipt.EventID || got.PublishedRootHash != receipt.PublishedRootHash {
				t.Fatalf("receipt identity=%s/%s, want %s/%s", got.EventID, got.PublishedRootHash, receipt.EventID, receipt.PublishedRootHash)
			}
			if tt.wantLegacyData {
				if got.EventDigest != receipt.EventDigest || got.AuthorSeq != receipt.AuthorSeq {
					t.Fatalf("legacy metadata digest=%q seq=%d, want %q/%d", got.EventDigest, got.AuthorSeq, receipt.EventDigest, receipt.AuthorSeq)
				}
			} else if got.EventDigest != "" || got.AuthorSeq != 0 {
				t.Fatalf("v2-only receipt acquired legacy metadata digest=%q seq=%d", got.EventDigest, got.AuthorSeq)
			}
		})
	}
}

func TestInstanceDeleteReceiptFactVersionsPublishConcurrentlyWithoutIDCollision(t *testing.T) {
	store := openProvisionerTestDB(t)
	manager := tasks.NewManager(store)
	taskID, identity, receipt := instanceDeleteReceiptFactCompatibilityFixture(t)
	legacy := legacyInstanceDeleteReceiptFactForTest(t, taskID, identity, receipt)
	current, err := newInstanceDeleteReceiptFact(receipt, identity)
	if err != nil {
		t.Fatal(err)
	}
	if legacy.ID == current.ID {
		t.Fatalf("legacy and current receipt facts share deterministic ID %q", legacy.ID)
	}

	start := make(chan struct{})
	errs := make(chan error, 2)
	var writers sync.WaitGroup
	for _, fact := range []tasks.OperationFact{legacy, current} {
		fact := fact
		writers.Add(1)
		go func() {
			defer writers.Done()
			<-start
			errs <- manager.EnsureOperationFact(context.Background(), fact)
		}()
	}
	close(start)
	writers.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("publish additive receipt fact: %v", err)
		}
	}

	storedLegacy, legacyFound, err := manager.OperationFact(context.Background(), taskID, tasks.OperationFactKindReceipt)
	if err != nil {
		t.Fatal(err)
	}
	storedCurrent, currentFound, err := manager.OperationFact(context.Background(), taskID, tasks.OperationFactKindReceiptV2)
	if err != nil {
		t.Fatal(err)
	}
	if !legacyFound || !currentFound {
		t.Fatalf("coexisting facts found legacy=%t current=%t", legacyFound, currentFound)
	}
	got, found, err := reconcileInstanceDeleteReceiptFacts(taskID, identity, &storedLegacy, &storedCurrent)
	if err != nil || !found {
		t.Fatalf("reconcile coexisting receipt facts found=%t error=%v", found, err)
	}
	if got.EventDigest != receipt.EventDigest || got.AuthorSeq != receipt.AuthorSeq {
		t.Fatalf("coexisting legacy metadata digest=%q seq=%d, want %q/%d", got.EventDigest, got.AuthorSeq, receipt.EventDigest, receipt.AuthorSeq)
	}

	var currentPayload map[string]json.RawMessage
	if err := json.Unmarshal(storedCurrent.Payload, &currentPayload); err != nil {
		t.Fatal(err)
	}
	for _, legacyField := range []string{"event_digest", "author_seq"} {
		if _, exists := currentPayload[legacyField]; exists {
			t.Fatalf("v2 receipt payload retained legacy-only field %q: %s", legacyField, storedCurrent.Payload)
		}
	}
}

func instanceDeleteReceiptFactCompatibilityFixture(
	t *testing.T,
) (string, instanceDeleteOperationIdentity, instanceDeleteOperationReceipt) {
	t.Helper()
	taskID := db.MustNewUUIDv7()
	identity := instanceDeleteOperationIdentity{
		Key:          strings.Repeat("1", 32),
		IntentDigest: strings.Repeat("2", 64),
		AuthorPeerID: "peer-legacy-author",
		ExpectedInvariant: instanceDeleteInvariant{
			Kind:       instanceDeleteInvariantAbsent,
			InstanceID: db.MustNewUUIDv7(),
		},
	}
	receipt := instanceDeleteOperationReceipt{
		OperationID:           taskID,
		Operation:             instanceLifecycleOperationDelete,
		ExpectedInvariant:     identity.ExpectedInvariant,
		EventID:               strings.Repeat("a", 64),
		PublishedRootHash:     strings.Repeat("b", 32),
		EventDigest:           strings.Repeat("d", 64),
		AuthorSeq:             17,
		OperationIntentDigest: identity.IntentDigest,
		OperationAuthorPeerID: identity.AuthorPeerID,
	}
	return taskID, identity, receipt
}

func legacyInstanceDeleteReceiptFactForTest(
	t *testing.T,
	taskID string,
	identity instanceDeleteOperationIdentity,
	receipt instanceDeleteOperationReceipt,
) tasks.OperationFact {
	t.Helper()
	fact, err := tasks.NewOperationFact(
		taskID,
		tasks.OperationFactKindReceipt,
		identity.publishedWriteOperation(),
		taskSubjectInstance,
		identity.ExpectedInvariant.InstanceID,
		instanceDeleteReceiptFactPayloadV1{
			OperationID:           receipt.OperationID,
			Operation:             receipt.Operation,
			ExpectedInvariant:     receipt.ExpectedInvariant,
			EventID:               receipt.EventID,
			PublishedRootHash:     receipt.PublishedRootHash,
			EventDigest:           receipt.EventDigest,
			AuthorSeq:             receipt.AuthorSeq,
			OperationIntentDigest: receipt.OperationIntentDigest,
			OperationAuthorPeerID: receipt.OperationAuthorPeerID,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return fact
}
