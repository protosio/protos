package db

import (
	"context"
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
	"github.com/protosio/protos/internal/testswarmion"
)

func TestBootstrapRetryUsesOnlyTypedNotReadyOutcome(t *testing.T) {
	retryable := fmt.Errorf("failed to open swarmion runtime: %w", &swarmionapp.BootstrapNotReadyError{
		Stage:             swarmionapp.BootstrapNotReadyStageInitialRootClosures,
		MissingRootCount:  1,
		MissingEventCount: 1,
		ProviderCount:     1,
		Cause:             context.DeadlineExceeded,
	})
	if !errors.Is(retryable, swarmionapp.ErrBootstrapNotReady) {
		t.Fatalf("typed bootstrap error=%v, want ErrBootstrapNotReady", retryable)
	}
	var details *swarmionapp.BootstrapNotReadyError
	if !errors.As(retryable, &details) || details == nil ||
		details.Stage != swarmionapp.BootstrapNotReadyStageInitialRootClosures ||
		details.MissingRootCount != 1 || details.MissingEventCount != 1 || details.ProviderCount != 1 {
		t.Fatalf("typed bootstrap diagnostics=%+v error=%v", details, retryable)
	}

	// Human diagnostics never grant retry authority, even when they contain the
	// exact text emitted by an older content-provider failure.
	textOnly := errors.New("failed to open swarmion runtime: pending_no_provider: no connected providers")
	if errors.Is(textOnly, swarmionapp.ErrBootstrapNotReady) {
		t.Fatalf("untyped bootstrap diagnostic unexpectedly matched retry sentinel: %v", textOnly)
	}

	tests := []error{
		nil,
		textOnly,
		errors.New("failed to open swarmion runtime: signature mismatch"),
		errors.New("failed to open swarmion runtime: manifest mismatch"),
		errors.New("failed to create swarmion transport: listen tcp: bind: address already in use"),
	}
	for _, err := range tests {
		if errors.Is(err, swarmionapp.ErrBootstrapNotReady) {
			t.Fatalf("permanent/untyped bootstrap error=%v unexpectedly matched ErrBootstrapNotReady", err)
		}
	}
}

func TestInterruptedFreshBootstrapAutomaticallyResumesFromRoutedPeer(t *testing.T) {
	t.Setenv("SWARMION_CHECKPOINT_SCHEDULER_PROFILE", "single_node_development")
	const databaseName = "protos_interrupted_bootstrap_retry"

	providerWorkDir := filepath.Join(t.TempDir(), "provider")
	providerSigner := newBootstrapTestSigner(t)
	providerTransport := testswarmion.New(t, providerSigner)
	provider, err := Open(providerWorkDir, databaseName, providerSigner, providerTransport.Link)
	if err != nil {
		t.Fatalf("open provider database: %v", err)
	}
	t.Cleanup(func() { _ = provider.Close() })
	if err := provider.Init(); err != nil {
		t.Fatalf("initialize provider database: %v", err)
	}

	markerID := MustNewUUIDv7()
	receipt, err := InsertWithReceiptContext(
		context.Background(),
		provider,
		transactionTestUserInsert(markerID, "interrupted-bootstrap-provider-marker"),
	)
	if err != nil {
		t.Fatalf("publish provider marker row: %v", err)
	}
	durableCtx, durableCancel := context.WithTimeout(context.Background(), 15*time.Second)
	if _, err := provider.WaitForPublishedWriteApplied(durableCtx, receipt, "prepare interrupted-bootstrap provider"); err != nil {
		durableCancel()
		t.Fatalf("checkpoint provider marker row: %v", err)
	}
	durableCancel()

	joinerWorkDir := filepath.Join(t.TempDir(), "joiner")
	joinerSigner := newBootstrapTestSigner(t)
	interruptedTransport := testswarmion.New(t, joinerSigner)
	interrupted, err := Open(joinerWorkDir, databaseName, joinerSigner, interruptedTransport.Link)
	if err != nil {
		t.Fatalf("open fresh joiner database: %v", err)
	}

	// Starting a real fresh join against an unavailable p2p-only candidate
	// durably creates Swarmion's bootstrap-in-progress marker, then the caller
	// deadline interrupts it before a foundation or writable SQL view exists.
	markerPath := filepath.Join(joinerWorkDir, ".swarmion", "bootstrap-in-progress.json")
	interruptFreshBootstrapAfterMarker(t, interrupted, providerSigner.GetID(), markerPath)
	if interrupted.Initialized() {
		_ = interrupted.Close()
		t.Fatal("interrupted fresh bootstrap exposed an initialized SQL view")
	}
	if _, err := os.Stat(markerPath); err != nil {
		_ = interrupted.Close()
		t.Fatalf("interrupted fresh bootstrap marker: %v", err)
	}
	sentinelPath := filepath.Join(joinerWorkDir, databaseName, "repository-preservation-sentinel")
	if err := os.WriteFile(sentinelPath, []byte("preserve interrupted repository\n"), 0o600); err != nil {
		_ = interrupted.Close()
		t.Fatalf("write repository preservation sentinel: %v", err)
	}
	if err := interrupted.Close(); err != nil {
		t.Fatalf("close interrupted bootstrap database: %v", err)
	}
	if err := interruptedTransport.Host.Close(); err != nil {
		t.Fatalf("close pre-restart caller-owned host: %v", err)
	}

	// A process restart creates a new caller-owned host with the same identity.
	// DB.Open first waits with no provider. The application then re-establishes
	// the physical route, and the borrowed Link's ordered route event wakes the
	// automatic retry without an explicit database-init RPC.
	restartedTransport := testswarmion.New(t, joinerSigner)
	reopened, err := Open(joinerWorkDir, databaseName, joinerSigner, restartedTransport.Link)
	if err != nil {
		t.Fatalf("reopen interrupted bootstrap database: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	if reopened.Initialized() {
		t.Fatal("zero-provider interrupted bootstrap unexpectedly initialized before reconnect")
	}
	pendingReadiness := reopened.RepositoryReadiness()
	if pendingReadiness.Initialized || !pendingReadiness.ExistingRepository || !pendingReadiness.BootstrapPending || pendingReadiness.BootstrapError != nil {
		t.Fatalf("interrupted repository readiness before reconnect = %+v, want existing pending recovery", pendingReadiness)
	}
	testswarmion.Connect(t, restartedTransport, providerTransport)

	recoveryCtx, recoveryCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer recoveryCancel()
	for {
		if recoveryErr := reopened.AutomaticBootstrapError(); recoveryErr != nil {
			t.Fatalf("automatic interrupted-bootstrap recovery failed: %v", recoveryErr)
		}
		if reopened.Initialized() {
			break
		}
		select {
		case <-recoveryCtx.Done():
			t.Fatalf("automatic interrupted-bootstrap recovery remained pending: %v", recoveryCtx.Err())
		case <-time.After(20 * time.Millisecond):
		}
	}
	recoveredReadiness := reopened.RepositoryReadiness()
	if !recoveredReadiness.Initialized || !recoveredReadiness.ExistingRepository || recoveredReadiness.BootstrapPending || recoveredReadiness.BootstrapError != nil {
		t.Fatalf("interrupted repository readiness after recovery = %+v, want initialized existing repository", recoveredReadiness)
	}

	// OpenHost's successful return is the SQL-readiness boundary: no polling of
	// Swarmion state or explicit InitFromPeer RPC is needed before this read.
	if got := transactionTestUserName(t, reopened, markerID); got != "interrupted-bootstrap-provider-marker" {
		t.Fatalf("recovered provider marker name=%q", got)
	}
	if _, err := os.Stat(markerPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("completed automatic bootstrap marker stat=%v, want removed", err)
	}
	if contents, err := os.ReadFile(sentinelPath); err != nil || string(contents) != "preserve interrupted repository\n" {
		t.Fatalf("interrupted repository was not preserved: contents=%q err=%v", contents, err)
	}
}

func TestInitFromPeerMigrationFailureClosesOnlySwarmionScope(t *testing.T) {
	t.Setenv("SWARMION_CHECKPOINT_SCHEDULER_PROFILE", "single_node_development")
	const databaseName = "protos_peer_init_migration_failure"

	providerWorkDir := filepath.Join(t.TempDir(), "provider")
	providerSigner := newBootstrapTestSigner(t)
	providerTransport := testswarmion.New(t, providerSigner)
	provider, err := Open(providerWorkDir, databaseName, providerSigner, providerTransport.Link)
	if err != nil {
		t.Fatalf("open provider database: %v", err)
	}
	t.Cleanup(func() { _ = provider.Close() })
	if err := provider.Init(); err != nil {
		t.Fatalf("initialize provider database: %v", err)
	}

	joinerSigner := newBootstrapTestSigner(t)
	joinerTransport := testswarmion.New(t, joinerSigner)
	joiner, err := Open(filepath.Join(t.TempDir(), "joiner"), databaseName, joinerSigner, joinerTransport.Link)
	if err != nil {
		t.Fatalf("open joiner database: %v", err)
	}
	t.Cleanup(func() { _ = joiner.Close() })
	testswarmion.Connect(t, joinerTransport, providerTransport)

	injectedErr := errors.New("injected post-open migration failure")
	var migrationCalls atomic.Int32
	joiner.mu.Lock()
	joiner.runMigrationsForTest = func(context.Context) error {
		migrationCalls.Add(1)
		return injectedErr
	}
	joiner.mu.Unlock()

	initCtx, cancelInit := context.WithTimeout(context.Background(), 20*time.Second)
	err = joiner.InitFromPeerContext(initCtx, providerSigner.GetID(), []string{"/p2p/" + providerSigner.GetID()})
	cancelInit()
	if !errors.Is(err, injectedErr) {
		t.Fatalf("peer initialization error=%v, want injected migration failure", err)
	}
	if got := migrationCalls.Load(); got != 1 {
		t.Fatalf("migration finalizer calls=%d, want 1", got)
	}
	assertMigrationFailureClosedSwarmionScope(t, joiner, joinerTransport, providerTransport)

	// Reusing the same DB and borrowed Link proves failure cleanup unregistered
	// the old Swarmion scope without closing the caller-owned physical host.
	joiner.mu.Lock()
	joiner.runMigrationsForTest = nil
	joiner.mu.Unlock()
	retryCtx, cancelRetry := context.WithTimeout(context.Background(), 20*time.Second)
	err = joiner.InitFromPeerContext(retryCtx, providerSigner.GetID(), []string{"/p2p/" + providerSigner.GetID()})
	cancelRetry()
	if err != nil {
		t.Fatalf("retry peer initialization on same borrowed host: %v", err)
	}
	if !joiner.Initialized() {
		t.Fatal("successful peer initialization did not pass the migration readiness boundary")
	}
	var migrationRows int
	if err := joiner.GetSqlDB().QueryRowContext(context.Background(), `SELECT COUNT(*) FROM sqddl_history`).Scan(&migrationRows); err != nil {
		t.Fatalf("query migration history after peer initialization: %v", err)
	}
	if migrationRows == 0 {
		t.Fatal("peer initialization returned without embedded migration history")
	}

	var repeatedMigrationCalls atomic.Int32
	joiner.mu.Lock()
	joiner.runMigrationsForTest = func(context.Context) error {
		repeatedMigrationCalls.Add(1)
		return errors.New("already-ready database unexpectedly reran migrations")
	}
	joiner.mu.Unlock()
	repeatCtx, cancelRepeat := context.WithTimeout(context.Background(), time.Second)
	err = joiner.InitFromPeerContext(repeatCtx, providerSigner.GetID(), []string{"/p2p/" + providerSigner.GetID()})
	cancelRepeat()
	if err != nil {
		t.Fatalf("repeat initialization of ready database: %v", err)
	}
	if got := repeatedMigrationCalls.Load(); got != 0 {
		t.Fatalf("repeat initialization reran migration finalizer %d time(s)", got)
	}
}

func TestAutomaticBootstrapMigrationFailureClosesOnlySwarmionScope(t *testing.T) {
	t.Setenv("SWARMION_CHECKPOINT_SCHEDULER_PROFILE", "single_node_development")
	const databaseName = "protos_automatic_bootstrap_migration_failure"

	providerSigner := newBootstrapTestSigner(t)
	providerTransport := testswarmion.New(t, providerSigner)
	provider, err := Open(filepath.Join(t.TempDir(), "provider"), databaseName, providerSigner, providerTransport.Link)
	if err != nil {
		t.Fatalf("open provider database: %v", err)
	}
	t.Cleanup(func() { _ = provider.Close() })
	if err := provider.Init(); err != nil {
		t.Fatalf("initialize provider database: %v", err)
	}

	joinerWorkDir := filepath.Join(t.TempDir(), "joiner")
	joinerSigner := newBootstrapTestSigner(t)
	interruptedTransport := testswarmion.New(t, joinerSigner)
	interrupted, err := Open(joinerWorkDir, databaseName, joinerSigner, interruptedTransport.Link)
	if err != nil {
		t.Fatalf("open fresh joiner database: %v", err)
	}
	markerPath := filepath.Join(joinerWorkDir, ".swarmion", "bootstrap-in-progress.json")
	interruptFreshBootstrapAfterMarker(t, interrupted, providerSigner.GetID(), markerPath)
	if interrupted.Initialized() {
		_ = interrupted.Close()
		t.Fatal("interrupted bootstrap exposed an initialized database")
	}
	if err := interrupted.Close(); err != nil {
		t.Fatalf("close interrupted bootstrap database: %v", err)
	}
	if err := interruptedTransport.Host.Close(); err != nil {
		t.Fatalf("close pre-restart caller-owned host: %v", err)
	}

	restartedTransport := testswarmion.New(t, joinerSigner)
	reopened, err := Open(joinerWorkDir, databaseName, joinerSigner, restartedTransport.Link)
	if err != nil {
		t.Fatalf("reopen interrupted bootstrap database: %v", err)
	}
	activeStore := reopened
	t.Cleanup(func() {
		if activeStore != nil {
			_ = activeStore.Close()
		}
	})
	if reopened.Initialized() {
		t.Fatal("zero-provider interrupted bootstrap unexpectedly initialized")
	}
	pendingReadiness := reopened.RepositoryReadiness()
	if pendingReadiness.Initialized || !pendingReadiness.ExistingRepository || !pendingReadiness.BootstrapPending || pendingReadiness.BootstrapError != nil {
		t.Fatalf("automatic migration retry readiness before reconnect = %+v, want existing pending recovery", pendingReadiness)
	}
	notifier := &countingNotifier{}
	reopened.RegisterTableChangeCallback("", notifier)

	injectedErr := errors.New("injected automatic post-open migration failure")
	var migrationCalls atomic.Int32
	reopened.mu.Lock()
	reopened.runMigrationsForTest = func(context.Context) error {
		migrationCalls.Add(1)
		return injectedErr
	}
	reopened.mu.Unlock()
	testswarmion.Connect(t, restartedTransport, providerTransport)

	recoveryCtx, cancelRecovery := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelRecovery()
	for {
		recoveryErr := reopened.AutomaticBootstrapError()
		if recoveryErr != nil {
			if !errors.Is(recoveryErr, injectedErr) {
				t.Fatalf("automatic bootstrap error=%v, want injected migration failure", recoveryErr)
			}
			break
		}
		if reopened.Initialized() {
			t.Fatal("automatic recovery published initialized before migrations completed")
		}
		select {
		case <-recoveryCtx.Done():
			t.Fatalf("automatic bootstrap migration failure was not reported: %v", recoveryCtx.Err())
		case <-time.After(20 * time.Millisecond):
		}
	}
	if got := migrationCalls.Load(); got != 1 {
		t.Fatalf("automatic migration finalizer calls=%d, want 1", got)
	}
	if got := notifier.Count(); got != 0 {
		t.Fatalf("automatic migration failure published %d table-change callback(s) before readiness", got)
	}
	failedReadiness := reopened.RepositoryReadiness()
	if failedReadiness.Initialized || !failedReadiness.ExistingRepository || failedReadiness.BootstrapPending || !errors.Is(failedReadiness.BootstrapError, injectedErr) {
		t.Fatalf("automatic migration failure readiness = %+v, want distinct permanent recovery error %v", failedReadiness, injectedErr)
	}
	assertMigrationFailureClosedSwarmionScope(t, reopened, restartedTransport, providerTransport)

	// A fresh DB wrapper on the same caller-owned Link can immediately reopen
	// the preserved repository. This also detects leaked Swarmion protocol
	// registrations from the failed automatic finalizer.
	if err := reopened.Close(); err != nil {
		t.Fatalf("close failed automatic bootstrap wrapper: %v", err)
	}
	activeStore = nil
	recovered, err := Open(joinerWorkDir, databaseName, joinerSigner, restartedTransport.Link)
	if err != nil {
		t.Fatalf("reopen repository after automatic migration failure: %v", err)
	}
	activeStore = recovered
	if !recovered.Initialized() {
		t.Fatal("repository did not initialize after automatic migration failure restart")
	}
}

func assertMigrationFailureClosedSwarmionScope(
	t *testing.T,
	store *DB,
	localTransport *testswarmion.Fixture,
	providerTransport *testswarmion.Fixture,
) {
	t.Helper()
	if store.Initialized() {
		t.Fatal("migration failure left database initialized")
	}
	if sqldb := store.GetSqlDB(); sqldb != nil {
		t.Fatal("migration failure left SQL database reachable")
	}
	if _, ok := store.SwarmionStatus(); ok {
		t.Fatal("migration failure left Swarmion runtime reachable")
	}
	if got := localTransport.Host.Network().Connectedness(providerTransport.Host.ID()); got != libp2pnetwork.Connected {
		t.Fatalf("migration cleanup changed borrowed physical route to %s, want connected", got)
	}
}

func interruptFreshBootstrapAfterMarker(
	t *testing.T,
	store *DB,
	providerPeerID string,
	markerPath string,
) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		result <- store.InitFromPeerContext(
			ctx,
			providerPeerID,
			[]string{"/p2p/" + providerPeerID},
		)
	}()

	deadline := time.NewTimer(15 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := os.Stat(markerPath); err == nil {
			cancel()
			break
		} else if !errors.Is(err, os.ErrNotExist) {
			cancel()
			t.Fatalf("inspect interrupted bootstrap marker: %v", err)
		}
		select {
		case err := <-result:
			cancel()
			t.Fatalf("fresh bootstrap returned before durable marker: %v", err)
		case <-deadline.C:
			cancel()
			select {
			case err := <-result:
				t.Fatalf("fresh bootstrap marker was not created: %v", err)
			case <-time.After(5 * time.Second):
				t.Fatal("fresh bootstrap did not stop after marker wait timeout")
			}
		case <-ticker.C:
		}
	}

	select {
	case err := <-result:
		if err == nil {
			t.Fatal("unrouted fresh bootstrap unexpectedly completed")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("fresh bootstrap did not stop after marker-boundary cancellation")
	}
}

func TestExistingRepositoryPermanentBootstrapErrorFailsClosed(t *testing.T) {
	t.Setenv("SWARMION_CHECKPOINT_SCHEDULER_PROFILE", "single_node_development")
	workDir := t.TempDir()
	signer := newBootstrapTestSigner(t)
	transport := testswarmion.New(t, signer)
	const databaseName = "protos_permanent_bootstrap_error"
	store, err := Open(workDir, databaseName, signer, transport.Link)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := store.Init(); err != nil {
		_ = store.Close()
		t.Fatalf("initialize database: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close database: %v", err)
	}

	markerPath := filepath.Join(workDir, ".swarmion", "bootstrap-in-progress.json")
	if err := os.MkdirAll(filepath.Dir(markerPath), 0o700); err != nil {
		t.Fatalf("create marker directory: %v", err)
	}
	if err := os.WriteFile(markerPath, []byte("not-json\n"), 0o600); err != nil {
		t.Fatalf("write corrupt marker: %v", err)
	}
	reopened, err := Open(workDir, databaseName, signer, transport.Link)
	if err == nil || reopened != nil {
		if reopened != nil {
			_ = reopened.Close()
		}
		t.Fatalf("permanent bootstrap error reopened database=%v err=%v", reopened, err)
	}
	if errors.Is(err, swarmionapp.ErrBootstrapNotReady) {
		t.Fatalf("permanent marker error was misclassified as retryable: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(workDir, databaseName, ".dolt", "repo_state.json")); statErr != nil {
		t.Fatalf("permanent bootstrap error removed repository: %v", statErr)
	}
	if contents, readErr := os.ReadFile(markerPath); readErr != nil || string(contents) != "not-json\n" {
		t.Fatalf("permanent bootstrap error mutated marker: contents=%q err=%v", contents, readErr)
	}
}

func TestAutomaticBootstrapRetryCloseCancelsAndDrainsRouteWatcher(t *testing.T) {
	t.Setenv("SWARMION_CHECKPOINT_SCHEDULER_PROFILE", "single_node_development")
	workDir := t.TempDir()
	signer := newBootstrapTestSigner(t)
	transport := testswarmion.New(t, signer)
	const databaseName = "protos_bootstrap_retry_close"
	store, err := Open(workDir, databaseName, signer, transport.Link)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := store.Init(); err != nil {
		_ = store.Close()
		t.Fatalf("initialize database: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close initialized database: %v", err)
	}

	markerPath := filepath.Join(workDir, ".swarmion", "bootstrap-in-progress.json")
	if err := os.MkdirAll(filepath.Dir(markerPath), 0o700); err != nil {
		t.Fatalf("create marker directory: %v", err)
	}
	if err := os.WriteFile(markerPath, []byte("{\"version\":1}\n"), 0o600); err != nil {
		t.Fatalf("write bootstrap marker: %v", err)
	}
	pending, err := Open(workDir, databaseName, signer, transport.Link)
	if err != nil {
		t.Fatalf("open pending interrupted bootstrap: %v", err)
	}
	if pending.Initialized() {
		_ = pending.Close()
		t.Fatal("zero-provider interrupted bootstrap unexpectedly initialized")
	}
	pendingReadiness := pending.RepositoryReadiness()
	if pendingReadiness.Initialized || !pendingReadiness.ExistingRepository || !pendingReadiness.BootstrapPending || pendingReadiness.BootstrapError != nil {
		_ = pending.Close()
		t.Fatalf("pending interrupted-bootstrap readiness = %+v, want existing pending recovery", pendingReadiness)
	}

	closed := make(chan error, 1)
	go func() { closed <- pending.Close() }()
	select {
	case err := <-closed:
		if err != nil {
			t.Fatalf("close pending interrupted bootstrap: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not cancel and drain the interrupted-bootstrap route watcher")
	}
	if err := pending.AutomaticBootstrapError(); err != nil {
		t.Fatalf("Close recorded cancellation as a permanent bootstrap error: %v", err)
	}
	closedReadiness := pending.RepositoryReadiness()
	if closedReadiness.Initialized || !closedReadiness.ExistingRepository || closedReadiness.BootstrapPending || closedReadiness.BootstrapError != nil {
		t.Fatalf("closed interrupted-bootstrap readiness = %+v, want preserved existing repository without an active worker", closedReadiness)
	}
	if _, err := os.Stat(markerPath); err != nil {
		t.Fatalf("canceling automatic bootstrap retry removed recovery marker: %v", err)
	}
}

func newBootstrapTestSigner(t testing.TB) testSwarmionRawSigner {
	t.Helper()
	privateKey, publicKey, err := libp2pcrypto.GenerateEd25519Key(cryptorand.Reader)
	if err != nil {
		t.Fatalf("generate bootstrap test signer: %v", err)
	}
	return testSwarmionRawSigner{privateKey: privateKey, publicKey: publicKey}
}
