package swarmionlink

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	libp2pcontrol "github.com/libp2p/go-libp2p/core/control"
	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/nustiueudinastea/swarmion/transports"
)

var (
	// ErrPeerFenced means an ordinary/stale admission attempt cannot reopen an
	// explicit deletion fence. Reopening requires the separate application
	// ownership-transfer path.
	ErrPeerFenced = errors.New("peer has an explicit deletion fence")
	// ErrPeerDeletionLeaseActive means provider deletion currently owns the
	// exact peer generation. Another fence/finalize attempt must defer.
	ErrPeerDeletionLeaseActive = errors.New("peer deletion lease is active")
)

// RouteFence is Protos' application-owned physical and borrowed-link admission
// boundary. It is deliberately process-local: durable deletion authority comes
// from replicated application state and Swarmion's peer-drain contract, never
// from this cache. A fresh process nonce makes every restart generation unique.
//
// Before authoritative application membership has been loaded, the fence is
// permissive so an existing repository can bootstrap its SQL projection. Once
// ReconcileAdmittedPeers is called, unknown peers fail closed. Explicitly
// admitted bootstrap and provisioning peers remain usable alongside replicated
// active peers.
type RouteFence struct {
	mu sync.RWMutex

	bootNonce string
	next      uint64
	managed   bool
	admitted  map[peer.ID]struct{}
	explicit  map[peer.ID]string
	versions  map[peer.ID]string
	leases    map[peer.ID]string
	revisions map[peer.ID]uint64
	probes    map[peer.ID]uint64
	temporary map[peer.ID]map[uint64]struct{}
	tempNext  uint64

	tracker *routeTracker
	// Focused tests pause a computed tracker mutation to prove that a newer
	// fence revision cannot be undone by an older admission callback.
	beforeTrackerUpdateForTest func(peer.ID, uint64, bool)
}

// NewRouteFence creates one fence for one application-owned host lifecycle.
func NewRouteFence() (*RouteFence, error) {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, fmt.Errorf("create route-fence generation nonce: %w", err)
	}
	return &RouteFence{
		bootNonce: hex.EncodeToString(nonce[:]),
		admitted:  make(map[peer.ID]struct{}),
		explicit:  make(map[peer.ID]string),
		versions:  make(map[peer.ID]string),
		leases:    make(map[peer.ID]string),
		revisions: make(map[peer.ID]uint64),
		probes:    make(map[peer.ID]uint64),
		temporary: make(map[peer.ID]map[uint64]struct{}),
	}, nil
}

// IdentityProbe is a scoped exception for the outbound dial hooks only. Protos
// uses a random placeholder PeerID to authenticate an otherwise unknown
// provisioning endpoint and learn its real PeerID from ErrPeerIDMismatch. The
// placeholder never becomes admitted to Protos protocols or the Swarmion Link,
// and an explicit deletion fence always overrides the probe.
type IdentityProbe struct {
	fence  *RouteFence
	peerID peer.ID
	once   sync.Once
}

// BeginIdentityProbe permits the placeholder PeerID through InterceptPeerDial
// and InterceptAddrDial until the returned scope is closed. Secured connections
// remain subject to normal admission, so this cannot create an application or
// Swarmion route for an unknown identity.
func (f *RouteFence) BeginIdentityProbe(peerID peer.ID) (*IdentityProbe, error) {
	if f == nil {
		return nil, fmt.Errorf("route fence is nil")
	}
	if peerID == "" {
		return nil, fmt.Errorf("peer id is empty")
	}
	f.mu.Lock()
	if generation := f.explicit[peerID]; generation != "" {
		f.mu.Unlock()
		return nil, fmt.Errorf("%w for peer %s generation %s", ErrPeerFenced, peerID, generation)
	}
	f.probes[peerID]++
	f.mu.Unlock()
	return &IdentityProbe{fence: f, peerID: peerID}, nil
}

// Close revokes the dial-only exception. It is safe to call more than once.
func (p *IdentityProbe) Close() {
	if p == nil || p.fence == nil || p.peerID == "" {
		return
	}
	p.once.Do(func() {
		p.fence.mu.Lock()
		if count := p.fence.probes[p.peerID]; count > 1 {
			p.fence.probes[p.peerID] = count - 1
		} else {
			delete(p.fence.probes, p.peerID)
		}
		p.fence.mu.Unlock()
	})
}

// TemporaryPeerAdmission is a tokenized, physical-only allowance for a real
// PeerID learned by an identity probe. It permits the authenticated follow-up
// connection without exposing that peer to Protos protocols or the Swarmion
// Link. Promote converts only this exact token into permanent admission after
// the caller validates the peer's application-supported public key.
type TemporaryPeerAdmission struct {
	mu     sync.Mutex
	fence  *RouteFence
	peerID peer.ID
	token  uint64
	done   bool
}

// BeginTemporaryPeerAdmission opens a scoped physical allowance for peerID.
// Concurrent scopes are independent and ref-counted by opaque token. An
// explicit deletion fence always rejects a new scope.
func (f *RouteFence) BeginTemporaryPeerAdmission(peerID peer.ID) (*TemporaryPeerAdmission, error) {
	if f == nil {
		return nil, fmt.Errorf("route fence is nil")
	}
	if peerID == "" {
		return nil, fmt.Errorf("peer id is empty")
	}
	f.mu.Lock()
	if generation := f.explicit[peerID]; generation != "" {
		f.mu.Unlock()
		return nil, fmt.Errorf("%w for peer %s generation %s", ErrPeerFenced, peerID, generation)
	}
	f.tempNext++
	token := f.tempNext
	if f.temporary[peerID] == nil {
		f.temporary[peerID] = make(map[uint64]struct{})
	}
	f.temporary[peerID][token] = struct{}{}
	f.mu.Unlock()
	return &TemporaryPeerAdmission{fence: f, peerID: peerID, token: token}, nil
}

// Promote permanently admits the peer and consumes this exact temporary
// token. It fails closed if deletion fenced the peer after the scope began.
func (a *TemporaryPeerAdmission) Promote() error {
	if a == nil || a.fence == nil || a.peerID == "" || a.token == 0 {
		return fmt.Errorf("temporary peer admission is invalid")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.done {
		return fmt.Errorf("temporary peer admission is already closed")
	}

	f := a.fence
	f.mu.Lock()
	tokens := f.temporary[a.peerID]
	if _, exists := tokens[a.token]; !exists {
		f.mu.Unlock()
		a.done = true
		return fmt.Errorf("temporary peer admission token is no longer active")
	}
	delete(tokens, a.token)
	if len(tokens) == 0 {
		delete(f.temporary, a.peerID)
	}
	if generation := f.explicit[a.peerID]; generation != "" {
		revision := f.bumpRevisionLocked(a.peerID)
		tracker := f.tracker
		f.mu.Unlock()
		a.done = true
		f.updateTracker(tracker, a.peerID, revision, true)
		return fmt.Errorf("%w for peer %s generation %s", ErrPeerFenced, a.peerID, generation)
	}
	_, wasAdmitted := f.admitted[a.peerID]
	f.admitted[a.peerID] = struct{}{}
	if !wasAdmitted {
		f.next++
		f.versions[a.peerID] = fmt.Sprintf("%s-%d", f.bootNonce, f.next)
	}
	revision := f.bumpRevisionLocked(a.peerID)
	tracker := f.tracker
	f.mu.Unlock()
	a.done = true
	f.updateTracker(tracker, a.peerID, revision, false)
	return nil
}

// Close revokes only this scope's token. It never removes a permanent
// admission or another concurrent temporary scope.
func (a *TemporaryPeerAdmission) Close() {
	if a == nil || a.fence == nil || a.peerID == "" || a.token == 0 {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.done {
		return
	}
	a.fence.mu.Lock()
	tokens := a.fence.temporary[a.peerID]
	delete(tokens, a.token)
	if len(tokens) == 0 {
		delete(a.fence.temporary, a.peerID)
	}
	a.fence.mu.Unlock()
	a.done = true
}

// FencePeer atomically rejects every future physical/link route for peer and
// returns the new opaque generation that must be used for one Swarmion drain.
// Repeating FencePeer intentionally allocates another generation so recovery
// can never finalize an earlier process or attempt's lease.
func (f *RouteFence) FencePeer(peerID peer.ID) (string, error) {
	if f == nil {
		return "", fmt.Errorf("route fence is nil")
	}
	if peerID == "" {
		return "", fmt.Errorf("peer id is empty")
	}
	f.mu.Lock()
	if activeGeneration := f.leases[peerID]; activeGeneration != "" {
		f.mu.Unlock()
		return "", fmt.Errorf("%w for peer %s generation %s", ErrPeerDeletionLeaseActive, peerID, activeGeneration)
	}
	f.next++
	generation := fmt.Sprintf("%s-%d", f.bootNonce, f.next)
	f.explicit[peerID] = generation
	f.versions[peerID] = generation
	revision := f.bumpRevisionLocked(peerID)
	tracker := f.tracker
	f.mu.Unlock()
	f.updateTracker(tracker, peerID, revision, true)
	return generation, nil
}

// AdmitPeer explicitly permits a provisioning/bootstrap/active peer that does
// not have a deletion fence. It intentionally cannot clear an existing fence:
// a stale membership snapshot must never reopen a peer between drain finalize
// and provider destruction.
func (f *RouteFence) AdmitPeer(peerID peer.ID) error {
	if f == nil {
		return fmt.Errorf("route fence is nil")
	}
	if peerID == "" {
		return fmt.Errorf("peer id is empty")
	}
	f.mu.Lock()
	if generation := f.explicit[peerID]; generation != "" {
		f.mu.Unlock()
		return fmt.Errorf("%w for peer %s generation %s", ErrPeerFenced, peerID, generation)
	}
	_, wasAdmitted := f.admitted[peerID]
	f.admitted[peerID] = struct{}{}
	if !wasAdmitted {
		f.next++
		f.versions[peerID] = fmt.Sprintf("%s-%d", f.bootNonce, f.next)
	}
	blocked := f.blockedLocked(peerID)
	revision := f.bumpRevisionLocked(peerID)
	tracker := f.tracker
	f.mu.Unlock()
	if !blocked {
		f.updateTracker(tracker, peerID, revision, false)
	}
	return nil
}

// ReopenPeer is the explicit ownership-transfer/recreation boundary. Ordinary
// ConfigurePeers, pending admission, and bootstrap admission must never call
// it. It is rejected while deletion owns the peer generation.
func (f *RouteFence) ReopenPeer(peerID peer.ID) error {
	if f == nil {
		return fmt.Errorf("route fence is nil")
	}
	if peerID == "" {
		return fmt.Errorf("peer id is empty")
	}
	f.mu.Lock()
	if generation := f.leases[peerID]; generation != "" {
		f.mu.Unlock()
		return fmt.Errorf("%w for peer %s generation %s", ErrPeerDeletionLeaseActive, peerID, generation)
	}
	_, wasExplicit := f.explicit[peerID]
	_, wasAdmitted := f.admitted[peerID]
	delete(f.explicit, peerID)
	f.admitted[peerID] = struct{}{}
	if wasExplicit || !wasAdmitted {
		f.next++
		f.versions[peerID] = fmt.Sprintf("%s-%d", f.bootNonce, f.next)
	}
	blocked := f.blockedLocked(peerID)
	revision := f.bumpRevisionLocked(peerID)
	tracker := f.tracker
	f.mu.Unlock()
	if !blocked {
		f.updateTracker(tracker, peerID, revision, false)
	}
	return nil
}

// ReconcileAdmittedPeers replaces the authoritative replicated active set.
// temporaryPeers are application-known bootstrap or provisioning peers and are
// retained separately by the caller. Every other peer becomes unable to dial,
// accept streams, or appear in the borrowed Link route projection.
func (f *RouteFence) ReconcileAdmittedPeers(activePeers, temporaryPeers []peer.ID) {
	if f == nil {
		return
	}
	nextAdmitted := make(map[peer.ID]struct{}, len(activePeers)+len(temporaryPeers))
	for _, peerID := range append(append([]peer.ID(nil), activePeers...), temporaryPeers...) {
		if peerID != "" {
			nextAdmitted[peerID] = struct{}{}
		}
	}

	f.mu.Lock()
	previouslyManaged := f.managed
	previousAdmitted := f.admitted
	f.managed = true
	f.admitted = nextAdmitted
	type trackerUpdate struct {
		blocked  bool
		revision uint64
	}
	changed := make(map[peer.ID]trackerUpdate, len(previousAdmitted)+len(nextAdmitted))
	for peerID := range previousAdmitted {
		changed[peerID] = trackerUpdate{blocked: f.blockedLocked(peerID), revision: f.bumpRevisionLocked(peerID)}
	}
	for peerID := range nextAdmitted {
		changed[peerID] = trackerUpdate{blocked: f.blockedLocked(peerID), revision: f.bumpRevisionLocked(peerID)}
	}
	// On the first authoritative reconciliation, currently connected unknown
	// peers are withdrawn by the tracker scan even though they did not occur in
	// the previous explicit admission map.
	tracker := f.tracker
	f.mu.Unlock()

	if tracker == nil {
		return
	}
	if !previouslyManaged {
		tracker.reconcileFences()
		return
	}
	for peerID, update := range changed {
		f.updateTracker(tracker, peerID, update.revision, update.blocked)
	}
}

func (f *RouteFence) bumpRevisionLocked(peerID peer.ID) uint64 {
	f.revisions[peerID]++
	return f.revisions[peerID]
}

func (f *RouteFence) trackerState(peerID peer.ID) (bool, uint64) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.blockedLocked(peerID), f.revisions[peerID]
}

func (f *RouteFence) updateTracker(tracker *routeTracker, peerID peer.ID, revision uint64, blocked bool) {
	if tracker == nil {
		return
	}
	if hook := f.beforeTrackerUpdateForTest; hook != nil {
		hook(peerID, revision, blocked)
	}
	tracker.setPeerFencedRevision(peerID, revision, blocked)
}

// IsPeerFenced reports whether the application currently rejects peer.
func (f *RouteFence) IsPeerFenced(peerID peer.ID) bool {
	if f == nil || peerID == "" {
		return false
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.blockedLocked(peerID)
}

// IsPeerConnectionAllowed reports whether the physical libp2p host may keep
// or establish a connection. Temporary learned-ID scopes affect only this
// plane; IsPeerFenced remains true until permanent promotion so application and
// Swarmion protocol dispatch stay fail closed.
func (f *RouteFence) IsPeerConnectionAllowed(peerID peer.ID) bool {
	if f == nil || peerID == "" {
		return false
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.connectionAllowedLocked(peerID)
}

// GenerationMatches reports whether generation is the current explicit fence
// lease. It does not establish durable deletion safety.
func (f *RouteFence) GenerationMatches(peerID peer.ID, generation string) bool {
	if f == nil || peerID == "" || generation == "" {
		return false
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.explicit[peerID] == generation && f.versions[peerID] == generation && f.blockedLocked(peerID)
}

// LinkRouteWithdrawn reports whether the borrowed Link no longer exposes a
// route for peer. Fence establishment uses this as a synchronous postcondition
// before Swarmion is told that the application route is closed.
func (f *RouteFence) LinkRouteWithdrawn(peerID peer.ID) bool {
	if f == nil || peerID == "" {
		return false
	}
	f.mu.RLock()
	tracker := f.tracker
	fenced := f.blockedLocked(peerID)
	f.mu.RUnlock()
	return fenced && tracker != nil && !tracker.hasPeerRoute(peerID)
}

// WithGeneration acquires a peer-scoped logical deletion lease. It does not
// retain f.mu while fn performs potentially long provider I/O; ordinary and
// stale admission attempts observe the lease/explicit fence and fail promptly.
// The explicit fence remains after the lease ends and can only be cleared by
// ReopenPeer.
func (f *RouteFence) WithGeneration(
	ctx context.Context,
	peerID peer.ID,
	generation string,
	fn func() error,
) error {
	if f == nil || peerID == "" || generation == "" || fn == nil {
		return fmt.Errorf("invalid route-fence generation guard")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	if f.explicit[peerID] != generation || f.versions[peerID] != generation || !f.blockedLocked(peerID) {
		f.mu.Unlock()
		return fmt.Errorf("route-fence generation changed for peer %s", peerID)
	}
	if activeGeneration := f.leases[peerID]; activeGeneration != "" {
		f.mu.Unlock()
		return fmt.Errorf("%w for peer %s generation %s", ErrPeerDeletionLeaseActive, peerID, activeGeneration)
	}
	f.leases[peerID] = generation
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		if f.leases[peerID] == generation {
			delete(f.leases, peerID)
		}
		f.mu.Unlock()
	}()
	if err := ctx.Err(); err != nil {
		return err
	}
	return fn()
}

func (f *RouteFence) blockedLocked(peerID peer.ID) bool {
	if _, fenced := f.explicit[peerID]; fenced {
		return true
	}
	if !f.managed {
		return false
	}
	_, admitted := f.admitted[peerID]
	return !admitted
}

func (f *RouteFence) connectionAllowedLocked(peerID peer.ID) bool {
	if _, explicitlyFenced := f.explicit[peerID]; explicitlyFenced {
		return false
	}
	return !f.blockedLocked(peerID) || len(f.temporary[peerID]) > 0
}

func (f *RouteFence) attachTracker(tracker *routeTracker) {
	if f == nil {
		return
	}
	f.mu.Lock()
	f.tracker = tracker
	f.mu.Unlock()
}

// libp2p ConnectionGater implementation. The unauthenticated accept hook
// cannot identify a peer; the secured and upgraded hooks enforce the fence.
func (f *RouteFence) InterceptPeerDial(peerID peer.ID) bool {
	return f.peerDialAllowed(peerID)
}

func (f *RouteFence) InterceptAddrDial(peerID peer.ID, _ ma.Multiaddr) bool {
	return f.peerDialAllowed(peerID)
}

func (f *RouteFence) peerDialAllowed(peerID peer.ID) bool {
	if f == nil || peerID == "" {
		return false
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	// Explicit deletion is the physical safety boundary and wins over an
	// identity probe that began just before the fence was established.
	if _, explicitlyFenced := f.explicit[peerID]; explicitlyFenced {
		return false
	}
	if !f.connectionAllowedLocked(peerID) {
		return f.probes[peerID] > 0
	}
	return true
}

func (f *RouteFence) InterceptAccept(libp2pnetwork.ConnMultiaddrs) bool { return true }

func (f *RouteFence) InterceptSecured(_ libp2pnetwork.Direction, peerID peer.ID, _ libp2pnetwork.ConnMultiaddrs) bool {
	return f.IsPeerConnectionAllowed(peerID)
}

func (f *RouteFence) InterceptUpgraded(connection libp2pnetwork.Conn) (bool, libp2pcontrol.DisconnectReason) {
	if connection == nil || f.IsPeerConnectionAllowed(connection.RemotePeer()) {
		return true, 0
	}
	return false, 0
}

func policyDenied(peerID transports.PeerID, operation string) error {
	err := transports.NewError(
		transports.ErrorCodePolicyDenied,
		operation,
		transports.ErrPolicyDenied,
	)
	err.Peer = peerID
	return err
}
