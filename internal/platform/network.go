package platform

import (
	"fmt"
	"net"

	"github.com/foxcpp/wirebox"
	"github.com/foxcpp/wirebox/linkmgr"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const interfacePrefix = "protos"

var wgPort int = 10999

// initNetwork initializes the local network
func initNetwork(network net.IPNet) (string, net.IP, error) {
	manager, err := linkmgr.NewManager()
	if err != nil {
		return "", net.IP{}, fmt.Errorf("Failed to initialize network: %w", err)
	}

	ip := network.IP.Mask(network.Mask)
	ip[3]++
	linkAddrs := []linkmgr.Address{
		{
			IPNet: net.IPNet{
				IP:   ip,
				Mask: network.Mask,
			},
			Scope: linkmgr.ScopeLink,
		},
	}

	cfg := wgtypes.Config{
		ReplacePeers: true,
		ListenPort:   &wgPort,
		Peers:        []wgtypes.PeerConfig{},
	}
	interfaceName := interfacePrefix + "0"
	_, _, err = wirebox.CreateWG(manager, interfaceName, cfg, linkAddrs)
	if err != nil {
		return "", net.IP{}, fmt.Errorf("Failed to initialize network: %w", err)
	}
	return interfaceName, ip, nil
}
