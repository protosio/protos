package platform

import (
	"fmt"
	"net"

	"github.com/foxcpp/wirebox"
	"github.com/foxcpp/wirebox/linkmgr"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var wgPort int = 10999

// initNetwork initializes the local network
func initNetwork(interfaceName string) error {
	manager, err := linkmgr.NewManager()
	if err != nil {
		return fmt.Errorf("Failed to initialize network: %w", err)
	}

	linkAddrs := []linkmgr.Address{
		{
			IPNet: net.IPNet{
				IP:   net.ParseIP("10.100.100.1"),
				Mask: net.CIDRMask(24, 32),
			},
			Scope: linkmgr.ScopeLink,
		},
	}

	cfg := wgtypes.Config{
		ReplacePeers: true,
		ListenPort:   &wgPort,
		Peers:        []wgtypes.PeerConfig{},
	}
	_, _, err = wirebox.CreateWG(manager, interfaceName, cfg, linkAddrs)
	if err != nil {
		return fmt.Errorf("Failed to initialize network: %w", err)
	}
	return nil
}
