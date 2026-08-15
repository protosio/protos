package db

import (
	"context"
	"io/fs"
	"strings"
	"testing"

	"github.com/dolthub/vitess/go/vt/sqlparser"
)

func TestInstanceLifecycleOwnerMigrationBackfillsOnlyProvenAuthority(t *testing.T) {
	store := openMigrationDatabaseWithoutMigrations(t)
	ctx := context.Background()
	migrationsDir, err := fs.Sub(rootDir, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	legacySchema, err := fs.ReadFile(migrationsDir, "protos_01_tables.sql")
	if err != nil {
		t.Fatal(err)
	}
	legacyPieces, err := sqlparser.SplitStatementToPieces(string(legacySchema))
	if err != nil {
		t.Fatal(err)
	}
	legacyStatements := make([]preparedWriteStatement, 0, len(legacyPieces)+1)
	legacyStatements = append(legacyStatements, preparedWriteStatement{SQL: migrationHistoryCreateStatement})
	for _, piece := range legacyPieces {
		if piece = strings.TrimSpace(piece); piece != "" {
			legacyStatements = append(legacyStatements, preparedWriteStatement{SQL: piece})
		}
	}
	publishLegacyMigrationSchema(t, store, legacyStatements)

	deleteAuthorizedID := MustNewUUIDv7()
	deploymentOwnedID := MustNewUUIDv7()
	ambiguousID := MustNewUUIDv7()
	fixtureStatements := make([]preparedWriteStatement, 0, 10)
	for _, id := range []string{deleteAuthorizedID, deploymentOwnedID, ambiguousID} {
		fixtureStatements = append(fixtureStatements, preparedWriteStatement{SQL: `INSERT INTO cloud_machines_metadata
(id, cloud_id, provider_resource_id, public_ip, location, architecture, public_key)
VALUES (?, 'test', 'provider-id', '', 'test', '', '')`, Args: []any{MustUUIDBytes(id)}})
	}
	insertTask := func(id, subjectID, owner string) {
		t.Helper()
		fixtureStatements = append(fixtureStatements, preparedWriteStatement{SQL: `INSERT INTO tasks
(id, task_stream, subject_type, subject_id, owner_peer_id, status, title, message, progress, payload, result, attempts, max_attempts, created_at, updated_at)
VALUES (?, 'provisioners.instance.deploy', 'instance', ?, ?, 'succeeded', 'deploy', 'done', 100, '{}', '{}', 1, 1, '2026-08-11T00:00:00Z', '2026-08-11T00:00:00Z')`,
			Args: []any{MustUUIDBytes(id), subjectID, owner}})
	}
	insertTask(MustNewUUIDv7(), deleteAuthorizedID, "peer-deployment")
	insertTask(MustNewUUIDv7(), deploymentOwnedID, " peer-deployment-only ")
	insertTask(MustNewUUIDv7(), ambiguousID, "peer-a")
	insertTask(MustNewUUIDv7(), ambiguousID, "peer-b")
	// A malformed historical subject must remain inert rather than aborting the
	// entire ALTER/backfill transaction.
	insertTask(MustNewUUIDv7(), "not-a-uuid", "peer-malformed")

	factTaskID := MustNewUUIDv7()
	fixtureStatements = append(fixtureStatements, preparedWriteStatement{SQL: `INSERT INTO task_operation_facts
(id, task_id, fact_kind, operation_key, intent_digest, author_peer_id, subject_type, subject_id, payload)
VALUES (?, ?, 'instance_peer_drain_authorized_v1', 'operation-key', ?, ' peer-delete-author ', 'instance', ?, '{}')`,
		Args: []any{
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			MustUUIDBytes(factTaskID),
			"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			deleteAuthorizedID,
		}})
	if _, err := store.executeOrdinaryPublishedWriteContext(ctx, "legacy lifecycle-owner fixtures", false, false, fixtureStatements); err != nil {
		t.Fatalf("publish legacy lifecycle-owner fixtures: %v", err)
	}

	operationKey, err := NewPublishedWriteOperationKey()
	if err != nil {
		t.Fatal(err)
	}
	operation, err := NewPublishedWriteOperation(operationKey, "protos:test:instance-lifecycle-owner-migration:v1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.executePublishedWriteOperationContext(ctx, operation, "instance lifecycle-owner migration", func(ctx context.Context, executor sqlContextExecer) error {
		return store.applyMigration(ctx, store.GetSqlDB(), executor, migrationsDir, "protos_02_instance_lifecycle_owner.sql")
	}); err != nil {
		t.Fatalf("apply lifecycle-owner migration to legacy schema: %v", err)
	}

	for _, test := range []struct {
		id   string
		want string
	}{
		{deleteAuthorizedID, "peer-delete-author"},
		{deploymentOwnedID, "peer-deployment-only"},
		{ambiguousID, ""},
	} {
		var got string
		if err := store.GetSqlDB().QueryRowContext(ctx, `SELECT lifecycle_owner_peer_id FROM cloud_machines_metadata WHERE id = ?`, MustUUIDBytes(test.id)).Scan(&got); err != nil {
			t.Fatalf("read migrated owner %s: %v", test.id, err)
		}
		if got != test.want {
			t.Fatalf("migrated owner %s=%q, want %q", test.id, got, test.want)
		}
	}
}
