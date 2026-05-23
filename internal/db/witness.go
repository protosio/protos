package db

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"swarmion.dev/protocol"
	"swarmion.dev/runtime/adminrpc"
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
	witnessRankRequestTimeout   = 5 * time.Second
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

	eligible := stringSet(status.EligibleWitnessIDs)
	for _, candidate := range ranked {
		candidateEligible := setHas(eligible, candidate.PeerID)
		if candidate.PeerID != status.PeerID && candidateEligible {
			continue
		}
		if _, err := db.requestWitnessRank(ctx, app, status, candidate, candidateEligible); err != nil {
			notifyLog.Debugf("failed to request swarmion witness rank for peer %s: %s", candidate.PeerID, err.Error())
		}
	}

	formation := eligibleWitnessFormation(ranked, eligible)
	if len(formation) == 0 {
		return nil
	}
	if _, _, ok := WitnessFormationInStatus(status, formation); ok {
		return nil
	}
	activeEpochID := witnessChangeEpochID(status.ActiveEpochID)
	if containsString(formation, status.PeerID) {
		return db.applyWitnessFormation(ctx, app, activeEpochID, formation)
	}
	return db.applyRemoteWitnessFormation(ctx, status, activeEpochID, formation)
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
	rankFloor := eligibleUserClientRankFloor(candidates, eligible)
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

func eligibleUserClientRankFloor(candidates []rankedWitnessCandidate, eligible map[string]struct{}) int {
	rankFloor := 0
	for _, candidate := range candidates {
		if !setHas(eligible, candidate.PeerID) {
			continue
		}
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
	status swarmionapp.Status,
	candidate rankedWitnessCandidate,
	candidateEligible bool,
) (bool, error) {
	if candidate.PeerID == status.PeerID {
		if !db.reserveWitnessRankRequest("local:"+candidate.PeerID, candidate.Rank, !candidateEligible) {
			return false, nil
		}
		if _, err := app.PublishWitnessRankUpdate(ctx, candidate.Rank); err != nil {
			return false, err
		}
		notifyLog.Infof("published local swarmion witness rank %d for %s peer %s", candidate.Rank, candidate.DeviceType, candidate.PeerID)
		return true, nil
	}

	if !candidateEligible && db.reserveWitnessRankRequest("owner:"+candidate.PeerID, candidate.Rank, true) {
		if _, err := app.PublishWitnessRankUpdateForSubject(ctx, candidate.PeerID, candidate.Rank); err == nil {
			notifyLog.Infof("published owner-authorized swarmion witness rank %d for %s peer %s", candidate.Rank, candidate.DeviceType, candidate.PeerID)
			return true, nil
		} else {
			notifyLog.Debugf("failed to publish owner-authorized swarmion witness rank for peer %s: %s", candidate.PeerID, err.Error())
		}
	}

	if !containsString(status.ConnectedPeers, candidate.PeerID) {
		return false, nil
	}
	if !db.reserveWitnessRankRequest("remote:"+candidate.PeerID, candidate.Rank, true) {
		return false, nil
	}

	db.mu.Lock()
	network := db.network
	adminNamespace := fmt.Sprintf(swarmionAdminNamespaceTemplate, db.name)
	db.mu.Unlock()
	if network == nil {
		return false, nil
	}

	client, err := adminrpc.NewClient(network, adminrpc.Config{Namespace: adminNamespace})
	if err != nil {
		return false, err
	}
	callCtx, cancel := witnessRankContext(ctx)
	defer cancel()
	if _, err := client.WitnessEligibility(callCtx, candidate.PeerID, adminrpc.WitnessEligibilityRequest{
		Kind: "rank",
		Rank: candidate.Rank,
	}); err != nil {
		return false, err
	}
	notifyLog.Infof("requested swarmion witness rank %d for %s peer %s", candidate.Rank, candidate.DeviceType, candidate.PeerID)
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

func witnessRankContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		return context.WithTimeout(context.Background(), witnessRankRequestTimeout)
	}
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= witnessRankRequestTimeout {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, witnessRankRequestTimeout)
}

func witnessChangeEpochID(activeEpochID string) string {
	if activeEpochID == "" || strings.EqualFold(activeEpochID, "main") {
		return ""
	}
	return activeEpochID
}

func (db *DB) applyWitnessFormation(ctx context.Context, app *swarmionapp.App, activeEpochID string, formation []string) error {
	change := swarmionapp.WitnessChange{
		EpochID:    activeEpochID,
		WitnessIDs: append([]string(nil), formation...),
		DryRun:     true,
		Reason:     "reconcile Protos witness formation",
	}
	dryRun, err := app.ProposeWitnesses(ctx, change)
	if err != nil {
		return fmt.Errorf("dry-run swarmion witness formation: %w", err)
	}
	if !dryRun.Accepted {
		pruned := pruneWitnessFormation(formation, dryRun.IneligibleWitnessIDs, dryRun.UnreadyWitnessIDs)
		if len(pruned) != len(formation) && containsString(pruned, app.PeerID()) {
			change.WitnessIDs = pruned
			dryRun, err = app.ProposeWitnesses(ctx, change)
			if err != nil {
				return fmt.Errorf("dry-run pruned swarmion witness formation: %w", err)
			}
		}
	}
	if !dryRun.Accepted {
		notifyLog.Debugf("swarmion witness formation not ready: %s", dryRun.Reason)
		return nil
	}
	if len(dryRun.AddedWitnessIDs) == 0 && len(dryRun.RemovedActiveWitnessIDs) == 0 {
		return nil
	}

	change.DryRun = false
	applied, err := app.ApplyWitnesses(ctx, change)
	if err != nil {
		return fmt.Errorf("apply swarmion witness formation: %w", err)
	}
	if !applied.Accepted {
		return fmt.Errorf("swarmion rejected witness formation: %s", applied.Reason)
	}
	notifyLog.Infof(
		"applied swarmion witness formation %v for epoch %s; added=%v removed=%v",
		change.WitnessIDs,
		applied.EpochID,
		applied.AddedWitnessIDs,
		applied.RemovedActiveWitnessIDs,
	)
	return nil
}

func (db *DB) applyRemoteWitnessFormation(ctx context.Context, status swarmionapp.Status, activeEpochID string, formation []string) error {
	db.mu.Lock()
	network := db.network
	adminNamespace := fmt.Sprintf(swarmionAdminNamespaceTemplate, db.name)
	db.mu.Unlock()
	if network == nil {
		return nil
	}
	connected := stringSet(status.ConnectedPeers)
	client, err := adminrpc.NewClient(network, adminrpc.Config{Namespace: adminNamespace})
	if err != nil {
		return err
	}
	var lastReason string
	for _, peerID := range formation {
		if !setHas(connected, peerID) {
			continue
		}
		req := adminrpc.PlannedWitnessChangeRequest{
			EpochID:   activeEpochID,
			Witnesses: append([]string(nil), formation...),
			DryRun:    true,
			Reason:    "reconcile Protos witness formation",
		}
		dryRun, err := client.PlannedWitnessChange(ctx, peerID, req)
		if err != nil {
			lastReason = err.Error()
			continue
		}
		if !dryRun.Accepted {
			lastReason = dryRun.Reason
			continue
		}
		if len(dryRun.AddedWitnessIDs) == 0 && len(dryRun.RemovedActiveWitnessIDs) == 0 {
			return nil
		}

		req.DryRun = false
		req.Adopt = true
		applied, err := client.PlannedWitnessChange(ctx, peerID, req)
		if err != nil {
			return fmt.Errorf("apply remote swarmion witness formation via %s: %w", peerID, err)
		}
		if !applied.Accepted {
			return fmt.Errorf("remote swarmion witness formation via %s rejected: %s", peerID, applied.Reason)
		}
		notifyLog.Infof(
			"applied remote swarmion witness formation %v via %s for epoch %s; added=%v removed=%v",
			req.Witnesses,
			peerID,
			applied.EpochID,
			applied.AddedWitnessIDs,
			applied.RemovedActiveWitnessIDs,
		)
		return nil
	}
	if lastReason != "" {
		notifyLog.Debugf("remote swarmion witness formation not ready: %s", lastReason)
	}
	return nil
}

func pruneWitnessFormation(formation []string, removeSets ...[]string) []string {
	remove := map[string]struct{}{}
	for _, values := range removeSets {
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value != "" {
				remove[value] = struct{}{}
			}
		}
	}
	pruned := make([]string, 0, len(formation))
	for _, peerID := range formation {
		if _, found := remove[peerID]; found {
			continue
		}
		pruned = append(pruned, peerID)
	}
	return pruned
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

func setHas(set map[string]struct{}, value string) bool {
	_, found := set[value]
	return found
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
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
