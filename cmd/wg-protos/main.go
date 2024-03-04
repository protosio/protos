package main

import (
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/nustiueudinastea/wirebox/linkmgr"
	"github.com/protosio/protos/internal/network"
	"github.com/urfave/cli/v2"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var linkManager linkmgr.Manager
var DNSManager *network.DNSManager

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
				Name:  "wg",
				Usage: "Manage WireGuard interface",
				Subcommands: []*cli.Command{
					{
						Name:      "up",
						Usage:     "Creates a new Protos WG interface",
						ArgsUsage: "<interface> <cidr>",
						Action: func(c *cli.Context) error {
							iface := c.Args().First()
							if iface == "" {
								return fmt.Errorf("interface argument cannot be empty")
							}

							cidr := c.Args().Get(1)
							if cidr == "" {
								return fmt.Errorf("CIDR argument cannot be empty")
							}
							return createLink(iface, cidr)
						},
					},
					{
						Name:      "down",
						Usage:     "Deletes the existing Protos WG interface",
						ArgsUsage: "<interface>",
						Action: func(c *cli.Context) error {
							iface := c.Args().First()
							if iface == "" {
								return fmt.Errorf("interface argument cannot be empty")
							}

							return deleteLink(iface)
						},
					},
					{
						Name:      "configure",
						Usage:     "Configures the existing Protos WG interface",
						ArgsUsage: "<interface> <private key> <name:pubkey:publicIP:internalIP:CIDR> [<name:pubkey:publicIP:internalIP:CIDR> ...]",
						Action: func(c *cli.Context) error {
							args := c.Args().Slice()
							if len(args) < 3 {
								return fmt.Errorf("please provide at least these 3 arguments: <interface> <private key> <name:pubkey:publicIP:internalIP:CIDR> ")
							}
							return configureLink(c.Args().First(), c.Args().Get(1), c.Args().Slice()[2:])
						},
					},
				},
			},
			{
				Name:  "domain",
				Usage: "Manage domains",
				Subcommands: []*cli.Command{
					{
						Name:      "add",
						Usage:     "Add DNS server for domain",
						ArgsUsage: "<domain> <DNS server>",
						Action: func(c *cli.Context) error {
							domain := c.Args().First()
							if domain == "" {
								return fmt.Errorf("domain argument cannot be empty")
							}
							dnsServer := c.Args().Get(1)
							if dnsServer == "" {
								return fmt.Errorf("DNS server argument cannot be empty")
							}
							return addDomain(domain, dnsServer)
						},
					},
					{
						Name:      "del",
						Usage:     "Delete DNS server for domain",
						ArgsUsage: "<domain>",
						Action: func(c *cli.Context) error {
							domain := c.Args().First()
							if domain == "" {
								return fmt.Errorf("domain argument cannot be empty")
							}
							return delDomain(domain)
						},
					},
				},
			},
		},
	}

	app.Before = func(c *cli.Context) error {
		linkManager, err = linkmgr.NewManager()
		if err != nil {
			return err
		}
		DNSManager, err = network.NewDNSManager()
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
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to create VPN interface '%s': %w", iface, err)
		}
		lnk, err = linkManager.GetLink(iface)
		if err != nil {
			return fmt.Errorf("failed to create VPN interface '%s': %w", iface, err)
		}
	}

	ip, netp, err := net.ParseCIDR(network)
	if err != nil {
		return fmt.Errorf("failed to parse CIDR while creating VPN interface '%s': %w", iface, err)
	}
	netp.IP = ip
	err = lnk.AddAddr(linkmgr.Address{IPNet: *netp})
	if err != nil {
		return fmt.Errorf("failed to add address while creating VPN interface '%s': %w", iface, err)
	}

	return nil
}

func deleteLink(iface string) error {
	// remove vpn interface
	err := linkManager.DelLink(iface)
	if err != nil {
		if !strings.Contains(err.Error(), "no such network interface") {
			return fmt.Errorf("failed to delete VPN interface '%s': %w", iface, err)
		}
	}

	return nil
}

func configureLink(iface string, privateKey string, peerConfigs []string) error {

	// remove vpn interface
	lnk, err := linkManager.GetLink(iface)
	if err != nil {
		return fmt.Errorf("failed to configure VPN interface '%s': %w", iface, err)
	}

	keepAliveInterval := 25 * time.Second
	peers := []wgtypes.PeerConfig{}
	routes := []linkmgr.Route{}
	for _, peerConfig := range peerConfigs {
		_, pubkey, peerPublicIP, _, peerNet, err := parsePeerConfig(peerConfig)
		if err != nil {
			return fmt.Errorf("failed to configure VPN interface '%s': %w", iface, err)
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
		return fmt.Errorf("failed to configure VPN interface '%s': %w", iface, err)
	}

	var wgPrivateKey wgtypes.Key
	copy(wgPrivateKey[:], decodedPrivateKey)
	wgcfg := wgtypes.Config{
		PrivateKey:   &wgPrivateKey,
		Peers:        peers,
		ReplacePeers: true,
	}
	err = lnk.ConfigureWG(wgcfg)
	if err != nil {
		return fmt.Errorf("failed to configure VPN interface '%s': %w", iface, err)
	}

	// add the routes towards instances
	for _, route := range routes {
		err = lnk.AddRoute(route)
		if err != nil {
			return fmt.Errorf("failed to configure VPN interface '%s': %w", iface, fmt.Errorf("failed to add route: %w", err))
		}
	}

	return nil
}

func addDomain(domain string, dnsServer string) error {
	dnsServerIP := net.ParseIP(dnsServer)
	if dnsServerIP == nil {
		return fmt.Errorf("failed to parse DNS server IP '%s'", dnsServer)
	}

	err := DNSManager.AddDomainServer(domain, dnsServerIP, 10053)
	if err != nil {
		return fmt.Errorf("failed to add domain: %w", err)
	}
	return nil
}

func delDomain(domain string) error {
	err := DNSManager.DelDomainServer(domain)
	if err != nil {
		if !strings.Contains(err.Error(), "no such file or directory") {
			return fmt.Errorf("failed to delete DNS server for domain '%s': %w", domain, err)
		}
	}
	return nil
}

func parsePeerConfig(peerConfig string) (string, wgtypes.Key, net.IP, net.IP, net.IPNet, error) {
	parts := strings.Split(peerConfig, ":")
	if len(parts) != 5 {
		return "", wgtypes.Key{}, net.IP{}, net.IP{}, net.IPNet{}, fmt.Errorf("failed to parse the following peer config: '%s'", peerConfig)
	}

	publicKey, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", wgtypes.Key{}, net.IP{}, net.IP{}, net.IPNet{}, fmt.Errorf("failed to decode public key in peer config '%s': %w", peerConfig, err)
	}

	var wgPublicKey wgtypes.Key
	copy(wgPublicKey[:], publicKey)

	peerPublicIP := net.ParseIP(parts[2])
	if peerPublicIP == nil {
		return "", wgtypes.Key{}, net.IP{}, net.IP{}, net.IPNet{}, fmt.Errorf("failed to parse public IP in peer config '%s': %w", peerConfig, err)
	}
	peerInternalIP := net.ParseIP(parts[3])
	if peerPublicIP == nil {
		return "", wgtypes.Key{}, net.IP{}, net.IP{}, net.IPNet{}, fmt.Errorf("failed to parse internal IP in peer config '%s': %w", peerConfig, err)
	}
	_, peerNet, err := net.ParseCIDR(parts[4])
	if err != nil {
		return "", wgtypes.Key{}, net.IP{}, net.IP{}, net.IPNet{}, fmt.Errorf("failed to parse network in peer config '%s': %w", peerConfig, err)
	}

	return parts[0], wgPublicKey, peerPublicIP, peerInternalIP, *peerNet, nil
}
