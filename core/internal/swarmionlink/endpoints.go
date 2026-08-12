package swarmionlink

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"sync"

	libp2pevent "github.com/libp2p/go-libp2p/core/event"
	libp2phost "github.com/libp2p/go-libp2p/core/host"
	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2peventbus "github.com/libp2p/go-libp2p/p2p/host/eventbus"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/nustiueudinastea/swarmion/transports"
)

type endpointSubscription struct {
	host libp2phost.Host

	mu            sync.Mutex
	initializing  bool
	closed        bool
	terminal      error
	snapshot      transports.EndpointSnapshot
	events        chan transports.EndpointEvent
	done          chan struct{}
	stopOnce      sync.Once
	addressEvents libp2pevent.Subscription
	watchDone     chan struct{}
}

func newEndpointSubscription(
	host libp2phost.Host,
) (transports.EndpointSnapshot, transports.EndpointSubscription, error) {
	addressEvents, err := host.EventBus().Subscribe(
		new(libp2pevent.EvtLocalAddressesUpdated),
		libp2peventbus.BufSize(observationQueueSize),
		libp2peventbus.Name("protos-swarmion-local-endpoints"),
	)
	if err != nil {
		return transports.EndpointSnapshot{}, nil, transports.NewError(
			transports.ErrorCodeTemporarilyUnavailable,
			"observe libp2p local addresses",
			fmt.Errorf("subscribe to local address updates: %w", err),
		)
	}
	sub := &endpointSubscription{
		host:          host,
		initializing:  true,
		events:        make(chan transports.EndpointEvent, observationQueueSize),
		done:          make(chan struct{}),
		addressEvents: addressEvents,
		watchDone:     make(chan struct{}),
	}
	// Subscribe before taking the initial snapshot. Listener callbacks and the
	// event watcher both refresh from Host.Addrs, so a racing transition is
	// either in the snapshot or delivered at a strictly newer revision.
	host.Network().Notify(sub)
	sub.mu.Lock()
	sub.snapshot = transports.EndpointSnapshot{
		Revision:  1,
		Endpoints: localEndpoints(host),
	}
	snapshot := cloneEndpointSnapshot(sub.snapshot)
	sub.initializing = false
	sub.mu.Unlock()
	go sub.watchAddressEvents()
	return snapshot, sub, nil
}

func (s *endpointSubscription) Listen(libp2pnetwork.Network, ma.Multiaddr) {
	s.refresh()
}

func (s *endpointSubscription) ListenClose(libp2pnetwork.Network, ma.Multiaddr) {
	s.refresh()
}

func (s *endpointSubscription) Connected(libp2pnetwork.Network, libp2pnetwork.Conn) {}

func (s *endpointSubscription) Disconnected(libp2pnetwork.Network, libp2pnetwork.Conn) {}

func (s *endpointSubscription) watchAddressEvents() {
	defer close(s.watchDone)
	for range s.addressEvents.Out() {
		s.refresh()
	}
	s.mu.Lock()
	if !s.closed {
		s.failLocked(transports.NewError(
			transports.ErrorCodeClosedLink,
			"observe local endpoints",
			transports.ErrClosedLink,
		))
	}
	s.mu.Unlock()
}

func (s *endpointSubscription) refresh() {
	endpoints := localEndpoints(s.host)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || reflect.DeepEqual(endpoints, s.snapshot.Endpoints) {
		return
	}
	s.snapshot.Revision++
	s.snapshot.Endpoints = endpoints
	if s.initializing {
		return
	}
	event := transports.EndpointEvent{
		Revision: s.snapshot.Revision,
		Snapshot: cloneEndpointSnapshot(s.snapshot),
	}
	select {
	case s.events <- event:
	default:
		s.failLocked(transports.NewError(
			transports.ErrorCodeObservationOverflow,
			"observe local endpoints",
			transports.ErrObservationOverflow,
		))
	}
}

func (s *endpointSubscription) Next(ctx context.Context) (transports.EndpointEvent, error) {
	if err := s.terminalError(); err != nil {
		return transports.EndpointEvent{}, err
	}
	select {
	case event := <-s.events:
		if err := s.terminalError(); err != nil {
			return transports.EndpointEvent{}, err
		}
		return event, nil
	case <-s.done:
		return transports.EndpointEvent{}, s.terminalError()
	case <-ctx.Done():
		return transports.EndpointEvent{}, ctx.Err()
	}
}

func (s *endpointSubscription) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	if !s.closed {
		s.failLocked(transports.NewError(
			transports.ErrorCodeClosedLink,
			"observe local endpoints",
			transports.ErrClosedLink,
		))
	}
	s.mu.Unlock()
	s.stopNotifications()
	select {
	case <-s.watchDone:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *endpointSubscription) terminalError() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.terminal
}

func (s *endpointSubscription) failLocked(err error) {
	if s.closed {
		return
	}
	s.closed = true
	s.terminal = err
	close(s.done)
	go s.stopNotifications()
}

func (s *endpointSubscription) stopNotifications() {
	s.stopOnce.Do(func() {
		s.host.Network().StopNotify(s)
		_ = s.addressEvents.Close()
	})
}

func localEndpoints(host libp2phost.Host) []transports.Endpoint {
	info := peer.AddrInfo{ID: host.ID(), Addrs: host.Addrs()}
	addresses, err := peer.AddrInfoToP2pAddrs(&info)
	if err != nil {
		return nil
	}
	endpoints := make([]transports.Endpoint, 0, len(addresses))
	for _, address := range addresses {
		endpoints = append(endpoints, transports.Endpoint{
			URI:    address.String(),
			Source: transports.EndpointSourceApplication,
		})
	}
	sort.Slice(endpoints, func(i, j int) bool {
		return endpoints[i].URI < endpoints[j].URI
	})
	return endpoints
}

func cloneEndpointSnapshot(snapshot transports.EndpointSnapshot) transports.EndpointSnapshot {
	return transports.EndpointSnapshot{
		Revision:  snapshot.Revision,
		Endpoints: append([]transports.Endpoint(nil), snapshot.Endpoints...),
	}
}

var (
	_ libp2pnetwork.Notifiee          = (*endpointSubscription)(nil)
	_ transports.EndpointSubscription = (*endpointSubscription)(nil)
)
