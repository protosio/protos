package swarmionlink

import (
	"context"
	"sync"

	libp2phost "github.com/libp2p/go-libp2p/core/host"
	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	libp2ppeer "github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/nustiueudinastea/swarmion/transports"
)

const observationQueueSize = 64

// routeTracker is the Link-lifetime route authority. Generations must outlive
// one observation subscription so resubscription cannot revive stale leases.
// It only observes the caller-owned host and never changes a connection.
type routeTracker struct {
	host  libp2phost.Host
	fence *RouteFence

	mu            sync.Mutex
	revision      uint64
	connections   map[transports.PeerID]map[string]transports.RouteKind
	generations   map[transports.PeerID]uint64
	fenceRevision map[transports.PeerID]uint64
	routes        map[transports.PeerID]transports.RouteState
	subscriptions map[*routeSubscription]struct{}
}

type routeSubscription struct {
	tracker *routeTracker

	mu       sync.Mutex
	closed   bool
	terminal error
	events   chan transports.RouteEvent
	done     chan struct{}
}

func newRouteTracker(host libp2phost.Host, fence *RouteFence) *routeTracker {
	tracker := &routeTracker{
		host:          host,
		fence:         fence,
		connections:   make(map[transports.PeerID]map[string]transports.RouteKind),
		generations:   make(map[transports.PeerID]uint64),
		fenceRevision: make(map[transports.PeerID]uint64),
		routes:        make(map[transports.PeerID]transports.RouteState),
		subscriptions: make(map[*routeSubscription]struct{}),
	}

	// Register before scanning. Callbacks and the scan share the tracker mutex
	// and connection IDs, so connection changes cannot be lost or counted twice.
	host.Network().Notify(tracker)
	tracker.mu.Lock()
	for _, connection := range host.Network().Conns() {
		tracker.addConnectionLocked(connection)
	}
	tracker.mu.Unlock()
	return tracker
}

func (t *routeTracker) subscribe() (transports.RouteSnapshot, transports.RouteSubscription, error) {
	if t == nil || t.host == nil {
		return transports.RouteSnapshot{}, nil, transports.NewError(
			transports.ErrorCodeClosedLink,
			"subscribe routes",
			transports.ErrClosedLink,
		)
	}
	sub := &routeSubscription{
		tracker: t,
		events:  make(chan transports.RouteEvent, observationQueueSize),
		done:    make(chan struct{}),
	}
	t.mu.Lock()
	snapshot := t.snapshotLocked()
	t.subscriptions[sub] = struct{}{}
	t.mu.Unlock()
	return snapshot, sub, nil
}

func (t *routeTracker) Listen(libp2pnetwork.Network, ma.Multiaddr) {}

func (t *routeTracker) ListenClose(libp2pnetwork.Network, ma.Multiaddr) {}

func (t *routeTracker) Connected(_ libp2pnetwork.Network, connection libp2pnetwork.Conn) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.addConnectionLocked(connection)
}

func (t *routeTracker) Disconnected(_ libp2pnetwork.Network, connection libp2pnetwork.Conn) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.removeConnectionLocked(connection)
}

func (t *routeTracker) addConnectionLocked(connection libp2pnetwork.Conn) {
	if connection == nil || connection.IsClosed() {
		return
	}
	if t.fence != nil && t.fence.IsPeerFenced(connection.RemotePeer()) {
		return
	}
	kind := transports.RouteKindDirect
	if connection.Stat().Limited {
		kind = transports.RouteKindRelayed
	}
	t.addRouteLocked(transports.PeerID(connection.RemotePeer().String()), connection.ID(), kind)
}

// setPeerFenced changes only the borrowed Link projection. The application
// separately closes physical connections after fencing; late disconnect
// callbacks are therefore harmless no-ops.
func (t *routeTracker) setPeerFencedRevision(peerID libp2ppeer.ID, revision uint64, fenced bool) {
	if t == nil || t.host == nil || peerID == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	transportPeer := transports.PeerID(peerID.String())
	if revision < t.fenceRevision[transportPeer] {
		return
	}
	t.fenceRevision[transportPeer] = revision
	if fenced {
		connections := t.connections[transportPeer]
		if len(connections) == 0 {
			return
		}
		route := t.routes[transportPeer]
		delete(t.connections, transportPeer)
		delete(t.routes, transportPeer)
		t.revision++
		t.emitLocked(transports.RouteEvent{Revision: t.revision, Peer: transportPeer, Reachable: false, Route: route})
		return
	}
	for _, connection := range t.host.Network().ConnsToPeer(peerID) {
		t.addConnectionLocked(connection)
	}
}

func (t *routeTracker) hasPeerRoute(peerID libp2ppeer.ID) bool {
	if t == nil || peerID == "" {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.connections[transports.PeerID(peerID.String())]) > 0
}

func (t *routeTracker) reconcileFences() {
	if t == nil || t.host == nil {
		return
	}
	peers := make(map[libp2ppeer.ID]struct{})
	for _, connection := range t.host.Network().Conns() {
		if connection != nil {
			peers[connection.RemotePeer()] = struct{}{}
		}
	}
	for peerID := range peers {
		blocked, revision := false, uint64(0)
		if t.fence != nil {
			blocked, revision = t.fence.trackerState(peerID)
		}
		t.setPeerFencedRevision(peerID, revision, blocked)
	}
}

func (t *routeTracker) addRouteLocked(
	peerID transports.PeerID,
	connectionID string,
	kind transports.RouteKind,
) {
	connections := t.connections[peerID]
	wasReachable := len(connections) > 0
	if connections == nil {
		connections = make(map[string]transports.RouteKind)
		t.connections[peerID] = connections
	}
	if _, exists := connections[connectionID]; exists {
		return
	}
	connections[connectionID] = kind
	preferred := preferredRouteKind(connections)
	if wasReachable {
		current := t.routes[peerID]
		if current.Kind == preferred {
			return
		}
		current.Kind = preferred
		t.routes[peerID] = current
		t.revision++
		t.emitLocked(transports.RouteEvent{
			Revision: t.revision, Peer: peerID, Reachable: true, Route: current,
		})
		return
	}

	t.generations[peerID]++
	route := transports.RouteState{
		Peer: peerID, Generation: t.generations[peerID], Kind: preferred,
	}
	t.routes[peerID] = route
	t.revision++
	t.emitLocked(transports.RouteEvent{
		Revision:  t.revision,
		Peer:      peerID,
		Reachable: true,
		Route:     route,
	})
}

func (t *routeTracker) removeConnectionLocked(connection libp2pnetwork.Conn) {
	if connection == nil {
		return
	}
	t.removeRouteLocked(transports.PeerID(connection.RemotePeer().String()), connection.ID())
}

func (t *routeTracker) removeRouteLocked(peerID transports.PeerID, connectionID string) {
	connections := t.connections[peerID]
	if _, exists := connections[connectionID]; !exists {
		return
	}
	delete(connections, connectionID)
	if len(connections) > 0 {
		current := t.routes[peerID]
		preferred := preferredRouteKind(connections)
		if current.Kind != preferred {
			current.Kind = preferred
			t.routes[peerID] = current
			t.revision++
			t.emitLocked(transports.RouteEvent{
				Revision: t.revision, Peer: peerID, Reachable: true, Route: current,
			})
		}
		return
	}
	delete(t.connections, peerID)
	route := t.routes[peerID]
	delete(t.routes, peerID)
	t.revision++
	t.emitLocked(transports.RouteEvent{
		Revision:  t.revision,
		Peer:      peerID,
		Reachable: false,
		Route:     route,
	})
}

func preferredRouteKind(connections map[string]transports.RouteKind) transports.RouteKind {
	for _, kind := range connections {
		if kind == transports.RouteKindDirect {
			return transports.RouteKindDirect
		}
	}
	for _, kind := range connections {
		if kind == transports.RouteKindRelayed {
			return transports.RouteKindRelayed
		}
	}
	return transports.RouteKindDirect
}

func (t *routeTracker) emitLocked(event transports.RouteEvent) {
	for sub := range t.subscriptions {
		if sub.enqueue(event) {
			continue
		}
		delete(t.subscriptions, sub)
	}
}

func (t *routeTracker) snapshotLocked() transports.RouteSnapshot {
	routes := make(map[transports.PeerID]transports.RouteState, len(t.routes))
	for peerID, route := range t.routes {
		routes[peerID] = route
	}
	return transports.RouteSnapshot{Revision: t.revision, Routes: routes}
}

func (t *routeTracker) unsubscribe(sub *routeSubscription, err error) {
	if t == nil || sub == nil {
		return
	}
	t.mu.Lock()
	delete(t.subscriptions, sub)
	sub.fail(err)
	t.mu.Unlock()
}

func (s *routeSubscription) enqueue(event transports.RouteEvent) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	select {
	case s.events <- event:
		return true
	default:
		s.failLocked(transports.NewError(
			transports.ErrorCodeObservationOverflow,
			"observe routes",
			transports.ErrObservationOverflow,
		))
		return false
	}
}

func (s *routeSubscription) Next(ctx context.Context) (transports.RouteEvent, error) {
	if err := s.terminalError(); err != nil {
		return transports.RouteEvent{}, err
	}
	select {
	case event := <-s.events:
		if err := s.terminalError(); err != nil {
			return transports.RouteEvent{}, err
		}
		return event, nil
	case <-s.done:
		return transports.RouteEvent{}, s.terminalError()
	case <-ctx.Done():
		return transports.RouteEvent{}, ctx.Err()
	}
}

func (s *routeSubscription) Close(_ context.Context) error {
	if s == nil {
		return nil
	}
	closedErr := transports.NewError(
		transports.ErrorCodeClosedLink,
		"observe routes",
		transports.ErrClosedLink,
	)
	if s.tracker != nil {
		s.tracker.unsubscribe(s, closedErr)
	} else {
		s.fail(closedErr)
	}
	return nil
}

func (s *routeSubscription) terminalError() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.terminal
}

func (s *routeSubscription) fail(err error) {
	s.mu.Lock()
	s.failLocked(err)
	s.mu.Unlock()
}

func (s *routeSubscription) failLocked(err error) {
	if s.closed {
		return
	}
	s.closed = true
	s.terminal = err
	close(s.done)
}

var (
	_ libp2pnetwork.Notifiee       = (*routeTracker)(nil)
	_ transports.RouteSubscription = (*routeSubscription)(nil)
)
