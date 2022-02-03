package p2p

import (
	"fmt"
	"time"

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

	time.Sleep(500 * time.Millisecond)

	rpcpeerI, found := ps.p2p.peers.Get(peerID.String())
	if !found {
		return fmt.Errorf("failed to find local peer '%s'. Discarding message", peerID.String())
	}
	rpcpeer := rpcpeerI.(*rpcPeer)

	log.Debugf("Peer '%s'(%s) advertised dataset '%s' head '%s'", rpcpeer.machine.GetName(), peerID.String(), payload.Dataset, payload.Head)
	ps.dbSyncer.Sync(peerID.String(), payload.Dataset, payload.Head)

	return nil
}

func (ps *pubSub) RequestHeadHandler(peerID peer.ID, data interface{}) error {
	log.Debugf("Peer '%s' requested head advertisement", peerID.String())
	ps.dbSyncer.BroadcastLocalDatasets()
	return nil
}
