package p2p

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
)

const (
	instancePingHandler    = "instanceping"
	getInstanceLogsHandler = "getinstancelogs"
)

type PingReq struct{}
type PingResp struct{}

type GetInstanceLogsReq struct{}
type GetInstanceLogsResp struct {
	Logs string `json:"logs" validate:"required"`
}

// ClientInstance is a client to a remote ping server
type ClientInstance struct {
	p2p    *P2P
	peerID peer.ID
}

//
// client methods
//

// Ping is a remote ping call to peer
func (c *ClientInstance) Ping() (time.Duration, error) {

	req := PingReq{}
	respData := &PingResp{}

	// send the request
	err := c.p2p.sendRequest(c.peerID, instancePingHandler, req, respData)
	if err != nil {
		return 0, fmt.Errorf("ping request to '%s' failed: %s", c.peerID.String(), err.Error())
	}

	return 0, nil
}

// GetAppLogs retrieves logs fir a specific app
func (c *ClientInstance) GetInstanceLogs() ([]byte, error) {

	req := GetInstanceLogsReq{}
	respData := &GetInstanceLogsResp{}

	// send the request
	err := c.p2p.sendRequest(c.peerID, getInstanceLogsHandler, req, respData)
	if err != nil {
		return nil, fmt.Errorf("get instance logs request to '%s' failed: %w", c.peerID.String(), err)
	}

	logs, err := base64.StdEncoding.DecodeString(respData.Logs)
	if err != nil {
		return nil, fmt.Errorf("get instance logs request to '%s' failed: %w", c.peerID.String(), err)
	}

	return logs, nil
}

//
// server side handlers
//

type HandlersInstance struct {
	p2p *P2P
}

// HandlerPing responds to a ping request on the server side
func (hi *HandlersInstance) HandlerPing(peer peer.ID, data interface{}) (interface{}, error) {
	return PingResp{}, nil
}

// HandlerInit reetrieves logs for a specific app
func (h *HandlersInstance) HandlerGetInstanceLogs(peer peer.ID, data interface{}) (interface{}, error) {

	_, ok := data.(*GetInstanceLogsReq)
	if !ok {
		return GetInstanceLogsResp{}, fmt.Errorf("unknown data struct for get instance logs request")
	}

	logs, err := os.ReadFile("/var/log/protos.log")
	if err != nil {
		return nil, fmt.Errorf("failed to read protos logs: %w", err)
	}

	encodedLogs := base64.StdEncoding.EncodeToString(logs)
	initResp := GetAppLogsResp{
		Logs: encodedLogs,
	}

	return initResp, nil
}
