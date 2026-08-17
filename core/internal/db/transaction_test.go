package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/bokwoon95/sq"
	swarmion "github.com/nustiueudinastea/swarmion/runtime"
)

func transactionMetricsDelta(after, before TransactionMetricsSnapshot) TransactionMetricsSnapshot {
	return TransactionMetricsSnapshot{
		TransactionsStarted:                         after.TransactionsStarted - before.TransactionsStarted,
		CommitsAttempted:                            after.CommitsAttempted - before.CommitsAttempted,
		CommitsSucceeded:                            after.CommitsSucceeded - before.CommitsSucceeded,
		NoopCommitOutcomes:                          after.NoopCommitOutcomes - before.NoopCommitOutcomes,
		TypedConflicts:                              after.TypedConflicts - before.TypedConflicts,
		OperationReceiptsFoundAfterCommitErr:        after.OperationReceiptsFoundAfterCommitErr - before.OperationReceiptsFoundAfterCommitErr,
		UncertainEventReceiptsAfterCommitErr:        after.UncertainEventReceiptsAfterCommitErr - before.UncertainEventReceiptsAfterCommitErr,
		StaleWriteContextOutcomes:                   after.StaleWriteContextOutcomes - before.StaleWriteContextOutcomes,
		ProjectionTooWideOutcomes:                   after.ProjectionTooWideOutcomes - before.ProjectionTooWideOutcomes,
		OperationTransactionsAttempted:              after.OperationTransactionsAttempted - before.OperationTransactionsAttempted,
		OperationTransactionsExecuted:               after.OperationTransactionsExecuted - before.OperationTransactionsExecuted,
		OperationTransactionsAlreadyAccepted:        after.OperationTransactionsAlreadyAccepted - before.OperationTransactionsAlreadyAccepted,
		OperationTransactionsNoChange:               after.OperationTransactionsNoChange - before.OperationTransactionsNoChange,
		OperationTransactionsFailed:                 after.OperationTransactionsFailed - before.OperationTransactionsFailed,
		OperationWorkspaceDirtyOutcomes:             after.OperationWorkspaceDirtyOutcomes - before.OperationWorkspaceDirtyOutcomes,
		OperationRecoveryPersistenceFailures:        after.OperationRecoveryPersistenceFailures - before.OperationRecoveryPersistenceFailures,
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
	if err := store.ReadRows(context.Background(), "SELECT COUNT(*) FROM users WHERE id = ?", []any{MustUUIDBytes(id)}, func(rows *sql.Rows) error {
		if !rows.Next() {
			return sql.ErrNoRows
		}
		return rows.Scan(&count)
	}); err != nil {
		t.Fatalf("count transaction test user %s: %v", id, err)
	}
	return count
}

func transactionTestUserName(t *testing.T, store *DB, id string) string {
	t.Helper()
	var name string
	if err := store.ReadRows(context.Background(), "SELECT name FROM users WHERE id = ?", []any{MustUUIDBytes(id)}, func(rows *sql.Rows) error {
		if !rows.Next() {
			return sql.ErrNoRows
		}
		return rows.Scan(&name)
	}); err != nil {
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

func transactionTestOperation(t *testing.T, store *DB, intent string) PublishedWriteOperation {
	t.Helper()
	operation, err := store.NewPublishedWriteOperation("io.protos.tests.transaction/v1", []byte(intent))
	if err != nil {
		t.Fatalf("create operation: %v", err)
	}
	return operation
}

func transactionTestRuntime(t *testing.T, store *DB) *swarmion.DatabaseRuntime {
	t.Helper()
	store.mu.Lock()
	runtime := store.runtime
	store.mu.Unlock()
	if runtime == nil {
		t.Fatal("Swarmion database runtime is unavailable")
	}
	return runtime
}

func requireAcceptedOperationResult(t *testing.T, store *DB, operation PublishedWriteOperation, receipt PublishedWriteReceipt) swarmion.OperationResult {
	t.Helper()
	result, err := store.LookupPublishedWriteOperation(context.Background(), operation)
	if err != nil {
		t.Fatalf("resolve published operation: %v", err)
	}
	if result.Disposition() != swarmion.OperationAccepted {
		t.Fatalf("operation disposition=%s diagnostic=%v, want accepted", result.Disposition(), result.Diagnostic())
	}
	resolved, err := PublishedWriteReceiptFromResult(operation, result)
	if err != nil {
		t.Fatalf("decode operation receipt: %v", err)
	}
	if resolved.EventID != receipt.EventID || resolved.PublishedRootHash != receipt.PublishedRootHash {
		t.Fatalf("operation receipt=%+v, want %s/%s", resolved, receipt.EventID, receipt.PublishedRootHash)
	}
	return result
}

func TestOperationIdentityUsesSchemaAndBinaryPartFraming(t *testing.T) {
	key, err := swarmion.NewOperationKey()
	if err != nil {
		t.Fatal(err)
	}
	first, err := swarmion.NewOperationIdentity(key, "io.protos.tests.identity/v1", []byte("a"), []byte("bc"))
	if err != nil {
		t.Fatal(err)
	}
	rebuilt, err := swarmion.NewOperationIdentity(key, "io.protos.tests.identity/v1", []byte("a"), []byte("bc"))
	if err != nil {
		t.Fatal(err)
	}
	changed, err := swarmion.NewOperationIdentity(key, "io.protos.tests.identity/v1", []byte("ab"), []byte("c"))
	if err != nil {
		t.Fatal(err)
	}
	if first != rebuilt || first == changed {
		t.Fatalf("binary framing identity first=%s rebuilt=%s changed=%s", first, rebuilt, changed)
	}
}

func TestStableOperationExecutedThenAlreadyAcceptedSkipsChangedBody(t *testing.T) {
	store := openPeerTestDB(t)
	operation := transactionTestOperation(t, store, "stable-operation")
	firstID, secondID := MustNewUUIDv7(), MustNewUUIDv7()
	first, err := UpdateAndInsertWithOperationReceiptContext(context.Background(), store, operation, nil, []InsertMapper{transactionTestUserInsert(firstID, "operation-executed")})
	if err != nil || !first.Committed || !first.HasExactEventIdentity() {
		t.Fatalf("execute stable operation receipt=%+v error=%v", first, err)
	}
	replayed, err := UpdateAndInsertWithOperationReceiptContext(context.Background(), store, operation, nil, []InsertMapper{transactionTestUserInsert(secondID, "must-not-run")})
	if err != nil || replayed.EventID != first.EventID || replayed.PublishedRootHash != first.PublishedRootHash {
		t.Fatalf("replay receipt=%+v error=%v, want %+v", replayed, err, first)
	}
	if got := transactionTestUserCount(t, store, secondID); got != 0 {
		t.Fatalf("already-accepted body executed: count=%d", got)
	}
	requireAcceptedOperationResult(t, store, operation, first)
}

func TestStableOperationStatementFailureIsAtomicAndResolvesAbsent(t *testing.T) {
	store := openPeerTestDB(t)
	operation := transactionTestOperation(t, store, "operation-atomicity")
	id := MustNewUUIDv7()
	receipt, err := UpdateAndInsertWithOperationReceiptContext(context.Background(), store, operation, nil, []InsertMapper{
		transactionTestUserInsert(id, "atomic-first"), transactionTestUserInsert(id, "atomic-duplicate"),
	})
	if err == nil || receipt.HasExactEventIdentity() || transactionTestUserCount(t, store, id) != 0 {
		t.Fatalf("failed operation receipt=%+v count=%d error=%v", receipt, transactionTestUserCount(t, store, id), err)
	}
	result, err := store.LookupPublishedWriteOperation(context.Background(), operation)
	if err != nil {
		t.Fatal(err)
	}
	if result.Disposition() != swarmion.OperationRetryPermitted {
		t.Fatalf("failed operation disposition=%s diagnostic=%v, want exact absence", result.Disposition(), result.Diagnostic())
	}
}

func TestStableNoChangeConsumesIdentityAndReturnsExactReceipt(t *testing.T) {
	store := openPeerTestDB(t)
	operation := transactionTestOperation(t, store, "no-change-operation")
	missingID, replayID := MustNewUUIDv7(), MustNewUUIDv7()
	first, err := UpdateAndInsertWithOperationReceiptContext(context.Background(), store, operation, []UpdateMapper{transactionTestUserUpdate(missingID, "no-change")}, nil)
	if err != nil || !first.Committed || !first.HasExactEventIdentity() {
		t.Fatalf("stable no-change receipt=%+v error=%v", first, err)
	}
	replayed, err := UpdateAndInsertWithOperationReceiptContext(context.Background(), store, operation, nil, []InsertMapper{transactionTestUserInsert(replayID, "must-not-replay")})
	if err != nil || replayed.EventID != first.EventID || transactionTestUserCount(t, store, replayID) != 0 {
		t.Fatalf("replayed no-change receipt=%+v error=%v", replayed, err)
	}
	requireAcceptedOperationResult(t, store, operation, first)
}

func TestOrdinaryMultiStatementApplyFailureRollsBackAllChanges(t *testing.T) {
	store := openPeerTestDB(t)
	id := MustNewUUIDv7()
	receipt, err := InsertWithReceiptContext(context.Background(), store,
		transactionTestUserInsert(id, "transaction-atomic-first"),
		transactionTestUserInsert(id, "transaction-atomic-duplicate"),
	)
	if err == nil || receipt.HasExactEventIdentity() || transactionTestUserCount(t, store, id) != 0 {
		t.Fatalf("atomic failure receipt=%+v count=%d error=%v", receipt, transactionTestUserCount(t, store, id), err)
	}
}

func TestOrdinaryMissingRowUpdateReturnsNoChange(t *testing.T) {
	store := openPeerTestDB(t)
	missingID := MustNewUUIDv7()
	var (
		acceptance    swarmion.AcceptanceHistory
		hasAcceptance bool
	)
	store.executeOperationForTest = func(ctx context.Context, runtime *swarmion.DatabaseRuntime, request swarmion.OperationRequest) swarmion.OperationResult {
		result := runtime.Execute(ctx, request)
		acceptance, hasAcceptance = result.Acceptance()
		return result
	}
	receipt, err := UpdateWithReceiptContext(context.Background(), store, transactionTestUserUpdate(missingID, "missing-row-noop"))
	if err != nil || !receipt.Committed || !receipt.HasExactEventIdentity() || transactionTestUserCount(t, store, missingID) != 0 {
		t.Fatalf("missing-row no-op receipt=%+v error=%v", receipt, err)
	}
	if !hasAcceptance || acceptance != swarmion.AcceptanceNoChange {
		t.Fatalf("missing-row no-op acceptance=%q present=%t, want %q", acceptance, hasAcceptance, swarmion.AcceptanceNoChange)
	}
}
