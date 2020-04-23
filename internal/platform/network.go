package platform

import (
	"encoding/base64"
	"fmt"
	"net"

	"github.com/foxcpp/wirebox"
	"github.com/foxcpp/wirebox/linkmgr"
	"github.com/protosio/protos/pkg/types"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const interfacePrefix = "protos"

var wgPort int = 10999

// initNetwork initializes the local network
func initNetwork(network net.IPNet, devices []types.UserDevice) (string, net.IP, error) {
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

	peers := []wgtypes.PeerConfig{}
	for _, dev := range devices {
		publicKey, err := base64.StdEncoding.DecodeString(dev.PublicKey)
		if err != nil {
			return "", nil, fmt.Errorf("Failed to decode base64 encoded key for device '%s': %w", dev.Name, err)
		}
		_, devNetwork, err := net.ParseCIDR(dev.Network)
		if err != nil {
			return "", nil, fmt.Errorf("Failed to parse network for device '%s': %w", dev.Name, err)
		}
		var pkey wgtypes.Key
		copy(pkey[:], publicKey)

		peers = append(peers, wgtypes.PeerConfig{PublicKey: pkey, ReplaceAllowedIPs: true, AllowedIPs: []net.IPNet{*devNetwork}})
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
