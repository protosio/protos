package network

import (
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/protosio/protos/internal/cloud"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/user"
	"github.com/protosio/protos/internal/wireguard"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const (
	protosNetworkInterface = "protos0"
	wgProtosBinary         = "wg-protos"
)

func (m *Manager) Up() error {

	err := m.createLink(protosNetworkInterface, m.key.IPv6Address().String())
	if err != nil {
		return fmt.Errorf("failed to create wg interface: %w", err)
	}

	err = m.addDomain(m.domain, "127.0.0.1")
	if err != nil {
		return fmt.Errorf("failed to add domain: %w", err)
	}

	return nil
}

func (m *Manager) Down() error {
	err := m.deleteLink(protosNetworkInterface)
	if err != nil {
		return fmt.Errorf("failed to delete link: %w", err)
	}

	err = m.delDomain(m.domain)
	if err != nil {
		return fmt.Errorf("failed to delete domain: %w", err)
	}

	return nil
}

func (m *Manager) ConfigurePeers(instances []cloud.InstanceInfo, devices []user.UserDevice) error {

	log.Debug("Configuring network peers")
	peers := []wgtypes.PeerConfig{}
	routes := []wireguard.Route{}
	keepAliveInterval := 25 * time.Second

	for _, instance := range instances {
		fmt.Println("Instance: %v", instance)
		if len(instance.PublicKey) == 0 || instance.PublicIP == "" || instance.Name == "" {
			continue
		}

		pubkey, err := pcrypto.ConvertPublicEd25519ToCurve25519(instance.PublicKey)
		if err != nil {
			return fmt.Errorf("failed to configure network (%s): %w", instance.Name, err)
		}

		// FIXME: this might need adjusting because most likely it will not work like this
		// peerConf := fmt.Sprintf("%s:%s:%s:%s:%s", instance.Name, pubkey.String(), instance.PublicIP, pubKey.IPv6Address().String(), pubKey.IPv6Address().String())
		// peerConfigs = append(peerConfigs, peerConf)

		instancePublicIP := net.ParseIP(instance.PublicIP)
		if instancePublicIP == nil {
			return fmt.Errorf("failed to parse CIDR while configuring VPN interface '%s': bad IP %s", instance.Name, "instance.PublicIP")
		}

		peerConf := wgtypes.PeerConfig{
			PublicKey:                   pubkey,
			PersistentKeepaliveInterval: &keepAliveInterval,
			Endpoint:                    &net.UDPAddr{IP: instancePublicIP, Port: 10999},
		}
		peers = append(peers, peerConf)
	}

	// if len(peerConfigs) > 0 {
	fmt.Println("PeerConfigs: %v", peers)
	err := m.configureLink(protosNetworkInterface, m.key.PrivateWG().String(), peers, routes)
	if err != nil {
		return fmt.Errorf("failed to configure link: %w", err)
	}
	// }

	return nil
}

func (m *Manager) createLink(iface string, address string) error {
	// create protos vpn interface and configure the address
	lnk, err := m.linkManager.CreateLink(iface)
	if err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to create VPN interface '%s': %w", iface, err)
		}
		lnk, err = m.linkManager.GetLink(iface)
		if err != nil {
			return fmt.Errorf("failed to create VPN interface '%s': %w", iface, err)
		}
	}

	ip, netp, err := net.ParseCIDR(fmt.Sprintf("%s/128", address))
	if err != nil {
		return fmt.Errorf("failed to parse CIDR while creating VPN interface '%s': %w", iface, err)
	}
	netp.IP = ip
	err = lnk.AddAddr(wireguard.Address{IPNet: *netp})
	if err != nil {
		return fmt.Errorf("failed to add address while creating VPN interface '%s': %w", iface, err)
	}

	return nil
}

func (m *Manager) deleteLink(iface string) error {
	// remove vpn interface
	err := m.linkManager.DelLink(iface)
	if err != nil {
		if !strings.Contains(err.Error(), "no such network interface") {
			return fmt.Errorf("failed to delete VPN interface '%s': %w", iface, err)
		}
	}

	return nil
}

func (m *Manager) configureLink(iface string, privateKey string, peers []wgtypes.PeerConfig, routes []wireguard.Route) error {

	// remove vpn interface
	lnk, err := m.linkManager.GetLink(iface)
	if err != nil {
		return fmt.Errorf("failed to configure VPN interface '%s': %w", iface, err)
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

func (m *Manager) addDomain(domain string, dnsServer string) error {
	dnsServerIP := net.ParseIP(dnsServer)
	if dnsServerIP == nil {
		return fmt.Errorf("failed to parse DNS server IP '%s'", dnsServer)
	}

	err := m.dnsManager.AddDomainServer(domain, dnsServerIP, 10053)
	if err != nil {
		return fmt.Errorf("failed to add domain: %w", err)
	}
	return nil
}

func (m *Manager) delDomain(domain string) error {
	err := m.dnsManager.DelDomainServer(domain)
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
	if peerInternalIP == nil {
		return "", wgtypes.Key{}, net.IP{}, net.IP{}, net.IPNet{}, fmt.Errorf("failed to parse internal IP in peer config '%s': %w", peerConfig, err)
	}
	_, peerNet, err := net.ParseCIDR(parts[4])
	if err != nil {
		return "", wgtypes.Key{}, net.IP{}, net.IP{}, net.IPNet{}, fmt.Errorf("failed to parse network in peer config '%s': %w", peerConfig, err)
	}

	return parts[0], wgPublicKey, peerPublicIP, peerInternalIP, *peerNet, nil
}
