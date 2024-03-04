package p2p

import (
	"encoding/base64"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	handlerAppGetLogs   = "getapplogs"
	handlerAppGetStatus = "getappstatus"
)

type AppGetLogsReq struct {
	AppName string `json:"app_name" validate:"required"`
}
type AppGetLogsResp struct {
	Logs string `json:"logs" validate:"required"`
}

type AppGetStatusReq struct {
	AppName string `json:"app_name" validate:"required"`
}
type AppGetStatusResp struct {
	Status string `json:"status" validate:"required"`
}

// ClientInit is a client to a remote init server
type ClientAppManager struct {
	p2p    *P2P
	peerID peer.ID
}

//
// client methods
//

// GetAppLogs retrieves logs fir a specific app
func (c *ClientAppManager) GetAppLogs(name string) ([]byte, error) {

	req := AppGetLogsReq{
		AppName: name,
	}

	respData := &AppGetLogsResp{}

	// send the request
	err := c.p2p.sendRequest(c.peerID, handlerAppGetLogs, req, respData)
	if err != nil {
		return nil, fmt.Errorf("get app logs request to '%s' failed: %w", c.peerID.String(), err)
	}

	logs, err := base64.StdEncoding.DecodeString(respData.Logs)
	if err != nil {
		return nil, fmt.Errorf("get app logs request to '%s' failed: %w", c.peerID.String(), err)
	}

	return logs, nil
}

// GetAppLogs retrieves logs fir a specific app
func (c *ClientAppManager) GetAppStatus(name string) (string, error) {

	req := AppGetStatusReq{
		AppName: name,
	}

	respData := &AppGetStatusResp{}

	// send the request
	err := c.p2p.sendRequest(c.peerID, handlerAppGetStatus, req, respData)
	if err != nil {
		return "", fmt.Errorf("get app status request to '%s' failed: %w", c.peerID.String(), err)
	}

	return respData.Status, nil
}

//
// server side handlers
//

type HandlersAppManager struct {
	p2p *P2P
}

// HandlerInit reetrieves logs for a specific app
func (h *HandlersAppManager) HandlerGetAppLogs(peer peer.ID, data interface{}) (interface{}, error) {

	req, ok := data.(*AppGetLogsReq)
	if !ok {
		return AppGetLogsResp{}, fmt.Errorf("unknown data struct for get app logs request")
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
	initResp := AppGetLogsResp{
		Logs: encodedLogs,
	}

	return initResp, nil
}

// HandlerInit reetrieves logs for a specific app
func (h *HandlersAppManager) HandlerAppGetStatus(peer peer.ID, data interface{}) (interface{}, error) {

	req, ok := data.(*AppGetStatusReq)
	if !ok {
		return AppGetStatusResp{}, fmt.Errorf("unknown data struct for get app status request")
	}

	validate := validator.New()
	err := validate.Struct(req)
	if err != nil {
		return nil, fmt.Errorf("failed to validate get app status request: %w", err)
	}

	status, err := h.p2p.appManager.GetStatus(req.AppName)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve status for app '%s': %w", req.AppName, err)
	}

	resp := AppGetStatusResp{
		Status: status,
	}

	return resp, nil
}
