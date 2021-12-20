package vpn

import (
	"fmt"
	"net"
	"os/exec"
	"time"

	"github.com/foxcpp/wirebox/linkmgr"
	"github.com/protosio/protos/internal/auth"
	"github.com/protosio/protos/internal/cloud"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/ssh"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const (
	instanceDS             = "instance"
	protosNetworkInterface = "protos0"
	wgProtosBinary         = "wg-protos"
)

type instanceGetter interface {
}

type VPN struct {
	nm linkmgr.Manager
	um *auth.UserManager
	cm cloud.CloudManager
	sm *ssh.Manager
}

func (vpn *VPN) Start() error {
	usr, err := vpn.um.GetAdmin()
	if err != nil {
		return fmt.Errorf("Failed to get admin while starting VPN: %w", err)
	}

	dev, err := usr.GetCurrentDevice()
	if err != nil {
		return fmt.Errorf("Failed to get current device while starting VPN: %w", err)
	}

	// create protos vpn interface and configure the address
	lnk, err := vpn.nm.CreateLink(protosNetworkInterface)
	if err != nil {
		return fmt.Errorf("Failed to create link while starting VPN: %w", err)
	}

	ip, netp, err := net.ParseCIDR(dev.Network)
	if err != nil {
		return fmt.Errorf("Failed to parse CIDR while starting VPN: %w", err)
	}
	netp.IP = ip
	err = lnk.AddAddr(linkmgr.Address{IPNet: *netp})
	if err != nil {
		return fmt.Errorf("Failed to add address while starting VPN: %w", err)
	}

	// create wireguard peer configurations and route list
	instances, err := vpn.cm.GetInstances()
	if err != nil {
		return fmt.Errorf("Failed to retrieve instances while starting VPN: %w", err)
	}
	var masterInstaceIP net.IP
	keepAliveInterval := 25 * time.Second
	peers := []wgtypes.PeerConfig{}
	routes := []linkmgr.Route{}
	for _, instance := range instances {

		pubkey, err := vpn.sm.ConvertPublicEd25519ToCurve25519(instance.PublicKey)
		if err != nil {
			return fmt.Errorf("Failed to start VPN for instance '%s': %w", instance.Name, err)
		}

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
	key, err := usr.GetKeyCurrentDevice()
	if err != nil {
		return fmt.Errorf("Failed to get device key while starting VPN: %w", err)
	}

	var pkey wgtypes.Key
	copy(pkey[:], key.Seed())
	wgcfg := wgtypes.Config{
		PrivateKey: &pkey,
		Peers:      peers,
	}
	err = lnk.ConfigureWG(wgcfg)
	if err != nil {
		return fmt.Errorf("Failed to configure WG interface while starting VPN: %w", err)
	}

	// add the routes towards instances
	for _, route := range routes {
		err = lnk.AddRoute(route)
		if err != nil {
			return fmt.Errorf("Failed to add route while starting VPN: %w", err)
		}
	}

	// add DNS server for domain
	dns, err := NewDNS()
	if err != nil {
		return fmt.Errorf("Failed to configure DNS while starting VPN: %w", err)
	}

	err = dns.AddDomainServer(usr.GetInfo().Domain, masterInstaceIP)
	if err != nil {
		return fmt.Errorf("Failed to add DNS domain while starting VPN: %w", err)
	}

	return nil
}

func (vpn *VPN) Stop() error {
	usr, err := vpn.um.GetAdmin()
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

	err = dns.DelDomainServer(usr.GetInfo().Domain)
	if err != nil {
		return err
	}

	return nil
}

func (vpn *VPN) StartWithExec() error {
	usr, err := vpn.um.GetAdmin()
	if err != nil {
		return fmt.Errorf("Failed to get admin while starting VPN: %w", err)
	}

	dev, err := usr.GetCurrentDevice()
	if err != nil {
		return fmt.Errorf("Failed to get current device while starting VPN: %w", err)
	}

	ip, netp, err := net.ParseCIDR(dev.Network)
	if err != nil {
		return fmt.Errorf("Failed to parse CIDR while starting VPN: %w", err)
	}
	netp.IP = ip

	// create wireguard peer configurations and route list
	instances, err := vpn.cm.GetInstances()
	if err != nil {
		return fmt.Errorf("Failed to retrieve instances while starting VPN: %w", err)
	}

	peerConfigs := []string{}
	var masterInstaceIP string

	for _, instance := range instances {

		if masterInstaceIP == "" {
			masterInstaceIP = instance.InternalIP
		}

		pubkey, err := vpn.sm.ConvertPublicEd25519ToCurve25519(instance.PublicKey)
		if err != nil {
			return fmt.Errorf("Failed to start VPN for instance '%s': %w", instance.Name, err)
		}

		_, instanceNetwork, err := net.ParseCIDR(instance.Network)
		if err != nil {
			return fmt.Errorf("Failed to parse network for instance '%s': %w", instance.Name, err)
		}

		peerConf := fmt.Sprintf("%s:%s:%s:%s:%s", instance.Name, pubkey.String(), instance.PublicIP, instance.InternalIP, instanceNetwork.String())
		peerConfigs = append(peerConfigs, peerConf)
	}

	cmd := exec.Command("sudo", wgProtosBinary, "wg", "up", protosNetworkInterface, netp.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to create link using wg-protos: \n---- wg-protos output ----\n%s-------------------", string(output))
	}

	key, err := usr.GetKeyCurrentDevice()
	if err != nil {
		return fmt.Errorf("Failed to get device key while starting VPN: %w", err)
	}

	if len(peerConfigs) > 0 {
		configureArgs := []string{wgProtosBinary, "wg", "configure", protosNetworkInterface, key.PrivateWG().String()}
		configureArgs = append(configureArgs, peerConfigs...)
		cmd = exec.Command("sudo", configureArgs...)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("Failed to configure link using wg-protos: \n---- wg-protos output ----\n%s-------------------", string(output))
		}
	}

	// add domain DNS
	if masterInstaceIP != "" {
		cmd = exec.Command("sudo", wgProtosBinary, "domain", "add", usr.GetInfo().Domain, masterInstaceIP)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("Failed to add domain using wg-protos: \n---- wg-protos output ----\n%s-------------------", string(output))
		}
	}

	return nil
}

func (vpn *VPN) StopWithExec() error {
	usr, err := vpn.um.GetAdmin()
	if err != nil {
		return fmt.Errorf("Failed to get admin while stopping VPN: %w", err)
	}

	cmd := exec.Command("sudo", wgProtosBinary, "wg", "down", protosNetworkInterface)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to delete link using wg-protos: \n---- wg-protos output ----\n%s-------------------", string(output))
	}

	// add domain DNS
	cmd = exec.Command("sudo", wgProtosBinary, "domain", "del", usr.GetInfo().Domain)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to delete domain using wg-protos: \n---- wg-protos output ----\n%s-------------------", string(output))
	}

	return nil
}

func New(db db.DB, um *auth.UserManager, cm cloud.CloudManager, sm *ssh.Manager) (*VPN, error) {
	linkManager, err := linkmgr.NewManager()
	if err != nil {
		return nil, err
	}
	return &VPN{um: um, cm: cm, nm: linkManager, sm: sm}, nil
}
