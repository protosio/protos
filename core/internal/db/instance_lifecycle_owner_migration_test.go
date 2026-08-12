package db

import (
	"context"
	"io/fs"
	"testing"
)

func TestInstanceLifecycleOwnerMigrationBackfillsOnlyProvenAuthority(t *testing.T) {
	workDir, databaseName, signer, link := newMigrationOperationTestDatabase(t)
	store, err := Open(workDir, databaseName, signer, link)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Init(); err != nil {
		t.Fatal(err)
	}
	sqldb := store.GetSqlDB()
	ctx := context.Background()
	if _, err := sqldb.ExecContext(ctx, `ALTER TABLE cloud_machines_metadata DROP COLUMN lifecycle_owner_peer_id`); err != nil {
		t.Fatalf("restore pre-v0.1 metadata shape: %v", err)
	}

	deleteAuthorizedID := MustNewUUIDv7()
	deploymentOwnedID := MustNewUUIDv7()
	ambiguousID := MustNewUUIDv7()
	for _, id := range []string{deleteAuthorizedID, deploymentOwnedID, ambiguousID} {
		if _, err := sqldb.ExecContext(ctx, `INSERT INTO cloud_machines_metadata
(id, cloud_id, provider_resource_id, public_ip, location, architecture, public_key)
VALUES (?, 'test', 'provider-id', '', 'test', '', '')`, MustUUIDBytes(id)); err != nil {
			t.Fatalf("insert legacy metadata %s: %v", id, err)
		}
	}
	insertTask := func(id, subjectID, owner string) {
		t.Helper()
		if _, err := sqldb.ExecContext(ctx, `INSERT INTO tasks
(id, task_stream, subject_type, subject_id, owner_peer_id, status, title, message, progress, payload, result, attempts, max_attempts, created_at, updated_at)
VALUES (?, 'provisioners.instance.deploy', 'instance', ?, ?, 'succeeded', 'deploy', 'done', 100, '{}', '{}', 1, 1, '2026-08-11T00:00:00Z', '2026-08-11T00:00:00Z')`,
			MustUUIDBytes(id), subjectID, owner); err != nil {
			t.Fatalf("insert legacy deployment task: %v", err)
		}
	}
	insertTask(MustNewUUIDv7(), deleteAuthorizedID, "peer-deployment")
	insertTask(MustNewUUIDv7(), deploymentOwnedID, " peer-deployment-only ")
	insertTask(MustNewUUIDv7(), ambiguousID, "peer-a")
	insertTask(MustNewUUIDv7(), ambiguousID, "peer-b")
	// A malformed historical subject must remain inert rather than aborting the
	// entire ALTER/backfill transaction.
	insertTask(MustNewUUIDv7(), "not-a-uuid", "peer-malformed")

	factTaskID := MustNewUUIDv7()
	if _, err := sqldb.ExecContext(ctx, `INSERT INTO task_operation_facts
(id, task_id, fact_kind, operation_key, intent_digest, author_peer_id, subject_type, subject_id, payload)
VALUES (?, ?, 'instance_peer_drain_authorized_v1', 'operation-key', ?, ' peer-delete-author ', 'instance', ?, '{}')`,
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		MustUUIDBytes(factTaskID),
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		deleteAuthorizedID,
	); err != nil {
		t.Fatalf("insert historical P fact: %v", err)
	}

	migrationsDir, err := fs.Sub(rootDir, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.applyMigration(ctx, sqldb, sqldb, migrationsDir, "protos_02_instance_lifecycle_owner.sql"); err != nil {
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
		if err := sqldb.QueryRowContext(ctx, `SELECT lifecycle_owner_peer_id FROM cloud_machines_metadata WHERE id = ?`, MustUUIDBytes(test.id)).Scan(&got); err != nil {
			t.Fatalf("read migrated owner %s: %v", test.id, err)
		}
		if got != test.want {
			t.Fatalf("migrated owner %s=%q, want %q", test.id, got, test.want)
		}
	}
}
