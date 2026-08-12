package db

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	swarmionprotocol "github.com/nustiueudinastea/swarmion/protocol"
	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
)

const publishedWriteAvailabilityTimeout = 10 * time.Second

// ErrPublishedWriteAvailabilityPending identifies an exact published event
// whose retention by another peer was not proved before the bounded wait
// ended. It never means that replaying the write is safe.
var ErrPublishedWriteAvailabilityPending = errors.New("swarmion published write availability pending")

// ErrPublishedWriteConfirmationUnresolved identifies an ordinary publication
// that returned both an error and an exact event/root receipt. The write may
// already have been accepted, so callers must retain the carried confirmation
// and resolve that exact receipt instead of replaying the mutation.
var ErrPublishedWriteConfirmationUnresolved = errors.New("published write confirmation unresolved")

// PublishedWriteAvailabilityOptions selects the exact other-peer retention
// boundary for a published write. MinimumOtherPeers defaults to one. PeerIDs
// and MaxObservationAge retain Swarmion's scoped availability semantics.
type PublishedWriteAvailabilityOptions struct {
	MinimumOtherPeers int
	PeerIDs           []string
	MaxObservationAge time.Duration
}

// PublishedWriteAvailabilityObservation preserves both Protos' complete
// receipt and Swarmion's passive availability evidence for its exact
// EventID/published-root pair.
type PublishedWriteAvailabilityObservation struct {
	Receipt PublishedWriteReceipt
	Status  swarmionapp.ReceiptAvailabilityStatus
}

// PublishedWriteConfirmationStage is the strongest post-write boundary that
// Protos observed before returning from an ordinary mutation. Availability is
// deliberately separate from checkpoint application: OtherPeerAvailable does
// not imply AppliedDurably or authorize a checkpoint-based invariant.
type PublishedWriteConfirmationStage string

const (
	PublishedWriteConfirmationNoChange           PublishedWriteConfirmationStage = "no_change"
	PublishedWriteConfirmationLocalAccepted      PublishedWriteConfirmationStage = "local_accepted"
	PublishedWriteConfirmationOtherPeerAvailable PublishedWriteConfirmationStage = "other_peer_available"
)

// PublishedWriteConfirmation preserves the exact accepted receipt even when a
// second-peer proof is not presently available. AvailabilityPending is a
// conservative observation outcome and never grants permission to replay the
// mutation.
type PublishedWriteConfirmation struct {
	Receipt             PublishedWriteReceipt
	Stage               PublishedWriteConfirmationStage
	Availability        swarmionapp.ReceiptAvailabilityStatus
	AvailabilityPending bool
	AvailabilityError   string
}

func (confirmation PublishedWriteConfirmation) OtherPeerAvailable() bool {
	return confirmation.Stage == PublishedWriteConfirmationOtherPeerAvailable && confirmation.Availability.Available
}

// PublishedWriteConfirmationUnresolvedError preserves an exact receipt when
// the publication call itself did not prove local acceptance. Stage is
// intentionally empty: an exact tracking address is not proof of either
// LocalAccepted or OtherPeerAvailable.
//
// Cause is deliberately not unwrapped. Some publication causes prove that a
// write without a receipt is safe to retry; once an exact receipt exists that
// classification must not escape through this boundary and authorize replay.
type PublishedWriteConfirmationUnresolvedError struct {
	Confirmation PublishedWriteConfirmation
	Cause        error
}

func (e *PublishedWriteConfirmationUnresolvedError) Error() string {
	if e == nil {
		return ErrPublishedWriteConfirmationUnresolved.Error()
	}
	message := fmt.Sprintf(
		"%s: event_id=%s published_root=%s",
		ErrPublishedWriteConfirmationUnresolved,
		e.Confirmation.Receipt.EventID,
		e.Confirmation.Receipt.PublishedRootHash,
	)
	if e.Cause != nil {
		message += ": " + e.Cause.Error()
	}
	return message
}

func (e *PublishedWriteConfirmationUnresolvedError) Is(target error) bool {
	return target == ErrPublishedWriteConfirmationUnresolved
}

// PublishedWriteConfirmationFromError returns the exact, unresolved
// confirmation carried by an availability helper failure. It also works when
// a caller has added its own contextual wrapper around the typed error.
func PublishedWriteConfirmationFromError(err error) (PublishedWriteConfirmation, bool) {
	var unresolved *PublishedWriteConfirmationUnresolvedError
	if !errors.As(err, &unresolved) || unresolved == nil {
		return PublishedWriteConfirmation{}, false
	}
	return unresolved.Confirmation, true
}

// PublishedWriteAvailabilityPendingError preserves the latest exact
// observation made by a bounded wait. Callers may resume observation of this
// receipt, but must not republish the mutation.
type PublishedWriteAvailabilityPendingError struct {
	Observation PublishedWriteAvailabilityObservation
	Reason      string
	Cause       error
}

func (e *PublishedWriteAvailabilityPendingError) Error() string {
	if e == nil {
		return ErrPublishedWriteAvailabilityPending.Error()
	}
	status := e.Observation.Status
	message := fmt.Sprintf(
		"%s: reason=%q event_id=%s published_root=%s known=%t local_root_ready=%t required_other_peers=%d confirmed_other_peers=%d candidate_scope=%q eligible_peer_ids=%v no_current_eligible_peers=%t available=%t availability_reason_code=%q availability_reason=%q",
		ErrPublishedWriteAvailabilityPending,
		e.Reason,
		e.Observation.Receipt.EventID,
		e.Observation.Receipt.PublishedRootHash,
		status.Known,
		status.LocalRootReady,
		status.RequiredOtherPeers,
		status.ConfirmedOtherPeers,
		status.CandidateScope,
		status.EligiblePeerIDs,
		status.NoCurrentEligiblePeers,
		status.Available,
		status.ReasonCode,
		status.Reason,
	)
	if e.Cause != nil {
		message += ": " + e.Cause.Error()
	}
	return message
}

func (e *PublishedWriteAvailabilityPendingError) Is(target error) bool {
	return target == ErrPublishedWriteAvailabilityPending
}

func (e *PublishedWriteAvailabilityPendingError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// ReceiptAvailabilityRequestFromPublishedWriteReceipt converts the complete
// Protos receipt into Swarmion's immutable observation address. The receipt
// need not claim a certain commit outcome: an exact uncertain receipt may be
// observed safely, and observation never grants publication replay authority.
func ReceiptAvailabilityRequestFromPublishedWriteReceipt(
	receipt PublishedWriteReceipt,
	options PublishedWriteAvailabilityOptions,
) (swarmionapp.ReceiptAvailabilityRequest, error) {
	eventID, root, err := validateEventReceiptIdentity(receipt.EventID, receipt.PublishedRootHash)
	if err != nil {
		return swarmionapp.ReceiptAvailabilityRequest{}, err
	}
	if options.MinimumOtherPeers < 0 {
		return swarmionapp.ReceiptAvailabilityRequest{}, fmt.Errorf("minimum other peers cannot be negative")
	}
	if options.MaxObservationAge < 0 {
		return swarmionapp.ReceiptAvailabilityRequest{}, fmt.Errorf("maximum observation age cannot be negative")
	}
	minimumOtherPeers := options.MinimumOtherPeers
	if minimumOtherPeers == 0 {
		minimumOtherPeers = 1
	}
	peerIDs, err := normalizePublishedWriteAvailabilityPeerIDs(options.PeerIDs)
	if err != nil {
		return swarmionapp.ReceiptAvailabilityRequest{}, err
	}
	return swarmionapp.ReceiptAvailabilityRequest{
		Receipt: swarmionapp.EventReceipt{
			EventID:           eventID.String(),
			PublishedRootHash: root.String(),
		},
		MinimumOtherPeers: minimumOtherPeers,
		PeerIDs:           peerIDs,
		MaxObservationAge: options.MaxObservationAge,
	}, nil
}

func normalizePublishedWriteAvailabilityPeerIDs(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{}, len(raw))
	peerIDs := make([]string, 0, len(raw))
	for _, candidate := range raw {
		peerID := strings.TrimSpace(candidate)
		if peerID == "" {
			return nil, fmt.Errorf("availability peer ids cannot contain an empty peer")
		}
		if _, exists := seen[peerID]; exists {
			continue
		}
		seen[peerID] = struct{}{}
		peerIDs = append(peerIDs, peerID)
	}
	sort.Strings(peerIDs)
	return peerIDs, nil
}

type publishedWriteAvailabilityRuntime interface {
	ReceiptAvailabilityStatus(context.Context, swarmionapp.ReceiptAvailabilityRequest) (swarmionapp.ReceiptAvailabilityStatus, error)
	WaitReceiptAvailability(context.Context, swarmionapp.ReceiptAvailabilityRequest) (swarmionapp.ReceiptAvailabilityStatus, error)
}

var _ publishedWriteAvailabilityRuntime = (*swarmionapp.DatabaseRuntime)(nil)

// ObservePublishedWriteAvailability performs one passive availability read.
// It is suitable for detecting a topology where waiting for another peer is
// currently impossible. It does not publish, fetch, checkpoint, or retry.
func (db *DB) ObservePublishedWriteAvailability(
	ctx context.Context,
	receipt PublishedWriteReceipt,
) (PublishedWriteAvailabilityObservation, error) {
	return db.ObservePublishedWriteAvailabilityWithOptions(ctx, receipt, PublishedWriteAvailabilityOptions{})
}

// ObservePublishedWriteAvailabilityWithOptions is the scoped form of
// ObservePublishedWriteAvailability for an explicit peer/count/freshness
// boundary.
func (db *DB) ObservePublishedWriteAvailabilityWithOptions(
	ctx context.Context,
	receipt PublishedWriteReceipt,
	options PublishedWriteAvailabilityOptions,
) (PublishedWriteAvailabilityObservation, error) {
	request, err := ReceiptAvailabilityRequestFromPublishedWriteReceipt(receipt, options)
	if err != nil {
		return PublishedWriteAvailabilityObservation{Receipt: receipt}, err
	}
	runtime, err := db.publishedWriteAvailabilityRuntime()
	if err != nil {
		return PublishedWriteAvailabilityObservation{Receipt: canonicalPublishedWriteAvailabilityReceipt(receipt, request)}, err
	}
	boundedCtx, cancel := boundedPublishedWriteAvailabilityContext(ctx)
	defer cancel()
	return observePublishedWriteAvailability(boundedCtx, runtime, receipt, request)
}

// WaitForPublishedWriteAvailability waits for one other peer to advertise
// exact live-root retention. The entire passive observation is capped at ten
// seconds and any shorter caller deadline or cancellation wins.
func (db *DB) WaitForPublishedWriteAvailability(
	ctx context.Context,
	receipt PublishedWriteReceipt,
	reason string,
) (PublishedWriteAvailabilityObservation, error) {
	return db.WaitForPublishedWriteAvailabilityWithOptions(ctx, receipt, reason, PublishedWriteAvailabilityOptions{})
}

// WaitForPublishedWriteAvailabilityWithOptions waits for an explicit
// peer/count/freshness boundary. Failure preserves the latest exact status
// observed within the single bounded call and never republishes the write.
func (db *DB) WaitForPublishedWriteAvailabilityWithOptions(
	ctx context.Context,
	receipt PublishedWriteReceipt,
	reason string,
	options PublishedWriteAvailabilityOptions,
) (PublishedWriteAvailabilityObservation, error) {
	request, err := ReceiptAvailabilityRequestFromPublishedWriteReceipt(receipt, options)
	if err != nil {
		return PublishedWriteAvailabilityObservation{Receipt: receipt}, err
	}
	runtime, err := db.publishedWriteAvailabilityRuntime()
	if err != nil {
		return PublishedWriteAvailabilityObservation{Receipt: canonicalPublishedWriteAvailabilityReceipt(receipt, request)}, err
	}
	boundedCtx, cancel := boundedPublishedWriteAvailabilityContext(ctx)
	defer cancel()
	return waitForPublishedWriteAvailability(boundedCtx, runtime, receipt, request, reason)
}

// publishedWriteAvailabilityRuntime returns only Swarmion's database-scoped
// SDK handle. Receipt availability must never unwrap or retain runtime App.
func (db *DB) publishedWriteAvailabilityRuntime() (*swarmionapp.DatabaseRuntime, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	db.mu.Lock()
	runtime := db.runtime
	db.mu.Unlock()
	if runtime == nil {
		return nil, fmt.Errorf("db is not initialized")
	}
	return runtime, nil
}

func boundedPublishedWriteAvailabilityContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(ctx, publishedWriteAvailabilityTimeout)
}

func observePublishedWriteAvailability(
	ctx context.Context,
	runtime publishedWriteAvailabilityRuntime,
	receipt PublishedWriteReceipt,
	request swarmionapp.ReceiptAvailabilityRequest,
) (PublishedWriteAvailabilityObservation, error) {
	observation := PublishedWriteAvailabilityObservation{
		Receipt: canonicalPublishedWriteAvailabilityReceipt(receipt, request),
	}
	if runtime == nil {
		return observation, fmt.Errorf("swarmion receipt availability runtime is not initialized")
	}
	status, err := runtime.ReceiptAvailabilityStatus(ctx, request)
	if err != nil {
		return observation, fmt.Errorf("read swarmion published write availability: %w", err)
	}
	return publishedWriteAvailabilityObservation(observation.Receipt, request, status)
}

func waitForPublishedWriteAvailability(
	ctx context.Context,
	runtime publishedWriteAvailabilityRuntime,
	receipt PublishedWriteReceipt,
	request swarmionapp.ReceiptAvailabilityRequest,
	reason string,
) (PublishedWriteAvailabilityObservation, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "protos published write"
	}
	latest, err := observePublishedWriteAvailability(ctx, runtime, receipt, request)
	if err != nil {
		return latest, err
	}
	if latest.Status.Available {
		return latest, nil
	}

	status, waitErr := runtime.WaitReceiptAvailability(ctx, request)
	if receiptAvailabilityStatusHasIdentity(status) {
		observed, validationErr := publishedWriteAvailabilityObservation(latest.Receipt, request, status)
		if validationErr != nil {
			return latest, validationErr
		}
		latest = observed
	}
	if waitErr == nil && latest.Status.Available {
		return latest, nil
	}
	if waitErr == nil {
		waitErr = fmt.Errorf("receipt availability wait returned before other-peer availability")
	}
	return latest, &PublishedWriteAvailabilityPendingError{
		Observation: latest,
		Reason:      reason,
		Cause:       waitErr,
	}
}

func publishedWriteAvailabilityObservation(
	receipt PublishedWriteReceipt,
	request swarmionapp.ReceiptAvailabilityRequest,
	status swarmionapp.ReceiptAvailabilityStatus,
) (PublishedWriteAvailabilityObservation, error) {
	observation := PublishedWriteAvailabilityObservation{Receipt: receipt, Status: status}
	expectedEventID := swarmionprotocol.ParseEventID(request.Receipt.EventID)
	expectedRoot := swarmionprotocol.ParseRootHash(request.Receipt.PublishedRootHash)
	if swarmionprotocol.ParseEventID(status.Receipt.EventID) != expectedEventID ||
		swarmionprotocol.ParseRootHash(status.Receipt.PublishedRootHash) != expectedRoot {
		return observation, fmt.Errorf(
			"%w: availability status identity mismatch requested=%s/%s returned=%s/%s",
			errSwarmionPublishedWriteIncomplete,
			expectedEventID,
			expectedRoot,
			status.Receipt.EventID,
			status.Receipt.PublishedRootHash,
		)
	}
	if status.RequiredOtherPeers != request.MinimumOtherPeers || status.ConfirmedOtherPeers < 0 {
		return observation, fmt.Errorf(
			"%w: inconsistent availability threshold required=%d want=%d confirmed=%d",
			errSwarmionPublishedWriteIncomplete,
			status.RequiredOtherPeers,
			request.MinimumOtherPeers,
			status.ConfirmedOtherPeers,
		)
	}
	expectedScope := swarmionapp.ReceiptAvailabilityCandidateScopeCurrentLogicalPeers
	if len(request.PeerIDs) > 0 {
		expectedScope = swarmionapp.ReceiptAvailabilityCandidateScopeExplicitPeerIDs
	}
	if status.CandidateScope != expectedScope {
		return observation, fmt.Errorf(
			"%w: inconsistent availability candidate scope returned=%q want=%q",
			errSwarmionPublishedWriteIncomplete,
			status.CandidateScope,
			expectedScope,
		)
	}
	eligiblePeerIDs, err := normalizePublishedWriteAvailabilityPeerIDs(status.EligiblePeerIDs)
	if err != nil || !equalStrings(eligiblePeerIDs, status.EligiblePeerIDs) {
		return observation, fmt.Errorf(
			"%w: availability eligible peer ids are not canonical: %v",
			errSwarmionPublishedWriteIncomplete,
			status.EligiblePeerIDs,
		)
	}
	if expectedScope == swarmionapp.ReceiptAvailabilityCandidateScopeExplicitPeerIDs &&
		!equalStrings(eligiblePeerIDs, request.PeerIDs) {
		return observation, fmt.Errorf(
			"%w: explicit availability candidates returned=%v want=%v",
			errSwarmionPublishedWriteIncomplete,
			eligiblePeerIDs,
			request.PeerIDs,
		)
	}
	expectedNoCurrentEligiblePeers := expectedScope == swarmionapp.ReceiptAvailabilityCandidateScopeCurrentLogicalPeers &&
		len(eligiblePeerIDs) == 0
	if status.NoCurrentEligiblePeers != expectedNoCurrentEligiblePeers {
		return observation, fmt.Errorf(
			"%w: inconsistent no-current-eligible-peers=%t scope=%q eligible=%v",
			errSwarmionPublishedWriteIncomplete,
			status.NoCurrentEligiblePeers,
			status.CandidateScope,
			eligiblePeerIDs,
		)
	}
	if status.ConfirmedOtherPeers > len(eligiblePeerIDs) {
		return observation, fmt.Errorf(
			"%w: confirmed availability peers=%d exceed eligible peers=%d",
			errSwarmionPublishedWriteIncomplete,
			status.ConfirmedOtherPeers,
			len(eligiblePeerIDs),
		)
	}
	eligiblePeers := make(map[string]struct{}, len(eligiblePeerIDs))
	for _, peerID := range eligiblePeerIDs {
		eligiblePeers[peerID] = struct{}{}
	}
	evidencePeers := make(map[string]struct{}, len(status.Peers))
	confirmedFromEvidence := 0
	for _, peer := range status.Peers {
		peerID := strings.TrimSpace(peer.PeerID)
		if peerID == "" || peerID != peer.PeerID {
			return observation, fmt.Errorf(
				"%w: availability evidence contains a noncanonical peer id %q",
				errSwarmionPublishedWriteIncomplete,
				peer.PeerID,
			)
		}
		if _, eligible := eligiblePeers[peerID]; !eligible {
			return observation, fmt.Errorf(
				"%w: availability evidence peer %q is not eligible",
				errSwarmionPublishedWriteIncomplete,
				peerID,
			)
		}
		if _, duplicate := evidencePeers[peerID]; duplicate {
			return observation, fmt.Errorf(
				"%w: availability evidence repeats peer %q",
				errSwarmionPublishedWriteIncomplete,
				peerID,
			)
		}
		evidencePeers[peerID] = struct{}{}
		if peer.Retained && peer.Fresh {
			confirmedFromEvidence++
		}
	}
	if status.ConfirmedOtherPeers != confirmedFromEvidence {
		return observation, fmt.Errorf(
			"%w: confirmed availability peers=%d do not match retained fresh evidence=%d",
			errSwarmionPublishedWriteIncomplete,
			status.ConfirmedOtherPeers,
			confirmedFromEvidence,
		)
	}
	expectedAvailable := status.ConfirmedOtherPeers >= status.RequiredOtherPeers
	if status.Available != expectedAvailable || (status.Available && (!status.Known || !status.LocalRootReady)) {
		return observation, fmt.Errorf(
			"%w: inconsistent availability status available=%t known=%t local_root_ready=%t required=%d confirmed=%d",
			errSwarmionPublishedWriteIncomplete,
			status.Available,
			status.Known,
			status.LocalRootReady,
			status.RequiredOtherPeers,
			status.ConfirmedOtherPeers,
		)
	}
	if status.NoCurrentEligiblePeers {
		if status.Available || status.ConfirmedOtherPeers != 0 || len(status.Peers) != 0 ||
			status.ReasonCode != swarmionapp.ReceiptAvailabilityReasonNoCurrentEligiblePeers {
			return observation, fmt.Errorf(
				"%w: inconsistent no-current-eligible-peers status available=%t confirmed=%d evidence=%d reason_code=%q",
				errSwarmionPublishedWriteIncomplete,
				status.Available,
				status.ConfirmedOtherPeers,
				len(status.Peers),
				status.ReasonCode,
			)
		}
	}
	expectedReasonCode := swarmionapp.ReceiptAvailabilityReasonNone
	switch {
	case expectedNoCurrentEligiblePeers:
		expectedReasonCode = swarmionapp.ReceiptAvailabilityReasonNoCurrentEligiblePeers
	case !status.Known:
		expectedReasonCode = swarmionapp.ReceiptAvailabilityReasonReceiptNotLocallyLive
	case !status.LocalRootReady:
		expectedReasonCode = swarmionapp.ReceiptAvailabilityReasonReceiptNotLocallyRootReady
	case expectedScope == swarmionapp.ReceiptAvailabilityCandidateScopeCurrentLogicalPeers &&
		len(eligiblePeerIDs) < status.RequiredOtherPeers:
		expectedReasonCode = swarmionapp.ReceiptAvailabilityReasonInsufficientCurrentEligible
	case !status.Available:
		expectedReasonCode = swarmionapp.ReceiptAvailabilityReasonInsufficientOtherPeerReceipts
	}
	if status.ReasonCode != expectedReasonCode {
		return observation, fmt.Errorf(
			"%w: inconsistent availability reason code returned=%q want=%q",
			errSwarmionPublishedWriteIncomplete,
			status.ReasonCode,
			expectedReasonCode,
		)
	}
	if (expectedReasonCode == swarmionapp.ReceiptAvailabilityReasonReceiptNotLocallyLive ||
		expectedReasonCode == swarmionapp.ReceiptAvailabilityReasonReceiptNotLocallyRootReady ||
		expectedReasonCode == swarmionapp.ReceiptAvailabilityReasonInsufficientCurrentEligible) &&
		len(status.Peers) != 0 {
		return observation, fmt.Errorf(
			"%w: early availability status reason_code=%q unexpectedly contains %d peer evidence records",
			errSwarmionPublishedWriteIncomplete,
			expectedReasonCode,
			len(status.Peers),
		)
	}
	observation.Status.Receipt = request.Receipt
	return observation, nil
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func receiptAvailabilityStatusHasIdentity(status swarmionapp.ReceiptAvailabilityStatus) bool {
	return strings.TrimSpace(status.Receipt.EventID) != "" || strings.TrimSpace(status.Receipt.PublishedRootHash) != ""
}

func canonicalPublishedWriteAvailabilityReceipt(
	receipt PublishedWriteReceipt,
	request swarmionapp.ReceiptAvailabilityRequest,
) PublishedWriteReceipt {
	receipt.EventID = request.Receipt.EventID
	receipt.PublishedRootHash = request.Receipt.PublishedRootHash
	return receipt
}

// ConfirmPublishedWriteAvailability observes the normal one-other-peer
// boundary for an already accepted ordinary write. Swarmion selects current
// database-scoped logical peers for the unscoped request. A
// NoCurrentEligiblePeers result returns promptly as weak local acceptance;
// otherwise Protos performs one passive, bounded availability wait. A pending
// or observation error is carried in the confirmation and does not turn the
// accepted write into an error or a replayable outcome.
func (db *DB) ConfirmPublishedWriteAvailability(
	ctx context.Context,
	receipt PublishedWriteReceipt,
	reason string,
) PublishedWriteConfirmation {
	confirmation := PublishedWriteConfirmation{
		Receipt: receipt,
	}
	if !receipt.HasExactEventIdentity() {
		confirmation.Stage = PublishedWriteConfirmationNoChange
		return confirmation
	}
	if !receipt.Committed {
		confirmation.AvailabilityPending = true
		confirmation.AvailabilityError = ErrPublishedWriteConfirmationUnresolved.Error()
		return confirmation
	}
	confirmation.Stage = PublishedWriteConfirmationLocalAccepted
	request, err := ReceiptAvailabilityRequestFromPublishedWriteReceipt(receipt, PublishedWriteAvailabilityOptions{})
	if err != nil {
		confirmation.AvailabilityPending = true
		confirmation.AvailabilityError = err.Error()
		return confirmation
	}
	runtime, err := db.publishedWriteAvailabilityRuntime()
	if err != nil {
		confirmation.AvailabilityPending = true
		confirmation.AvailabilityError = err.Error()
		return confirmation
	}
	boundedCtx, cancel := boundedPublishedWriteAvailabilityContext(ctx)
	defer cancel()
	return confirmPublishedWriteAvailability(boundedCtx, runtime, receipt, request, reason)
}

func confirmPublishedWriteAvailability(
	ctx context.Context,
	runtime publishedWriteAvailabilityRuntime,
	receipt PublishedWriteReceipt,
	request swarmionapp.ReceiptAvailabilityRequest,
	reason string,
) PublishedWriteConfirmation {
	confirmation := PublishedWriteConfirmation{
		Receipt: receipt,
	}
	if !receipt.Committed {
		confirmation.AvailabilityPending = true
		confirmation.AvailabilityError = ErrPublishedWriteConfirmationUnresolved.Error()
		return confirmation
	}
	confirmation.Stage = PublishedWriteConfirmationLocalAccepted
	initial, err := observePublishedWriteAvailability(ctx, runtime, receipt, request)
	confirmation.Receipt = initial.Receipt
	confirmation.Availability = initial.Status
	if err != nil {
		confirmation.AvailabilityPending = true
		confirmation.AvailabilityError = err.Error()
		return confirmation
	}
	if initial.Status.Available {
		confirmation.Stage = PublishedWriteConfirmationOtherPeerAvailable
		return confirmation
	}
	// Peers is receipt evidence, not topology. Only Swarmion's explicit
	// NoCurrentEligiblePeers status identifies the prompt weak single-peer
	// outcome for an ordinary unscoped request.
	if initial.Status.NoCurrentEligiblePeers {
		confirmation.AvailabilityPending = true
		return confirmation
	}

	status, waitErr := runtime.WaitReceiptAvailability(ctx, request)
	if receiptAvailabilityStatusHasIdentity(status) {
		observed, validationErr := publishedWriteAvailabilityObservation(initial.Receipt, request, status)
		if validationErr != nil {
			confirmation.AvailabilityPending = true
			confirmation.AvailabilityError = validationErr.Error()
			return confirmation
		}
		confirmation.Receipt = observed.Receipt
		confirmation.Availability = observed.Status
	}
	if waitErr != nil {
		confirmation.AvailabilityPending = true
		if errors.Is(waitErr, swarmionapp.ErrReceiptAvailabilityPending) &&
			confirmation.Availability.NoCurrentEligiblePeers {
			return confirmation
		}
		confirmation.AvailabilityError = waitErr.Error()
		return confirmation
	}
	if !confirmation.Availability.Available {
		confirmation.AvailabilityPending = true
		confirmation.AvailabilityError = (&PublishedWriteAvailabilityPendingError{
			Observation: PublishedWriteAvailabilityObservation{
				Receipt: confirmation.Receipt,
				Status:  confirmation.Availability,
			},
			Reason: reason,
			Cause:  fmt.Errorf("receipt availability wait returned before other-peer availability"),
		}).Error()
		return confirmation
	}
	confirmation.Stage = PublishedWriteConfirmationOtherPeerAvailable
	return confirmation
}

// InsertWithAvailabilityContext publishes an ordinary insert and observes its
// one-other-peer availability boundary. Only publication failures are returned
// as errors; a conservative availability miss remains an accepted confirmation.
func InsertWithAvailabilityContext(ctx context.Context, db *DB, mappers ...InsertMapper) (PublishedWriteConfirmation, error) {
	return confirmOrdinaryPublishedWrite(
		func() (PublishedWriteReceipt, error) {
			return InsertWithReceiptContext(ctx, db, mappers...)
		},
		func(receipt PublishedWriteReceipt) PublishedWriteConfirmation {
			return db.ConfirmPublishedWriteAvailability(ctx, receipt, "confirm ordinary insert on another peer")
		},
	)
}

// UpdateWithAvailabilityContext is the update counterpart to
// InsertWithAvailabilityContext. A conventional no-op reports no_change.
func UpdateWithAvailabilityContext(ctx context.Context, db *DB, mappers ...UpdateMapper) (PublishedWriteConfirmation, error) {
	return confirmOrdinaryPublishedWrite(
		func() (PublishedWriteReceipt, error) {
			return UpdateWithReceiptContext(ctx, db, mappers...)
		},
		func(receipt PublishedWriteReceipt) PublishedWriteConfirmation {
			return db.ConfirmPublishedWriteAvailability(ctx, receipt, "confirm ordinary update on another peer")
		},
	)
}

// UpdateAndInsertWithAvailabilityContext publishes the mapper set atomically
// and observes its exact receipt on another peer.
func UpdateAndInsertWithAvailabilityContext(
	ctx context.Context,
	db *DB,
	updates []UpdateMapper,
	inserts []InsertMapper,
) (PublishedWriteConfirmation, error) {
	return confirmOrdinaryPublishedWrite(
		func() (PublishedWriteReceipt, error) {
			return UpdateAndInsertWithReceiptContext(ctx, db, updates, inserts)
		},
		func(receipt PublishedWriteReceipt) PublishedWriteConfirmation {
			return db.ConfirmPublishedWriteAvailability(ctx, receipt, "confirm ordinary update and insert on another peer")
		},
	)
}

// DeleteWithAvailabilityContext publishes an ordinary application-state
// delete and observes its exact receipt on another peer. This helper is not for
// provider-resource deletion; those workflows retain their AppliedDurably and
// checkpoint-invariant boundary.
func DeleteWithAvailabilityContext(ctx context.Context, db *DB, mappers ...DeleteMapper) (PublishedWriteConfirmation, error) {
	return confirmOrdinaryPublishedWrite(
		func() (PublishedWriteReceipt, error) {
			return DeleteWithReceiptContext(ctx, db, mappers...)
		},
		func(receipt PublishedWriteReceipt) PublishedWriteConfirmation {
			return db.ConfirmPublishedWriteAvailability(ctx, receipt, "confirm ordinary delete on another peer")
		},
	)
}

func confirmOrdinaryPublishedWrite(
	publish func() (PublishedWriteReceipt, error),
	confirm func(PublishedWriteReceipt) PublishedWriteConfirmation,
) (PublishedWriteConfirmation, error) {
	receipt, err := publish()
	if err != nil {
		return publishedWriteConfirmationForPublicationError(receipt, err)
	}
	return confirm(receipt), nil
}

func publishedWriteConfirmationForPublicationError(
	receipt PublishedWriteReceipt,
	cause error,
) (PublishedWriteConfirmation, error) {
	// An exact receipt accompanying a publication error may already have been
	// accepted, but it is not proof of local acceptance. Preserve the identity
	// while leaving Stage unset. A typed unresolved error makes the receipt
	// recoverable even through callers that otherwise retain only the error.
	confirmation := PublishedWriteConfirmation{Receipt: receipt}
	if cause == nil || !receipt.HasExactEventIdentity() {
		return confirmation, cause
	}
	return confirmation, &PublishedWriteConfirmationUnresolvedError{
		Confirmation: confirmation,
		Cause:        cause,
	}
}
