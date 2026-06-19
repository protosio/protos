package db

import (
	"errors"
	"testing"

	swarmionprotocol "github.com/nustiueudinastea/swarmion/protocol"
	swarmionapp "github.com/nustiueudinastea/swarmion/runtime/app"
)

func TestRetryableCommittedWriteErrorsIncludeDoltWorkingSetContention(t *testing.T) {
	tests := []error{
		errors.New(`failed to load database "protos": the database is locked by another dolt process`),
		errors.New("update tentative working set: cannot update manifest: database is read only"),
		errors.New("update tentative working set: cannot update manifest"),
		errSwarmionCheckpointedWriteRejected,
	}

	for _, err := range tests {
		if !isRetryableCommittedWriteError(err) {
			t.Fatalf("expected retryable error: %v", err)
		}
	}
}

func TestRetryableApplyRequiresResetOnlyAfterStaging(t *testing.T) {
	transient := errors.New(`re-open tentative dolt db: ping dolt driver: failed to load database "protos": the database is locked by another dolt process`)
	if retryableApplyRequiresReset(writeApplyError{err: transient}) {
		t.Fatal("pre-stage transient workspace access error should retry without reset")
	}
	if !retryableApplyRequiresReset(writeApplyError{err: transient, staged: true}) {
		t.Fatal("post-stage transient workspace access error should reset before retry")
	}
	if !retryableApplyRequiresReset(errors.New("stale write context")) {
		t.Fatal("generic retryable write error should reset before retry")
	}
}

func TestParseStagedCommitResult(t *testing.T) {
	result := parseStagedCommitResult(
		[]string{
			"committed",
			"hash",
			"event_id",
			"published_root_hash",
			"write_base_root_hash",
			"workspace_head_root_hash",
			"workspace_staged_root_hash",
			"workspace_working_root_hash",
		},
		[]any{
			[]byte("true"),
			[]byte("commit-hash"),
			"event-id",
			"published-root",
			"base-root",
			"head-root",
			"staged-root",
			"working-root",
		},
	)

	if !result.Committed {
		t.Fatal("commit result should be marked committed")
	}
	if result.Hash != "commit-hash" || result.EventID != "event-id" || result.PublishedRootHash != "published-root" {
		t.Fatalf("unexpected commit result: %+v", result)
	}
	if result.WriteBaseRootHash != "base-root" || result.WorkspaceHeadRootHash != "head-root" ||
		result.WorkspaceStagedRootHash != "staged-root" || result.WorkspaceWorkingRootHash != "working-root" {
		t.Fatalf("unexpected workspace roots: %+v", result)
	}
}

func TestStagedCommitCheckpointReached(t *testing.T) {
	commit := stagedCommitResult{
		Committed:         true,
		Hash:              "commit-hash",
		EventID:           "event-1",
		PublishedRootHash: "published-root",
	}
	advancedStatus := swarmionCheckpointStatus("advanced-root")
	acceptedSnapshot := swarmionCheckpointSnapshot(commit.EventID, commit.PublishedRootHash, true, swarmionprotocol.CheckpointEventDecisionAccepted)
	reached, err := stagedCommitCheckpointReached(advancedStatus, acceptedSnapshot, commit)
	if err != nil {
		t.Fatalf("accepted checkpoint should not fail: %v", err)
	}
	if !reached {
		t.Fatal("accepted checkpoint event should satisfy committed write visibility")
	}

	pendingSnapshot := swarmionCheckpointSnapshot(commit.EventID, commit.PublishedRootHash, false, swarmionprotocol.CheckpointEventDecisionAccepted)
	reached, err = stagedCommitCheckpointReached(advancedStatus, pendingSnapshot, commit)
	if err != nil {
		t.Fatalf("pending checkpoint should not fail: %v", err)
	}
	if reached {
		t.Fatal("event summary alone should not satisfy visibility until the current checkpoint covers the event root")
	}

	rejectedSnapshot := swarmionCheckpointSnapshot(commit.EventID, commit.PublishedRootHash, false, swarmionprotocol.CheckpointEventDecisionRejectedConflict)
	reached, err = stagedCommitCheckpointReached(advancedStatus, rejectedSnapshot, commit)
	if !errors.Is(err, errSwarmionCheckpointedWriteRejected) {
		t.Fatalf("rejected checkpoint should return retryable rejection error: %v", err)
	}
	if reached {
		t.Fatal("rejected checkpoint should not satisfy committed write visibility")
	}

	mismatchedSnapshot := swarmionCheckpointSnapshot(commit.EventID, "other-root", true, swarmionprotocol.CheckpointEventDecisionAccepted)
	reached, err = stagedCommitCheckpointReached(advancedStatus, mismatchedSnapshot, commit)
	if !errors.Is(err, errSwarmionCheckpointedWriteRejected) {
		t.Fatalf("checkpoint root mismatch should return retryable rejection error: %v", err)
	}
	if reached {
		t.Fatal("checkpoint root mismatch should not satisfy committed write visibility")
	}

	unstableStatus := advancedStatus
	unstableStatus.TentativeRootHash = swarmionprotocol.NewRootHash("different-root")
	reached, err = stagedCommitCheckpointReached(unstableStatus, acceptedSnapshot, commit)
	if err != nil {
		t.Fatalf("unstable checkpoint should wait without failing: %v", err)
	}
	if reached {
		t.Fatal("unstable checkpoint roots should not satisfy committed write visibility")
	}
}

func swarmionCheckpointStatus(root string) swarmionapp.Status {
	rootHash := swarmionprotocol.NewRootHash(root)
	return swarmionapp.Status{
		CheckpointRootHash:               rootHash,
		TentativeRootHash:                rootHash,
		DurableMainRootHash:              rootHash,
		RuntimeCheckpointDesiredRootHash: rootHash,
	}
}

func swarmionCheckpointSnapshot(eventID string, root string, covered bool, decision swarmionprotocol.CheckpointEventDecision) *swarmionprotocol.NodeState {
	checkpointID := swarmionprotocol.NewCheckpointCommitID("checkpoint-commit")
	event := swarmionprotocol.NewEventID(eventID)
	rootHash := swarmionprotocol.NewRootHash(root)
	snapshot := swarmionprotocol.NewNodeState("", nil)
	snapshot.CheckpointCommitID = checkpointID
	snapshot.CheckpointProllyRootHash = swarmionprotocol.NewRootHash("checkpoint-root")
	snapshot.CheckpointEventRoots[event] = rootHash
	snapshot.CheckpointEventDecisions[event] = decision
	if covered {
		snapshot.CheckpointCommitEvents[checkpointID] = map[swarmionprotocol.EventID]swarmionprotocol.RootHash{event: rootHash}
		snapshot.CheckpointCommitDecisions[checkpointID] = map[swarmionprotocol.EventID]swarmionprotocol.CheckpointEventDecision{event: decision}
	}
	return snapshot
}
