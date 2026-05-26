package db

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"swarmion.dev/protocol"
	swarmionapp "swarmion.dev/runtime/app"
)

const (
	WitnessDeviceTypePhone           = "phone"
	WitnessDeviceTypeLocalUserClient = "local_user_client"
	WitnessDeviceTypeUserDevice      = "user_device"
	WitnessDeviceTypeLocalVM         = "local_vm"
	WitnessDeviceTypeCloudVM         = "cloud_vm"

	protosWitnessSelectionLimit = 2
	witnessRankRetryAfter       = 30 * time.Second
)

var witnessRankByDeviceType = map[string]int{
	WitnessDeviceTypePhone:           10,
	WitnessDeviceTypeLocalVM:         30,
	WitnessDeviceTypeLocalUserClient: 50,
	WitnessDeviceTypeUserDevice:      50,
	WitnessDeviceTypeCloudVM:         100,
}

type WitnessCandidate struct {
	PeerID     string
	DeviceType string
	Rank       int
}

type pendingWitnessRankRequest struct {
	Rank        int
	RequestedAt time.Time
}

type rankedWitnessCandidate struct {
	PeerID     string
	DeviceType string
	Rank       int
}

func WitnessRankForDeviceType(deviceType string) (int, bool) {
	rank, found := witnessRankByDeviceType[normalizeWitnessDeviceType(deviceType)]
	return rank, found
}

func DefaultWitnessRankForDeviceType(deviceType string) int {
	rank, _ := WitnessRankForDeviceType(deviceType)
	return rank
}

func WitnessDeviceTypeForUserDeviceName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, marker := range []string{"phone", "iphone", "android", "mobile"} {
		if strings.Contains(name, marker) {
			return WitnessDeviceTypePhone
		}
	}
	return WitnessDeviceTypeLocalUserClient
}

func DefaultWitnessRankForUserDeviceName(name string) int {
	return DefaultWitnessRankForDeviceType(WitnessDeviceTypeForUserDeviceName(name))
}

func WitnessDeviceTypeForMachine(kind string, kindID string) string {
	if normalizeWitnessDeviceType(kind) == WitnessDeviceTypeLocalVM || strings.EqualFold(kindID, "local_macos") {
		return WitnessDeviceTypeLocalVM
	}
	return normalizeWitnessDeviceType(kind)
}

func DefaultWitnessRankForMachine(kind string, kindID string) int {
	return DefaultWitnessRankForDeviceType(WitnessDeviceTypeForMachine(kind, kindID))
}

func normalizeWitnessDeviceType(deviceType string) string {
	deviceType = strings.ToLower(strings.TrimSpace(deviceType))
	deviceType = strings.ReplaceAll(deviceType, "-", "_")
	switch deviceType {
	case "phone", "mobile", "ios", "iphone", "android":
		return WitnessDeviceTypePhone
	case "laptop", "desktop", "workstation", "local", "local_client", "local_user", "local_user_client":
		return WitnessDeviceTypeLocalUserClient
	case "device", "user":
		return WitnessDeviceTypeUserDevice
	case "vm", "localvm", "local_vm":
		return WitnessDeviceTypeLocalVM
	case "cloud", "cloudvm", "cloud_vm", "vps":
		return WitnessDeviceTypeCloudVM
	default:
		return deviceType
	}
}

func (db *DB) ReconcileWitnesses(ctx context.Context, candidates []WitnessCandidate) error {
	if db == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	db.mu.Lock()
	app := db.app
	db.mu.Unlock()
	if app == nil {
		return nil
	}

	if _, err := app.CatchUpFinalized(ctx, "reconcile Protos witnesses"); err != nil {
		return fmt.Errorf("catch up swarmion finalized state for witness reconciliation: %w", err)
	}
	status := app.Status()
	if status.Fatal != nil {
		return fmt.Errorf("swarmion fatal state blocks witness reconciliation: %s", status.Fatal.State)
	}
	if err := blockOnIncompatiblePeers(ctx, app); err != nil {
		return err
	}

	ranked := rankedWitnessCandidates(candidates)
	if len(ranked) == 0 {
		return nil
	}

	eligible := eligibleWitnessesWithExpectedRanks(status, ranked)
	witnessCandidates := witnessCandidateSetForApply(ranked)
	if len(witnessCandidates) == 0 {
		return nil
	}

	formation := stringSet(rankedCandidatePeerIDs(witnessCandidates))
	for _, candidate := range ranked {
		if candidate.PeerID != status.PeerID || setHas(formation, candidate.PeerID) {
			continue
		}
		if _, err := db.requestWitnessRank(ctx, app, candidate, setHas(eligible, candidate.PeerID)); err != nil {
			notifyLog.Debugf("failed to request swarmion witness rank for peer %s: %s", candidate.PeerID, err.Error())
		}
	}

	if !setHas(formation, status.PeerID) {
		notifyLog.Debugf(
			"local swarmion peer %s is outside desired witness candidate formation %v",
			status.PeerID,
			rankedCandidatePeerIDs(witnessCandidates),
		)
		return nil
	}
	return db.applyWitnessCandidates(ctx, app, status, witnessCandidates)
}

func WitnessFormationInStatus(status swarmionapp.Status, formation []string) ([]string, string, bool) {
	if witnessSetEquals(status.ActiveWitnessIDs, formation) {
		return append([]string(nil), status.ActiveWitnessIDs...), status.ActiveEpochID, true
	}
	for epochID, snapshot := range status.EpochSnapshots {
		activeWitnesses := peerIDsToStrings(snapshot.ActiveWitnessIDs)
		if witnessSetEquals(activeWitnesses, formation) {
			return activeWitnesses, epochID, true
		}
	}
	return nil, "", false
}

func blockOnIncompatiblePeers(ctx context.Context, app *swarmionapp.App) error {
	compatibility, err := app.Compatibility(ctx)
	if err != nil {
		return fmt.Errorf("check swarmion compatibility: %w", err)
	}
	for _, item := range compatibility {
		if !item.Blocking {
			continue
		}
		reason := strings.TrimSpace(item.Reason)
		if reason == "" {
			reason = "remote peer advertises an incompatible swarm manifest"
		}
		return fmt.Errorf("swarmion compatibility blocks witness reconciliation for peer %s: %s", item.PeerID, reason)
	}
	return nil
}

func rankedWitnessCandidates(candidates []WitnessCandidate) []rankedWitnessCandidate {
	byPeer := make(map[string]rankedWitnessCandidate, len(candidates))
	for _, candidate := range candidates {
		peerID := strings.TrimSpace(candidate.PeerID)
		if peerID == "" {
			continue
		}
		deviceType := normalizeWitnessDeviceType(candidate.DeviceType)
		rank := candidate.Rank
		if rank <= 0 {
			var found bool
			rank, found = WitnessRankForDeviceType(deviceType)
			if !found {
				continue
			}
		}
		if rank <= 0 {
			continue
		}
		ranked := rankedWitnessCandidate{
			PeerID:     peerID,
			DeviceType: deviceType,
			Rank:       rank,
		}
		if existing, found := byPeer[peerID]; !found || ranked.Rank > existing.Rank {
			byPeer[peerID] = ranked
		}
	}

	out := make([]rankedWitnessCandidate, 0, len(byPeer))
	for _, candidate := range byPeer {
		out = append(out, candidate)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Rank != out[j].Rank {
			return out[i].Rank > out[j].Rank
		}
		return out[i].PeerID < out[j].PeerID
	})
	return out
}

func eligibleWitnessFormation(candidates []rankedWitnessCandidate, eligible map[string]struct{}) []string {
	cloudFormation := eligibleCloudWitnessFormation(candidates, eligible)
	if len(cloudFormation) >= protosWitnessSelectionLimit {
		return cloudFormation
	}
	formation := make([]string, 0, len(candidates))
	rankFloor := userClientRankFloor(candidates)
	holdPartialCloudPromotion := len(cloudFormation) > 0 && rankFloor > 0
	for _, candidate := range candidates {
		if !setHas(eligible, candidate.PeerID) {
			continue
		}
		if holdPartialCloudPromotion && candidate.DeviceType == WitnessDeviceTypeCloudVM {
			continue
		}
		if candidate.Rank < rankFloor {
			continue
		}
		formation = append(formation, candidate.PeerID)
		if len(formation) == protosWitnessSelectionLimit {
			break
		}
	}
	return formation
}

func eligibleWitnessCandidateSet(candidates []rankedWitnessCandidate, eligible map[string]struct{}) []rankedWitnessCandidate {
	cloudFormation := eligibleCloudWitnessFormation(candidates, eligible)
	if len(cloudFormation) >= protosWitnessSelectionLimit {
		return cloudHandoffWitnessCandidates(candidates, eligible, cloudFormation)
	}
	return rankedCandidatesByPeerID(candidates, eligibleWitnessFormation(candidates, eligible))
}

func witnessCandidateSetForApply(candidates []rankedWitnessCandidate) []rankedWitnessCandidate {
	cloudFormation := eligibleCloudWitnessFormation(candidates, allCandidatePeerSet(candidates))
	if len(cloudFormation) >= protosWitnessSelectionLimit {
		return cloudHandoffWitnessCandidates(candidates, allCandidatePeerSet(candidates), cloudFormation)
	}
	return eligibleWitnessCandidateSet(candidates, allCandidatePeerSet(candidates))
}

func allCandidatePeerSet(candidates []rankedWitnessCandidate) map[string]struct{} {
	out := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.PeerID) != "" {
			out[candidate.PeerID] = struct{}{}
		}
	}
	return out
}

func activeWitnessCandidateFormationSatisfied(status swarmionapp.Status, candidates []rankedWitnessCandidate) bool {
	formation := rankedCandidatePeerIDs(candidates)
	if len(formation) == 0 {
		return false
	}
	if snapshot, found := status.EpochSnapshots[status.ActiveEpochID]; found {
		activeFormation := peerIDsToStrings(snapshot.FormationSet)
		if len(activeFormation) == 0 {
			activeFormation = peerIDsToStrings(snapshot.ActiveWitnessIDs)
		}
		return witnessSetEquals(activeFormation, formation)
	}
	return witnessSetEquals(status.ActiveWitnessIDs, formation)
}

func cloudHandoffWitnessCandidates(candidates []rankedWitnessCandidate, eligible map[string]struct{}, cloudFormation []string) []rankedWitnessCandidate {
	want := stringSet(cloudFormation)
	for _, candidate := range candidates {
		if !setHas(eligible, candidate.PeerID) {
			continue
		}
		if candidate.DeviceType != WitnessDeviceTypeCloudVM {
			want[candidate.PeerID] = struct{}{}
		}
	}
	return rankedCandidatesByPeerID(candidates, setValues(want))
}

func rankedCandidatesByPeerID(candidates []rankedWitnessCandidate, peerIDs []string) []rankedWitnessCandidate {
	if len(peerIDs) == 0 {
		return nil
	}
	want := stringSet(peerIDs)
	out := make([]rankedWitnessCandidate, 0, len(peerIDs))
	for _, candidate := range candidates {
		if setHas(want, candidate.PeerID) {
			out = append(out, candidate)
		}
	}
	return out
}

func rankedCandidatePeerIDs(candidates []rankedWitnessCandidate) []string {
	out := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.PeerID) != "" {
			out = append(out, candidate.PeerID)
		}
	}
	return out
}

func eligibleWitnessesWithExpectedRanks(status swarmionapp.Status, candidates []rankedWitnessCandidate) map[string]struct{} {
	out := map[string]struct{}{}
	eligible := stringSet(status.EligibleWitnessIDs)
	for _, candidate := range candidates {
		if !setHas(eligible, candidate.PeerID) {
			continue
		}
		if status.EligibleWitnessRanks != nil {
			if rank, found := status.EligibleWitnessRanks[candidate.PeerID]; !found || rank != candidate.Rank {
				continue
			}
		}
		out[candidate.PeerID] = struct{}{}
	}
	return out
}

func eligibleCloudWitnessFormation(candidates []rankedWitnessCandidate, eligible map[string]struct{}) []string {
	formation := make([]string, 0, protosWitnessSelectionLimit)
	for _, candidate := range candidates {
		if candidate.DeviceType != WitnessDeviceTypeCloudVM || !setHas(eligible, candidate.PeerID) {
			continue
		}
		formation = append(formation, candidate.PeerID)
		if len(formation) == protosWitnessSelectionLimit {
			break
		}
	}
	return formation
}

func userClientRankFloor(candidates []rankedWitnessCandidate) int {
	rankFloor := 0
	for _, candidate := range candidates {
		switch candidate.DeviceType {
		case WitnessDeviceTypeLocalUserClient, WitnessDeviceTypeUserDevice:
			if candidate.Rank > rankFloor {
				rankFloor = candidate.Rank
			}
		}
	}
	return rankFloor
}

func (db *DB) requestWitnessRank(
	ctx context.Context,
	app *swarmionapp.App,
	candidate rankedWitnessCandidate,
	candidateEligible bool,
) (bool, error) {
	if candidateEligible {
		return false, nil
	}
	if !db.reserveWitnessRankRequest("local:"+candidate.PeerID, candidate.Rank, true) {
		return false, nil
	}
	if _, err := app.PublishWitnessRankUpdate(ctx, candidate.Rank); err != nil {
		return false, err
	}
	notifyLog.Infof("published local swarmion witness rank %d for %s peer %s", candidate.Rank, candidate.DeviceType, candidate.PeerID)
	return true, nil
}

func (db *DB) reserveWitnessRankRequest(peerID string, rank int, retry bool) bool {
	now := time.Now()
	db.witnessMu.Lock()
	defer db.witnessMu.Unlock()
	if db.witnessRankRequests == nil {
		db.witnessRankRequests = make(map[string]pendingWitnessRankRequest)
	}
	if pending, found := db.witnessRankRequests[peerID]; found &&
		pending.Rank == rank {
		if !retry || now.Sub(pending.RequestedAt) < witnessRankRetryAfter {
			return false
		}
	}
	db.witnessRankRequests[peerID] = pendingWitnessRankRequest{
		Rank:        rank,
		RequestedAt: now,
	}
	return true
}

func (db *DB) applyWitnessCandidates(ctx context.Context, app *swarmionapp.App, status swarmionapp.Status, candidates []rankedWitnessCandidate) error {
	candidateSet := swarmionapp.WitnessCandidates{
		Candidates: swarmionWitnessCandidates(candidates),
		Source:     swarmionWitnessCandidateSource(status),
		Reason:     "reconcile Protos witness candidates",
	}
	applied, err := app.ApplyWitnessCandidates(ctx, candidateSet)
	if err != nil {
		return fmt.Errorf("apply swarmion witness candidates: %w", err)
	}
	if !applied.Accepted {
		notifyLog.Infof(
			"swarmion witness candidates not ready: kind=%s reason=%s blockers=%v pending_rank_updates=%v formation=%v selected=%v",
			applied.Kind,
			applied.Reason,
			applied.BlockerCodes,
			applied.PendingRankUpdatePeerIDs,
			applied.FormationIDs,
			applied.SelectedWitnessIDs,
		)
		return nil
	}
	if applied.Noop {
		notifyLog.Debugf(
			"swarmion witness candidates no-op for local peer %s: kind=%s reason=%s formation=%v active_epoch=%s",
			app.PeerID(),
			applied.Kind,
			applied.Reason,
			applied.FormationIDs,
			applied.EpochID,
		)
		return nil
	}
	if !applied.AdoptedLocalEpoch {
		notifyLog.Infof(
			"swarmion witness candidates accepted before local epoch adoption: kind=%s epoch=%s reason=%s safe_to_author=%v unsafe_to_author=%s",
			applied.Kind,
			applied.EpochID,
			applied.Reason,
			applied.SafeToAuthor,
			applied.UnsafeToAuthorReason,
		)
		return nil
	}
	if !applied.SafeToAuthor {
		notifyLog.Infof(
			"swarmion witness candidates adopted but local authoring is not ready yet: kind=%s epoch=%s unsafe_to_author=%s",
			applied.Kind,
			applied.EpochID,
			applied.UnsafeToAuthorReason,
		)
		return nil
	}
	notifyLog.Infof(
		"applied swarmion witness candidates kind=%s formation=%v selected=%v epoch=%s safe_to_disconnect=%v safe_to_author=%v",
		applied.Kind,
		applied.FormationIDs,
		applied.SelectedWitnessIDs,
		applied.EpochID,
		applied.SafeToDisconnectPeerIDs,
		applied.SafeToAuthor,
	)
	return nil
}

func swarmionWitnessCandidateSource(status swarmionapp.Status) swarmionapp.WitnessCandidateSource {
	return swarmionapp.WitnessCandidateSource{
		ActiveEpochID:     status.ActiveEpochID,
		FinalizedCommitID: status.FinalizedCommitID.String(),
		FinalizedRootHash: status.FinalizedRootHash.String(),
	}
}

func swarmionWitnessCandidates(candidates []rankedWitnessCandidate) []swarmionapp.WitnessCandidate {
	out := make([]swarmionapp.WitnessCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, swarmionapp.WitnessCandidate{
			PeerID: candidate.PeerID,
			Rank:   candidate.Rank,
		})
	}
	return out
}

func stringSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out[value] = struct{}{}
		}
	}
	return out
}

func setValues(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for value := range set {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

func setHas(set map[string]struct{}, value string) bool {
	_, found := set[value]
	return found
}

func witnessSetEquals(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	rightSet := stringSet(right)
	for _, value := range left {
		if !setHas(rightSet, value) {
			return false
		}
	}
	return true
}

func peerIDsToStrings(values []protocol.PeerID) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		peerID := strings.TrimSpace(string(value))
		if peerID != "" {
			out = append(out, peerID)
		}
	}
	return out
}
