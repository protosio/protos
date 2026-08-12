package p2p

import (
	"context"
	"fmt"
	stdnet "net"
	"time"

	p2pgrpc "github.com/birros/go-libp2p-grpc"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	p2pproto "github.com/protosio/protos/internal/p2p/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const peerClientPingTimeout = 3 * time.Second

// createClientForPeer returns the remote client that can reach all remote handlers.
func (p2p *P2P) createClientForPeer(ctx context.Context, peerID peer.ID) (client *Client, err error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// grpc conn
	conn, err := grpc.NewClient(
		"passthrough:///"+peerID.String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		p2p.withP2PDialer(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to grpc dial peer '%s': %w", peerID.String(), err)
	}

	client = &Client{
		PingerClient:   p2pproto.NewPingerClient(conn),
		PeerDBClient:   p2pproto.NewPeerDBClient(conn),
		AppsClient:     p2pproto.NewAppsClient(conn),
		ImagesClient:   p2pproto.NewImagesClient(conn),
		InstanceClient: p2pproto.NewInstanceClient(conn),
	}

	for {
		pingCtx, cancel := context.WithTimeout(ctx, peerClientPingTimeout)
		resp, pingErr := client.Ping(pingCtx, &p2pproto.PingRequest{
			Ping: "pong",
		})
		cancel()
		if pingErr != nil {
			if ctx.Err() != nil {
				_ = conn.Close()
				return nil, fmt.Errorf("failed to ping peer %s: %w", peerID.String(), pingErr)
			}
			select {
			case <-ctx.Done():
				_ = conn.Close()
				return nil, fmt.Errorf("failed to ping peer %s: %w", peerID.String(), pingErr)
			case <-time.After(200 * time.Millisecond):
			}
			continue
		}
		client.setCapabilities(resp.GetCapabilities())
		break
	}
	client.grpcConnection = conn

	return client, nil
}

func (p2p *P2P) withP2PDialer() grpc.DialOption {
	return grpc.WithContextDialer(func(ctx context.Context, peerIDString string) (stdnet.Conn, error) {
		peerID, err := peer.Decode(peerIDString)
		if err != nil {
			return nil, err
		}

		connectedness := p2p.host.Network().Connectedness(peerID)
		if usablePeerConnectedness(connectedness) {
			if connectedness == network.Limited {
				ctx = network.WithAllowLimitedConn(ctx, "protos p2p grpc")
			}
		} else {
			return nil, fmt.Errorf("not connected to peer %s: %s", peerIDString, connectedness)
		}

		stream, err := p2p.host.NewStream(ctx, peerID, protosRPCProtocol)
		if err != nil {
			return nil, err
		}
		return &p2pgrpc.Conn{Stream: stream}, nil
	})
}

// ensureControlClient serializes client creation across TCP, QUIC, direct, and
// relay connection callbacks. A shared host can report several sibling routes
// for one peer; they must all reuse one logical gRPC connection.
func (p2p *P2P) ensureControlClient(ctx context.Context, peerID peer.ID) (*Client, error) {
	p2p.clientMu.Lock()
	defer p2p.clientMu.Unlock()

	if client, found := p2p.clients.Get(peerID.String()); found && client != nil &&
		usablePeerConnectedness(p2p.host.Network().Connectedness(peerID)) {
		return client, nil
	}
	client, err := p2p.createClientForPeer(ctx, peerID)
	if err != nil {
		return nil, err
	}
	p2p.clients.Set(peerID.String(), client)
	return client, nil
}

func (p2p *P2P) newConnectionHandler(netw network.Network, conn network.Conn) {
	if p2p.routeFence != nil && !p2p.routeFence.IsPeerConnectionAllowed(conn.RemotePeer()) {
		log.Debugf("closing connection admitted concurrently with route fence for peer %s", conn.RemotePeer())
		_ = netw.ClosePeer(conn.RemotePeer())
		return
	}
	p2p.addPhysicalRoute(conn.RemotePeer(), conn.ID())
	go func() {
		if conn.Stat().Limited {
			log.Debugf("new limited connection with peer %s", conn.RemotePeer().String())
		}

		if !p2p.controlPeerAllowed(conn.RemotePeer()) {
			// A physical route is shared with Swarmion and may be valid even when
			// this peer is not admitted to Protos control. Admission is enforced
			// again on every application protocol stream.
			log.Debugf("physical connection with peer '%s' is not admitted to Protos control", conn.RemotePeer().String())
			return
		}

		log.Debugf("new connection with peer %s. Creating client", conn.RemotePeer().String())
		clientCtx, cancel := context.WithTimeout(context.Background(), p2p.controlClientReadinessTimeout())
		_, err := p2p.ensureControlClient(clientCtx, conn.RemotePeer())
		cancel()
		if err != nil {
			p2p.markPeerFailed(conn.RemotePeer().String(), nil, err)
			log.Errorf("failed to create client for new peer %s: %s", conn.RemotePeer().String(), err.Error())
			return
		}

		if machine, found := p2p.machines.Get(conn.RemotePeer().String()); found {
			p2p.markPeerConnected(conn.RemotePeer().String(), machine)
		} else {
			p2p.markPeerConnected(conn.RemotePeer().String(), nil)
		}
	}()
}

func (p2p *P2P) closeConnectionHandler(netw network.Network, conn network.Conn) {
	log.Debug("disconnecting from peer ", conn.RemotePeer().String())
	peerID := conn.RemotePeer()
	if remaining := p2p.removePhysicalRoute(peerID, conn.ID()); remaining != 0 {
		return
	}
	p2p.handleLastPhysicalRouteLost(netw, peerID)
}

func (p2p *P2P) initializePhysicalRoutes() {
	if p2p == nil || p2p.host == nil {
		return
	}
	for _, conn := range p2p.host.Network().Conns() {
		p2p.addPhysicalRoute(conn.RemotePeer(), conn.ID())
	}
}

func (p2p *P2P) addPhysicalRoute(peerID peer.ID, connectionID string) {
	if p2p == nil || peerID == "" || connectionID == "" {
		return
	}
	p2p.routeMu.Lock()
	defer p2p.routeMu.Unlock()
	if p2p.peerRoutes == nil {
		p2p.peerRoutes = make(map[string]map[string]struct{})
	}
	routes := p2p.peerRoutes[peerID.String()]
	if routes == nil {
		routes = make(map[string]struct{})
		p2p.peerRoutes[peerID.String()] = routes
	}
	routes[connectionID] = struct{}{}
}

func (p2p *P2P) removePhysicalRoute(peerID peer.ID, connectionID string) int {
	if p2p == nil || peerID == "" || connectionID == "" {
		return 0
	}
	p2p.routeMu.Lock()
	defer p2p.routeMu.Unlock()
	routes := p2p.peerRoutes[peerID.String()]
	delete(routes, connectionID)
	remaining := len(routes)
	if remaining == 0 {
		delete(p2p.peerRoutes, peerID.String())
	}
	return remaining
}

func (p2p *P2P) physicalRouteCount(peerID peer.ID) int {
	p2p.routeMu.Lock()
	defer p2p.routeMu.Unlock()
	return len(p2p.peerRoutes[peerID.String()])
}

func (p2p *P2P) handleLastPhysicalRouteLost(netw network.Network, peerID peer.ID) {
	// A new sibling route may have arrived after the final removal callback.
	// Check the tracked set again under its own generation-free membership lock.
	if p2p.physicalRouteCount(peerID) != 0 {
		return
	}
	// Initialize/recover from a callback missed before this manager registered.
	if usablePeerConnectedness(netw.Connectedness(peerID)) {
		for _, conn := range netw.ConnsToPeer(peerID) {
			p2p.addPhysicalRoute(peerID, conn.ID())
		}
		if p2p.physicalRouteCount(peerID) != 0 {
			return
		}
	}

	_, found := p2p.machines.Get(peerID.String())
	if !found {
		log.Debugf("disconnected from unknown peer %s", peerID.String())
		return
	}

	log.Debugf("disconnected from peer %s", peerID.String())
	p2p.relayReservations.Delete(peerID.String())
	p2p.clientMu.Lock()
	if client, found := p2p.clients.Get(peerID.String()); found && client != nil && client.grpcConnection != nil {
		_ = client.grpcConnection.Close()
	}
	p2p.clients.Delete(peerID.String())
	p2p.clientMu.Unlock()
	p2p.markPeerDisconnected(peerID.String())
	p2p.requestReconcile()
}
