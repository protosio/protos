// Package swarmionlink adapts a caller-owned Protos libp2p host to Swarmion's
// borrowed transport contract.
//
// The adapter never creates or closes the host, closes a peer connection,
// installs connection gating, trims peers, or changes the caller's dial
// backoff. Protos remains responsible for the physical network for the entire
// lifetime of every Swarmion session using the adapter.
package swarmionlink

import (
	"context"
	"errors"
	"fmt"
	"time"

	libp2phost "github.com/libp2p/go-libp2p/core/host"
	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multistream"

	"github.com/nustiueudinastea/swarmion/transports"
)

// Link is a borrowed view of a caller-owned libp2p host. It deliberately has
// no host or peer-connection close method.
type Link struct {
	host     libp2phost.Host
	registry *protocolRegistry
	routes   *routeTracker
	fence    *RouteFence
}

// New wraps host without taking ownership of it. The application must keep the
// host alive until every registration and route subscription has been closed.
// Applications that also register their own protocols should use NewRegistry
// and share Registry.Link with Swarmion instead.
func New(host libp2phost.Host) (*Link, error) {
	registry, err := NewRegistry(host)
	if err != nil {
		return nil, err
	}
	return registry.Link(), nil
}

// LocalPeer returns the identity authenticated by the caller-owned libp2p host.
func (l *Link) LocalPeer() transports.PeerID {
	if l == nil || l.host == nil {
		return ""
	}
	return transports.PeerID(l.host.ID().String())
}

// RegisterProtocols atomically reserves and activates a protocol bundle.
// Exact handlers already installed on the caller's mux are treated as
// collisions and are never replaced.
func (l *Link) RegisterProtocols(
	ctx context.Context,
	bundle transports.ProtocolBundle,
) (transports.Registration, error) {
	if l == nil || l.registry == nil {
		return nil, transports.NewError(
			transports.ErrorCodeClosedLink,
			"register protocols",
			transports.ErrClosedLink,
		)
	}
	handlers := make([]registeredProtocolHandler, 0, len(bundle.Handlers))
	for _, handler := range bundle.Handlers {
		handler := handler
		handlers = append(handlers, registeredProtocolHandler{
			id: handler.ID,
			handle: func(ctx context.Context, raw libp2pnetwork.Stream) {
				handler.Handler(ctx, wrapStream(raw))
			},
			valid: handler.Handler != nil,
		})
	}
	return l.registry.register(ctx, handlers)
}

// OpenStream opens a Swarmion stream over an existing application-owned route.
// WithNoDial is intentional: only EnsureRoute may ask the caller-owned host to
// establish physical reachability.
func (l *Link) OpenStream(
	ctx context.Context,
	remote transports.PeerID,
	protocolID transports.ProtocolID,
) (transports.Stream, error) {
	if l == nil || l.host == nil {
		return nil, transports.NewError(
			transports.ErrorCodeClosedLink,
			"open stream",
			transports.ErrClosedLink,
		)
	}
	if ctx == nil {
		return nil, invalidArgumentError("open stream", remote, protocolID, fmt.Errorf("context is nil"))
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	remoteID, err := peer.Decode(string(remote))
	if err != nil {
		return nil, invalidArgumentError("open stream", remote, protocolID, err)
	}
	if remoteID == l.host.ID() {
		return nil, invalidArgumentError("open stream", remote, protocolID, fmt.Errorf("remote peer is local peer"))
	}
	if protocolID == "" {
		return nil, invalidArgumentError("open stream", remote, protocolID, fmt.Errorf("protocol is empty"))
	}
	if l.fence != nil && l.fence.IsPeerFenced(remoteID) {
		return nil, policyDenied(remote, "open stream")
	}

	streamCtx := libp2pnetwork.WithNoDial(ctx, "physical dialing is application-owned")
	streamCtx = libp2pnetwork.WithAllowLimitedConn(streamCtx, "use an application-approved routed connection")
	raw, err := l.host.Network().NewStream(streamCtx, remoteID)
	if err != nil {
		return nil, classifyOpenError(remote, protocolID, err)
	}
	if l.fence != nil && l.fence.IsPeerFenced(remoteID) {
		_ = raw.Reset()
		return nil, policyDenied(remote, "open stream")
	}

	// Negotiate eagerly instead of using BasicHost.NewStream's cached lazy
	// selection. Eager negotiation makes Registration.Close observable at
	// establishment time even if Identify cached protocol support earlier.
	selected := protocol.ID("")
	negotiated := make(chan error, 1)
	go func() {
		var negotiateErr error
		selected, negotiateErr = multistream.SelectOneOf([]protocol.ID{protocol.ID(protocolID)}, raw)
		negotiated <- negotiateErr
	}()
	select {
	case err = <-negotiated:
		if err != nil {
			_ = raw.ResetWithError(libp2pnetwork.StreamProtocolNegotiationFailed)
			return nil, classifyOpenError(remote, protocolID, err)
		}
	case <-ctx.Done():
		_ = raw.ResetWithError(libp2pnetwork.StreamProtocolNegotiationFailed)
		<-negotiated
		return nil, ctx.Err()
	}
	if err := raw.SetProtocol(selected); err != nil {
		_ = raw.ResetWithError(libp2pnetwork.StreamResourceLimitExceeded)
		return nil, classifyOpenError(remote, protocolID, err)
	}
	wrapper := wrapStream(raw)
	if wrapper.RemotePeer() != remote {
		_ = raw.Reset()
		linkErr := transports.NewError(
			transports.ErrorCodePolicyDenied,
			"authenticate stream peer",
			fmt.Errorf("authenticated peer %s does not match requested peer %s", wrapper.RemotePeer(), remote),
		)
		linkErr.Peer = remote
		linkErr.Protocol = protocolID
		return nil, linkErr
	}
	return wrapper, nil
}

func classifyOpenError(
	remote transports.PeerID,
	protocolID transports.ProtocolID,
	cause error,
) error {
	code := transports.ErrorCodeTemporarilyUnavailable
	if errors.Is(cause, multistream.ErrNotSupported[protocol.ID]{}) {
		code = transports.ErrorCodeUnsupportedProtocol
	} else if errors.Is(cause, libp2pnetwork.ErrResourceLimitExceeded) {
		code = transports.ErrorCodeResourceExhausted
	}
	linkErr := transports.NewError(code, "open stream", cause)
	linkErr.Peer = remote
	linkErr.Protocol = protocolID
	return linkErr
}

// SubscribeRoutes creates an atomic initial view plus an ordered observation
// stream. Physical connections are collapsed per authenticated PeerID.
func (l *Link) SubscribeRoutes(
	ctx context.Context,
) (transports.RouteSnapshot, transports.RouteSubscription, error) {
	if l == nil || l.host == nil {
		return transports.RouteSnapshot{}, nil, transports.NewError(
			transports.ErrorCodeClosedLink,
			"subscribe routes",
			transports.ErrClosedLink,
		)
	}
	if ctx == nil {
		return transports.RouteSnapshot{}, nil, invalidArgumentError("subscribe routes", "", "", fmt.Errorf("context is nil"))
	}
	if err := ctx.Err(); err != nil {
		return transports.RouteSnapshot{}, nil, err
	}
	if l.routes == nil {
		return transports.RouteSnapshot{}, nil, transports.NewError(
			transports.ErrorCodeClosedLink,
			"subscribe routes",
			transports.ErrClosedLink,
		)
	}
	return l.routes.subscribe()
}

// SubscribeLocalEndpoints reports the application host's currently advertised
// libp2p addresses. The adapter observes subsequent listener, NAT, observed,
// and relay address changes without changing the listener or address policy.
func (l *Link) SubscribeLocalEndpoints(
	ctx context.Context,
) (transports.EndpointSnapshot, transports.EndpointSubscription, error) {
	if l == nil || l.host == nil {
		return transports.EndpointSnapshot{}, nil, transports.NewError(
			transports.ErrorCodeClosedLink,
			"subscribe local endpoints",
			transports.ErrClosedLink,
		)
	}
	if ctx == nil {
		return transports.EndpointSnapshot{}, nil, invalidArgumentError("subscribe local endpoints", "", "", fmt.Errorf("context is nil"))
	}
	if err := ctx.Err(); err != nil {
		return transports.EndpointSnapshot{}, nil, err
	}
	return newEndpointSubscription(l.host)
}

// EnsureRoute submits a positive connection request to the caller-owned host.
// libp2p's Host.Connect arbitrates policy and coalesces concurrent dials. The
// adapter does not clear the peerstore, remove addresses, or reset dial
// backoff; endpoint hints are used only for this explicit request.
func (l *Link) EnsureRoute(ctx context.Context, request transports.RouteRequest) error {
	if l == nil || l.host == nil {
		return transports.NewError(
			transports.ErrorCodeClosedLink,
			"ensure route",
			transports.ErrClosedLink,
		)
	}
	if ctx == nil {
		return invalidArgumentError("ensure route", request.Peer, "", fmt.Errorf("context is nil"))
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	remote, err := peer.Decode(string(request.Peer))
	if err != nil || remote == l.host.ID() {
		if err == nil {
			err = fmt.Errorf("remote peer is local peer")
		}
		return invalidArgumentError("ensure route", request.Peer, "", err)
	}
	if l.fence != nil && l.fence.IsPeerFenced(remote) {
		return policyDenied(request.Peer, "ensure route")
	}
	connectedness := l.host.Network().Connectedness(remote)
	if connectedness == libp2pnetwork.Connected || connectedness == libp2pnetwork.Limited {
		return nil
	}

	now := time.Now()
	addresses := make([]ma.Multiaddr, 0, len(request.Endpoints))
	seen := make(map[string]struct{}, len(request.Endpoints))
	validHintCount := 0
	for _, endpoint := range request.Endpoints {
		if endpoint.URI == "" || (!endpoint.ExpiresAt.IsZero() && !endpoint.ExpiresAt.After(now)) {
			continue
		}
		validHintCount++
		address, parseErr := ma.NewMultiaddr(endpoint.URI)
		if parseErr != nil {
			return invalidArgumentError("ensure route", request.Peer, "", parseErr)
		}
		if info, infoErr := peer.AddrInfoFromP2pAddr(address); infoErr == nil {
			if info.ID != remote {
				return invalidArgumentError(
					"ensure route", request.Peer, "",
					fmt.Errorf("endpoint authenticates peer %s, not %s", info.ID, remote),
				)
			}
			if len(info.Addrs) != 1 {
				return endpointError(request.Peer, fmt.Errorf("endpoint has no dial address"))
			}
			address = info.Addrs[0]
		}
		key := string(address.Bytes())
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		addresses = append(addresses, address)
	}
	if len(request.Endpoints) > 0 && validHintCount == 0 {
		return endpointError(request.Peer, fmt.Errorf("all endpoint hints are empty or expired"))
	}

	err = l.host.Connect(ctx, peer.AddrInfo{ID: remote, Addrs: addresses})
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	code := transports.ErrorCodeTemporarilyUnavailable
	if errors.Is(err, libp2pnetwork.ErrNoRemoteAddrs) {
		code = transports.ErrorCodeUnknownEndpoint
	} else if errors.Is(err, libp2pnetwork.ErrResourceLimitExceeded) {
		code = transports.ErrorCodeResourceExhausted
	}
	linkErr := transports.NewError(code, "ensure route", err)
	linkErr.Peer = request.Peer
	return linkErr
}

func endpointError(peerID transports.PeerID, cause error) error {
	linkErr := transports.NewError(transports.ErrorCodeUnknownEndpoint, "ensure route", cause)
	linkErr.Peer = peerID
	return linkErr
}

func invalidArgumentError(
	operation string,
	peerID transports.PeerID,
	protocolID transports.ProtocolID,
	cause error,
) error {
	linkErr := transports.NewError(transports.ErrorCodeInvalidArgument, operation, cause)
	linkErr.Peer = peerID
	linkErr.Protocol = protocolID
	return linkErr
}

var _ transports.Link = (*Link)(nil)
