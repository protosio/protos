package provisioners

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/tasks"
)

func TestDeleteInstanceRecordsRealDissentAfterCompetingRecreation(t *testing.T) {
	t.Setenv("SWARMION_CHECKPOINT_SCHEDULER_PROFILE", "single_node_development")
	store := openProvisionerTestDB(t)

	target := InstanceInfo{
		ID:                  db.MustNewUUIDv7(),
		Name:                "delete-receipt-target",
		Kind:                KindLocalVM,
		KindID:              "delete-receipt-test",
		ProviderResourceID:  "target-provider-id",
		DesiredStatus:       ServerStateRunning,
		ReplicationPriority: 1,
		Location:            "test",
	}
	sentinel := InstanceInfo{
		ID:                  db.MustNewUUIDv7(),
		Name:                "delete-receipt-sentinel",
		Kind:                KindLocalVM,
		KindID:              "delete-receipt-test",
		ProviderResourceID:  "sentinel-provider-id",
		DesiredStatus:       ServerStateRunning,
		ReplicationPriority: 1,
		Location:            "test",
	}
	target = instanceWithTestLifecycleOwner(t, store, target)
	sentinel = instanceWithTestLifecycleOwner(t, store, sentinel)
	targetMachine, targetMetadata := createInstanceInsertMapper(target)
	sentinelMachine, sentinelMetadata := createInstanceInsertMapper(sentinel)
	setupReceipt, err := db.InsertWithReceiptContext(
		context.Background(),
		store,
		targetMachine,
		targetMetadata,
		sentinelMachine,
		sentinelMetadata,
	)
	if err != nil {
		t.Fatalf("insert delete receipt fixture: %v", err)
	}
	waitForRealDeleteTestEventApplied(t, store, setupReceipt, "prepare delete receipt fixture")

	beforeStatus, ok := store.SwarmionStatus()
	if !ok {
		t.Fatal("read Swarmion status before delete")
	}
	beforeDissent := store.EventReceiptMetrics().ContentDissentObservations

	manager := &Manager{db: store}
	var (
		persisted             []instanceDeleteOperationReceipt
		deleteCovered         db.EventReceiptObservation
		recreationObservation db.EventReceiptObservation
		finalHeadObservation  db.EventReceiptObservation
		orchestrated          bool
	)
	persistReceipt := func(receipt instanceDeleteOperationReceipt, _ int, _ string) error {
		persisted = append(persisted, *cloneInstanceDeleteOperationReceipt(&receipt))
		if orchestrated {
			return nil
		}
		orchestrated = true

		// First prove that the production delete event itself reaches the normal
		// covered state. The later writes below deliberately move the durable head
		// to a state which no longer contains that complete historical root.
		deleteCovered = waitForRealDeleteTestEventApplied(
			t,
			store,
			receipt.publishedWriteReceipt(),
			"observe covered instance delete",
		)
		if deleteCovered.Status.ContentCoverage != swarmionapp.BranchEventContentCoverageCovered || !deleteCovered.Status.Durable {
			t.Fatalf("initial delete receipt=%+v, want applied_durably with content_covered", deleteCovered)
		}

		// Recreate the same logical provision at the exact primary key. This is
		// the competing application write which the durable invariant must detect.
		recreatedMachine, recreatedMetadata := createInstanceInsertMapper(target)
		recreationReceipt, err := db.InsertWithReceiptContext(
			context.Background(),
			store,
			recreatedMachine,
			recreatedMetadata,
		)
		if err != nil {
			t.Fatalf("recreate deleted instance: %v", err)
		}
		if recreationReceipt.EventID == receipt.EventID {
			t.Fatalf("recreation reused delete event ID %s", receipt.EventID)
		}
		recreationObservation = waitForRealDeleteTestEventApplied(
			t,
			store,
			recreationReceipt,
			"apply competing instance recreation",
		)

		// A recreation is additive relative to a root where the row is absent and
		// can therefore still cover that root. Overwrite a second existing row to
		// deterministically force RootCovers to report a conflict/non-cover while
		// retaining the recreated target at the new durable head.
		sentinel.DesiredStatus = ServerStateStopped
		sentinelUpdate, _ := createInstanceUpdateMapper(sentinel)
		finalHeadReceipt, err := db.UpdateWithReceiptContext(
			context.Background(),
			store,
			sentinelUpdate,
		)
		if err != nil {
			t.Fatalf("overwrite delete receipt sentinel: %v", err)
		}
		finalHeadObservation = waitForRealDeleteTestEventApplied(
			t,
			store,
			finalHeadReceipt,
			"apply delete receipt dissent head",
		)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	operationID := db.MustNewUUIDv7()
	identity := instanceDeleteOperationIdentityForTest(t, store, operationID, target, true)
	err = manager.deleteInstanceRecords(ctx, operationID, identity, target, persistReceipt)
	if !errors.Is(err, ErrInstanceDeleteInvariantConflict) {
		t.Fatalf("delete completion error=%v, want %v", err, ErrInstanceDeleteInvariantConflict)
	}
	if errors.Is(err, db.ErrEventReceiptPending) || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("dissent/recreation was treated as unresolved transport work: %v", err)
	}

	if len(persisted) != 2 {
		t.Fatalf("persisted receipt observations=%d, want initial and applied", len(persisted))
	}
	initialReceipt := persisted[0]
	finalReceipt := persisted[1]
	if initialReceipt.EventID == "" || initialReceipt.PublishedRootHash == "" ||
		finalReceipt.EventID != initialReceipt.EventID ||
		finalReceipt.PublishedRootHash != initialReceipt.PublishedRootHash {
		t.Fatalf("delete receipt identity changed during completion: initial=%+v final=%+v", initialReceipt, finalReceipt)
	}
	if deleteCovered.Receipt.EventID != initialReceipt.EventID || deleteCovered.Receipt.PublishedRootHash != initialReceipt.PublishedRootHash {
		t.Fatalf("covered observation used a different delete receipt: observation=%+v receipt=%+v", deleteCovered.Receipt, initialReceipt)
	}
	if !finalReceipt.AppliedDurably || finalReceipt.ContentDurable ||
		finalReceipt.ContentCoverage != swarmionapp.BranchEventContentCoverageDissent {
		t.Fatalf("final delete receipt=%+v, want applied_durably content_dissent without full-root durability", finalReceipt)
	}
	if finalReceipt.Proof == nil || !finalReceipt.Proof.EventContained || !finalReceipt.Proof.ProofRan ||
		finalReceipt.Proof.Covered || !finalReceipt.Proof.Conflict {
		t.Fatalf("final delete proof=%+v, want contained, executed conflict/non-cover proof", finalReceipt.Proof)
	}
	if finalReceipt.DurableCheckpointCommitID == "" ||
		finalReceipt.DurableCheckpointCommitID != finalHeadObservation.Status.DurableCheckpointCommitID ||
		finalReceipt.DurableCheckpointRootHash != finalHeadObservation.Status.DurableCheckpointRootHash {
		t.Fatalf(
			"delete invariant used durable head %s/%s, want later head %s/%s",
			finalReceipt.DurableCheckpointCommitID,
			finalReceipt.DurableCheckpointRootHash,
			finalHeadObservation.Status.DurableCheckpointCommitID,
			finalHeadObservation.Status.DurableCheckpointRootHash,
		)
	}
	if recreationObservation.Status.DurableCheckpointCommitID == "" {
		t.Fatal("competing recreation did not reach a durable checkpoint")
	}
	effectFact, err := newInstanceDeleteEffectFact(operationID, identity)
	if err != nil {
		t.Fatal(err)
	}
	var effectFactCount int
	if err := store.ReadRowsAsOf(
		context.Background(),
		deleteCovered.Status.CheckpointCommitID,
		"SELECT COUNT(*) FROM task_operation_facts AS OF ? WHERE id = ? AND fact_kind = ?",
		[]any{effectFact.ID, tasks.OperationFactKindEffect},
		func(rows *sql.Rows) error {
			if !rows.Next() {
				return errors.New("effect fact count query returned no row")
			}
			return rows.Scan(&effectFactCount)
		},
	); err != nil {
		t.Fatalf("query atomic delete effect fact: %v", err)
	}
	if effectFactCount != 1 {
		t.Fatalf("delete checkpoint contains %d operation effect facts, want exactly one", effectFactCount)
	}

	present, err := (swarmionInstanceDeleteReceiptTracker{database: store}).InstanceExistsAtCheckpoint(
		context.Background(),
		finalReceipt.DurableCheckpointCommitID,
		target.ID,
	)
	if err != nil {
		t.Fatalf("query recreated instance at durable checkpoint: %v", err)
	}
	if !present {
		t.Fatalf("instance %s is absent at competing durable head", target.ID)
	}

	afterStatus, ok := store.SwarmionStatus()
	if !ok {
		t.Fatal("read Swarmion status after delete conflict")
	}
	if delta := afterStatus.CheckpointEventCount - beforeStatus.CheckpointEventCount; delta != 3 {
		t.Fatalf("checkpoint event delta=%d, want exactly delete, recreation, and sentinel overwrite with no duplicate delete", delta)
	}
	if delta := store.EventReceiptMetrics().ContentDissentObservations - beforeDissent; delta != 1 {
		t.Fatalf("backend dissent metric delta=%d, want 1", delta)
	}
}

func waitForRealDeleteTestEventApplied(
	t *testing.T,
	store *db.DB,
	receipt db.PublishedWriteReceipt,
	reason string,
) db.EventReceiptObservation {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	observation, err := store.WaitForPublishedWriteApplied(ctx, receipt, reason)
	if err != nil {
		t.Fatalf("wait for %s: %v", reason, err)
	}
	if !observation.Status.AppliedDurably {
		t.Fatalf("%s receipt did not reach applied_durably: %+v", reason, observation)
	}
	return observation
}
