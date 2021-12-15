package p2p

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"net"

	"github.com/go-playground/validator/v10"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/auth"
	"github.com/protosio/protos/internal/resource"
	"github.com/protosio/protos/internal/ssh"
)

const initHandler = "init"

// MetaConfigurator allows for the configuration of the meta package
type MetaConfigurator interface {
	SetDomain(domainName string)
	SetNetwork(network net.IPNet) net.IP
	SetAdminUser(username string)
	SetInstanceName(name string)
	CreateProtosResources() (map[string]*resource.Resource, error)
	GetKey() (*ssh.Key, error)
}

// UserCreator allows the creation of a new user
type UserCreator interface {
	CreateUser(username string, password string, name string, domain string, isadmin bool, devices []auth.UserDevice) (*auth.User, error)
}

// PlatformConfigurator allows the configuration of the container runtime and network
type PlatformConfigurator interface {
	Init(network net.IPNet, devices []auth.UserDevice) error
}

type InitReq struct {
	Username     string            `json:"username" validate:"required"`
	Name         string            `json:"name" validate:"required"`
	Domain       string            `json:"domain" validate:"fqdn"`
	Network      string            `json:"network" validate:"cidrv4"` // CIDR notation
	Password     string            `json:"password" validate:"min=10,max=100"`
	InstanceName string            `json:"instance_name" validate:"required"`
	Devices      []auth.UserDevice `json:"devices" validate:"gt=0,dive"`
}

type InitResp struct {
	InstancePubKey string `json:"instancepubkey" validate:"base64"` // ed25519 base64 encoded public key
	InstanceIP     string `json:"instanceip" validate:"ipv4"`       // internal IP of the instance
}

// ClientInit is a client to a remote init server
type ClientInit struct {
	p2p    *P2P
	peerID peer.ID
}

// NewRemoteInit creates a new remote init client
func NewRemoteInit(p2p *P2P, peerID peer.ID) *ClientInit {
	ip := &ClientInit{
		p2p:    p2p,
		peerID: peerID,
	}
	return ip
}

//
// client methods
//

// Init is a remote call to peer, which triggers an init on the remote machine
func (ip *ClientInit) Init(username string, password string, name string, domain string, instanceName string, network string, devices []auth.UserDevice) (net.IP, ed25519.PublicKey, error) {

	req := InitReq{
		Username:     username,
		Password:     password,
		Name:         name,
		Domain:       domain,
		Network:      network,
		InstanceName: instanceName,
		Devices:      devices,
	}

	respData := &InitResp{}

	// send the request
	err := ip.p2p.sendRequest(ip.peerID, initHandler, req, respData)
	if err != nil {
		return nil, nil, fmt.Errorf("Init request to '%s' failed: %s", ip.peerID.String(), err.Error())
	}

	// prepare IP and public key of instance
	ipAddr := net.ParseIP(respData.InstanceIP)
	if ipAddr == nil {
		return nil, nil, fmt.Errorf("Failed to parse IP: %w", err)
	}
	var pubKey ed25519.PublicKey
	pubKey, err = base64.StdEncoding.DecodeString(respData.InstancePubKey)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to decode public key: %w", err)
	}

	return ipAddr, pubKey, nil
}

//
// server side handlers
//

type HandlersInit struct {
	metaConfigurator     MetaConfigurator
	userCreator          UserCreator
	platformConfigurator PlatformConfigurator
	p2p                  *P2P
}

// PerformInit does the actual initialisation on the remote side
func (hi *HandlersInit) PerformInit(data interface{}) (interface{}, error) {

	req, ok := data.(*InitReq)
	if !ok {
		return InitResp{}, fmt.Errorf("Unknown data struct for init request")
	}

	validate := validator.New()
	err := validate.Struct(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to validate init request: %w", err)
	}

	_, network, err := net.ParseCIDR(req.Network)
	if err != nil {
		return nil, fmt.Errorf("Cannot perform initialization, network '%s' is invalid: %w", req.Network, err)
	}

	hi.metaConfigurator.SetDomain(req.Domain)
	hi.metaConfigurator.SetInstanceName(req.InstanceName)
	ipNet := hi.metaConfigurator.SetNetwork(*network)

	user, err := hi.userCreator.CreateUser(req.Username, req.Password, req.Name, req.Domain, true, req.Devices)
	if err != nil {
		return nil, fmt.Errorf("Cannot perform initialization, faild to create user: %w", err)
	}
	hi.metaConfigurator.SetAdminUser(user.GetUsername())

	// create resources
	_, err = hi.metaConfigurator.CreateProtosResources()
	if err != nil {
		log.Error(err)
		return nil, fmt.Errorf("Cannot perform initialization, faild to create resources: %w", err)
	}

	key, err := hi.metaConfigurator.GetKey()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to retrieve key")
	}

	err = hi.platformConfigurator.Init(*network, req.Devices)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize platform")
	}

	initResp := InitResp{
		InstancePubKey: key.PublicString(),
		InstanceIP:     ipNet.String(),
	}

	return initResp, nil
}
