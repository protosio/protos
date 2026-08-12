package p2p

import (
	"context"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/nustiueudinastea/swarmion/transports"
	"github.com/protosio/protos/internal/swarmionlink"
)

func (p2p *P2P) registerProtocolScope(registry *swarmionlink.Registry) (transports.Registration, error) {
	listener, controlHandler, err := registry.NewApplicationListener(
		protosRPCProtocol,
		func(_ context.Context, stream network.Stream) error {
			if p2p.controlPeerAllowed(stream.Conn().RemotePeer()) {
				return nil
			}
			log.Warnf("rejecting Protos control stream from unadmitted peer %s", stream.Conn().RemotePeer())
			return transports.ErrPolicyDenied
		},
	)
	if err != nil {
		return nil, err
	}
	registration, err := registry.RegisterApplicationProtocols(context.Background(), swarmionlink.ApplicationProtocolBundle{
		Handlers: []swarmionlink.ApplicationProtocolHandler{
			controlHandler,
			{ID: imageBlobStreamProtocol, Handler: p2p.admitProtocolStream(imageBlobStreamProtocol, p2p.handleImageBlobStream)},
			{ID: imageArchiveUploadProtocol, Handler: p2p.admitProtocolStream(imageArchiveUploadProtocol, p2p.handleImageArchiveUploadStream)},
		},
	})
	if err != nil {
		_ = listener.Close()
		return nil, err
	}
	p2p.grpcListener = listener
	return registration, nil
}

func (p2p *P2P) admitProtocolStream(protocolID protocol.ID, handler network.StreamHandler) func(context.Context, network.Stream) {
	return func(_ context.Context, stream network.Stream) {
		if !p2p.controlPeerAllowed(stream.Conn().RemotePeer()) {
			log.Warnf("rejecting Protos protocol %s from unadmitted peer %s", protocolID, stream.Conn().RemotePeer())
			_ = stream.Reset()
			return
		}
		handler(stream)
	}
}

func (p2p *P2P) controlPeerAllowed(peerID peer.ID) bool {
	externalDB := p2p.externalDatabase()
	if peerID == "" || externalDB == nil {
		return false
	}
	readiness := externalDB.RepositoryReadiness()
	if readiness.Initialized {
		return p2p.peerKnownOrPending(peerID)
	}
	if readiness.ExistingRepository {
		return false
	}
	return p2p.initPeerAllowed(peerID)
}

func (p2p *P2P) peerKnownOrPending(peerID peer.ID) bool {
	if p2p == nil || peerID == "" {
		return false
	}
	if _, found := p2p.machines.Get(peerID.String()); found {
		return true
	}
	if _, found := p2p.pendingPeers.Get(peerID.String()); found {
		return true
	}
	return false
}
