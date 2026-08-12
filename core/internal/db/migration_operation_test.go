package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"io/fs"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/dolthub/vitess/go/vt/sqlparser"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
	swarmiontransport "github.com/nustiueudinastea/swarmion/transports"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/testswarmion"
)

func TestMigrationBatchReceiptSurvivesRestartWithoutRepublishing(t *testing.T) {
	workDir, databaseName, signer, link := newMigrationOperationTestDatabase(t)
	migrationsDir, filenames, operation := embeddedMigrationBatchForTest(t)

	store, err := Open(workDir, databaseName, signer, link)
	if err != nil {
		t.Fatalf("open new migration database: %v", err)
	}
	if err := store.Init(); err != nil {
		_ = store.Close()
		t.Fatalf("initialize migration database: %v", err)
	}

	initialMetrics := store.TransactionMetrics()
	assertSingleMigrationTransaction(t, initialMetrics, false)
	initialReceipt := requireMigrationOperationReceipt(t, store, operation)
	initialPublished := requireExactMigrationPublishedReceipt(t, initialReceipt)
	initialObservation := waitForMigrationReceiptApplied(t, store, initialPublished)
	assertDurableMigrationHistory(
		t,
		store,
		operation,
		initialReceipt.EventID,
		initialObservation.Status.DurableCheckpointCommitID,
		migrationsDir,
		filenames,
	)
	initialStatus, ok := store.SwarmionStatus()
	if !ok {
		_ = store.Close()
		t.Fatal("read initial Swarmion status")
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close initialized migration database: %v", err)
	}

	reopened, err := Open(workDir, databaseName, signer, link)
	if err != nil {
		t.Fatalf("reopen migration database: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })

	if got := reopened.TransactionMetrics(); got != (TransactionMetricsSnapshot{}) {
		t.Fatalf("restart migration recovery started a backend write transaction: %+v", got)
	}
	recoveredReceipt := requireMigrationOperationReceipt(t, reopened, operation)
	if recoveredReceipt.EventID != initialReceipt.EventID ||
		recoveredReceipt.PublishedRootHash != initialReceipt.PublishedRootHash ||
		recoveredReceipt.EventDigest != initialReceipt.EventDigest {
		t.Fatalf("recovered migration receipt=%+v, want original %+v", recoveredReceipt, initialReceipt)
	}
	recoveredStatus, ok := reopened.SwarmionStatus()
	if !ok {
		t.Fatal("read reopened Swarmion status")
	}
	if recoveredStatus.CheckpointEventCount != initialStatus.CheckpointEventCount {
		t.Fatalf(
			"restart changed checkpoint event count from %d to %d; migration was republished",
			initialStatus.CheckpointEventCount,
			recoveredStatus.CheckpointEventCount,
		)
	}
	recoveredPublished := requireExactMigrationPublishedReceipt(t, recoveredReceipt)
	recoveredObservation := waitForMigrationReceiptApplied(t, reopened, recoveredPublished)
	assertDurableMigrationHistory(
		t,
		reopened,
		operation,
		recoveredReceipt.EventID,
		recoveredObservation.Status.DurableCheckpointCommitID,
		migrationsDir,
		filenames,
	)
	assertMigrationHistoryRowCount(t, reopened, len(filenames))
}

func TestMigrationBatchRestartResumesExecutedPendingReceiptWithoutReexecution(t *testing.T) {
	workDir, databaseName, signer, link := newMigrationOperationTestDatabase(t)
	t.Setenv("SWARMION_CHECKPOINT_SCHEDULER_PROFILE", "")
	t.Setenv("SWARMION_CONTINUOUS_EVENT_DEADLINE", "3s")
	_, filenames, operation := embeddedMigrationBatchForTest(t)

	store, err := Open(workDir, databaseName, signer, link)
	if err != nil {
		t.Fatalf("open new migration database: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	var (
		accepted              atomic.Bool
		lookupFailureInjected atomic.Bool
		transactionBodies     atomic.Int32
		acceptedReceipt       swarmionapp.BranchOperationReceipt
	)
	store.runOperationTransactionForTest = func(
		ctx context.Context,
		sqldb *sql.DB,
		request swarmionapp.OperationTransactionRequest,
	) (swarmionapp.OperationTransactionResult, error) {
		if request.Operation.Key != operation.Key || request.Operation.IntentDigest != operation.IntentDigest {
			return swarmionapp.RunOperationTransaction(ctx, sqldb, request)
		}
		transactionBodies.Add(1)
		result, err := swarmionapp.RunOperationTransaction(ctx, sqldb, request)
		if err != nil {
			return result, err
		}
		if result.Outcome != swarmionapp.OperationTransactionOutcomeExecuted || result.Receipt == nil {
			return result, errors.New("injected migration crash window requires a newly executed receipt")
		}
		acceptedReceipt = *result.Receipt
		accepted.Store(true)
		return swarmionapp.OperationTransactionResult{}, errors.New("injected process loss after executed result and before receipt-status wait")
	}
	store.lookupPublishedWriteForTest = func(
		ctx context.Context,
		candidate PublishedWriteOperation,
	) (swarmionapp.BranchOperationReceipt, error) {
		if candidate == operation && accepted.Load() && lookupFailureInjected.CompareAndSwap(false, true) {
			return swarmionapp.BranchOperationReceipt{}, errors.New("injected receipt-response loss before migration status wait")
		}
		return directOperationReceiptLookup(store, ctx, candidate)
	}

	if err := store.Init(); err == nil {
		t.Fatal("initialize unexpectedly survived the injected post-execution crash window")
	}
	if !accepted.Load() || !lookupFailureInjected.Load() {
		t.Fatalf(
			"crash window was not reached: accepted=%t lookup_failure=%t",
			accepted.Load(),
			lookupFailureInjected.Load(),
		)
	}
	if got := transactionBodies.Load(); got != 1 {
		t.Fatalf("migration transaction bodies=%d before restart, want exactly one", got)
	}
	if acceptedReceipt.Resolution != swarmionapp.BranchOperationReceiptFound ||
		strings.TrimSpace(acceptedReceipt.EventID) == "" ||
		strings.TrimSpace(acceptedReceipt.PublishedRootHash) == "" {
		t.Fatalf("executed migration did not return an exact receipt: %+v", acceptedReceipt)
	}
	if store.Initialized() {
		t.Fatal("failed migration finalization left database initialized")
	}
	if store.GetSqlDB() != nil {
		t.Fatal("failed migration finalization left SQL database reachable")
	}
	if _, ok := store.SwarmionStatus(); ok {
		t.Fatal("failed migration finalization left Swarmion runtime reachable")
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close already-cleaned migration database in pending crash window: %v", err)
	}

	reopened, err := Open(workDir, databaseName, signer, link)
	if err != nil {
		t.Fatalf("reopen migration database from pending receipt: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })

	if got := reopened.TransactionMetrics(); got != (TransactionMetricsSnapshot{}) {
		t.Fatalf("restart re-executed the migration operation instead of resuming its receipt: %+v", got)
	}
	recovered := requireMigrationOperationReceipt(t, reopened, operation)
	if recovered.EventID != acceptedReceipt.EventID ||
		recovered.PublishedRootHash != acceptedReceipt.PublishedRootHash ||
		recovered.EventDigest != acceptedReceipt.EventDigest {
		t.Fatalf("recovered migration receipt=%+v, want executed receipt %+v", recovered, acceptedReceipt)
	}
	if got := transactionBodies.Load(); got != 1 {
		t.Fatalf("migration transaction bodies=%d after restart, want no re-execution", got)
	}
	assertMigrationHistoryRowCount(t, reopened, len(filenames))
	status, ok := reopened.SwarmionStatus()
	if !ok {
		t.Fatal("read Swarmion status after pending migration recovery")
	}
	if status.CheckpointEventCount != 1 {
		t.Fatalf("checkpoint event count=%d, want the sole pre-restart migration event", status.CheckpointEventCount)
	}
}

func TestMigrationBatchTracksExactReceiptAcrossUncertainCommitAndLookupFailure(t *testing.T) {
	workDir, databaseName, signer, link := newMigrationOperationTestDatabase(t)
	migrationsDir, filenames, operation := embeddedMigrationBatchForTest(t)

	store, err := Open(workDir, databaseName, signer, link)
	if err != nil {
		t.Fatalf("open new migration database: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	var (
		accepted              atomic.Bool
		lookupFailureInjected atomic.Bool
		transactionBodies     atomic.Int32
		uncertainReceipt      PublishedWriteReceipt
	)
	store.runOperationTransactionForTest = func(
		ctx context.Context,
		sqldb *sql.DB,
		request swarmionapp.OperationTransactionRequest,
	) (swarmionapp.OperationTransactionResult, error) {
		if request.Operation.Key != operation.Key || request.Operation.IntentDigest != operation.IntentDigest {
			return swarmionapp.RunOperationTransaction(ctx, sqldb, request)
		}
		transactionBodies.Add(1)
		result, err := swarmionapp.RunOperationTransaction(ctx, sqldb, request)
		if err != nil {
			return result, err
		}
		if result.Outcome != swarmionapp.OperationTransactionOutcomeExecuted || result.Receipt == nil {
			return result, errors.New("injected migration response uncertainty requires a newly executed receipt")
		}
		published, err := PublishedWriteReceiptFromOperation(*result.Receipt)
		if err != nil {
			return result, err
		}
		published.Committed = false
		published.OutcomeUncertain = true
		published.PendingCheckpoint = true
		uncertainReceipt = published
		accepted.Store(true)
		uncertainErr := &swarmionapp.CommitOutcomeUncertainError{
			EventReceipt: swarmionapp.BranchEventReceiptStatusRequest{
				EventID:                   result.Receipt.EventID,
				ExpectedPublishedRootHash: result.Receipt.PublishedRootHash,
			},
			OperationReceipt: &swarmionapp.BranchOperationReceiptRequest{
				Key:          operation.Key,
				IntentDigest: operation.IntentDigest,
				AuthorPeerID: operation.AuthorPeerID,
			},
			ReceiptPersistence: swarmionapp.CommitReceiptPersistenceUnknown,
			Cause:              errors.New("injected migration response uncertainty after accepted operation transaction"),
		}
		return swarmionapp.OperationTransactionResult{}, &swarmionapp.OperationTransactionError{
			Phase:          swarmionapp.OperationTransactionPhaseCommit,
			StatementIndex: -1,
			CommitStatus:   swarmionapp.OperationTransactionCommitReturnedError,
			RollbackStatus: swarmionapp.OperationTransactionRollbackNotAttempted,
			Cause:          uncertainErr,
		}
	}
	store.lookupPublishedWriteForTest = func(
		ctx context.Context,
		candidate PublishedWriteOperation,
	) (swarmionapp.BranchOperationReceipt, error) {
		if candidate == operation && accepted.Load() && lookupFailureInjected.CompareAndSwap(false, true) {
			return swarmionapp.BranchOperationReceipt{}, errors.New("injected post-commit migration receipt lookup failure")
		}
		return directOperationReceiptLookup(store, ctx, candidate)
	}

	if err := store.Init(); err != nil {
		t.Fatalf("initialize after accepted uncertain migration response: %v", err)
	}
	if !lookupFailureInjected.Load() {
		t.Fatal("post-commit migration receipt lookup failure was not injected")
	}
	if got := transactionBodies.Load(); got != 1 {
		t.Fatalf("migration transaction bodies=%d, want exactly one without replay", got)
	}
	if !uncertainReceipt.HasExactEventIdentity() || uncertainReceipt.Committed || !uncertainReceipt.OutcomeUncertain {
		t.Fatalf("uncertain migration response did not retain its exact event address: %+v", uncertainReceipt)
	}
	assertSingleMigrationTransaction(t, store.TransactionMetrics(), true)

	resolved := requireMigrationOperationReceipt(t, store, operation)
	if resolved.EventID != uncertainReceipt.EventID || resolved.PublishedRootHash != uncertainReceipt.PublishedRootHash {
		t.Fatalf("resolved migration receipt=%+v, want uncertain exact receipt %+v", resolved, uncertainReceipt)
	}
	published := requireExactMigrationPublishedReceipt(t, resolved)
	observation := waitForMigrationReceiptApplied(t, store, published)
	assertDurableMigrationHistory(
		t,
		store,
		operation,
		resolved.EventID,
		observation.Status.DurableCheckpointCommitID,
		migrationsDir,
		filenames,
	)
	assertMigrationHistoryRowCount(t, store, len(filenames))
	status, ok := store.SwarmionStatus()
	if !ok {
		t.Fatal("read Swarmion status after uncertain migration commit")
	}
	if status.CheckpointEventCount != 1 {
		t.Fatalf("checkpoint event count=%d, want one accepted migration event without replay", status.CheckpointEventCount)
	}
}

func TestMigrationBatchAdoptsCompatibleLegacySchemaWithoutDuplicateDDL(t *testing.T) {
	store := openMigrationDatabaseWithoutMigrations(t)
	migrationsDir, filenames, operation := embeddedMigrationBatchForTest(t)

	// Model the last schema that could exist before the v0.1 lifecycle-owner
	// transition. The migration operation is atomic with its history rows, so a
	// database containing the v0.1 ALTER but no corresponding history is not a
	// supported crash boundary. Applying only v0.0 here proves that migration
	// adoption skips its compatible legacy DDL and then executes v0.1 normally.
	legacyFilenames := []string{"protos_01_tables.sql"}
	legacyStatements := make([]preparedWriteStatement, 0)
	for _, filename := range legacyFilenames {
		contents, err := fs.ReadFile(migrationsDir, filename)
		if err != nil {
			t.Fatalf("read embedded migration %s: %v", filename, err)
		}
		pieces, err := sqlparser.SplitStatementToPieces(string(contents))
		if err != nil {
			t.Fatalf("split embedded migration %s: %v", filename, err)
		}
		for _, piece := range pieces {
			if piece = strings.TrimSpace(piece); piece != "" {
				legacyStatements = append(legacyStatements, preparedWriteStatement{SQL: piece})
			}
		}
	}
	publishLegacyMigrationSchema(t, store, legacyStatements)

	var (
		helperCalls atomic.Int32
		request     swarmionapp.OperationTransactionRequest
	)
	store.runOperationTransactionForTest = func(
		ctx context.Context,
		sqldb *sql.DB,
		candidate swarmionapp.OperationTransactionRequest,
	) (swarmionapp.OperationTransactionResult, error) {
		helperCalls.Add(1)
		request = candidate
		return swarmionapp.RunOperationTransaction(ctx, sqldb, candidate)
	}
	if err := store.runMigrations(context.Background()); err != nil {
		t.Fatalf("adopt compatible legacy schema: %v", err)
	}
	if got := helperCalls.Load(); got != 1 {
		t.Fatalf("migration helper calls=%d, want exactly one", got)
	}
	if request.Operation.Key != operation.Key || request.Operation.IntentDigest != operation.IntentDigest {
		t.Fatalf("migration operation changed during legacy adoption: got=%+v want=%+v", request.Operation, operation)
	}
	var historyCreates int
	for _, statement := range request.Statements {
		normalized := strings.ToUpper(strings.Join(strings.Fields(statement.Query), " "))
		switch {
		case strings.HasPrefix(normalized, "CREATE TABLE"):
			historyCreates++
			if !strings.Contains(normalized, "SQDDL_HISTORY") {
				t.Fatalf("compatible legacy table was submitted again: %s", statement.Query)
			}
		case strings.HasPrefix(normalized, "CREATE INDEX"):
			t.Fatalf("compatible legacy index was submitted again: %s", statement.Query)
		}
	}
	if historyCreates != 1 {
		t.Fatalf("history table creates=%d, want one genuinely needed CREATE TABLE", historyCreates)
	}
	assertMigrationHistoryRowCount(t, store, len(filenames))
	_ = requireMigrationOperationReceipt(t, store, operation)

	var (
		startedAt sql.NullTime
		duration  int64
	)
	if err := store.ReadRows(context.Background(), "SELECT started_at, time_taken_ns FROM sqddl_history WHERE filename = ?", []any{filenames[0]}, func(rows *sql.Rows) error {
		if !rows.Next() {
			return sql.ErrNoRows
		}
		return rows.Scan(&startedAt, &duration)
	}); err != nil {
		t.Fatalf("read migration timing: %v", err)
	}
	if !startedAt.Valid || duration < 0 {
		t.Fatalf("migration timing started_at=%v duration=%d", startedAt, duration)
	}
}

func TestMigrationBatchRejectsIncompatibleLegacyObjectWithoutReceipt(t *testing.T) {
	store := openMigrationDatabaseWithoutMigrations(t)
	migrationsDir, filenames, operation := embeddedMigrationBatchForTest(t)
	// Model an old local workspace directly. The current declarative contract
	// correctly refuses to publish this incompatible schema, but migration
	// preflight must still fail closed if such a legacy workspace is opened.
	if _, err := store.GetSqlDB().ExecContext(context.Background(), `CREATE TABLE machines (
id BINARY(16) NOT NULL PRIMARY KEY,
name INT NOT NULL
)`); err != nil {
		t.Fatalf("stage incompatible legacy table: %v", err)
	}

	var helperCalls atomic.Int32
	store.runOperationTransactionForTest = func(
		ctx context.Context,
		sqldb *sql.DB,
		request swarmionapp.OperationTransactionRequest,
	) (swarmionapp.OperationTransactionResult, error) {
		helperCalls.Add(1)
		return swarmionapp.RunOperationTransaction(ctx, sqldb, request)
	}
	_, err := store.executePublishedWriteOperationContext(
		context.Background(),
		operation,
		"incompatible legacy migration",
		func(ctx context.Context, executor sqlContextExecer) error {
			if err := appendMigrationStatementIfNeeded(ctx, store.GetSqlDB(), executor, migrationHistoryCreateStatement); err != nil {
				return err
			}
			for _, filename := range filenames {
				if err := store.applyMigration(ctx, store.GetSqlDB(), executor, migrationsDir, filename); err != nil {
					return err
				}
			}
			return nil
		},
	)
	if !errors.Is(err, ErrMigrationSchemaConflict) {
		t.Fatalf("incompatible migration error=%v, want ErrMigrationSchemaConflict", err)
	}
	if got := helperCalls.Load(); got != 0 {
		t.Fatalf("operation helper ran %d times for incompatible preflight, want zero", got)
	}
	resolved, lookupErr := directOperationReceiptLookup(store, context.Background(), operation)
	if lookupErr != nil {
		t.Fatalf("lookup rejected migration operation: %v", lookupErr)
	}
	if resolved.Resolution != swarmionapp.BranchOperationReceiptAbsent {
		t.Fatalf("rejected migration receipt resolution=%q, want absent: %+v", resolved.Resolution, resolved)
	}
	historyColumns, inspectErr := loadMigrationTableColumns(context.Background(), store.GetSqlDB(), "sqddl_history")
	if inspectErr != nil {
		t.Fatalf("inspect migration history after rejection: %v", inspectErr)
	}
	if len(historyColumns) != 0 {
		t.Fatalf("incompatible preflight partially created sqddl_history: %+v", historyColumns)
	}
}

func TestMigrationOperationApplyFailureRollsBackAllDDLAndHasNoReceipt(t *testing.T) {
	store := openMigrationDatabaseWithoutMigrations(t)
	migrationsDir := fstest.MapFS{
		"partial.sql": &fstest.MapFile{Data: []byte(`CREATE TABLE partial_atomic_migration (
id BINARY(16) NOT NULL PRIMARY KEY
);
INSERT INTO definitely_missing_migration_table (id) VALUES (1);`)},
	}
	filenames := []string{"partial.sql"}
	operation, err := migrationBatchPublishedWriteOperation(migrationsDir, filenames)
	if err != nil {
		t.Fatalf("build partial migration operation: %v", err)
	}

	_, err = store.executePublishedWriteOperationContext(
		context.Background(),
		operation,
		"partial migration rollback",
		func(ctx context.Context, executor sqlContextExecer) error {
			if err := appendMigrationStatementIfNeeded(ctx, store.GetSqlDB(), executor, migrationHistoryCreateStatement); err != nil {
				return err
			}
			return store.applyMigration(ctx, store.GetSqlDB(), executor, migrationsDir, filenames[0])
		},
	)
	if err == nil {
		t.Fatal("partial migration unexpectedly succeeded")
	}
	for _, tableName := range []string{"sqddl_history", "partial_atomic_migration"} {
		columns, inspectErr := loadMigrationTableColumns(context.Background(), store.GetSqlDB(), tableName)
		if inspectErr != nil {
			t.Fatalf("inspect table %q after rollback: %v", tableName, inspectErr)
		}
		if len(columns) != 0 {
			t.Fatalf("failed migration left partial table %q: %+v", tableName, columns)
		}
	}
	resolved, lookupErr := directOperationReceiptLookup(store, context.Background(), operation)
	if lookupErr != nil {
		t.Fatalf("lookup failed migration operation: %v", lookupErr)
	}
	if resolved.Resolution != swarmionapp.BranchOperationReceiptAbsent {
		t.Fatalf("failed migration receipt resolution=%q, want absent: %+v", resolved.Resolution, resolved)
	}
}

func openMigrationDatabaseWithoutMigrations(t *testing.T) *DB {
	t.Helper()
	workDir, databaseName, signer, link := newMigrationOperationTestDatabase(t)
	store, err := Open(workDir, databaseName, signer, link)
	if err != nil {
		t.Fatalf("open raw migration database: %v", err)
	}
	if err := store.openSwarmion(context.Background(), nil); err != nil {
		_ = store.Close()
		t.Fatalf("initialize raw migration database: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func publishLegacyMigrationSchema(t *testing.T, store *DB, statements []preparedWriteStatement) {
	t.Helper()
	receipt, err := store.executeOrdinaryPublishedWriteContext(
		context.Background(),
		"legacy migration schema",
		false,
		false,
		statements,
	)
	if err != nil {
		t.Fatalf("publish legacy migration schema: %v", err)
	}
	if !receipt.Committed {
		t.Fatalf("legacy schema write was not locally published: %+v", receipt)
	}
}

func newMigrationOperationTestDatabase(t *testing.T) (string, string, testSwarmionRawSigner, swarmiontransport.Link) {
	t.Helper()
	t.Setenv("SWARMION_CHECKPOINT_SCHEDULER_PROFILE", "single_node_development")
	cfg := config.Get()
	previousP2PPort := cfg.P2PPort
	cfg.P2PPort = 0
	t.Cleanup(func() { cfg.P2PPort = previousP2PPort })

	privateKey, publicKey, err := libp2pcrypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatalf("generate migration test signer: %v", err)
	}
	signer := testSwarmionRawSigner{
		privateKey: privateKey,
		publicKey:  publicKey,
	}
	return t.TempDir(), "protos_migration_operation_test", signer, testswarmion.NewBorrowedLink(t, signer)
}

func embeddedMigrationBatchForTest(t *testing.T) (fs.FS, []string, PublishedWriteOperation) {
	t.Helper()
	migrationsDir, err := fs.Sub(rootDir, "migrations")
	if err != nil {
		t.Fatalf("open embedded migrations: %v", err)
	}
	entries, err := fs.ReadDir(migrationsDir, ".")
	if err != nil {
		t.Fatalf("list embedded migrations: %v", err)
	}
	filenames := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() && strings.HasSuffix(name, ".sql") && !strings.HasSuffix(name, ".undo.sql") {
			filenames = append(filenames, name)
		}
	}
	sort.Strings(filenames)
	if len(filenames) == 0 {
		t.Fatal("embedded migration batch is empty")
	}
	operation, err := migrationBatchPublishedWriteOperation(migrationsDir, filenames)
	if err != nil {
		t.Fatalf("build embedded migration operation: %v", err)
	}
	return migrationsDir, filenames, operation
}

func requireMigrationOperationReceipt(
	t *testing.T,
	store *DB,
	operation PublishedWriteOperation,
) swarmionapp.BranchOperationReceipt {
	t.Helper()
	resolved, err := directOperationReceiptLookup(store, context.Background(), operation)
	if err != nil {
		t.Fatalf("resolve migration operation receipt: %v", err)
	}
	if resolved.Resolution != swarmionapp.BranchOperationReceiptFound ||
		strings.TrimSpace(resolved.EventID) == "" ||
		strings.TrimSpace(resolved.PublishedRootHash) == "" ||
		resolved.IntentDigest != operation.IntentDigest {
		t.Fatalf("migration operation receipt is not an exact found result: %+v", resolved)
	}
	return resolved
}

func requireExactMigrationPublishedReceipt(
	t *testing.T,
	resolved swarmionapp.BranchOperationReceipt,
) PublishedWriteReceipt {
	t.Helper()
	published, err := PublishedWriteReceiptFromOperation(resolved)
	if err != nil {
		t.Fatalf("convert migration operation receipt: %v", err)
	}
	if !published.Committed || !published.HasExactEventIdentity() {
		t.Fatalf("migration operation did not resolve an exact published receipt: %+v", published)
	}
	return published
}

func waitForMigrationReceiptApplied(
	t *testing.T,
	store *DB,
	published PublishedWriteReceipt,
) EventReceiptObservation {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	observation, err := store.WaitForPublishedWriteApplied(ctx, published, "migration operation receipt test")
	if err != nil {
		t.Fatalf("wait for migration event receipt: %v", err)
	}
	if !observation.Status.AppliedDurably || strings.TrimSpace(observation.Status.DurableCheckpointCommitID) == "" {
		t.Fatalf("migration event was not applied at an addressable durable checkpoint: %+v", observation)
	}
	return observation
}

func assertDurableMigrationHistory(
	t *testing.T,
	store *DB,
	operation PublishedWriteOperation,
	eventID string,
	checkpointCommitID string,
	migrationsDir fs.FS,
	filenames []string,
) {
	t.Helper()
	if err := store.validateMigrationHistoryAtCheckpoint(
		context.Background(),
		operation.Key,
		eventID,
		checkpointCommitID,
		migrationsDir,
		filenames,
	); err != nil {
		t.Fatalf("validate durable migration history: %v", err)
	}
}

func assertMigrationHistoryRowCount(t *testing.T, store *DB, want int) {
	t.Helper()
	var got int
	if err := store.ReadRows(
		context.Background(),
		"SELECT COUNT(*) FROM sqddl_history",
		nil,
		func(rows *sql.Rows) error {
			if !rows.Next() {
				return sql.ErrNoRows
			}
			return rows.Scan(&got)
		},
	); err != nil {
		t.Fatalf("count migration history rows: %v", err)
	}
	if got != want {
		t.Fatalf("migration history row count=%d, want %d", got, want)
	}
}

func assertSingleMigrationTransaction(t *testing.T, metrics TransactionMetricsSnapshot, uncertain bool) {
	t.Helper()
	wantSucceeded := uint64(1)
	wantFailed := uint64(0)
	wantUncertain := uint64(0)
	wantOperationExecuted := uint64(1)
	wantOperationFailed := uint64(0)
	if uncertain {
		wantSucceeded = 0
		wantFailed = 1
		wantUncertain = 1
		wantOperationExecuted = 0
		wantOperationFailed = 1
	}
	if metrics.TransactionsStarted != 1 || metrics.CommitsAttempted != 1 ||
		metrics.CommitsSucceeded != wantSucceeded || metrics.CommitsFailed != wantFailed ||
		metrics.UncertainEventReceiptsAfterCommitErr != wantUncertain ||
		metrics.RollbacksAttempted != 0 || metrics.TypedConflicts != 0 ||
		metrics.OperationTransactionsAttempted != 1 ||
		metrics.OperationTransactionsExecuted != wantOperationExecuted ||
		metrics.OperationTransactionsAlreadyAccepted != 0 ||
		metrics.OperationTransactionsNoChange != 0 ||
		metrics.OperationTransactionsFailed != wantOperationFailed ||
		metrics.OperationWorkspaceDirtyOutcomes != 0 {
		t.Fatalf("migration transaction metrics=%+v", metrics)
	}
}
