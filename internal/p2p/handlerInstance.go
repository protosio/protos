package p2p

import (
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"runtime"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/protosio/protos/internal/pcrypto"
)

const (
	instanceInitHandler     = "instanceinit"
	instancePingHandler     = "instanceping"
	instanceGetLogsHandler  = "instancegetlogs"
	instanceGetPeersHandler = "instancegetpeers"

	initMachineName = "initMachine"
)

// MetaConfigurator allows for the configuration of the meta package
type MetaConfigurator interface {
	SetNetwork(network net.IPNet) net.IP
	SetInstanceName(name string)
	GetPrivateKey() (*pcrypto.Key, error)
}

type initMachine struct {
	name      string
	publicIP  string
	publicKey []byte
}

func (im *initMachine) GetPublicKey() []byte {
	return im.publicKey
}
func (im *initMachine) GetPublicIP() string {
	return im.publicIP
}
func (im *initMachine) GetName() string {
	return im.name
}

type InitReq struct {
	OriginDevice          string `json:"origin_device" validate:"required"`
	OriginDevicePublicKey string `json:"origin_device_public_key" validate:"required"`
	Network               string `json:"network" validate:"cidrv4"` // CIDR notation
	InstanceName          string `json:"instance_name" validate:"required"`
}
type InitResp struct {
	InstanceIP   string `json:"instanceip" validate:"ipv4"` // internal IP of the instance
	Architecture string `json:"architecture" validate:"required"`
}

type PingReq struct{}
type PingResp struct{}

type GetInstanceLogsReq struct{}
type GetInstanceLogsResp struct {
	Logs string `json:"logs" validate:"required"`
}

type GetInstancePeersReq struct{}
type GetInstancePeersResp struct {
	Peers map[string]string `json:"peer" validate:"required"`
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

// Init is a remote call to peer, which triggers an init on the remote machine
func (c *ClientInstance) Init(instanceName string, network string, deviceName string, devicePublicKey []byte) (net.IP, string, error) {

	encodedPubKey := base64.StdEncoding.EncodeToString(devicePublicKey)
	req := InitReq{
		OriginDevice:          deviceName,
		OriginDevicePublicKey: encodedPubKey,
		Network:               network,
		InstanceName:          instanceName,
	}

	respData := &InitResp{}

	// send the request
	err := c.p2p.sendRequest(c.peerID, instanceInitHandler, req, respData)
	if err != nil {
		return nil, "", fmt.Errorf("init request to '%s' failed: %s", c.peerID.String(), err.Error())
	}

	// prepare IP and public key of instance
	ipAddr := net.ParseIP(respData.InstanceIP)
	if ipAddr == nil {
		return nil, "", fmt.Errorf("failed to parse IP: %w", err)
	}

	return ipAddr, respData.Architecture, nil
}

// GetInstanceLogs retrieves logs from remote instance
func (c *ClientInstance) GetInstanceLogs() ([]byte, error) {

	req := GetInstanceLogsReq{}
	respData := &GetInstanceLogsResp{}

	// send the request
	err := c.p2p.sendRequest(c.peerID, instanceGetLogsHandler, req, respData)
	if err != nil {
		return nil, fmt.Errorf("get instance logs request to '%s' failed: %w", c.peerID.String(), err)
	}

	logs, err := base64.StdEncoding.DecodeString(respData.Logs)
	if err != nil {
		return nil, fmt.Errorf("get instance logs request to '%s' failed: %w", c.peerID.String(), err)
	}

	return logs, nil
}

// GetInstancePeers retrieves peers from remote instance
func (c *ClientInstance) GetInstancePeers() (map[string]string, error) {

	req := GetInstancePeersReq{}
	respData := &GetInstancePeersResp{}

	// send the request
	err := c.p2p.sendRequest(c.peerID, instanceGetPeersHandler, req, respData)
	if err != nil {
		return nil, fmt.Errorf("get instance peers request to '%s' failed: %w", c.peerID.String(), err)
	}

	return respData.Peers, nil
}

//
// server side handlers
//

type HandlersInstance struct {
	p2p              *P2P
	metaConfigurator MetaConfigurator
}

// HandlerPing responds to a ping request on the server side
func (hi *HandlersInstance) HandlerPing(peer peer.ID, data interface{}) (interface{}, error) {
	return PingResp{}, nil
}

// HandlerGetInstanceLogs retrieves logs for the local instance
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

// HandlerGetInstancePeers retrieves the peers for the local instance
func (h *HandlersInstance) HandlerGetInstancePeers(peer peer.ID, data interface{}) (interface{}, error) {

	_, ok := data.(*GetInstancePeersReq)
	if !ok {
		return GetInstancePeersResp{}, fmt.Errorf("unknown data struct for get instance peers request")
	}

	peers := map[string]string{}

	for rpcpeerItem := range h.p2p.peers.IterBuffered() {
		rpcpeer := rpcpeerItem.Val.(*rpcPeer)
		client := rpcpeer.GetClient()
		machine := rpcpeer.GetMachine()
		peerName := fmt.Sprintf("unknown(%s)", rpcpeerItem.Key)
		peerStatus := "disconnected"
		if machine != nil {
			peerName = fmt.Sprintf("%s(%s)", machine.GetName(), rpcpeerItem.Key)
		}
		if client != nil {
			peerStatus = "connected"
		}
		peers[peerName] = peerStatus
	}

	resp := GetInstancePeersResp{
		Peers: peers,
	}

	return resp, nil
}

// HandlerInit does the initialisation on the server side
func (h *HandlersInstance) HandlerInit(peer peer.ID, data interface{}) (interface{}, error) {

	req, ok := data.(*InitReq)
	if !ok {
		return InitResp{}, fmt.Errorf("unknown data struct for init request")
	}

	validate := validator.New()
	err := validate.Struct(req)
	if err != nil {
		return nil, fmt.Errorf("failed to validate init request: %w", err)
	}

	_, network, err := net.ParseCIDR(req.Network)
	if err != nil {
		return nil, fmt.Errorf("cannot perform initialization, network '%s' is invalid: %w", req.Network, err)
	}

	pubKey, err := base64.StdEncoding.DecodeString(req.OriginDevicePublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	im := &initMachine{
		name:      initMachineName,
		publicKey: pubKey,
	}

	h.p2p.initMode = false
	_, err = h.p2p.AddPeer(im)
	if err != nil {
		return nil, fmt.Errorf("failed to add init device as rpc client: %w", err)
	}

	h.metaConfigurator.SetInstanceName(req.InstanceName)
	ipNet := h.metaConfigurator.SetNetwork(*network)

	resp := InitResp{
		InstanceIP:   ipNet.String(),
		Architecture: runtime.GOARCH,
	}

	return resp, nil
}
