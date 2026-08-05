package db

import (
	"errors"
	"strings"
	"testing"

	swarmionapp "github.com/nustiueudinastea/swarmion/runtime/app"
)

func TestRetryableCommittedWriteErrorsIncludeDoltWorkingSetContention(t *testing.T) {
	tests := []error{
		errors.New(`failed to load database "protos": the database is locked by another dolt process`),
		errors.New("update tentative working set: cannot update manifest: database is read only"),
		errors.New("update tentative working set: cannot update manifest"),
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
	durableStatus := swarmionapp.BranchRootStatus{RootHash: commit.PublishedRootHash, Checkpointed: true, Durable: true}
	reached, err := stagedCommitCheckpointReached(durableStatus, commit)
	if err != nil {
		t.Fatalf("durable root should not fail: %v", err)
	}
	if !reached {
		t.Fatal("durable root should satisfy committed write visibility")
	}

	pendingStatus := swarmionapp.BranchRootStatus{RootHash: commit.PublishedRootHash, Pending: true, PendingReason: "checkpoint_ordering"}
	reached, err = stagedCommitCheckpointReached(pendingStatus, commit)
	if err != nil {
		t.Fatalf("pending checkpoint should not fail: %v", err)
	}
	if reached {
		t.Fatal("pending root should not satisfy durable visibility")
	}

	parkedStatus := swarmionapp.BranchRootStatus{RootHash: commit.PublishedRootHash, Parked: true, ParkedReason: swarmionapp.BranchRootParkedReasonConflict}
	reached, err = stagedCommitCheckpointReached(parkedStatus, commit)
	if err != nil {
		t.Fatalf("parked root should remain a status, not a rejection error: %v", err)
	}
	if reached {
		t.Fatal("parked root should not satisfy durable visibility")
	}
	if waitErr := stagedCommitCheckpointWaitError(parkedStatus, commit); !strings.Contains(waitErr.Error(), "root_status=parked_conflict") || !strings.Contains(waitErr.Error(), "revisitable=true") {
		t.Fatalf("parked wait error=%v, want revisitable lifecycle", waitErr)
	}

	mismatchedStatus := durableStatus
	mismatchedStatus.RootHash = "other-root"
	reached, err = stagedCommitCheckpointReached(mismatchedStatus, commit)
	if err != nil {
		t.Fatalf("mismatched root status should wait without failing: %v", err)
	}
	if reached {
		t.Fatal("mismatched root status should not satisfy durable visibility")
	}
}
