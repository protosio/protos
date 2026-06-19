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

func (p2p *P2P) newConnectionHandler(netw network.Network, conn network.Conn) {
	go func() {
		if conn.Stat().Limited {
			log.Debugf("new limited connection with peer %s", conn.RemotePeer().String())
		}

		if p2p.externalDB.Initialized() {
			if !p2p.peerKnownOrPending(conn.RemotePeer()) {
				log.Warnf("new connection with peer '%s' with unknown machine. Closing connection", conn.RemotePeer().String())
				p2p.removeExternalDBPeer(conn.RemotePeer().String())
				conn.Close()
				return
			}
		} else if !p2p.initPeerAllowed(conn.RemotePeer()) {
			log.Warnf("new init connection with peer '%s' is not allowed. Closing connection", conn.RemotePeer().String())
			conn.Close()
			return
		}

		log.Debugf("new connection with peer %s. Creating client", conn.RemotePeer().String())
		clientCtx, cancel := context.WithTimeout(context.Background(), peerClientReconnectAttemptTimeout)
		client, err := p2p.createClientForPeer(clientCtx, conn.RemotePeer())
		cancel()
		if err != nil {
			p2p.markPeerFailed(conn.RemotePeer().String(), nil, err)
			log.Errorf("failed to create client for new peer %s: %s", conn.RemotePeer().String(), err.Error())
			conn.Close()
			return
		}

		p2p.clients.Set(conn.RemotePeer().String(), client)
		if machine, found := p2p.machines.Get(conn.RemotePeer().String()); found {
			p2p.markPeerConnected(conn.RemotePeer().String(), machine)
		} else {
			p2p.markPeerConnected(conn.RemotePeer().String(), nil)
		}
		if err := p2p.externalDB.AddPeer(conn.RemotePeer().String(), client.grpcConnection); err != nil {
			p2p.markPeerFailed(conn.RemotePeer().String(), nil, err)
			log.Errorf("failed to add peer %s to external DB: %s", conn.RemotePeer().String(), err.Error())
			conn.Close()
			return
		}
	}()
}

func (p2p *P2P) closeConnectionHandler(netw network.Network, conn network.Conn) {
	log.Debug("disconnecting from peer ", conn.RemotePeer().String())
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("error while disconnecting from peer '%s': %v", conn.RemotePeer().String(), err)
		}
	}()

	_, found := p2p.machines.Get(conn.RemotePeer().String())
	if !found {
		log.Debugf("disconnected from unknown peer %s", conn.RemotePeer().String())
		p2p.removeExternalDBPeer(conn.RemotePeer().String())
		return
	}

	log.Debugf("disconnected from peer %s", conn.RemotePeer().String())
	p2p.clients.Delete(conn.RemotePeer().String())
	p2p.markPeerDisconnected(conn.RemotePeer().String())
	p2p.removeExternalDBPeer(conn.RemotePeer().String())
	p2p.requestReconcile()
}
