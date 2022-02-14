package p2p

import (
	"encoding/base64"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/libp2p/go-libp2p-core/peer"
)

const (
	getAppLogsHandler = "app"
)

type GetAppLogsReq struct {
	AppName string `json:"app_name" validate:"required"`
}

type GetAppLogsResp struct {
	Logs string `json:"logs" validate:"required"`
}

// ClientInit is a client to a remote init server
type ClientAppManager struct {
	p2p    *P2P
	peerID peer.ID
}

//
// client methods
//

// Init is a remote call to peer, which triggers an init on the remote machine
func (cam *ClientAppManager) GetAppLogs(name string) ([]byte, error) {

	req := GetAppLogsReq{
		AppName: name,
	}

	respData := &GetAppLogsResp{}

	// send the request
	err := cam.p2p.sendRequest(cam.peerID, getAppLogsHandler, req, respData)
	if err != nil {
		return nil, fmt.Errorf("get app logs request to '%s' failed: %w", cam.peerID.String(), err)
	}

	logs, err := base64.StdEncoding.DecodeString(respData.Logs)
	if err != nil {
		return nil, fmt.Errorf("get app logs request to '%s' failed: %w", cam.peerID.String(), err)
	}

	return logs, nil
}

//
// server side handlers
//

type HandlersAppManager struct {
	p2p *P2P
}

// HandlerInit does the initialisation on the server side
func (h *HandlersAppManager) HandlerGetAppLogs(peer peer.ID, data interface{}) (interface{}, error) {

	req, ok := data.(*GetAppLogsReq)
	if !ok {
		return GetAppLogsResp{}, fmt.Errorf("unknown data struct for get app logs request")
	}

	validate := validator.New()
	err := validate.Struct(req)
	if err != nil {
		return nil, fmt.Errorf("failed to validate get app logs request: %w", err)
	}

	logs, err := h.p2p.appManager.GetLogs(req.AppName)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve logs for app '%s': %w", req.AppName, err)
	}

	encodedLogs := base64.StdEncoding.EncodeToString(logs)
	initResp := GetAppLogsResp{
		Logs: encodedLogs,
	}

	return initResp, nil
}
