package swarmionlink

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	libp2phost "github.com/libp2p/go-libp2p/core/host"
	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/nustiueudinastea/swarmion/transports"
)

// hostMuxMutationMu closes the check/install race if more than one Link is
// accidentally created for the same caller-owned host. Protos should still
// create exactly one Link per host and share it across Swarmion scopes.
var hostMuxMutationMu sync.Mutex

// Registry is the single collision and lifetime authority for every protocol
// installed by Protos and Swarmion on one application-owned host. It does not
// expose or own host shutdown.
type Registry struct {
	protocols *protocolRegistry
	link      *Link
	fence     *RouteFence
}

// DrainPeerStreams synchronously resets and waits for every in-flight Protos
// or Swarmion handler belonging to peer. The peer must already be fenced; the
// dispatch barrier closes the final race between a pre-fence inbound stream
// and handler registration without blocking unrelated handlers during the
// bounded drain wait.
func (r *Registry) DrainPeerStreams(ctx context.Context, peerID peer.ID) error {
	if r == nil || r.protocols == nil {
		return fmt.Errorf("protocol registry is unavailable")
	}
	return r.protocols.drainPeer(ctx, peerID)
}

// ApplicationProtocolHandler handles one application protocol synchronously.
// The stream is authenticated by libp2p. The handler must not retain it after
// returning; the registry closes it on return and resets it when the
// registration is withdrawn. Closing the registration cancels ctx and waits
// for every active handler to finish, subject to the Close caller's context.
type ApplicationProtocolHandler struct {
	ID      protocol.ID
	Handler func(context.Context, libp2pnetwork.Stream)
}

// ApplicationProtocolBundle is installed atomically with respect to both
// application registrations and Swarmion's fixed protocol bundle.
type ApplicationProtocolBundle struct {
	Handlers []ApplicationProtocolHandler
}

// NewRegistry creates the one shared protocol registry and borrowed Swarmion
// Link for host. It observes but never closes or reconfigures the host.
func NewRegistry(host libp2phost.Host) (*Registry, error) {
	var fence *RouteFence
	if provider, ok := host.(interface{ SwarmionRouteFence() *RouteFence }); ok {
		fence = provider.SwarmionRouteFence()
	}
	if fence == nil {
		var err error
		fence, err = NewRouteFence()
		if err != nil {
			return nil, err
		}
	}
	return NewRegistryWithRouteFence(host, fence)
}

// NewRegistryWithRouteFence binds the borrowed Link and all application
// protocol registrations to the exact gate installed on the physical host.
func NewRegistryWithRouteFence(host libp2phost.Host, fence *RouteFence) (*Registry, error) {
	if host == nil {
		return nil, transports.NewError(
			transports.ErrorCodeInvalidArgument,
			"borrow libp2p host",
			fmt.Errorf("host is nil"),
		)
	}
	if fence == nil {
		return nil, transports.NewError(
			transports.ErrorCodeInvalidArgument,
			"borrow libp2p host",
			fmt.Errorf("route fence is nil"),
		)
	}
	var nextGen atomic.Uint64
	protocols := newProtocolRegistry(host, &nextGen, fence)
	routes := newRouteTracker(host, fence)
	link := &Link{
		host:     host,
		registry: protocols,
		routes:   routes,
		fence:    fence,
	}
	fence.attachTracker(routes)
	return &Registry{protocols: protocols, link: link, fence: fence}, nil
}

// Link returns the registry's stable borrowed Swarmion Link. Callers must pass
// this exact instance to Swarmion rather than constructing another adapter for
// the same host.
func (r *Registry) Link() *Link {
	if r == nil {
		return nil
	}
	return r.link
}

// RouteFence returns Protos' application-owned peer admission controller. It
// is not a durable membership or deletion record.
func (r *Registry) RouteFence() *RouteFence {
	if r == nil {
		return nil
	}
	return r.fence
}

// RegisterApplicationProtocols installs a complete application protocol bundle
// through the same atomic, collision-aware, generation-safe registry used by
// Link.RegisterProtocols.
func (r *Registry) RegisterApplicationProtocols(
	ctx context.Context,
	bundle ApplicationProtocolBundle,
) (transports.Registration, error) {
	if r == nil || r.protocols == nil {
		return nil, transports.NewError(
			transports.ErrorCodeClosedLink,
			"register application protocols",
			transports.ErrClosedLink,
		)
	}
	handlers := make([]registeredProtocolHandler, 0, len(bundle.Handlers))
	for _, handler := range bundle.Handlers {
		handlers = append(handlers, registeredProtocolHandler{
			id:     transports.ProtocolID(handler.ID),
			handle: handler.Handler,
			valid:  handler.Handler != nil,
		})
	}
	return r.protocols.register(ctx, handlers)
}

type protocolRegistry struct {
	host    libp2phost.Host
	nextGen *atomic.Uint64
	fence   *RouteFence

	dispatchMu sync.RWMutex
	mu         sync.Mutex
	entries    map[transports.ProtocolID]*protocolEntry
}

type protocolEntry struct {
	generation uint64
	active     bool
	handler    registeredProtocolHandler
	scope      *registration
}

type registeredProtocolHandler struct {
	id     transports.ProtocolID
	handle func(context.Context, libp2pnetwork.Stream)
	valid  bool
}

type registration struct {
	registry   *protocolRegistry
	generation uint64
	protocols  []transports.ProtocolID

	mu       sync.Mutex
	stopped  bool
	ctx      context.Context
	cancel   context.CancelFunc
	inFlight sync.WaitGroup
	streams  map[libp2pnetwork.Stream]struct{}
	changed  chan struct{}
	drained  chan struct{}
	stopOnce sync.Once
}

func newProtocolRegistry(host libp2phost.Host, nextGen *atomic.Uint64, fence *RouteFence) *protocolRegistry {
	return &protocolRegistry{
		host:    host,
		nextGen: nextGen,
		fence:   fence,
		entries: make(map[transports.ProtocolID]*protocolEntry),
	}
}

func (r *protocolRegistry) register(
	ctx context.Context,
	handlers []registeredProtocolHandler,
) (result transports.Registration, err error) {
	if ctx == nil {
		return nil, invalidArgumentError("register protocols", "", "", fmt.Errorf("context is nil"))
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(handlers) == 0 {
		return nil, invalidArgumentError("register protocols", "", "", fmt.Errorf("bundle is empty"))
	}

	seen := make(map[transports.ProtocolID]struct{}, len(handlers))
	for _, handler := range handlers {
		if handler.id == "" {
			return nil, invalidArgumentError("register protocols", "", handler.id, fmt.Errorf("protocol is empty"))
		}
		if !handler.valid || handler.handle == nil {
			return nil, invalidArgumentError("register protocols", "", handler.id, fmt.Errorf("handler is nil"))
		}
		if _, duplicate := seen[handler.id]; duplicate {
			return nil, invalidArgumentError("register protocols", "", handler.id, fmt.Errorf("duplicate protocol in bundle"))
		}
		seen[handler.id] = struct{}{}
	}

	hostMuxMutationMu.Lock()
	defer hostMuxMutationMu.Unlock()
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	registeredOnMux := make(map[protocol.ID]struct{})
	for _, registered := range r.host.Mux().Protocols() {
		registeredOnMux[registered] = struct{}{}
	}
	for id := range seen {
		if _, exists := r.entries[id]; exists {
			return nil, protocolInUseError(id, nil)
		}
		if _, exists := registeredOnMux[protocol.ID(id)]; exists {
			return nil, protocolInUseError(id, nil)
		}
	}

	generation := r.nextGen.Add(1)
	handlerCtx, cancel := context.WithCancel(context.Background())
	scope := &registration{
		registry:   r,
		generation: generation,
		protocols:  make([]transports.ProtocolID, 0, len(handlers)),
		ctx:        handlerCtx,
		cancel:     cancel,
		streams:    make(map[libp2pnetwork.Stream]struct{}),
		changed:    make(chan struct{}, 1),
		drained:    make(chan struct{}),
	}
	for _, handler := range handlers {
		scope.protocols = append(scope.protocols, handler.id)
		r.entries[handler.id] = &protocolEntry{
			generation: generation,
			handler:    handler,
			scope:      scope,
		}
	}

	installed := make([]transports.ProtocolID, 0, len(handlers))
	defer func() {
		if recovered := recover(); recovered != nil {
			for _, id := range installed {
				r.host.RemoveStreamHandler(protocol.ID(id))
			}
			for _, id := range scope.protocols {
				entry := r.entries[id]
				if entry != nil && entry.generation == generation {
					delete(r.entries, id)
				}
			}
			scope.cancel()
			result = nil
			err = transports.NewError(
				transports.ErrorCodeTemporarilyUnavailable,
				"register protocol bundle",
				fmt.Errorf("host mux panicked: %v", recovered),
			)
		}
	}()

	for _, handler := range handlers {
		id := handler.id
		r.host.SetStreamHandler(protocol.ID(id), func(raw libp2pnetwork.Stream) {
			r.dispatch(id, generation, raw)
		})
		installed = append(installed, id)
	}
	// Trampolines reject streams until the complete bundle is installed.
	for _, id := range scope.protocols {
		r.entries[id].active = true
	}
	return scope, nil
}

func (r *protocolRegistry) dispatch(
	id transports.ProtocolID,
	generation uint64,
	raw libp2pnetwork.Stream,
) {
	r.dispatchMu.RLock()
	if raw == nil || (r.fence != nil && r.fence.IsPeerFenced(raw.Conn().RemotePeer())) {
		r.dispatchMu.RUnlock()
		if raw != nil {
			_ = raw.Reset()
		}
		return
	}
	r.mu.Lock()
	entry := r.entries[id]
	if entry == nil || entry.generation != generation || !entry.active || !entry.scope.begin(raw) {
		r.mu.Unlock()
		r.dispatchMu.RUnlock()
		_ = raw.Reset()
		return
	}
	handler := entry.handler.handle
	handlerCtx := entry.scope.handlerContext()
	r.mu.Unlock()
	r.dispatchMu.RUnlock()

	defer entry.scope.end(raw)
	defer raw.Close()
	handler(handlerCtx, raw)
}

func (r *protocolRegistry) drainPeer(ctx context.Context, peerID peer.ID) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if peerID == "" {
		return fmt.Errorf("peer id is empty")
	}
	if r.fence == nil || !r.fence.IsPeerFenced(peerID) {
		return fmt.Errorf("peer %s is not fenced", peerID)
	}

	// A writer barrier waits for every dispatch which passed the pre-fence
	// check to publish its stream in a registration. Later dispatches see the
	// fence and reject before entering a handler.
	r.dispatchMu.Lock()
	r.mu.Lock()
	unique := make(map[*registration]struct{}, len(r.entries))
	for _, entry := range r.entries {
		if entry != nil && entry.scope != nil {
			unique[entry.scope] = struct{}{}
		}
	}
	r.mu.Unlock()
	r.dispatchMu.Unlock()

	for scope := range unique {
		if err := scope.drainPeer(ctx, peerID); err != nil {
			return err
		}
	}
	return nil
}

func (r *protocolRegistry) unregister(scope *registration) {
	hostMuxMutationMu.Lock()
	defer hostMuxMutationMu.Unlock()
	r.mu.Lock()
	scope.stop()
	for _, id := range scope.protocols {
		entry := r.entries[id]
		if entry == nil || entry.generation != scope.generation {
			continue
		}
		entry.active = false
		delete(r.entries, id)
		r.host.RemoveStreamHandler(protocol.ID(id))
	}
	r.mu.Unlock()
}

func (r *registration) begin(stream libp2pnetwork.Stream) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stopped {
		return false
	}
	r.inFlight.Add(1)
	r.streams[stream] = struct{}{}
	r.signalChangedLocked()
	return true
}

func (r *registration) end(stream libp2pnetwork.Stream) {
	r.mu.Lock()
	delete(r.streams, stream)
	r.signalChangedLocked()
	r.mu.Unlock()
	r.inFlight.Done()
}

func (r *registration) signalChangedLocked() {
	select {
	case r.changed <- struct{}{}:
	default:
	}
}

func (r *registration) drainPeer(ctx context.Context, peerID peer.ID) error {
	for {
		r.mu.Lock()
		streams := make([]libp2pnetwork.Stream, 0)
		for stream := range r.streams {
			if stream != nil && stream.Conn() != nil && stream.Conn().RemotePeer() == peerID {
				streams = append(streams, stream)
			}
		}
		changed := r.changed
		r.mu.Unlock()
		if len(streams) == 0 {
			return nil
		}
		for _, stream := range streams {
			_ = stream.Reset()
		}
		select {
		case <-changed:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (r *registration) handlerContext() context.Context {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.ctx
}

func (r *registration) stop() {
	r.stopOnce.Do(func() {
		r.mu.Lock()
		r.stopped = true
		r.cancel()
		streams := make([]libp2pnetwork.Stream, 0, len(r.streams))
		for stream := range r.streams {
			streams = append(streams, stream)
		}
		r.mu.Unlock()
		for _, stream := range streams {
			_ = stream.Reset()
		}
		go func() {
			r.inFlight.Wait()
			close(r.drained)
		}()
	})
}

func (r *registration) Close(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	r.registry.unregister(r)
	select {
	case <-r.drained:
		return nil
	case <-ctx.Done():
		return transports.NewError(
			transports.ErrorCodeTemporarilyUnavailable,
			"close protocol registration",
			ctx.Err(),
		)
	}
}

func protocolInUseError(id transports.ProtocolID, cause error) error {
	if cause == nil {
		cause = transports.ErrProtocolInUse
	}
	linkErr := transports.NewError(transports.ErrorCodeProtocolInUse, "register protocols", cause)
	linkErr.Protocol = id
	return linkErr
}

var _ transports.Registration = (*registration)(nil)
