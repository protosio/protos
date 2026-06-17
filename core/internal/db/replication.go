package db

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	swarmionapp "github.com/nustiueudinastea/swarmion/runtime/app"
)

const (
	ReplicationDeviceClassPhone           = "phone"
	ReplicationDeviceClassLocalUserClient = "local_user_client"
	ReplicationDeviceClassUserDevice      = "user_device"
	ReplicationDeviceClassLocalVM         = "local_vm"
	ReplicationDeviceClassCloudVM         = "cloud_vm"
)

var replicationPriorityByDeviceClass = map[string]int{
	ReplicationDeviceClassPhone:           10,
	ReplicationDeviceClassLocalVM:         30,
	ReplicationDeviceClassLocalUserClient: 50,
	ReplicationDeviceClassUserDevice:      50,
	ReplicationDeviceClassCloudVM:         100,
}

type ReplicationCandidate struct {
	PeerID      string
	DeviceClass string
	Priority    int
}

type prioritizedReplicationCandidate struct {
	PeerID      string
	DeviceClass string
	Priority    int
}

func ReplicationPriorityForDeviceClass(deviceClass string) (int, bool) {
	priority, found := replicationPriorityByDeviceClass[normalizeReplicationDeviceClass(deviceClass)]
	return priority, found
}

func DefaultReplicationPriorityForDeviceClass(deviceClass string) int {
	priority, _ := ReplicationPriorityForDeviceClass(deviceClass)
	return priority
}

func ReplicationDeviceClassForUserDeviceName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, marker := range []string{"phone", "iphone", "android", "mobile"} {
		if strings.Contains(name, marker) {
			return ReplicationDeviceClassPhone
		}
	}
	return ReplicationDeviceClassLocalUserClient
}

func DefaultReplicationPriorityForUserDeviceName(name string) int {
	return DefaultReplicationPriorityForDeviceClass(ReplicationDeviceClassForUserDeviceName(name))
}

func ReplicationDeviceClassForMachine(kind string, kindID string) string {
	if normalizeReplicationDeviceClass(kind) == ReplicationDeviceClassLocalVM || strings.EqualFold(kindID, "local_macos") {
		return ReplicationDeviceClassLocalVM
	}
	return normalizeReplicationDeviceClass(kind)
}

func DefaultReplicationPriorityForMachine(kind string, kindID string) int {
	return DefaultReplicationPriorityForDeviceClass(ReplicationDeviceClassForMachine(kind, kindID))
}

func normalizeReplicationDeviceClass(deviceClass string) string {
	deviceClass = strings.ToLower(strings.TrimSpace(deviceClass))
	deviceClass = strings.ReplaceAll(deviceClass, "-", "_")
	switch deviceClass {
	case "phone", "mobile", "ios", "iphone", "android":
		return ReplicationDeviceClassPhone
	case "laptop", "desktop", "workstation", "local", "local_client", "local_user", "local_user_client":
		return ReplicationDeviceClassLocalUserClient
	case "device", "user":
		return ReplicationDeviceClassUserDevice
	case "vm", "localvm", "local_vm":
		return ReplicationDeviceClassLocalVM
	case "cloud", "cloudvm", "cloud_vm", "vps":
		return ReplicationDeviceClassCloudVM
	default:
		return deviceClass
	}
}

func (db *DB) ReconcileReplicationPeers(ctx context.Context, candidates []ReplicationCandidate) error {
	if db == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Minute)
		defer cancel()
	}

	db.mu.Lock()
	app := db.app
	db.mu.Unlock()
	if app == nil {
		return nil
	}

	if err := catchUpSwarmionCheckpoint(ctx, app, "reconcile Protos replication metadata"); err != nil {
		return fmt.Errorf("catch up swarmion checkpoint state for replication metadata reconciliation: %w", err)
	}
	status := app.Status()
	if status.Fatal != nil {
		return fmt.Errorf("swarmion fatal state blocks replication metadata reconciliation: %s", status.Fatal.State)
	}
	if err := blockOnIncompatiblePeers(ctx, app); err != nil {
		return err
	}

	prioritized := prioritizedReplicationCandidates(candidates)
	if len(prioritized) == 0 {
		return nil
	}
	if db.shouldLogReplicationPolicyNotice(prioritized) {
		notifyLog.Debugf(
			"swarmion checkpoint runtime has no replication-policy mutation API; retained %d Protos replication-priority candidates as local metadata",
			len(prioritized),
		)
	}
	return nil
}

func (db *DB) shouldLogReplicationPolicyNotice(prioritized []prioritizedReplicationCandidate) bool {
	if db == nil {
		return false
	}
	signature := replicationPolicyNoticeSignature(prioritized)
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.replicationNoticeSig == signature {
		return false
	}
	db.replicationNoticeSig = signature
	return true
}

func replicationPolicyNoticeSignature(prioritized []prioritizedReplicationCandidate) string {
	var b strings.Builder
	for _, candidate := range prioritized {
		if candidate.PeerID == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('|')
		}
		b.WriteString(candidate.PeerID)
		b.WriteByte(':')
		b.WriteString(candidate.DeviceClass)
		b.WriteByte(':')
		b.WriteString(fmt.Sprint(candidate.Priority))
	}
	return b.String()
}

func (db *DB) RemoveReplicationPeerState(ctx context.Context, peerID string, _ []ReplicationCandidate) error {
	peerID = strings.TrimSpace(peerID)
	if db == nil || peerID == "" {
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

	var lastErr error
	for attempt := 0; ; attempt++ {
		attemptCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		err := removeReplicationPeerStateWithApp(attemptCtx, app, peerID)
		cancel()
		if err == nil {
			return nil
		}
		if !retryablePeerRemovalError(err) {
			return err
		}
		lastErr = err
		notifyLog.Debugf("retrying swarmion peer removal for %s after retryable blocker on attempt %d: %s", peerID, attempt+1, err.Error())
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("%w: %w", ctx.Err(), lastErr)
			}
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

var errSwarmionPeerRemovalNotReady = errors.New("swarmion peer removal not ready")

func removeReplicationPeerStateWithApp(ctx context.Context, app *swarmionapp.App, peerID string) error {
	if err := catchUpSwarmionCheckpoint(ctx, app, "remove Protos replication peer state"); err != nil {
		return fmt.Errorf("catch up swarmion checkpoint state for replication peer removal: %w", err)
	}
	status := app.Status()
	if status.Fatal != nil {
		return fmt.Errorf("swarmion fatal state blocks replication peer removal: %s", status.Fatal.State)
	}
	if err := blockOnIncompatiblePeers(ctx, app); err != nil {
		return err
	}

	readiness, err := app.PeerRemovalReadiness(ctx, swarmionapp.PeerRemovalReadinessRequest{PeerID: peerID})
	if err != nil {
		return fmt.Errorf("read swarmion peer removal readiness for %s: %w", peerID, err)
	}
	if err := StrictPeerRemovalReadinessError(readiness); err == nil {
		notifyLog.Debugf("swarmion peer %s is already ready for durable removal: %s", peerID, PeerRemovalReadinessSummary(readiness))
		return nil
	} else {
		notifyLog.Debugf("swarmion peer %s requires eviction before durable removal: %s", peerID, PeerRemovalReadinessSummary(readiness))
	}

	eviction, err := app.EvictPeer(ctx, swarmionapp.PeerEvictionRequest{PeerID: peerID})
	if err != nil {
		return fmt.Errorf("evict swarmion peer %s after removal: %w", peerID, err)
	}
	if eviction.RemovalReadiness != nil {
		readiness = *eviction.RemovalReadiness
	} else {
		readiness, err = app.PeerRemovalReadiness(ctx, swarmionapp.PeerRemovalReadinessRequest{PeerID: peerID})
		if err != nil {
			return fmt.Errorf("read swarmion peer removal readiness for %s: %w", peerID, err)
		}
	}
	if err := PeerRemovalReadinessError(readiness); err != nil {
		notifyLog.Debugf("swarmion peer %s is not ready after eviction: %s", peerID, PeerRemovalReadinessSummary(readiness))
		return err
	}
	notifyLog.Debugf("swarmion peer %s is ready after eviction: %s", peerID, PeerRemovalReadinessSummary(readiness))
	return nil
}

func retryablePeerRemovalError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errSwarmionCheckpointCatchUpRetryable) || errors.Is(err, errSwarmionPeerRemovalNotReady) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "checkpoint target changed before catch-up") ||
		strings.Contains(message, "stale write context") ||
		strings.Contains(message, "replay-base conflict") ||
		strings.Contains(message, "conflicts with protocol root") ||
		strings.Contains(message, "context deadline exceeded")
}

func StrictPeerRemovalReadinessError(readiness swarmionapp.PeerRemovalReadinessResponse) error {
	return peerRemovalReadinessError(readiness, false)
}

func PeerRemovalReadinessError(readiness swarmionapp.PeerRemovalReadinessResponse) error {
	return peerRemovalReadinessError(readiness, true)
}

func PeerRemovalReadinessSummary(readiness swarmionapp.PeerRemovalReadinessResponse) string {
	peerID := strings.TrimSpace(readiness.PeerID)
	if peerID == "" {
		peerID = "peer"
	}
	flags := make([]string, 0, 5)
	if readiness.StillConnected {
		flags = append(flags, "connected")
	}
	if readiness.StillActiveViewPeer {
		flags = append(flags, "active-view")
	}
	if readiness.StillStateProvider {
		flags = append(flags, "state-provider")
	}
	if readiness.StillCheckpointProvider {
		flags = append(flags, "checkpoint-provider")
	}
	if readiness.StillContentProvider {
		flags = append(flags, "content-provider")
	}
	if len(flags) == 0 {
		flags = append(flags, "no-runtime-flags")
	}
	reason := strings.TrimSpace(readiness.BlockingReason)
	if reason == "" {
		reason = strings.TrimSpace(readiness.CheckpointProviderReason)
	}
	if reason == "" && len(readiness.RemainingObligations) > 0 {
		reason = strings.Join(readiness.RemainingObligations, "; ")
	}
	if reason == "" {
		reason = "none"
	}
	observed := "never"
	if !readiness.LastObservedAt.IsZero() {
		observed = readiness.LastObservedAt.Format(time.RFC3339)
	}
	return fmt.Sprintf(
		"peer=%s safe=%t flags=%s reason=%q obligations=%d last_observed=%s",
		peerID,
		readiness.SafeToRemoveDurableResource,
		strings.Join(flags, ","),
		reason,
		len(readiness.RemainingObligations),
		observed,
	)
}

func peerRemovalReadinessError(readiness swarmionapp.PeerRemovalReadinessResponse, allowStaleLocalObservation bool) error {
	if readiness.SafeToRemoveDurableResource {
		return nil
	}
	if allowStaleLocalObservation && peerRemovalReadinessOnlyStaleLocalObservation(readiness) {
		notifyLog.Debugf("treating stale swarmion local observation as non-blocking for removed peer %s: %s", readiness.PeerID, readiness.BlockingReason)
		return nil
	}
	reason := strings.TrimSpace(readiness.BlockingReason)
	if reason == "" {
		reason = strings.Join(readiness.RemainingObligations, "; ")
	}
	if reason == "" {
		reason = "runtime reports remaining peer obligations"
	}
	peerID := strings.TrimSpace(readiness.PeerID)
	if peerID == "" {
		peerID = "peer"
	}
	return fmt.Errorf("%w for %s: %s", errSwarmionPeerRemovalNotReady, peerID, reason)
}

func peerRemovalReadinessOnlyStaleLocalObservation(readiness swarmionapp.PeerRemovalReadinessResponse) bool {
	if peerRemovalReadinessOnlyStaleTransportObservation(readiness) {
		return true
	}
	return peerRemovalReadinessOnlyStaleCheckpointObservation(readiness)
}

func peerRemovalReadinessOnlyStaleTransportObservation(readiness swarmionapp.PeerRemovalReadinessResponse) bool {
	if !readiness.StillConnected && !readiness.StillActiveViewPeer {
		return false
	}
	if readiness.StillStateProvider || readiness.StillCheckpointProvider || readiness.StillContentProvider {
		return false
	}
	for _, obligation := range readiness.RemainingObligations {
		normalized := strings.ToLower(strings.TrimSpace(obligation))
		if normalized != "peer is still connected" && normalized != "peer is still in the active view" {
			return false
		}
	}
	return true
}

func peerRemovalReadinessOnlyStaleCheckpointObservation(readiness swarmionapp.PeerRemovalReadinessResponse) bool {
	if !readiness.StillCheckpointProvider {
		return false
	}
	if readiness.StillConnected || readiness.StillActiveViewPeer || readiness.StillStateProvider || readiness.StillContentProvider {
		return false
	}
	for _, obligation := range readiness.RemainingObligations {
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(obligation)), "peer still advertises checkpoint state") {
			return false
		}
	}
	return true
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
		if details := compatibilityBoundaryDetails(item); details != "" {
			reason += " (" + details + ")"
		}
		return fmt.Errorf("swarmion compatibility blocks replication reconciliation for peer %s: %s", item.PeerID, reason)
	}
	return nil
}

func compatibilityBoundaryDetails(item swarmionapp.ManifestCompatibility) string {
	var details []string
	if item.LocalInitialRootHash != "" || item.LocalInitialCommitID != "" {
		details = append(details, fmt.Sprintf("local initial root=%s commit=%s", item.LocalInitialRootHash, item.LocalInitialCommitID))
	}
	if item.RemoteInitialRootHash != "" || item.RemoteInitialCommitID != "" {
		details = append(details, fmt.Sprintf("remote initial root=%s commit=%s", item.RemoteInitialRootHash, item.RemoteInitialCommitID))
	}
	return strings.Join(details, "; ")
}

func prioritizedReplicationCandidates(candidates []ReplicationCandidate) []prioritizedReplicationCandidate {
	byPeer := make(map[string]prioritizedReplicationCandidate, len(candidates))
	for _, candidate := range candidates {
		peerID := strings.TrimSpace(candidate.PeerID)
		if peerID == "" {
			continue
		}
		deviceClass := normalizeReplicationDeviceClass(candidate.DeviceClass)
		priority := candidate.Priority
		if priority <= 0 {
			var found bool
			priority, found = ReplicationPriorityForDeviceClass(deviceClass)
			if !found {
				continue
			}
		}
		if priority <= 0 {
			continue
		}
		prioritized := prioritizedReplicationCandidate{
			PeerID:      peerID,
			DeviceClass: deviceClass,
			Priority:    priority,
		}
		if existing, found := byPeer[peerID]; !found || prioritized.Priority > existing.Priority {
			byPeer[peerID] = prioritized
		}
	}

	out := make([]prioritizedReplicationCandidate, 0, len(byPeer))
	for _, candidate := range byPeer {
		out = append(out, candidate)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority > out[j].Priority
		}
		return out[i].PeerID < out[j].PeerID
	})
	return out
}
