package vpn

import (
	"fmt"
	"net"
	"time"

	"github.com/foxcpp/wirebox/linkmgr"
	"github.com/protosio/protos/internal/cloud"
	"github.com/protosio/protos/internal/core"
	"github.com/protosio/protos/internal/user"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const (
	instanceDS             = "instance"
	protosNetworkInterface = "protos0"
)

type VPN struct {
	nm linkmgr.Manager
	db core.DB
}

func (vpn *VPN) Start() error {
	usr, err := user.Get(vpn.db)
	if err != nil {
		return err
	}

	// create protos vpn interface and configure the address
	lnk, err := vpn.nm.CreateLink(protosNetworkInterface)
	if err != nil {
		return err
	}
	ip, netp, err := net.ParseCIDR(usr.Device.Network)
	if err != nil {
		return err
	}
	netp.IP = ip
	err = lnk.AddAddr(linkmgr.Address{IPNet: *netp})
	if err != nil {
		return err
	}

	// create wireguard peer configurations and route list
	var instances []cloud.InstanceInfo
	err = vpn.db.GetSet(instanceDS, &instances)
	if err != nil {
		return err
	}
	var masterInstaceIP net.IP
	keepAliveInterval := 25 * time.Second
	peers := []wgtypes.PeerConfig{}
	routes := []linkmgr.Route{}
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
		routes = append(routes, linkmgr.Route{Dest: *instanceNetwork})

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
	dns, err := NewDNS()
	if err != nil {
		return err
	}

	err = dns.AddDomainServer(usr.Domain, masterInstaceIP)
	if err != nil {
		return err
	}

	return nil
}

func (vpn *VPN) Stop() error {
	usr, err := user.Get(vpn.db)
	if err != nil {
		return err
	}

	// remove VPN link
	_, err = vpn.nm.GetLink(protosNetworkInterface)
	if err != nil {
		return err
	}

	err = vpn.nm.DelLink(protosNetworkInterface)
	if err != nil {
		return err
	}

	// remove DNS server for domain
	dns, err := NewDNS()
	if err != nil {
		return err
	}

	err = dns.DelDomainServer(usr.Domain)
	if err != nil {
		return err
	}

	return nil
}

func New(db core.DB) (VPN, error) {
	manager, err := linkmgr.NewManager()
	if err != nil {
		return VPN{}, err
	}
	return VPN{db: db, nm: manager}, nil
}
