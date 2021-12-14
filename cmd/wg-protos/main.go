package main

import (
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/foxcpp/wirebox/linkmgr"
	"github.com/protosio/protos/internal/vpn"
	"github.com/urfave/cli/v2"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var linkManager linkmgr.Manager
var DNSManager vpn.DNSManager

func main() {

	version, err := semver.NewVersion("0.1.0-dev.4")
	if err != nil {
		panic(err)
	}

	app := &cli.App{
		Name:    "wg-protos",
		Usage:   "Protos WireGuard helper",
		Authors: []*cli.Author{{Name: "Alex Giurgiu", Email: "alex@giurgiu.io"}},
		Version: version.String(),
		Commands: []*cli.Command{
			{
				Name:      "up",
				Usage:     "Creates a new Protos WG interface",
				ArgsUsage: "<cidr>",
				Action: func(c *cli.Context) error {
					cidr := c.Args().First()
					if cidr == "" {
						return fmt.Errorf("CIDR argument cannot be empty")
					}
					return createLink("protos0", cidr)
				},
			},
			{
				Name:      "down",
				Usage:     "Deletes the existing Protos WG interface",
				ArgsUsage: "[domain]",
				Action: func(c *cli.Context) error {
					return deleteLink("protos0", c.Args().First())
				},
			},
			{
				Name:      "configure",
				Usage:     "Configures the existing Protos WG interface",
				ArgsUsage: "<private key> <domain> <name:pubkey:publicIP:internalIP:CIDR> [<name:pubkey:publicIP:internalIP:CIDR> ...]",
				Action: func(c *cli.Context) error {
					args := c.Args().Slice()
					if len(args) < 3 {
						return fmt.Errorf("Please provide at least these 3 arguments argument: <private key> <domain> <name:pubkey:publicIP:internalIP:CIDR> ")
					}
					return configureLink("protos0", c.Args().Get(0), c.Args().Get(1), c.Args().Slice()[2:])
				},
			},
		},
	}

	app.Before = func(c *cli.Context) error {
		linkManager, err = linkmgr.NewManager()
		if err != nil {
			return err
		}
		DNSManager, err = vpn.NewDNS()
		if err != nil {
			return err
		}
		return nil
	}

	err = app.Run(os.Args)
	if err != nil {
		fmt.Printf("Error while running wg-protos: %s\n", err.Error())
		os.Exit(1)
	}
}

func createLink(iface string, network string) error {
	// create protos vpn interface and configure the address
	lnk, err := linkManager.CreateLink(iface)
	if err != nil {
		return fmt.Errorf("Failed to create VPN interface '%s': %w", iface, err)
	}

	ip, netp, err := net.ParseCIDR(network)
	if err != nil {
		return fmt.Errorf("Failed to parse CIDR while creating VPN interface '%s': %w", iface, err)
	}
	netp.IP = ip
	err = lnk.AddAddr(linkmgr.Address{IPNet: *netp})
	if err != nil {
		return fmt.Errorf("Failed to add address while creating VPN interface '%s': %w", iface, err)
	}

	return nil
}

func deleteLink(iface string, domain string) error {
	// remove vpn interface
	_, err := linkManager.GetLink(iface)
	if err != nil {
		return err
	}

	if domain != "" {
		// delete DNS server for domain
		err = DNSManager.DelDomainServer(domain)
		if err != nil {
			return fmt.Errorf("Failed to remove domain '%s': %w", domain, err)
		}
	}

	err = linkManager.DelLink(iface)
	if err != nil {
		return err
	}

	return nil
}

func configureLink(iface string, privateKey string, domain string, peerConfigs []string) error {

	// remove vpn interface
	lnk, err := linkManager.GetLink(iface)
	if err != nil {
		return err
	}

	keepAliveInterval := 25 * time.Second
	peers := []wgtypes.PeerConfig{}
	routes := []linkmgr.Route{}
	var mainInstaceIP net.IP
	for _, peerConfig := range peerConfigs {
		instance, pubkey, peerPublicIP, peerInternalIP, peerNet, err := parsePeerConfig(peerConfig)
		if err != nil {
			return fmt.Errorf("Failed to parse peer config for instance '%s': %w", instance, err)
		}

		if mainInstaceIP == nil {
			mainInstaceIP = peerInternalIP
		}

		routes = append(routes, linkmgr.Route{Dest: peerNet})
		peerConf := wgtypes.PeerConfig{
			PublicKey:                   pubkey,
			PersistentKeepaliveInterval: &keepAliveInterval,
			Endpoint:                    &net.UDPAddr{IP: peerPublicIP, Port: 10999},
			AllowedIPs:                  []net.IPNet{peerNet},
		}
		peers = append(peers, peerConf)
	}

	decodedPrivateKey, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		return fmt.Errorf("Failed to decode private key for interface '%s': %w", iface, err)
	}

	var wgPrivateKey wgtypes.Key
	copy(wgPrivateKey[:], decodedPrivateKey)
	wgcfg := wgtypes.Config{
		PrivateKey: &wgPrivateKey,
		Peers:      peers,
	}
	err = lnk.ConfigureWG(wgcfg)
	if err != nil {
		return fmt.Errorf("Failed to configure WG interface: %w", err)
	}

	// add the routes towards instances
	for _, route := range routes {
		err = lnk.AddRoute(route)
		if err != nil {
			return fmt.Errorf("Failed to add route: %w", err)
		}
	}

	// set DNS server to the IP of the first instance
	if len(peers) > 0 {
		// add DNS server for domain
		err = DNSManager.AddDomainServer(domain, mainInstaceIP)
		if err != nil {
			return fmt.Errorf("Failed to add domain: %w", err)
		}
	}

	return nil
}

func parsePeerConfig(peerConfig string) (string, wgtypes.Key, net.IP, net.IP, net.IPNet, error) {
	parts := strings.Split(peerConfig, ":")
	if len(parts) != 5 {
		return "", wgtypes.Key{}, net.IP{}, net.IP{}, net.IPNet{}, fmt.Errorf("Failed to parse peer config: '%s'", peerConfig)
	}

	publicKey, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", wgtypes.Key{}, net.IP{}, net.IP{}, net.IPNet{}, fmt.Errorf("Failed to decode public key in peer config '%s': %w", peerConfig, err)
	}

	var wgPublicKey wgtypes.Key
	copy(wgPublicKey[:], publicKey)

	peerPublicIP := net.ParseIP(parts[2])
	if peerPublicIP == nil {
		return "", wgtypes.Key{}, net.IP{}, net.IP{}, net.IPNet{}, fmt.Errorf("Failed to parse public IP in peer config '%s': %w", peerConfig, err)
	}
	peerInternalIP := net.ParseIP(parts[3])
	if peerPublicIP == nil {
		return "", wgtypes.Key{}, net.IP{}, net.IP{}, net.IPNet{}, fmt.Errorf("Failed to parse internal IP in peer config '%s': %w", peerConfig, err)
	}
	ip, peerNet, err := net.ParseCIDR(parts[4])
	if err != nil {
		return "", wgtypes.Key{}, net.IP{}, net.IP{}, net.IPNet{}, fmt.Errorf("Failed to parse network in peer config '%s': %w", peerConfig, err)
	}
	fmt.Println(parts[0])
	fmt.Println(parts[1])
	fmt.Println(peerPublicIP)
	fmt.Println(peerInternalIP)
	fmt.Println(peerNet)
	fmt.Println(ip)

	return parts[0], wgPublicKey, peerPublicIP, peerInternalIP, *peerNet, nil
}
