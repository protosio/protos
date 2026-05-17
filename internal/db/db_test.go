package db_test

import (
	"context"
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

	if _, err := store.ExecSQLAndCommit(
		"INSERT INTO users (username, name, is_disabled) VALUES ('alex', 'Alex', false)",
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
}
