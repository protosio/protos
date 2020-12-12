package p2p

import (
	"fmt"
	"net"

	"github.com/go-playground/validator/v10"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/auth"
	"github.com/protosio/protos/internal/resource"
	"github.com/protosio/protos/internal/ssh"
)

// MetaConfigurator allows for the configuration of the meta package
type MetaConfigurator interface {
	SetDomain(domainName string)
	SetNetwork(network net.IPNet) net.IP
	SetAdminUser(username string)
	CreateProtosResources() (map[string]*resource.Resource, error)
	GetKey() (*ssh.Key, error)
}

// UserCreator allows the creation of a new user
type UserCreator interface {
	CreateUser(username string, password string, name string, domain string, isadmin bool, devices []auth.UserDevice) (*auth.User, error)
}

type InitRequest struct {
	Username string            `json:"username" validate:"required"`
	Name     string            `json:"name" validate:"required"`
	Domain   string            `json:"domain" validate:"fqdn"`
	Network  string            `json:"network" validate:"cidrv4"` // CIDR notation
	Password string            `json:"password" validate:"min=10,max=100"`
	Devices  []auth.UserDevice `json:"devices" validate:"gt=0,dive"`
}

type InitResp struct {
	InstancePubKey string `json:"instancepubkey" validate:"base64"` // ed25519 base64 encoded public key
	InstanceIP     string `json:"instanceip" validate:"ipv4"`       // internal IP of the instance
}

type InitProtocol struct {
	metaConfigurator MetaConfigurator
	userCreator      UserCreator
	p2p              *P2P
}

// Init is a remote call to peer, which triggers an init on the remote machine
func (ip *InitProtocol) Init(id string, username string, password string, name string, domain string, network string, devices []auth.UserDevice) (InitResp, error) {
	peerID, err := peer.IDFromString(id)
	if err != nil {
		return InitResp{}, fmt.Errorf("Failed to parse peer ID from string: %w", err)
	}

	req := InitRequest{
		Username: username,
		Password: password,
		Name:     name,
		Domain:   domain,
		Network:  network,
		Devices:  devices,
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

	req, ok := data.(*InitRequest)
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

	ip.metaConfigurator.SetDomain(req.Domain)
	ipNet := ip.metaConfigurator.SetNetwork(*network)

	user, err := ip.userCreator.CreateUser(req.Username, req.Password, req.Name, req.Domain, true, req.Devices)
	if err != nil {
		return nil, fmt.Errorf("Cannot perform initialization, faild to create user: %w", err)
	}
	ip.metaConfigurator.SetAdminUser(user.GetUsername())

	// create resources
	_, err = ip.metaConfigurator.CreateProtosResources()
	if err != nil {
		log.Error(err)
		return nil, fmt.Errorf("Cannot perform initialization, faild to create resources: %w", err)
	}

	key, err := ip.metaConfigurator.GetKey()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to retrieve key")
	}

	initResp := InitResp{
		InstancePubKey: key.PublicWG().String(),
		InstanceIP:     ipNet.String(),
	}

	return initResp, nil
}

// NewInitProtocol creates a new init protocol handler
func NewInitProtocol(p2p *P2P, metaConfigurator MetaConfigurator, userCreator UserCreator) *InitProtocol {
	ip := &InitProtocol{
		p2p:              p2p,
		metaConfigurator: metaConfigurator,
		userCreator:      userCreator,
	}
	return ip
}
