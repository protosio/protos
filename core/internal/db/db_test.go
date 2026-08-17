package db_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bokwoon95/sq"
	swarmionprotocol "github.com/nustiueudinastea/swarmion/protocol"
	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/testswarmion"
)

func TestOperationAwareDeletePublishesAllStatementsOnceAndReplaysReceipt(t *testing.T) {
	useSingleNodeDevelopmentScheduler(t)
	workDir := t.TempDir()
	key, err := pcrypto.GetLocalKey(workDir)
	if err != nil {
		t.Fatal(err)
	}
	store, err := db.Open(workDir, "protos_operation_delete_test", key, testswarmion.NewBorrowedLink(t, key))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Init(); err != nil {
		t.Fatal(err)
	}

	insertPair := func(userID string, organisationID string, suffix string) {
		t.Helper()
		receipt, err := db.InsertWithReceiptContext(
			context.Background(),
			store,
			func() sq.InsertQuery {
				user := sq.New[db.USER]("")
				return sq.InsertInto(user).ColumnValues(func(col *sq.Column) {
					col.SetBytes(user.ID, db.MustUUIDBytes(userID))
					col.SetString(user.USERNAME, "operation-user-"+suffix)
					col.SetString(user.NAME, "Operation User "+suffix)
					col.SetBool(user.IS_DISABLED, false)
				})
			},
			func() sq.InsertQuery {
				organisation := sq.New[db.ORGANISATION]("")
				return sq.InsertInto(organisation).ColumnValues(func(col *sq.Column) {
					col.SetBytes(organisation.ID, db.MustUUIDBytes(organisationID))
					col.SetString(organisation.NAME, "operation-organisation-"+suffix)
					col.SetString(organisation.CREATED_AT, time.Now().UTC().Format(time.RFC3339Nano))
				})
			},
		)
		if err != nil {
			t.Fatalf("insert operation fixtures: %v", err)
		}
		waitForPublishedEventApplied(t, store, receipt)
	}
	deletePair := func(userID string, organisationID string) []db.DeleteMapper {
		return []db.DeleteMapper{
			func() sq.DeleteQuery {
				user := sq.New[db.USER]("")
				return sq.DeleteFrom(user).Where(db.UUIDEq(user.ID, userID))
			},
			func() sq.DeleteQuery {
				organisation := sq.New[db.ORGANISATION]("")
				return sq.DeleteFrom(organisation).Where(db.UUIDEq(organisation.ID, organisationID))
			},
		}
	}
	rowCount := func(table string, id string) int {
		t.Helper()
		var count int
		if err := store.ReadRows(
			context.Background(),
			"SELECT COUNT(*) FROM "+table+" WHERE id = ?",
			[]any{db.MustUUIDBytes(id)},
			func(rows *sql.Rows) error {
				if !rows.Next() {
					return sql.ErrNoRows
				}
				return rows.Scan(&count)
			},
		); err != nil {
			t.Fatalf("count %s fixture: %v", table, err)
		}
		return count
	}

	firstUserID := db.MustNewUUIDv7()
	firstOrganisationID := db.MustNewUUIDv7()
	insertPair(firstUserID, firstOrganisationID, "first")
	operation, err := store.NewPublishedWriteOperation(
		"io.protos.tests.multi-delete/v1",
		[]byte(firstUserID),
		[]byte(firstOrganisationID),
	)
	if err != nil {
		t.Fatal(err)
	}
	first, err := db.DeleteWithOperationReceiptContext(
		context.Background(),
		store,
		operation,
		deletePair(firstUserID, firstOrganisationID)...,
	)
	if err != nil {
		t.Fatalf("publish operation delete: %v", err)
	}
	waitForPublishedEventApplied(t, store, first)
	if rowCount("users", firstUserID) != 0 || rowCount("organisations", firstOrganisationID) != 0 {
		t.Fatal("operation transaction did not execute every delete statement")
	}
	if !first.HasExactEventIdentity() || first.AuthorPeerID == "" || first.OperationIntentDigest != operation.IntentDigest() {
		t.Fatalf("operation receipt metadata is incomplete: %+v", first)
	}

	secondUserID := db.MustNewUUIDv7()
	secondOrganisationID := db.MustNewUUIDv7()
	insertPair(secondUserID, secondOrganisationID, "second")
	replayed, err := db.DeleteWithOperationReceiptContext(
		context.Background(),
		store,
		operation,
		deletePair(secondUserID, secondOrganisationID)...,
	)
	if err != nil {
		t.Fatalf("replay operation delete: %v", err)
	}
	if replayed.EventID != first.EventID || replayed.PublishedRootHash != first.PublishedRootHash {
		t.Fatalf("replayed receipt=%+v, want original %+v", replayed, first)
	}
	if rowCount("users", secondUserID) != 1 || rowCount("organisations", secondOrganisationID) != 1 {
		t.Fatal("operation replay executed its changed SQL body")
	}

}

func TestSQLReadConnectorPreservesBinaryAndTextResultTypes(t *testing.T) {
	useSingleNodeDevelopmentScheduler(t)
	workDir := t.TempDir()
	key, err := pcrypto.GetLocalKey(workDir)
	if err != nil {
		t.Fatal(err)
	}
	store, err := db.Open(workDir, "protos_sqlread_binary_test", key, testswarmion.NewBorrowedLink(t, key))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Init(); err != nil {
		t.Fatal(err)
	}

	wantBinary := []byte{0x00, 0xff, 0x80, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d}
	const wantText = "sqlread-text"
	_, err = db.InsertWithReceiptContext(context.Background(), store, func() sq.InsertQuery {
		user := sq.New[db.USER]("")
		return sq.InsertInto(user).ColumnValues(func(col *sq.Column) {
			col.SetBytes(user.ID, wantBinary)
			col.SetString(user.USERNAME, "sqlread-binary")
			col.SetString(user.NAME, wantText)
			col.SetBool(user.IS_DISABLED, false)
		})
	})
	if err != nil {
		t.Fatalf("insert binary fixture: %v", err)
	}

	assertValues := func(source string, gotBinary []byte, gotText string) {
		t.Helper()
		if !slices.Equal(gotBinary, wantBinary) {
			t.Fatalf("%s binary = %x, want %x", source, gotBinary, wantBinary)
		}
		if gotText != wantText {
			t.Fatalf("%s text = %q, want %q", source, gotText, wantText)
		}
	}

	var directBinary []byte
	var directText string
	if err := store.GetSqlDB().QueryRowContext(
		context.Background(),
		"SELECT id, name FROM users WHERE username = ?",
		"sqlread-binary",
	).Scan(&directBinary, &directText); err != nil {
		t.Fatalf("query through GetSqlDB: %v", err)
	}
	assertValues("GetSqlDB", directBinary, directText)

	if err := store.ReadRows(
		context.Background(),
		"SELECT id, name FROM users WHERE username = ?",
		[]any{"sqlread-binary"},
		func(rows *sql.Rows) error {
			if !rows.Next() {
				return sql.ErrNoRows
			}
			var gotBinary []byte
			var gotText string
			if err := rows.Scan(&gotBinary, &gotText); err != nil {
				return err
			}
			assertValues("ReadRows", gotBinary, gotText)
			return nil
		},
	); err != nil {
		t.Fatalf("query through ReadRows: %v", err)
	}
}

func useSingleNodeDevelopmentScheduler(t *testing.T) {
	t.Helper()
	t.Setenv("SWARMION_CHECKPOINT_SCHEDULER_PROFILE", "single_node_development")
}

// waitForPublishedEventApplied is intentionally test-only. Ordinary writes
// return after publication, while commit-history assertions require the exact
// event checkpoint to be applied in the materialized durable lineage first.
// Content dissent is kept visible and is never mislabeled full-root durable.
func waitForPublishedEventApplied(t *testing.T, store *db.DB, receipt db.PublishedWriteReceipt) db.EventReceiptObservation {
	t.Helper()
	if !receipt.Committed || strings.TrimSpace(receipt.EventID) == "" || strings.TrimSpace(receipt.PublishedRootHash) == "" {
		t.Fatalf("write did not return an exact event receipt: %+v", receipt)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	observation, err := store.WaitForPublishedWriteApplied(ctx, receipt, "test exact event application")
	if err != nil {
		t.Fatalf("wait for exact event application: %v", err)
	}
	if !observation.Status.AppliedDurably {
		t.Fatalf("event receipt did not reach applied_durably: %+v", observation)
	}
	switch observation.Status.ContentCoverage {
	case swarmionapp.BranchEventContentCoverageCovered:
		if !observation.Status.Durable {
			t.Fatalf("covered event receipt did not report full-root durability: %+v", observation)
		}
	case swarmionapp.BranchEventContentCoverageDissent:
		if observation.Status.Durable {
			t.Fatalf("content dissent was mislabeled full-root durable: %+v", observation)
		}
		t.Logf("event applied durably with content dissent; invariant assertion follows: %+v", observation.Status)
	case swarmionapp.BranchEventContentCoverageUnavailable:
		t.Logf("event applied durably with content proof unavailable; invariant assertion follows: %+v", observation.Status)
	default:
		t.Fatalf("event applied durably with invalid coverage: %+v", observation)
	}
	return observation
}

func TestSwarmionBackedDBInitAndWrite(t *testing.T) {
	useSingleNodeDevelopmentScheduler(t)
	cfg := config.Get()
	previousP2PPort := cfg.P2PPort
	cfg.P2PPort = 0
	t.Cleanup(func() {
		cfg.P2PPort = previousP2PPort
	})

	workDir := t.TempDir()
	key, err := pcrypto.GetLocalKey(workDir)
	if err != nil {
		t.Fatalf("get local key: %v", err)
	}

	store, err := db.Open(workDir, "protos_test", key, testswarmion.NewBorrowedLink(t, key))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer store.Close()

	if store.Initialized() {
		t.Fatal("new database should not be initialized before Init")
	}
	initStarted := time.Now()
	if err := store.Init(); err != nil {
		t.Fatalf("init db: %v", err)
	}
	t.Logf("backend initialization wall time=%s", time.Since(initStarted))
	if !store.Initialized() {
		t.Fatal("database should be initialized after Init")
	}
	manifestPath := filepath.Join(workDir, ".swarmion", "protos_test.swarm-manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read persisted swarmion manifest: %v", err)
	}
	var manifest swarmionprotocol.SwarmManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("parse persisted swarmion manifest: %v", err)
	}
	if manifest.InitialRootHash.String() == "" || manifest.InitialCommitID.String() == "" {
		t.Fatalf("persisted swarmion manifest missing initial boundary: %#v", manifest)
	}

	beforeWrite, ok := store.SwarmionStatus()
	if !ok {
		t.Fatal("read swarmion status before write")
	}
	userID := db.MustNewUUIDv7()
	writeStarted := time.Now()
	receipt, err := db.InsertWithReceiptContext(context.Background(), store, func() sq.InsertQuery {
		u := sq.New[db.USER]("")
		return sq.InsertInto(u).ColumnValues(func(col *sq.Column) {
			col.SetBytes(u.ID, db.MustUUIDBytes(userID))
			col.SetString(u.USERNAME, "alex")
			col.SetString(u.NAME, "Alex")
			col.SetBool(u.IS_DISABLED, false)
		})
	})
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	publicationElapsed := time.Since(writeStarted)
	if publicationElapsed > time.Second {
		t.Fatalf("published write returned after %s, want local prompt return", publicationElapsed)
	}
	writeObservation := waitForPublishedEventApplied(t, store, receipt)
	afterWrite, ok := store.SwarmionStatus()
	if !ok {
		t.Fatal("read swarmion status after durable write")
	}
	if got := afterWrite.CheckpointEventCount - beforeWrite.CheckpointEventCount; got != 1 {
		t.Fatalf("checkpoint event delta=%d, want one event without duplicate retry", got)
	}
	t.Logf("write publication=%s durable=%s", publicationElapsed, time.Since(writeStarted))

	var count int
	if err := store.ReadRows(context.Background(), "SELECT COUNT(*) FROM users WHERE username = 'alex'", nil, func(rows *sql.Rows) error {
		if !rows.Next() {
			return fmt.Errorf("query user count returned no rows")
		}
		return rows.Scan(&count)
	}); err != nil {
		t.Fatalf("query user count: %v", err)
	}
	if count != 1 {
		t.Fatalf("user count = %d, want 1", count)
	}
	var durableCount int
	if err := store.ReadRowsAsOf(
		context.Background(),
		writeObservation.Status.DurableCheckpointCommitID,
		"SELECT COUNT(*) FROM users AS OF ? WHERE username = ?",
		[]any{"alex"},
		func(rows *sql.Rows) error {
			if !rows.Next() {
				return fmt.Errorf("durable user invariant query returned no rows")
			}
			return rows.Scan(&durableCount)
		},
	); err != nil {
		t.Fatalf("query user invariant at durable checkpoint: %v", err)
	}
	if durableCount != 1 {
		t.Fatalf("durable checkpoint user count = %d, want 1", durableCount)
	}

	if _, err := store.GetLastCommit("main"); err != nil {
		t.Fatalf("get last commit: %v", err)
	}

	result, err := store.ExecuteSQL(context.Background(), "SELECT username, name FROM users WHERE username = 'alex'", 20)
	if err != nil {
		t.Fatalf("execute sql select: %v", err)
	}
	if len(result.Columns) != 2 || result.Columns[0] != "username" || result.Columns[1] != "name" {
		t.Fatalf("columns = %v, want username/name", result.Columns)
	}
	if len(result.Rows) != 1 || len(result.Rows[0].Cells) != 2 || result.Rows[0].Cells[0].Value != "alex" {
		t.Fatalf("rows = %#v, want alex row", result.Rows)
	}
	if _, err := store.ExecuteSQL(context.Background(), "INSERT INTO users (username) VALUES ('bad')", 20); err == nil {
		t.Fatal("expected ExecuteSQL to reject mutating SQL")
	}
	if _, err := store.ExecuteSQL(context.Background(), "SELECT username FROM users; SELECT name FROM users", 20); err == nil {
		t.Fatal("expected ExecuteSQL to reject multiple SQL statements")
	}
}

func TestSwarmionBackedDBRapidConsecutiveWrites(t *testing.T) {
	useSingleNodeDevelopmentScheduler(t)
	cfg := config.Get()
	previousP2PPort := cfg.P2PPort
	cfg.P2PPort = 0
	t.Cleanup(func() {
		cfg.P2PPort = previousP2PPort
	})

	workDir := t.TempDir()
	key, err := pcrypto.GetLocalKey(workDir)
	if err != nil {
		t.Fatalf("get local key: %v", err)
	}

	store, err := db.Open(workDir, "protos_test_rapid", key, testswarmion.NewBorrowedLink(t, key))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer store.Close()

	if err := store.Init(); err != nil {
		t.Fatalf("init db: %v", err)
	}

	for _, username := range []string{"first", "second"} {
		userID := db.MustNewUUIDv7()
		if _, err := db.InsertWithReceiptContext(context.Background(), store, func() sq.InsertQuery {
			u := sq.New[db.USER]("")
			return sq.InsertInto(u).ColumnValues(func(col *sq.Column) {
				col.SetBytes(u.ID, db.MustUUIDBytes(userID))
				col.SetString(u.USERNAME, username)
				col.SetString(u.NAME, username)
				col.SetBool(u.IS_DISABLED, false)
			})
		}); err != nil {
			t.Fatalf("insert %s user: %v", username, err)
		}
	}

	var count int
	if err := store.ReadRows(context.Background(), "SELECT COUNT(*) FROM users WHERE username IN ('first', 'second')", nil, func(rows *sql.Rows) error {
		if !rows.Next() {
			return fmt.Errorf("query user count returned no rows")
		}
		return rows.Scan(&count)
	}); err != nil {
		t.Fatalf("query user count: %v", err)
	}
	if count != 2 {
		t.Fatalf("user count = %d, want 2", count)
	}
}

func TestSwarmionBackedDBCommitDiffUsesContractSchema(t *testing.T) {
	useSingleNodeDevelopmentScheduler(t)
	cfg := config.Get()
	previousP2PPort := cfg.P2PPort
	cfg.P2PPort = 0
	t.Cleanup(func() {
		cfg.P2PPort = previousP2PPort
	})

	workDir := t.TempDir()
	key, err := pcrypto.GetLocalKey(workDir)
	if err != nil {
		t.Fatalf("get local key: %v", err)
	}

	store, err := db.Open(workDir, "protos_test_commit_diff", key, testswarmion.NewBorrowedLink(t, key))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer store.Close()

	if err := store.Init(); err != nil {
		t.Fatalf("init db: %v", err)
	}

	userID := db.MustNewUUIDv7()
	insertReceipt, err := db.InsertWithReceiptContext(context.Background(), store, func() sq.InsertQuery {
		u := sq.New[db.USER]("")
		return sq.InsertInto(u).ColumnValues(func(col *sq.Column) {
			col.SetBytes(u.ID, db.MustUUIDBytes(userID))
			col.SetString(u.USERNAME, "alex")
			col.SetString(u.NAME, "Alex")
			col.SetBool(u.IS_DISABLED, false)
		})
	})
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	waitForPublishedEventApplied(t, store, insertReceipt)

	insertCommit, err := store.GetLastCommit("main")
	if err != nil {
		t.Fatalf("get insert commit: %v", err)
	}
	insertDiff, err := store.GetCommitDiff(context.Background(), insertCommit.Hash, "")
	if err != nil {
		t.Fatalf("get insert diff: %v", err)
	}
	if !strings.Contains(insertDiff.CUE, "users:") ||
		!strings.Contains(insertDiff.CUE, strconv.Quote(userID)) ||
		!strings.Contains(insertDiff.CUE, `_op: "added"`) ||
		!strings.Contains(insertDiff.CUE, `username: {`) {
		t.Fatalf("insert diff cue missing expected user addition:\n%s", insertDiff.CUE)
	}
	if !strings.Contains(insertDiff.UnifiedDiff, "diff --cue a/users/") ||
		!strings.Contains(insertDiff.UnifiedDiff, "--- /dev/null") ||
		!strings.Contains(insertDiff.UnifiedDiff, `+	username: "alex"`) {
		t.Fatalf("insert unified diff missing expected user addition:\n%s", insertDiff.UnifiedDiff)
	}
	if !strings.Contains(insertDiff.SQL, "INSERT INTO `users`") ||
		!strings.Contains(insertDiff.SQL, "`username`") ||
		!strings.Contains(insertDiff.SQL, "'alex'") ||
		!strings.Contains(insertDiff.SQL, "UNHEX(REPLACE(") {
		t.Fatalf("insert SQL diff missing expected user insert:\n%s", insertDiff.SQL)
	}

	updateReceipt, err := db.UpdateWithReceiptContext(context.Background(), store, func() sq.UpdateQuery {
		u := sq.New[db.USER]("")
		return sq.Update(u).SetFunc(func(col *sq.Column) {
			col.SetString(u.NAME, "Alexander")
		}).Where(db.UUIDEq(u.ID, userID))
	})
	if err != nil {
		t.Fatalf("update user: %v", err)
	}
	waitForPublishedEventApplied(t, store, updateReceipt)

	updateCommit, err := store.GetLastCommit("main")
	if err != nil {
		t.Fatalf("get update commit: %v", err)
	}
	updateDiff, err := store.GetCommitDiff(context.Background(), updateCommit.Hash, "")
	if err != nil {
		t.Fatalf("get update diff: %v", err)
	}
	if !strings.Contains(updateDiff.CUE, `_op: "modified"`) ||
		!strings.Contains(updateDiff.CUE, `name: {`) ||
		!strings.Contains(updateDiff.CUE, `before: "Alex"`) ||
		!strings.Contains(updateDiff.CUE, `after: "Alexander"`) ||
		strings.Contains(updateDiff.CUE, `username: {`) {
		t.Fatalf("update diff cue missing expected name-only change:\n%s", updateDiff.CUE)
	}
	if !strings.Contains(updateDiff.UnifiedDiff, "diff --cue a/users/") ||
		!strings.Contains(updateDiff.UnifiedDiff, `-	name: "Alex"`) ||
		!strings.Contains(updateDiff.UnifiedDiff, `+	name: "Alexander"`) {
		t.Fatalf("update unified diff missing expected name change:\n%s", updateDiff.UnifiedDiff)
	}
	if !strings.Contains(updateDiff.SQL, "UPDATE `users` SET `name` = 'Alexander'") ||
		!strings.Contains(updateDiff.SQL, "WHERE `id` = UNHEX(REPLACE(") {
		t.Fatalf("update SQL diff missing expected name update:\n%s", updateDiff.SQL)
	}

	taskID := db.MustNewUUIDv7()
	taskEventID := db.MustNewUUIDv7()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	taskReceipt, err := db.InsertWithReceiptContext(context.Background(), store,
		func() sq.InsertQuery {
			task := sq.New[db.TASK]("")
			return sq.InsertInto(task).ColumnValues(func(col *sq.Column) {
				col.SetBytes(task.ID, db.MustUUIDBytes(taskID))
				col.SetString(task.TASK_STREAM, "instances.image_archive.upload")
				col.SetString(task.SUBJECT_TYPE, "instance_image_archive")
				col.SetString(task.SUBJECT_ID, "local-vm")
				col.SetString(task.OWNER_PEER_ID, "peer-owner")
				col.SetString(task.STATUS, "running")
				col.SetString(task.TITLE, "Upload image archive")
				col.SetString(task.MESSAGE, "upload in progress")
				col.SetInt(task.PROGRESS, 42)
				col.SetJSON(task.PAYLOAD, map[string]string{"image_ref": "test:latest"})
				col.SetJSON(task.RESULT, map[string]string{})
				col.SetString(task.ERROR_MESSAGE, "")
				col.SetInt(task.ATTEMPTS, 1)
				col.SetInt(task.MAX_ATTEMPTS, 1)
				col.SetString(task.CREATED_AT, now)
				col.SetString(task.UPDATED_AT, now)
				col.SetString(task.STARTED_AT, now)
				col.SetString(task.FINISHED_AT, "")
			})
		},
		func() sq.InsertQuery {
			event := sq.New[db.TASK_EVENT]("")
			return sq.InsertInto(event).ColumnValues(func(col *sq.Column) {
				col.SetBytes(event.ID, db.MustUUIDBytes(taskEventID))
				col.SetBytes(event.TASK_ID, db.MustUUIDBytes(taskID))
				col.SetString(event.STATUS, "running")
				col.SetString(event.MESSAGE, "upload in progress")
				col.SetInt(event.PROGRESS, 42)
				col.SetJSON(event.DETAILS, map[string]any{"percent": 42})
				col.SetString(event.CREATED_AT, now)
			})
		},
	)
	if err != nil {
		t.Fatalf("insert task/event: %v", err)
	}
	waitForPublishedEventApplied(t, store, taskReceipt)

	taskCommit, err := store.GetLastCommit("main")
	if err != nil {
		t.Fatalf("get task commit: %v", err)
	}
	taskDiff, err := store.GetCommitDiff(context.Background(), taskCommit.Hash, "")
	if err != nil {
		t.Fatalf("get task diff: %v", err)
	}
	if len(taskDiff.RelatedTasks) != 1 {
		t.Fatalf("related task count = %d, want 1: %#v", len(taskDiff.RelatedTasks), taskDiff.RelatedTasks)
	}
	relatedTask := taskDiff.RelatedTasks[0]
	if relatedTask.ID != taskID ||
		relatedTask.OwnerPeerID != "peer-owner" ||
		relatedTask.Progress != 42 ||
		relatedTask.EventCount != 1 ||
		!slices.Contains(relatedTask.ChangeSources, "tasks") ||
		!slices.Contains(relatedTask.ChangeSources, "task_events") ||
		!strings.Contains(relatedTask.Summary, "Upload image archive") {
		t.Fatalf("related task context mismatch: %#v", relatedTask)
	}
	if !strings.Contains(taskDiff.CUE, `_related_tasks: [`) ||
		!strings.Contains(taskDiff.CUE, `owner_peer_id: "peer-owner"`) ||
		!strings.Contains(taskDiff.UnifiedDiff, "diff --cue a/tasks/") ||
		!strings.Contains(taskDiff.UnifiedDiff, "diff --cue a/task_events/") {
		t.Fatalf("task diff missing related context:\nCUE:\n%s\nUnified:\n%s", taskDiff.CUE, taskDiff.UnifiedDiff)
	}
	if !strings.Contains(taskDiff.SQL, "INSERT INTO `tasks`") ||
		!strings.Contains(taskDiff.SQL, "INSERT INTO `task_events`") ||
		!strings.Contains(taskDiff.SQL, "CAST('{\"percent\":42}' AS JSON)") {
		t.Fatalf("task SQL diff missing expected task inserts:\n%s", taskDiff.SQL)
	}
}

func TestSwarmionBackedDBProvisionerWriteThenUserRead(t *testing.T) {
	useSingleNodeDevelopmentScheduler(t)
	cfg := config.Get()
	previousP2PPort := cfg.P2PPort
	cfg.P2PPort = 0
	t.Cleanup(func() {
		cfg.P2PPort = previousP2PPort
	})

	workDir := t.TempDir()
	key, err := pcrypto.GetLocalKey(workDir)
	if err != nil {
		t.Fatalf("get local key: %v", err)
	}

	store, err := db.Open(workDir, "protos_test_provisioner_read", key, testswarmion.NewBorrowedLink(t, key))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer store.Close()

	if err := store.Init(); err != nil {
		t.Fatalf("init db: %v", err)
	}

	userID := db.MustNewUUIDv7()
	if _, err := db.InsertWithReceiptContext(context.Background(), store, func() sq.InsertQuery {
		u := sq.New[db.USER]("")
		return sq.InsertInto(u).ColumnValues(func(col *sq.Column) {
			col.SetBytes(u.ID, db.MustUUIDBytes(userID))
			col.SetString(u.USERNAME, "alex")
			col.SetString(u.NAME, "Alex")
			col.SetBool(u.IS_DISABLED, false)
		})
	}); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	providerID := db.MustNewUUIDv7()
	if _, err := db.InsertWithReceiptContext(context.Background(), store, func() sq.InsertQuery {
		cp := sq.New[db.CLOUD_PROVIDER]("")
		return sq.InsertInto(cp).ColumnValues(func(col *sq.Column) {
			col.SetBytes(cp.ID, db.MustUUIDBytes(providerID))
			col.SetString(cp.NAME, "local-test")
			col.SetString(cp.TYPE, "local_macos")
			col.SetJSON(cp.AUTH, map[string]string{"VM_DIR": filepath.Join(workDir, "vms")})
		})
	}); err != nil {
		t.Fatalf("insert provisioner: %v", err)
	}

	if _, err := store.GetCombinedCommits("main", "tentative"); err != nil {
		t.Fatalf("get combined commits after provisioner insert: %v", err)
	}

	var name string
	if err := store.ReadRows(context.Background(), "SELECT name FROM users WHERE username = 'alex'", nil, func(rows *sql.Rows) error {
		if !rows.Next() {
			return fmt.Errorf("query user returned no rows")
		}
		return rows.Scan(&name)
	}); err != nil {
		t.Fatalf("query user after provisioner insert: %v", err)
	}
	if name != "Alex" {
		t.Fatalf("user name = %q, want Alex", name)
	}
}
