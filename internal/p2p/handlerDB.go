package p2p

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/libp2p/go-libp2p-core/peer"
)

const sendDatasetsHeadsHandler = "senddatasetsheads"

//
// client for db pubsub methods
//

type pubsubPayloadBroadcastHead struct {
	Dataset string
	Head    string
}

type ClientPubSub struct {
	p2p *P2P
}

func (ps *ClientPubSub) BroadcastRequestHead() error {
	log.Debug("Requesting dataset heads from all peers")
	return ps.p2p.BroadcastMsg(pubsubRequestHead, emptyReq{})
}

func (ps *ClientPubSub) BroadcastHead(dataset string, head string) error {
	payload := pubsubPayloadBroadcastHead{
		Dataset: dataset,
		Head:    head,
	}
	return ps.p2p.BroadcastMsg(pubsubBroadcastHead, payload)
}

//
// handlers for db pubsub methods
//

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

func (ps *pubSub) BroadcastRequestHeadHandler(peerID peer.ID, data interface{}) error {
	log.Debugf("Peer '%s' requested head advertisement", peerID.String())
	rpcPeer, err := ps.p2p.getRPCPeer(peerID)
	if err != nil {
		return fmt.Errorf("discarding message from '%s': %v", peerID.String(), err)
	}

	heads := ps.dbSyncer.GetAllDatasetsHeads()
	err = rpcPeer.client.SendDatasetsHeads(heads)
	if err != nil {
		return fmt.Errorf("error sending datasets heads to '%s': %v", peerID.String(), err)
	}

	return nil
}

//
// send head client
//

type SendDatasetsHeadsReq struct {
	Heads map[string]string `json:"heads" validate:"required"`
}

// ClientDB is a client to a remote DB server
type ClientDB struct {
	p2p    *P2P
	peerID peer.ID
}

func (cdb *ClientDB) SendDatasetsHeads(heads map[string]string) error {
	req := SendDatasetsHeadsReq{
		Heads: heads,
	}

	respData := &emptyResp{}

	// send the request
	err := cdb.p2p.sendRequest(cdb.peerID, sendDatasetsHeadsHandler, req, respData)
	if err != nil {
		return fmt.Errorf("send datasets heads request to '%s' failed: %s", cdb.peerID.String(), err.Error())
	}

	return nil
}

//
// send head handlers
//

type HandlersDB struct {
	dbSyncer DBSyncer
}

// SendDatasetsHeads handles SendDatasetsHeads requests on the server side
func (hdb *HandlersDB) SendDatasetsHeadsHandler(peerID peer.ID, data interface{}) (interface{}, error) {

	req, ok := data.(*SendDatasetsHeadsReq)
	if !ok {
		return emptyResp{}, fmt.Errorf("unknown data struct for SendHead request")
	}

	validate := validator.New()
	err := validate.Struct(req)
	if err != nil {
		return nil, fmt.Errorf("failed to validate SendHead request from '%s': %w", peerID.String(), err)
	}

	for dataset, head := range req.Heads {
		hdb.dbSyncer.Sync(peerID.String(), dataset, head)
	}

	return emptyResp{}, nil
}
