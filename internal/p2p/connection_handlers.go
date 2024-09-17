package p2p

import (
	"context"
	"fmt"
	"time"

	p2pgrpc "github.com/birros/go-libp2p-grpc"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	p2pproto "github.com/protosio/protos/internal/p2p/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// createClientForPeer returns the remote client that can reach all remote handlers
func (p2p *P2P) createClientForPeer(peerID peer.ID) (client *Client, err error) {

	// grpc conn
	conn, err := grpc.Dial(
		peerID.String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		p2pgrpc.WithP2PDialer(p2p.host, protosRPCProtocol),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to grpc dial peer '%s': %w", peerID.String(), err)
	}

	client = &Client{
		PingerClient:   p2pproto.NewPingerClient(conn),
		PeerDBClient:   p2pproto.NewPeerDBClient(conn),
		AppsClient:     p2pproto.NewAppsClient(conn),
		InstanceClient: p2pproto.NewInstanceClient(conn),
	}

	tries := 0
	for {
		_, err = client.Ping(context.TODO(), &p2pproto.PingRequest{
			Ping: "pong",
		})
		if err != nil {
			if tries < 60 {
				time.Sleep(200 * time.Millisecond)
				tries++
				continue
			} else {
				return nil, fmt.Errorf("failed to ping peer %s: %w", peerID.String(), err)
			}
		} else {
			break
		}
	}
	client.grpcConnection = conn

	return client, nil
}

func (p2p *P2P) newConnectionHandler(netw network.Network, conn network.Conn) {
	go func() {
		if conn.Stat().Transient {
			return
		}

		if p2p.externalDB.Initialized() {
			_, found := p2p.machines.Get(conn.RemotePeer().String())
			if !found {
				log.Warnf("new connection with peer '%s' with unknown machine. Closing connection", conn.RemotePeer().String())
				conn.Close()
				return
			}
		}

		log.Debugf("new connection with peer %s. Creating client", conn.RemotePeer().String())
		client, err := p2p.createClientForPeer(conn.RemotePeer())
		if err != nil {
			log.Errorf("failed to create client for new peer %s: %s", conn.RemotePeer().String(), err.Error())
			conn.Close()
			return
		}

		p2p.clients.Set(conn.RemotePeer().String(), client)
		if err := p2p.externalDB.AddPeer(conn.RemotePeer().String(), client.grpcConnection); err != nil {
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
		return
	}

	log.Debugf("disconnected from peer %s", conn.RemotePeer().String())
	p2p.clients.Delete(conn.RemotePeer().String())
	if p2p.externalDB != nil {
		if err := p2p.externalDB.RemovePeer(conn.RemotePeer().String()); err != nil {
			log.Errorf("failed to remove DB peer for %s: %v", conn.RemotePeer().String(), err)
		}
	}
}
