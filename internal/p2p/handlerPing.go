package p2p

import (
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

const pingHandler = "ping"

type PingReq struct {
}

type PingResp struct {
}

// ClientPing is a client to a remote ping server
type ClientPing struct {
	p2p    *P2P
	peerID peer.ID
}

//
// client methods
//

// Ping is a remote ping call to peer
func (cp *ClientPing) Ping() (time.Duration, error) {

	req := PingReq{}
	respData := &PingResp{}

	// send the request
	err := cp.p2p.sendRequest(cp.peerID, pingHandler, req, respData)
	if err != nil {
		return 0, fmt.Errorf("ping request to '%s' failed: %s", cp.peerID.String(), err.Error())
	}

	return 0, nil
}

//
// server side handlers
//

type HandlersPing struct {
}

// HandlerPing responds to a ping request on the server side
func (hi *HandlersPing) HandlerPing(peer peer.ID, data interface{}) (interface{}, error) {
	return PingResp{}, nil
}
