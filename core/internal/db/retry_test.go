package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	swarmionprotocol "github.com/nustiueudinastea/swarmion/protocol"
	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
)

func TestSQLViewReadinessRetriesRequireTypedPreExecutionOutcome(t *testing.T) {
	viewNotReady := &swarmionapp.SQLViewNotReadyError{LineID: "main", Reason: "test view not ready"}
	queryCalls := 0
	rows, err := queryWhenSQLViewReady(context.Background(), func() (*sql.Rows, error) { //nolint:rowserrcheck,sqlclosecheck // synthetic nil rows exercise retry classification
		queryCalls++
		if queryCalls == 1 {
			return nil, viewNotReady
		}
		return nil, nil
	})
	if err != nil || rows != nil || queryCalls != 2 {
		t.Fatalf("typed query readiness retry rows=%v error=%v calls=%d, want nil/nil/2", rows, err, queryCalls)
	}

	execCalls := 0
	result, err := execWhenSQLViewReady(context.Background(), func() (sql.Result, error) {
		execCalls++
		if execCalls == 1 {
			return nil, fmt.Errorf("remote projection: %w", viewNotReady)
		}
		return nil, nil
	})
	if err != nil || result != nil || execCalls != 2 {
		t.Fatalf("typed statement readiness retry result=%v error=%v calls=%d, want nil/nil/2", result, err, execCalls)
	}

	textCalls := 0
	_, err = queryWhenSQLViewReady(context.Background(), func() (*sql.Rows, error) { //nolint:rowserrcheck,sqlclosecheck // synthetic nil rows exercise text-only rejection
		textCalls++
		return nil, errors.New("swarmion SQL view is not ready")
	})
	if err == nil || textCalls != 1 {
		t.Fatalf("diagnostic-text readiness error=%v calls=%d, want one attempt", err, textCalls)
	}

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	canceledCalls := 0
	_, err = queryWhenSQLViewReady(canceledCtx, func() (*sql.Rows, error) { //nolint:rowserrcheck,sqlclosecheck // synthetic nil rows exercise canceled readiness
		canceledCalls++
		return nil, viewNotReady
	})
	if !errors.Is(err, context.Canceled) || !errors.Is(err, swarmionapp.ErrSQLViewNotReady) || canceledCalls != 1 {
		t.Fatalf("canceled readiness error=%v calls=%d, want typed readiness plus cancellation after one attempt", err, canceledCalls)
	}
}

func TestUncertainOrdinaryReceiptWaitCancellationRetainsExactReceipt(t *testing.T) {
	receipt := eventReceiptForTest()
	receipt.Committed = false
	receipt.OutcomeUncertain = true
	database := &DB{}
	ctx, cancel := context.WithCancel(context.Background())
	database.observePublishedWriteForTest = func(_ context.Context, candidate PublishedWriteReceipt) (EventReceiptObservation, error) {
		cancel()
		return EventReceiptObservation{
			Receipt: candidate,
			State:   EventReceiptStatePending,
		}, nil
	}

	err := database.waitForOrdinaryPublishedWriteKnown(ctx, receipt)
	var pending *EventReceiptPendingError
	if !errors.As(err, &pending) || pending == nil {
		t.Fatalf("canceled uncertain-receipt wait=%v, want typed pending error", err)
	}
	if pending.Observation.Receipt.EventID != receipt.EventID ||
		pending.Observation.Receipt.PublishedRootHash != receipt.PublishedRootHash {
		t.Fatalf("canceled wait lost exact receipt: %+v", pending.Observation.Receipt)
	}
	if !errors.Is(err, context.Canceled) || IsRetryablePublishedWriteError(err) {
		t.Fatalf("canceled exact-receipt wait classification=%v, want cancellation without replay permission", err)
	}
}

func TestTransientReadErrorsIncludeDoltWorkingSetContention(t *testing.T) {
	tests := []error{
		errors.New(`failed to load database "protos": the database is locked by another dolt process`),
		errors.New("update tentative working set: cannot update manifest: database is read only"),
		errors.New("update tentative working set: cannot update manifest"),
	}

	for _, err := range tests {
		if !isTransientWorkspaceAccessError(err) {
			t.Fatalf("expected retryable error: %v", err)
		}
	}
}

func TestRetryablePublishedWriteErrorsRequireTypedNotAcceptedProof(t *testing.T) {
	viewNotReady := &swarmionapp.SQLViewNotReadyError{
		LineID: "main",
		Reason: "test view not ready",
	}
	for _, err := range []error{
		errors.New("duplicate primary key"),
		fmt.Errorf("operation lookup: %w", swarmionprotocol.ErrOperationKeyConflict),
		ErrOperationReceiptUnavailable,
		errors.New("staged SQL root=abc conflicts with protocol root=def"),
		&swarmionapp.ContentConflictError{CandidateRootHash: "abc", ProtocolRootHash: "def"},
		viewNotReady,
		errors.Join(viewNotReady, errors.New("publication failed")),
		errors.New("swarmion SQL view is not ready"),
	} {
		if IsRetryablePublishedWriteError(err) {
			t.Fatalf("application error must fail without a publication retry: %v", err)
		}
	}

	safeOutcome := swarmionapp.PublicationOutcome{
		Identity:        swarmionapp.OperationIdentity{Key: "retry-test", IntentDigest: strings.Repeat("a", 64)},
		Scope:           swarmionapp.DatabasePublicationScope("/protos/db/retry-test"),
		AuthorPeerID:    "peer-a",
		State:           swarmionapp.PublicationRejectedSafeToRetry,
		RejectionReason: swarmionapp.PublicationRejectionNotAccepted,
	}
	safe := &swarmionapp.PublicationRejectedError{Outcome: safeOutcome, Cause: errors.New("event admission rejected")}
	if !IsRetryablePublishedWriteError(safe) {
		t.Fatalf("typed not-accepted outcome should be retryable: %v", safe)
	}
	for _, stale := range []error{
		swarmionprotocol.ErrStaleWriteContext,
		fmt.Errorf("capture current SQL projection: %w", swarmionprotocol.ErrStaleWriteContext),
		swarmionprotocol.ErrProjectionTooWide,
		&swarmionprotocol.ProjectionTooWideError{HeadCount: 9, MaxHeads: 8},
		fmt.Errorf("capture current SQL projection: %w", &swarmionprotocol.ProjectionTooWideError{HeadCount: 9, MaxHeads: 8}),
	} {
		if !IsRetryablePublishedWriteError(stale) {
			t.Fatalf("typed pre-publication projection error should be retryable: %v", stale)
		}
	}
	if IsRetryablePublishedWriteError(errors.Join(safe, errors.New("unrelated sibling"))) {
		t.Fatal("joined sibling errors must not preserve publication retry authority")
	}
	if IsRetryablePublishedWriteError(errors.Join(swarmionprotocol.ErrStaleWriteContext, errors.New("unrelated sibling"))) {
		t.Fatal("joined sibling errors must not preserve stale-context retry authority")
	}

	receipt := eventReceiptForTest()
	for _, test := range []struct {
		name     string
		boundary error
		cause    error
	}{
		{
			name:     "event pending rejection",
			boundary: &EventReceiptPendingError{Observation: EventReceiptObservation{Receipt: receipt}, Cause: safe},
			cause:    safe,
		},
		{
			name:     "event pending stale projection",
			boundary: &EventReceiptPendingError{Observation: EventReceiptObservation{Receipt: receipt}, Cause: swarmionprotocol.ErrStaleWriteContext},
			cause:    swarmionprotocol.ErrStaleWriteContext,
		},
		{
			name: "availability pending rejection",
			boundary: &PublishedWriteAvailabilityPendingError{
				Observation: PublishedWriteAvailabilityObservation{Receipt: receipt},
				Cause:       safe,
			},
			cause: safe,
		},
		{
			name: "availability pending stale projection",
			boundary: &PublishedWriteAvailabilityPendingError{
				Observation: PublishedWriteAvailabilityObservation{Receipt: receipt},
				Cause:       swarmionprotocol.ErrStaleWriteContext,
			},
			cause: swarmionprotocol.ErrStaleWriteContext,
		},
		{
			name: "receipt identity conflict rejection",
			boundary: &PublishedWriteReceiptIdentityConflictError{
				Receipt: receipt,
				Cause:   safe,
			},
			cause: safe,
		},
		{
			name: "receipt identity conflict stale projection",
			boundary: &PublishedWriteReceiptIdentityConflictError{
				Receipt: receipt,
				Cause:   swarmionprotocol.ErrStaleWriteContext,
			},
			cause: swarmionprotocol.ErrStaleWriteContext,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			wrapped := fmt.Errorf("caller context: %w", test.boundary)
			if !errors.Is(wrapped, test.cause) {
				t.Fatalf("synthetic boundary did not retain its diagnostic cause: %v", wrapped)
			}
			if IsRetryablePublishedWriteError(wrapped) {
				t.Fatalf("exact-receipt boundary leaked nested retry authority: %v", wrapped)
			}
		})
	}

	for _, boundary := range []error{
		&PublishedWriteConfirmationUnresolvedError{
			Confirmation: PublishedWriteConfirmation{Receipt: receipt},
			Cause:        safe,
		},
		&EventReceiptParkedError{Observation: EventReceiptObservation{Receipt: receipt}},
	} {
		if IsRetryablePublishedWriteError(fmt.Errorf("caller context: %w", boundary)) {
			t.Fatalf("explicit non-retryable lifecycle boundary granted retry: %v", boundary)
		}
	}
}

func TestValidateEventReceiptIdentityRequiresEventAndRoot(t *testing.T) {
	receipt := eventReceiptForTest()
	if _, _, err := validateEventReceiptIdentity(receipt.EventID, receipt.PublishedRootHash); err != nil {
		t.Fatalf("valid receipt identity: %v", err)
	}
	for _, incomplete := range []PublishedWriteReceipt{
		{PublishedRootHash: receipt.PublishedRootHash},
		{EventID: receipt.EventID},
	} {
		if _, _, err := validateEventReceiptIdentity(incomplete.EventID, incomplete.PublishedRootHash); !errors.Is(err, errSwarmionPublishedWriteIncomplete) {
			t.Fatalf("incomplete receipt error=%v, want %v", err, errSwarmionPublishedWriteIncomplete)
		}
	}
}

func TestObserveEventReceiptAppliedDurablyTerminatesForEveryCoverage(t *testing.T) {
	for _, coverage := range []swarmionapp.BranchEventContentCoverage{
		swarmionapp.BranchEventContentCoverageCovered,
		swarmionapp.BranchEventContentCoverageDissent,
		swarmionapp.BranchEventContentCoverageUnavailable,
	} {
		t.Run(string(coverage), func(t *testing.T) {
			receipt := eventReceiptForTest()
			runtime := &scriptedEventReceiptRuntime{statuses: []swarmionapp.ReceiptStatus{appliedEventStatusForTest(receipt, coverage)}}
			observation, err := observePublishedWriteReceipt(context.Background(), runtime, receipt)
			if err != nil {
				t.Fatalf("observe applied receipt: %v", err)
			}
			if observation.State != EventReceiptStateAppliedDurably || !observation.Status.AppliedDurably {
				t.Fatalf("observation=%+v, want applied_durably", observation)
			}
			if observation.Receipt.CheckpointCommitID == "" || observation.Receipt.CheckpointRootHash == "" {
				t.Fatalf("observation did not retain checkpoint identity: %+v", observation)
			}
			if !reflect.DeepEqual(runtime.trace, []string{"status"}) {
				t.Fatalf("runtime trace=%v, want event status only", runtime.trace)
			}
		})
	}
}

func TestObserveEventReceiptUsesExactEventParking(t *testing.T) {
	receipt := eventReceiptForTest()
	tests := []struct {
		name   string
		reason string
		parked bool
		want   EventReceiptState
	}{
		{name: "pending", want: EventReceiptStatePending},
		{name: "conflict", reason: swarmionapp.BranchRootParkedReasonConflict, parked: true, want: EventReceiptStateParkedConflict},
		{name: "dependency parked", reason: swarmionapp.BranchRootParkedReasonDependencyParked, parked: true, want: EventReceiptStateDependencyParked},
		{name: "stale anchor", reason: swarmionapp.BranchRootParkedReasonStaleAnchor, parked: true, want: EventReceiptStateStaleAnchor},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := pendingEventStatusForTest(receipt)
			status.Parked = tt.parked
			status.ParkedReason = tt.reason
			status.Revisitable = tt.parked
			runtime := &scriptedEventReceiptRuntime{
				statuses: []swarmionapp.ReceiptStatus{status},
			}
			observation, err := observePublishedWriteReceipt(context.Background(), runtime, receipt)
			if err != nil {
				t.Fatalf("observe parked receipt: %v", err)
			}
			if observation.State != tt.want {
				t.Fatalf("state=%q, want %q: %+v", observation.State, tt.want, observation)
			}
			if !reflect.DeepEqual(runtime.trace, []string{"status"}) {
				t.Fatalf("runtime trace=%v, want exact event status only", runtime.trace)
			}
		})
	}
}

func TestValidateEventReceiptStatusRejectsInconsistentExactParking(t *testing.T) {
	receipt := eventReceiptForTest()
	eventID := swarmionprotocol.ParseEventID(receipt.EventID)
	root := swarmionprotocol.ParseRootHash(receipt.PublishedRootHash)
	tests := []struct {
		name   string
		mutate func(*swarmionapp.ReceiptStatus)
	}{
		{
			name: "parked without revisitable",
			mutate: func(status *swarmionapp.ReceiptStatus) {
				status.Parked = true
				status.ParkedReason = swarmionapp.BranchRootParkedReasonConflict
			},
		},
		{
			name: "parked with unknown reason",
			mutate: func(status *swarmionapp.ReceiptStatus) {
				status.Parked = true
				status.ParkedReason = "future_reason"
				status.Revisitable = true
			},
		},
		{
			name: "non parked with stale metadata",
			mutate: func(status *swarmionapp.ReceiptStatus) {
				status.ParkedReason = swarmionapp.BranchRootParkedReasonConflict
				status.Revisitable = true
			},
		},
		{
			name: "checkpointed and parked",
			mutate: func(status *swarmionapp.ReceiptStatus) {
				status.Checkpointed = true
				status.CheckpointCommitID = swarmionprotocol.NewCheckpointCommitID("parked checkpoint").String()
				status.CheckpointRootHash = swarmionprotocol.NewRootHash("parked checkpoint root").String()
				status.Parked = true
				status.ParkedReason = swarmionapp.BranchRootParkedReasonDependencyParked
				status.Revisitable = true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := pendingEventStatusForTest(receipt)
			tt.mutate(&status)
			if err := validateEventReceiptStatus(eventID, root, status); !errors.Is(err, errSwarmionPublishedWriteIncomplete) {
				t.Fatalf("validation error=%v, want %v", err, errSwarmionPublishedWriteIncomplete)
			}
		})
	}
}

func TestWaitForEventReceiptUsesTypedWaitAndStopsOnDissent(t *testing.T) {
	receipt := eventReceiptForTest()
	runtime := &scriptedEventReceiptRuntime{
		statuses: []swarmionapp.ReceiptStatus{
			pendingEventStatusForTest(receipt),
			appliedEventStatusForTest(receipt, swarmionapp.BranchEventContentCoverageDissent),
		},
	}
	observation, err := waitForPublishedWriteApplied(context.Background(), runtime, receipt, "delete test")
	if err != nil {
		t.Fatalf("wait for dissent receipt: %v", err)
	}
	if observation.Status.ContentCoverage != swarmionapp.BranchEventContentCoverageDissent || observation.Status.Durable {
		t.Fatalf("observation=%+v, want applied dissent without full-root durable", observation)
	}
	wantTrace := []string{"wait_status"}
	if !reflect.DeepEqual(runtime.trace, wantTrace) {
		t.Fatalf("runtime trace=%v, want %v", runtime.trace, wantTrace)
	}
}

func TestWaitForEventReceiptReturnsTypedParkedWithoutPublication(t *testing.T) {
	receipt := eventReceiptForTest()
	parked := pendingEventStatusForTest(receipt)
	parked.Parked = true
	parked.ParkedReason = swarmionapp.BranchRootParkedReasonDependencyParked
	parked.Revisitable = true
	runtime := &scriptedEventReceiptRuntime{
		statuses: []swarmionapp.ReceiptStatus{parked},
	}
	observation, err := waitForPublishedWriteApplied(context.Background(), runtime, receipt, "delete parked test")
	if !errors.Is(err, ErrEventReceiptParked) {
		t.Fatalf("wait error=%v, want %v", err, ErrEventReceiptParked)
	}
	var parkedErr *EventReceiptParkedError
	if !errors.As(err, &parkedErr) || parkedErr.Observation.State != EventReceiptStateDependencyParked {
		t.Fatalf("parked error=%#v, observation=%+v", parkedErr, observation)
	}
	if parkedErr.Observation.Status.ParkedReason != swarmionapp.BranchRootParkedReasonDependencyParked || !parkedErr.Observation.Status.Revisitable {
		t.Fatalf("parked status=%+v, want exact revisitable dependency classification", parkedErr.Observation.Status)
	}
	if !reflect.DeepEqual(runtime.trace, []string{"wait_status"}) {
		t.Fatalf("runtime trace=%v, want exact-event wait only", runtime.trace)
	}
}

func TestWaitForEventReceiptPendingIsBoundedAndRetainsCheckpoint(t *testing.T) {
	receipt := eventReceiptForTest()
	pending := pendingEventStatusForTest(receipt)
	pending.Checkpointed = true
	pending.CheckpointCommitID = swarmionprotocol.NewCheckpointCommitID("pending checkpoint").String()
	pending.CheckpointRootHash = swarmionprotocol.NewRootHash("pending checkpoint root").String()
	runtime := &scriptedEventReceiptRuntime{
		statuses: []swarmionapp.ReceiptStatus{pending},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	observation, err := waitForPublishedWriteApplied(ctx, runtime, receipt, "bounded pending test")
	if !errors.Is(err, ErrEventReceiptPending) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("wait error=%v, want typed pending deadline", err)
	}
	var pendingErr *EventReceiptPendingError
	if !errors.As(err, &pendingErr) {
		t.Fatalf("wait error type=%T, want *EventReceiptPendingError", err)
	}
	if observation.Receipt.CheckpointCommitID != pending.CheckpointCommitID || pendingErr.Observation.Receipt.CheckpointRootHash != pending.CheckpointRootHash {
		t.Fatalf("pending observation lost checkpoint identity: observation=%+v error=%+v", observation, pendingErr)
	}
	if !reflect.DeepEqual(runtime.trace, []string{"wait_status"}) {
		t.Fatalf("runtime trace=%v, want exact-event wait only", runtime.trace)
	}
}

func TestWaitForEventReceiptTerminalCloseRetainsSatisfiedLatestSnapshot(t *testing.T) {
	receipt := eventReceiptForTest()
	status := appliedEventStatusForTest(receipt, swarmionapp.BranchEventContentCoverageCovered)
	closed := &swarmionapp.DatabaseRuntimeClosedError{}
	runtime := &terminalEventReceiptRuntime{
		result: swarmionapp.ReceiptWaitResult{
			Snapshot: swarmionapp.ReceiptSnapshot{
				Receipt: swarmionapp.EventReceipt{
					EventID:           receipt.EventID,
					PublishedRootHash: receipt.PublishedRootHash,
				},
				Event:      status,
				ObservedAt: time.Now(),
			},
			Condition: swarmionapp.ReceiptConditionAppliedDurably,
			Satisfied: false,
		},
		err: closed,
	}

	observation, err := waitForPublishedWriteApplied(context.Background(), runtime, receipt, "terminal handoff test")
	if !errors.Is(err, ErrEventReceiptPending) || !errors.Is(err, swarmionapp.ErrDatabaseRuntimeClosed) {
		t.Fatalf("terminal wait error=%v, want pending plus typed runtime closure", err)
	}
	var pending *EventReceiptPendingError
	if !errors.As(err, &pending) || pending == nil {
		t.Fatalf("terminal wait error=%T, want *EventReceiptPendingError", err)
	}
	if observation.State != EventReceiptStateAppliedDurably || !observation.Status.AppliedDurably ||
		pending.Observation.State != EventReceiptStateAppliedDurably ||
		pending.Observation.Receipt.EventID != receipt.EventID ||
		pending.Observation.Receipt.PublishedRootHash != receipt.PublishedRootHash {
		t.Fatalf("terminal wait lost its latest satisfied snapshot: observation=%+v pending=%+v", observation, pending.Observation)
	}
}

func TestRecordContentDissentObservationDeduplicatesExactEventAndHeadTuple(t *testing.T) {
	receipt := eventReceiptForTest()
	database := &DB{}
	observation := EventReceiptObservation{
		Receipt: receipt,
		Status:  appliedEventStatusForTest(receipt, swarmionapp.BranchEventContentCoverageDissent),
		State:   EventReceiptStateAppliedDurably,
	}
	database.recordTerminalEventReceiptObservation("metric test", observation)
	database.recordTerminalEventReceiptObservation("same tuple, different caller", observation)
	if got := database.EventReceiptMetrics().ContentDissentObservations; got != 1 {
		t.Fatalf("dissent observations=%d, want one emission for the repeated exact tuple", got)
	}
}

func TestRecordContentDissentObservationEmitsWhenDurableHeadChanges(t *testing.T) {
	receipt := eventReceiptForTest()
	database := &DB{}
	observation := EventReceiptObservation{
		Receipt: receipt,
		Status:  appliedEventStatusForTest(receipt, swarmionapp.BranchEventContentCoverageDissent),
		State:   EventReceiptStateAppliedDurably,
	}
	database.recordTerminalEventReceiptObservation("initial head", observation)

	observation.Status.DurableCheckpointCommitID = swarmionprotocol.NewCheckpointCommitID("new durable checkpoint").String()
	observation.Status.DurableCheckpointRootHash = swarmionprotocol.NewRootHash("new durable checkpoint root").String()
	observation.Status.QueryableRootHash = swarmionprotocol.NewRootHash("new queryable root").String()
	database.recordTerminalEventReceiptObservation("new head", observation)
	if got := database.EventReceiptMetrics().ContentDissentObservations; got != 2 {
		t.Fatalf("dissent observations=%d, want one emission for each durable/queryable head tuple", got)
	}
}

type scriptedEventReceiptRuntime struct {
	statuses []swarmionapp.ReceiptStatus
	trace    []string
	statusAt int
}

type terminalEventReceiptRuntime struct {
	result swarmionapp.ReceiptWaitResult
	err    error
}

func (r *terminalEventReceiptRuntime) ObserveReceipt(context.Context, swarmionapp.ReceiptTrackingRequest) (swarmionapp.ReceiptSnapshot, error) {
	return swarmionapp.ReceiptSnapshot{}, errors.New("terminal event receipt runtime does not support observation")
}

func (r *terminalEventReceiptRuntime) WaitReceipt(context.Context, swarmionapp.ReceiptWaitRequest) (swarmionapp.ReceiptWaitResult, error) {
	return r.result, r.err
}

func (r *scriptedEventReceiptRuntime) ObserveReceipt(_ context.Context, request swarmionapp.ReceiptTrackingRequest) (swarmionapp.ReceiptSnapshot, error) {
	r.trace = append(r.trace, "status")
	if len(r.statuses) == 0 {
		return swarmionapp.ReceiptSnapshot{}, fmt.Errorf("no scripted event status")
	}
	index := r.statusAt
	if index >= len(r.statuses) {
		index = len(r.statuses) - 1
	}
	r.statusAt++
	return swarmionapp.ReceiptSnapshot{
		Receipt:    request.Receipt,
		Event:      r.statuses[index],
		ObservedAt: time.Now(),
	}, nil
}

func (r *scriptedEventReceiptRuntime) WaitReceipt(ctx context.Context, request swarmionapp.ReceiptWaitRequest) (swarmionapp.ReceiptWaitResult, error) {
	r.trace = append(r.trace, "wait_status")
	if len(r.statuses) == 0 {
		return swarmionapp.ReceiptWaitResult{}, fmt.Errorf("no scripted event status")
	}
	result := func(status swarmionapp.ReceiptStatus) swarmionapp.ReceiptWaitResult {
		return swarmionapp.ReceiptWaitResult{
			Snapshot: swarmionapp.ReceiptSnapshot{
				Receipt:    request.Tracking.Receipt,
				Event:      status,
				ObservedAt: time.Now(),
			},
			Satisfied: eventReceiptConditionSatisfiedForTest(status, request.Condition),
			Condition: request.Condition,
		}
	}
	for r.statusAt < len(r.statuses) {
		status := r.statuses[r.statusAt]
		r.statusAt++
		if status.AppliedDurably {
			return result(status), nil
		}
		if status.Parked {
			return result(status), fmt.Errorf("scripted parked event remains revisitable")
		}
	}
	status := r.statuses[len(r.statuses)-1]
	<-ctx.Done()
	return result(status), ctx.Err()
}

func eventReceiptConditionSatisfiedForTest(status swarmionapp.ReceiptStatus, condition swarmionapp.ReceiptCondition) bool {
	switch condition {
	case swarmionapp.ReceiptConditionLocallyAccepted:
		return status.Known
	case swarmionapp.ReceiptConditionCheckpointed:
		return status.Checkpointed
	case swarmionapp.ReceiptConditionAppliedDurably:
		return status.AppliedDurably
	case swarmionapp.ReceiptConditionContentEvaluated:
		return status.AppliedDurably &&
			(status.ContentCoverage == swarmionapp.BranchEventContentCoverageCovered ||
				status.ContentCoverage == swarmionapp.BranchEventContentCoverageDissent)
	case swarmionapp.ReceiptConditionContentCovered:
		return status.Durable && status.ContentCoverage == swarmionapp.BranchEventContentCoverageCovered
	case swarmionapp.ReceiptConditionParked:
		return status.Parked
	default:
		return false
	}
}

func eventReceiptForTest() PublishedWriteReceipt {
	return PublishedWriteReceipt{
		Committed:         true,
		EventID:           swarmionprotocol.NewEventID("backend event receipt test").String(),
		PublishedRootHash: swarmionprotocol.NewRootHash("backend event receipt root").String(),
	}
}

func pendingEventStatusForTest(receipt PublishedWriteReceipt) swarmionapp.ReceiptStatus {
	return swarmionapp.ReceiptStatus{
		EventID:                   receipt.EventID,
		ExpectedPublishedRootHash: receipt.PublishedRootHash,
		Known:                     true,
		ContentCoverage:           swarmionapp.BranchEventContentCoveragePending,
	}
}

func appliedEventStatusForTest(receipt PublishedWriteReceipt, coverage swarmionapp.BranchEventContentCoverage) swarmionapp.ReceiptStatus {
	status := swarmionapp.ReceiptStatus{
		EventID:                   receipt.EventID,
		ExpectedPublishedRootHash: receipt.PublishedRootHash,
		Known:                     true,
		Checkpointed:              true,
		AppliedDurably:            true,
		Durable:                   coverage == swarmionapp.BranchEventContentCoverageCovered,
		ContentCoverage:           coverage,
		CheckpointCommitID:        swarmionprotocol.NewCheckpointCommitID("event checkpoint").String(),
		CheckpointRootHash:        swarmionprotocol.NewRootHash("event checkpoint root").String(),
		DurableCheckpointCommitID: swarmionprotocol.NewCheckpointCommitID("durable checkpoint").String(),
		DurableCheckpointRootHash: swarmionprotocol.NewRootHash("durable checkpoint root").String(),
		QueryableRootHash:         swarmionprotocol.NewRootHash("queryable root").String(),
	}
	proof := &swarmionapp.BranchRootDurableProofObservation{
		TargetRootHash:              status.ExpectedPublishedRootHash,
		QueryableRootHash:           status.QueryableRootHash,
		DurableCheckpointCommitID:   status.DurableCheckpointCommitID,
		DurableCheckpointRootHash:   status.DurableCheckpointRootHash,
		CandidateCheckpointCommitID: status.CheckpointCommitID,
		CandidateCheckpointRootHash: status.CheckpointRootHash,
		CandidateEventID:            status.EventID,
		EventContained:              true,
	}
	switch coverage {
	case swarmionapp.BranchEventContentCoverageCovered:
		proof.ProofRan = true
		proof.Covered = true
		proof.MergedRootHash = status.QueryableRootHash
	case swarmionapp.BranchEventContentCoverageDissent:
		proof.ProofRan = true
		proof.Conflict = true
	case swarmionapp.BranchEventContentCoverageUnavailable:
		proof.ProofUnavailable = true
	}
	status.DurableProofObservation = proof
	return status
}
