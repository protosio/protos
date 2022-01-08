package p2p

import (
	"fmt"

	"github.com/libp2p/go-libp2p-core/peer"
)

type pubsubPayloadBroadcastHead struct {
	Dataset string
	Head    string
}

type ClientPubSub struct {
	p2p *P2P
}

func (ps *ClientPubSub) RequestHead() error {
	payload := emptyReq{}
	return ps.p2p.BroadcastMsg(pubsubRequestHead, payload)
}

func (ps *ClientPubSub) BroadcastHead(dataset string, head string) error {
	payload := pubsubPayloadBroadcastHead{
		Dataset: dataset,
		Head:    head,
	}
	return ps.p2p.BroadcastMsg(pubsubBroadcastHead, payload)
}

type pubSub struct {
	dbSyncer DBSyncer
	p2p      *P2P
}

func (ps *pubSub) BroadcastHeadHandler(peerID peer.ID, data interface{}) error {
	payload, ok := data.(*pubsubPayloadBroadcastHead)
	if !ok {
		return fmt.Errorf("unknown data struct for broadcast head message")
	}

	log.Debugf("Peer '%s' advertised dataset '%s' head '%s'", peerID.String(), payload.Dataset, payload.Head)

	if !ps.dbSyncer.HasCS(peerID.String()) {
		p2pClient, err := ps.p2p.getClientForPeer(peerID)
		if err != nil {
			return fmt.Errorf("failed to retrieve p2p client for '%s': %s", peerID, err.Error())
		}
		ps.dbSyncer.AddRemoteCS(peerID.String(), p2pClient)
	}

	ps.dbSyncer.Sync(peerID.String(), payload.Dataset, payload.Head)
	return nil
}

func (ps *pubSub) RequestHeadHandler(peerID peer.ID, data interface{}) error {
	log.Debugf("Peer '%s' requested head advertisement", peerID.String())
	ps.dbSyncer.BroadcastLocalDatasets()
	return nil
}
