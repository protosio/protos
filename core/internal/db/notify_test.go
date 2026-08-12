package db

import (
	"context"
	"sync"
	"testing"
	"time"

	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
	"github.com/protosio/protos/internal/util"
)

type countingNotifier struct {
	mu    sync.Mutex
	count int
}

func (n *countingNotifier) Notify() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.count++
}

func (n *countingNotifier) Count() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.count
}

func newCallbackTestDB() *DB {
	return &DB{
		tableChangeCallbacks: util.NewMap[string, tableChangeCallback](),
		runtimeCallbacks:     util.NewMap[string, Notifier](),
	}
}

func waitForNotifierCount(t *testing.T, notifier *countingNotifier, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if notifier.Count() == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("callback count = %d, want %d", notifier.Count(), want)
}

func TestRuntimeChangeCallbackReceivesRuntimeDispatch(t *testing.T) {
	store := newCallbackTestDB()
	notifier := &countingNotifier{}

	store.RegisterRuntimeChangeCallback(notifier)
	store.dispatchRuntimeChangeCallbacks(false)

	if notifier.Count() != 1 {
		t.Fatalf("runtime callback count = %d, want 1", notifier.Count())
	}
}

func TestRuntimeChangeCallbackDoesNotReceiveTableDispatch(t *testing.T) {
	store := newCallbackTestDB()
	notifier := &countingNotifier{}

	store.RegisterRuntimeChangeCallback(notifier)
	store.dispatchTableChangeCallbacks(false, "apps")

	if notifier.Count() != 0 {
		t.Fatalf("runtime callback count after table dispatch = %d, want 0", notifier.Count())
	}
}

func TestRuntimeDispatchDeduplicatesGlobalAndRuntimeCallback(t *testing.T) {
	store := newCallbackTestDB()
	notifier := &countingNotifier{}

	store.RegisterTableChangeCallback("", notifier)
	store.RegisterRuntimeChangeCallback(notifier)
	store.dispatchRuntimeChangeCallbacks(false)

	if notifier.Count() != 1 {
		t.Fatalf("callback count = %d, want 1", notifier.Count())
	}
}

func TestWriteNotificationDoesNotDispatchTableCallbackBeforePublish(t *testing.T) {
	store := newCallbackTestDB()
	notifier := &countingNotifier{}

	store.RegisterTableChangeCallback("apps", notifier)
	err := store.handleWriteNotification(context.Background(), swarmionapp.WriteNotification{
		Accepted:      true,
		ChangedTables: []string{"apps"},
	})
	if err != nil {
		t.Fatalf("handleWriteNotification() error = %v", err)
	}

	if notifier.Count() != 0 {
		t.Fatalf("callback count after admission notification = %d, want 0", notifier.Count())
	}
}

func TestCheckpointRootEventDispatchesChangedTableCallback(t *testing.T) {
	store := newCallbackTestDB()
	notifier := &countingNotifier{}

	store.RegisterTableChangeCallback("apps", notifier)
	events := make(chan swarmionapp.CheckpointRootEvent, 1)
	events <- swarmionapp.CheckpointRootEvent{ChangedTables: []string{"apps"}}
	close(events)

	store.forwardSwarmionCheckpointRootEvents(events)

	waitForNotifierCount(t, notifier, 1)
}

func TestCheckpointRootEventWithoutChangedTablesDispatchesAllTableCallbacks(t *testing.T) {
	store := newCallbackTestDB()
	notifier := &countingNotifier{}

	store.RegisterTableChangeCallback("apps", notifier)
	events := make(chan swarmionapp.CheckpointRootEvent, 1)
	events <- swarmionapp.CheckpointRootEvent{}
	close(events)

	store.forwardSwarmionCheckpointRootEvents(events)

	waitForNotifierCount(t, notifier, 1)
}
