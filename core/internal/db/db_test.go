package db_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

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

	userID := db.MustNewUUIDv7()
	if _, err := store.ExecSQLAndCommit(
		fmt.Sprintf("INSERT INTO users (id, username, name, is_disabled) VALUES (UNHEX(REPLACE('%s', '-', '')), 'alex', 'Alex', false)", userID),
		"insert test user",
	); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	var count int
	rows, err := store.QueryContext(context.Background(), "SELECT COUNT(*) FROM users WHERE username = 'alex'")
	if err != nil {
		t.Fatalf("query user count: %v", err)
	}
	if !rows.Next() {
		rows.Close()
		t.Fatal("query user count returned no rows")
	}
	if err := rows.Scan(&count); err != nil {
		rows.Close()
		t.Fatalf("query user count: %v", err)
	}
	if err := rows.Close(); err != nil {
		t.Fatalf("close user count rows: %v", err)
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

func countString(values []string, target string) int {
	count := 0
	for _, value := range values {
		if value == target {
			count++
		}
	}
	return count
}
