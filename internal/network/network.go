package network

import (
	"net"

	"github.com/protosio/protos/internal/util"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// var wgPort int = 10999
var log = util.GetLogger("network")

type Manager struct {
	privateKey wgtypes.Key
	network    net.IPNet
	domain     string
}

func NewManager(network net.IPNet, privateKey wgtypes.Key, domain string) (*Manager, error) {
	manager := &Manager{network: network, privateKey: privateKey, domain: domain}
	err := manager.Up()
	if err != nil {
		return nil, err
	}
	return manager, nil
}
