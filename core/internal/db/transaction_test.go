package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bokwoon95/sq"
	swarmionprotocol "github.com/nustiueudinastea/swarmion/protocol"
	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
)

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

func TestOperationTransactionErrorRecordsExactLifecycleMetrics(t *testing.T) {
	tests := []struct {
		name           string
		failure        *swarmionapp.OperationTransactionError
		commitRecorded bool
		known          bool
		want           TransactionMetricsSnapshot
	}{
		{
			name: "begin rejected",
			failure: &swarmionapp.OperationTransactionError{
				Phase:          swarmionapp.OperationTransactionPhaseBegin,
				StatementIndex: -1,
				CommitStatus:   swarmionapp.OperationTransactionCommitNotStarted,
				RollbackStatus: swarmionapp.OperationTransactionRollbackNotAttempted,
				Cause:          swarmionapp.ErrOperationWorkspaceDirty,
			},
			known: true,
		},
		{
			name: "execute rolled back",
			failure: &swarmionapp.OperationTransactionError{
				Phase:          swarmionapp.OperationTransactionPhaseExecute,
				StatementIndex: 0,
				CommitStatus:   swarmionapp.OperationTransactionCommitNotStarted,
				RollbackStatus: swarmionapp.OperationTransactionRollbackSucceeded,
				Cause:          errors.New("statement failed"),
			},
			known: true,
			want: TransactionMetricsSnapshot{
				TransactionsStarted:   1,
				RollbacksAttempted:    1,
				RollbacksSucceeded:    1,
				RollbacksApplyPhase:   1,
				RollbacksApplyFailure: 1,
			},
		},
		{
			name: "execute rollback failed",
			failure: &swarmionapp.OperationTransactionError{
				Phase:          swarmionapp.OperationTransactionPhaseExecute,
				StatementIndex: 0,
				CommitStatus:   swarmionapp.OperationTransactionCommitNotStarted,
				RollbackStatus: swarmionapp.OperationTransactionRollbackFailed,
				Cause:          context.DeadlineExceeded,
				RollbackCause:  errors.New("rollback failed"),
			},
			known: true,
			want: TransactionMetricsSnapshot{
				TransactionsStarted:      1,
				RollbacksAttempted:       1,
				RollbacksFailed:          1,
				RollbacksApplyPhase:      1,
				RollbacksContextDeadline: 1,
			},
		},
		{
			name: "commit returned error",
			failure: &swarmionapp.OperationTransactionError{
				Phase:          swarmionapp.OperationTransactionPhaseCommit,
				StatementIndex: -1,
				CommitStatus:   swarmionapp.OperationTransactionCommitReturnedError,
				RollbackStatus: swarmionapp.OperationTransactionRollbackNotAttempted,
				Cause:          errors.New("commit failed"),
			},
			commitRecorded: true,
			known:          true,
			want: TransactionMetricsSnapshot{
				TransactionsStarted: 1,
				CommitsAttempted:    1,
				CommitsFailed:       1,
			},
		},
		{
			name: "receipt failed after commit",
			failure: &swarmionapp.OperationTransactionError{
				Phase:          swarmionapp.OperationTransactionPhaseReceipt,
				StatementIndex: -1,
				CommitStatus:   swarmionapp.OperationTransactionCommitSucceeded,
				RollbackStatus: swarmionapp.OperationTransactionRollbackNotAttempted,
				Cause:          errors.New("receipt lookup failed"),
			},
			commitRecorded: true,
			known:          true,
			want: TransactionMetricsSnapshot{
				TransactionsStarted: 1,
				CommitsAttempted:    1,
				CommitsSucceeded:    1,
			},
		},
		{
			name: "inconsistent lifecycle is opaque",
			failure: &swarmionapp.OperationTransactionError{
				Phase:          swarmionapp.OperationTransactionPhaseExecute,
				StatementIndex: 0,
				CommitStatus:   swarmionapp.OperationTransactionCommitNotStarted,
				RollbackStatus: swarmionapp.OperationTransactionRollbackNotAttempted,
				Cause:          errors.New("malformed helper result"),
			},
		},
		{
			name: "database sql rollback already done is opaque",
			failure: &swarmionapp.OperationTransactionError{
				Phase:          swarmionapp.OperationTransactionPhaseExecute,
				StatementIndex: 0,
				CommitStatus:   swarmionapp.OperationTransactionCommitNotStarted,
				RollbackStatus: swarmionapp.OperationTransactionRollbackAlreadyDone,
				Cause:          context.Canceled,
			},
			want: TransactionMetricsSnapshot{TransactionsStarted: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &DB{}
			commitRecorded, known := store.recordOperationTransactionFailure(tt.failure)
			if commitRecorded != tt.commitRecorded || known != tt.known {
				t.Fatalf("recorded=(commit=%t known=%t), want (commit=%t known=%t)", commitRecorded, known, tt.commitRecorded, tt.known)
			}
			if got := store.TransactionMetrics(); got != tt.want {
				t.Fatalf("lifecycle metrics=%+v, want %+v", got, tt.want)
			}
		})
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

func transactionTestUserDelete(id string) DeleteMapper {
	return func() sq.DeleteQuery {
		user := sq.New[USER]("")
		return sq.DeleteFrom(user).Where(UUIDEq(user.ID, id))
	}
}

func TestOrdinaryMultiStatementApplyFailureRollsBackAllChanges(t *testing.T) {
	store := openPeerTestDB(t)
	id := MustNewUUIDv7()
	before := store.TransactionMetrics()

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

	delta := transactionMetricsDelta(store.TransactionMetrics(), before)
	if delta.TransactionsStarted != 1 || delta.CommitsAttempted != 0 ||
		delta.RollbacksAttempted != 1 || delta.RollbacksSucceeded != 1 ||
		delta.RollbacksFailed != 0 || delta.RollbacksApplyPhase != 1 ||
		delta.RollbacksBeforeCommitPhase != 0 || delta.RollbacksPanicPhase != 0 ||
		delta.RollbacksApplyFailure != 1 {
		t.Fatalf("apply-failure transaction metrics=%+v", delta)
	}
	t.Logf("fault-injection rollback frequency=%d/%d metrics=%+v", delta.RollbacksAttempted, delta.TransactionsStarted, delta)
}

func TestOrdinaryMissingRowUpdateReturnsTypedNoop(t *testing.T) {
	store := openPeerTestDB(t)
	missingID := MustNewUUIDv7()
	before := store.TransactionMetrics()

	receipt, err := UpdateWithReceiptContext(
		context.Background(),
		store,
		transactionTestUserUpdate(missingID, "missing-row-noop"),
	)
	if err != nil {
		t.Fatalf("missing-row update: %v", err)
	}
	if receipt.HasExactEventIdentity() {
		t.Fatalf("no-op update authored an event: %+v", receipt)
	}
	if got := transactionTestUserCount(t, store, missingID); got != 0 {
		t.Fatalf("no-op update created a row: count=%d", got)
	}

	delta := transactionMetricsDelta(store.TransactionMetrics(), before)
	if delta.TransactionsStarted != 1 || delta.CommitsAttempted != 1 ||
		delta.CommitsSucceeded != 1 || delta.CommitsFailed != 0 ||
		delta.NoopCommitOutcomes != 1 || delta.RollbacksAttempted != 0 {
		t.Fatalf("typed no-op transaction metrics=%+v", delta)
	}
}

type acceptedCommitResponseErrorTransaction struct {
	sqlWriteTransaction
	err error
}

func (tx *acceptedCommitResponseErrorTransaction) Commit() error {
	if err := tx.sqlWriteTransaction.Commit(); err != nil {
		return err
	}
	return tx.err
}

type acceptedCommitUncertainResponseTransaction struct {
	sqlWriteTransaction
	store     *DB
	operation PublishedWriteOperation
}

func (tx *acceptedCommitUncertainResponseTransaction) Commit() error {
	if err := tx.sqlWriteTransaction.Commit(); err != nil {
		return err
	}
	receipt, err := directOperationReceiptLookup(tx.store, context.Background(), tx.operation)
	if err != nil {
		return err
	}
	return &swarmionapp.CommitOutcomeUncertainError{
		EventReceipt: swarmionapp.BranchEventReceiptStatusRequest{
			EventID:                   receipt.EventID,
			ExpectedPublishedRootHash: receipt.PublishedRootHash,
		},
		OperationReceipt: &swarmionapp.BranchOperationReceiptRequest{
			Key:          tx.operation.Key,
			IntentDigest: tx.operation.IntentDigest,
			AuthorPeerID: tx.operation.AuthorPeerID,
		},
		ReceiptPersistence: swarmionapp.CommitReceiptPersistenceUnknown,
		Cause:              errors.New("injected uncertain response after accepted commit"),
	}
}

func directOperationReceiptLookup(
	store *DB,
	ctx context.Context,
	operation PublishedWriteOperation,
) (swarmionapp.BranchOperationReceipt, error) {
	store.mu.Lock()
	runtime := store.runtime
	store.mu.Unlock()
	if runtime == nil {
		return swarmionapp.BranchOperationReceipt{Resolution: swarmionapp.BranchOperationReceiptUnavailable}, nil
	}
	return runtime.ResolveOperationReceipt(ctx, swarmionapp.BranchOperationReceiptRequest{
		Key:          operation.Key,
		IntentDigest: operation.IntentDigest,
		AuthorPeerID: operation.AuthorPeerID,
	})
}

type sqlViewNotReadyOnCallTransaction struct {
	sqlWriteTransaction
	calls      *atomic.Int32
	rejectCall int32
	onReject   func()
}

func (tx *sqlViewNotReadyOnCallTransaction) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if tx.calls.Add(1) == tx.rejectCall {
		if tx.onReject != nil {
			tx.onReject()
		}
		return nil, &swarmionapp.SQLViewNotReadyError{
			LineID:             "main",
			TargetRootHash:     "target",
			VisibleRootHash:    "visible",
			CheckpointRootHash: "checkpoint",
			Reason:             "injected pre-execution readiness deferral",
		}
	}
	return tx.sqlWriteTransaction.ExecContext(ctx, query, args...)
}

type observingWriteTransaction struct {
	sqlWriteTransaction
	onExecError func(string, error)
	calls       *atomic.Int32
}

func (tx *observingWriteTransaction) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if tx.calls != nil {
		tx.calls.Add(1)
	}
	result, err := tx.sqlWriteTransaction.ExecContext(ctx, query, args...)
	if err != nil && tx.onExecError != nil {
		tx.onExecError(query, err)
	}
	return result, err
}

func TestOrdinarySQLViewNotReadyRollsBackWithoutStatementReplay(t *testing.T) {
	store := openPeerTestDB(t)
	firstID := MustNewUUIDv7()
	secondID := MustNewUUIDv7()
	var statementCalls atomic.Int32
	var attempts []PublishedWriteOperation
	store.wrapWriteTransactionForTest = func(tx sqlWriteTransaction, operation PublishedWriteOperation, _ string) sqlWriteTransaction {
		attempts = append(attempts, operation)
		return &sqlViewNotReadyOnCallTransaction{
			sqlWriteTransaction: tx,
			calls:               &statementCalls,
			rejectCall:          2,
		}
	}
	before := store.TransactionMetrics()

	receipt, err := InsertWithReceiptContext(
		context.Background(),
		store,
		transactionTestUserInsert(firstID, "sql-view-readiness-retry-first"),
		transactionTestUserInsert(secondID, "sql-view-readiness-retry-second"),
	)
	if !errors.Is(err, swarmionapp.ErrSQLViewNotReady) {
		t.Fatalf("SQL-view readiness error=%v, want typed ErrSQLViewNotReady", err)
	}
	if receipt.HasExactEventIdentity() ||
		transactionTestUserCount(t, store, firstID) != 0 ||
		transactionTestUserCount(t, store, secondID) != 0 {
		t.Fatalf(
			"readiness rollback receipt=%+v first_count=%d second_count=%d",
			receipt,
			transactionTestUserCount(t, store, firstID),
			transactionTestUserCount(t, store, secondID),
		)
	}
	if len(attempts) != 1 {
		t.Fatalf("readiness retry operations=%+v, want one logical transaction attempt", attempts)
	}
	if statementCalls.Load() != 2 {
		t.Fatalf("statement calls=%d, want first statement + one rejected second statement", statementCalls.Load())
	}

	delta := transactionMetricsDelta(store.TransactionMetrics(), before)
	if delta.TransactionsStarted != 1 || delta.CommitsAttempted != 0 ||
		delta.CommitsSucceeded != 0 || delta.CommitsFailed != 0 ||
		delta.RollbacksAttempted != 1 || delta.RollbacksSucceeded != 1 ||
		delta.RollbacksFailed != 0 || delta.RollbacksSQLViewNotReady != 1 ||
		delta.SQLViewNotReadyOutcomes != 1 || delta.TypedConflicts != 0 {
		t.Fatalf("SQL-view readiness rollback metrics=%+v", delta)
	}
	t.Logf("typed readiness fault-injection statement retries=0 transactions=%d rollbacks=%d metrics=%+v", delta.TransactionsStarted, delta.RollbacksAttempted, delta)
}

type rollbackResponseErrorTransaction struct {
	sqlWriteTransaction
	err error
}

func (tx *rollbackResponseErrorTransaction) Rollback() error {
	if err := tx.sqlWriteTransaction.Rollback(); err != nil {
		return err
	}
	return tx.err
}

func TestSQLViewNotReadyDoesNotRetryWhenRollbackReportsFailure(t *testing.T) {
	store := openPeerTestDB(t)
	id := MustNewUUIDv7()
	var statementCalls atomic.Int32
	var transactionAttempts atomic.Int32
	writeCtx, cancelWrite := context.WithCancel(context.Background())
	defer cancelWrite()
	store.wrapWriteTransactionForTest = func(tx sqlWriteTransaction, _ PublishedWriteOperation, _ string) sqlWriteTransaction {
		transactionAttempts.Add(1)
		viewNotReady := &sqlViewNotReadyOnCallTransaction{
			sqlWriteTransaction: tx,
			calls:               &statementCalls,
			rejectCall:          1,
			onReject:            cancelWrite,
		}
		return &rollbackResponseErrorTransaction{
			sqlWriteTransaction: viewNotReady,
			err:                 errors.New("injected rollback response failure"),
		}
	}
	before := store.TransactionMetrics()

	receipt, err := InsertWithReceiptContext(
		writeCtx,
		store,
		transactionTestUserInsert(id, "must-not-retry-after-rollback-failure"),
	)
	if err == nil || IsRetryablePublishedWriteError(err) {
		t.Fatalf("rollback-failure error=%v, want non-retryable failure", err)
	}
	if receipt.HasExactEventIdentity() || transactionAttempts.Load() != 1 || statementCalls.Load() != 1 {
		t.Fatalf("rollback-failure receipt=%+v transactions=%d statements=%d", receipt, transactionAttempts.Load(), statementCalls.Load())
	}
	if got := transactionTestUserCount(t, store, id); got != 0 {
		t.Fatalf("rollback-failure transaction left partial content: count=%d", got)
	}

	delta := transactionMetricsDelta(store.TransactionMetrics(), before)
	if delta.TransactionsStarted != 1 || delta.CommitsAttempted != 0 ||
		delta.RollbacksAttempted != 1 || delta.RollbacksSucceeded != 0 ||
		delta.RollbacksFailed != 1 || delta.RollbacksApplyPhase != 1 ||
		delta.RollbacksBeforeCommitPhase != 0 || delta.RollbacksPanicPhase != 0 ||
		delta.RollbacksSQLViewNotReady != 1 ||
		delta.SQLViewNotReadyOutcomes != 1 {
		t.Fatalf("rollback-failure readiness metrics=%+v", delta)
	}
}

func TestStableOperationExecutedThenAlreadyAcceptedSkipsChangedBody(t *testing.T) {
	store := openPeerTestDB(t)
	operationKey, err := NewPublishedWriteOperationKey()
	if err != nil {
		t.Fatal(err)
	}
	operation, err := NewPublishedWriteOperation(operationKey, "protos:test:operation-outcomes:v1")
	if err != nil {
		t.Fatal(err)
	}
	firstID := MustNewUUIDv7()
	secondID := MustNewUUIDv7()
	var outcomes []swarmionapp.OperationTransactionOutcome
	store.runOperationTransactionForTest = func(
		ctx context.Context,
		sqldb *sql.DB,
		request swarmionapp.OperationTransactionRequest,
	) (swarmionapp.OperationTransactionResult, error) {
		if request.Operation.Key != operation.Key || request.Operation.IntentDigest != operation.IntentDigest {
			t.Fatalf("operation transaction request=%+v, want key=%q intent=%q", request.Operation, operation.Key, operation.IntentDigest)
		}
		if request.NoChangePolicy != swarmionapp.OperationNoChangePolicyRecordReceipt {
			t.Fatalf("operation no-change policy=%q, want record_receipt", request.NoChangePolicy)
		}
		result, err := swarmionapp.RunOperationTransaction(ctx, sqldb, request)
		if err == nil {
			outcomes = append(outcomes, result.Outcome)
		}
		return result, err
	}
	before := store.TransactionMetrics()

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
		t.Fatalf("executed operation receipt=%+v count=%d", first, transactionTestUserCount(t, store, firstID))
	}

	repeated, err := UpdateAndInsertWithOperationReceiptContext(
		context.Background(),
		store,
		operation,
		nil,
		[]InsertMapper{transactionTestUserInsert(secondID, "already-accepted-body-must-not-run")},
	)
	if err != nil {
		t.Fatalf("repeat stable operation: %v", err)
	}
	if repeated.EventID != first.EventID || repeated.PublishedRootHash != first.PublishedRootHash {
		t.Fatalf("already-accepted receipt=%+v, want original %+v", repeated, first)
	}
	if got := transactionTestUserCount(t, store, secondID); got != 0 {
		t.Fatalf("already-accepted operation executed changed SQL body: count=%d", got)
	}
	if len(outcomes) != 2 ||
		outcomes[0] != swarmionapp.OperationTransactionOutcomeExecuted ||
		outcomes[1] != swarmionapp.OperationTransactionOutcomeAlreadyAccepted {
		t.Fatalf("operation outcomes=%v, want [executed already_accepted]", outcomes)
	}

	delta := transactionMetricsDelta(store.TransactionMetrics(), before)
	if delta.TransactionsStarted != 2 || delta.CommitsAttempted != 2 ||
		delta.CommitsSucceeded != 2 || delta.CommitsFailed != 0 ||
		delta.RollbacksAttempted != 0 || delta.OperationTransactionsAttempted != 2 ||
		delta.OperationTransactionsExecuted != 1 || delta.OperationTransactionsAlreadyAccepted != 1 ||
		delta.OperationTransactionsNoChange != 0 || delta.OperationTransactionsFailed != 0 {
		t.Fatalf("stable operation outcome metrics=%+v", delta)
	}
}

func TestStableOperationStatementFailureReportsExactRollbackLifecycle(t *testing.T) {
	store := openPeerTestDB(t)
	operationKey, err := NewPublishedWriteOperationKey()
	if err != nil {
		t.Fatal(err)
	}
	operation, err := NewPublishedWriteOperation(operationKey, "protos:test:operation-apply-failure-metrics:v1")
	if err != nil {
		t.Fatal(err)
	}
	id := MustNewUUIDv7()
	before := store.TransactionMetrics()

	receipt, err := UpdateAndInsertWithOperationReceiptContext(
		context.Background(),
		store,
		operation,
		nil,
		[]InsertMapper{
			transactionTestUserInsert(id, "operation-atomic-first"),
			transactionTestUserInsert(id, "operation-atomic-duplicate"),
		},
	)
	if err == nil {
		t.Fatal("duplicate operation statement unexpectedly committed")
	}
	var failure *swarmionapp.OperationTransactionError
	if !errors.As(err, &failure) || failure == nil ||
		failure.Phase != swarmionapp.OperationTransactionPhaseExecute ||
		failure.StatementIndex != 1 ||
		failure.CommitStatus != swarmionapp.OperationTransactionCommitNotStarted ||
		failure.RollbackStatus != swarmionapp.OperationTransactionRollbackSucceeded {
		t.Fatalf("operation statement failure lifecycle=%+v error=%v", failure, err)
	}
	if receipt.HasExactEventIdentity() || IsRetryablePublishedWriteError(err) {
		t.Fatalf("failed operation receipt=%+v retryable=%t error=%v", receipt, IsRetryablePublishedWriteError(err), err)
	}
	if got := transactionTestUserCount(t, store, id); got != 0 {
		t.Fatalf("operation helper left partial multi-statement content: count=%d", got)
	}
	resolved, lookupErr := directOperationReceiptLookup(store, context.Background(), operation)
	if lookupErr != nil || resolved.Resolution != swarmionapp.BranchOperationReceiptAbsent {
		t.Fatalf("failed operation receipt resolution=%+v error=%v, want absent", resolved, lookupErr)
	}

	delta := transactionMetricsDelta(store.TransactionMetrics(), before)
	if delta.TransactionsStarted != 1 || delta.CommitsAttempted != 0 ||
		delta.RollbacksAttempted != 1 || delta.RollbacksSucceeded != 1 || delta.RollbacksFailed != 0 ||
		delta.RollbacksApplyPhase != 1 || delta.RollbacksApplyFailure != 1 ||
		delta.OperationTransactionsAttempted != 1 || delta.OperationTransactionsFailed != 1 ||
		delta.OperationTransactionLifecycleOpaqueFailures != 0 {
		t.Fatalf("operation statement-failure lifecycle metrics=%+v", delta)
	}
	t.Logf(
		"operation helper rollback lifecycle: known_rollbacks=%d known_transactions=%d opaque_lifecycles=%d",
		delta.RollbacksAttempted,
		delta.TransactionsStarted,
		delta.OperationTransactionLifecycleOpaqueFailures,
	)
}

func TestStableOperationRollbackAlreadyDoneDoesNotFabricateRollbackOutcome(t *testing.T) {
	store := openPeerTestDB(t)
	operationKey, err := NewPublishedWriteOperationKey()
	if err != nil {
		t.Fatal(err)
	}
	operation, err := NewPublishedWriteOperation(operationKey, "protos:test:operation-rollback-already-done:v1")
	if err != nil {
		t.Fatal(err)
	}
	store.runOperationTransactionForTest = func(
		context.Context,
		*sql.DB,
		swarmionapp.OperationTransactionRequest,
	) (swarmionapp.OperationTransactionResult, error) {
		return swarmionapp.OperationTransactionResult{}, &swarmionapp.OperationTransactionError{
			Phase:          swarmionapp.OperationTransactionPhaseExecute,
			StatementIndex: 0,
			CommitStatus:   swarmionapp.OperationTransactionCommitNotStarted,
			RollbackStatus: swarmionapp.OperationTransactionRollbackAlreadyDone,
			Cause:          context.Canceled,
		}
	}
	before := store.TransactionMetrics()

	receipt, err := UpdateAndInsertWithOperationReceiptContext(
		context.Background(),
		store,
		operation,
		nil,
		[]InsertMapper{transactionTestUserInsert(MustNewUUIDv7(), "operation-body-not-run")},
	)
	if err == nil || !errors.Is(err, context.Canceled) || receipt.HasExactEventIdentity() {
		t.Fatalf("rollback-already-done receipt=%+v error=%v", receipt, err)
	}
	resolved, lookupErr := directOperationReceiptLookup(store, context.Background(), operation)
	if lookupErr != nil || resolved.Resolution != swarmionapp.BranchOperationReceiptAbsent {
		t.Fatalf("rollback-already-done receipt resolution=%+v error=%v, want absent", resolved, lookupErr)
	}

	delta := transactionMetricsDelta(store.TransactionMetrics(), before)
	if delta.TransactionsStarted != 1 || delta.CommitsAttempted != 0 ||
		delta.RollbacksAttempted != 0 || delta.RollbacksSucceeded != 0 || delta.RollbacksFailed != 0 ||
		delta.OperationTransactionsAttempted != 1 || delta.OperationTransactionsFailed != 1 ||
		delta.OperationTransactionLifecycleOpaqueFailures != 1 {
		t.Fatalf("rollback-already-done lifecycle metrics=%+v", delta)
	}
}

func TestStableOperationRecoveredPostCommitErrorIsNotCountedAsRollbackOpaque(t *testing.T) {
	store := openPeerTestDB(t)
	operationKey, err := NewPublishedWriteOperationKey()
	if err != nil {
		t.Fatal(err)
	}
	operation, err := NewPublishedWriteOperation(operationKey, "protos:test:operation-post-commit-error-metrics:v1")
	if err != nil {
		t.Fatal(err)
	}
	id := MustNewUUIDv7()
	var runnerCalls atomic.Int32
	store.runOperationTransactionForTest = func(
		ctx context.Context,
		sqldb *sql.DB,
		request swarmionapp.OperationTransactionRequest,
	) (swarmionapp.OperationTransactionResult, error) {
		result, err := swarmionapp.RunOperationTransaction(ctx, sqldb, request)
		if err != nil {
			return result, err
		}
		runnerCalls.Add(1)
		return swarmionapp.OperationTransactionResult{}, errors.New("injected response loss after accepted operation")
	}
	before := store.TransactionMetrics()

	receipt, err := UpdateAndInsertWithOperationReceiptContext(
		context.Background(),
		store,
		operation,
		nil,
		[]InsertMapper{transactionTestUserInsert(id, "operation-accepted-before-response-loss")},
	)
	if err != nil || !receipt.HasExactEventIdentity() || runnerCalls.Load() != 1 {
		t.Fatalf("recovered post-commit operation receipt=%+v runner_calls=%d error=%v", receipt, runnerCalls.Load(), err)
	}
	if got := transactionTestUserCount(t, store, id); got != 1 {
		t.Fatalf("accepted operation row count=%d, want 1", got)
	}

	delta := transactionMetricsDelta(store.TransactionMetrics(), before)
	if delta.TransactionsStarted != 1 || delta.CommitsAttempted != 1 || delta.CommitsSucceeded != 1 ||
		delta.CommitsFailed != 0 || delta.RollbacksAttempted != 0 ||
		delta.OperationTransactionsAttempted != 1 || delta.OperationTransactionsFailed != 1 ||
		delta.OperationReceiptsFoundAfterCommitErr != 1 ||
		delta.OperationTransactionLifecycleOpaqueFailures != 0 {
		t.Fatalf("recovered post-commit lifecycle metrics=%+v", delta)
	}
}

func TestOrdinarySuccessfulCommitReturnsBeforeReceiptLookup(t *testing.T) {
	store := openPeerTestDB(t)
	id := MustNewUUIDv7()
	var lookupCalls atomic.Int32
	store.lookupPublishedWriteForTest = func(ctx context.Context, operation PublishedWriteOperation) (swarmionapp.BranchOperationReceipt, error) {
		call := lookupCalls.Add(1)
		if call > 2 {
			return swarmionapp.BranchOperationReceipt{}, errors.New("injected post-commit receipt lookup failure")
		}
		return directOperationReceiptLookup(store, ctx, operation)
	}
	before := store.TransactionMetrics()

	err := InsertPublishedContext(
		context.Background(),
		store,
		transactionTestUserInsert(id, "ordinary-commit-does-not-wait-for-receipt"),
	)
	if err != nil {
		t.Fatalf("ordinary successful commit depended on receipt lookup: %v", err)
	}
	if calls := lookupCalls.Load(); calls != 2 {
		t.Fatalf("ordinary operation receipt lookups=%d, want only two pre-commit absence checks", calls)
	}
	if got := transactionTestUserCount(t, store, id); got != 1 {
		t.Fatalf("ordinary committed row count=%d, want 1", got)
	}
	delta := transactionMetricsDelta(store.TransactionMetrics(), before)
	if delta.TransactionsStarted != 1 || delta.CommitsAttempted != 1 || delta.CommitsSucceeded != 1 ||
		delta.CommitsFailed != 0 || delta.RollbacksAttempted != 0 {
		t.Fatalf("ordinary immediate commit metrics=%+v", delta)
	}
}

func TestOrdinaryExactUncertainCommitOutcomeDoesNotEscapeAsReplaySignal(t *testing.T) {
	store := openPeerTestDB(t)
	id := MustNewUUIDv7()
	var wrapped atomic.Bool
	store.wrapWriteTransactionForTest = func(tx sqlWriteTransaction, operation PublishedWriteOperation, _ string) sqlWriteTransaction {
		if !wrapped.CompareAndSwap(false, true) {
			return tx
		}
		return &acceptedCommitUncertainResponseTransaction{
			sqlWriteTransaction: tx,
			store:               store,
			operation:           operation,
		}
	}
	var lookupCalls atomic.Int32
	store.lookupPublishedWriteForTest = func(ctx context.Context, operation PublishedWriteOperation) (swarmionapp.BranchOperationReceipt, error) {
		call := lookupCalls.Add(1)
		if call == 3 {
			return swarmionapp.BranchOperationReceipt{}, errors.New("injected unavailable lookup after uncertain commit response")
		}
		return directOperationReceiptLookup(store, ctx, operation)
	}
	before := store.TransactionMetrics()

	err := InsertPublishedContext(
		context.Background(),
		store,
		transactionTestUserInsert(id, "ordinary-uncertain-exact-receipt"),
	)
	if err != nil {
		t.Fatalf("ordinary exact uncertain outcome escaped as a replay signal: %v", err)
	}
	if calls := lookupCalls.Load(); calls != 3 {
		t.Fatalf("ordinary uncertain outcome lookup calls=%d, want 3", calls)
	}
	if got := transactionTestUserCount(t, store, id); got != 1 {
		t.Fatalf("ordinary uncertain accepted row count=%d, want 1", got)
	}
	delta := transactionMetricsDelta(store.TransactionMetrics(), before)
	if delta.TransactionsStarted != 1 || delta.CommitsAttempted != 1 || delta.CommitsSucceeded != 0 ||
		delta.CommitsFailed != 1 || delta.UncertainEventReceiptsAfterCommitErr != 1 || delta.RollbacksAttempted != 0 {
		t.Fatalf("ordinary uncertain exact receipt metrics=%+v", delta)
	}
}

func TestOrdinaryFoundOperationReceiptFromUncertainCommitStillProvesPublication(t *testing.T) {
	store := openPeerTestDB(t)
	id := MustNewUUIDv7()
	var wrapped atomic.Bool
	store.wrapWriteTransactionForTest = func(tx sqlWriteTransaction, operation PublishedWriteOperation, _ string) sqlWriteTransaction {
		if !wrapped.CompareAndSwap(false, true) {
			return tx
		}
		return &acceptedCommitUncertainResponseTransaction{
			sqlWriteTransaction: tx,
			store:               store,
			operation:           operation,
		}
	}
	store.observePublishedWriteForTest = func(_ context.Context, receipt PublishedWriteReceipt) (EventReceiptObservation, error) {
		return EventReceiptObservation{
			Receipt: receipt,
			State:   EventReceiptStatePending,
		}, errors.New("injected exact receipt remains unknown after operation lookup found")
	}
	before := store.TransactionMetrics()

	err := InsertPublishedContext(
		context.Background(),
		store,
		transactionTestUserInsert(id, "ordinary-uncertain-found-operation-receipt"),
	)
	if err == nil {
		t.Fatal("found operation receipt bypassed local-publication proof")
	}
	var pending *EventReceiptPendingError
	if !errors.As(err, &pending) || pending == nil || !pending.Observation.Receipt.HasExactEventIdentity() {
		t.Fatalf("found uncertain operation did not retain its exact receipt: %v", err)
	}
	if pending.Observation.Receipt.Committed || !pending.Observation.Receipt.OutcomeUncertain {
		t.Fatalf("found uncertain operation was mislabeled locally published: %+v", pending.Observation.Receipt)
	}
	if IsRetryablePublishedWriteError(err) {
		t.Fatalf("found uncertain operation escaped as replay-safe: %v", err)
	}
	if got := transactionTestUserCount(t, store, id); got != 1 {
		t.Fatalf("accepted row count after uncertain found receipt=%d, want 1", got)
	}
	delta := transactionMetricsDelta(store.TransactionMetrics(), before)
	if delta.TransactionsStarted != 1 || delta.CommitsAttempted != 1 || delta.CommitsSucceeded != 0 ||
		delta.CommitsFailed != 1 || delta.OperationReceiptsFoundAfterCommitErr != 1 ||
		delta.UncertainEventReceiptsAfterCommitErr != 1 || delta.RollbacksAttempted != 0 {
		t.Fatalf("ordinary uncertain found-receipt metrics=%+v", delta)
	}
}

func TestOrdinaryWithReceiptAPIsRequireKnownUncertainReceiptWithoutReplay(t *testing.T) {
	type receiptCall func(context.Context) (PublishedWriteReceipt, error)
	tests := []struct {
		name  string
		build func(*testing.T, *DB, string) receiptCall
	}{
		{
			name: "insert",
			build: func(_ *testing.T, store *DB, label string) receiptCall {
				id := MustNewUUIDv7()
				return func(ctx context.Context) (PublishedWriteReceipt, error) {
					return InsertWithReceiptContext(ctx, store, transactionTestUserInsert(id, label))
				}
			},
		},
		{
			name: "update",
			build: func(t *testing.T, store *DB, label string) receiptCall {
				id := MustNewUUIDv7()
				if _, err := InsertWithReceiptContext(context.Background(), store, transactionTestUserInsert(id, label+"-seed")); err != nil {
					t.Fatalf("seed update target: %v", err)
				}
				return func(ctx context.Context) (PublishedWriteReceipt, error) {
					return UpdateWithReceiptContext(ctx, store, transactionTestUserUpdate(id, label))
				}
			},
		},
		{
			name: "update_and_insert",
			build: func(t *testing.T, store *DB, label string) receiptCall {
				updateID := MustNewUUIDv7()
				insertID := MustNewUUIDv7()
				if _, err := InsertWithReceiptContext(context.Background(), store, transactionTestUserInsert(updateID, label+"-seed")); err != nil {
					t.Fatalf("seed update-and-insert target: %v", err)
				}
				return func(ctx context.Context) (PublishedWriteReceipt, error) {
					return UpdateAndInsertWithReceiptContext(
						ctx,
						store,
						[]UpdateMapper{transactionTestUserUpdate(updateID, label)},
						[]InsertMapper{transactionTestUserInsert(insertID, label+"-insert")},
					)
				}
			},
		},
		{
			name: "delete",
			build: func(t *testing.T, store *DB, label string) receiptCall {
				id := MustNewUUIDv7()
				if _, err := InsertWithReceiptContext(context.Background(), store, transactionTestUserInsert(id, label+"-seed")); err != nil {
					t.Fatalf("seed delete target: %v", err)
				}
				return func(ctx context.Context) (PublishedWriteReceipt, error) {
					return DeleteWithReceiptContext(ctx, store, transactionTestUserDelete(id))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := openPeerTestDB(t)
			for _, resolves := range []bool{false, true} {
				mode := "pending"
				if resolves {
					mode = "resolves"
				}
				t.Run(mode, func(t *testing.T) {
					call := tt.build(t, store, "ordinary-with-receipt-"+tt.name+"-"+mode)
					before := store.TransactionMetrics()
					beforeSequence := testRuntimeSnapshot(t, store).LocalAuthorSeq
					var commitCalls atomic.Int32
					store.wrapWriteTransactionForTest = func(tx sqlWriteTransaction, operation PublishedWriteOperation, _ string) sqlWriteTransaction {
						commitCalls.Add(1)
						return &acceptedCommitUncertainResponseTransaction{
							sqlWriteTransaction: tx,
							store:               store,
							operation:           operation,
						}
					}
					var observationCalls atomic.Int32
					ctx := context.Background()
					if !resolves {
						var cancel context.CancelFunc
						ctx, cancel = context.WithCancel(ctx)
						defer cancel()
						store.observePublishedWriteForTest = func(_ context.Context, receipt PublishedWriteReceipt) (EventReceiptObservation, error) {
							observationCalls.Add(1)
							cancel()
							return EventReceiptObservation{Receipt: receipt, State: EventReceiptStatePending}, nil
						}
					} else {
						store.observePublishedWriteForTest = func(_ context.Context, receipt PublishedWriteReceipt) (EventReceiptObservation, error) {
							known := observationCalls.Add(1) >= 2
							return EventReceiptObservation{
								Receipt: receipt,
								Status:  swarmionapp.BranchEventReceiptStatus{Known: known},
								State:   EventReceiptStatePending,
							}, nil
						}
					}

					receipt, err := call(ctx)
					store.wrapWriteTransactionForTest = nil
					store.observePublishedWriteForTest = nil
					if !receipt.HasExactEventIdentity() {
						t.Fatalf("%s receipt lost exact event identity: %+v error=%v", mode, receipt, err)
					}
					if commitCalls.Load() != 1 {
						t.Fatalf("%s transaction attempts=%d, want exactly one", mode, commitCalls.Load())
					}
					afterSnapshot := testRuntimeSnapshot(t, store)
					if afterSnapshot.LocalAuthorSeq != beforeSequence+1 {
						t.Fatalf("%s author sequence=%d, want %d (one event)", mode, afterSnapshot.LocalAuthorSeq, beforeSequence+1)
					}
					// Checkpointing may retire an accepted event from the moving
					// tentative clock before this assertion runs. The exact receipt
					// status remains authoritative across that transition.
					acceptedStatus, statusErr := store.SwarmionEventReceiptStatus(
						context.Background(),
						receipt.EventID,
						receipt.PublishedRootHash,
					)
					if statusErr != nil || !acceptedStatus.Known {
						t.Fatalf("%s exact event status=%+v error=%v, want known accepted receipt", mode, acceptedStatus, statusErr)
					}
					delta := transactionMetricsDelta(store.TransactionMetrics(), before)
					if delta.TransactionsStarted != 1 || delta.CommitsAttempted != 1 || delta.CommitsFailed != 1 ||
						delta.OperationReceiptsFoundAfterCommitErr != 1 || delta.RollbacksAttempted != 0 {
						t.Fatalf("%s transaction metrics=%+v, want one accepted event without replay/rollback", mode, delta)
					}

					if !resolves {
						var pending *EventReceiptPendingError
						if !errors.As(err, &pending) || pending == nil || !errors.Is(err, ErrEventReceiptPending) {
							t.Fatalf("unknown exact receipt error=%v, want typed pending", err)
						}
						if pending.Observation.Receipt.EventID != receipt.EventID ||
							pending.Observation.Receipt.PublishedRootHash != receipt.PublishedRootHash {
							t.Fatalf("pending error lost exact receipt: %+v, return=%+v", pending.Observation.Receipt, receipt)
						}
						if receipt.Committed || !receipt.OutcomeUncertain || IsRetryablePublishedWriteError(err) {
							t.Fatalf("pending receipt/error incorrectly classified: receipt=%+v error=%v", receipt, err)
						}

						// Status resolution resumes this exact receipt and does not execute
						// another SQL transaction or author another event.
						resumeMetrics := store.TransactionMetrics()
						resumeSequence := afterSnapshot.LocalAuthorSeq
						store.observePublishedWriteForTest = func(_ context.Context, candidate PublishedWriteReceipt) (EventReceiptObservation, error) {
							return EventReceiptObservation{
								Receipt: candidate,
								Status:  swarmionapp.BranchEventReceiptStatus{Known: true},
								State:   EventReceiptStatePending,
							}, nil
						}
						if waitErr := store.waitForOrdinaryPublishedWriteKnown(context.Background(), receipt); waitErr != nil {
							t.Fatalf("resume exact pending receipt: %v", waitErr)
						}
						store.observePublishedWriteForTest = nil
						if testRuntimeSnapshot(t, store).LocalAuthorSeq != resumeSequence || store.TransactionMetrics() != resumeMetrics {
							t.Fatal("resolving the exact pending receipt reexecuted the mutation")
						}
					} else {
						if err != nil {
							t.Fatalf("known receipt returned error: %v", err)
						}
						if !receipt.Committed || receipt.OutcomeUncertain || observationCalls.Load() != 2 {
							t.Fatalf("known receipt was not resolved before success: receipt=%+v observations=%d", receipt, observationCalls.Load())
						}
					}
				})
			}
		})
	}
}

func TestOrdinaryUncertainAndOperationReceiptIdentityMismatchFailsClosed(t *testing.T) {
	store := openPeerTestDB(t)
	otherID := MustNewUUIDv7()
	otherReceipt, err := InsertWithReceiptContext(
		context.Background(),
		store,
		transactionTestUserInsert(otherID, "ordinary-uncertain-mismatched-receipt-source"),
	)
	if err != nil {
		t.Fatalf("publish mismatched receipt source: %v", err)
	}

	id := MustNewUUIDv7()
	var wrapped atomic.Bool
	store.wrapWriteTransactionForTest = func(tx sqlWriteTransaction, operation PublishedWriteOperation, _ string) sqlWriteTransaction {
		if !wrapped.CompareAndSwap(false, true) {
			return tx
		}
		return &acceptedCommitUncertainResponseTransaction{
			sqlWriteTransaction: tx,
			store:               store,
			operation:           operation,
		}
	}
	var lookupCalls atomic.Int32
	store.lookupPublishedWriteForTest = func(ctx context.Context, operation PublishedWriteOperation) (swarmionapp.BranchOperationReceipt, error) {
		if lookupCalls.Add(1) == 3 {
			return swarmionapp.BranchOperationReceipt{
				Resolution:        swarmionapp.BranchOperationReceiptFound,
				EventID:           otherReceipt.EventID,
				PublishedRootHash: otherReceipt.PublishedRootHash,
				AuthorPeerID:      operation.AuthorPeerID,
				IntentDigest:      operation.IntentDigest,
			}, nil
		}
		return directOperationReceiptLookup(store, ctx, operation)
	}
	before := store.TransactionMetrics()

	err = InsertPublishedContext(
		context.Background(),
		store,
		transactionTestUserInsert(id, "ordinary-uncertain-mismatched-receipt-target"),
	)
	if err == nil || !errors.Is(err, ErrPublishedWriteReceiptIdentityConflict) {
		t.Fatalf("mismatched exact receipts error=%v, want identity conflict", err)
	}
	var conflict *PublishedWriteReceiptIdentityConflictError
	if !errors.As(err, &conflict) || conflict == nil || !conflict.Receipt.HasExactEventIdentity() {
		t.Fatalf("mismatched exact receipts did not retain uncertain identity: %v", err)
	}
	if conflict.Receipt.EventID == otherReceipt.EventID ||
		conflict.ResolvedEventID != otherReceipt.EventID ||
		conflict.ResolvedPublishedRootHash != otherReceipt.PublishedRootHash {
		t.Fatalf("identity conflict diagnostics=%+v, want distinct uncertain and resolved receipts", conflict)
	}
	if IsRetryablePublishedWriteError(err) {
		t.Fatalf("receipt identity conflict escaped as replay-safe: %v", err)
	}
	if got := transactionTestUserCount(t, store, id); got != 1 {
		t.Fatalf("accepted target row count after identity conflict=%d, want 1", got)
	}
	delta := transactionMetricsDelta(store.TransactionMetrics(), before)
	if delta.TransactionsStarted != 1 || delta.CommitsAttempted != 1 || delta.CommitsSucceeded != 0 ||
		delta.CommitsFailed != 1 || delta.RollbacksAttempted != 0 {
		t.Fatalf("ordinary mismatched-receipt metrics=%+v", delta)
	}
}

func TestOrdinaryUnresolvedUncertainReceiptReturnsNonRetryableTrackingError(t *testing.T) {
	store := openPeerTestDB(t)
	id := MustNewUUIDv7()
	var wrapped atomic.Bool
	store.wrapWriteTransactionForTest = func(tx sqlWriteTransaction, operation PublishedWriteOperation, _ string) sqlWriteTransaction {
		if !wrapped.CompareAndSwap(false, true) {
			return tx
		}
		return &acceptedCommitUncertainResponseTransaction{
			sqlWriteTransaction: tx,
			store:               store,
			operation:           operation,
		}
	}
	var lookupCalls atomic.Int32
	store.lookupPublishedWriteForTest = func(ctx context.Context, operation PublishedWriteOperation) (swarmionapp.BranchOperationReceipt, error) {
		call := lookupCalls.Add(1)
		if call == 3 {
			return swarmionapp.BranchOperationReceipt{}, errors.New("injected unavailable lookup after uncertain commit response")
		}
		return directOperationReceiptLookup(store, ctx, operation)
	}
	store.observePublishedWriteForTest = func(_ context.Context, receipt PublishedWriteReceipt) (EventReceiptObservation, error) {
		return EventReceiptObservation{
			Receipt: receipt,
			State:   EventReceiptStatePending,
		}, errors.New("injected exact receipt status unavailable")
	}
	before := store.TransactionMetrics()

	err := InsertPublishedContext(
		context.Background(),
		store,
		transactionTestUserInsert(id, "ordinary-uncertain-unresolved-receipt"),
	)
	if err == nil {
		t.Fatal("ordinary write reported success without proving local publication")
	}
	var pending *EventReceiptPendingError
	if !errors.As(err, &pending) || pending == nil || !pending.Observation.Receipt.HasExactEventIdentity() {
		t.Fatalf("ordinary unresolved outcome did not retain its exact receipt: %v", err)
	}
	if !errors.Is(err, ErrEventReceiptPending) {
		t.Fatalf("ordinary unresolved outcome=%v, want typed pending tracking error", err)
	}
	if IsRetryablePublishedWriteError(err) {
		t.Fatalf("ordinary unresolved exact receipt escaped as replay-safe: %v", err)
	}
	if calls := lookupCalls.Load(); calls != 3 {
		t.Fatalf("ordinary unresolved outcome lookup calls=%d, want 3", calls)
	}
	if got := transactionTestUserCount(t, store, id); got != 1 {
		t.Fatalf("accepted row count after unresolved response=%d, want 1", got)
	}
	delta := transactionMetricsDelta(store.TransactionMetrics(), before)
	if delta.TransactionsStarted != 1 || delta.CommitsAttempted != 1 || delta.CommitsSucceeded != 0 ||
		delta.CommitsFailed != 1 || delta.UncertainEventReceiptsAfterCommitErr != 1 || delta.RollbacksAttempted != 0 {
		t.Fatalf("ordinary unresolved exact receipt metrics=%+v", delta)
	}
}

func TestStableOperationResponseLossAndLookupFailureRecoversWithoutDuplicate(t *testing.T) {
	store := openPeerTestDB(t)
	operationKey, err := NewPublishedWriteOperationKey()
	if err != nil {
		t.Fatal(err)
	}
	operation, err := NewPublishedWriteOperation(operationKey, "protos:test:response-loss-lookup-failure:v1")
	if err != nil {
		t.Fatal(err)
	}
	firstID := MustNewUUIDv7()
	secondID := MustNewUUIDv7()
	var runnerCalls atomic.Int32
	var acceptedResult swarmionapp.OperationTransactionResult
	store.runOperationTransactionForTest = func(
		ctx context.Context,
		sqldb *sql.DB,
		request swarmionapp.OperationTransactionRequest,
	) (swarmionapp.OperationTransactionResult, error) {
		result, err := swarmionapp.RunOperationTransaction(ctx, sqldb, request)
		if err != nil {
			return result, err
		}
		if runnerCalls.Add(1) == 1 {
			acceptedResult = result
			return swarmionapp.OperationTransactionResult{}, errors.New("injected response loss after accepted operation transaction")
		}
		return result, nil
	}
	var lookupCalls atomic.Int32
	store.lookupPublishedWriteForTest = func(context.Context, PublishedWriteOperation) (swarmionapp.BranchOperationReceipt, error) {
		lookupCalls.Add(1)
		return swarmionapp.BranchOperationReceipt{}, errors.New("injected receipt lookup failure after response loss")
	}
	before := store.TransactionMetrics()

	accepted, err := UpdateAndInsertWithOperationReceiptContext(
		context.Background(),
		store,
		operation,
		nil,
		[]InsertMapper{transactionTestUserInsert(firstID, "accepted-before-response-loss")},
	)
	if err == nil || accepted.Committed || accepted.HasExactEventIdentity() {
		t.Fatalf("unresolved response-loss receipt=%+v error=%v, want lookup error without guessed acceptance", accepted, err)
	}
	if acceptedResult.Outcome != swarmionapp.OperationTransactionOutcomeExecuted || acceptedResult.Receipt == nil {
		t.Fatalf("injected accepted result=%+v, want executed with exact receipt", acceptedResult)
	}
	if calls := lookupCalls.Load(); calls != 1 {
		t.Fatalf("post-error receipt lookup calls=%d, want 1", calls)
	}
	if got := transactionTestUserCount(t, store, firstID); got != 1 {
		t.Fatalf("accepted row count after response and lookup failure=%d, want 1", got)
	}

	store.lookupPublishedWriteForTest = nil
	recovered, err := UpdateAndInsertWithOperationReceiptContext(
		context.Background(),
		store,
		operation,
		nil,
		[]InsertMapper{transactionTestUserInsert(secondID, "must-not-run-after-recovery")},
	)
	if err != nil || !recovered.HasExactEventIdentity() {
		t.Fatalf("recover stable operation receipt=%+v error=%v", recovered, err)
	}
	if recovered.EventID != acceptedResult.Receipt.EventID || recovered.PublishedRootHash != acceptedResult.Receipt.PublishedRootHash {
		t.Fatalf("recovered receipt=%+v, want original %+v", recovered, *acceptedResult.Receipt)
	}
	if got := transactionTestUserCount(t, store, firstID); got != 1 {
		t.Fatalf("original stable operation row count after recovery=%d, want 1", got)
	}
	if got := transactionTestUserCount(t, store, secondID); got != 0 {
		t.Fatalf("stable operation SQL body replayed after receipt recovery: count=%d", got)
	}
	if calls := runnerCalls.Load(); calls != 2 {
		t.Fatalf("operation runner calls=%d, want one accepted call and one recovery call", calls)
	}
	delta := transactionMetricsDelta(store.TransactionMetrics(), before)
	if delta.TransactionsStarted != 1 || delta.CommitsAttempted != 1 || delta.CommitsSucceeded != 1 ||
		delta.CommitsFailed != 0 || delta.RollbacksAttempted != 0 ||
		delta.OperationTransactionsAttempted != 2 || delta.OperationTransactionsAlreadyAccepted != 1 ||
		delta.OperationTransactionsFailed != 1 || delta.OperationTransactionsExecuted != 0 ||
		delta.OperationReceiptsFoundAfterCommitErr != 0 ||
		delta.OperationTransactionLifecycleOpaqueFailures != 1 {
		t.Fatalf("stable operation receipt recovery metrics=%+v", delta)
	}
}

func TestStableOperationNoChangeRecordsReceiptAndConsumesIdentity(t *testing.T) {
	store := openPeerTestDB(t)
	operationKey, err := NewPublishedWriteOperationKey()
	if err != nil {
		t.Fatal(err)
	}
	operation, err := NewPublishedWriteOperation(operationKey, "protos:test:operation-no-change:v1")
	if err != nil {
		t.Fatal(err)
	}
	missingID := MustNewUUIDv7()
	replayID := MustNewUUIDv7()
	var outcomes []swarmionapp.OperationTransactionOutcome
	store.runOperationTransactionForTest = func(
		ctx context.Context,
		sqldb *sql.DB,
		request swarmionapp.OperationTransactionRequest,
	) (swarmionapp.OperationTransactionResult, error) {
		if request.NoChangePolicy != swarmionapp.OperationNoChangePolicyRecordReceipt {
			t.Fatalf("operation no-change policy=%q, want record_receipt", request.NoChangePolicy)
		}
		result, err := swarmionapp.RunOperationTransaction(ctx, sqldb, request)
		if err == nil {
			outcomes = append(outcomes, result.Outcome)
		}
		return result, err
	}
	before := store.TransactionMetrics()

	receipt, err := UpdateAndInsertWithOperationReceiptContext(
		context.Background(),
		store,
		operation,
		[]UpdateMapper{transactionTestUserUpdate(missingID, "no-change")},
		nil,
	)
	if err != nil {
		t.Fatalf("record strict no-change operation: %v", err)
	}
	if !receipt.Committed || !receipt.HasExactEventIdentity() {
		t.Fatalf("strict no-change operation receipt is incomplete: %+v", receipt)
	}
	if got := transactionTestUserCount(t, store, missingID); got != 0 {
		t.Fatalf("strict no-change update created a row: count=%d", got)
	}
	resolved, err := directOperationReceiptLookup(store, context.Background(), operation)
	if err != nil || resolved.Resolution != swarmionapp.BranchOperationReceiptFound ||
		resolved.EventID != receipt.EventID || resolved.PublishedRootHash != receipt.PublishedRootHash {
		t.Fatalf("strict no-change operation receipt resolution=%+v error=%v, want %+v", resolved, err, receipt)
	}

	replayed, err := UpdateAndInsertWithOperationReceiptContext(
		context.Background(),
		store,
		operation,
		nil,
		[]InsertMapper{transactionTestUserInsert(replayID, "strict-no-change-body-must-not-replay")},
	)
	if err != nil {
		t.Fatalf("replay strict no-change operation: %v", err)
	}
	if replayed.EventID != receipt.EventID || replayed.PublishedRootHash != receipt.PublishedRootHash {
		t.Fatalf("strict no-change replay receipt=%+v, want original %+v", replayed, receipt)
	}
	if got := transactionTestUserCount(t, store, replayID); got != 0 {
		t.Fatalf("strict no-change operation replayed changed body: count=%d", got)
	}
	if len(outcomes) != 2 ||
		outcomes[0] != swarmionapp.OperationTransactionOutcomeNoChange ||
		outcomes[1] != swarmionapp.OperationTransactionOutcomeAlreadyAccepted {
		t.Fatalf("strict no-change operation outcomes=%v, want [no_change already_accepted]", outcomes)
	}

	delta := transactionMetricsDelta(store.TransactionMetrics(), before)
	if delta.TransactionsStarted != 2 || delta.CommitsAttempted != 2 ||
		delta.CommitsSucceeded != 2 || delta.CommitsFailed != 0 ||
		delta.NoopCommitOutcomes != 1 || delta.RollbacksAttempted != 0 ||
		delta.OperationTransactionsAttempted != 2 || delta.OperationTransactionsNoChange != 1 ||
		delta.OperationTransactionsExecuted != 0 || delta.OperationTransactionsAlreadyAccepted != 1 ||
		delta.OperationTransactionsFailed != 0 {
		t.Fatalf("strict no-change operation metrics=%+v", delta)
	}
}

func TestOperationWorkspaceDirtyIsCASDiscardedBeforeOperation(t *testing.T) {
	store := openPeerTestDB(t)
	ctx := context.Background()
	draftID := MustNewUUIDv7()
	bodyID := MustNewUUIDv7()
	if _, err := store.GetSqlDB().ExecContext(
		ctx,
		"INSERT INTO users (id, username, name, is_disabled) VALUES (?, ?, ?, ?)",
		MustUUIDBytes(draftID),
		"ordinary-draft",
		"ordinary-draft",
		false,
	); err != nil {
		t.Fatalf("prepare ordinary unpublished draft: %v", err)
	}

	operationKey, err := NewPublishedWriteOperationKey()
	if err != nil {
		t.Fatal(err)
	}
	operation, err := NewPublishedWriteOperation(operationKey, "protos:test:operation-dirty-workspace:v1")
	if err != nil {
		t.Fatal(err)
	}
	var runnerCalls atomic.Int32
	store.runOperationTransactionForTest = func(
		ctx context.Context,
		sqldb *sql.DB,
		request swarmionapp.OperationTransactionRequest,
	) (swarmionapp.OperationTransactionResult, error) {
		runnerCalls.Add(1)
		return swarmionapp.RunOperationTransaction(ctx, sqldb, request)
	}
	before := store.TransactionMetrics()

	receipt, err := UpdateAndInsertWithOperationReceiptContext(
		ctx,
		store,
		operation,
		nil,
		[]InsertMapper{transactionTestUserInsert(bodyID, "operation-body-must-not-run")},
	)
	if err != nil {
		t.Fatalf("CAS-clean dirty workspace operation receipt=%+v error=%v", receipt, err)
	}
	if !receipt.Committed || !receipt.HasExactEventIdentity() || runnerCalls.Load() != 1 {
		t.Fatalf("dirty-workspace operation receipt=%+v runner_calls=%d, want one accepted attempt after preflight cleanup", receipt, runnerCalls.Load())
	}
	if got := transactionTestUserCount(t, store, draftID); got != 0 {
		t.Fatalf("abandoned ordinary draft survived CAS discard: count=%d", got)
	}
	if got := transactionTestUserCount(t, store, bodyID); got != 1 {
		t.Fatalf("operation body did not execute after CAS cleanup: count=%d", got)
	}
	resolved, lookupErr := directOperationReceiptLookup(store, ctx, operation)
	if lookupErr != nil || resolved.Resolution != swarmionapp.BranchOperationReceiptFound ||
		!swarmionapp.SameBranchOperationReceiptIdentity(swarmionapp.BranchOperationReceipt{
			Resolution:        swarmionapp.BranchOperationReceiptFound,
			EventID:           receipt.EventID,
			PublishedRootHash: receipt.PublishedRootHash,
			EventDigest:       receipt.EventDigest,
			AuthorPeerID:      receipt.AuthorPeerID,
			AuthorSeq:         receipt.AuthorSeq,
			IntentDigest:      receipt.OperationIntentDigest,
		}, resolved) {
		t.Fatalf("accepted operation receipt resolution=%+v error=%v, want same found receipt", resolved, lookupErr)
	}

	delta := transactionMetricsDelta(store.TransactionMetrics(), before)
	if delta.TransactionsStarted != 1 || delta.CommitsAttempted != 1 || delta.CommitsSucceeded != 1 ||
		delta.RollbacksAttempted != 0 || delta.OperationTransactionsAttempted != 1 ||
		delta.OperationTransactionsFailed != 0 || delta.OperationWorkspaceDirtyOutcomes != 0 ||
		delta.OperationTransactionsExecuted != 1 || delta.OperationTransactionsAlreadyAccepted != 0 ||
		delta.OperationTransactionsNoChange != 0 {
		t.Fatalf("dirty-workspace operation metrics=%+v", delta)
	}
}

type safeNotAcceptedCommitTransaction struct {
	sqlWriteTransaction
}

func (tx *safeNotAcceptedCommitTransaction) Commit() error {
	if err := tx.Rollback(); err != nil {
		return errors.Join(
			&swarmionapp.CommitNotAcceptedError{Cause: errors.New("injected admission rejection")},
			err,
		)
	}
	return &swarmionapp.CommitNotAcceptedError{Cause: errors.New("injected admission rejection")}
}

func TestSafeNotAcceptedRetryBudgetExhaustionIsTerminal(t *testing.T) {
	store := openPeerTestDB(t)
	id := MustNewUUIDv7()
	var attempts []PublishedWriteOperation
	store.wrapWriteTransactionForTest = func(tx sqlWriteTransaction, operation PublishedWriteOperation, _ string) sqlWriteTransaction {
		attempts = append(attempts, operation)
		return &safeNotAcceptedCommitTransaction{sqlWriteTransaction: tx}
	}
	before := store.TransactionMetrics()

	receipt, err := InsertWithReceiptContext(
		context.Background(),
		store,
		transactionTestUserInsert(id, "safe-not-accepted-budget"),
	)
	if err == nil || IsRetryablePublishedWriteError(err) || !errors.Is(err, swarmionapp.ErrCommitNotAcceptedSafeToRetry) {
		t.Fatalf("retry exhaustion receipt=%+v error=%v, want terminal budget error retaining typed diagnostic", receipt, err)
	}
	if len(attempts) != ordinaryWriteSafeRetryMaxAttempts {
		t.Fatalf("safe-not-accepted attempts=%d, want %d", len(attempts), ordinaryWriteSafeRetryMaxAttempts)
	}
	for index := 1; index < len(attempts); index++ {
		if attempts[index] != attempts[0] {
			t.Fatalf("safe-not-accepted retry changed operation identity: %+v", attempts)
		}
	}
	if got := transactionTestUserCount(t, store, id); got != 0 {
		t.Fatalf("safe-not-accepted retry left row visible: count=%d", got)
	}
	delta := transactionMetricsDelta(store.TransactionMetrics(), before)
	if delta.TransactionsStarted != uint64(ordinaryWriteSafeRetryMaxAttempts) ||
		delta.CommitsAttempted != uint64(ordinaryWriteSafeRetryMaxAttempts) ||
		delta.CommitsSucceeded != 0 || delta.CommitsFailed != uint64(ordinaryWriteSafeRetryMaxAttempts) {
		t.Fatalf("safe-not-accepted retry metrics=%+v", delta)
	}
}

type staleWriteCommitOnceTransaction struct {
	sqlWriteTransaction
	attempts *atomic.Int32
}

func (tx *staleWriteCommitOnceTransaction) Commit() error {
	if tx.attempts.Add(1) != 1 {
		return tx.sqlWriteTransaction.Commit()
	}
	// The real Swarmion stale-context outcome restores the pre-write workspace
	// before returning. Mirror that cleanup in this injected driver boundary so
	// the regression measures only the backend's retry decision.
	if err := tx.Rollback(); err != nil {
		return errors.Join(swarmionprotocol.ErrStaleWriteContext, err)
	}
	return swarmionprotocol.ErrStaleWriteContext
}

type projectionTooWideCommitOnceTransaction struct {
	sqlWriteTransaction
	commitAttempts *atomic.Int32
	statementCalls *atomic.Int32
}

func (tx *projectionTooWideCommitOnceTransaction) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	tx.statementCalls.Add(1)
	return tx.sqlWriteTransaction.ExecContext(ctx, query, args...)
}

func (tx *projectionTooWideCommitOnceTransaction) Commit() error {
	if tx.commitAttempts.Add(1) != 1 {
		return tx.sqlWriteTransaction.Commit()
	}
	// ErrProjectionTooWide guarantees that no event was accepted and the SQL
	// workspace was restored. Mirror that driver contract in this injected
	// boundary before the backend selects its readiness barrier.
	if err := tx.Rollback(); err != nil {
		return errors.Join(swarmionprotocol.ErrProjectionTooWide, err)
	}
	return &swarmionapp.ProjectionTooWideError{
		HeadCount: swarmionprotocol.MaxParentRefs + 1,
		MaxHeads:  swarmionprotocol.MaxParentRefs,
	}
}

func TestProjectionTooWideWaitsForWriteReadinessAndRerunsWholeHelper(t *testing.T) {
	store := openPeerTestDB(t)
	firstID := MustNewUUIDv7()
	secondID := MustNewUUIDv7()
	var (
		commitAttempts atomic.Int32
		statementCalls atomic.Int32
		writeWaits     atomic.Int32
		viewWaits      atomic.Int32
		operations     []PublishedWriteOperation
	)
	store.wrapWriteTransactionForTest = func(tx sqlWriteTransaction, operation PublishedWriteOperation, _ string) sqlWriteTransaction {
		operations = append(operations, operation)
		return &projectionTooWideCommitOnceTransaction{
			sqlWriteTransaction: tx,
			commitAttempts:      &commitAttempts,
			statementCalls:      &statementCalls,
		}
	}
	store.waitSQLWriteReadyForTest = func(context.Context) error {
		writeWaits.Add(1)
		if got := statementCalls.Load(); got != 2 {
			t.Fatalf("write-readiness barrier observed statement calls=%d, want one complete two-statement attempt", got)
		}
		return nil
	}
	store.waitSQLViewReadyForTest = func(context.Context) error {
		viewWaits.Add(1)
		return errors.New("read-visibility barrier must not handle projection width")
	}
	before := store.TransactionMetrics()

	receipt, err := InsertWithReceiptContext(
		context.Background(),
		store,
		transactionTestUserInsert(firstID, "projection-write-ready-first"),
		transactionTestUserInsert(secondID, "projection-write-ready-second"),
	)
	if err != nil || !receipt.HasExactEventIdentity() {
		t.Fatalf("projection-width whole-helper retry receipt=%+v error=%v", receipt, err)
	}
	if len(operations) != 2 || operations[0] != operations[1] {
		t.Fatalf("projection-width retry changed stable operation identity: %+v", operations)
	}
	if got := statementCalls.Load(); got != 4 {
		t.Fatalf("statement calls=%d, want both statements rerun in a fresh transaction", got)
	}
	if writeWaits.Load() != 1 || viewWaits.Load() != 0 {
		t.Fatalf("readiness waits write=%d view=%d, want write=1 view=0", writeWaits.Load(), viewWaits.Load())
	}
	if got := transactionTestUserCount(t, store, firstID); got != 1 {
		t.Fatalf("first row count=%d, want 1 after first attempt rollback and whole-helper retry", got)
	}
	if got := transactionTestUserCount(t, store, secondID); got != 1 {
		t.Fatalf("second row count=%d, want 1 after first attempt rollback and whole-helper retry", got)
	}

	delta := transactionMetricsDelta(store.TransactionMetrics(), before)
	if delta.TransactionsStarted != 2 || delta.CommitsAttempted != 2 ||
		delta.CommitsSucceeded != 1 || delta.CommitsFailed != 1 ||
		delta.StaleWriteContextOutcomes != 1 || delta.ProjectionTooWideOutcomes != 1 ||
		delta.RollbacksAttempted != 0 {
		t.Fatalf("projection-width retry metrics=%+v", delta)
	}
}

func TestOrdinaryStaleWriteContextRerunsWholeOperationWithSameIdentity(t *testing.T) {
	store := openPeerTestDB(t)
	id := MustNewUUIDv7()
	var commits atomic.Int32
	var operations []PublishedWriteOperation
	store.wrapWriteTransactionForTest = func(tx sqlWriteTransaction, operation PublishedWriteOperation, _ string) sqlWriteTransaction {
		operations = append(operations, operation)
		return &staleWriteCommitOnceTransaction{sqlWriteTransaction: tx, attempts: &commits}
	}
	before := store.TransactionMetrics()

	receipt, err := InsertWithReceiptContext(
		context.Background(),
		store,
		transactionTestUserInsert(id, "stale-context-full-operation-retry"),
	)
	if err != nil || !receipt.HasExactEventIdentity() {
		t.Fatalf("stale-context retry receipt=%+v error=%v", receipt, err)
	}
	if len(operations) != 2 || operations[0] != operations[1] {
		t.Fatalf("stale-context attempts changed stable operation identity: %+v", operations)
	}
	if got := transactionTestUserCount(t, store, id); got != 1 {
		t.Fatalf("whole-operation retry row count=%d, want 1", got)
	}

	delta := transactionMetricsDelta(store.TransactionMetrics(), before)
	if delta.TransactionsStarted != 2 || delta.CommitsAttempted != 2 ||
		delta.CommitsSucceeded != 1 || delta.CommitsFailed != 1 ||
		delta.StaleWriteContextOutcomes != 1 || delta.ProjectionTooWideOutcomes != 0 ||
		delta.RollbacksAttempted != 0 {
		t.Fatalf("stale-context retry metrics=%+v", delta)
	}
}

func TestStableOperationStaleWriteContextAuthorizesOnlyWholeHelperReplay(t *testing.T) {
	store := openPeerTestDB(t)
	operationKey, err := NewPublishedWriteOperationKey()
	if err != nil {
		t.Fatal(err)
	}
	operation, err := NewPublishedWriteOperation(operationKey, "protos:test:stale-operation-helper:v1")
	if err != nil {
		t.Fatal(err)
	}
	id := MustNewUUIDv7()
	var runnerCalls atomic.Int32
	store.runOperationTransactionForTest = func(
		ctx context.Context,
		sqldb *sql.DB,
		request swarmionapp.OperationTransactionRequest,
	) (swarmionapp.OperationTransactionResult, error) {
		if request.Operation.Key != operation.Key || request.Operation.IntentDigest != operation.IntentDigest {
			t.Fatalf("operation helper replay changed identity: %+v", request.Operation)
		}
		if runnerCalls.Add(1) == 1 {
			return swarmionapp.OperationTransactionResult{}, &swarmionapp.OperationTransactionError{
				Phase:          swarmionapp.OperationTransactionPhaseCommit,
				StatementIndex: -1,
				CommitStatus:   swarmionapp.OperationTransactionCommitReturnedError,
				RollbackStatus: swarmionapp.OperationTransactionRollbackNotAttempted,
				Cause:          swarmionapp.ErrStaleWriteContext,
			}
		}
		return swarmionapp.RunOperationTransaction(ctx, sqldb, request)
	}
	before := store.TransactionMetrics()

	first, firstErr := UpdateAndInsertWithOperationReceiptContext(
		context.Background(),
		store,
		operation,
		nil,
		[]InsertMapper{transactionTestUserInsert(id, "stale-operation-helper")},
	)
	if firstErr == nil || first.HasExactEventIdentity() ||
		!errors.Is(firstErr, swarmionapp.ErrStaleWriteContext) ||
		!IsRetryablePublishedWriteError(firstErr) {
		t.Fatalf("first stale helper result receipt=%+v retryable=%t error=%v", first, IsRetryablePublishedWriteError(firstErr), firstErr)
	}
	if got := transactionTestUserCount(t, store, id); got != 0 {
		t.Fatalf("stale helper first attempt changed SQL: count=%d", got)
	}

	replayed, replayErr := UpdateAndInsertWithOperationReceiptContext(
		context.Background(),
		store,
		operation,
		nil,
		[]InsertMapper{transactionTestUserInsert(id, "stale-operation-helper")},
	)
	if replayErr != nil || !replayed.HasExactEventIdentity() || runnerCalls.Load() != 2 {
		t.Fatalf("whole-helper replay receipt=%+v calls=%d error=%v", replayed, runnerCalls.Load(), replayErr)
	}
	if got := transactionTestUserCount(t, store, id); got != 1 {
		t.Fatalf("whole-helper replay row count=%d, want 1", got)
	}

	delta := transactionMetricsDelta(store.TransactionMetrics(), before)
	if delta.OperationTransactionsAttempted != 2 || delta.OperationTransactionsFailed != 1 ||
		delta.OperationTransactionsExecuted != 1 || delta.StaleWriteContextOutcomes != 1 ||
		delta.CommitsAttempted != 2 || delta.CommitsFailed != 1 || delta.CommitsSucceeded != 1 ||
		delta.RollbacksAttempted != 0 {
		t.Fatalf("stable stale-operation metrics=%+v", delta)
	}
}

func TestProjectionTooWideIsTypedWholeOperationRetryOnly(t *testing.T) {
	err := &swarmionapp.ProjectionTooWideError{
		HeadCount: swarmionprotocol.MaxParentRefs + 1,
		MaxHeads:  swarmionprotocol.MaxParentRefs,
	}
	if !errors.Is(err, swarmionapp.ErrProjectionTooWide) ||
		!errors.Is(err, swarmionapp.ErrStaleWriteContext) ||
		!IsRetryablePublishedWriteError(err) {
		t.Fatalf("projection-too-wide classification retryable=%t error=%v", IsRetryablePublishedWriteError(err), err)
	}
	terminal := fmt.Errorf("%w: %w", errPublishedWriteRetryExhausted, err)
	if IsRetryablePublishedWriteError(terminal) {
		t.Fatalf("exhausted projection retry remained replayable: %v", terminal)
	}
}

func TestAcceptedOrdinaryCommitErrorTextCannotMasqueradeAsNoop(t *testing.T) {
	store := openPeerTestDB(t)
	id := MustNewUUIDv7()
	if _, err := InsertWithReceiptContext(
		context.Background(),
		store,
		transactionTestUserInsert(id, "accepted-before-diagnostic"),
	); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	var wrapped atomic.Bool
	store.wrapWriteTransactionForTest = func(tx sqlWriteTransaction, _ PublishedWriteOperation, name string) sqlWriteTransaction {
		if name == "update" && wrapped.CompareAndSwap(false, true) {
			return &acceptedCommitResponseErrorTransaction{
				sqlWriteTransaction: tx,
				err:                 errors.New("response lost after acceptance: nothing to commit is unrelated diagnostic text"),
			}
		}
		return tx
	}
	before := store.TransactionMetrics()

	receipt, err := UpdateWithReceiptContext(
		context.Background(),
		store,
		transactionTestUserUpdate(id, "accepted-after-diagnostic"),
	)
	if err != nil {
		t.Fatalf("resolve accepted ordinary update: %v", err)
	}
	if !receipt.HasExactEventIdentity() || transactionTestUserName(t, store, id) != "accepted-after-diagnostic" {
		t.Fatalf("accepted ordinary update was mislabeled no-op receipt=%+v name=%q", receipt, transactionTestUserName(t, store, id))
	}

	delta := transactionMetricsDelta(store.TransactionMetrics(), before)
	if delta.TransactionsStarted != 1 || delta.CommitsAttempted != 1 ||
		delta.CommitsFailed != 1 || delta.NoopCommitOutcomes != 0 ||
		delta.OperationReceiptsFoundAfterCommitErr != 1 || delta.RollbacksAttempted != 0 {
		t.Fatalf("accepted diagnostic-text transaction metrics=%+v", delta)
	}
}

func transactionHeadMovementIterations(t *testing.T) int {
	t.Helper()
	raw := os.Getenv("PROTOS_TX_HEAD_MOVE_ITERATIONS")
	if raw == "" {
		return 20
	}
	iterations, err := strconv.Atoi(raw)
	if err != nil || iterations < 1 {
		t.Fatalf("invalid PROTOS_TX_HEAD_MOVE_ITERATIONS=%q", raw)
	}
	return iterations
}

func TestOrdinaryTransactionPreservesPriorWriteAcrossProtocolHeadMovement(t *testing.T) {
	// Force a short, non-eager deadline path so A is locally published before B
	// starts, then materialize A while B's transaction workspace is live.
	t.Setenv("SWARMION_CHECKPOINT_SCHEDULER_PROFILE", "")
	t.Setenv("SWARMION_CONTINUOUS_EVENT_DEADLINE", "25ms")
	t.Setenv("SWARMION_CONTINUOUS_DEADLINE_PERMIT_GAP", "2ms")
	t.Setenv("SWARMION_CONTINUOUS_DEADLINE_STAGGER_GAP", "2ms")
	t.Setenv("SWARMION_CONTINUOUS_DEADLINE_STAGGER_SLOTS", "1")
	t.Setenv("SWARMION_STABLE_PREFIX_ADAPTIVE_TARGETS", "false")
	t.Setenv("SWARMION_STABLE_PREFIX_EVIDENCE_TARGET", "100")
	t.Setenv("SWARMION_STABLE_PREFIX_PLAN_MATCH_TARGET", "100")

	store := openPeerTestDB(t)
	iterations := transactionHeadMovementIterations(t)
	beforeAll := store.TransactionMetrics()
	var committed int

	for iteration := 0; iteration < iterations; iteration++ {
		aID := MustNewUUIDv7()
		aName := fmt.Sprintf("head-move-a-%d", iteration)
		bName := fmt.Sprintf("head-move-b-%d", iteration)
		aReceipt, err := InsertWithReceiptContext(
			context.Background(),
			store,
			transactionTestUserInsert(aID, aName),
		)
		if err != nil {
			t.Fatalf("iteration %d publish A: %v", iteration, err)
		}
		statusBeforeB, ok := store.SwarmionStatus()
		if !ok {
			t.Fatalf("iteration %d read status before B", iteration)
		}
		checkpointBeforeB := statusBeforeB.ProtocolCheckpointCommitID.String()
		hookCalls := 0
		var checkpointDuringB string
		store.beforeWriteTransactionCommitForTest = func(_ context.Context, _ PublishedWriteOperation, name string) error {
			if name != "update" {
				return nil
			}
			hookCalls++
			waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			observation, waitErr := store.WaitForPublishedWriteApplied(waitCtx, aReceipt, "transaction head movement regression")
			if waitErr != nil {
				return waitErr
			}
			checkpointDuringB = observation.Status.DurableCheckpointCommitID
			return nil
		}

		beforeB := store.TransactionMetrics()
		bReceipt, bErr := UpdateWithReceiptContext(
			context.Background(),
			store,
			transactionTestUserUpdate(aID, bName),
		)
		store.beforeWriteTransactionCommitForTest = nil
		if hookCalls != 1 {
			t.Fatalf("iteration %d before-commit hook calls=%d, want 1", iteration, hookCalls)
		}
		if checkpointDuringB == "" || checkpointDuringB == checkpointBeforeB {
			t.Fatalf("iteration %d protocol checkpoint did not move while B was live: before=%s during=%s", iteration, checkpointBeforeB, checkpointDuringB)
		}
		if bErr != nil {
			t.Fatalf("iteration %d B commit across same-author head movement: %v", iteration, bErr)
		}
		if got := transactionTestUserCount(t, store, aID); got != 1 {
			t.Fatalf("iteration %d successful A disappeared after B: count=%d", iteration, got)
		}
		bDelta := transactionMetricsDelta(store.TransactionMetrics(), beforeB)
		if bDelta.TransactionsStarted != 1 || bDelta.CommitsAttempted != 1 || bDelta.CommitsSucceeded != 1 ||
			bDelta.CommitsFailed != 0 || bDelta.TypedConflicts != 0 || bDelta.RollbacksAttempted != 0 {
			t.Fatalf("iteration %d B transaction metrics=%+v", iteration, bDelta)
		}
		committed++
		if !bReceipt.HasExactEventIdentity() || transactionTestUserName(t, store, aID) != bName {
			t.Fatalf("iteration %d committed B receipt=%+v name=%q", iteration, bReceipt, transactionTestUserName(t, store, aID))
		}
	}

	delta := transactionMetricsDelta(store.TransactionMetrics(), beforeAll)
	if delta.RollbacksAttempted != 0 {
		t.Fatalf("nominal head-movement rollback frequency=%d/%d metrics=%+v", delta.RollbacksAttempted, delta.TransactionsStarted, delta)
	}
	t.Logf(
		"nominal head-movement iterations=%d committed=%d typed_conflicts=0 rollbacks=%d/%d metrics=%+v",
		iterations,
		committed,
		delta.RollbacksAttempted,
		delta.TransactionsStarted,
		delta,
	)
}

func TestOrdinaryTransactionPreservesPendingWriteChainAcrossProtocolHeadMovement(t *testing.T) {
	// Build two causally ordered local writes before the first one checkpoints,
	// then keep the third transaction live while the protocol advances only part
	// way through that local chain. This is the shape exercised by rapid task
	// lifecycle transitions (queued -> running -> pending/succeeded).
	t.Setenv("SWARMION_CHECKPOINT_SCHEDULER_PROFILE", "")
	t.Setenv("SWARMION_CONTINUOUS_EVENT_DEADLINE", "300ms")
	t.Setenv("SWARMION_CONTINUOUS_DEADLINE_PERMIT_GAP", "500ms")
	t.Setenv("SWARMION_CONTINUOUS_DEADLINE_STAGGER_GAP", "2ms")
	t.Setenv("SWARMION_CONTINUOUS_DEADLINE_STAGGER_SLOTS", "1")
	t.Setenv("SWARMION_STABLE_PREFIX_ADAPTIVE_TARGETS", "false")
	t.Setenv("SWARMION_STABLE_PREFIX_EVIDENCE_TARGET", "100")
	t.Setenv("SWARMION_STABLE_PREFIX_PLAN_MATCH_TARGET", "100")

	store := openPeerTestDB(t)
	iterations := transactionHeadMovementIterations(t)
	beforeAll := store.TransactionMetrics()
	var committed, pendingB int
	var cTotalElapsed, cMaxElapsed time.Duration

	for iteration := 0; iteration < iterations; iteration++ {
		id := MustNewUUIDv7()
		bSideID := MustNewUUIDv7()
		cSideID := MustNewUUIDv7()
		aName := fmt.Sprintf("pending-chain-a-%d", iteration)
		bName := fmt.Sprintf("pending-chain-b-%d", iteration)
		cName := fmt.Sprintf("pending-chain-c-%d", iteration)
		aReceipt, err := InsertWithReceiptContext(
			context.Background(),
			store,
			transactionTestUserInsert(id, aName),
		)
		if err != nil {
			t.Fatalf("iteration %d publish A: %v", iteration, err)
		}
		bReceipt, err := UpdateAndInsertWithReceiptContext(
			context.Background(),
			store,
			[]UpdateMapper{transactionTestUserUpdate(id, bName)},
			[]InsertMapper{transactionTestUserInsert(bSideID, fmt.Sprintf("pending-chain-b-side-%d", iteration))},
		)
		if err != nil {
			t.Fatalf("iteration %d publish B: %v", iteration, err)
		}
		if transactionTestUserName(t, store, id) != bName || transactionTestUserCount(t, store, bSideID) != 1 {
			t.Fatalf("iteration %d B was not fully visible before C", iteration)
		}
		aBeforeAdvance, err := store.ObservePublishedWriteReceipt(context.Background(), aReceipt)
		if err != nil {
			t.Fatalf("iteration %d observe A before checkpoint advance: %v", iteration, err)
		}
		if aBeforeAdvance.State != EventReceiptStatePending {
			t.Fatalf("iteration %d A state after B acceptance=%s, want pending", iteration, aBeforeAdvance.State)
		}
		windowCtx, cancelWindow := context.WithTimeout(context.Background(), 5*time.Second)
		for {
			if err := windowCtx.Err(); err != nil {
				cancelWindow()
				t.Fatalf("iteration %d wait for A-applied/B-pending window: %v", iteration, err)
			}
			aObservation, observeErr := store.ObservePublishedWriteReceipt(windowCtx, aReceipt)
			if observeErr != nil {
				cancelWindow()
				t.Fatalf("iteration %d observe A while waiting for chain window: %v", iteration, observeErr)
			}
			bObservation, observeErr := store.ObservePublishedWriteReceipt(windowCtx, bReceipt)
			if observeErr != nil {
				cancelWindow()
				t.Fatalf("iteration %d observe B while waiting for chain window: %v", iteration, observeErr)
			}
			if bObservation.State != EventReceiptStatePending {
				cancelWindow()
				t.Fatalf("iteration %d B became %s before observing A-only durable checkpoint", iteration, bObservation.State)
			}
			if aObservation.State == EventReceiptStateAppliedDurably &&
				aObservation.Status.DurableCheckpointRootHash == aReceipt.PublishedRootHash {
				break
			}
			timer := time.NewTimer(5 * time.Millisecond)
			select {
			case <-timer.C:
			case <-windowCtx.Done():
				timer.Stop()
			}
		}
		cancelWindow()

		var bStateDuringC EventReceiptState
		var cReadinessDiagnostics []string
		var cStatementCalls atomic.Int32
		store.wrapWriteTransactionForTest = func(tx sqlWriteTransaction, _ PublishedWriteOperation, name string) sqlWriteTransaction {
			if name != "update and insert" {
				return tx
			}
			return &observingWriteTransaction{
				sqlWriteTransaction: tx,
				calls:               &cStatementCalls,
				onExecError: func(query string, execErr error) {
					var notReady *swarmionapp.SQLViewNotReadyError
					if !errors.As(execErr, &notReady) || notReady == nil {
						return
					}
					status, _ := store.SwarmionStatus()
					bObservation, bObserveErr := store.ObservePublishedWriteReceipt(context.Background(), bReceipt)
					cReadinessDiagnostics = append(cReadinessDiagnostics, fmt.Sprintf(
						"query=%q a_root=%s b_root=%s b_state=%s b_observe_error=%v target=%s visible=%s error_checkpoint=%s status_checkpoint=%s status_tentative=%s reason=%q",
						query,
						aReceipt.PublishedRootHash,
						bReceipt.PublishedRootHash,
						bObservation.State,
						bObserveErr,
						notReady.TargetRootHash,
						notReady.VisibleRootHash,
						notReady.CheckpointRootHash,
						status.CheckpointRootHash,
						status.TentativeRootHash,
						notReady.Reason,
					))
				},
			}
		}
		store.beforeWriteTransactionCommitForTest = func(_ context.Context, _ PublishedWriteOperation, name string) error {
			if name != "update and insert" {
				return nil
			}
			observation, observeErr := store.ObservePublishedWriteReceipt(context.Background(), bReceipt)
			if observeErr != nil {
				return observeErr
			}
			bStateDuringC = observation.State
			return nil
		}

		beforeC := store.TransactionMetrics()
		cStarted := time.Now()
		cReceipt, cErr := UpdateAndInsertWithReceiptContext(
			context.Background(),
			store,
			[]UpdateMapper{transactionTestUserUpdate(id, cName)},
			[]InsertMapper{transactionTestUserInsert(cSideID, fmt.Sprintf("pending-chain-c-side-%d", iteration))},
		)
		cElapsed := time.Since(cStarted)
		cTotalElapsed += cElapsed
		if cElapsed > cMaxElapsed {
			cMaxElapsed = cElapsed
		}
		store.beforeWriteTransactionCommitForTest = nil
		store.wrapWriteTransactionForTest = nil
		if cErr != nil {
			t.Fatalf("iteration %d C commit across pending same-author chain: %v readiness=%v", iteration, cErr, cReadinessDiagnostics)
		}
		cDelta := transactionMetricsDelta(store.TransactionMetrics(), beforeC)
		if cDelta.TransactionsStarted != 1 || cDelta.CommitsAttempted != 1 || cDelta.CommitsSucceeded != 1 ||
			cDelta.CommitsFailed != 0 || cDelta.TypedConflicts != 0 || cDelta.RollbacksAttempted != 0 ||
			cDelta.SQLViewNotReadyOutcomes != 0 || cStatementCalls.Load() != 2 || len(cReadinessDiagnostics) != 0 {
			t.Fatalf("iteration %d C transaction metrics=%+v readiness=%v", iteration, cDelta, cReadinessDiagnostics)
		}

		committed++
		if !cReceipt.HasExactEventIdentity() || transactionTestUserName(t, store, id) != cName ||
			transactionTestUserCount(t, store, bSideID) != 1 || transactionTestUserCount(t, store, cSideID) != 1 {
			t.Fatalf(
				"iteration %d committed C did not preserve the causal chain receipt=%+v primary=%q b_side=%d c_side=%d",
				iteration,
				cReceipt,
				transactionTestUserName(t, store, id),
				transactionTestUserCount(t, store, bSideID),
				transactionTestUserCount(t, store, cSideID),
			)
		}
		switch bStateDuringC {
		case EventReceiptStatePending:
			pendingB++
		default:
			t.Fatalf(
				"iteration %d B state during C=%s, want pending metrics=%+v readiness=%v",
				iteration,
				bStateDuringC,
				cDelta,
				cReadinessDiagnostics,
			)
		}

		waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cObservation, waitErr := store.WaitForPublishedWriteApplied(waitCtx, cReceipt, "settle pending transaction chain regression")
		cancel()
		if waitErr != nil {
			t.Fatalf("iteration %d settle terminal receipt: %v", iteration, waitErr)
		}
		if transactionTestUserName(t, store, id) != cName ||
			transactionTestUserCount(t, store, bSideID) != 1 || transactionTestUserCount(t, store, cSideID) != 1 {
			t.Fatalf("iteration %d settled C lost same-author causal content", iteration)
		}
		var durablePrimaryName string
		if err := store.ReadRowsAsOf(
			context.Background(),
			cObservation.Status.DurableCheckpointCommitID,
			"SELECT name FROM users AS OF ? WHERE id = ?",
			[]any{MustUUIDBytes(id)},
			func(rows *sql.Rows) error {
				if !rows.Next() {
					return sql.ErrNoRows
				}
				return rows.Scan(&durablePrimaryName)
			},
		); err != nil {
			t.Fatalf("iteration %d read primary at C durable checkpoint: %v", iteration, err)
		}
		var durableSideCount int
		if err := store.ReadRowsAsOf(
			context.Background(),
			cObservation.Status.DurableCheckpointCommitID,
			"SELECT COUNT(*) FROM users AS OF ? WHERE id IN (?, ?)",
			[]any{MustUUIDBytes(bSideID), MustUUIDBytes(cSideID)},
			func(rows *sql.Rows) error {
				if !rows.Next() {
					return sql.ErrNoRows
				}
				return rows.Scan(&durableSideCount)
			},
		); err != nil {
			t.Fatalf("iteration %d read side rows at C durable checkpoint: %v", iteration, err)
		}
		if durablePrimaryName != cName || durableSideCount != 2 {
			t.Fatalf(
				"iteration %d C durable checkpoint lost causal content primary=%q side_count=%d",
				iteration,
				durablePrimaryName,
				durableSideCount,
			)
		}
	}

	delta := transactionMetricsDelta(store.TransactionMetrics(), beforeAll)
	if pendingB == 0 {
		t.Fatalf("pending-chain regression never observed B pending during C across %d iterations", iterations)
	}
	if delta.CommitsFailed != 0 || delta.TypedConflicts != 0 || delta.RollbacksAttempted != 0 ||
		delta.SQLViewNotReadyOutcomes != 0 {
		t.Fatalf("pending-chain rollback frequency=%d/%d metrics=%+v", delta.RollbacksAttempted, delta.TransactionsStarted, delta)
	}
	t.Logf(
		"pending-chain iterations=%d committed=%d B_pending=%d typed_conflicts=0 rollbacks=%d/%d sql_view_not_ready=0 C_mean=%s C_max=%s metrics=%+v",
		iterations,
		committed,
		pendingB,
		delta.RollbacksAttempted,
		delta.TransactionsStarted,
		cTotalElapsed/time.Duration(iterations),
		cMaxElapsed,
		delta,
	)
}
