package main

import (
	"fmt"
	"net"
	"time"

	"github.com/protosio/protos/internal/cloud"
	"github.com/protosio/protos/internal/network"
	"github.com/protosio/protos/internal/user"
	"github.com/urfave/cli/v2"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const protosNetworkInterface = "protos0"

var cmdVPN *cli.Command = &cli.Command{
	Name:  "vpn",
	Usage: "Manage VPN",
	Subcommands: []*cli.Command{
		{
			Name:  "start",
			Usage: "Start the VPN",
			Action: func(c *cli.Context) error {
				return startVPN()
			},
		},
		{
			Name:  "stop",
			Usage: "Stop the VPN",
			Action: func(c *cli.Context) error {
				return stopVPN()
			},
		},
	},
}

func startVPN() error {

	usr, err := user.Get(envi)
	if err != nil {
		return err
	}

	manager, err := network.NewManager()
	if err != nil {
		return err
	}

	// create protos vpn interface and configure the address
	lnk, err := manager.CreateLink(protosNetworkInterface)
	if err != nil {
		return err
	}
	ip, netp, err := net.ParseCIDR(usr.Device.Network)
	if err != nil {
		return err
	}
	netp.IP = ip
	err = lnk.AddAddr(network.Address{IPNet: *netp})
	if err != nil {
		return err
	}

	// create wireguard peer configurations and route list
	var instances []cloud.InstanceInfo
	err = envi.DB.GetSet(instanceDS, &instances)
	if err != nil {
		return err
	}
	var masterInstaceIP net.IP
	keepAliveInterval := 25 * time.Second
	peers := []wgtypes.PeerConfig{}
	routes := []network.Route{}
	for _, instance := range instances {
		var pubkey wgtypes.Key
		copy(pubkey[:], instance.PublicKey)

		_, instanceNetwork, err := net.ParseCIDR(instance.Network)
		if err != nil {
			return fmt.Errorf("Failed to parse network for instance '%s': %w", instance.Name, err)
		}
		instancePublicIP := net.ParseIP(instance.PublicIP)
		if instancePublicIP == nil {
			return fmt.Errorf("Failed to parse public IP for instance '%s'", instance.Name)
		}
		routes = append(routes, network.Route{Dest: *instanceNetwork})

		instanceInternalIP := net.ParseIP(instance.InternalIP)
		if instancePublicIP == nil {
			return fmt.Errorf("Failed to parse internal IP for instance '%s'", instance.Name)
		}
		masterInstaceIP = instanceInternalIP

		peerConf := wgtypes.PeerConfig{
			PublicKey:                   pubkey,
			PersistentKeepaliveInterval: &keepAliveInterval,
			Endpoint:                    &net.UDPAddr{IP: instancePublicIP, Port: 10999},
			AllowedIPs:                  []net.IPNet{*instanceNetwork},
		}
		peers = append(peers, peerConf)
	}

	// configure wireguard
	var pkey wgtypes.Key
	copy(pkey[:], usr.Device.KeySeed)
	wgcfg := wgtypes.Config{
		PrivateKey: &pkey,
		Peers:      peers,
	}
	err = lnk.ConfigureWG(wgcfg)
	if err != nil {
		return err
	}

	// add the routes towards instances
	for _, route := range routes {
		err = lnk.AddRoute(route)
		if err != nil {
			return err
		}
	}

	// add DNS server for domain
	dns, err := network.NewDNS()
	if err != nil {
		return err
	}

	err = dns.AddDomainServer(usr.Domain, masterInstaceIP)
	if err != nil {
		return err
	}

	return nil
}

func stopVPN() error {
	usr, err := user.Get(envi)
	if err != nil {
		return err
	}

	manager, err := network.NewManager()
	if err != nil {
		return err
	}

	// remove VPN link
	_, err = manager.GetLink(protosNetworkInterface)
	if err != nil {
		return err
	}

	err = manager.DelLink(protosNetworkInterface)
	if err != nil {
		return err
	}

	// remove DNS server for domain
	dns, err := network.NewDNS()
	if err != nil {
		return err
	}

	err = dns.DelDomainServer(usr.Domain)
	if err != nil {
		return err
	}

	return nil
}
