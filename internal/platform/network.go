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
func initNetwork(network net.IPNet, devices []types.UserDevice, key wgtypes.Key) (string, net.IP, error) {
	manager, err := linkmgr.NewManager()
	if err != nil {
		return "", net.IP{}, fmt.Errorf("Failed to initialize network: %w", err)
	}

	// allocate the first IP in the network for Wireguard
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

	// create the peer and routes lists. At the moment these are all the devices that a user has
	routes := []linkmgr.Route{}
	peers := []wgtypes.PeerConfig{}
	for _, userDevice := range devices {
		publicKey, err := base64.StdEncoding.DecodeString(userDevice.PublicKey)
		if err != nil {
			return "", nil, fmt.Errorf("Failed to decode base64 encoded key for device '%s': %w", userDevice.Name, err)
		}
		_, deviceNetwork, err := net.ParseCIDR(userDevice.Network)
		if err != nil {
			return "", nil, fmt.Errorf("Failed to parse network for device '%s': %w", userDevice.Name, err)
		}
		routes = append(routes, linkmgr.Route{Dest: *deviceNetwork, Src: ip})
		var pkey wgtypes.Key
		copy(pkey[:], publicKey)

		peers = append(peers, wgtypes.PeerConfig{PublicKey: pkey, ReplaceAllowedIPs: true, AllowedIPs: []net.IPNet{*deviceNetwork}})
	}

	// create the wireguard interface
	cfg := wgtypes.Config{
		ReplacePeers: true,
		ListenPort:   &wgPort,
		Peers:        peers,
		PrivateKey:   &key,
	}
	interfaceName := interfacePrefix + "0"
	link, _, err := wirebox.CreateWG(manager, interfaceName, cfg, linkAddrs)
	if err != nil {
		return "", net.IP{}, fmt.Errorf("Failed to initialize network: %w", err)
	}

	// add the routes to the wireguard interface
	for _, route := range routes {
		err = link.AddRoute(route)
		if err != nil {
			return "", net.IP{}, fmt.Errorf("Failed to initialize network: %w", err)
		}
	}
	return interfaceName, ip, nil
}
