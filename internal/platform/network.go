package platform

import (
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/foxcpp/wirebox"
	"github.com/foxcpp/wirebox/linkmgr"
	"github.com/protosio/protos/internal/auth"
	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const interfacePrefix = "protos"

var wgPort int = 10999
var netBridge *netlink.Bridge

func compareRoutes(a linkmgr.Route, b linkmgr.Route) bool {
	if a.Dest.String() == b.Dest.String() && a.Src.Equal(b.Src) {
		return true
	}
	return false
}

// diffRoutes ignores IPv6 addresses at the moment
func diffRoutes(a []linkmgr.Route, b []linkmgr.Route) ([]linkmgr.Route, []linkmgr.Route) {
	extraA := []linkmgr.Route{}
	for _, ar := range a {
		matched := false
		for _, br := range b {
			if compareRoutes(ar, br) {
				matched = true
			}
		}
		if !matched && !strings.Contains(ar.Dest.IP.String(), ":") {
			extraA = append(extraA, ar)
		}
	}

	extraB := []linkmgr.Route{}
	for _, br := range b {
		matched := false
		for _, ar := range a {
			if compareRoutes(br, ar) {
				matched = true
			}
		}
		if !matched {
			extraB = append(extraB, br)
		}
	}
	return extraA, extraB
}

// initNetwork initializes the local network
func initNetwork(network net.IPNet, devices []auth.UserDevice, key wgtypes.Key) (string, error) {
	manager, err := linkmgr.NewManager()
	if err != nil {
		return "", fmt.Errorf("Failed to initialize network: %w", err)
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
	newRoutes := []linkmgr.Route{{Dest: network, Src: ip}}
	peers := []wgtypes.PeerConfig{}
	if len(devices) == 0 {
		return "", fmt.Errorf("Network initialization failed because 0 user devices were provided")
	}
	for _, userDevice := range devices {
		log.Debugf("Using route '%s' for device '%s(%s)'", userDevice.Network, userDevice.Name, userDevice.MachineID)
		publicKey, err := base64.StdEncoding.DecodeString(userDevice.PublicKey)
		if err != nil {
			return "", fmt.Errorf("Failed to decode base64 encoded key for device '%s': %w", userDevice.Name, err)
		}
		_, deviceNetwork, err := net.ParseCIDR(userDevice.Network)
		if err != nil {
			return "", fmt.Errorf("Failed to parse network for device '%s': %w", userDevice.Name, err)
		}
		newRoutes = append(newRoutes, linkmgr.Route{Dest: *deviceNetwork})
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
		return "", fmt.Errorf("Failed to create interface during network initialization: %w", err)
	}

	existingRoutes, err := link.GetRoutes()
	if err != nil {
		return "", fmt.Errorf("Failed to get routes during network initialization: %w", err)
	}

	delRoutes, addRoutes := diffRoutes(existingRoutes, newRoutes)

	// add the new routes to the wireguard interface
	for _, route := range addRoutes {
		err = link.AddRoute(route)
		if err != nil {
			return "", fmt.Errorf("Failed to add route during network initialization: %w", err)
		}
	}

	// delete old routes from the wireguard interface
	for _, route := range delRoutes {
		err = link.DelRoute(route)
		if err != nil {
			return "", fmt.Errorf("Failed to delete route during network initialization: %w", err)
		}
	}

	// cheating by sleeping 2 seconds
	log.Debugf("Waiting for link '%s' to come up", interfaceName)
	time.Sleep(2 * time.Second)

	brName := interfacePrefix + "1"
	log.Debugf("Setting up bridge interface '%s'", brName)
	netBridge, err = initBridge(brName)
	if err != nil {
		return "", err
	}

	return interfaceName, nil
}
