package p2p

import (
	"fmt"
	"net"
	"runtime"

	"github.com/go-playground/validator/v10"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/protosio/protos/internal/pcrypto"
)

const initHandler = "init"

// MetaConfigurator allows for the configuration of the meta package
type MetaConfigurator interface {
	SetNetwork(network net.IPNet) net.IP
	SetInstanceName(name string)
	GetPrivateKey() (*pcrypto.Key, error)
}

type initMachine struct{}

func (im *initMachine) GetPublicKey() []byte {
	return []byte{}
}
func (im *initMachine) GetPublicIP() string {
	return ""
}
func (im *initMachine) GetName() string {
	return "initMachine"
}

type InitReq struct {
	Network      string `json:"network" validate:"cidrv4"` // CIDR notation
	InstanceName string `json:"instance_name" validate:"required"`
}

type InitResp struct {
	InstanceIP   string `json:"instanceip" validate:"ipv4"` // internal IP of the instance
	Architecture string `json:"architecture" validate:"required"`
}

// ClientInit is a client to a remote init server
type ClientInit struct {
	p2p    *P2P
	peerID peer.ID
}

//
// client methods
//

// Init is a remote call to peer, which triggers an init on the remote machine
func (ip *ClientInit) Init(instanceName string, network string) (net.IP, string, error) {

	req := InitReq{
		Network:      network,
		InstanceName: instanceName,
	}

	respData := &InitResp{}

	// send the request
	err := ip.p2p.sendRequest(ip.peerID, initHandler, req, respData)
	if err != nil {
		return nil, "", fmt.Errorf("init request to '%s' failed: %s", ip.peerID.String(), err.Error())
	}

	// prepare IP and public key of instance
	ipAddr := net.ParseIP(respData.InstanceIP)
	if ipAddr == nil {
		return nil, "", fmt.Errorf("failed to parse IP: %w", err)
	}

	return ipAddr, respData.Architecture, nil
}

//
// server side handlers
//

type HandlersInit struct {
	metaConfigurator MetaConfigurator
	p2p              *P2P
}

// PerformInit does the actual initialisation on the remote side
func (hi *HandlersInit) PerformInit(data interface{}) (interface{}, error) {

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

	hi.metaConfigurator.SetInstanceName(req.InstanceName)
	ipNet := hi.metaConfigurator.SetNetwork(*network)
	hi.p2p.initMode = false

	initResp := InitResp{
		InstanceIP:   ipNet.String(),
		Architecture: runtime.GOARCH,
	}

	return initResp, nil
}
