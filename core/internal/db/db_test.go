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
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/pcrypto"
)

func TestSwarmionBackedDBInitAndWrite(t *testing.T) {
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

	store, err := db.Open(workDir, "protos_test", key)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer store.Close()

	if store.Initialized() {
		t.Fatal("new database should not be initialized before Init")
	}
	if err := store.Init(); err != nil {
		t.Fatalf("init db: %v", err)
	}
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

	userID := db.MustNewUUIDv7()
	if err := db.Insert(store, func() sq.InsertQuery {
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

	store, err := db.Open(workDir, "protos_test_rapid", key)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer store.Close()

	if err := store.Init(); err != nil {
		t.Fatalf("init db: %v", err)
	}

	for _, username := range []string{"first", "second"} {
		userID := db.MustNewUUIDv7()
		if err := db.Insert(store, func() sq.InsertQuery {
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

	store, err := db.Open(workDir, "protos_test_commit_diff", key)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer store.Close()

	if err := store.Init(); err != nil {
		t.Fatalf("init db: %v", err)
	}

	userID := db.MustNewUUIDv7()
	if err := db.Insert(store, func() sq.InsertQuery {
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

	if err := db.Update(store, func() sq.UpdateQuery {
		u := sq.New[db.USER]("")
		return sq.Update(u).SetFunc(func(col *sq.Column) {
			col.SetString(u.NAME, "Alexander")
		}).Where(db.UUIDEq(u.ID, userID))
	}); err != nil {
		t.Fatalf("update user: %v", err)
	}

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
	if err := db.Insert(store,
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
	); err != nil {
		t.Fatalf("insert task/event: %v", err)
	}

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

	store, err := db.Open(workDir, "protos_test_provisioner_read", key)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer store.Close()

	if err := store.Init(); err != nil {
		t.Fatalf("init db: %v", err)
	}

	userID := db.MustNewUUIDv7()
	if err := db.Insert(store, func() sq.InsertQuery {
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
	if err := db.Insert(store, func() sq.InsertQuery {
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

func TestDialableListenMultiaddrsIncludeExplicitIPs(t *testing.T) {
	cfg := config.Get()
	previousP2PPort := cfg.P2PPort
	cfg.P2PPort = 10500
	t.Cleanup(func() {
		cfg.P2PPort = previousP2PPort
	})

	workDir := t.TempDir()
	key, err := pcrypto.GetLocalKey(workDir)
	if err != nil {
		t.Fatalf("get local key: %v", err)
	}
	store, err := db.Open(workDir, "protos_test", key)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer store.Close()

	got := store.DialableListenMultiaddrs([]string{"200::1", "192.168.64.1", "200::1"})
	wants := []string{
		fmt.Sprintf("/ip6/200::1/tcp/10501/p2p/%s", key.GetID()),
		fmt.Sprintf("/ip4/192.168.64.1/tcp/10501/p2p/%s", key.GetID()),
	}
	for _, want := range wants {
		if !slices.Contains(got, want) {
			t.Fatalf("DialableListenMultiaddrs() = %v, want %s", got, want)
		}
	}
	if count := countString(got, wants[0]); count != 1 {
		t.Fatalf("explicit IPv6 addr appeared %d times, want 1 in %v", count, got)
	}
}

func TestRemovedSwarmionPeerCleanupIsIdempotent(t *testing.T) {
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
	store, err := db.Open(workDir, "protos_test", key)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer store.Close()
	if err := store.Init(); err != nil {
		t.Fatalf("init db: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for i := 0; i < 2; i++ {
		if err := store.ReconcileRemovedSwarmionPeers(ctx, map[string]struct{}{}); err != nil {
			t.Fatalf("reconcile removed peers attempt %d: %v", i+1, err)
		}
		if err := store.PrepareSwarmionShutdown(ctx); err != nil {
			t.Fatalf("prepare shutdown attempt %d: %v", i+1, err)
		}
	}
}

func countString(values []string, target string) int {
	count := 0
	for _, value := range values {
		if value == target {
			count++
		}
	}
	return count
}
