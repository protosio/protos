package provisioners

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

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
	}
}

func appliedInstanceDeleteObservation(receipt instanceDeleteOperationReceipt, coverage swarmionapp.BranchEventContentCoverage) db.EventReceiptObservation {
	contentDurable := coverage == swarmionapp.BranchEventContentCoverageCovered
	return db.EventReceiptObservation{
		Receipt: db.PublishedWriteReceipt{
			Committed:          true,
			Checkpointed:       true,
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
	identity, err := newInstanceDeleteOperationIdentity(store, operationID, instance, localOnly, status.PeerID)
	if err != nil {
		t.Fatal(err)
	}
	return identity
}

// publishedWriteOperationWithAuthorForTest models a recovery record persisted
// by a different peer. It uses the public strict JSON contract instead of
// mutating opaque runtime fields.
func publishedWriteOperationWithAuthorForTest(
	t *testing.T,
	operation db.PublishedWriteOperation,
	author string,
) db.PublishedWriteOperation {
	t.Helper()
	encoded, err := json.Marshal(operation)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatal(err)
	}
	recovery, ok := document["recovery"].(map[string]any)
	if !ok {
		t.Fatalf("operation recovery JSON has unexpected shape: %s", encoded)
	}
	recovery["author_peer_id"] = strings.TrimSpace(author)
	encoded, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	var rewritten db.PublishedWriteOperation
	if err := json.Unmarshal(encoded, &rewritten); err != nil {
		t.Fatal(err)
	}
	if err := rewritten.Validate(); err != nil {
		t.Fatalf("rewritten foreign operation: %v", err)
	}
	return rewritten
}

func publishedWriteOperationWithKeyForTest(
	t *testing.T,
	operation db.PublishedWriteOperation,
	key string,
) db.PublishedWriteOperation {
	t.Helper()
	encoded, err := json.Marshal(operation)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatal(err)
	}
	identity, ok := document["identity"].(map[string]any)
	if !ok {
		t.Fatalf("operation identity JSON has unexpected shape: %s", encoded)
	}
	recovery, ok := document["recovery"].(map[string]any)
	if !ok {
		t.Fatalf("operation recovery JSON has unexpected shape: %s", encoded)
	}
	recoveryIdentity, ok := recovery["identity"].(map[string]any)
	if !ok {
		t.Fatalf("recovery identity JSON has unexpected shape: %s", encoded)
	}
	identity["key"] = strings.TrimSpace(key)
	recoveryIdentity["key"] = strings.TrimSpace(key)
	encoded, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	var rewritten db.PublishedWriteOperation
	if err := json.Unmarshal(encoded, &rewritten); err != nil {
		t.Fatal(err)
	}
	if err := rewritten.Validate(); err != nil {
		t.Fatalf("rewritten operation key: %v", err)
	}
	return rewritten
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
	store := openProvisionerTestDB(t)
	receipt := testInstanceDeleteReceipt()
	identity := instanceDeleteOperationIdentityForTest(t, store, receipt.OperationID, InstanceInfo{ID: receipt.ExpectedInvariant.InstanceID}, true)
	receipt.ExpectedInvariant = identity.ExpectedInvariant
	receipt.OperationIntentDigest = identity.Operation.IntentDigest()
	receipt.OperationAuthorPeerID = identity.Operation.AuthorPeerID()
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

func TestParkedDeleteRecoveryReobservesWithoutTaskWriteOrRepublish(t *testing.T) {
	store := openProvisionerTestDB(t)
	manager := newLifecycleTestManager(t, store, newProvisionerRegistry())
	taskID := db.MustNewUUIDv7()
	instance := InstanceInfo{ID: db.MustNewUUIDv7(), Name: "parked-delete"}
	identity := instanceDeleteOperationIdentityForTest(t, store, taskID, instance, true)
	effectFact, err := newInstanceDeleteEffectFact(taskID, identity)
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := db.DeleteAndInsertWithOperationReceiptContext(
		context.Background(),
		store,
		identity.Operation,
		nil,
		[]db.InsertMapper{tasks.InsertOperationFactMapper(effectFact)},
	)
	if err != nil {
		t.Fatalf("publish parked-recovery fixture operation: %v", err)
	}
	eventID := accepted.EventID
	publishedRoot := accepted.PublishedRootHash
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

	record, err := tasks.EnqueueContext(context.Background(), manager.tasks, tasks.EnqueueOptions[instanceLifecycleTaskPayload]{
		ID:          taskID,
		Stream:      InstanceLifecycleTaskStream,
		SubjectType: taskSubjectInstance,
		SubjectID:   instanceLifecycleSubjectID(instance.ID, instanceLifecycleOperationDelete),
		Title:       "parked delete",
		Payload: instanceLifecycleTaskPayload{
			InstanceID:      instance.ID,
			InstanceName:    instance.Name,
			Operation:       instanceLifecycleOperationDelete,
			DesiredStatus:   ServerStateDeleting,
			LocalOnly:       true,
			RecoveryModel:   instanceDeleteRecoveryModel,
			DeleteOperation: &identity,
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
	if observeCalls != len(parkedStates) {
		t.Fatalf("parked recovery observations=%d, want %d", observeCalls, len(parkedStates))
	}
	for _, receipt := range observedReceipts {
		if receipt.EventID != eventID || receipt.PublishedRootHash != publishedRoot || receipt.AuthorPeerID != identity.Operation.AuthorPeerID() {
			t.Fatalf("parked recovery changed exact foreign receipt: %+v", receipt)
		}
	}
	if metricsAfter := store.TransactionMetrics(); metricsAfter != metricsBefore {
		t.Fatalf("parked recovery wrote a transaction: before=%+v after=%+v", metricsBefore, metricsAfter)
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

func TestOperationFactRecoveryPublishesCurrentReceiptFact(t *testing.T) {
	store := openProvisionerTestDB(t)
	manager := newLifecycleTestManager(t, store, newProvisionerRegistry())
	taskID := db.MustNewUUIDv7()
	instance := InstanceInfo{ID: db.MustNewUUIDv7(), Name: "immutable-delete-recovery"}
	identity := instanceDeleteOperationIdentityForTest(t, store, taskID, instance, true)

	effectFact, err := newInstanceDeleteEffectFact(taskID, identity)
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := db.DeleteAndInsertWithOperationReceiptContext(
		context.Background(),
		store,
		identity.Operation,
		nil,
		[]db.InsertMapper{tasks.InsertOperationFactMapper(effectFact)},
	)
	if err != nil {
		t.Fatalf("publish immutable delete effect operation: %v", err)
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

	record, err := tasks.EnqueueContext(context.Background(), manager.tasks, tasks.EnqueueOptions[instanceLifecycleTaskPayload]{
		ID:          taskID,
		Stream:      InstanceLifecycleTaskStream,
		SubjectType: taskSubjectInstance,
		SubjectID:   instanceLifecycleSubjectID(instance.ID, instanceLifecycleOperationDelete),
		Title:       "immutable delete recovery",
		Payload: instanceLifecycleTaskPayload{
			InstanceID:      instance.ID,
			InstanceName:    instance.Name,
			Operation:       instanceLifecycleOperationDelete,
			DesiredStatus:   ServerStateDeleting,
			LocalOnly:       true,
			RecoveryModel:   instanceDeleteRecoveryModel,
			DeleteOperation: &identity,
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
	recoveredRecord, err := manager.tasks.Get(taskID)
	if err != nil {
		t.Fatal(err)
	}
	recoveredPayload := decodeDeleteRestartPayload(t, recoveredRecord)
	if recoveredRecord.Status != tasks.StatusPending {
		t.Fatalf("recovered record=%+v payload=%+v, want pending", recoveredRecord, recoveredPayload)
	}
	receiptFact, found, err := manager.tasks.OperationFact(context.Background(), taskID, tasks.OperationFactKindReceipt)
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
}

func TestCurrentReceiptFactRejectsRemovedFields(t *testing.T) {
	store := openProvisionerTestDB(t)
	taskID := db.MustNewUUIDv7()
	instance := InstanceInfo{ID: db.MustNewUUIDv7()}
	identity := instanceDeleteOperationIdentityForTest(t, store, taskID, instance, true)
	receipt := testInstanceDeleteReceipt()
	receipt.OperationID = taskID
	receipt.ExpectedInvariant = identity.ExpectedInvariant
	receipt.OperationIntentDigest = identity.Operation.IntentDigest()
	receipt.OperationAuthorPeerID = identity.Operation.AuthorPeerID()
	fact, err := newInstanceDeleteReceiptFact(receipt, identity)
	if err != nil {
		t.Fatal(err)
	}
	fact.Payload = []byte(strings.TrimSuffix(string(fact.Payload), "}") + `,"event_digest":"retired"}`)
	if _, err := instanceDeleteReceiptFromFact(fact, identity); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("decode receipt fact error = %v, want unknown-field rejection", err)
	}
}

func TestCurrentReceiptFactWithoutEffectFailsRecoveryBeforeLookup(t *testing.T) {
	store := openProvisionerTestDB(t)
	manager := newLifecycleTestManager(t, store, newProvisionerRegistry())
	taskID := db.MustNewUUIDv7()
	instance := InstanceInfo{ID: db.MustNewUUIDv7(), Name: "receipt-without-effect"}
	identity := instanceDeleteOperationIdentityForTest(t, store, taskID, instance, true)
	receipt := testInstanceDeleteReceipt()
	receipt.OperationID = taskID
	receipt.ExpectedInvariant = identity.ExpectedInvariant
	receipt.OperationIntentDigest = identity.Operation.IntentDigest()
	receipt.OperationAuthorPeerID = identity.Operation.AuthorPeerID()
	fact, err := newInstanceDeleteReceiptFact(receipt, identity)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.tasks.EnsureOperationFact(context.Background(), fact); err != nil {
		t.Fatal(err)
	}
	lookupCalls := 0
	manager.lookupDeleteRecoveryOperation = func(context.Context, db.PublishedWriteOperation) (swarmionapp.OperationResult, error) {
		lookupCalls++
		return swarmionapp.OperationResult{}, errors.New("lookup must not run")
	}
	record, err := tasks.EnqueueContext(context.Background(), manager.tasks, tasks.EnqueueOptions[instanceLifecycleTaskPayload]{
		ID:          taskID,
		Stream:      InstanceLifecycleTaskStream,
		SubjectType: taskSubjectInstance,
		SubjectID:   instanceLifecycleSubjectID(instance.ID, instanceLifecycleOperationDelete),
		Title:       "receipt without effect",
		Payload: instanceLifecycleTaskPayload{
			InstanceID:      instance.ID,
			InstanceName:    instance.Name,
			Operation:       instanceLifecycleOperationDelete,
			LocalOnly:       true,
			RecoveryModel:   instanceDeleteRecoveryModel,
			DeleteOperation: &identity,
		},
		MaxAttempts: instanceDeleteMaxAttempts,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.tasks.Update(record.ID, tasks.StatusRunning, 90, "interrupted", nil); err != nil {
		t.Fatal(err)
	}
	recovered, recoverErr := manager.tasks.RecoverOwnedRunning()
	if recovered != 0 || recoverErr == nil || !tasks.IsPermanent(recoverErr) ||
		!strings.Contains(recoverErr.Error(), "without its atomic effect fact") {
		t.Fatalf("recovery = %d, error = %v; want permanent receipt-without-effect rejection", recovered, recoverErr)
	}
	if lookupCalls != 0 {
		t.Fatalf("operation lookups = %d, want zero", lookupCalls)
	}
}

func TestRetiredRawDeletePayloadCannotSelectCurrentRecoveryModel(t *testing.T) {
	raw := []byte(`{"instance_id":"instance-a","operation":"delete","local_only":true,"operation_state_model":"immutable_operation_facts_v1"}`)
	var payload instanceLifecycleTaskPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.RecoveryModel != "" {
		t.Fatalf("retired payload selected current recovery model %q", payload.RecoveryModel)
	}
	if err := validateInstanceDeleteTaskPayloadModel(payload, "task-a"); err == nil || !strings.Contains(err.Error(), "unsupported recovery model") {
		t.Fatalf("retired payload validation error = %v, want unsupported recovery model", err)
	}
}

func TestUnsupportedInstanceDeletePayloadFailsClosedBeforeImperativeWork(t *testing.T) {
	tests := []struct {
		name          string
		recoveryModel string
		withOperation bool
		wantError     string
	}{
		{
			name:          "missing recovery model",
			withOperation: true,
			wantError:     "unsupported recovery model",
		},
		{
			name:          "retired mutable checkpoint model",
			recoveryModel: "unsupported_mutable_task_checkpoint",
			withOperation: true,
			wantError:     "unsupported recovery model",
		},
		{
			name:          "missing immutable operation identity",
			recoveryModel: instanceDeleteRecoveryModel,
			wantError:     "missing its immutable operation identity",
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
			record, err := tasks.EnqueueContext(context.Background(), manager.tasks, tasks.EnqueueOptions[instanceLifecycleTaskPayload]{
				ID:          taskID,
				Stream:      InstanceLifecycleTaskStream,
				SubjectType: taskSubjectInstance,
				SubjectID:   instanceLifecycleSubjectID(instanceID, instanceLifecycleOperationDelete),
				Title:       "unsupported delete payload",
				Payload: instanceLifecycleTaskPayload{
					InstanceID:      instanceID,
					Operation:       instanceLifecycleOperationDelete,
					LocalOnly:       true,
					RecoveryModel:   tt.recoveryModel,
					DeleteOperation: identity,
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
	record, err := tasks.EnqueueContext(context.Background(), manager.tasks, tasks.EnqueueOptions[instanceLifecycleTaskPayload]{
		ID:          taskID,
		Stream:      InstanceLifecycleTaskStream,
		SubjectType: taskSubjectInstance,
		SubjectID:   instanceLifecycleSubjectID(instanceID, instanceLifecycleOperationDelete),
		Title:       "retired delete recovery payload",
		Payload: instanceLifecycleTaskPayload{
			InstanceID:      instanceID,
			Operation:       instanceLifecycleOperationDelete,
			LocalOnly:       true,
			RecoveryModel:   "unsupported_mutable_task_checkpoint",
			DeleteOperation: &identity,
		},
		MaxAttempts: instanceDeleteMaxAttempts,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.tasks.Update(record.ID, tasks.StatusRunning, 90, "interrupted unsupported delete", nil); err != nil {
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
		!strings.Contains(recoverErr.Error(), "unsupported recovery model") {
		t.Fatalf("unsupported recovery=%d error=%v, want permanent fail-closed error", recovered, recoverErr)
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
			err:  errors.New("delete publication became inconclusive without a receipt"),
			want: errors.New("delete publication became inconclusive without a receipt"),
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
				if !operation.Equal(payload.DeleteOperation.publishedWriteOperation()) || got.ID != instance.ID {
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
			if resolved.Disposition() != swarmionapp.OperationRetryPermitted {
				t.Fatalf("unsafe delete disposition=%s diagnostic=%v, want retry permitted absence", resolved.Disposition(), resolved.Diagnostic())
			}
			if _, err := manager.GetInstance(instance.ID); err != nil {
				t.Fatalf("unsafe delete removed instance despite failed final publication: %v", err)
			}
		})
	}
}
