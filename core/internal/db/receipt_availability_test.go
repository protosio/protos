package db

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	swarmionprotocol "github.com/nustiueudinastea/swarmion/protocol"
	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
)

type scriptedPublishedWriteAvailabilityRuntime struct {
	snapshot   swarmionapp.ReceiptSnapshot
	observeErr error
	waitResult swarmionapp.ReceiptWaitResult
	waitErr    error
	wait       func(context.Context, swarmionapp.ReceiptWaitRequest) (swarmionapp.ReceiptWaitResult, error)
	requests   []swarmionapp.ReceiptTrackingRequest
	trace      []string
}

func (r *scriptedPublishedWriteAvailabilityRuntime) ObserveReceipt(
	_ context.Context,
	request swarmionapp.ReceiptTrackingRequest,
) (swarmionapp.ReceiptSnapshot, error) {
	r.trace = append(r.trace, "status")
	r.requests = append(r.requests, request)
	return r.snapshot, r.observeErr
}

func (r *scriptedPublishedWriteAvailabilityRuntime) WaitReceipt(
	ctx context.Context,
	request swarmionapp.ReceiptWaitRequest,
) (swarmionapp.ReceiptWaitResult, error) {
	r.trace = append(r.trace, "wait")
	r.requests = append(r.requests, request.Tracking)
	if r.wait != nil {
		return r.wait(ctx, request)
	}
	return r.waitResult, r.waitErr
}

func TestReceiptTrackingRequestFromPublishedWriteReceiptPreservesExactIdentityAndScope(t *testing.T) {
	receipt := eventReceiptForTest()
	receipt.Committed = false
	receipt.OutcomeUncertain = true
	request, err := ReceiptTrackingRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{
		MinimumOtherPeers: 2,
		PeerIDs:           []string{" peer-b ", "peer-a", "peer-b"},
		MaxObservationAge: 3 * time.Second,
	})
	if err != nil {
		t.Fatalf("convert exact receipt: %v", err)
	}
	if request.Receipt.EventID != receipt.EventID || request.Receipt.PublishedRootHash != receipt.PublishedRootHash {
		t.Fatalf("request changed exact identity: receipt=%+v request=%+v", receipt, request)
	}
	if request.OtherPeerRetention == nil || request.OtherPeerRetention.MinimumPeers != 2 ||
		request.OtherPeerRetention.MaxObservationAge != 3*time.Second ||
		!reflect.DeepEqual(request.OtherPeerRetention.PeerIDs, []string{"peer-a", "peer-b"}) {
		t.Fatalf("request scope was not normalized: %+v", request)
	}
	if receipt.Committed || !receipt.OutcomeUncertain {
		t.Fatalf("test receipt no longer represents an uncertain accepted outcome: %+v", receipt)
	}

	defaultRequest, err := ReceiptTrackingRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil || defaultRequest.OtherPeerRetention == nil || defaultRequest.OtherPeerRetention.MinimumPeers != 1 {
		t.Fatalf("default request=%+v error=%v, want one other peer", defaultRequest, err)
	}

	for _, options := range []PublishedWriteAvailabilityOptions{
		{MinimumOtherPeers: -1},
		{MaxObservationAge: -time.Nanosecond},
		{PeerIDs: []string{"peer-a", " "}},
		{PeerIDs: []string{"peer-a", string([]byte{0xff})}},
		{PeerIDs: []string{"peer-a", "peer-\ufffd"}},
	} {
		if _, err := ReceiptTrackingRequestFromPublishedWriteReceipt(receipt, options); err == nil {
			t.Fatalf("invalid options %+v were accepted", options)
		}
	}
}

func TestWaitForPublishedWriteAvailabilityReturnsExactOtherPeerSuccess(t *testing.T) {
	receipt := eventReceiptForTest()
	request, err := ReceiptTrackingRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil {
		t.Fatal(err)
	}
	runtime := &scriptedPublishedWriteAvailabilityRuntime{
		snapshot: receiptSnapshotForAvailabilityTest(receipt, pendingPublishedWriteAvailabilityStatusForTest(receipt)),
		waitResult: receiptWaitResultForAvailabilityTest(
			receipt,
			availablePublishedWriteAvailabilityStatusForTest(receipt),
		),
	}

	observation, err := waitForPublishedWriteAvailability(
		context.Background(),
		runtime,
		receipt,
		request,
		"task write replicated",
	)
	if err != nil {
		t.Fatalf("wait for exact availability: %v", err)
	}
	if !observation.Status.Available || observation.Status.ConfirmedOtherPeers != 1 {
		t.Fatalf("availability observation=%+v, want one confirmed peer", observation)
	}
	if observation.Receipt.EventID != receipt.EventID || observation.Receipt.PublishedRootHash != receipt.PublishedRootHash ||
		observation.Status.Receipt.EventID != receipt.EventID || observation.Status.Receipt.PublishedRootHash != receipt.PublishedRootHash {
		t.Fatalf("success lost exact receipt identity: %+v", observation)
	}
	if !reflect.DeepEqual(runtime.trace, []string{"status", "wait"}) {
		t.Fatalf("runtime trace=%v, want passive status then wait", runtime.trace)
	}
	for _, called := range runtime.requests {
		if !reflect.DeepEqual(called, request) {
			t.Fatalf("availability request changed between observations: got=%+v want=%+v", called, request)
		}
	}
}

func TestWaitForPublishedWriteAvailabilityTimeoutPreservesLatestExactStatus(t *testing.T) {
	receipt := eventReceiptForTest()
	request, err := ReceiptTrackingRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil {
		t.Fatal(err)
	}
	latest := pendingPublishedWriteAvailabilityStatusForTest(receipt)
	latest.Reason = "insufficient authenticated other-peer receipt evidence"
	runtime := &scriptedPublishedWriteAvailabilityRuntime{snapshot: receiptSnapshotForAvailabilityTest(receipt, latest)}
	runtime.wait = func(ctx context.Context, request swarmionapp.ReceiptWaitRequest) (swarmionapp.ReceiptWaitResult, error) {
		<-ctx.Done()
		result := receiptWaitResultForAvailabilityTest(receipt, latest)
		return result, &swarmionapp.ReceiptPendingError{Result: result, Cause: ctx.Err()}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	observation, err := waitForPublishedWriteAvailability(ctx, runtime, receipt, request, "bounded task replication")
	var pending *PublishedWriteAvailabilityPendingError
	if !errors.As(err, &pending) || pending == nil || !errors.Is(err, ErrPublishedWriteAvailabilityPending) {
		t.Fatalf("timeout error=%v, want typed availability pending", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout error=%v, want caller deadline cause", err)
	}
	if observation.Status.Reason != latest.Reason || pending.Observation.Status.Reason != latest.Reason ||
		observation.Receipt.EventID != receipt.EventID || pending.Observation.Receipt.PublishedRootHash != receipt.PublishedRootHash {
		t.Fatalf("timeout lost latest exact observation: returned=%+v pending=%+v", observation, pending.Observation)
	}
	if observation.Status.ReasonCode != swarmionapp.OtherPeerRetentionReasonInsufficientOtherPeerReceipts ||
		!reflect.DeepEqual(observation.Status.EligiblePeerIDs, []string{"peer-b"}) {
		t.Fatalf("timeout lost stable topology status: %+v", observation.Status)
	}
	if observation.Status.Available {
		t.Fatalf("pending availability became success: observation=%+v error=%v", observation, err)
	}
}

func TestWaitForPublishedWriteAvailabilityPreservesPromptNoEligibleStatus(t *testing.T) {
	receipt := eventReceiptForTest()
	request, err := ReceiptTrackingRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil {
		t.Fatal(err)
	}
	status := noCurrentEligiblePeersAvailabilityStatusForTest(receipt)
	runtime := &scriptedPublishedWriteAvailabilityRuntime{snapshot: receiptSnapshotForAvailabilityTest(receipt, status)}
	runtime.wait = func(context.Context, swarmionapp.ReceiptWaitRequest) (swarmionapp.ReceiptWaitResult, error) {
		result := receiptWaitResultForAvailabilityTest(receipt, status)
		return result, &swarmionapp.ReceiptPendingError{Result: result}
	}

	observation, err := waitForPublishedWriteAvailability(
		context.Background(),
		runtime,
		receipt,
		request,
		"single-peer write",
	)
	if !errors.Is(err, ErrPublishedWriteAvailabilityPending) ||
		!errors.Is(err, swarmionapp.ErrReceiptPending) ||
		errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("no-eligible error=%v, want prompt typed pending", err)
	}
	var upstream *swarmionapp.ReceiptPendingError
	if !errors.As(err, &upstream) || !reflect.DeepEqual(upstream.Result.Snapshot.OtherPeerRetention, &status) ||
		!reflect.DeepEqual(observation.Status, status) {
		t.Fatalf("no-eligible observation=%+v error=%v, want exact upstream status", observation, err)
	}
}

func TestPublishedWriteAvailabilityRejectsMismatchedReturnedIdentity(t *testing.T) {
	receipt := eventReceiptForTest()
	request, err := ReceiptTrackingRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil {
		t.Fatal(err)
	}
	mismatch := pendingPublishedWriteAvailabilityStatusForTest(receipt)
	mismatch.Receipt.EventID = swarmionprotocol.NewEventID("different availability event").String()
	runtime := &scriptedPublishedWriteAvailabilityRuntime{snapshot: receiptSnapshotForAvailabilityTest(receipt, mismatch)}

	observation, err := observePublishedWriteAvailability(context.Background(), runtime, receipt, request)
	if !errors.Is(err, errSwarmionPublishedWriteIncomplete) {
		t.Fatalf("mismatched status error=%v, want exact-identity failure", err)
	}
	if observation.Receipt.EventID != receipt.EventID || observation.Receipt.PublishedRootHash != receipt.PublishedRootHash {
		t.Fatalf("identity failure lost requested receipt: %+v", observation)
	}
	if !reflect.DeepEqual(runtime.trace, []string{"status"}) {
		t.Fatalf("identity failure trace=%v, want one passive status read", runtime.trace)
	}
}

func TestPublishedWriteAvailabilityRejectsInconsistentTopologyStatus(t *testing.T) {
	receipt := eventReceiptForTest()
	unscoped, err := ReceiptTrackingRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil {
		t.Fatal(err)
	}
	explicit, err := ReceiptTrackingRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{
		PeerIDs: []string{"peer-b"},
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		request swarmionapp.ReceiptTrackingRequest
		mutate  func(*swarmionapp.OtherPeerRetentionStatus)
	}{
		{
			name:    "wrong candidate scope",
			request: unscoped,
			mutate: func(status *swarmionapp.OtherPeerRetentionStatus) {
				status.CandidateScope = swarmionapp.OtherPeerRetentionCandidateScopeExplicitPeerIDs
			},
		},
		{
			name:    "noncanonical eligible peers",
			request: unscoped,
			mutate: func(status *swarmionapp.OtherPeerRetentionStatus) {
				status.EligiblePeerIDs = []string{"peer-c", "peer-b"}
			},
		},
		{
			name:    "explicit candidates changed",
			request: explicit,
			mutate: func(status *swarmionapp.OtherPeerRetentionStatus) {
				status.CandidateScope = swarmionapp.OtherPeerRetentionCandidateScopeExplicitPeerIDs
				status.EligiblePeerIDs = []string{"peer-c"}
			},
		},
		{
			name:    "no current peers with evidence",
			request: unscoped,
			mutate: func(status *swarmionapp.OtherPeerRetentionStatus) {
				*status = noCurrentEligiblePeersAvailabilityStatusForTest(receipt)
				status.Peers = []swarmionapp.OtherPeerRetentionPeerStatus{{PeerID: "historical-peer"}}
			},
		},
		{
			name:    "wrong stable reason code",
			request: unscoped,
			mutate: func(status *swarmionapp.OtherPeerRetentionStatus) {
				status.ReasonCode = swarmionapp.OtherPeerRetentionReasonReceiptNotLocallyLive
			},
		},
		{
			name:    "evidence peer is not eligible",
			request: unscoped,
			mutate: func(status *swarmionapp.OtherPeerRetentionStatus) {
				status.Peers = []swarmionapp.OtherPeerRetentionPeerStatus{{PeerID: "peer-c"}}
			},
		},
		{
			name:    "evidence peer is duplicated",
			request: unscoped,
			mutate: func(status *swarmionapp.OtherPeerRetentionStatus) {
				status.Peers = []swarmionapp.OtherPeerRetentionPeerStatus{{PeerID: "peer-b"}, {PeerID: "peer-b"}}
			},
		},
		{
			name:    "confirmed count lacks retained evidence",
			request: unscoped,
			mutate: func(status *swarmionapp.OtherPeerRetentionStatus) {
				status.ConfirmedOtherPeers = 1
				status.Available = true
				status.ReasonCode = swarmionapp.OtherPeerRetentionReasonNone
				status.Reason = ""
				status.Peers = []swarmionapp.OtherPeerRetentionPeerStatus{{PeerID: "peer-b"}}
			},
		},
		{
			name:    "not locally live status contains evidence",
			request: unscoped,
			mutate: func(status *swarmionapp.OtherPeerRetentionStatus) {
				status.Known = false
				status.ReasonCode = swarmionapp.OtherPeerRetentionReasonReceiptNotLocallyLive
				status.Reason = "the exact receipt is not currently live on this runtime"
				status.Peers = []swarmionapp.OtherPeerRetentionPeerStatus{{PeerID: "peer-b"}}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status := pendingPublishedWriteAvailabilityStatusForTest(receipt)
			test.mutate(&status)
			snapshot := receiptSnapshotForAvailabilityTest(receipt, status)
			if _, err := publishedWriteAvailabilityObservation(receipt, test.request, snapshot); !errors.Is(err, errSwarmionPublishedWriteIncomplete) {
				t.Fatalf("inconsistent status=%+v error=%v, want validation failure", status, err)
			}
		})
	}
}

func TestWaitForPublishedWriteAvailabilityRejectsContradictoryLifecycleResult(t *testing.T) {
	receipt := eventReceiptForTest()
	request, err := ReceiptTrackingRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil {
		t.Fatal(err)
	}
	initial := pendingPublishedWriteAvailabilityStatusForTest(receipt)
	available := availablePublishedWriteAvailabilityStatusForTest(receipt)

	tests := []struct {
		name   string
		mutate func(*swarmionapp.ReceiptWaitResult)
	}{
		{
			name: "wrong condition",
			mutate: func(result *swarmionapp.ReceiptWaitResult) {
				result.Condition = swarmionapp.ReceiptConditionCheckpointed
			},
		},
		{
			name: "satisfaction contradicts retention",
			mutate: func(result *swarmionapp.ReceiptWaitResult) {
				result.Satisfied = false
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := receiptWaitResultForAvailabilityTest(receipt, available)
			test.mutate(&result)
			runtime := &scriptedPublishedWriteAvailabilityRuntime{
				snapshot:   receiptSnapshotForAvailabilityTest(receipt, initial),
				waitResult: result,
			}

			observation, err := waitForPublishedWriteAvailability(
				context.Background(),
				runtime,
				receipt,
				request,
				"contradictory lifecycle result",
			)
			if !errors.Is(err, errSwarmionPublishedWriteIncomplete) {
				t.Fatalf("contradictory wait error=%v, want fail-closed lifecycle validation", err)
			}
			if observation.Status.Available || observation.Status.ReasonCode != initial.ReasonCode {
				t.Fatalf("contradictory wait replaced latest safe snapshot: %+v", observation)
			}
		})
	}
}

func TestWaitForPublishedWriteAvailabilityRetainsAvailableSnapshotOnRuntimeClose(t *testing.T) {
	receipt := eventReceiptForTest()
	request, err := ReceiptTrackingRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil {
		t.Fatal(err)
	}
	initial := pendingPublishedWriteAvailabilityStatusForTest(receipt)
	available := availablePublishedWriteAvailabilityStatusForTest(receipt)
	result := receiptWaitResultForAvailabilityTest(receipt, available)
	result.Satisfied = false
	runtime := &scriptedPublishedWriteAvailabilityRuntime{
		snapshot:   receiptSnapshotForAvailabilityTest(receipt, initial),
		waitResult: result,
		waitErr:    &swarmionapp.DatabaseRuntimeClosedError{},
	}

	observation, err := waitForPublishedWriteAvailability(
		context.Background(),
		runtime,
		receipt,
		request,
		"runtime-close terminal handoff",
	)
	if !errors.Is(err, ErrPublishedWriteAvailabilityPending) ||
		!errors.Is(err, swarmionapp.ErrDatabaseRuntimeClosed) {
		t.Fatalf("terminal availability error=%v, want pending plus typed runtime closure", err)
	}
	if !observation.Status.Available || observation.Status.ConfirmedOtherPeers < 1 {
		t.Fatalf("terminal availability lost its latest validated evidence: %+v", observation)
	}
}

func TestConfirmPublishedWriteAvailabilityKeepsRuntimeCloseTerminalAfterAvailableSnapshot(t *testing.T) {
	receipt := eventReceiptForTest()
	request, err := ReceiptTrackingRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil {
		t.Fatal(err)
	}
	result := receiptWaitResultForAvailabilityTest(receipt, availablePublishedWriteAvailabilityStatusForTest(receipt))
	result.Satisfied = false
	runtime := &scriptedPublishedWriteAvailabilityRuntime{
		snapshot:   receiptSnapshotForAvailabilityTest(receipt, pendingPublishedWriteAvailabilityStatusForTest(receipt)),
		waitResult: result,
		waitErr:    &swarmionapp.DatabaseRuntimeClosedError{},
	}

	confirmation := confirmPublishedWriteAvailability(
		context.Background(),
		runtime,
		receipt,
		request,
		"runtime-close terminal handoff",
	)
	if confirmation.Stage != PublishedWriteConfirmationLocalAccepted ||
		!confirmation.AvailabilityPending || !confirmation.Availability.Available ||
		!strings.Contains(confirmation.AvailabilityError, swarmionapp.ErrDatabaseRuntimeClosed.Error()) {
		t.Fatalf("terminal confirmation=%+v, want retained availability with local-only pending closure", confirmation)
	}
	if confirmation.OtherPeerAvailable() {
		t.Fatalf("terminal confirmation was promoted to success: %+v", confirmation)
	}
}

func TestPublishedWriteAvailabilityWaitIsCallerBoundedAndNeverGrantsReplay(t *testing.T) {
	receipt := eventReceiptForTest()
	receipt.Committed = false
	receipt.OutcomeUncertain = true
	request, err := ReceiptTrackingRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil {
		t.Fatal(err)
	}
	runtime := &scriptedPublishedWriteAvailabilityRuntime{
		snapshot: receiptSnapshotForAvailabilityTest(receipt, pendingPublishedWriteAvailabilityStatusForTest(receipt)),
	}
	var remaining time.Duration
	runtime.wait = func(ctx context.Context, candidate swarmionapp.ReceiptWaitRequest) (swarmionapp.ReceiptWaitResult, error) {
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("availability wait received an unbounded context")
		}
		remaining = time.Until(deadline)
		if candidate.Tracking.Receipt != request.Receipt || candidate.Condition != swarmionapp.ReceiptConditionOtherPeerRetained {
			t.Fatalf("wait changed uncertain receipt identity or condition: got=%+v want=%+v", candidate, request)
		}
		return swarmionapp.ReceiptWaitResult{}, errors.New("injected passive wait interruption")
	}
	boundedCtx, cancel := boundedPublishedWriteAvailabilityContext(context.Background())
	defer cancel()

	observation, err := waitForPublishedWriteAvailability(boundedCtx, runtime, receipt, request, "uncertain receipt")
	if err == nil || !errors.Is(err, ErrPublishedWriteAvailabilityPending) {
		t.Fatalf("interrupted wait error=%v, want typed pending", err)
	}
	if remaining <= 8*time.Second || remaining > publishedWriteAvailabilityTimeout {
		t.Fatalf("wait context remaining=%s, want caller-bounded ten-second cap", remaining)
	}
	if observation.Receipt.Committed || !observation.Receipt.OutcomeUncertain {
		t.Fatalf("passive wait changed uncertain receipt: observation=%+v error=%v", observation, err)
	}
	if !reflect.DeepEqual(runtime.trace, []string{"status", "wait"}) {
		t.Fatalf("runtime trace=%v, want observation only", runtime.trace)
	}
}

func TestConfirmPublishedWriteAvailabilityReturnsImmediatelyWithoutCurrentEligiblePeers(t *testing.T) {
	receipt := eventReceiptForTest()
	request, err := ReceiptTrackingRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil {
		t.Fatal(err)
	}
	runtime := &scriptedPublishedWriteAvailabilityRuntime{
		// Swarmion has already separated retained historical evidence from the
		// current unscoped candidate set. Peers is evidence only and remains empty.
		snapshot: receiptSnapshotForAvailabilityTest(receipt, noCurrentEligiblePeersAvailabilityStatusForTest(receipt)),
		wait: func(context.Context, swarmionapp.ReceiptWaitRequest) (swarmionapp.ReceiptWaitResult, error) {
			t.Fatal("single-peer confirmation must not wait")
			return swarmionapp.ReceiptWaitResult{}, nil
		},
	}

	confirmation := confirmPublishedWriteAvailability(context.Background(), runtime, receipt, request, "bootstrap write")
	if confirmation.Stage != PublishedWriteConfirmationLocalAccepted || !confirmation.AvailabilityPending {
		t.Fatalf("confirmation=%+v, want prompt local acceptance", confirmation)
	}
	if confirmation.AvailabilityError != "" {
		t.Fatalf("single-peer confirmation reported an error: %+v", confirmation)
	}
	if !reflect.DeepEqual(runtime.trace, []string{"status"}) {
		t.Fatalf("single-peer trace=%v, want one scoped availability read", runtime.trace)
	}
	if confirmation.Availability.RequiredOtherPeers != 1 ||
		!confirmation.Availability.NoCurrentEligiblePeers ||
		confirmation.Availability.ReasonCode != swarmionapp.OtherPeerRetentionReasonNoCurrentEligiblePeers {
		t.Fatalf("single-peer confirmation lost requested threshold: %+v", confirmation)
	}
	if len(runtime.requests) != 1 || runtime.requests[0].OtherPeerRetention == nil ||
		len(runtime.requests[0].OtherPeerRetention.PeerIDs) != 0 {
		t.Fatalf("ordinary single-peer request was not unscoped: %+v", runtime.requests)
	}
}

func TestConfirmPublishedWriteAvailabilityWaitsForObservedOtherPeer(t *testing.T) {
	receipt := eventReceiptForTest()
	request, err := ReceiptTrackingRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil {
		t.Fatal(err)
	}
	pending := unknownPublishedWriteAvailabilityStatusForTest(receipt)
	// Swarmion can report no receipt peers while the just-authored event/root is
	// still entering its local availability index. Current topology still proves
	// that waiting for peer-b is meaningful, so this transient must not take the
	// single-peer early return.
	runtime := &scriptedPublishedWriteAvailabilityRuntime{
		snapshot: receiptSnapshotForAvailabilityTest(receipt, pending),
		waitResult: receiptWaitResultForAvailabilityTest(
			receipt,
			availablePublishedWriteAvailabilityStatusForTest(receipt),
		),
	}

	confirmation := confirmPublishedWriteAvailability(context.Background(), runtime, receipt, request, "ordinary task write")
	if confirmation.Stage != PublishedWriteConfirmationOtherPeerAvailable ||
		!confirmation.OtherPeerAvailable() || confirmation.AvailabilityPending || confirmation.AvailabilityError != "" {
		t.Fatalf("confirmation=%+v, want one-other-peer availability", confirmation)
	}
	if !reflect.DeepEqual(runtime.trace, []string{"status", "wait"}) {
		t.Fatalf("multi-peer trace=%v, want observe then event-driven wait", runtime.trace)
	}
	for _, called := range runtime.requests {
		if called.OtherPeerRetention == nil || len(called.OtherPeerRetention.PeerIDs) != 0 {
			t.Fatalf("ordinary multi-peer request must remain unscoped: %+v", called)
		}
	}
}

func TestConfirmPublishedWriteAvailabilityPreservesAcceptedWriteOnPendingWait(t *testing.T) {
	receipt := eventReceiptForTest()
	request, err := ReceiptTrackingRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil {
		t.Fatal(err)
	}
	pending := pendingPublishedWriteAvailabilityStatusForTest(receipt)
	runtime := &scriptedPublishedWriteAvailabilityRuntime{
		snapshot: receiptSnapshotForAvailabilityTest(receipt, pending),
		waitErr:  context.DeadlineExceeded,
	}

	confirmation := confirmPublishedWriteAvailability(context.Background(), runtime, receipt, request, "ordinary task write")
	if confirmation.Stage != PublishedWriteConfirmationLocalAccepted || !confirmation.AvailabilityPending ||
		confirmation.AvailabilityError == "" || confirmation.Receipt.EventID != receipt.EventID {
		t.Fatalf("confirmation=%+v, want accepted non-replayable pending stage", confirmation)
	}
}

func TestConfirmPublishedWriteAvailabilityPreservesPromptDepartureStatus(t *testing.T) {
	receipt := eventReceiptForTest()
	request, err := ReceiptTrackingRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil {
		t.Fatal(err)
	}
	initial := pendingPublishedWriteAvailabilityStatusForTest(receipt)
	departed := noCurrentEligiblePeersAvailabilityStatusForTest(receipt)
	runtime := &scriptedPublishedWriteAvailabilityRuntime{
		snapshot: receiptSnapshotForAvailabilityTest(receipt, initial),
		wait: func(context.Context, swarmionapp.ReceiptWaitRequest) (swarmionapp.ReceiptWaitResult, error) {
			result := receiptWaitResultForAvailabilityTest(receipt, departed)
			return result, &swarmionapp.ReceiptPendingError{Result: result}
		},
	}

	confirmation := confirmPublishedWriteAvailability(context.Background(), runtime, receipt, request, "peer departed")
	if confirmation.Stage != PublishedWriteConfirmationLocalAccepted || !confirmation.AvailabilityPending ||
		confirmation.AvailabilityError != "" || !confirmation.Availability.NoCurrentEligiblePeers ||
		confirmation.Availability.ReasonCode != swarmionapp.OtherPeerRetentionReasonNoCurrentEligiblePeers {
		t.Fatalf("departure confirmation=%+v, want prompt weak local acceptance", confirmation)
	}
	if !reflect.DeepEqual(runtime.trace, []string{"status", "wait"}) {
		t.Fatalf("departure trace=%v, want read then event-driven wait", runtime.trace)
	}
	for _, called := range runtime.requests {
		if called.OtherPeerRetention == nil || len(called.OtherPeerRetention.PeerIDs) != 0 {
			t.Fatalf("departure request unexpectedly became fixed-target: %+v", called)
		}
	}
}

func TestConfirmPublishedWriteAvailabilityDoesNotHideRuntimeClosureBehindNoCurrentStatus(t *testing.T) {
	receipt := eventReceiptForTest()
	request, err := ReceiptTrackingRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil {
		t.Fatal(err)
	}
	initial := pendingPublishedWriteAvailabilityStatusForTest(receipt)
	departed := noCurrentEligiblePeersAvailabilityStatusForTest(receipt)
	runtime := &scriptedPublishedWriteAvailabilityRuntime{
		snapshot: receiptSnapshotForAvailabilityTest(receipt, initial),
		wait: func(context.Context, swarmionapp.ReceiptWaitRequest) (swarmionapp.ReceiptWaitResult, error) {
			result := receiptWaitResultForAvailabilityTest(receipt, departed)
			return result, &swarmionapp.DatabaseRuntimeClosedError{
				Cause: &swarmionapp.ReceiptPendingError{Result: result},
			}
		},
	}

	confirmation := confirmPublishedWriteAvailability(context.Background(), runtime, receipt, request, "runtime closed")
	if confirmation.Stage != PublishedWriteConfirmationLocalAccepted || !confirmation.AvailabilityPending ||
		!confirmation.Availability.NoCurrentEligiblePeers ||
		!strings.Contains(confirmation.AvailabilityError, swarmionapp.ErrDatabaseRuntimeClosed.Error()) {
		t.Fatalf("runtime-close confirmation=%+v, want retained weak status plus terminal diagnostic", confirmation)
	}
}

func TestConfirmPublishedWriteAvailabilityPreservesStatusFailureWhenRuntimeStatusIsUnready(t *testing.T) {
	receipt := eventReceiptForTest()
	request, err := ReceiptTrackingRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil {
		t.Fatal(err)
	}
	statusErr := errors.New("receipt status is unavailable")
	runtime := &scriptedPublishedWriteAvailabilityRuntime{observeErr: statusErr}

	confirmation := confirmPublishedWriteAvailability(context.Background(), runtime, receipt, request, "runtime starting")
	if confirmation.Stage != PublishedWriteConfirmationLocalAccepted || !confirmation.AvailabilityPending ||
		!strings.Contains(confirmation.AvailabilityError, statusErr.Error()) {
		t.Fatalf("confirmation=%+v, want prompt accepted status diagnostic", confirmation)
	}
	if !reflect.DeepEqual(runtime.trace, []string{"status"}) {
		t.Fatalf("unready runtime trace=%v, want receipt status diagnostic", runtime.trace)
	}
}

func TestConfirmPublishedWriteAvailabilityNoChangeDoesNotRequireRuntime(t *testing.T) {
	confirmation := (*DB)(nil).ConfirmPublishedWriteAvailability(context.Background(), PublishedWriteReceipt{}, "no-op")
	if confirmation.Stage != PublishedWriteConfirmationNoChange || confirmation.AvailabilityPending {
		t.Fatalf("no-op confirmation=%+v", confirmation)
	}
}

func TestConfirmPublishedWriteAvailabilityDoesNotClaimUnresolvedReceiptAccepted(t *testing.T) {
	receipt := eventReceiptForTest()
	receipt.Committed = false
	receipt.OutcomeUncertain = true

	confirmation := (*DB)(nil).ConfirmPublishedWriteAvailability(context.Background(), receipt, "unresolved write")
	if confirmation.Stage != "" || !confirmation.AvailabilityPending ||
		confirmation.AvailabilityError != ErrPublishedWriteConfirmationUnresolved.Error() {
		t.Fatalf("unresolved confirmation=%+v, want no acceptance claim", confirmation)
	}
}

func TestOrdinaryAvailabilityBoundaryPreservesExactFailureWithoutAvailability(t *testing.T) {
	receipt := eventReceiptForTest()
	receipt.Committed = false
	receipt.OutcomeUncertain = true
	cause := errors.New("injected execution failure after exact receipt")

	publicationCalls := 0
	availabilityCalls := 0
	confirmation, err := confirmOrdinaryPublishedWrite(
		func() (PublishedWriteReceipt, error) {
			publicationCalls++
			return receipt, cause
		},
		func(PublishedWriteReceipt) PublishedWriteConfirmation {
			availabilityCalls++
			return PublishedWriteConfirmation{Stage: PublishedWriteConfirmationOtherPeerAvailable}
		},
	)
	if publicationCalls != 1 || availabilityCalls != 0 {
		t.Fatalf(
			"boundary calls publication=%d availability=%d, want one publication and no availability observation",
			publicationCalls,
			availabilityCalls,
		)
	}
	if confirmation.Stage != "" || confirmation.OtherPeerAvailable() || confirmation.AvailabilityPending {
		t.Fatalf("unresolved exact confirmation was falsely classified as accepted: %+v", confirmation)
	}
	if !reflect.DeepEqual(confirmation.Receipt, receipt) {
		t.Fatalf("unresolved confirmation receipt=%+v, want %+v", confirmation.Receipt, receipt)
	}
	if !errors.Is(err, ErrPublishedWriteConfirmationUnresolved) {
		t.Fatalf("boundary error=%v, want unresolved sentinel", err)
	}
	var unresolved *PublishedWriteConfirmationUnresolvedError
	if !errors.As(err, &unresolved) || unresolved == nil {
		t.Fatalf("boundary error=%v, want typed unresolved error", err)
	}
	if !errors.Is(unresolved.Cause, cause) || !reflect.DeepEqual(unresolved.Confirmation, confirmation) {
		t.Fatalf("typed unresolved error=%+v, want exact confirmation and original cause", unresolved)
	}
	if errors.Is(err, cause) {
		t.Fatalf("exact unresolved result exposed its diagnostic cause as control flow: %v", err)
	}
	extracted, ok := PublishedWriteConfirmationFromError(fmt.Errorf("caller context: %w", err))
	if !ok || !reflect.DeepEqual(extracted, confirmation) {
		t.Fatalf("wrapped error extraction=(%+v, %t), want exact unresolved confirmation", extracted, ok)
	}
}

func TestOrdinaryAvailabilityBoundaryLeavesNonExactPublicationErrorUnchanged(t *testing.T) {
	cause := errors.New("injected pre-publication failure")
	publicationCalls := 0
	availabilityCalls := 0
	confirmation, err := confirmOrdinaryPublishedWrite(
		func() (PublishedWriteReceipt, error) {
			publicationCalls++
			return PublishedWriteReceipt{}, cause
		},
		func(PublishedWriteReceipt) PublishedWriteConfirmation {
			availabilityCalls++
			return PublishedWriteConfirmation{Stage: PublishedWriteConfirmationOtherPeerAvailable}
		},
	)
	if !errors.Is(err, cause) || errors.Is(err, ErrPublishedWriteConfirmationUnresolved) {
		t.Fatalf("non-exact error=%v, want original error unchanged", err)
	}
	if publicationCalls != 1 || availabilityCalls != 0 {
		t.Fatalf("non-exact calls publication=%d availability=%d, want one publication and no availability", publicationCalls, availabilityCalls)
	}
	if confirmation.Stage != "" || confirmation.Receipt.HasExactEventIdentity() {
		t.Fatalf("non-exact confirmation=%+v, want unclassified empty receipt", confirmation)
	}
	if _, ok := PublishedWriteConfirmationFromError(err); ok {
		t.Fatalf("non-exact error unexpectedly exposed a confirmation: %v", err)
	}
}

func pendingPublishedWriteAvailabilityStatusForTest(receipt PublishedWriteReceipt) swarmionapp.OtherPeerRetentionStatus {
	evaluatedAt := time.Unix(1_725_000_000, 0).UTC()
	return swarmionapp.OtherPeerRetentionStatus{
		Receipt: swarmionapp.EventReceipt{
			EventID:           receipt.EventID,
			PublishedRootHash: receipt.PublishedRootHash,
		},
		EvaluatedAt:        evaluatedAt,
		AuthorPeerID:       "peer-a",
		AuthorSeq:          1,
		Known:              true,
		LocalRootReady:     true,
		LocalReadyThrough:  1,
		RequiredOtherPeers: 1,
		CandidateScope:     swarmionapp.OtherPeerRetentionCandidateScopeCurrentLogicalPeers,
		EligiblePeerIDs:    []string{"peer-b"},
		Peers: []swarmionapp.OtherPeerRetentionPeerStatus{{
			PeerID:     "peer-b",
			ReasonCode: swarmionapp.OtherPeerRetentionReasonPeerNotObserved,
			Reason:     "no authenticated receipt observation exists for this peer",
		}},
		ReasonCode: swarmionapp.OtherPeerRetentionReasonInsufficientOtherPeerReceipts,
		Reason:     "insufficient authenticated other-peer receipt evidence",
	}
}

func unknownPublishedWriteAvailabilityStatusForTest(receipt PublishedWriteReceipt) swarmionapp.OtherPeerRetentionStatus {
	status := pendingPublishedWriteAvailabilityStatusForTest(receipt)
	status.AuthorPeerID = ""
	status.AuthorSeq = 0
	status.Known = false
	status.LocalRootReady = false
	status.LocalReadyThrough = 0
	status.Peers = nil
	status.ReasonCode = swarmionapp.OtherPeerRetentionReasonReceiptNotLocallyLive
	status.Reason = "the exact receipt is not currently live on this runtime"
	return status
}

func noCurrentEligiblePeersAvailabilityStatusForTest(receipt PublishedWriteReceipt) swarmionapp.OtherPeerRetentionStatus {
	status := pendingPublishedWriteAvailabilityStatusForTest(receipt)
	status.EligiblePeerIDs = []string{}
	status.Peers = nil
	status.NoCurrentEligiblePeers = true
	status.ReasonCode = swarmionapp.OtherPeerRetentionReasonNoCurrentEligiblePeers
	status.Reason = "no current logical peers are eligible for other-peer availability"
	return status
}

func availablePublishedWriteAvailabilityStatusForTest(receipt PublishedWriteReceipt) swarmionapp.OtherPeerRetentionStatus {
	status := pendingPublishedWriteAvailabilityStatusForTest(receipt)
	status.ConfirmedOtherPeers = 1
	status.Available = true
	status.ReasonCode = swarmionapp.OtherPeerRetentionReasonNone
	status.Reason = ""
	status.Peers = []swarmionapp.OtherPeerRetentionPeerStatus{{
		PeerID:             "peer-b",
		Observed:           true,
		ObservedAt:         status.EvaluatedAt.Add(-time.Second),
		Fresh:              true,
		RootReady:          true,
		PrefixComparable:   true,
		PrefixAgrees:       true,
		Retained:           true,
		ReadyThrough:       1,
		PrefixProofThrough: 1,
	}}
	return status
}

func receiptSnapshotForAvailabilityTest(
	_ PublishedWriteReceipt,
	status swarmionapp.OtherPeerRetentionStatus,
) swarmionapp.ReceiptSnapshot {
	return swarmionapp.ReceiptSnapshot{
		Receipt: status.Receipt,
		Event: swarmionapp.ReceiptStatus{
			EventID:                   status.Receipt.EventID,
			ExpectedPublishedRootHash: status.Receipt.PublishedRootHash,
			Known:                     true,
			ContentCoverage:           swarmionapp.BranchEventContentCoveragePending,
		},
		OtherPeerRetention: &status,
		ObservedAt:         status.EvaluatedAt,
	}
}

func receiptWaitResultForAvailabilityTest(
	receipt PublishedWriteReceipt,
	status swarmionapp.OtherPeerRetentionStatus,
) swarmionapp.ReceiptWaitResult {
	return swarmionapp.ReceiptWaitResult{
		Snapshot:  receiptSnapshotForAvailabilityTest(receipt, status),
		Satisfied: status.Available,
		Condition: swarmionapp.ReceiptConditionOtherPeerRetained,
	}
}
