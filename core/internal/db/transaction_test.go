package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/bokwoon95/sq"
	swarmionprotocol "github.com/nustiueudinastea/swarmion/protocol"
	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
)

// transactionMetricsDelta remains shared by the restart and pending-chain
// suites. The assertions in this file intentionally focus on the public
// publication contract rather than Swarmion's removed transaction phases.
func transactionMetricsDelta(after, before TransactionMetricsSnapshot) TransactionMetricsSnapshot {
	return TransactionMetricsSnapshot{
		TransactionsStarted:                         after.TransactionsStarted - before.TransactionsStarted,
		CommitsAttempted:                            after.CommitsAttempted - before.CommitsAttempted,
		CommitsSucceeded:                            after.CommitsSucceeded - before.CommitsSucceeded,
		CommitsFailed:                               after.CommitsFailed - before.CommitsFailed,
		NoopCommitOutcomes:                          after.NoopCommitOutcomes - before.NoopCommitOutcomes,
		RollbacksAttempted:                          after.RollbacksAttempted - before.RollbacksAttempted,
		RollbacksSucceeded:                          after.RollbacksSucceeded - before.RollbacksSucceeded,
		RollbacksFailed:                             after.RollbacksFailed - before.RollbacksFailed,
		RollbacksApplyPhase:                         after.RollbacksApplyPhase - before.RollbacksApplyPhase,
		RollbacksBeforeCommitPhase:                  after.RollbacksBeforeCommitPhase - before.RollbacksBeforeCommitPhase,
		RollbacksPanicPhase:                         after.RollbacksPanicPhase - before.RollbacksPanicPhase,
		RollbacksApplyFailure:                       after.RollbacksApplyFailure - before.RollbacksApplyFailure,
		RollbacksContextCanceled:                    after.RollbacksContextCanceled - before.RollbacksContextCanceled,
		RollbacksContextDeadline:                    after.RollbacksContextDeadline - before.RollbacksContextDeadline,
		RollbacksSQLViewNotReady:                    after.RollbacksSQLViewNotReady - before.RollbacksSQLViewNotReady,
		RollbacksPanic:                              after.RollbacksPanic - before.RollbacksPanic,
		TypedConflicts:                              after.TypedConflicts - before.TypedConflicts,
		OperationReceiptsFoundAfterCommitErr:        after.OperationReceiptsFoundAfterCommitErr - before.OperationReceiptsFoundAfterCommitErr,
		UncertainEventReceiptsAfterCommitErr:        after.UncertainEventReceiptsAfterCommitErr - before.UncertainEventReceiptsAfterCommitErr,
		SQLViewNotReadyOutcomes:                     after.SQLViewNotReadyOutcomes - before.SQLViewNotReadyOutcomes,
		StaleWriteContextOutcomes:                   after.StaleWriteContextOutcomes - before.StaleWriteContextOutcomes,
		ProjectionTooWideOutcomes:                   after.ProjectionTooWideOutcomes - before.ProjectionTooWideOutcomes,
		OperationTransactionsAttempted:              after.OperationTransactionsAttempted - before.OperationTransactionsAttempted,
		OperationTransactionsExecuted:               after.OperationTransactionsExecuted - before.OperationTransactionsExecuted,
		OperationTransactionsAlreadyAccepted:        after.OperationTransactionsAlreadyAccepted - before.OperationTransactionsAlreadyAccepted,
		OperationTransactionsNoChange:               after.OperationTransactionsNoChange - before.OperationTransactionsNoChange,
		OperationTransactionsFailed:                 after.OperationTransactionsFailed - before.OperationTransactionsFailed,
		OperationWorkspaceDirtyOutcomes:             after.OperationWorkspaceDirtyOutcomes - before.OperationWorkspaceDirtyOutcomes,
		OperationTransactionLifecycleOpaqueFailures: after.OperationTransactionLifecycleOpaqueFailures - before.OperationTransactionLifecycleOpaqueFailures,
	}
}

func transactionTestUserInsert(id, username string) InsertMapper {
	return func() sq.InsertQuery {
		user := sq.New[USER]("")
		return sq.InsertInto(user).ColumnValues(func(col *sq.Column) {
			col.SetBytes(user.ID, MustUUIDBytes(id))
			col.SetString(user.USERNAME, username)
			col.SetString(user.NAME, username)
			col.SetBool(user.IS_DISABLED, false)
		})
	}
}

func transactionTestUserCount(t *testing.T, store *DB, id string) int {
	t.Helper()
	var count int
	if err := store.ReadRows(
		context.Background(),
		"SELECT COUNT(*) FROM users WHERE id = ?",
		[]any{MustUUIDBytes(id)},
		func(rows *sql.Rows) error {
			if !rows.Next() {
				return sql.ErrNoRows
			}
			return rows.Scan(&count)
		},
	); err != nil {
		t.Fatalf("count transaction test user %s: %v", id, err)
	}
	return count
}

func transactionTestUserName(t *testing.T, store *DB, id string) string {
	t.Helper()
	var name string
	if err := store.ReadRows(
		context.Background(),
		"SELECT name FROM users WHERE id = ?",
		[]any{MustUUIDBytes(id)},
		func(rows *sql.Rows) error {
			if !rows.Next() {
				return sql.ErrNoRows
			}
			return rows.Scan(&name)
		},
	); err != nil {
		t.Fatalf("read transaction test user %s: %v", id, err)
	}
	return name
}

func transactionTestUserUpdate(id, name string) UpdateMapper {
	return func() sq.UpdateQuery {
		user := sq.New[USER]("")
		return sq.Update(user).SetFunc(func(col *sq.Column) {
			col.SetString(user.NAME, name)
		}).Where(UUIDEq(user.ID, id))
	}
}

func transactionTestOperation(t *testing.T, intent string) PublishedWriteOperation {
	t.Helper()
	key, err := NewPublishedWriteOperationKey()
	if err != nil {
		t.Fatalf("create operation key: %v", err)
	}
	operation, err := NewPublishedWriteOperation(key, intent)
	if err != nil {
		t.Fatalf("create operation: %v", err)
	}
	return operation
}

func transactionTestRuntime(t *testing.T, store *DB) *swarmionapp.DatabaseRuntime {
	t.Helper()
	store.mu.Lock()
	runtime := store.runtime
	store.mu.Unlock()
	if runtime == nil {
		t.Fatal("Swarmion database runtime is unavailable")
	}
	return runtime
}

func requireAcceptedOperationResolution(
	t *testing.T,
	store *DB,
	operation PublishedWriteOperation,
	receipt PublishedWriteReceipt,
) swarmionapp.OperationResolution {
	t.Helper()
	resolution, err := store.LookupPublishedWriteOperation(context.Background(), operation)
	if err != nil {
		t.Fatalf("resolve published operation: %v", err)
	}
	if resolution.State != swarmionapp.OperationResolvedAccepted || resolution.Receipt == nil {
		t.Fatalf("operation resolution=%+v, want exact accepted receipt", resolution)
	}
	if resolution.Receipt.EventID != receipt.EventID || resolution.Receipt.PublishedRootHash != receipt.PublishedRootHash {
		t.Fatalf("operation resolution receipt=%+v, want %s/%s", resolution.Receipt, receipt.EventID, receipt.PublishedRootHash)
	}
	return resolution
}

func TestPublishedWriteOperationIdentityIsStableAndIntentBound(t *testing.T) {
	key, err := NewPublishedWriteOperationKey()
	if err != nil {
		t.Fatal(err)
	}
	first, err := NewPublishedWriteOperation(key, "protos:test:identity:v1", "a", "bc")
	if err != nil {
		t.Fatal(err)
	}
	rebuilt, err := NewPublishedWriteOperation(key, "protos:test:identity:v1", "a", "bc")
	if err != nil {
		t.Fatal(err)
	}
	changed, err := NewPublishedWriteOperation(key, "protos:test:identity:v1", "ab", "c")
	if err != nil {
		t.Fatal(err)
	}
	if first != rebuilt {
		t.Fatalf("rebuilt identity=%+v, want %+v", rebuilt, first)
	}
	if first.IntentDigest == changed.IntentDigest {
		t.Fatalf("length-framed intent digest collided: first=%+v changed=%+v", first, changed)
	}
}

func TestStableOperationExecutedThenAlreadyAcceptedSkipsChangedBody(t *testing.T) {
	store := openPeerTestDB(t)
	operation := transactionTestOperation(t, "protos:test:stable-operation:v1")
	firstID := MustNewUUIDv7()
	secondID := MustNewUUIDv7()

	first, err := UpdateAndInsertWithOperationReceiptContext(
		context.Background(),
		store,
		operation,
		nil,
		[]InsertMapper{transactionTestUserInsert(firstID, "operation-executed")},
	)
	if err != nil {
		t.Fatalf("execute stable operation: %v", err)
	}
	if !first.Committed || !first.HasExactEventIdentity() || transactionTestUserCount(t, store, firstID) != 1 {
		t.Fatalf("executed receipt=%+v first_count=%d", first, transactionTestUserCount(t, store, firstID))
	}

	replayed, err := UpdateAndInsertWithOperationReceiptContext(
		context.Background(),
		store,
		operation,
		nil,
		[]InsertMapper{transactionTestUserInsert(secondID, "already-accepted-body-must-not-run")},
	)
	if err != nil {
		t.Fatalf("replay stable operation: %v", err)
	}
	if replayed.EventID != first.EventID || replayed.PublishedRootHash != first.PublishedRootHash {
		t.Fatalf("replayed receipt=%+v, want original %+v", replayed, first)
	}
	if got := transactionTestUserCount(t, store, secondID); got != 0 {
		t.Fatalf("already-accepted operation executed changed body: count=%d", got)
	}
	requireAcceptedOperationResolution(t, store, operation, first)
}

func TestStableOperationStatementFailureIsAtomicAndFailsClosed(t *testing.T) {
	store := openPeerTestDB(t)
	operation := transactionTestOperation(t, "protos:test:operation-atomicity:v1")
	id := MustNewUUIDv7()

	receipt, err := UpdateAndInsertWithOperationReceiptContext(
		context.Background(),
		store,
		operation,
		nil,
		[]InsertMapper{
			transactionTestUserInsert(id, "atomic-first"),
			transactionTestUserInsert(id, "atomic-duplicate"),
		},
	)
	if err == nil {
		t.Fatal("duplicate operation statements unexpectedly succeeded")
	}
	if receipt.HasExactEventIdentity() {
		t.Fatalf("rejected operation returned a receipt: %+v", receipt)
	}
	if got := transactionTestUserCount(t, store, id); got != 0 {
		t.Fatalf("failed operation left a partial row: count=%d", got)
	}
	if IsRetryablePublishedWriteError(err) {
		t.Fatalf("untyped statement failure granted replay authority: %T %v", err, err)
	}
	resolution, lookupErr := store.LookupPublishedWriteOperation(context.Background(), operation)
	if lookupErr != nil {
		t.Fatalf("resolve rejected operation: %v", lookupErr)
	}
	if resolution.State != swarmionapp.OperationResolvedAbsent || !resolution.SafeToExecute || resolution.Receipt != nil {
		t.Fatalf("rejected operation resolution=%+v, want safe exact absence", resolution)
	}
}

func TestSafeRejectionAlonePermitsApplicationOwnedRetry(t *testing.T) {
	store := openPeerTestDB(t)
	runtime := transactionTestRuntime(t, store)
	operation := transactionTestOperation(t, "protos:test:safe-rejection:v1")
	id := MustNewUUIDv7()
	injected := errors.New("injected pre-publication rejection")

	store.executeOperationForTest = func(_ context.Context, request swarmionapp.OperationRequest) (swarmionapp.PublicationOutcome, error) {
		outcome := swarmionapp.PublicationOutcome{
			Identity:        request.Identity,
			Scope:           runtime.PublicationScope(),
			AuthorPeerID:    runtime.PeerID(),
			State:           swarmionapp.PublicationRejectedSafeToRetry,
			RejectionReason: swarmionapp.PublicationRejectionNotAccepted,
		}
		return outcome, &swarmionapp.PublicationRejectedError{Outcome: outcome, Cause: injected}
	}
	receipt, err := UpdateAndInsertWithOperationReceiptContext(
		context.Background(),
		store,
		operation,
		nil,
		[]InsertMapper{transactionTestUserInsert(id, "safe-rejection-retry")},
	)
	if receipt.HasExactEventIdentity() || !errors.Is(err, swarmionapp.ErrPublicationRejectedSafeToRetry) || !IsRetryablePublishedWriteError(err) {
		t.Fatalf("safe rejection receipt=%+v retryable=%t error=%v", receipt, IsRetryablePublishedWriteError(err), err)
	}
	if IsRetryablePublishedWriteError(errors.Join(err, errors.New("untrusted sibling"))) {
		t.Fatal("joined diagnostic sibling retained replay authority")
	}
	if got := transactionTestUserCount(t, store, id); got != 0 {
		t.Fatalf("rejected attempt executed SQL: count=%d", got)
	}

	store.executeOperationForTest = nil
	retried, err := UpdateAndInsertWithOperationReceiptContext(
		context.Background(),
		store,
		operation,
		nil,
		[]InsertMapper{transactionTestUserInsert(id, "safe-rejection-retry")},
	)
	if err != nil || !retried.HasExactEventIdentity() || transactionTestUserCount(t, store, id) != 1 {
		t.Fatalf("application-owned retry receipt=%+v count=%d error=%v", retried, transactionTestUserCount(t, store, id), err)
	}
}

func TestZeroOutcomeCannotBorrowSafeRejectionRetryAuthority(t *testing.T) {
	store := openPeerTestDB(t)
	runtime := transactionTestRuntime(t, store)
	operation := transactionTestOperation(t, "protos:test:zero-outcome-safe-rejection:v1")
	id := MustNewUUIDv7()
	executeCalls := 0

	store.executeOperationForTest = func(_ context.Context, request swarmionapp.OperationRequest) (swarmionapp.PublicationOutcome, error) {
		executeCalls++
		rejected := swarmionapp.PublicationOutcome{
			Identity:        request.Identity,
			Scope:           runtime.PublicationScope(),
			AuthorPeerID:    runtime.PeerID(),
			State:           swarmionapp.PublicationRejectedSafeToRetry,
			RejectionReason: swarmionapp.PublicationRejectionNotAccepted,
		}
		return swarmionapp.PublicationOutcome{}, &swarmionapp.PublicationRejectedError{
			Outcome: rejected,
			Cause:   errors.New("injected rejection without its correlated outcome"),
		}
	}
	receipt, err := UpdateAndInsertWithOperationReceiptContext(
		context.Background(),
		store,
		operation,
		nil,
		[]InsertMapper{transactionTestUserInsert(id, "zero-outcome-safe-rejection")},
	)
	if err == nil || !errors.Is(err, swarmionapp.ErrPublicationRejectedSafeToRetry) ||
		!errors.Is(err, errSwarmionPublicationOutcomeMissing) {
		t.Fatalf("zero-outcome rejection receipt=%+v error=%v, want rejection plus missing-outcome contract failure", receipt, err)
	}
	if IsRetryablePublishedWriteError(err) {
		t.Fatalf("zero-outcome rejection borrowed retry authority: %v", err)
	}
	if receipt.HasExactEventIdentity() || executeCalls != 1 || transactionTestUserCount(t, store, id) != 0 {
		t.Fatalf(
			"zero-outcome rejection receipt=%+v execute_calls=%d row_count=%d, want empty/1/0",
			receipt,
			executeCalls,
			transactionTestUserCount(t, store, id),
		)
	}
}

func TestUnresolvedPublicationReturnsExactNonRetryableReceipt(t *testing.T) {
	store := openPeerTestDB(t)
	runtime := transactionTestRuntime(t, store)
	operation := transactionTestOperation(t, "protos:test:unresolved-publication:v1")
	id := MustNewUUIDv7()

	store.executeOperationForTest = func(ctx context.Context, request swarmionapp.OperationRequest) (swarmionapp.PublicationOutcome, error) {
		accepted, err := runtime.Execute(ctx, request)
		if err != nil {
			return accepted, err
		}
		if accepted.State != swarmionapp.PublicationAccepted || accepted.Receipt == nil {
			return accepted, errors.New("test requires a newly accepted exact receipt")
		}
		unresolved := accepted
		unresolved.State = swarmionapp.PublicationUnresolved
		unresolved.BodyExecuted = false
		unresolved.AlreadyAccepted = false
		return unresolved, &swarmionapp.PublicationUnresolvedError{Outcome: unresolved}
	}
	receipt, err := UpdateAndInsertWithOperationReceiptContext(
		context.Background(),
		store,
		operation,
		nil,
		[]InsertMapper{transactionTestUserInsert(id, "unresolved-but-accepted")},
	)
	if !errors.Is(err, swarmionapp.ErrPublicationUnresolved) {
		t.Fatalf("unresolved publication error=%v", err)
	}
	if !errors.Is(err, ErrPublishedWriteConfirmationUnresolved) {
		t.Fatalf("unresolved publication error=%v, want exact-receipt non-authorizing boundary", err)
	}
	if !receipt.HasExactEventIdentity() || receipt.Committed || !receipt.OutcomeUncertain {
		t.Fatalf("unresolved publication lost its exact uncertain receipt: %+v", receipt)
	}
	if IsRetryablePublishedWriteError(err) {
		t.Fatalf("unresolved publication granted replay authority: %v", err)
	}
	if got := transactionTestUserCount(t, store, id); got != 1 {
		t.Fatalf("accepted unresolved publication row count=%d, want 1", got)
	}
	requireAcceptedOperationResolution(t, store, operation, receipt)
}

func TestUnresolvedExactReceiptRejectsMismatchedRetryMarkers(t *testing.T) {
	tests := []struct {
		name   string
		marker error
		cause  func(swarmionapp.OperationRequest, *swarmionapp.DatabaseRuntime) error
	}{
		{
			name:   "safe rejection",
			marker: swarmionapp.ErrPublicationRejectedSafeToRetry,
			cause: func(request swarmionapp.OperationRequest, runtime *swarmionapp.DatabaseRuntime) error {
				rejected := swarmionapp.PublicationOutcome{
					Identity:        request.Identity,
					Scope:           runtime.PublicationScope(),
					AuthorPeerID:    runtime.PeerID(),
					State:           swarmionapp.PublicationRejectedSafeToRetry,
					RejectionReason: swarmionapp.PublicationRejectionNotAccepted,
				}
				return &swarmionapp.PublicationRejectedError{
					Outcome: rejected,
					Cause:   errors.New("injected mismatched rejection"),
				}
			},
		},
		{
			name:   "stale write context",
			marker: swarmionprotocol.ErrStaleWriteContext,
			cause: func(swarmionapp.OperationRequest, *swarmionapp.DatabaseRuntime) error {
				return swarmionprotocol.ErrStaleWriteContext
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := openPeerTestDB(t)
			runtime := transactionTestRuntime(t, store)
			operation := transactionTestOperation(t, "protos:test:unresolved-mismatched-marker:"+test.name)
			id := MustNewUUIDv7()
			executeCalls := 0

			store.executeOperationForTest = func(ctx context.Context, request swarmionapp.OperationRequest) (swarmionapp.PublicationOutcome, error) {
				executeCalls++
				accepted, err := runtime.Execute(ctx, request)
				if err != nil {
					return accepted, err
				}
				if accepted.State != swarmionapp.PublicationAccepted || accepted.Receipt == nil {
					return accepted, errors.New("test requires a newly accepted exact receipt")
				}
				unresolved := accepted
				unresolved.State = swarmionapp.PublicationUnresolved
				unresolved.BodyExecuted = false
				unresolved.AlreadyAccepted = false
				return unresolved, test.cause(request, runtime)
			}

			receipt, err := UpdateAndInsertWithOperationReceiptContext(
				context.Background(),
				store,
				operation,
				nil,
				[]InsertMapper{transactionTestUserInsert(id, "unresolved-mismatched-marker")},
			)
			if !errors.Is(err, test.marker) || !errors.Is(err, ErrPublishedWriteConfirmationUnresolved) {
				t.Fatalf("unresolved mismatched-marker error=%v, want marker plus exact-receipt boundary", err)
			}
			if IsRetryablePublishedWriteError(err) {
				t.Fatalf("unresolved exact receipt leaked mismatched retry authority: %v", err)
			}
			if !receipt.HasExactEventIdentity() || receipt.Committed || !receipt.OutcomeUncertain ||
				executeCalls != 1 || transactionTestUserCount(t, store, id) != 1 {
				t.Fatalf(
					"unresolved result receipt=%+v execute_calls=%d row_count=%d, want exact uncertain/1/1",
					receipt,
					executeCalls,
					transactionTestUserCount(t, store, id),
				)
			}
		})
	}
}

func TestOrdinaryUnresolvedExactReceiptTracksInsteadOfRetryingStaleMarker(t *testing.T) {
	store := openPeerTestDB(t)
	runtime := transactionTestRuntime(t, store)
	id := MustNewUUIDv7()
	executeCalls := 0

	store.executeOperationForTest = func(ctx context.Context, request swarmionapp.OperationRequest) (swarmionapp.PublicationOutcome, error) {
		executeCalls++
		accepted, err := runtime.Execute(ctx, request)
		if err != nil {
			return accepted, err
		}
		if accepted.State != swarmionapp.PublicationAccepted || accepted.Receipt == nil {
			return accepted, errors.New("test requires a newly accepted exact receipt")
		}
		unresolved := accepted
		unresolved.State = swarmionapp.PublicationUnresolved
		unresolved.BodyExecuted = false
		unresolved.AlreadyAccepted = false
		return unresolved, swarmionprotocol.ErrStaleWriteContext
	}

	receipt, err := InsertWithReceiptContext(
		context.Background(),
		store,
		transactionTestUserInsert(id, "ordinary-unresolved-track-only"),
	)
	if err != nil || !receipt.Committed || receipt.OutcomeUncertain || !receipt.HasExactEventIdentity() {
		t.Fatalf("ordinary unresolved tracking receipt=%+v error=%v, want resolved exact acceptance", receipt, err)
	}
	if executeCalls != 1 || transactionTestUserCount(t, store, id) != 1 {
		t.Fatalf(
			"ordinary unresolved tracking execute_calls=%d row_count=%d, want track-only 1/1",
			executeCalls,
			transactionTestUserCount(t, store, id),
		)
	}
}

func TestInconclusivePublicationRecoversExactReceiptWithoutReexecution(t *testing.T) {
	store := openPeerTestDB(t)
	runtime := transactionTestRuntime(t, store)
	operation := transactionTestOperation(t, "protos:test:inconclusive-publication:v1")
	firstID := MustNewUUIDv7()
	secondID := MustNewUUIDv7()

	store.executeOperationForTest = func(ctx context.Context, request swarmionapp.OperationRequest) (swarmionapp.PublicationOutcome, error) {
		accepted, err := runtime.Execute(ctx, request)
		if err != nil {
			return accepted, err
		}
		if accepted.State != swarmionapp.PublicationAccepted || accepted.Receipt == nil {
			return accepted, errors.New("test requires a newly accepted exact receipt")
		}
		inconclusive := accepted
		inconclusive.State = swarmionapp.PublicationInconclusive
		inconclusive.Receipt = nil
		inconclusive.BodyExecuted = false
		inconclusive.AlreadyAccepted = false
		return inconclusive, &swarmionapp.PublicationInconclusiveError{Outcome: inconclusive}
	}
	recovered, err := UpdateAndInsertWithOperationReceiptContext(
		context.Background(),
		store,
		operation,
		nil,
		[]InsertMapper{transactionTestUserInsert(firstID, "accepted-before-response-loss")},
	)
	if err != nil || !recovered.Committed || !recovered.HasExactEventIdentity() {
		t.Fatalf("inconclusive recovery receipt=%+v error=%v", recovered, err)
	}
	if got := transactionTestUserCount(t, store, firstID); got != 1 {
		t.Fatalf("recovered publication row count=%d, want 1", got)
	}

	store.executeOperationForTest = nil
	replayed, err := UpdateAndInsertWithOperationReceiptContext(
		context.Background(),
		store,
		operation,
		nil,
		[]InsertMapper{transactionTestUserInsert(secondID, "must-not-run-after-recovery")},
	)
	if err != nil || replayed.EventID != recovered.EventID || replayed.PublishedRootHash != recovered.PublishedRootHash {
		t.Fatalf("replayed recovered operation receipt=%+v error=%v, want %+v", replayed, err, recovered)
	}
	if got := transactionTestUserCount(t, store, secondID); got != 0 {
		t.Fatalf("recovered identity executed changed SQL body: count=%d", got)
	}
}

func TestStableNoChangeConsumesIdentityAndReturnsExactReceipt(t *testing.T) {
	store := openPeerTestDB(t)
	operation := transactionTestOperation(t, "protos:test:no-change-operation:v1")
	missingID := MustNewUUIDv7()
	replayID := MustNewUUIDv7()

	first, err := UpdateAndInsertWithOperationReceiptContext(
		context.Background(),
		store,
		operation,
		[]UpdateMapper{transactionTestUserUpdate(missingID, "no-change")},
		nil,
	)
	if err != nil || !first.Committed || !first.HasExactEventIdentity() {
		t.Fatalf("stable no-change receipt=%+v error=%v", first, err)
	}
	if got := transactionTestUserCount(t, store, missingID); got != 0 {
		t.Fatalf("no-change operation created a row: count=%d", got)
	}

	replayed, err := UpdateAndInsertWithOperationReceiptContext(
		context.Background(),
		store,
		operation,
		nil,
		[]InsertMapper{transactionTestUserInsert(replayID, "no-change-body-must-not-replay")},
	)
	if err != nil || replayed.EventID != first.EventID || replayed.PublishedRootHash != first.PublishedRootHash {
		t.Fatalf("replayed no-change receipt=%+v error=%v, want %+v", replayed, err, first)
	}
	if got := transactionTestUserCount(t, store, replayID); got != 0 {
		t.Fatalf("consumed no-change identity executed changed body: count=%d", got)
	}
	requireAcceptedOperationResolution(t, store, operation, first)
}

func TestNoChangeTerminalErrorRetainsExactReceipt(t *testing.T) {
	store := openPeerTestDB(t)
	runtime := transactionTestRuntime(t, store)
	operation := transactionTestOperation(t, "protos:test:no-change-terminal-receipt:v1")
	missingID := MustNewUUIDv7()

	store.executeOperationForTest = func(ctx context.Context, request swarmionapp.OperationRequest) (swarmionapp.PublicationOutcome, error) {
		outcome, err := runtime.Execute(ctx, request)
		if err != nil {
			return outcome, err
		}
		if outcome.State != swarmionapp.PublicationNoChange || outcome.Receipt == nil {
			return outcome, errors.New("test requires a no-change outcome with an exact receipt")
		}
		return outcome, &swarmionapp.DatabaseRuntimeClosedError{}
	}
	receipt, err := UpdateAndInsertWithOperationReceiptContext(
		context.Background(),
		store,
		operation,
		[]UpdateMapper{transactionTestUserUpdate(missingID, "no-change-before-terminal-error")},
		nil,
	)
	if !errors.Is(err, swarmionapp.ErrDatabaseRuntimeClosed) {
		t.Fatalf("no-change terminal error=%v, want database-runtime closure", err)
	}
	if !receipt.Committed || !receipt.HasExactEventIdentity() || receipt.OperationIntentDigest != operation.IntentDigest {
		t.Fatalf("no-change terminal result lost its exact receipt: %+v", receipt)
	}
	if IsRetryablePublishedWriteError(err) {
		t.Fatalf("no-change terminal error granted retry authority: %v", err)
	}
	requireAcceptedOperationResolution(t, store, operation, receipt)
}

func TestOrdinaryWriteRetriesRawStaleContextWithSameIdentity(t *testing.T) {
	store := openPeerTestDB(t)
	runtime := transactionTestRuntime(t, store)
	id := MustNewUUIDv7()
	before := store.TransactionMetrics()
	var (
		executeCalls int
		identity     swarmionapp.OperationIdentity
	)
	store.executeOperationForTest = func(ctx context.Context, request swarmionapp.OperationRequest) (swarmionapp.PublicationOutcome, error) {
		executeCalls++
		if executeCalls == 1 {
			identity = request.Identity
			return swarmionapp.PublicationOutcome{}, swarmionprotocol.ErrStaleWriteContext
		}
		if request.Identity != identity {
			return swarmionapp.PublicationOutcome{}, errors.New("stale-context retry changed the operation identity")
		}
		return runtime.Execute(ctx, request)
	}

	receipt, err := InsertWithReceiptContext(context.Background(), store, transactionTestUserInsert(id, "stale-context-retry"))
	if err != nil || !receipt.Committed || !receipt.HasExactEventIdentity() {
		t.Fatalf("stale-context retry receipt=%+v error=%v", receipt, err)
	}
	if executeCalls != 2 || transactionTestUserCount(t, store, id) != 1 {
		t.Fatalf("stale-context retry calls=%d row_count=%d, want 2/1", executeCalls, transactionTestUserCount(t, store, id))
	}
	delta := transactionMetricsDelta(store.TransactionMetrics(), before)
	if delta.StaleWriteContextOutcomes != 1 || delta.ProjectionTooWideOutcomes != 0 || delta.OperationTransactionLifecycleOpaqueFailures != 0 {
		t.Fatalf("stale-context retry metrics=%+v, want one known non-opaque stale outcome", delta)
	}
}

func TestOrdinaryWriteWaitsForMutationReadinessAfterProjectionTooWide(t *testing.T) {
	store := openPeerTestDB(t)
	runtime := transactionTestRuntime(t, store)
	id := MustNewUUIDv7()
	before := store.TransactionMetrics()
	var (
		executeCalls int
		waitCalls    int
		identity     swarmionapp.OperationIdentity
	)
	store.waitMutationReadyForTest = func(context.Context) error {
		waitCalls++
		return nil
	}
	store.executeOperationForTest = func(ctx context.Context, request swarmionapp.OperationRequest) (swarmionapp.PublicationOutcome, error) {
		executeCalls++
		if executeCalls == 1 {
			identity = request.Identity
			return swarmionapp.PublicationOutcome{}, &swarmionprotocol.ProjectionTooWideError{HeadCount: 9, MaxHeads: 8}
		}
		if request.Identity != identity {
			return swarmionapp.PublicationOutcome{}, errors.New("projection-width retry changed the operation identity")
		}
		return runtime.Execute(ctx, request)
	}

	receipt, err := InsertWithReceiptContext(context.Background(), store, transactionTestUserInsert(id, "projection-width-retry"))
	if err != nil || !receipt.Committed || !receipt.HasExactEventIdentity() {
		t.Fatalf("projection-width retry receipt=%+v error=%v", receipt, err)
	}
	if executeCalls != 2 || waitCalls < 2 || transactionTestUserCount(t, store, id) != 1 {
		t.Fatalf("projection-width retry execute_calls=%d wait_calls=%d row_count=%d, want 2/>=2/1", executeCalls, waitCalls, transactionTestUserCount(t, store, id))
	}
	delta := transactionMetricsDelta(store.TransactionMetrics(), before)
	if delta.StaleWriteContextOutcomes != 1 || delta.ProjectionTooWideOutcomes != 1 || delta.OperationTransactionLifecycleOpaqueFailures != 0 {
		t.Fatalf("projection-width retry metrics=%+v, want one known non-opaque projection outcome", delta)
	}
}

func TestOrdinaryMultiStatementApplyFailureRollsBackAllChanges(t *testing.T) {
	store := openPeerTestDB(t)
	id := MustNewUUIDv7()

	receipt, err := InsertWithReceiptContext(
		context.Background(),
		store,
		transactionTestUserInsert(id, "transaction-atomic-first"),
		transactionTestUserInsert(id, "transaction-atomic-duplicate"),
	)
	if err == nil {
		t.Fatal("duplicate second statement unexpectedly committed")
	}
	if receipt.HasExactEventIdentity() {
		t.Fatalf("failed apply returned a published receipt: %+v", receipt)
	}
	if got := transactionTestUserCount(t, store, id); got != 0 {
		t.Fatalf("partial multi-statement write remained visible: count=%d", got)
	}
}

func TestOrdinaryMissingRowUpdateReturnsNoChange(t *testing.T) {
	store := openPeerTestDB(t)
	missingID := MustNewUUIDv7()

	receipt, err := UpdateWithReceiptContext(
		context.Background(),
		store,
		transactionTestUserUpdate(missingID, "missing-row-noop"),
	)
	if err != nil {
		t.Fatalf("missing-row update: %v", err)
	}
	if receipt.HasExactEventIdentity() {
		t.Fatalf("ordinary no-op update exposed an event receipt: %+v", receipt)
	}
	if got := transactionTestUserCount(t, store, missingID); got != 0 {
		t.Fatalf("no-op update created a row: count=%d", got)
	}
}
