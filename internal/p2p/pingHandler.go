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

// ClientInit is a client to a remote init server
type ClientPing struct {
	p2p    *P2P
	peerID peer.ID
}

// NewRemotePing creates a new remote ping client
func NewRemotePing(p2p *P2P, peerID peer.ID) *ClientPing {
	ip := &ClientPing{
		p2p:    p2p,
		peerID: peerID,
	}
	return ip
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

// PerformInit does the actual initialisation on the remote side
func (hi *HandlersPing) PerformPing(data interface{}) (interface{}, error) {
	return PingResp{}, nil
}
