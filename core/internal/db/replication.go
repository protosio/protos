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

// ErrReplicationPeerDrainPending means a generation-matched drain has not yet
// established its local coverage/no-post-fence-heartbeat prerequisites. It is
// a bounded defer signal, never proof that the peer or resource may be removed.
var (
	ErrReplicationPeerDrainPending     = errors.New("swarmion peer drain pending")
	ErrReplicationPeerDrainUnavailable = errors.New("swarmion peer drain unavailable")
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

type swarmionPeerDrainDiagnostics interface {
	Status() (swarmionapp.Status, error)
	Peers() ([]swarmionapp.PeerInfo, error)
	PeerStatus(context.Context) ([]swarmionapp.PeerStatus, error)
	Compatibility(context.Context) ([]swarmionapp.ManifestCompatibility, error)
}

type swarmionCheckpointMaintenance interface {
	ReconcileCheckpoint(context.Context, swarmionapp.CheckpointReconcileRequest) (swarmionapp.CheckpointReconcileResult, error)
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
	app := db.runtime
	db.mu.Unlock()
	if app == nil {
		return nil
	}

	diagnostics := app.Diagnostics()
	status, err := diagnostics.Status()
	if err != nil {
		return fmt.Errorf("read Swarmion status for replication reconciliation: %w", err)
	}
	if status.Fatal != nil {
		return fmt.Errorf("swarmion fatal state blocks replication metadata reconciliation: %s", status.Fatal.State)
	}
	if err := blockOnIncompatiblePeers(ctx, diagnostics); err != nil {
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

// PrepareReplicationPeerDrain establishes local replacement checkpoint state
// while the target peer is still usable. It never fences or clears peer state.
func (db *DB) PrepareReplicationPeerDrain(ctx context.Context, peerID string, candidates []ReplicationCandidate) error {
	peerID = strings.TrimSpace(peerID)
	if db == nil {
		return fmt.Errorf("%w: database is nil", ErrReplicationPeerDrainUnavailable)
	}
	if peerID == "" {
		return fmt.Errorf("peer id is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	db.mu.Lock()
	app := db.runtime
	runtimeOpenedAt := db.runtimeOpenedAt
	db.mu.Unlock()
	if app == nil {
		return fmt.Errorf("%w: database runtime is not initialized", ErrReplicationPeerDrainUnavailable)
	}
	return prepareReplicationPeerDrainWithRuntime(ctx, app.Diagnostics(), app.Maintenance(), peerID, candidates, runtimeOpenedAt)
}

func prepareReplicationPeerDrainWithRuntime(
	ctx context.Context,
	diagnostics swarmionPeerDrainDiagnostics,
	maintenance swarmionCheckpointMaintenance,
	peerID string,
	candidates []ReplicationCandidate,
	runtimeOpenedAt time.Time,
) error {
	if diagnostics == nil || maintenance == nil {
		return fmt.Errorf("%w: database runtime is not initialized", ErrReplicationPeerDrainUnavailable)
	}
	if err := catchUpSwarmionCheckpoint(ctx, maintenance, "prepare Protos replication peer drain"); err != nil {
		return fmt.Errorf("catch up swarmion checkpoint state before draining peer %s: %w", peerID, err)
	}
	status, err := diagnostics.Status()
	if err != nil {
		return fmt.Errorf("%w for peer %s: read Swarmion status: %w", ErrReplicationPeerDrainUnavailable, peerID, err)
	}
	if status.Fatal != nil {
		return fmt.Errorf("swarmion fatal state blocks peer drain for %s: %s", peerID, status.Fatal.State)
	}
	if err := blockOnIncompatiblePeers(ctx, diagnostics); err != nil {
		return err
	}
	prioritized := prioritizedReplicationCandidates(candidates)
	if len(prioritized) == 0 {
		return fmt.Errorf("%w for peer %s: no surviving replication candidate", ErrReplicationPeerDrainPending, peerID)
	}

	peerStatuses, err := diagnostics.PeerStatus(ctx)
	if err != nil {
		return fmt.Errorf("read swarmion peer status before draining peer %s: %w", peerID, err)
	}
	peerStatusByID := make(map[string]swarmionapp.PeerStatus, len(peerStatuses))
	for _, item := range peerStatuses {
		peerStatusByID[strings.TrimSpace(item.PeerID)] = item
	}
	peerInfoByID := make(map[string]swarmionapp.PeerInfo)
	peers, err := diagnostics.Peers()
	if err != nil {
		return fmt.Errorf("%w for peer %s: read Swarmion peers: %w", ErrReplicationPeerDrainUnavailable, peerID, err)
	}
	for _, item := range peers {
		peerInfoByID[strings.TrimSpace(string(item.ID))] = item
	}

	// A target heartbeat received by this runtime lifecycle is required before
	// fencing. Persisted observations from a prior process never establish that
	// the target was usable during replacement preparation.
	target, observed := peerInfoByID[peerID]
	if !observed || target.LastSeenUnixMillis <= 0 ||
		(!runtimeOpenedAt.IsZero() && target.LastSeenUnixMillis < runtimeOpenedAt.UnixMilli()) {
		return fmt.Errorf("%w for peer %s: target has not been freshly observed by this runtime", ErrReplicationPeerDrainPending, peerID)
	}

	localCommit := status.DurableMainCommitID
	localRoot := status.DurableMainRootHash
	checkpointCurrent := !localCommit.IsZero() && localRoot != (swarmionprotocol.RootHash{}) &&
		localCommit == status.CheckpointCommitID && localRoot == status.CheckpointRootHash
	if !checkpointCurrent {
		return fmt.Errorf(
			"%w for peer %s: local durable checkpoint is not current (durable=%s/%s checkpoint=%s/%s)",
			ErrReplicationPeerDrainPending,
			peerID,
			localCommit,
			localRoot,
			status.CheckpointCommitID,
			status.CheckpointRootHash,
		)
	}

	for _, candidate := range prioritized {
		if candidate.PeerID == status.PeerID {
			return nil
		}
		info, hasInfo := peerInfoByID[candidate.PeerID]
		peerStatus, hasStatus := peerStatusByID[candidate.PeerID]
		if !hasInfo || !hasStatus || !info.Participating || !peerStatus.Participating ||
			!peerStatus.Routed || !peerStatus.Compatible || peerStatus.Incompatible {
			continue
		}
		if info.LastSeenUnixMillis <= 0 ||
			(!runtimeOpenedAt.IsZero() && info.LastSeenUnixMillis < runtimeOpenedAt.UnixMilli()) {
			continue
		}
		if info.CheckpointCommitID == localCommit && info.CheckpointRootHash == localRoot {
			return nil
		}
	}
	return fmt.Errorf(
		"%w for peer %s: no surviving candidate currently advertises durable checkpoint %s/%s",
		ErrReplicationPeerDrainPending,
		peerID,
		localCommit,
		localRoot,
	)
}

// StartReplicationPeerDrain begins one public SDK session bound to the exact
// application-owned route-fence token. The returned session owns its passive
// event forwarding; callers must Close it before installing a replacement
// fence.
func (db *DB) StartReplicationPeerDrain(ctx context.Context, peerID, routeFenceToken string) (*swarmionapp.PeerDrainSession, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	app, req, err := db.peerDrainRuntime(peerID, routeFenceToken)
	if err != nil {
		return nil, err
	}
	session, err := app.StartPeerDrain(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("start swarmion peer drain for %s: %w", req.PeerID, err)
	}
	if session == nil {
		return nil, fmt.Errorf("start swarmion peer drain for %s returned a nil session", req.PeerID)
	}
	return session, nil
}

func (db *DB) peerDrainRuntime(peerID, routeFenceToken string) (*swarmionapp.DatabaseRuntime, swarmionapp.PeerDrainRequest, error) {
	// PeerID is canonical and RouteFenceToken is an opaque byte-exact identity.
	// Do not trim or canonicalize either value before the public SDK validates
	// the request.
	req := swarmionapp.PeerDrainRequest{PeerID: peerID, RouteFenceToken: routeFenceToken}
	if db == nil {
		return nil, req, fmt.Errorf("%w: database is nil", ErrReplicationPeerDrainUnavailable)
	}
	if err := req.Validate(); err != nil {
		return nil, req, fmt.Errorf("invalid peer drain request: %w", err)
	}
	db.mu.Lock()
	app := db.runtime
	db.mu.Unlock()
	if app == nil {
		return nil, req, fmt.Errorf("%w: database runtime is not initialized", ErrReplicationPeerDrainUnavailable)
	}
	return app, req, nil
}

func PeerDrainStatusSummary(status swarmionapp.PeerDrainSnapshot) string {
	peerID := strings.TrimSpace(status.PeerID)
	if peerID == "" {
		peerID = "peer"
	}
	reason := strings.Join(status.BlockingReasons, "; ")
	if reason == "" {
		reason = "none"
	}
	reasonCodes := make([]string, 0, len(status.BlockingReasonCodes))
	for _, code := range status.BlockingReasonCodes {
		if code := strings.TrimSpace(string(code)); code != "" {
			reasonCodes = append(reasonCodes, code)
		}
	}
	blockingReasonCodes := strings.Join(reasonCodes, "; ")
	if blockingReasonCodes == "" {
		blockingReasonCodes = "none"
	}
	checkpointCoverageReasonCode := strings.TrimSpace(string(status.CheckpointCoverageReasonCode))
	if checkpointCoverageReasonCode == "" {
		checkpointCoverageReasonCode = "none"
	}
	return fmt.Sprintf(
		"peer=%s generation=%s active=%t finalized=%t generation_matches=%t checkpoint_covered=%t checkpoint_coverage_reason_code=%q pre_fence_heartbeat_observed=%t post_fence_heartbeat_accepted=%t ingress_fence_sequence=%d ready_to_finalize=%t blocking_reason_codes=%q reason=%q",
		peerID,
		status.RouteGeneration,
		status.Active,
		status.Finalized,
		status.RouteGenerationMatches,
		status.LocalCheckpointCovered,
		checkpointCoverageReasonCode,
		status.PreFenceHeartbeatIngressObserved,
		status.PostFenceHeartbeatAccepted,
		status.HeartbeatIngressFenceSequence,
		status.ReadyToFinalize,
		blockingReasonCodes,
		reason,
	)
}

func blockOnIncompatiblePeers(ctx context.Context, app interface {
	Compatibility(context.Context) ([]swarmionapp.ManifestCompatibility, error)
}) error {
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
