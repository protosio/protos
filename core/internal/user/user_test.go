package user

import (
	"context"
	"testing"
	"time"

	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/testswarmion"
)

func TestUserWritesReturnSinglePeerAvailabilityConfirmation(t *testing.T) {
	store := newTestUserDB(t)
	keys := pcrypto.CreateManager(store)
	manager := CreateManager(store, keys)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	created, userConfirmation, err := manager.CreateUserWithConfirmationContext(ctx, "alex", "Alex", true)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	assertSinglePeerUserWriteConfirmation(t, userConfirmation)

	deviceKey, err := keys.GenerateKey()
	if err != nil {
		t.Fatalf("generate device key: %v", err)
	}
	deviceConfirmation, err := manager.AddDeviceWithConfirmationContext(ctx, created.ID, "laptop", deviceKey)
	if err != nil {
		t.Fatalf("add device: %v", err)
	}
	assertSinglePeerUserWriteConfirmation(t, deviceConfirmation)
	if err := ctx.Err(); err != nil {
		t.Fatalf("single-peer user writes did not return promptly: %v", err)
	}

	devices, err := manager.GetAllDevices(false)
	if err != nil {
		t.Fatalf("list devices: %v", err)
	}
	if len(devices) != 1 || devices[0].UserID != created.ID {
		t.Fatalf("devices = %+v, want one device for user %s", devices, created.ID)
	}
}

func assertSinglePeerUserWriteConfirmation(t *testing.T, confirmation db.PublishedWriteConfirmation) {
	t.Helper()
	if confirmation.Stage != db.PublishedWriteConfirmationLocalAccepted {
		t.Fatalf("confirmation stage = %q, want %q", confirmation.Stage, db.PublishedWriteConfirmationLocalAccepted)
	}
	if !confirmation.AvailabilityPending {
		t.Fatal("single-peer write should preserve pending other-peer availability")
	}
	if !confirmation.Receipt.HasExactEventIdentity() {
		t.Fatalf("confirmation did not preserve exact receipt: %+v", confirmation.Receipt)
	}
}

func newTestUserDB(t *testing.T) *db.DB {
	t.Helper()
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
	t.Cleanup(func() {
		_ = store.Close()
	})
	if err := store.Init(); err != nil {
		t.Fatalf("init db: %v", err)
	}
	return store
}
