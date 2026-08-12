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
	runtimeStatus swarmionapp.Status
	status        swarmionapp.ReceiptAvailabilityStatus
	statusErr     error
	waitStatus    swarmionapp.ReceiptAvailabilityStatus
	waitErr       error
	wait          func(context.Context, swarmionapp.ReceiptAvailabilityRequest) (swarmionapp.ReceiptAvailabilityStatus, error)
	requests      []swarmionapp.ReceiptAvailabilityRequest
	trace         []string
}

func (r *scriptedPublishedWriteAvailabilityRuntime) Status() swarmionapp.Status {
	return r.runtimeStatus
}

func (r *scriptedPublishedWriteAvailabilityRuntime) ReceiptAvailabilityStatus(
	_ context.Context,
	request swarmionapp.ReceiptAvailabilityRequest,
) (swarmionapp.ReceiptAvailabilityStatus, error) {
	r.trace = append(r.trace, "status")
	r.requests = append(r.requests, request)
	return r.status, r.statusErr
}

func (r *scriptedPublishedWriteAvailabilityRuntime) WaitReceiptAvailability(
	ctx context.Context,
	request swarmionapp.ReceiptAvailabilityRequest,
) (swarmionapp.ReceiptAvailabilityStatus, error) {
	r.trace = append(r.trace, "wait")
	r.requests = append(r.requests, request)
	if r.wait != nil {
		return r.wait(ctx, request)
	}
	return r.waitStatus, r.waitErr
}

func TestReceiptAvailabilityRequestFromPublishedWriteReceiptPreservesExactIdentityAndScope(t *testing.T) {
	receipt := eventReceiptForTest()
	receipt.Committed = false
	receipt.OutcomeUncertain = true
	request, err := ReceiptAvailabilityRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{
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
	if request.MinimumOtherPeers != 2 || request.MaxObservationAge != 3*time.Second ||
		!reflect.DeepEqual(request.PeerIDs, []string{"peer-a", "peer-b"}) {
		t.Fatalf("request scope was not normalized: %+v", request)
	}
	if receipt.Committed || !receipt.OutcomeUncertain {
		t.Fatalf("test receipt no longer represents an uncertain accepted outcome: %+v", receipt)
	}

	defaultRequest, err := ReceiptAvailabilityRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil || defaultRequest.MinimumOtherPeers != 1 {
		t.Fatalf("default request=%+v error=%v, want one other peer", defaultRequest, err)
	}

	for _, options := range []PublishedWriteAvailabilityOptions{
		{MinimumOtherPeers: -1},
		{MaxObservationAge: -time.Nanosecond},
		{PeerIDs: []string{"peer-a", " "}},
	} {
		if _, err := ReceiptAvailabilityRequestFromPublishedWriteReceipt(receipt, options); err == nil {
			t.Fatalf("invalid options %+v were accepted", options)
		}
	}
}

func TestWaitForPublishedWriteAvailabilityReturnsExactOtherPeerSuccess(t *testing.T) {
	receipt := eventReceiptForTest()
	request, err := ReceiptAvailabilityRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil {
		t.Fatal(err)
	}
	runtime := &scriptedPublishedWriteAvailabilityRuntime{
		status:     pendingPublishedWriteAvailabilityStatusForTest(receipt),
		waitStatus: availablePublishedWriteAvailabilityStatusForTest(receipt),
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
	request, err := ReceiptAvailabilityRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil {
		t.Fatal(err)
	}
	latest := pendingPublishedWriteAvailabilityStatusForTest(receipt)
	latest.Known = true
	latest.LocalRootReady = true
	latest.Reason = "insufficient_other_peer_receipts"
	runtime := &scriptedPublishedWriteAvailabilityRuntime{status: latest}
	runtime.wait = func(ctx context.Context, _ swarmionapp.ReceiptAvailabilityRequest) (swarmionapp.ReceiptAvailabilityStatus, error) {
		<-ctx.Done()
		// Swarmion's wait deliberately returns a zero status on context expiry.
		// The DB wrapper must preserve its latest passive exact observation.
		return swarmionapp.ReceiptAvailabilityStatus{}, ctx.Err()
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
	if observation.Status.Available || IsRetryablePublishedWriteError(err) {
		t.Fatalf("pending availability became success/replay authority: observation=%+v error=%v", observation, err)
	}
}

func TestPublishedWriteAvailabilityRejectsMismatchedReturnedIdentity(t *testing.T) {
	receipt := eventReceiptForTest()
	request, err := ReceiptAvailabilityRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil {
		t.Fatal(err)
	}
	mismatch := pendingPublishedWriteAvailabilityStatusForTest(receipt)
	mismatch.Receipt.EventID = swarmionprotocol.NewEventID("different availability event").String()
	runtime := &scriptedPublishedWriteAvailabilityRuntime{status: mismatch}

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

func TestPublishedWriteAvailabilityWaitIsCallerBoundedAndNeverGrantsReplay(t *testing.T) {
	receipt := eventReceiptForTest()
	receipt.Committed = false
	receipt.OutcomeUncertain = true
	request, err := ReceiptAvailabilityRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil {
		t.Fatal(err)
	}
	runtime := &scriptedPublishedWriteAvailabilityRuntime{status: pendingPublishedWriteAvailabilityStatusForTest(receipt)}
	var remaining time.Duration
	runtime.wait = func(ctx context.Context, candidate swarmionapp.ReceiptAvailabilityRequest) (swarmionapp.ReceiptAvailabilityStatus, error) {
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("availability wait received an unbounded context")
		}
		remaining = time.Until(deadline)
		if candidate.Receipt != request.Receipt {
			t.Fatalf("wait changed uncertain receipt identity: got=%+v want=%+v", candidate.Receipt, request.Receipt)
		}
		return swarmionapp.ReceiptAvailabilityStatus{}, errors.New("injected passive wait interruption")
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
	if observation.Receipt.Committed || !observation.Receipt.OutcomeUncertain || IsRetryablePublishedWriteError(err) {
		t.Fatalf("passive wait changed uncertain receipt or granted replay: observation=%+v error=%v", observation, err)
	}
	if !reflect.DeepEqual(runtime.trace, []string{"status", "wait"}) {
		t.Fatalf("runtime trace=%v, want observation only", runtime.trace)
	}
}

func TestConfirmPublishedWriteAvailabilityReturnsImmediatelyWithoutObservedOtherPeers(t *testing.T) {
	receipt := eventReceiptForTest()
	request, err := ReceiptAvailabilityRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil {
		t.Fatal(err)
	}
	runtime := &scriptedPublishedWriteAvailabilityRuntime{
		// A routed-only peer is not an eligible logical heartbeat origin for
		// receipt availability and must not force the bounded wait.
		runtimeStatus: swarmionapp.Status{PeerID: "peer-local", RoutedPeers: []string{"peer-routed"}},
		status: func() swarmionapp.ReceiptAvailabilityStatus {
			status := pendingPublishedWriteAvailabilityStatusForTest(receipt)
			// Historical receipt evidence is deliberately stale after the current
			// routed/logical topology has become empty.
			status.Peers = []swarmionapp.ReceiptAvailabilityPeerStatus{{PeerID: "peer-old", Observed: true}}
			return status
		}(),
		wait: func(context.Context, swarmionapp.ReceiptAvailabilityRequest) (swarmionapp.ReceiptAvailabilityStatus, error) {
			t.Fatal("single-peer confirmation must not wait")
			return swarmionapp.ReceiptAvailabilityStatus{}, nil
		},
	}

	confirmation := confirmPublishedWriteAvailability(context.Background(), runtime, receipt, request, "bootstrap write")
	if confirmation.Stage != PublishedWriteConfirmationLocalAccepted || !confirmation.AvailabilityPending {
		t.Fatalf("confirmation=%+v, want prompt local acceptance", confirmation)
	}
	if confirmation.AvailabilityError != "" {
		t.Fatalf("single-peer confirmation reported an error: %+v", confirmation)
	}
	if len(runtime.trace) != 0 {
		t.Fatalf("single-peer trace=%v, want topology-only fast path", runtime.trace)
	}
	if confirmation.Availability.RequiredOtherPeers != 1 {
		t.Fatalf("single-peer confirmation lost requested threshold: %+v", confirmation)
	}
}

func TestConfirmPublishedWriteAvailabilityWaitsForObservedOtherPeer(t *testing.T) {
	receipt := eventReceiptForTest()
	request, err := ReceiptAvailabilityRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil {
		t.Fatal(err)
	}
	pending := pendingPublishedWriteAvailabilityStatusForTest(receipt)
	// Swarmion can report no receipt peers while the just-authored event/root is
	// still entering its local availability index. Current topology still proves
	// that waiting for peer-b is meaningful, so this transient must not take the
	// single-peer early return.
	pending.Known = false
	pending.LocalRootReady = false
	runtime := &scriptedPublishedWriteAvailabilityRuntime{
		runtimeStatus: swarmionapp.Status{PeerID: "peer-local", LogicalPeers: []string{"peer-b"}},
		status:        pending,
		waitStatus:    availablePublishedWriteAvailabilityStatusForTest(receipt),
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
		if !reflect.DeepEqual(called.PeerIDs, []string{"peer-b"}) {
			t.Fatalf("multi-peer request was not scoped to current topology: %+v", called)
		}
	}
}

func TestConfirmPublishedWriteAvailabilityPreservesAcceptedWriteOnPendingWait(t *testing.T) {
	receipt := eventReceiptForTest()
	request, err := ReceiptAvailabilityRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil {
		t.Fatal(err)
	}
	pending := pendingPublishedWriteAvailabilityStatusForTest(receipt)
	pending.Peers = []swarmionapp.ReceiptAvailabilityPeerStatus{{PeerID: "peer-b", Observed: true}}
	runtime := &scriptedPublishedWriteAvailabilityRuntime{
		runtimeStatus: swarmionapp.Status{PeerID: "peer-local", LogicalPeers: []string{"peer-b"}},
		status:        pending,
		waitErr:       context.DeadlineExceeded,
	}

	confirmation := confirmPublishedWriteAvailability(context.Background(), runtime, receipt, request, "ordinary task write")
	if confirmation.Stage != PublishedWriteConfirmationLocalAccepted || !confirmation.AvailabilityPending ||
		confirmation.AvailabilityError == "" || confirmation.Receipt.EventID != receipt.EventID {
		t.Fatalf("confirmation=%+v, want accepted non-replayable pending stage", confirmation)
	}
}

func TestConfirmPublishedWriteAvailabilityPreservesStatusFailureWhenRuntimeStatusIsUnready(t *testing.T) {
	receipt := eventReceiptForTest()
	request, err := ReceiptAvailabilityRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil {
		t.Fatal(err)
	}
	statusErr := errors.New("receipt status is unavailable")
	runtime := &scriptedPublishedWriteAvailabilityRuntime{statusErr: statusErr}

	confirmation := confirmPublishedWriteAvailability(context.Background(), runtime, receipt, request, "runtime starting")
	if confirmation.Stage != PublishedWriteConfirmationLocalAccepted || !confirmation.AvailabilityPending ||
		!strings.Contains(confirmation.AvailabilityError, statusErr.Error()) {
		t.Fatalf("confirmation=%+v, want prompt accepted status diagnostic", confirmation)
	}
	if !reflect.DeepEqual(runtime.trace, []string{"status"}) {
		t.Fatalf("unready runtime trace=%v, want receipt status diagnostic", runtime.trace)
	}
}

func TestCurrentPublishedWriteAvailabilityPeersUsesLogicalPeersAndIgnoresSelf(t *testing.T) {
	peers, ready := currentPublishedWriteAvailabilityPeers(swarmionapp.Status{
		PeerID:       " peer-local ",
		RoutedPeers:  []string{"peer-b", "peer-local", " peer-a "},
		LogicalPeers: []string{"peer-b", "peer-c", " "},
	})
	if !ready || !reflect.DeepEqual(peers, []string{"peer-b", "peer-c"}) {
		t.Fatalf("peers=%v ready=%t, want current logical peers without self", peers, ready)
	}
	if peers, ready := currentPublishedWriteAvailabilityPeers(swarmionapp.Status{}); ready || peers != nil {
		t.Fatalf("zero runtime status peers=%v ready=%t, want unavailable", peers, ready)
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

func TestOrdinaryAvailabilityBoundaryPreservesExactPublicationErrorWithoutAvailabilityOrReplay(t *testing.T) {
	receipt := eventReceiptForTest()
	receipt.Committed = false
	receipt.OutcomeUncertain = true
	cause := fmt.Errorf(
		"injected publication response: %w",
		&swarmionapp.CommitNotAcceptedError{Cause: errors.New("injected not-accepted marker")},
	)
	if !IsRetryablePublishedWriteError(cause) {
		t.Fatalf("test cause=%v must carry a retryable marker", cause)
	}

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
	if errors.Is(err, cause) || IsRetryablePublishedWriteError(err) {
		t.Fatalf("exact unresolved result exposed replay authority: %v", err)
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

func pendingPublishedWriteAvailabilityStatusForTest(receipt PublishedWriteReceipt) swarmionapp.ReceiptAvailabilityStatus {
	return swarmionapp.ReceiptAvailabilityStatus{
		Receipt: swarmionapp.EventReceipt{
			EventID:           receipt.EventID,
			PublishedRootHash: receipt.PublishedRootHash,
		},
		Known:              true,
		LocalRootReady:     true,
		RequiredOtherPeers: 1,
		Reason:             "insufficient_other_peer_receipts",
	}
}

func availablePublishedWriteAvailabilityStatusForTest(receipt PublishedWriteReceipt) swarmionapp.ReceiptAvailabilityStatus {
	status := pendingPublishedWriteAvailabilityStatusForTest(receipt)
	status.ConfirmedOtherPeers = 1
	status.Available = true
	status.Reason = ""
	status.Peers = []swarmionapp.ReceiptAvailabilityPeerStatus{{
		PeerID:   "peer-b",
		Observed: true,
		Fresh:    true,
		Retained: true,
	}}
	return status
}
