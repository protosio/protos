package db

import (
	"context"
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

	db.mu.Lock()
	app := db.app
	db.mu.Unlock()
	if app == nil {
		return nil
	}

	if _, err := app.CatchUpCheckpoint(ctx, "reconcile Protos replication metadata"); err != nil {
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
	notifyLog.Debugf(
		"swarmion checkpoint runtime has no replication-policy mutation API; retained %d Protos replication-priority candidates as local metadata",
		len(prioritized),
	)
	return nil
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
	for attempt := 0; attempt < 4; attempt++ {
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
		notifyLog.Debugf("retrying swarmion peer removal for %s after transient state change: %s", peerID, err.Error())
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
	return lastErr
}

func removeReplicationPeerStateWithApp(ctx context.Context, app *swarmionapp.App, peerID string) error {
	if _, err := app.CatchUpCheckpoint(ctx, "remove Protos replication peer state"); err != nil {
		return fmt.Errorf("catch up swarmion checkpoint state for replication peer removal: %w", err)
	}
	status := app.Status()
	if status.Fatal != nil {
		return fmt.Errorf("swarmion fatal state blocks replication peer removal: %s", status.Fatal.State)
	}
	if err := blockOnIncompatiblePeers(ctx, app); err != nil {
		return err
	}

	if _, err := app.EvictPeer(ctx, swarmionapp.PeerEvictionRequest{PeerID: peerID}); err != nil {
		return fmt.Errorf("evict swarmion peer %s after removal: %w", peerID, err)
	}
	if err := assertSwarmionPeerNotProvider(app.Status(), peerID); err != nil {
		return err
	}
	return nil
}

func retryablePeerRemovalError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "checkpoint target changed before catch-up") ||
		strings.Contains(message, "stale write context") ||
		strings.Contains(message, "replay-base conflict") ||
		strings.Contains(message, "conflicts with protocol root") ||
		strings.Contains(message, "context deadline exceeded")
}

func assertSwarmionPeerNotProvider(status swarmionapp.Status, peerID string) error {
	if setHas(stringSet(status.StateProviders), peerID) {
		return fmt.Errorf("swarmion peer %s is still a state provider after removal", peerID)
	}
	return nil
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
		return fmt.Errorf("swarmion compatibility blocks replication reconciliation for peer %s: %s", item.PeerID, reason)
	}
	return nil
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
