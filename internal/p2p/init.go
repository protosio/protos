package p2p

import (
	"fmt"

	"github.com/libp2p/go-libp2p-core/peer"
)

type InitRequest struct {
	Username string `json:"username" validate:"required"`
	Name     string `json:"name" validate:"required"`
	Domain   string `json:"domain" validate:"fqdn"`
	Network  string `json:"network" validate:"cidrv4"` // CIDR notation
}

type InitResp struct {
	InstancePubKey string `json:"instancepubkey" validate:"base64"` // ed25519 base64 encoded public key
	InstanceIP     string `json:"instanceip" validate:"ipv4"`       // internal IP of the instance
}

type InitProtocol struct {
	p2p *P2P
}

// Init is a remote call to peer, which triggers an init on the remote machine
func (ip *InitProtocol) Init(id string, username string, name string, domain string, network string) (InitResp, error) {
	peerID, err := peer.IDFromString(id)
	if err != nil {
		return InitResp{}, fmt.Errorf("Failed to parse peer ID from string: %w", err)
	}

	req := InitRequest{
		Username: username,
		Name:     name,
		Domain:   domain,
		Network:  network,
	}

	respData := &InitResp{}

	// send the request
	log.Infof("Sending init request '%s'", peerID.String())
	err = ip.p2p.sendRequest(peerID, "init", req, respData)
	if err != nil {
		return InitResp{}, fmt.Errorf("Init request to '%s' failed: %s", peerID.String(), err.Error())
	}

	return *respData, nil
}

// Do satisfies the Handler interface
func (ip *InitProtocol) Do(data interface{}) (interface{}, error) {
	initResp := InitResp{InstancePubKey: "pub key ssdasdas", InstanceIP: "1.1.1.1"}
	return initResp, nil
}

// NewInitProtocol creates a new init protocol handler
func NewInitProtocol(p2p *P2P) *InitProtocol {
	ip := &InitProtocol{p2p: p2p}
	return ip
}
