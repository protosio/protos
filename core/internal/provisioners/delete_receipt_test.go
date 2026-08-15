package provisioners

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	swarmionprotocol "github.com/nustiueudinastea/swarmion/protocol"
	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/tasks"
)

type fakeInstanceDeleteReceiptTracker struct {
	observation    db.EventReceiptObservation
	waitErr        error
	instanceExists bool
	invariantErr   error
	waitCalls      int
	invariantCalls int
	received       []db.PublishedWriteReceipt
	checkpointIDs  []string
}

func (tracker *fakeInstanceDeleteReceiptTracker) WaitForPublishedWriteApplied(_ context.Context, receipt db.PublishedWriteReceipt, _ string) (db.EventReceiptObservation, error) {
	tracker.waitCalls++
	tracker.received = append(tracker.received, receipt)
	return tracker.observation, tracker.waitErr
}

func (tracker *fakeInstanceDeleteReceiptTracker) InstanceExistsAtCheckpoint(_ context.Context, checkpointCommitID string, _ string) (bool, error) {
	tracker.invariantCalls++
	tracker.checkpointIDs = append(tracker.checkpointIDs, checkpointCommitID)
	return tracker.instanceExists, tracker.invariantErr
}

func testInstanceDeleteReceipt() instanceDeleteOperationReceipt {
	return instanceDeleteOperationReceipt{
		OperationID: "delete-operation-1",
		Operation:   instanceLifecycleOperationDelete,
		ExpectedInvariant: instanceDeleteInvariant{
			Kind:       instanceDeleteInvariantAbsent,
			InstanceID: "0198f0fd-0907-7c6b-b571-8bbd9d79473e",
		},
		EventID:           strings.Repeat("a", 64),
		PublishedRootHash: strings.Repeat("b", 32),
		AuthorSeq:         1,
		CommitHash:        "commit-1",
	}
}

func appliedInstanceDeleteObservation(receipt instanceDeleteOperationReceipt, coverage swarmionapp.BranchEventContentCoverage) db.EventReceiptObservation {
	contentDurable := coverage == swarmionapp.BranchEventContentCoverageCovered
	return db.EventReceiptObservation{
		Receipt: db.PublishedWriteReceipt{
			Committed:          true,
			Checkpointed:       true,
			CommitHash:         receipt.CommitHash,
			EventID:            receipt.EventID,
			PublishedRootHash:  receipt.PublishedRootHash,
			CheckpointCommitID: "event-checkpoint-commit",
			CheckpointRootHash: "event-checkpoint-root",
		},
		Status: swarmionapp.ReceiptStatus{
			EventID:                   receipt.EventID,
			ExpectedPublishedRootHash: receipt.PublishedRootHash,
			Known:                     true,
			Checkpointed:              true,
			AppliedDurably:            true,
			Durable:                   contentDurable,
			ContentCoverage:           coverage,
			CheckpointCommitID:        "event-checkpoint-commit",
			CheckpointRootHash:        "event-checkpoint-root",
			DurableCheckpointCommitID: "durable-head-commit",
			DurableCheckpointRootHash: "durable-head-root",
			QueryableRootHash:         "queryable-root",
			DurableProofObservation: &swarmionapp.BranchRootDurableProofObservation{
				TargetRootHash:              receipt.PublishedRootHash,
				QueryableRootHash:           "queryable-root",
				DurableCheckpointCommitID:   "durable-head-commit",
				DurableCheckpointRootHash:   "durable-head-root",
				CandidateCheckpointCommitID: "event-checkpoint-commit",
				CandidateCheckpointRootHash: "event-checkpoint-root",
				CandidateEventID:            receipt.EventID,
				EventContained:              true,
				ProofRan:                    true,
				Covered:                     contentDurable,
				Conflict:                    !contentDurable,
			},
		},
		State: db.EventReceiptStateAppliedDurably,
	}
}

func captureInstanceDeleteReceipts(target *[]instanceDeleteOperationReceipt) func(instanceDeleteOperationReceipt, int, string) error {
	return func(receipt instanceDeleteOperationReceipt, _ int, _ string) error {
		*target = append(*target, *cloneInstanceDeleteOperationReceipt(&receipt))
		return nil
	}
}

func waitForTestPublishedEvent(t *testing.T, store *db.DB, receipt db.PublishedWriteReceipt, reason string) db.EventReceiptObservation {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	observation, err := store.WaitForPublishedWriteApplied(ctx, receipt, reason)
	if err != nil {
		t.Fatalf("wait for published event %s/%s: %v", receipt.EventID, receipt.PublishedRootHash, err)
	}
	if !observation.Status.AppliedDurably {
		t.Fatalf("published event did not reach applied_durably: %+v", observation)
	}
	return observation
}

func insertInstanceForDeleteReceiptTest(t *testing.T, store *db.DB, instance *InstanceInfo) db.PublishedWriteReceipt {
	t.Helper()
	if instance == nil {
		t.Fatal("test instance is nil")
	}
	if strings.TrimSpace(instance.LifecycleOwnerPeerID) == "" {
		status, ok := store.SwarmionStatus()
		if !ok || strings.TrimSpace(status.PeerID) == "" {
			t.Fatal("test database Swarmion identity is unavailable")
		}
		instance.LifecycleOwnerPeerID = status.PeerID
	}
	machine, metadata := createInstanceInsertMapper(*instance)
	receipt, err := db.InsertWithReceiptContext(context.Background(), store, machine, metadata)
	if err != nil {
		t.Fatalf("insert instance %s: %v", instance.ID, err)
	}
	waitForTestPublishedEvent(t, store, receipt, "prepare instance delete receipt test")
	return receipt
}

func instanceWithTestLifecycleOwner(t *testing.T, store *db.DB, instance InstanceInfo) InstanceInfo {
	t.Helper()
	status, ok := store.SwarmionStatus()
	if !ok || strings.TrimSpace(status.PeerID) == "" {
		t.Fatal("test database Swarmion identity is unavailable")
	}
	instance.LifecycleOwnerPeerID = status.PeerID
	return instance
}

func instanceDeleteOperationIdentityForTest(
	t *testing.T,
	store *db.DB,
	operationID string,
	instance InstanceInfo,
	localOnly bool,
) instanceDeleteOperationIdentity {
	t.Helper()
	status, ok := store.SwarmionStatus()
	if !ok || strings.TrimSpace(status.PeerID) == "" {
		t.Fatal("swarmion author is unavailable")
	}
	identity, err := newInstanceDeleteOperationIdentity(operationID, instance, localOnly, status.PeerID)
	if err != nil {
		t.Fatal(err)
	}
	return identity
}

func publishInstanceDeleteForTest(t *testing.T, store *db.DB, instance InstanceInfo, operationID string) instanceDeleteOperationReceipt {
	t.Helper()
	machine, metadata := createInstanceDeleteMapper(instance.ID)
	published, err := db.DeleteWithReceiptContext(context.Background(), store, machine, metadata)
	if err != nil {
		t.Fatalf("publish instance delete: %v", err)
	}
	return instanceDeleteOperationReceipt{
		OperationID: operationID,
		Operation:   instanceLifecycleOperationDelete,
		ExpectedInvariant: instanceDeleteInvariant{
			Kind:       instanceDeleteInvariantAbsent,
			InstanceID: instance.ID,
		},
		EventID:            published.EventID,
		PublishedRootHash:  published.PublishedRootHash,
		AuthorSeq:          published.AuthorSeq,
		CommitHash:         published.CommitHash,
		CheckpointCommitID: published.CheckpointCommitID,
		CheckpointRootHash: published.CheckpointRootHash,
		Checkpointed:       published.Checkpointed,
	}
}

func TestCompleteInstanceDeleteReceiptCovered(t *testing.T) {
	receipt := testInstanceDeleteReceipt()
	tracker := &fakeInstanceDeleteReceiptTracker{observation: appliedInstanceDeleteObservation(receipt, swarmionapp.BranchEventContentCoverageCovered)}
	manager := &Manager{deleteReceiptTracker: tracker}
	var persisted []instanceDeleteOperationReceipt

	if err := manager.completeInstanceDeleteReceipt(context.Background(), receipt, captureInstanceDeleteReceipts(&persisted)); err != nil {
		t.Fatal(err)
	}
	if tracker.waitCalls != 1 || tracker.invariantCalls != 1 {
		t.Fatalf("tracker calls wait=%d invariant=%d, want 1/1", tracker.waitCalls, tracker.invariantCalls)
	}
	if len(persisted) != 1 {
		t.Fatalf("persisted receipts=%d, want 1", len(persisted))
	}
	got := persisted[0]
	if !got.AppliedDurably || !got.ContentDurable || got.ContentCoverage != swarmionapp.BranchEventContentCoverageCovered {
		t.Fatalf("persisted receipt=%+v, want applied covered content", got)
	}
	if got.CheckpointCommitID != "event-checkpoint-commit" || got.CheckpointRootHash != "event-checkpoint-root" ||
		got.DurableCheckpointCommitID != "durable-head-commit" || got.DurableCheckpointRootHash != "durable-head-root" {
		t.Fatalf("persisted checkpoint identity incomplete: %+v", got)
	}
}

func TestCompleteInstanceDeleteReceiptDissentIsProtocolComplete(t *testing.T) {
	receipt := testInstanceDeleteReceipt()
	tracker := &fakeInstanceDeleteReceiptTracker{observation: appliedInstanceDeleteObservation(receipt, swarmionapp.BranchEventContentCoverageDissent)}
	manager := &Manager{deleteReceiptTracker: tracker}
	var persisted []instanceDeleteOperationReceipt

	if err := manager.completeInstanceDeleteReceipt(context.Background(), receipt, captureInstanceDeleteReceipts(&persisted)); err != nil {
		t.Fatal(err)
	}
	if tracker.waitCalls != 1 || len(tracker.received) != 1 {
		t.Fatalf("receipt status calls=%d received=%d, want one exact receipt check", tracker.waitCalls, len(tracker.received))
	}
	if tracker.invariantCalls != 1 {
		t.Fatalf("content dissent performed %d durable invariant reads, want 1", tracker.invariantCalls)
	}
	if tracker.received[0].EventID != receipt.EventID || tracker.received[0].PublishedRootHash != receipt.PublishedRootHash {
		t.Fatalf("tracked receipt=%+v, want original exact identity", tracker.received[0])
	}
	if len(persisted) != 1 || !persisted[0].AppliedDurably || persisted[0].ContentDurable || persisted[0].ContentCoverage != swarmionapp.BranchEventContentCoverageDissent {
		t.Fatalf("persisted dissent receipt=%+v", persisted)
	}
	if persisted[0].Proof == nil || !persisted[0].Proof.ProofRan || persisted[0].Proof.Covered || !persisted[0].Proof.Conflict {
		t.Fatalf("dissent proof=%+v, want ran/non-cover/conflict", persisted[0].Proof)
	}
}

func TestCompleteInstanceDeleteReceiptParkedPersistsExactReceiptWithoutInvariantRead(t *testing.T) {
	receipt := testInstanceDeleteReceipt()
	observation := db.EventReceiptObservation{
		Receipt: receipt.publishedWriteReceipt(),
		Status: swarmionapp.ReceiptStatus{
			EventID:                   receipt.EventID,
			ExpectedPublishedRootHash: receipt.PublishedRootHash,
			Known:                     true,
			Parked:                    true,
			ParkedReason:              swarmionapp.BranchRootParkedReasonConflict,
			Revisitable:               true,
			ContentCoverage:           swarmionapp.BranchEventContentCoveragePending,
		},
		State: db.EventReceiptStateParkedConflict,
	}
	parkedErr := &db.EventReceiptParkedError{Observation: observation, Reason: "test parked delete"}
	tracker := &fakeInstanceDeleteReceiptTracker{observation: observation, waitErr: parkedErr}
	manager := &Manager{deleteReceiptTracker: tracker}
	var persisted []instanceDeleteOperationReceipt

	err := manager.completeInstanceDeleteReceipt(context.Background(), receipt, captureInstanceDeleteReceipts(&persisted))
	if !errors.Is(err, db.ErrEventReceiptParked) {
		t.Fatalf("error=%v, want parked receipt", err)
	}
	if tracker.waitCalls != 1 || tracker.invariantCalls != 0 {
		t.Fatalf("parked calls wait=%d invariant=%d, want 1/0", tracker.waitCalls, tracker.invariantCalls)
	}
	if len(persisted) != 1 || persisted[0].EventID != receipt.EventID || persisted[0].PublishedRootHash != receipt.PublishedRootHash {
		t.Fatalf("parked receipt was not retained exactly: %+v", persisted)
	}
}

func TestCompleteInstanceDeleteReceiptCompetingRecreationIsConflict(t *testing.T) {
	receipt := testInstanceDeleteReceipt()
	tracker := &fakeInstanceDeleteReceiptTracker{
		observation:    appliedInstanceDeleteObservation(receipt, swarmionapp.BranchEventContentCoverageDissent),
		instanceExists: true,
	}
	manager := &Manager{deleteReceiptTracker: tracker}
	var persisted []instanceDeleteOperationReceipt

	err := manager.completeInstanceDeleteReceipt(context.Background(), receipt, captureInstanceDeleteReceipts(&persisted))
	if !errors.Is(err, ErrInstanceDeleteInvariantConflict) {
		t.Fatalf("error=%v, want %v", err, ErrInstanceDeleteInvariantConflict)
	}
	if tracker.invariantCalls != 1 || len(tracker.checkpointIDs) != 1 || tracker.checkpointIDs[0] != "durable-head-commit" {
		t.Fatalf("durable invariant reads=%d checkpoints=%v", tracker.invariantCalls, tracker.checkpointIDs)
	}
	if tracker.waitCalls != 1 || len(tracker.received) != 1 {
		t.Fatalf("receipt was unexpectedly reissued: calls=%d receipts=%v", tracker.waitCalls, tracker.received)
	}
}

func TestCompleteInstanceDeleteReceiptPendingReturnsBoundedDiagnostic(t *testing.T) {
	receipt := testInstanceDeleteReceipt()
	observation := db.EventReceiptObservation{
		Receipt: db.PublishedWriteReceipt{
			Committed:          true,
			Checkpointed:       true,
			EventID:            receipt.EventID,
			PublishedRootHash:  receipt.PublishedRootHash,
			CheckpointCommitID: "pending-checkpoint-commit",
			CheckpointRootHash: "pending-checkpoint-root",
		},
		Status: swarmionapp.ReceiptStatus{
			EventID:                   receipt.EventID,
			ExpectedPublishedRootHash: receipt.PublishedRootHash,
			Known:                     true,
			Checkpointed:              true,
			ContentCoverage:           swarmionapp.BranchEventContentCoveragePending,
		},
		State: db.EventReceiptStatePending,
	}
	pendingErr := &db.EventReceiptPendingError{Observation: observation, Reason: "test pending budget", Cause: context.DeadlineExceeded}
	tracker := &fakeInstanceDeleteReceiptTracker{observation: observation, waitErr: pendingErr}
	manager := &Manager{deleteReceiptTracker: tracker}
	var persisted []instanceDeleteOperationReceipt

	err := manager.completeInstanceDeleteReceipt(context.Background(), receipt, captureInstanceDeleteReceipts(&persisted))
	if !errors.Is(err, db.ErrEventReceiptPending) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error=%v, want pending deadline", err)
	}
	if !strings.Contains(err.Error(), receipt.EventID) || !strings.Contains(err.Error(), receipt.PublishedRootHash) {
		t.Fatalf("pending error lacks exact receipt diagnostic: %v", err)
	}
	if tracker.invariantCalls != 0 {
		t.Fatalf("pending work performed %d invariant reads, want 0", tracker.invariantCalls)
	}
	if len(persisted) != 1 || persisted[0].CheckpointCommitID != "pending-checkpoint-commit" || persisted[0].AppliedDurably {
		t.Fatalf("pending receipt checkpoint was not preserved: %+v", persisted)
	}
}

func TestValidateInstanceDeleteOperationReceiptRequiresRecoveryProvenance(t *testing.T) {
	receipt := testInstanceDeleteReceipt()
	identity := instanceDeleteOperationIdentity{
		Key:          strings.Repeat("1", 64),
		IntentDigest: strings.Repeat("2", 64),
		AuthorPeerID: "receipt-author",
		ExpectedInvariant: instanceDeleteInvariant{
			Kind:       instanceDeleteInvariantAbsent,
			InstanceID: receipt.ExpectedInvariant.InstanceID,
		},
	}
	receipt.ExpectedInvariant = identity.ExpectedInvariant
	receipt.OperationIntentDigest = identity.IntentDigest
	receipt.OperationAuthorPeerID = identity.AuthorPeerID
	if err := validateInstanceDeleteOperationReceipt(receipt, identity, receipt.OperationID, receipt.ExpectedInvariant.InstanceID); err != nil {
		t.Fatalf("validate complete receipt provenance: %v", err)
	}

	withoutIntent := receipt
	withoutIntent.OperationIntentDigest = ""
	if err := validateInstanceDeleteOperationReceipt(withoutIntent, identity, receipt.OperationID, receipt.ExpectedInvariant.InstanceID); err == nil || !strings.Contains(err.Error(), "intent digest") {
		t.Fatalf("missing intent provenance error=%v", err)
	}

	withoutAuthor := receipt
	withoutAuthor.OperationAuthorPeerID = ""
	if err := validateInstanceDeleteOperationReceipt(withoutAuthor, identity, receipt.OperationID, receipt.ExpectedInvariant.InstanceID); err == nil || !strings.Contains(err.Error(), "author") {
		t.Fatalf("missing author provenance error=%v", err)
	}
}

func TestDeleteInstanceRecordsReturnsOperationKeyConflictWithoutRetry(t *testing.T) {
	store := openProvisionerTestDB(t)
	instance := InstanceInfo{
		ID:            db.MustNewUUIDv7(),
		Name:          "operation-conflict-delete",
		Kind:          KindLocalVM,
		KindID:        "operation-conflict-provider",
		DesiredStatus: ServerStateRunning,
		Location:      "test",
	}
	insertInstanceForDeleteReceiptTest(t, store, &instance)

	operationID := db.MustNewUUIDv7()
	identity := instanceDeleteOperationIdentityForTest(t, store, operationID, instance, true)
	conflicting, err := db.NewPublishedWriteOperation(identity.Key, "different immutable delete intent")
	if err != nil {
		t.Fatal(err)
	}
	conflicting.AuthorPeerID = identity.AuthorPeerID
	machine, metadata := createInstanceDeleteMapper(instance.ID)
	accepted, err := db.DeleteWithOperationReceiptContext(context.Background(), store, conflicting, machine, metadata)
	if err != nil {
		t.Fatalf("bind conflicting operation key: %v", err)
	}
	waitForTestPublishedEvent(t, store, accepted, "checkpoint conflicting operation key")
	before, _ := store.SwarmionStatus()

	manager := &Manager{db: store}
	persistCalls := 0
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err = manager.deleteInstanceRecords(ctx, operationID, identity, instance, func(instanceDeleteOperationReceipt, int, string) error {
		persistCalls++
		return nil
	})
	if !errors.Is(err, swarmionprotocol.ErrOperationKeyConflict) {
		t.Fatalf("delete error=%v, want operation key conflict", err)
	}
	if elapsed := time.Since(started); elapsed >= time.Second {
		t.Fatalf("operation conflict was retried for %s instead of failing promptly", elapsed)
	}
	if persistCalls != 0 {
		t.Fatalf("operation conflict persisted %d receipts, want none", persistCalls)
	}
	after, _ := store.SwarmionStatus()
	if after.ClockEvents != before.ClockEvents || after.CheckpointEventCount != before.CheckpointEventCount {
		t.Fatalf("operation conflict published another event: clock %d->%d checkpoint %d->%d", before.ClockEvents, after.ClockEvents, before.CheckpointEventCount, after.CheckpointEventCount)
	}
}

func TestInstanceDeleteReceiptReachesAppliedDurablyWithContentCovered(t *testing.T) {
	store := openProvisionerTestDB(t)
	instance := InstanceInfo{
		ID:            db.MustNewUUIDv7(),
		Name:          "covered-delete",
		Kind:          KindCloudVM,
		KindID:        "covered-provider",
		DesiredStatus: ServerStateRunning,
		Location:      "test",
	}
	insertInstanceForDeleteReceiptTest(t, store, &instance)
	receipt := publishInstanceDeleteForTest(t, store, instance, "covered-delete-operation")
	manager := &Manager{db: store}
	var persisted []instanceDeleteOperationReceipt

	if err := manager.completeInstanceDeleteReceipt(context.Background(), receipt, captureInstanceDeleteReceipts(&persisted)); err != nil {
		t.Fatal(err)
	}
	if len(persisted) != 1 || !persisted[0].AppliedDurably || !persisted[0].ContentDurable ||
		persisted[0].ContentCoverage != swarmionapp.BranchEventContentCoverageCovered {
		t.Fatalf("covered delete receipt=%+v", persisted)
	}
}

func TestInstanceDeleteReceiptContentDissentStillChecksDurableInvariant(t *testing.T) {
	store := openProvisionerTestDB(t)
	target := InstanceInfo{
		ID:            db.MustNewUUIDv7(),
		Name:          "dissent-target",
		Kind:          KindCloudVM,
		KindID:        "test-provider",
		DesiredStatus: ServerStateRunning,
		Location:      "test",
	}
	unrelated := InstanceInfo{
		ID:            db.MustNewUUIDv7(),
		Name:          "dissent-unrelated",
		Kind:          KindCloudVM,
		KindID:        "test-provider",
		DesiredStatus: ServerStateRunning,
		Location:      "test",
	}
	insertInstanceForDeleteReceiptTest(t, store, &target)
	insertInstanceForDeleteReceiptTest(t, store, &unrelated)
	receipt := publishInstanceDeleteForTest(t, store, target, "dissent-delete-operation")
	waitForTestPublishedEvent(t, store, receipt.publishedWriteReceipt(), "materialize target delete before unrelated write")

	unrelated.DesiredStatus = ServerStateStopped
	unrelatedMachine, _ := createInstanceUpdateMapper(unrelated)
	updateReceipt, err := db.UpdateWithReceiptContext(context.Background(), store, unrelatedMachine)
	if err != nil {
		t.Fatal(err)
	}
	waitForTestPublishedEvent(t, store, updateReceipt, "advance durable head after target delete")
	before, _ := store.SwarmionStatus()
	manager := &Manager{db: store}
	var persisted []instanceDeleteOperationReceipt

	if err := manager.completeInstanceDeleteReceipt(context.Background(), receipt, captureInstanceDeleteReceipts(&persisted)); err != nil {
		t.Fatal(err)
	}
	after, _ := store.SwarmionStatus()
	if len(persisted) != 1 || !persisted[0].AppliedDurably || persisted[0].ContentDurable ||
		persisted[0].ContentCoverage != swarmionapp.BranchEventContentCoverageDissent {
		t.Fatalf("dissent delete receipt=%+v", persisted)
	}
	if after.ClockEvents != before.ClockEvents || after.CheckpointEventCount != before.CheckpointEventCount {
		t.Fatalf("dissent tracking published or checkpointed another event: clock %d->%d checkpoint %d->%d", before.ClockEvents, after.ClockEvents, before.CheckpointEventCount, after.CheckpointEventCount)
	}
}

func TestInstanceDeleteReceiptLaterRecreationReturnsDurableConflict(t *testing.T) {
	store := openProvisionerTestDB(t)
	instance := InstanceInfo{
		ID:            db.MustNewUUIDv7(),
		Name:          "recreated-target",
		Kind:          KindCloudVM,
		KindID:        "test-provider",
		DesiredStatus: ServerStateRunning,
		Location:      "test",
	}
	insertInstanceForDeleteReceiptTest(t, store, &instance)
	receipt := publishInstanceDeleteForTest(t, store, instance, "recreated-delete-operation")
	waitForTestPublishedEvent(t, store, receipt.publishedWriteReceipt(), "materialize target delete before recreation")

	instance.Name = "later-recreation"
	insertInstanceForDeleteReceiptTest(t, store, &instance)
	before, _ := store.SwarmionStatus()
	manager := &Manager{db: store}
	var persisted []instanceDeleteOperationReceipt
	err := manager.completeInstanceDeleteReceipt(context.Background(), receipt, captureInstanceDeleteReceipts(&persisted))
	if !errors.Is(err, ErrInstanceDeleteInvariantConflict) {
		t.Fatalf("error=%v, want %v", err, ErrInstanceDeleteInvariantConflict)
	}
	after, _ := store.SwarmionStatus()
	if len(persisted) != 1 || !persisted[0].AppliedDurably {
		t.Fatalf("recreated delete receipt=%+v", persisted)
	}
	if after.ClockEvents != before.ClockEvents || after.CheckpointEventCount != before.CheckpointEventCount {
		t.Fatalf("recreation conflict republished delete: clock %d->%d checkpoint %d->%d", before.ClockEvents, after.ClockEvents, before.CheckpointEventCount, after.CheckpointEventCount)
	}
}

func TestInstanceDeleteTaskResumesPersistedReceiptBeforeInstanceLookup(t *testing.T) {
	store := openProvisionerTestDB(t)
	receipt := testInstanceDeleteReceipt()
	taskID := db.MustNewUUIDv7()
	receipt.OperationID = taskID
	identity := instanceDeleteOperationIdentityForTest(t, store, taskID, InstanceInfo{ID: receipt.ExpectedInvariant.InstanceID}, true)
	receipt.ExpectedInvariant = identity.ExpectedInvariant
	receipt.OperationIntentDigest = identity.IntentDigest
	receipt.OperationAuthorPeerID = identity.AuthorPeerID
	tracker := &fakeInstanceDeleteReceiptTracker{observation: appliedInstanceDeleteObservation(receipt, swarmionapp.BranchEventContentCoverageDissent)}
	manager := newLifecycleTestManager(t, store, newProvisionerRegistry())
	manager.deleteReceiptTracker = tracker
	legacyFact := legacyInstanceDeleteReceiptFactForTest(t, taskID, identity, receipt)
	if err := manager.tasks.EnsureOperationFact(context.Background(), legacyFact); err != nil {
		t.Fatalf("publish legacy receipt fact: %v", err)
	}

	record, err := tasks.Enqueue(manager.tasks, tasks.EnqueueOptions[instanceLifecycleTaskPayload]{
		ID:          taskID,
		Stream:      InstanceLifecycleTaskStream,
		SubjectType: taskSubjectInstance,
		SubjectID:   instanceLifecycleSubjectID(receipt.ExpectedInvariant.InstanceID, instanceLifecycleOperationDelete),
		Title:       "resume delete receipt",
		Payload: instanceLifecycleTaskPayload{
			InstanceID:          receipt.ExpectedInvariant.InstanceID,
			InstanceName:        "already-deleted",
			Operation:           instanceLifecycleOperationDelete,
			DesiredStatus:       ServerStateDeleting,
			LocalOnly:           true,
			OperationStateModel: instanceDeleteOperationFactsV1,
			DeleteOperation:     &identity,
			DeleteReceipt:       cloneInstanceDeleteOperationReceipt(&receipt),
		},
		MaxAttempts: instanceDeleteMaxAttempts,
	})
	if err != nil {
		t.Fatal(err)
	}
	// There is deliberately no instance row. A recovery path that starts the
	// delete again would fail its initial lookup; receipt-first recovery succeeds.
	if err := manager.tasks.RunPending(context.Background()); err != nil {
		t.Fatal(err)
	}

	done, err := manager.tasks.Get(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != tasks.StatusSucceeded || done.Attempts != 1 {
		t.Fatalf("recovered task status=%s attempts=%d", done.Status, done.Attempts)
	}
	var result instanceLifecycleTaskResult
	if err := json.Unmarshal(done.Result, &result); err != nil {
		t.Fatal(err)
	}
	if result.DeleteReceipt == nil || result.DeleteReceipt.EventID != receipt.EventID || result.DeleteReceipt.PublishedRootHash != receipt.PublishedRootHash {
		t.Fatalf("recovered result lost original receipt: %+v", result.DeleteReceipt)
	}
	if tracker.waitCalls != 1 || len(tracker.received) != 1 || tracker.received[0].EventID != receipt.EventID {
		t.Fatalf("recovery did not resume exactly one receipt: calls=%d receipts=%+v", tracker.waitCalls, tracker.received)
	}
	var finalPayload instanceLifecycleTaskPayload
	if err := json.Unmarshal(done.Payload, &finalPayload); err != nil {
		t.Fatal(err)
	}
	if finalPayload.DeleteReceipt == nil || !finalPayload.DeleteReceipt.AppliedDurably ||
		finalPayload.DeleteReceipt.EventID != receipt.EventID ||
		finalPayload.DeleteReceipt.PublishedRootHash != receipt.PublishedRootHash {
		t.Fatalf("task payload did not retain the applied receipt: %+v", finalPayload.DeleteReceipt)
	}
}

func TestInstanceDeleteInvariantConflictIsPermanentTaskOutcome(t *testing.T) {
	store := openProvisionerTestDB(t)
	taskID := db.MustNewUUIDv7()
	instance := InstanceInfo{ID: db.MustNewUUIDv7(), Name: "recreated-after-delete"}
	identity := instanceDeleteOperationIdentityForTest(t, store, taskID, instance, true)
	receipt := testInstanceDeleteReceipt()
	receipt.OperationID = taskID
	receipt.ExpectedInvariant = identity.ExpectedInvariant
	receipt.OperationIntentDigest = identity.IntentDigest
	receipt.OperationAuthorPeerID = identity.AuthorPeerID
	tracker := &fakeInstanceDeleteReceiptTracker{
		observation:    appliedInstanceDeleteObservation(receipt, swarmionapp.BranchEventContentCoverageDissent),
		instanceExists: true,
	}
	manager := newLifecycleTestManager(t, store, newProvisionerRegistry())
	manager.deleteReceiptTracker = tracker

	before, err := store.LookupPublishedWriteOperation(context.Background(), identity.publishedWriteOperation())
	if err != nil {
		t.Fatal(err)
	}
	if before.State != swarmionapp.OperationResolvedAbsent {
		t.Fatalf("delete operation unexpectedly existed before task run: %+v", before)
	}
	record, err := tasks.Enqueue(manager.tasks, tasks.EnqueueOptions[instanceLifecycleTaskPayload]{
		ID:          taskID,
		Stream:      InstanceLifecycleTaskStream,
		SubjectType: taskSubjectInstance,
		SubjectID:   instanceLifecycleSubjectID(instance.ID, instanceLifecycleOperationDelete),
		Title:       "conflicting recovered delete",
		Payload: instanceLifecycleTaskPayload{
			InstanceID:          instance.ID,
			InstanceName:        instance.Name,
			Operation:           instanceLifecycleOperationDelete,
			DesiredStatus:       ServerStateDeleting,
			LocalOnly:           true,
			OperationStateModel: instanceDeleteOperationFactsV1,
			DeleteOperation:     &identity,
			DeleteReceipt:       cloneInstanceDeleteOperationReceipt(&receipt),
		},
		MaxAttempts: instanceDeleteMaxAttempts,
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < instanceDeleteMaxAttempts; i++ {
		_ = manager.tasks.RunPending(context.Background())
	}

	done, err := manager.tasks.Get(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != tasks.StatusFailed || done.Attempts != 1 {
		t.Fatalf("invariant-conflict task status=%s attempts=%d, want failed/1", done.Status, done.Attempts)
	}
	if !strings.Contains(done.ErrorMessage, ErrInstanceDeleteInvariantConflict.Error()) {
		t.Fatalf("task error=%q, want invariant conflict", done.ErrorMessage)
	}
	if tracker.waitCalls != 1 || tracker.invariantCalls != 1 || len(tracker.received) != 1 {
		t.Fatalf(
			"permanent conflict receipt calls wait=%d invariant=%d receipts=%d, want 1/1/1",
			tracker.waitCalls,
			tracker.invariantCalls,
			len(tracker.received),
		)
	}
	if tracker.received[0].EventID != receipt.EventID || tracker.received[0].PublishedRootHash != receipt.PublishedRootHash {
		t.Fatalf("permanent conflict changed exact receipt: got=%+v want=%+v", tracker.received[0], receipt)
	}
	after, err := store.LookupPublishedWriteOperation(context.Background(), identity.publishedWriteOperation())
	if err != nil {
		t.Fatal(err)
	}
	if after.State != swarmionapp.OperationResolvedAbsent {
		t.Fatalf("permanent conflict published a duplicate delete operation: %+v", after)
	}
}

func TestForeignParkedDeleteRecoveryReobservesWithoutTaskWriteOrRepublish(t *testing.T) {
	store := openProvisionerTestDB(t)
	manager := newLifecycleTestManager(t, store, newProvisionerRegistry())
	taskID := db.MustNewUUIDv7()
	instance := InstanceInfo{ID: db.MustNewUUIDv7(), Name: "foreign-parked-delete"}
	identity := instanceDeleteOperationIdentityForTest(t, store, taskID, instance, true)
	identity.AuthorPeerID = "foreign-delete-author"
	eventID := strings.Repeat("a", 64)
	publishedRoot := strings.Repeat("b", 32)
	lookupCalls := 0
	manager.lookupDeleteRecoveryOperation = func(ctx context.Context, operation db.PublishedWriteOperation) (swarmionapp.OperationResolution, error) {
		lookupCalls++
		if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > instanceDeleteRecoveryObserveLimit+time.Second {
			t.Fatalf("foreign recovery lookup did not receive its bounded context")
		}
		if operation != identity.publishedWriteOperation() {
			t.Fatalf("foreign recovery operation=%+v, want %+v", operation, identity.publishedWriteOperation())
		}
		return acceptedOperationResolutionForTest(operation, eventID, publishedRoot), nil
	}
	parkedStates := []struct {
		state  db.EventReceiptState
		reason string
	}{
		{state: db.EventReceiptStateParkedConflict, reason: swarmionapp.BranchRootParkedReasonConflict},
		{state: db.EventReceiptStateDependencyParked, reason: swarmionapp.BranchRootParkedReasonDependencyParked},
		{state: db.EventReceiptStateStaleAnchor, reason: swarmionapp.BranchRootParkedReasonStaleAnchor},
	}
	observeCalls := 0
	var observedReceipts []db.PublishedWriteReceipt
	manager.observeDeleteRecoveryReceipt = func(ctx context.Context, receipt db.PublishedWriteReceipt) (db.EventReceiptObservation, error) {
		if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > instanceDeleteRecoveryObserveLimit+time.Second {
			t.Fatalf("foreign recovery observation did not receive its bounded context")
		}
		if observeCalls >= len(parkedStates) {
			t.Fatalf("unexpected receipt observation %d", observeCalls+1)
		}
		parked := parkedStates[observeCalls]
		observeCalls++
		observedReceipts = append(observedReceipts, receipt)
		return db.EventReceiptObservation{
			Receipt: receipt,
			Status: swarmionapp.ReceiptStatus{
				EventID:                   receipt.EventID,
				ExpectedPublishedRootHash: receipt.PublishedRootHash,
				Known:                     true,
				Parked:                    true,
				ParkedReason:              parked.reason,
				Revisitable:               true,
				ContentCoverage:           swarmionapp.BranchEventContentCoveragePending,
			},
			State: parked.state,
		}, nil
	}

	record, err := tasks.Enqueue(manager.tasks, tasks.EnqueueOptions[instanceLifecycleTaskPayload]{
		ID:          taskID,
		Stream:      InstanceLifecycleTaskStream,
		SubjectType: taskSubjectInstance,
		SubjectID:   instanceLifecycleSubjectID(instance.ID, instanceLifecycleOperationDelete),
		Title:       "foreign parked delete",
		Payload: instanceLifecycleTaskPayload{
			InstanceID:          instance.ID,
			InstanceName:        instance.Name,
			Operation:           instanceLifecycleOperationDelete,
			DesiredStatus:       ServerStateDeleting,
			LocalOnly:           true,
			OperationStateModel: instanceDeleteOperationFactsV1,
			DeleteOperation:     &identity,
		},
		MaxAttempts: instanceDeleteMaxAttempts,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.tasks.Update(record.ID, tasks.StatusRunning, 90, "interrupted after foreign delete publication", nil); err != nil {
		t.Fatal(err)
	}
	runningBefore, err := manager.tasks.Get(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore, err := manager.tasks.Events(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	metricsBefore := store.TransactionMetrics()
	statusBefore, ok := store.RuntimeSnapshot()
	if !ok {
		t.Fatal("read Swarmion runtime snapshot before parked recovery")
	}

	for index := range parkedStates {
		recovered, recoverErr := manager.tasks.RecoverOwnedRunning()
		if recoverErr != nil {
			t.Fatalf("parked recovery %d: %v", index+1, recoverErr)
		}
		if recovered != 0 {
			t.Fatalf("parked recovery %d changed %d task(s), want 0", index+1, recovered)
		}
	}
	if lookupCalls != len(parkedStates) || observeCalls != len(parkedStates) {
		t.Fatalf("parked recovery calls lookup=%d observe=%d, want %d/%d", lookupCalls, observeCalls, len(parkedStates), len(parkedStates))
	}
	for _, receipt := range observedReceipts {
		if receipt.EventID != eventID || receipt.PublishedRootHash != publishedRoot || receipt.AuthorPeerID != identity.AuthorPeerID {
			t.Fatalf("parked recovery changed exact foreign receipt: %+v", receipt)
		}
	}
	if metricsAfter := store.TransactionMetrics(); metricsAfter != metricsBefore {
		t.Fatalf("parked foreign recovery wrote a transaction: before=%+v after=%+v", metricsBefore, metricsAfter)
	}
	statusAfter, ok := store.RuntimeSnapshot()
	if !ok {
		t.Fatal("read Swarmion runtime snapshot after parked recovery")
	}
	// Clock size and checkpoint event count are moving scheduler state: prior
	// task writes may checkpoint and retire while this recovery runs. The local
	// author sequence is the stable proof that recovery did not republish.
	if statusAfter.LocalAuthorSeq != statusBefore.LocalAuthorSeq {
		t.Fatalf(
			"parked foreign recovery republished work: local author sequence=%d->%d",
			statusBefore.LocalAuthorSeq,
			statusAfter.LocalAuthorSeq,
		)
	}
	runningAfter, err := manager.tasks.Get(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if runningAfter.Status != tasks.StatusRunning || runningAfter.Attempts != runningBefore.Attempts ||
		!runningAfter.UpdatedAt.Equal(runningBefore.UpdatedAt) || string(runningAfter.Payload) != string(runningBefore.Payload) {
		t.Fatalf("parked foreign recovery changed task: before=%+v after=%+v", runningBefore, runningAfter)
	}
	eventsAfter, err := manager.tasks.Events(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(eventsAfter) != len(eventsBefore) {
		t.Fatalf("parked foreign recovery inserted task events: before=%d after=%d", len(eventsBefore), len(eventsAfter))
	}
}

func TestOperationFactRecoveryPublishesV2WithoutDowngradeUnsafePayload(t *testing.T) {
	store := openProvisionerTestDB(t)
	manager := newLifecycleTestManager(t, store, newProvisionerRegistry())
	taskID := db.MustNewUUIDv7()
	instance := InstanceInfo{ID: db.MustNewUUIDv7(), Name: "immutable-delete-recovery"}
	identity := instanceDeleteOperationIdentityForTest(t, store, taskID, instance, true)
	identity.AuthorPeerID = "foreign-delete-author-a"

	effectFact, err := newInstanceDeleteEffectFact(taskID, identity)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.tasks.EnsureOperationFact(context.Background(), effectFact); err != nil {
		t.Fatal(err)
	}

	accepted := testInstanceDeleteReceipt().publishedWriteReceipt()
	accepted.AuthorPeerID = identity.AuthorPeerID
	accepted.AuthorSeq = 7
	accepted.OperationIntentDigest = identity.IntentDigest
	deleteLookups := 0
	manager.lookupDeleteRecoveryOperation = func(_ context.Context, operation db.PublishedWriteOperation) (swarmionapp.OperationResolution, error) {
		deleteLookups++
		if operation != identity.publishedWriteOperation() {
			t.Fatalf("operation lookup=%+v, want %+v", operation, identity.publishedWriteOperation())
		}
		return acceptedOperationResolutionForTest(operation, accepted.EventID, accepted.PublishedRootHash), nil
	}
	manager.observeDeleteRecoveryReceipt = func(_ context.Context, published db.PublishedWriteReceipt) (db.EventReceiptObservation, error) {
		return db.EventReceiptObservation{
			Receipt: published,
			State:   db.EventReceiptStateAppliedDurably,
			Status: swarmionapp.ReceiptStatus{
				EventID:                   published.EventID,
				ExpectedPublishedRootHash: published.PublishedRootHash,
				Known:                     true,
				Checkpointed:              true,
				AppliedDurably:            true,
				CheckpointCommitID:        "delete-checkpoint",
				CheckpointRootHash:        "delete-checkpoint-root",
				DurableCheckpointCommitID: "delete-durable-head",
				DurableCheckpointRootHash: "delete-durable-root",
				QueryableRootHash:         "delete-queryable-root",
				ContentCoverage:           swarmionapp.BranchEventContentCoverageCovered,
			},
		}, nil
	}

	record, err := tasks.Enqueue(manager.tasks, tasks.EnqueueOptions[instanceLifecycleTaskPayload]{
		ID:          taskID,
		Stream:      InstanceLifecycleTaskStream,
		SubjectType: taskSubjectInstance,
		SubjectID:   instanceLifecycleSubjectID(instance.ID, instanceLifecycleOperationDelete),
		Title:       "immutable delete recovery",
		Payload: instanceLifecycleTaskPayload{
			InstanceID:          instance.ID,
			InstanceName:        instance.Name,
			Operation:           instanceLifecycleOperationDelete,
			DesiredStatus:       ServerStateDeleting,
			LocalOnly:           true,
			OperationStateModel: instanceDeleteOperationFactsV1,
			DeleteOperation:     &identity,
		},
		MaxAttempts: instanceDeleteMaxAttempts,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.tasks.Update(record.ID, tasks.StatusRunning, 90, "deleting instance records", nil); err != nil {
		t.Fatal(err)
	}
	if recovered, err := manager.tasks.RecoverOwnedRunning(); err != nil || recovered != 1 {
		t.Fatalf("recover immutable delete task=%d error=%v, want 1/nil", recovered, err)
	}
	if deleteLookups != 1 {
		t.Fatalf("delete operation lookups=%d, want one", deleteLookups)
	}

	recoveredRecord, err := manager.tasks.Get(taskID)
	if err != nil {
		t.Fatal(err)
	}
	recoveredPayload := decodeDeleteRestartPayload(t, recoveredRecord)
	if recoveredRecord.Status != tasks.StatusPending || recoveredPayload.DeleteReceipt != nil {
		t.Fatalf("recovered record=%+v payload=%+v, want pending without a zero-valued legacy receipt", recoveredRecord, recoveredPayload)
	}
	receiptFact, found, err := manager.tasks.OperationFact(context.Background(), taskID, tasks.OperationFactKindReceiptV2)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("recovery did not publish the immutable exact receipt fact")
	}
	factReceipt, err := instanceDeleteReceiptFromFact(receiptFact, identity)
	if err != nil {
		t.Fatal(err)
	}
	if factReceipt.EventID != accepted.EventID || factReceipt.PublishedRootHash != accepted.PublishedRootHash {
		t.Fatalf("receipt fact=%+v, want exact recovered receipt %s/%s", factReceipt, accepted.EventID, accepted.PublishedRootHash)
	}
	if factReceipt.EventDigest != "" || factReceipt.AuthorSeq != 0 {
		t.Fatalf("v2 receipt fact unexpectedly contains legacy-only metadata: %+v", factReceipt)
	}
	if _, legacyFound, err := manager.tasks.OperationFact(context.Background(), taskID, tasks.OperationFactKindReceipt); err != nil {
		t.Fatal(err)
	} else if legacyFound {
		t.Fatal("new recovery wrote the legacy receipt fact kind")
	}
}

func TestUnsupportedInstanceDeletePayloadFailsClosedBeforeImperativeWork(t *testing.T) {
	tests := []struct {
		name          string
		stateModel    string
		withOperation bool
		wantError     string
	}{
		{
			name:          "missing state model",
			withOperation: true,
			wantError:     "unsupported operation state model",
		},
		{
			name:          "retired mutable checkpoint model",
			stateModel:    "mutable_task_checkpoint_v1",
			withOperation: true,
			wantError:     "unsupported operation state model",
		},
		{
			name:       "missing immutable operation identity",
			stateModel: instanceDeleteOperationFactsV1,
			wantError:  "missing its immutable operation identity",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := openProvisionerTestDB(t)
			manager := newLifecycleTestManager(t, store, newProvisionerRegistry())
			taskID := db.MustNewUUIDv7()
			instanceID := db.MustNewUUIDv7()
			var identity *instanceDeleteOperationIdentity
			if tt.withOperation {
				value := instanceDeleteOperationIdentityForTest(t, store, taskID, InstanceInfo{ID: instanceID}, true)
				identity = &value
			}
			publishCalls := 0
			manager.publishDeleteOperation = func(context.Context, db.PublishedWriteOperation, InstanceInfo) (db.PublishedWriteReceipt, error) {
				publishCalls++
				return db.PublishedWriteReceipt{}, errors.New("delete publication must not run")
			}
			record, err := tasks.Enqueue(manager.tasks, tasks.EnqueueOptions[instanceLifecycleTaskPayload]{
				ID:          taskID,
				Stream:      InstanceLifecycleTaskStream,
				SubjectType: taskSubjectInstance,
				SubjectID:   instanceLifecycleSubjectID(instanceID, instanceLifecycleOperationDelete),
				Title:       "unsupported delete payload",
				Payload: instanceLifecycleTaskPayload{
					InstanceID:          instanceID,
					Operation:           instanceLifecycleOperationDelete,
					LocalOnly:           true,
					OperationStateModel: tt.stateModel,
					DeleteOperation:     identity,
				},
				MaxAttempts: instanceDeleteMaxAttempts,
			})
			if err != nil {
				t.Fatal(err)
			}
			runErr := manager.tasks.RunPending(context.Background())
			if runErr == nil || !strings.Contains(runErr.Error(), tt.wantError) {
				t.Fatalf("run error=%v, want %q", runErr, tt.wantError)
			}
			done, err := manager.tasks.Get(record.ID)
			if err != nil {
				t.Fatal(err)
			}
			if done.Status != tasks.StatusFailed || done.Attempts != 1 || publishCalls != 0 {
				t.Fatalf("unsupported delete status=%s attempts=%d publications=%d, want failed/1/0", done.Status, done.Attempts, publishCalls)
			}
		})
	}
}

func TestUnsupportedRunningInstanceDeleteFailsRecoveryWithoutWriteOrReplay(t *testing.T) {
	store := openProvisionerTestDB(t)
	manager := newLifecycleTestManager(t, store, newProvisionerRegistry())
	taskID := db.MustNewUUIDv7()
	instanceID := db.MustNewUUIDv7()
	identity := instanceDeleteOperationIdentityForTest(t, store, taskID, InstanceInfo{ID: instanceID}, true)
	publishCalls := 0
	manager.publishDeleteOperation = func(context.Context, db.PublishedWriteOperation, InstanceInfo) (db.PublishedWriteReceipt, error) {
		publishCalls++
		return db.PublishedWriteReceipt{}, errors.New("delete publication must not run")
	}
	record, err := tasks.Enqueue(manager.tasks, tasks.EnqueueOptions[instanceLifecycleTaskPayload]{
		ID:          taskID,
		Stream:      InstanceLifecycleTaskStream,
		SubjectType: taskSubjectInstance,
		SubjectID:   instanceLifecycleSubjectID(instanceID, instanceLifecycleOperationDelete),
		Title:       "retired delete recovery payload",
		Payload: instanceLifecycleTaskPayload{
			InstanceID:          instanceID,
			Operation:           instanceLifecycleOperationDelete,
			LocalOnly:           true,
			OperationStateModel: "mutable_task_checkpoint_v1",
			DeleteOperation:     &identity,
		},
		MaxAttempts: instanceDeleteMaxAttempts,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.tasks.Update(record.ID, tasks.StatusRunning, 90, "interrupted legacy delete", nil); err != nil {
		t.Fatal(err)
	}
	before, err := manager.tasks.Get(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore, err := manager.tasks.Events(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	metricsBefore := store.TransactionMetrics()
	recovered, recoverErr := manager.tasks.RecoverOwnedRunning()
	if recovered != 0 || recoverErr == nil || !tasks.IsPermanent(recoverErr) ||
		!strings.Contains(recoverErr.Error(), "unsupported operation state model") {
		t.Fatalf("legacy recovery=%d error=%v, want permanent fail-closed error", recovered, recoverErr)
	}
	after, err := manager.tasks.Get(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Status != before.Status || !after.UpdatedAt.Equal(before.UpdatedAt) || string(after.Payload) != string(before.Payload) {
		t.Fatalf("unsupported recovery changed task: before=%+v after=%+v", before, after)
	}
	if metricsAfter := store.TransactionMetrics(); metricsAfter != metricsBefore {
		t.Fatalf("unsupported recovery wrote SQL: before=%+v after=%+v", metricsBefore, metricsAfter)
	}
	eventsAfter, err := manager.tasks.Events(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(eventsAfter) != len(eventsBefore) || publishCalls != 0 {
		t.Fatalf("unsupported recovery events=%d->%d publications=%d, want unchanged/0", len(eventsBefore), len(eventsAfter), publishCalls)
	}
}

func TestInstanceDeleteUnsafeNoReceiptPublicationDoesNotReplayTask(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "inconclusive publication without receipt",
			err:  fmt.Errorf("delete publication became inconclusive: %w", swarmionapp.ErrPublicationInconclusive),
			want: swarmionapp.ErrPublicationInconclusive,
		},
		{
			name: "ambiguous apply rollback",
			err:  errors.New("ambiguous instance delete apply/rollback response"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := openProvisionerTestDB(t)
			manager := newLifecycleTestManager(t, store, newProvisionerRegistry())
			instance := InstanceInfo{
				ID:                  db.MustNewUUIDv7(),
				Name:                "unsafe-delete-" + strings.ReplaceAll(tt.name, " ", "-"),
				Kind:                "receipt_test_record",
				KindID:              "unsafe-delete-test",
				ProviderResourceID:  "unsafe-delete-provider-id",
				DesiredStatus:       ServerStateRunning,
				ReplicationPriority: 1,
				Location:            "test",
			}
			insertInstanceForDeleteReceiptTest(t, store, &instance)
			record, err := manager.QueueDeleteInstanceLocal(context.Background(), instance.ID)
			if err != nil {
				t.Fatal(err)
			}
			payload := decodeDeleteRestartPayload(t, record)
			if payload.DeleteOperation == nil {
				t.Fatal("queued delete has no stable operation identity")
			}
			publishCalls := 0
			manager.publishDeleteOperation = func(
				_ context.Context,
				operation db.PublishedWriteOperation,
				got InstanceInfo,
			) (db.PublishedWriteReceipt, error) {
				publishCalls++
				if operation != payload.DeleteOperation.publishedWriteOperation() || got.ID != instance.ID {
					t.Fatalf("delete publication changed operation/instance: operation=%+v instance=%+v", operation, got)
				}
				return db.PublishedWriteReceipt{}, tt.err
			}
			for attempt := 0; attempt < instanceDeleteMaxAttempts; attempt++ {
				_ = manager.tasks.RunPending(context.Background())
			}
			done, err := manager.tasks.Get(record.ID)
			if err != nil {
				t.Fatal(err)
			}
			if done.Status != tasks.StatusFailed || done.Attempts != 1 || publishCalls != 1 {
				t.Fatalf(
					"unsafe delete task status=%s attempts=%d publications=%d error=%q, want failed/1/1",
					done.Status,
					done.Attempts,
					publishCalls,
					done.ErrorMessage,
				)
			}
			if tt.want != nil && !strings.Contains(done.ErrorMessage, tt.want.Error()) {
				t.Fatalf("unsafe delete error=%q, want %v", done.ErrorMessage, tt.want)
			}
			resolved, err := store.LookupPublishedWriteOperation(
				context.Background(),
				payload.DeleteOperation.publishedWriteOperation(),
			)
			if err != nil {
				t.Fatal(err)
			}
			if resolved.State != swarmionapp.OperationResolvedAbsent {
				t.Fatalf("unsafe delete authored an operation despite no receipt: %+v", resolved)
			}
			if _, err := manager.GetInstance(instance.ID); err != nil {
				t.Fatalf("unsafe delete removed instance despite failed final publication: %v", err)
			}
		})
	}
}
