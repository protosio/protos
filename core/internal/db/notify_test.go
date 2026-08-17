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

func canceledSubscriptionContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
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

func TestCheckpointEventExactDiffTargetsOnlyChangedTable(t *testing.T) {
	store := newCallbackTestDB()
	apps := &countingNotifier{}
	users := &countingNotifier{}

	store.RegisterTableChangeCallback("apps", apps)
	store.RegisterTableChangeCallback("users", users)
	events := make(chan swarmionapp.CheckpointEvent, 2)
	events <- swarmionapp.CheckpointEvent{
		Snapshot:              swarmionapp.CheckpointSnapshot{RootHash: "root-2", CommitID: "commit-2"},
		PreviousRootHash:      "root-1",
		ChangedTables:         []string{"apps"},
		ChangedTablesComplete: true,
	}
	events <- swarmionapp.CheckpointEvent{
		Snapshot:           swarmionapp.CheckpointSnapshot{RootHash: "root-2", CommitID: "commit-2"},
		Terminated:         true,
		TerminalReasonCode: swarmionapp.LifecycleEventTerminalReasonContextCanceled,
	}
	close(events)

	if unexpected := store.forwardSwarmionCheckpointRootEvents(canceledSubscriptionContext(), events); unexpected {
		t.Fatal("normal checkpoint terminal was classified as unexpected")
	}

	waitForNotifierCount(t, apps, 1)
	if users.Count() != 0 {
		t.Fatalf("unchanged-table callback count = %d, want 0", users.Count())
	}
}

func TestCheckpointEventGapAndIncompleteDiffInvalidateAllTables(t *testing.T) {
	store := newCallbackTestDB()
	apps := &countingNotifier{}
	users := &countingNotifier{}

	store.RegisterTableChangeCallback("apps", apps)
	store.RegisterTableChangeCallback("users", users)
	events := make(chan swarmionapp.CheckpointEvent, 3)
	events <- swarmionapp.CheckpointEvent{
		Snapshot:    swarmionapp.CheckpointSnapshot{RootHash: "root-2", CommitID: "commit-2"},
		SequenceGap: true,
	}
	events <- swarmionapp.CheckpointEvent{
		Snapshot:         swarmionapp.CheckpointSnapshot{RootHash: "root-3", CommitID: "commit-3"},
		PreviousRootHash: "root-2",
	}
	events <- swarmionapp.CheckpointEvent{
		Snapshot:           swarmionapp.CheckpointSnapshot{RootHash: "root-3", CommitID: "commit-3"},
		Terminated:         true,
		TerminalReasonCode: swarmionapp.LifecycleEventTerminalReasonContextCanceled,
	}
	close(events)

	if unexpected := store.forwardSwarmionCheckpointRootEvents(canceledSubscriptionContext(), events); unexpected {
		t.Fatal("normal checkpoint terminal was classified as unexpected")
	}
	waitForNotifierCount(t, apps, 2)
	waitForNotifierCount(t, users, 2)
}

func TestSubscriptionInitialAndNormalTerminalDoNotNotify(t *testing.T) {
	store := newCallbackTestDB()
	table := &countingNotifier{}
	runtime := &countingNotifier{}
	store.RegisterTableChangeCallback("apps", table)
	store.RegisterRuntimeChangeCallback(runtime)

	checkpointEvents := make(chan swarmionapp.CheckpointEvent, 2)
	checkpointEvents <- swarmionapp.CheckpointEvent{Initial: true}
	checkpointEvents <- swarmionapp.CheckpointEvent{Terminated: true, TerminalReasonCode: swarmionapp.LifecycleEventTerminalReasonContextCanceled}
	close(checkpointEvents)
	if unexpected := store.forwardSwarmionCheckpointRootEvents(canceledSubscriptionContext(), checkpointEvents); unexpected {
		t.Fatal("normal checkpoint terminal was classified as unexpected")
	}

	statusEvents := make(chan swarmionapp.StatusEvent, 2)
	statusEvents <- swarmionapp.StatusEvent{Initial: true}
	statusEvents <- swarmionapp.StatusEvent{Terminated: true, TerminalReasonCode: swarmionapp.LifecycleEventTerminalReasonContextCanceled}
	close(statusEvents)
	if unexpected := store.forwardSwarmionStatusEvents(canceledSubscriptionContext(), statusEvents); unexpected {
		t.Fatal("normal status terminal was classified as unexpected")
	}
	if table.Count() != 0 || runtime.Count() != 0 {
		t.Fatalf("initial/terminal callbacks table=%d runtime=%d, want 0/0", table.Count(), runtime.Count())
	}
}

func TestStatusOrdinaryAndGapEventsNotifyRuntime(t *testing.T) {
	store := newCallbackTestDB()
	notifier := &countingNotifier{}
	store.RegisterRuntimeChangeCallback(notifier)
	events := make(chan swarmionapp.StatusEvent, 4)
	events <- swarmionapp.StatusEvent{Initial: true}
	events <- swarmionapp.StatusEvent{Changes: []swarmionapp.StatusChange{swarmionapp.StatusEventTentativeRootChanged}}
	events <- swarmionapp.StatusEvent{SequenceGap: true}
	events <- swarmionapp.StatusEvent{Terminated: true, TerminalReasonCode: swarmionapp.LifecycleEventTerminalReasonContextCanceled}
	close(events)
	if unexpected := store.forwardSwarmionStatusEvents(canceledSubscriptionContext(), events); unexpected {
		t.Fatal("normal status terminal was classified as unexpected")
	}
	waitForNotifierCount(t, notifier, 2)
}

type fakeStatusEventSubscription struct {
	events <-chan swarmionapp.StatusEvent
	mu     sync.Mutex
	closed int
}

func (s *fakeStatusEventSubscription) Events() <-chan swarmionapp.StatusEvent { return s.events }
func (s *fakeStatusEventSubscription) Close() {
	s.mu.Lock()
	s.closed++
	s.mu.Unlock()
}
func (s *fakeStatusEventSubscription) closeCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

type fakeCheckpointEventSubscription struct {
	events <-chan swarmionapp.CheckpointEvent
	mu     sync.Mutex
	closed int
}

func (s *fakeCheckpointEventSubscription) Events() <-chan swarmionapp.CheckpointEvent {
	return s.events
}
func (s *fakeCheckpointEventSubscription) Close() {
	s.mu.Lock()
	s.closed++
	s.mu.Unlock()
}
func (s *fakeCheckpointEventSubscription) closeCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

func TestLiveWatchClosedStatusTerminalResubscribesAndInvalidatesRecoveredInitial(t *testing.T) {
	store := newCallbackTestDB()
	notifier := &countingNotifier{}
	store.RegisterRuntimeChangeCallback(notifier)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	firstEvents := make(chan swarmionapp.StatusEvent, 1)
	firstEvents <- swarmionapp.StatusEvent{
		Terminated:         true,
		TerminalReasonCode: swarmionapp.LifecycleEventTerminalReasonWatchClosed,
	}
	close(firstEvents)
	secondEvents := make(chan swarmionapp.StatusEvent, 2)
	secondEvents <- swarmionapp.StatusEvent{Initial: true}
	secondEvents <- swarmionapp.StatusEvent{Terminated: true, TerminalReasonCode: swarmionapp.LifecycleEventTerminalReasonContextCanceled}
	close(secondEvents)
	first := &fakeStatusEventSubscription{events: firstEvents}
	second := &fakeStatusEventSubscription{events: secondEvents}
	calls := 0
	store.runSwarmionStatusSubscription(ctx, func(context.Context) (statusEventSubscription, error) {
		calls++
		if calls == 1 {
			return first, nil
		}
		cancel()
		return second, nil
	})
	if calls != 2 || first.closeCount() != 1 || second.closeCount() != 1 {
		t.Fatalf("subscribe calls=%d closes=%d/%d, want 2 and 1/1", calls, first.closeCount(), second.closeCount())
	}
	waitForNotifierCount(t, notifier, 2)
}

func TestLiveForwardedContextCanceledStatusTerminalResubscribes(t *testing.T) {
	store := newCallbackTestDB()
	notifier := &countingNotifier{}
	store.RegisterRuntimeChangeCallback(notifier)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	firstEvents := make(chan swarmionapp.StatusEvent, 1)
	firstEvents <- swarmionapp.StatusEvent{
		Terminated:         true,
		TerminalReasonCode: swarmionapp.LifecycleEventTerminalReasonContextCanceled,
	}
	close(firstEvents)
	secondEvents := make(chan swarmionapp.StatusEvent, 1)
	secondEvents <- swarmionapp.StatusEvent{
		Terminated:         true,
		TerminalReasonCode: swarmionapp.LifecycleEventTerminalReasonContextCanceled,
	}
	close(secondEvents)
	first := &fakeStatusEventSubscription{events: firstEvents}
	second := &fakeStatusEventSubscription{events: secondEvents}
	calls := 0
	store.runSwarmionStatusSubscription(ctx, func(context.Context) (statusEventSubscription, error) {
		calls++
		if calls == 1 {
			return first, nil
		}
		cancel()
		return second, nil
	})
	if calls != 2 || first.closeCount() != 1 || second.closeCount() != 1 {
		t.Fatalf("subscribe calls=%d closes=%d/%d, want 2 and 1/1", calls, first.closeCount(), second.closeCount())
	}
	waitForNotifierCount(t, notifier, 1)
}

func TestLiveForwardedBranchClosedCheckpointTerminalResubscribes(t *testing.T) {
	store := newCallbackTestDB()
	notifier := &countingNotifier{}
	store.RegisterTableChangeCallback("apps", notifier)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	firstEvents := make(chan swarmionapp.CheckpointEvent, 1)
	firstEvents <- swarmionapp.CheckpointEvent{
		Terminated:         true,
		TerminalReasonCode: swarmionapp.LifecycleEventTerminalReasonBranchRuntimeClosed,
	}
	close(firstEvents)
	secondEvents := make(chan swarmionapp.CheckpointEvent, 1)
	secondEvents <- swarmionapp.CheckpointEvent{
		Terminated:         true,
		TerminalReasonCode: swarmionapp.LifecycleEventTerminalReasonContextCanceled,
	}
	close(secondEvents)
	first := &fakeCheckpointEventSubscription{events: firstEvents}
	second := &fakeCheckpointEventSubscription{events: secondEvents}
	calls := 0
	store.runSwarmionCheckpointSubscription(ctx, func(context.Context) (checkpointEventSubscription, error) {
		calls++
		if calls == 1 {
			return first, nil
		}
		cancel()
		return second, nil
	})
	if calls != 2 || first.closeCount() != 1 || second.closeCount() != 1 {
		t.Fatalf("subscribe calls=%d closes=%d/%d, want 2 and 1/1", calls, first.closeCount(), second.closeCount())
	}
	waitForNotifierCount(t, notifier, 1)
}

func TestLocallyCanceledSubscriptionTerminalDoesNotResubscribe(t *testing.T) {
	t.Run("status", func(t *testing.T) {
		store := newCallbackTestDB()
		notifier := &countingNotifier{}
		store.RegisterRuntimeChangeCallback(notifier)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		events := make(chan swarmionapp.StatusEvent, 1)
		events <- swarmionapp.StatusEvent{Terminated: true, TerminalReasonCode: swarmionapp.LifecycleEventTerminalReasonContextCanceled}
		close(events)
		subscription := &fakeStatusEventSubscription{events: events}
		calls := 0
		store.runSwarmionStatusSubscription(ctx, func(context.Context) (statusEventSubscription, error) {
			calls++
			return subscription, nil
		})
		if calls != 1 || subscription.closeCount() != 1 || notifier.Count() != 0 {
			t.Fatalf("status calls=%d closes=%d notifications=%d, want 1/1/0", calls, subscription.closeCount(), notifier.Count())
		}
	})

	t.Run("checkpoint", func(t *testing.T) {
		store := newCallbackTestDB()
		notifier := &countingNotifier{}
		store.RegisterTableChangeCallback("apps", notifier)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		events := make(chan swarmionapp.CheckpointEvent, 1)
		events <- swarmionapp.CheckpointEvent{Terminated: true, TerminalReasonCode: swarmionapp.LifecycleEventTerminalReasonContextCanceled}
		close(events)
		subscription := &fakeCheckpointEventSubscription{events: events}
		calls := 0
		store.runSwarmionCheckpointSubscription(ctx, func(context.Context) (checkpointEventSubscription, error) {
			calls++
			return subscription, nil
		})
		if calls != 1 || subscription.closeCount() != 1 || notifier.Count() != 0 {
			t.Fatalf("checkpoint calls=%d closes=%d notifications=%d, want 1/1/0", calls, subscription.closeCount(), notifier.Count())
		}
	})
}

func TestSubscriptionConstructorErrorsUseLocalContextAuthority(t *testing.T) {
	t.Run("live context canceled source retries", func(t *testing.T) {
		store := newCallbackTestDB()
		notifier := &countingNotifier{}
		store.RegisterRuntimeChangeCallback(notifier)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		events := make(chan swarmionapp.StatusEvent, 1)
		events <- swarmionapp.StatusEvent{Terminated: true, TerminalReasonCode: swarmionapp.LifecycleEventTerminalReasonContextCanceled}
		close(events)
		subscription := &fakeStatusEventSubscription{events: events}
		calls := 0
		store.runSwarmionStatusSubscription(ctx, func(context.Context) (statusEventSubscription, error) {
			calls++
			if calls == 1 {
				return nil, context.Canceled
			}
			cancel()
			return subscription, nil
		})
		if calls != 2 || subscription.closeCount() != 1 {
			t.Fatalf("status constructor calls=%d closes=%d, want 2/1", calls, subscription.closeCount())
		}
		waitForNotifierCount(t, notifier, 1)
	})

	t.Run("live closed runtime invalidates then stops", func(t *testing.T) {
		store := newCallbackTestDB()
		notifier := &countingNotifier{}
		store.RegisterTableChangeCallback("apps", notifier)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		calls := 0
		store.runSwarmionCheckpointSubscription(ctx, func(context.Context) (checkpointEventSubscription, error) {
			calls++
			return nil, swarmionapp.ErrDatabaseRuntimeClosed
		})
		if calls != 1 {
			t.Fatalf("checkpoint constructor calls=%d, want 1", calls)
		}
		waitForNotifierCount(t, notifier, 1)
	})
}

func TestSubscriptionWorkersCloseAndJoinWhenBackgroundRegistrySeals(t *testing.T) {
	store := newCallbackTestDB()
	ctx, cancel := context.WithCancel(context.Background())
	statusEvents := make(chan swarmionapp.StatusEvent, 2)
	checkpointEvents := make(chan swarmionapp.CheckpointEvent, 2)
	status := &fakeStatusEventSubscription{events: statusEvents}
	checkpoint := &fakeCheckpointEventSubscription{events: checkpointEvents}

	statusDone, admitted := store.backgroundWork.begin()
	if !admitted {
		t.Fatal("status worker was not admitted")
	}
	go func() {
		defer statusDone()
		store.runSwarmionStatusSubscription(ctx, func(context.Context) (statusEventSubscription, error) { return status, nil })
	}()
	checkpointDone, admitted := store.backgroundWork.begin()
	if !admitted {
		t.Fatal("checkpoint worker was not admitted")
	}
	go func() {
		defer checkpointDone()
		store.runSwarmionCheckpointSubscription(ctx, func(context.Context) (checkpointEventSubscription, error) { return checkpoint, nil })
	}()

	statusEvents <- swarmionapp.StatusEvent{Initial: true}
	checkpointEvents <- swarmionapp.CheckpointEvent{Initial: true}
	store.backgroundWork.seal()
	cancel()
	statusEvents <- swarmionapp.StatusEvent{Terminated: true, TerminalReasonCode: swarmionapp.LifecycleEventTerminalReasonContextCanceled}
	checkpointEvents <- swarmionapp.CheckpointEvent{Terminated: true, TerminalReasonCode: swarmionapp.LifecycleEventTerminalReasonContextCanceled}
	close(statusEvents)
	close(checkpointEvents)
	store.backgroundWork.wait()
	if status.closeCount() != 1 || checkpoint.closeCount() != 1 {
		t.Fatalf("joined subscription closes=%d/%d, want 1/1", status.closeCount(), checkpoint.closeCount())
	}
}

func TestInvalidCheckpointEventInvalidatesAllAndRequestsResubscribe(t *testing.T) {
	store := newCallbackTestDB()
	apps := &countingNotifier{}
	users := &countingNotifier{}
	store.RegisterTableChangeCallback("apps", apps)
	store.RegisterTableChangeCallback("users", users)
	events := make(chan swarmionapp.CheckpointEvent, 1)
	// ChangedTables without a complete adjacent diff is structurally invalid;
	// its table hint is ignored and the whole projection is invalidated.
	events <- swarmionapp.CheckpointEvent{ChangedTables: []string{"apps"}}
	close(events)
	if unexpected := store.forwardSwarmionCheckpointRootEvents(context.Background(), events); !unexpected {
		t.Fatal("invalid checkpoint event did not request resubscription")
	}
	waitForNotifierCount(t, apps, 1)
	waitForNotifierCount(t, users, 1)
}
