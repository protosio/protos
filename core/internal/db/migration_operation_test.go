package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strings"
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
	migrationsDir, filenames := embeddedMigrationBatchForTest(t)

	store, err := Open(workDir, databaseName, signer, link)
	if err != nil {
		t.Fatalf("open new migration database: %v", err)
	}
	if err := store.Init(); err != nil {
		_ = store.Close()
		t.Fatalf("initialize migration database: %v", err)
	}
	assertOperationRecoveryJournalEmpty(t, store, "completed initial migrations")
	operation, err := store.migrationBatchPublishedWriteOperation(migrationsDir, filenames)
	if err != nil {
		t.Fatalf("recover migration operation identity: %v", err)
	}

	assertSingleMigrationTransaction(t, store.TransactionMetrics())
	initialResult := requireMigrationOperationReceipt(t, store, operation)
	initialPublished := requireExactMigrationPublishedReceipt(t, operation, initialResult)
	initialObservation := waitForMigrationReceiptApplied(t, store, initialPublished)
	assertDurableMigrationHistory(
		t,
		store,
		operation,
		initialPublished.EventID,
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
	assertOperationRecoveryJournalEmpty(t, reopened, "completed restart migration recovery")

	if got := reopened.TransactionMetrics(); got != (TransactionMetricsSnapshot{}) {
		t.Fatalf("restart migration recovery started a backend write: %+v", got)
	}
	recoveredResult := requireMigrationOperationReceipt(t, reopened, operation)
	recoveredPublished := requireExactMigrationPublishedReceipt(t, operation, recoveredResult)
	if recoveredPublished.EventID != initialPublished.EventID ||
		recoveredPublished.PublishedRootHash != initialPublished.PublishedRootHash {
		t.Fatalf("recovered migration receipt=%+v, want original %+v", recoveredPublished, initialPublished)
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
	recoveredObservation := waitForMigrationReceiptApplied(t, reopened, recoveredPublished)
	assertDurableMigrationHistory(
		t,
		reopened,
		operation,
		recoveredPublished.EventID,
		recoveredObservation.Status.DurableCheckpointCommitID,
		migrationsDir,
		filenames,
	)
	assertMigrationHistoryRowCount(t, reopened, len(filenames))
}

func TestMigrationCatalogIdentityUsesOneBoundedBinaryPart(t *testing.T) {
	store := openMigrationDatabaseWithoutMigrations(t)
	migrations := make(fstest.MapFS, 80)
	filenames := make([]string, 0, 80)
	for index := 0; index < 80; index++ {
		filename := fmt.Sprintf("migration-%03d.sql", index)
		filenames = append(filenames, filename)
		migrations[filename] = &fstest.MapFile{Data: []byte(fmt.Sprintf("SELECT %d;", index))}
	}
	operation, err := store.migrationBatchPublishedWriteOperation(migrations, filenames)
	if err != nil {
		t.Fatalf("build 80-file migration identity: %v", err)
	}
	if err := operation.Validate(); err != nil {
		t.Fatalf("validate 80-file migration identity: %v", err)
	}

	reversed := append([]string(nil), filenames...)
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	framed, err := encodeMigrationCatalogIntent(migrations, reversed)
	if err != nil {
		t.Fatal(err)
	}
	recomputed, err := swarmionapp.NewOperationIdentity(operation.Key(), OperationSchemaMigrationBatch, framed)
	if err != nil {
		t.Fatal(err)
	}
	if recomputed.Key() != operation.Key() || recomputed.IntentDigest() != operation.IntentDigest() {
		t.Fatal("migration catalog framing depends on caller filename order")
	}
}

func assertOperationRecoveryJournalEmpty(t *testing.T, store *DB, reason string) {
	t.Helper()
	operations, err := store.loadPublishedWriteOperations()
	if err != nil {
		t.Fatalf("load operation recovery journal after %s: %v", reason, err)
	}
	if len(operations) != 0 {
		t.Fatalf("operation recovery journal after %s contains %d stale records: %+v", reason, len(operations), operations)
	}
}

func TestMigrationBatchAdoptsCompatiblePreContractSchemaWithoutDuplicateDDL(t *testing.T) {
	store := openMigrationDatabaseWithoutMigrations(t)
	migrationsDir, filenames := embeddedMigrationBatchForTest(t)
	operation, err := store.migrationBatchPublishedWriteOperation(migrationsDir, filenames)
	if err != nil {
		t.Fatalf("build migration operation: %v", err)
	}

	// Model the last schema that could exist before the migration operation
	// owned its history atomically. Applying the first migration directly proves
	// the declarative preflight adopts compatible objects instead of publishing
	// duplicate DDL.
	preContractFilenames := []string{"protos_01_tables.sql"}
	preContractStatements := make([]preparedWriteStatement, 0)
	for _, filename := range preContractFilenames {
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
				preContractStatements = append(preContractStatements, preparedWriteStatement{SQL: piece})
			}
		}
	}
	publishPreContractMigrationSchema(t, store, preContractStatements)

	if err := store.runMigrations(context.Background()); err != nil {
		t.Fatalf("adopt compatible pre-contract schema: %v", err)
	}
	assertMigrationHistoryRowCount(t, store, len(filenames))
	_ = requireMigrationOperationReceipt(t, store, operation)

	var (
		startedAt sql.NullTime
		duration  int64
	)
	if err := store.ReadRows(
		context.Background(),
		"SELECT started_at, time_taken_ns FROM sqddl_history WHERE filename = ?",
		[]any{filenames[0]},
		func(rows *sql.Rows) error {
			if !rows.Next() {
				return sql.ErrNoRows
			}
			return rows.Scan(&startedAt, &duration)
		},
	); err != nil {
		t.Fatalf("read migration timing: %v", err)
	}
	if !startedAt.Valid || duration < 0 {
		t.Fatalf("migration timing started_at=%v duration=%d", startedAt, duration)
	}
}

func TestMigrationBatchRejectsIncompatibleSchemaWithoutReceipt(t *testing.T) {
	store := openMigrationDatabaseWithoutMigrations(t)
	migrationsDir, filenames := embeddedMigrationBatchForTest(t)
	operation, buildErr := store.migrationBatchPublishedWriteOperation(migrationsDir, filenames)
	if buildErr != nil {
		t.Fatalf("build migration operation: %v", buildErr)
	}

	_, err := store.executePublishedWriteOperationContext(
		context.Background(),
		operation,
		"incompatible migration schema",
		func(ctx context.Context, executor sqlContextExecer) error {
			_, executeErr := executor.ExecContext(ctx, `CREATE TABLE machines (
id BINARY(16) NOT NULL PRIMARY KEY,
name INT NOT NULL
)`)
			return executeErr
		},
	)
	if err == nil {
		t.Fatal("incompatible migration unexpectedly succeeded")
	}
	resolution, lookupErr := store.LookupPublishedWriteOperation(context.Background(), operation)
	if lookupErr != nil {
		t.Fatalf("lookup rejected migration operation: %v", lookupErr)
	}
	if resolution.Disposition() != swarmionapp.OperationRetryPermitted {
		t.Fatalf("rejected migration disposition=%s diagnostic=%v, want retry permitted", resolution.Disposition(), resolution.Diagnostic())
	}
	historyColumns, inspectErr := loadMigrationTableColumns(context.Background(), store.GetSqlDB(), "sqddl_history")
	if inspectErr != nil {
		t.Fatalf("inspect migration history after rejection: %v", inspectErr)
	}
	if len(historyColumns) != 0 {
		t.Fatalf("incompatible migration partially created sqddl_history: %+v", historyColumns)
	}
	machineColumns, inspectErr := loadMigrationTableColumns(context.Background(), store.GetSqlDB(), "machines")
	if inspectErr != nil {
		t.Fatalf("inspect machines after rejection: %v", inspectErr)
	}
	if len(machineColumns) != 0 {
		t.Fatalf("incompatible migration partially created machines: %+v", machineColumns)
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
	operation, err := store.migrationBatchPublishedWriteOperation(migrationsDir, filenames)
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
	resolution, lookupErr := store.LookupPublishedWriteOperation(context.Background(), operation)
	if lookupErr != nil {
		t.Fatalf("lookup failed migration operation: %v", lookupErr)
	}
	if resolution.Disposition() != swarmionapp.OperationRetryPermitted {
		t.Fatalf("failed migration disposition=%s diagnostic=%v, want retry permitted", resolution.Disposition(), resolution.Diagnostic())
	}
}

func openMigrationDatabaseWithoutMigrations(t *testing.T) *DB {
	t.Helper()
	workDir, databaseName, signer, link := newMigrationOperationTestDatabase(t)
	store, err := Open(workDir, databaseName, signer, link)
	if err != nil {
		t.Fatalf("open raw migration database: %v", err)
	}
	ctx := context.Background()
	if err := store.openMu.LockContext(ctx); err != nil {
		_ = store.Close()
		t.Fatalf("lock raw migration database initialization: %v", err)
	}
	openErr := store.openSwarmionLocked(ctx, nil)
	store.openMu.Unlock()
	if openErr != nil {
		_ = store.Close()
		t.Fatalf("initialize raw migration database: %v", openErr)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func publishPreContractMigrationSchema(t *testing.T, store *DB, statements []preparedWriteStatement) {
	t.Helper()
	receipt, err := store.executeOrdinaryPublishedWriteContext(
		context.Background(),
		"pre-contract migration schema",
		false,
		false,
		statements,
	)
	if err != nil {
		t.Fatalf("publish pre-contract migration schema: %v", err)
	}
	if !receipt.Committed {
		t.Fatalf("pre-contract schema write was not locally published: %+v", receipt)
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

func embeddedMigrationBatchForTest(t *testing.T) (fs.FS, []string) {
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
	return migrationsDir, filenames
}

func requireMigrationOperationReceipt(
	t *testing.T,
	store *DB,
	operation PublishedWriteOperation,
) swarmionapp.OperationResult {
	t.Helper()
	resolution, err := store.LookupPublishedWriteOperation(context.Background(), operation)
	if err != nil {
		t.Fatalf("resolve migration operation receipt: %v", err)
	}
	if resolution.Disposition() != swarmionapp.OperationAccepted {
		t.Fatalf("migration operation disposition=%s diagnostic=%v, want accepted", resolution.Disposition(), resolution.Diagnostic())
	}
	if _, err := PublishedWriteReceiptFromResult(operation, resolution); err != nil {
		t.Fatalf("migration operation result is not exact: %v", err)
	}
	return resolution
}

func requireExactMigrationPublishedReceipt(
	t *testing.T,
	operation PublishedWriteOperation,
	resolution swarmionapp.OperationResult,
) PublishedWriteReceipt {
	t.Helper()
	published, err := PublishedWriteReceiptFromResult(operation, resolution)
	if err != nil {
		t.Fatalf("convert migration operation resolution: %v", err)
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
		operation.Key(),
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

func assertSingleMigrationTransaction(t *testing.T, metrics TransactionMetricsSnapshot) {
	t.Helper()
	if metrics.TransactionsStarted != 1 || metrics.CommitsAttempted != 1 ||
		metrics.CommitsSucceeded != 1 || metrics.TypedConflicts != 0 ||
		metrics.OperationTransactionsAttempted != 1 ||
		metrics.OperationTransactionsExecuted != 1 ||
		metrics.OperationTransactionsAlreadyAccepted != 0 ||
		metrics.OperationTransactionsNoChange != 0 ||
		metrics.OperationTransactionsFailed != 0 ||
		metrics.OperationWorkspaceDirtyOutcomes != 0 {
		t.Fatalf("migration transaction metrics=%+v", metrics)
	}
}
