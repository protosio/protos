package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bokwoon95/sq"
	swarmionprotocol "github.com/nustiueudinastea/swarmion/protocol"
	swarmion "github.com/nustiueudinastea/swarmion/runtime"
)

func TestNewPublishedWriteOperationPersistsCompleteRecoveryBeforeExecute(t *testing.T) {
	store := openPeerTestDB(t)
	operation, err := store.NewPublishedWriteOperation("io.protos.tests.pre-execute/v1", []byte{0, 1, 2})
	if err != nil {
		t.Fatalf("create published operation: %v", err)
	}
	operations, err := store.loadPublishedWriteOperations()
	if err != nil {
		t.Fatalf("load recovery journal: %v", err)
	}
	for _, persisted := range operations {
		if persisted.Equal(operation) {
			return
		}
	}
	t.Fatalf("complete operation recovery was not persisted before Execute: %s", operation.Identity)
}

func TestPublishedWriteOperationKeysAreRandomAndSchemaDigestIsStable(t *testing.T) {
	store := openPeerTestDB(t)
	first, err := store.NewPublishedWriteOperation("io.protos.tests.random-key/v1", []byte{0, 1, 0xff})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.NewPublishedWriteOperation("io.protos.tests.random-key/v1", []byte{0, 1, 0xff})
	if err != nil {
		t.Fatal(err)
	}
	if first.Key() == second.Key() {
		t.Fatalf("independent operations reused random key %q", first.Key())
	}
	if first.IntentDigest() != second.IntentDigest() {
		t.Fatalf("same immutable schema and binary parts produced different provenance digests: %s != %s", first.IntentDigest(), second.IntentDigest())
	}
	zeroParts, err := store.NewPublishedWriteOperation("io.protos.tests.random-key/v1")
	if err != nil {
		t.Fatal(err)
	}
	oneEmptyPart, err := store.NewPublishedWriteOperation("io.protos.tests.random-key/v1", []byte{})
	if err != nil {
		t.Fatal(err)
	}
	if zeroParts.IntentDigest() == oneEmptyPart.IntentDigest() {
		t.Fatal("zero intent parts and one empty binary part were conflated")
	}
}

func TestRetryAuthorityCannotCrossRuntimeCloseAndReplacement(t *testing.T) {
	store := openPeerTestDB(t)
	operation, err := store.NewPublishedWriteOperation("io.protos.tests.retry-runtime-owner/v1", []byte("same identity only"))
	if err != nil {
		t.Fatal(err)
	}
	oldRuntime := transactionTestRuntime(t, store)
	retryResult := oldRuntime.ResolveOperation(context.Background(), operation.Recovery)
	if retryResult.Disposition() != swarmion.OperationRetryPermitted {
		t.Fatalf("initial absence disposition=%s diagnostic=%v, want retry permitted", retryResult.Disposition(), retryResult.Diagnostic())
	}
	retryReason, ok := retryResult.RetryReason()
	if !ok {
		t.Fatal("initial absence has no retry reason")
	}

	var executeCalls atomic.Int32
	store.executeOperationForTest = func(ctx context.Context, runtime *swarmion.DatabaseRuntime, request swarmion.OperationRequest) swarmion.OperationResult {
		if operationIdentitiesEqual(request.Identity, operation.Identity) {
			executeCalls.Add(1)
			return retryResult
		}
		return runtime.Execute(ctx, request)
	}
	waitStarted := make(chan struct{})
	releaseWait := make(chan struct{})
	var signalWait sync.Once
	store.waitPublishedWriteRetryForTest = func(ctx context.Context, _ int, reason swarmion.OperationRetryReason) error {
		if reason != retryReason {
			return fmt.Errorf("retry wait reason=%q, want %q", reason, retryReason)
		}
		signalWait.Do(func() { close(waitStarted) })
		select {
		case <-releaseWait:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	var applyCalls atomic.Int32
	type writeOutcome struct {
		receipt PublishedWriteReceipt
		err     error
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	completed := make(chan writeOutcome, 1)
	go func() {
		receipt, executeErr := store.executePublishedWriteTransactionWithSafeRetryContext(
			ctx,
			operation,
			"runtime owner replacement",
			false,
			true,
			func(ctx context.Context, executor sqlContextExecer) error {
				applyCalls.Add(1)
				_, executeErr := executor.ExecContext(ctx, "CREATE TABLE retry_runtime_owner_guard (id INT PRIMARY KEY)")
				return executeErr
			},
		)
		completed <- writeOutcome{receipt: receipt, err: executeErr}
	}()

	select {
	case <-waitStarted:
	case <-ctx.Done():
		t.Fatalf("write did not reach retry wait: %v", ctx.Err())
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close runtime during retry wait: %v", err)
	}
	if retryResult.Disposition() != swarmion.OperationRecoveryRequired {
		t.Fatalf("closed-runtime retry disposition=%s, want recovery required", retryResult.Disposition())
	}
	if err := store.Init(); err != nil {
		t.Fatalf("replace runtime after close: %v", err)
	}
	newRuntime := transactionTestRuntime(t, store)
	if newRuntime == oldRuntime {
		t.Fatal("runtime replacement reused the prior runtime handle")
	}
	close(releaseWait)

	select {
	case outcome := <-completed:
		if outcome.receipt.HasExactEventIdentity() {
			t.Fatalf("revoked retry returned receipt %+v", outcome.receipt)
		}
		if !errors.Is(outcome.err, ErrOperationReceiptUnavailable) {
			t.Fatalf("revoked retry error=%v, want recovery-required receipt-unavailable classification", outcome.err)
		}
	case <-ctx.Done():
		t.Fatalf("write did not fail after runtime replacement: %v", ctx.Err())
	}
	if got := executeCalls.Load(); got != 1 {
		t.Fatalf("target Execute calls=%d, want only the original rejected attempt", got)
	}
	if got := applyCalls.Load(); got != 1 {
		t.Fatalf("SQL body preparations=%d, want no body preparation on revoked retry", got)
	}
}

func TestPublishedWriteAbsenceLeaseRejectsReplacedRuntimeGeneration(t *testing.T) {
	store := openPeerTestDB(t)
	operation, err := store.NewPublishedWriteOperation(
		"io.protos.tests.side-effect-runtime-owner/v1",
		[]byte("provider effect"),
	)
	if err != nil {
		t.Fatal(err)
	}
	oldGeneration, ok := store.SwarmionRuntimeGeneration()
	if !ok || oldGeneration == 0 {
		t.Fatal("initial runtime generation is unavailable")
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close initial runtime: %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("open replacement runtime: %v", err)
	}
	newGeneration, ok := store.SwarmionRuntimeGeneration()
	if !ok || newGeneration == oldGeneration {
		t.Fatalf("replacement generation=%d, old=%d", newGeneration, oldGeneration)
	}

	var actionCalls atomic.Int32
	receipt, accepted, err := store.WithPublishedWriteAbsenceLease(
		context.Background(),
		operation,
		oldGeneration,
		"stale provider effect",
		func() error {
			actionCalls.Add(1)
			return nil
		},
	)
	if !errors.Is(err, ErrOperationReceiptUnavailable) {
		t.Fatalf("stale generation error=%v, want receipt unavailable", err)
	}
	if accepted || receipt.HasExactEventIdentity() {
		t.Fatalf("stale generation returned accepted=%t receipt=%+v", accepted, receipt)
	}
	if got := actionCalls.Load(); got != 0 {
		t.Fatalf("stale generation ran side effect %d times", got)
	}
}

func TestGenerationBoundPublishedWriteRejectsReplacementBeforeBodyOrExecute(t *testing.T) {
	store := openPeerTestDB(t)
	operation, err := store.NewPublishedWriteOperation(
		"io.protos.tests.generation-bound-execute/v1",
		[]byte("phase P or D"),
	)
	if err != nil {
		t.Fatal(err)
	}
	oldGeneration, ok := store.SwarmionRuntimeGeneration()
	if !ok {
		t.Fatal("initial runtime generation is unavailable")
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if err := store.Init(); err != nil {
		t.Fatal(err)
	}

	var bodyCalls atomic.Int32
	var executeCalls atomic.Int32
	store.executeOperationForTest = func(ctx context.Context, runtime *swarmion.DatabaseRuntime, request swarmion.OperationRequest) swarmion.OperationResult {
		executeCalls.Add(1)
		return runtime.Execute(ctx, request)
	}
	receipt, err := store.executePublishedWriteOperationForRuntimeGenerationContext(
		context.Background(),
		operation,
		"generation-bound phase",
		oldGeneration,
		func(ctx context.Context, executor sqlContextExecer) error {
			bodyCalls.Add(1)
			_, applyErr := executor.ExecContext(ctx, "CREATE TABLE stale_generation_guard (id INT PRIMARY KEY)")
			return applyErr
		},
	)
	if !errors.Is(err, ErrOperationReceiptUnavailable) {
		t.Fatalf("generation-bound execution error=%v, want receipt unavailable", err)
	}
	if receipt.HasExactEventIdentity() {
		t.Fatalf("generation-bound failure returned receipt %+v", receipt)
	}
	if got := bodyCalls.Load(); got != 0 {
		t.Fatalf("generation-bound failure prepared SQL body %d times", got)
	}
	if got := executeCalls.Load(); got != 0 {
		t.Fatalf("generation-bound failure called Execute %d times", got)
	}
}

func TestPublishedWriteAbsenceLeaseJoinsCloseAcrossAction(t *testing.T) {
	store := openPeerTestDB(t)
	operation, err := store.NewPublishedWriteOperation(
		"io.protos.tests.side-effect-close-join/v1",
		[]byte("blocking provider effect"),
	)
	if err != nil {
		t.Fatal(err)
	}
	generation, ok := store.SwarmionRuntimeGeneration()
	if !ok {
		t.Fatal("runtime generation is unavailable")
	}
	actionStarted := make(chan struct{})
	releaseAction := make(chan struct{})
	leaseDone := make(chan error, 1)
	go func() {
		_, accepted, leaseErr := store.WithPublishedWriteAbsenceLease(
			context.Background(),
			operation,
			generation,
			"blocking provider effect",
			func() error {
				close(actionStarted)
				<-releaseAction
				return nil
			},
		)
		if accepted && leaseErr == nil {
			leaseErr = fmt.Errorf("unexpected accepted operation")
		}
		leaseDone <- leaseErr
	}()

	select {
	case <-actionStarted:
	case <-time.After(30 * time.Second):
		t.Fatal("side-effect lease did not reach its action")
	}
	closeDone := make(chan error, 1)
	go func() { closeDone <- store.Close() }()
	select {
	case closeErr := <-closeDone:
		t.Fatalf("Close returned before the leased side effect joined: %v", closeErr)
	case <-time.After(100 * time.Millisecond):
	}
	close(releaseAction)
	select {
	case leaseErr := <-leaseDone:
		if leaseErr != nil {
			t.Fatalf("side-effect lease: %v", leaseErr)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("side-effect lease did not return")
	}
	select {
	case closeErr := <-closeDone:
		if closeErr != nil {
			t.Fatalf("Close after leased side effect: %v", closeErr)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("Close did not join after the leased side effect returned")
	}
}

func TestOperationRecoveryJournalFirstCreationBuildsDurableDirectoryChain(t *testing.T) {
	store := openPeerTestDB(t)
	operation, err := store.NewPublishedWriteOperation("io.protos.tests.first-directory/v1")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	journal := &DB{workingDir: root, name: "first-directory"}
	if err := journal.persistPublishedWriteOperation(operation); err != nil {
		t.Fatalf("persist through new directory chain: %v", err)
	}
	wantDir := filepath.Join(root, swarmionStateDirName, operationRecoveryJournalDir, "first-directory")
	if info, err := os.Stat(wantDir); err != nil || !info.IsDir() {
		t.Fatalf("journal directory info=%v error=%v", info, err)
	}
}

func TestOperationRecoveryJournalNeverRegressesExpectedReceipt(t *testing.T) {
	store := openPeerTestDB(t)
	operation, err := store.NewPublishedWriteOperation("io.protos.tests.monotonic-recovery/v1", []byte("accepted"))
	if err != nil {
		t.Fatal(err)
	}
	runtime := transactionTestRuntime(t, store)
	userID := MustNewUUIDv7()
	result := runtime.Execute(context.Background(), swarmion.OperationRequest{
		Identity: operation.Identity,
		Statements: []swarmion.Statement{{
			Query: "INSERT INTO users (id, username, name, is_disabled) VALUES (?, ?, ?, ?)",
			Args:  []any{MustUUIDBytes(userID), "journal-monotonic", "journal-monotonic", false},
		}},
	})
	if result.Disposition() != swarmion.OperationAccepted {
		t.Fatalf("execute result=%s diagnostic=%v", result.Disposition(), result.Diagnostic())
	}
	enrichedRecovery, ok := result.Recovery()
	if !ok {
		t.Fatal("accepted result has no recovery")
	}
	if _, ok := enrichedRecovery.ExpectedReceipt(); !ok {
		t.Fatal("accepted result recovery has no expected receipt")
	}
	enriched := PublishedWriteOperation{Identity: operation.Identity, Recovery: enrichedRecovery}

	var wait sync.WaitGroup
	errorsCh := make(chan error, 35)
	for i := 0; i < 32; i++ {
		candidate := operation
		if i%2 == 0 {
			candidate = enriched
		}
		wait.Add(1)
		go func() {
			defer wait.Done()
			errorsCh <- store.persistPublishedWriteOperation(candidate)
		}()
	}
	wait.Add(1)
	go func() {
		defer wait.Done()
		resolved, err := store.LookupPublishedWriteOperation(context.Background(), operation)
		if err == nil && resolved.Disposition() != swarmion.OperationAccepted {
			err = &unexpectedOperationDispositionError{got: resolved.Disposition(), want: swarmion.OperationAccepted}
		}
		errorsCh <- err
	}()
	wait.Add(1)
	go func() {
		defer wait.Done()
		waitCtx, cancel := context.WithTimeout(context.Background(), committedWriteCheckpointTimeout)
		defer cancel()
		resolved, err := store.WaitPublishedWriteOperation(waitCtx, operation)
		if err == nil && resolved.Disposition() != swarmion.OperationAccepted {
			err = &unexpectedOperationDispositionError{got: resolved.Disposition(), want: swarmion.OperationAccepted}
		}
		errorsCh <- err
	}()
	wait.Add(1)
	go func() {
		defer wait.Done()
		resolved, _, _, err := store.executePublishedWriteAttemptContext(
			context.Background(), operation, "journal monotonic replay", false, true,
			0,
			nil,
			func(ctx context.Context, executor sqlContextExecer) error {
				_, err := executor.ExecContext(ctx,
					"INSERT INTO users (id, username, name, is_disabled) VALUES (?, ?, ?, ?)",
					MustUUIDBytes(userID), "journal-monotonic", "journal-monotonic", false,
				)
				return err
			},
		)
		if err == nil && resolved.Disposition() != swarmion.OperationAccepted {
			err = &unexpectedOperationDispositionError{got: resolved.Disposition(), want: swarmion.OperationAccepted}
		}
		errorsCh <- err
	}()
	wait.Wait()
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatalf("concurrent journal update: %v", err)
		}
	}

	operations, err := store.loadPublishedWriteOperations()
	if err != nil {
		t.Fatal(err)
	}
	for _, persisted := range operations {
		if !operationIdentitiesEqual(persisted.Identity, operation.Identity) {
			continue
		}
		if _, ok := persisted.Recovery.ExpectedReceipt(); !ok {
			t.Fatal("receipt-less concurrent update regressed enriched recovery")
		}
		return
	}
	t.Fatalf("persisted operation %s not found", operation.Identity)
}

type unexpectedOperationDispositionError struct {
	got  swarmion.OperationDisposition
	want swarmion.OperationDisposition
}

func (err *unexpectedOperationDispositionError) Error() string {
	return "operation disposition " + string(err.got) + ", want " + string(err.want)
}

func TestDurableCleanupCannotBeUndoneByLatePassiveResolution(t *testing.T) {
	store := openPeerTestDB(t)
	operation, err := store.NewPublishedWriteOperation("io.protos.tests.retired-recovery/v1", []byte("accepted"))
	if err != nil {
		t.Fatal(err)
	}
	runtime := transactionTestRuntime(t, store)
	result := runtime.Execute(context.Background(), swarmion.OperationRequest{
		Identity: operation.Identity,
		Statements: []swarmion.Statement{{
			Query: "INSERT INTO users (id, username, name, is_disabled) VALUES (?, ?, ?, ?)",
			Args:  []any{MustUUIDBytes(MustNewUUIDv7()), "retired-recovery", "retired-recovery", false},
		}},
	})
	recovery, ok := result.Recovery()
	if result.Disposition() != swarmion.OperationAccepted || !ok {
		t.Fatalf("execute disposition=%s recovery=%t diagnostic=%v", result.Disposition(), ok, result.Diagnostic())
	}
	enriched := PublishedWriteOperation{Identity: operation.Identity, Recovery: recovery}
	if err := store.persistPublishedWriteOperation(enriched); err != nil {
		t.Fatal(err)
	}
	if err := store.removePublishedWriteOperation(enriched); err != nil {
		t.Fatal(err)
	}
	resolved, err := store.LookupPublishedWriteOperation(context.Background(), operation)
	if err != nil || resolved.Disposition() != swarmion.OperationAccepted {
		t.Fatalf("late resolve disposition=%s error=%v", resolved.Disposition(), err)
	}
	operations, err := store.loadPublishedWriteOperations()
	if err != nil {
		t.Fatal(err)
	}
	for _, persisted := range operations {
		if operationIdentitiesEqual(persisted.Identity, operation.Identity) {
			t.Fatalf("late resolution recreated retired journal record %s", operation.Identity)
		}
	}
}

func TestRestartRecoveryRetiresAuthoritativeAbsenceWithoutExecuting(t *testing.T) {
	store := openPeerTestDB(t)
	operation, err := store.NewPublishedWriteOperation("io.protos.tests.restart-absence/v1", []byte("never-executed"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.recoverPublishedWriteOperations(context.Background()); err != nil {
		t.Fatalf("recover absent operation: %v", err)
	}
	operations, err := store.loadPublishedWriteOperations()
	if err != nil {
		t.Fatal(err)
	}
	for _, persisted := range operations {
		if operationIdentitiesEqual(persisted.Identity, operation.Identity) {
			t.Fatalf("authoritatively absent operation remains in journal: %s", operation.Identity)
		}
	}
}

func TestRestartRecoveryRetainsAcceptedOperationUntilAppliedDurably(t *testing.T) {
	store := openPeerTestDB(t)
	operation, err := store.NewPublishedWriteOperation("io.protos.tests.restart-accepted-pending/v1", []byte("pending"))
	if err != nil {
		t.Fatal(err)
	}
	result := transactionTestRuntime(t, store).Execute(context.Background(), swarmion.OperationRequest{
		Identity: operation.Identity,
		Statements: []swarmion.Statement{{
			Query: "INSERT INTO users (id, username, name, is_disabled) VALUES (?, ?, ?, ?)",
			Args:  []any{MustUUIDBytes(MustNewUUIDv7()), "restart-accepted-pending", "restart-accepted-pending", false},
		}},
	})
	if result.Disposition() != swarmion.OperationAccepted {
		t.Fatalf("execute disposition=%s diagnostic=%v", result.Disposition(), result.Diagnostic())
	}
	waitStarted := make(chan struct{})
	releaseWait := make(chan struct{})
	waitCompleted := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseWait) }) }
	t.Cleanup(release)
	store.waitPublishedWriteAppliedForTest = func(_ context.Context, receipt PublishedWriteReceipt, reason string) (EventReceiptObservation, error) {
		close(waitStarted)
		<-releaseWait
		defer close(waitCompleted)
		observation := EventReceiptObservation{Receipt: receipt, State: EventReceiptStatePending}
		return observation, &EventReceiptPendingError{Observation: observation, Reason: reason, Cause: context.DeadlineExceeded}
	}
	recovered := make(chan error, 1)
	go func() {
		recovered <- store.recoverPublishedWriteOperations(context.Background())
	}()
	select {
	case err := <-recovered:
		if err != nil {
			t.Fatalf("recover accepted pending operation: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("accepted pending recovery blocked database readiness")
	}
	select {
	case <-waitStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("accepted pending recovery did not admit lifecycle-owned cleanup")
	}
	release()
	select {
	case <-waitCompleted:
	case <-time.After(5 * time.Second):
		t.Fatal("accepted pending cleanup did not finish")
	}
	operations, err := store.loadPublishedWriteOperations()
	if err != nil {
		t.Fatal(err)
	}
	for _, persisted := range operations {
		if operationIdentitiesEqual(persisted.Identity, operation.Identity) {
			return
		}
	}
	t.Fatalf("accepted but non-durable operation %s was removed", operation.Identity)
}

func TestRestartRecoveryRetiresAcceptedOperationAfterAppliedDurably(t *testing.T) {
	store := openPeerTestDB(t)
	operation, err := store.NewPublishedWriteOperation("io.protos.tests.restart-accepted-durable/v1", []byte("durable"))
	if err != nil {
		t.Fatal(err)
	}
	result := transactionTestRuntime(t, store).Execute(context.Background(), swarmion.OperationRequest{
		Identity: operation.Identity,
		Statements: []swarmion.Statement{{
			Query: "INSERT INTO users (id, username, name, is_disabled) VALUES (?, ?, ?, ?)",
			Args:  []any{MustUUIDBytes(MustNewUUIDv7()), "restart-accepted-durable", "restart-accepted-durable", false},
		}},
	})
	if result.Disposition() != swarmion.OperationAccepted {
		t.Fatalf("execute disposition=%s diagnostic=%v", result.Disposition(), result.Diagnostic())
	}
	waitCompleted := make(chan struct{})
	store.waitPublishedWriteAppliedForTest = func(_ context.Context, receipt PublishedWriteReceipt, _ string) (EventReceiptObservation, error) {
		defer close(waitCompleted)
		return appliedDurablyObservationForTest(receipt), nil
	}
	if err := store.recoverPublishedWriteOperations(context.Background()); err != nil {
		t.Fatalf("recover durably applied operation: %v", err)
	}
	select {
	case <-waitCompleted:
	case <-time.After(5 * time.Second):
		t.Fatal("durably applied cleanup did not finish")
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		operations, err := store.loadPublishedWriteOperations()
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, persisted := range operations {
			if operationIdentitiesEqual(persisted.Identity, operation.Identity) {
				found = true
				break
			}
		}
		if !found {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("durably applied operation %s remains in the journal", operation.Identity)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestBackgroundCleanupRequiresExactAppliedDurablyObservation(t *testing.T) {
	source := openPeerTestDB(t)
	operation, err := source.NewPublishedWriteOperation("io.protos.tests.background-cleanup/v1", []byte("durability boundary"))
	if err != nil {
		t.Fatal(err)
	}
	journal := &DB{workingDir: t.TempDir(), name: "background-cleanup"}
	if err := journal.persistPublishedWriteOperation(operation); err != nil {
		t.Fatal(err)
	}
	receipt := PublishedWriteReceipt{
		Committed:             true,
		EventID:               swarmionprotocol.NewEventID("background cleanup event").String(),
		PublishedRootHash:     swarmionprotocol.NewRootHash("background cleanup root").String(),
		AuthorPeerID:          operation.AuthorPeerID(),
		OperationIntentDigest: operation.IntentDigest(),
	}
	journal.waitPublishedWriteAppliedForTest = func(_ context.Context, receipt PublishedWriteReceipt, _ string) (EventReceiptObservation, error) {
		return EventReceiptObservation{Receipt: receipt, State: EventReceiptStatePending}, nil
	}
	journal.schedulePublishedWriteOperationCleanup(operation, receipt)
	journal.backgroundWork.wait()
	operations, err := journal.loadPublishedWriteOperations()
	if err != nil {
		t.Fatal(err)
	}
	if len(operations) != 1 || !operations[0].Equal(operation) {
		t.Fatalf("pending observation retired recovery record: %+v", operations)
	}

	journal.waitPublishedWriteAppliedForTest = func(_ context.Context, receipt PublishedWriteReceipt, _ string) (EventReceiptObservation, error) {
		observation := appliedDurablyObservationForTest(receipt)
		observation.Receipt.EventID = swarmionprotocol.NewEventID("wrong cleanup event").String()
		return observation, nil
	}
	journal.schedulePublishedWriteOperationCleanup(operation, receipt)
	journal.backgroundWork.wait()
	operations, err = journal.loadPublishedWriteOperations()
	if err != nil {
		t.Fatal(err)
	}
	if len(operations) != 1 || !operations[0].Equal(operation) {
		t.Fatalf("mismatched applied-durable observation retired recovery record: %+v", operations)
	}

	journal.waitPublishedWriteAppliedForTest = func(_ context.Context, receipt PublishedWriteReceipt, _ string) (EventReceiptObservation, error) {
		return appliedDurablyObservationForTest(receipt), nil
	}
	journal.schedulePublishedWriteOperationCleanup(operation, receipt)
	journal.backgroundWork.wait()
	operations, err = journal.loadPublishedWriteOperations()
	if err != nil {
		t.Fatal(err)
	}
	if len(operations) != 0 {
		t.Fatalf("applied-durable observation retained recovery record: %+v", operations)
	}
}

func appliedDurablyObservationForTest(receipt PublishedWriteReceipt) EventReceiptObservation {
	return EventReceiptObservation{
		Receipt: receipt,
		State:   EventReceiptStateAppliedDurably,
		Status: swarmion.ReceiptStatus{
			EventID:                   receipt.EventID,
			ExpectedPublishedRootHash: receipt.PublishedRootHash,
			Known:                     true,
			Checkpointed:              true,
			AppliedDurably:            true,
		},
	}
}

func TestAcceptedExecuteReturnsExactReceiptWhenRecoveryEnrichmentFails(t *testing.T) {
	store := openPeerTestDB(t)
	operation, err := store.NewPublishedWriteOperation("io.protos.tests.accepted-enrichment-failure/v1", []byte("accepted"))
	if err != nil {
		t.Fatal(err)
	}
	injected := errors.New("synthetic recovery rename failure")
	store.beforeOperationRecoveryReplaceForTest = func(candidate PublishedWriteOperation) error {
		if operationIdentitiesEqual(candidate.Identity, operation.Identity) {
			return injected
		}
		return nil
	}
	store.waitPublishedWriteAppliedForTest = func(_ context.Context, receipt PublishedWriteReceipt, reason string) (EventReceiptObservation, error) {
		observation := EventReceiptObservation{Receipt: receipt, State: EventReceiptStatePending}
		return observation, &EventReceiptPendingError{Observation: observation, Reason: reason, Cause: context.DeadlineExceeded}
	}
	userID := MustNewUUIDv7()
	receipt, executeErr := store.executePublishedWriteOperationContext(
		context.Background(),
		operation,
		"accepted recovery enrichment failure",
		func(ctx context.Context, executor sqlContextExecer) error {
			_, err := executor.ExecContext(
				ctx,
				"INSERT INTO users (id, username, name, is_disabled) VALUES (?, ?, ?, ?)",
				MustUUIDBytes(userID), "accepted-enrichment-failure", "accepted-enrichment-failure", false,
			)
			return err
		},
	)
	if executeErr != nil {
		t.Fatalf("accepted execute exposed enrichment failure as a publication error: %v", executeErr)
	}
	if !receipt.Committed || !receipt.HasExactEventIdentity() {
		t.Fatalf("accepted execute lost exact authority: receipt=%+v error=%v", receipt, executeErr)
	}
	if got := store.TransactionMetrics().OperationRecoveryPersistenceFailures; got != 1 {
		t.Fatalf("recovery persistence failure metric=%d, want 1 for %v", got, injected)
	}
	resolved, resolveErr := store.LookupPublishedWriteOperation(context.Background(), operation)
	if resolveErr != nil {
		t.Fatalf("accepted lookup exposed enrichment failure: %v", resolveErr)
	}
	if resolved.Disposition() != swarmion.OperationAccepted {
		t.Fatalf("resolved disposition=%s diagnostic=%v, want accepted", resolved.Disposition(), resolved.Diagnostic())
	}
	resolvedReceipt, err := PublishedWriteReceiptFromResult(operation, resolved)
	if err != nil {
		t.Fatal(err)
	}
	if resolvedReceipt.EventID != receipt.EventID || resolvedReceipt.PublishedRootHash != receipt.PublishedRootHash {
		t.Fatalf("resolved receipt=%+v, want original %+v", resolvedReceipt, receipt)
	}
	if got := store.TransactionMetrics().OperationRecoveryPersistenceFailures; got != 2 {
		t.Fatalf("recovery persistence failure metric=%d, want Execute and Lookup failures", got)
	}
	store.beforeOperationRecoveryReplaceForTest = nil
}

func TestOrdinaryAcceptedWriteDoesNotExposeRecoveryEnrichmentFailureAsRetry(t *testing.T) {
	store := openPeerTestDB(t)
	injected := errors.New("synthetic ordinary recovery sync failure")
	store.beforeOperationRecoveryReplaceForTest = func(candidate PublishedWriteOperation) error {
		if _, hasReceipt := candidate.Recovery.ExpectedReceipt(); hasReceipt {
			return injected
		}
		return nil
	}
	store.waitPublishedWriteAppliedForTest = func(_ context.Context, receipt PublishedWriteReceipt, reason string) (EventReceiptObservation, error) {
		observation := EventReceiptObservation{Receipt: receipt, State: EventReceiptStatePending}
		return observation, &EventReceiptPendingError{Observation: observation, Reason: reason, Cause: context.DeadlineExceeded}
	}
	userID := MustNewUUIDv7()
	receipt, err := InsertWithReceiptContext(context.Background(), store, func() sq.InsertQuery {
		user := sq.New[USER]("")
		return sq.InsertInto(user).ColumnValues(func(column *sq.Column) {
			column.SetBytes(user.ID, MustUUIDBytes(userID))
			column.SetString(user.USERNAME, "ordinary-enrichment-failure")
			column.SetString(user.NAME, "ordinary-enrichment-failure")
			column.SetBool(user.IS_DISABLED, false)
		})
	})
	if err != nil {
		t.Fatalf("ordinary accepted write exposed a retryable error: %v", err)
	}
	if !receipt.Committed || !receipt.HasExactEventIdentity() {
		t.Fatalf("ordinary accepted write receipt=%+v, want exact committed evidence", receipt)
	}
	if got := store.TransactionMetrics().OperationRecoveryPersistenceFailures; got != 1 {
		t.Fatalf("recovery persistence failure metric=%d, want 1 for %v", got, injected)
	}
	var count int
	if err := store.ReadRows(
		context.Background(),
		"SELECT COUNT(*) FROM users WHERE id = ?",
		[]any{MustUUIDBytes(userID)},
		func(rows *sql.Rows) error {
			if !rows.Next() {
				return sql.ErrNoRows
			}
			return rows.Scan(&count)
		},
	); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("ordinary accepted write row count=%d, want 1", count)
	}
}

func TestUnavailableRecoveryRetainsExactJournalRecord(t *testing.T) {
	store := openPeerTestDB(t)
	operation, err := store.NewPublishedWriteOperation("io.protos.tests.unavailable-recovery/v1", []byte("ambiguous"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	err = store.recoverPublishedWriteOperations(context.Background())
	if !errors.Is(err, ErrOperationReceiptUnavailable) {
		t.Fatalf("closed-runtime recovery error=%v, want receipt unavailable", err)
	}
	operations, loadErr := store.loadPublishedWriteOperations()
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	for _, persisted := range operations {
		if persisted.Equal(operation) {
			return
		}
	}
	t.Fatalf("ambiguous operation %s was removed from journal", operation.Identity)
}

func TestPreviewKeyDigestOnlyJournalRecordFailsClosed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preview.json")
	legacy := []byte(`{"key":"preview-key","intent_digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`)
	if err := os.WriteFile(path, legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readPersistedPublishedWriteOperation(path); err == nil {
		t.Fatal("preview key/digest-only journal record was accepted as v1 recovery")
	}
}

func TestCloseSealsOperationCleanupAdmissionBeforeWaiting(t *testing.T) {
	store := &DB{}
	enteredAdmissionHook := make(chan struct{})
	releaseAdmissionHook := make(chan struct{})
	store.beforeOperationCleanupAdmissionForTest = func() {
		close(enteredAdmissionHook)
		<-releaseAdmissionHook
	}
	receipt := PublishedWriteReceipt{
		Committed:         true,
		EventID:           swarmionprotocol.NewEventID("close cleanup admission event").String(),
		PublishedRootHash: swarmionprotocol.NewRootHash("close cleanup admission root").String(),
	}
	scheduled := make(chan struct{})
	go func() {
		store.schedulePublishedWriteOperationCleanup(PublishedWriteOperation{}, receipt)
		close(scheduled)
	}()
	<-enteredAdmissionHook

	if err := store.closeSwarmionRuntimeLocked(); err != nil {
		t.Fatalf("close database: %v", err)
	}
	close(releaseAdmissionHook)
	<-scheduled
	if got := store.backgroundWork.admissionCount(); got != 0 {
		t.Fatalf("cleanup admitted after close sealed registry: admissions=%d", got)
	}
	if _, admitted := store.backgroundWork.begin(); admitted {
		t.Fatal("background work registry admitted work after close")
	}
}
