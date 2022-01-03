package network

import (
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"syscall"
	"time"

	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/foxcpp/wirebox"
	"github.com/foxcpp/wirebox/linkmgr"
	"github.com/protosio/protos/internal/auth"
	"github.com/protosio/protos/internal/cloud"
	"github.com/protosio/protos/internal/ssh"
	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const (
	wireguardNetworkInterface = "protosWG"
	bridgeNetworkInterface    = "protosBR"
	wgProtosBinary            = "wg-protos"
)

var wgPort int = 10999
var netBridge *netlink.Bridge

func compareRoutes(a netlink.Route, b netlink.Route) bool {
	if a.Dst.String() == b.Dst.String() && a.Src.Equal(b.Src) {
		return true
	}
	return false
}

// diffRoutes ignores IPv6 addresses at the moment
func diffRoutes(a []netlink.Route, b []netlink.Route) ([]netlink.Route, []netlink.Route) {
	extraA := []netlink.Route{}
	for _, ar := range a {
		matched := false
		for _, br := range b {
			if compareRoutes(ar, br) {
				matched = true
			}
		}
		if !matched && !strings.Contains(ar.Dst.IP.String(), ":") {
			extraA = append(extraA, ar)
		}
	}

	extraB := []netlink.Route{}
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

func configureBridge(name string, network net.IPNet) (*netlink.Bridge, error) {

	log.Debugf("Setting up bridge interface '%s'", bridgeNetworkInterface)
	brInterface := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name:   name,
			TxQLen: -1,
		},
	}

	err := netlink.LinkAdd(brInterface)
	if err != nil && err != syscall.EEXIST {
		return nil, fmt.Errorf("failed to create bridge interface '%q': %v", name, err)
	}

	l, err := netlink.LinkByName(name)
	if err != nil {
		return nil, fmt.Errorf("could not find newly created bridge interface '%q': %v", name, err)
	}
	brInterface, ok := l.(*netlink.Bridge)
	if !ok {
		return nil, fmt.Errorf("interface '%q' found but is not a bridge", name)
	}

	_, err = sysctl.Sysctl(fmt.Sprintf("net.ipv6.conf.%s.accept_ra", name), "0")
	if err != nil {
		return nil, fmt.Errorf("failed to disable ipv6 router ads on bridge interface '%s': %v", name, err)
	}

	if err := netlink.LinkSetUp(brInterface); err != nil {
		return nil, err
	}

	newRoutes := []netlink.Route{{Dst: &network, LinkIndex: l.Attrs().Index}}
	existingRoutes, err := netlink.RouteList(l, netlink.FAMILY_V4)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve routes: %w", err)
	}
	delRoutes, addRoutes := diffRoutes(existingRoutes, newRoutes)

	// add the new routes to the bridge interface
	for _, route := range addRoutes {
		route.LinkIndex = l.Attrs().Index
		err = netlink.RouteAdd(&route)
		if err != nil {
			return nil, fmt.Errorf("failed to add route: %w", err)
		}
	}

	// delete old routes from the bridge interface
	for _, route := range delRoutes {
		err = netlink.RouteDel(&route)
		if err != nil {
			return nil, fmt.Errorf("failed to delete route: %w", err)
		}
	}

	return brInterface, nil
}

func getNetNSInterfaceIP(nsPath string, filterNetwork net.IPNet) (net.IP, error) {
	var ipAddr net.IP
	fn := func(_ ns.NetNS) error {
		interfaces, err := net.Interfaces()
		if err != nil {
			return err
		}

		for _, iface := range interfaces {
			addresses, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addresses {
				ip, _, err := net.ParseCIDR(addr.String())
				if err != nil {
					continue
				}
				if filterNetwork.Contains(ip) {
					ipAddr = ip
					return nil
				}
			}
		}
		return nil
	}
	if err := ns.WithNetNSPath(nsPath, fn); err != nil {
		return ipAddr, err
	}
	return ipAddr, nil
}

// https://play.golang.org/p/m8TNTtygK0
func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

//
// public methods
//

// initNetwork initializes the protos network
func (m *Manager) Up() error {
	manager, err := linkmgr.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize network: %w", err)
	}

	_, err = sysctl.Sysctl("net.ipv4.ip_forward", "1")
	if err != nil {
		return fmt.Errorf("failed to set IP forwarding while initializing network: %w", err)
	}

	// configure the bridge interface
	netBridge, err = configureBridge(bridgeNetworkInterface, m.network)
	if err != nil {
		return fmt.Errorf("failed to create bridge interface during network initialization: %w", err)
	}

	// the instance gateway IP is also used for WG
	linkAddrs := []linkmgr.Address{
		{
			IPNet: net.IPNet{
				IP:   m.gateway,
				Mask: m.network.Mask,
			},
			Scope: linkmgr.ScopeLink,
		},
	}

	// create the wireguard interface
	cfg := wgtypes.Config{
		ReplacePeers: true,
		ListenPort:   &wgPort,
		PrivateKey:   &m.privateKey,
	}

	_, _, err = wirebox.CreateWG(manager, wireguardNetworkInterface, cfg, linkAddrs)
	if err != nil {
		return fmt.Errorf("failed to create WireGuard interface during network initialization: %w", err)
	}

	// cheating by sleeping 2 seconds
	log.Debugf("Waiting for link '%s' to come up", wireguardNetworkInterface)
	time.Sleep(2 * time.Second)

	return nil
}

func (m *Manager) Down() error {
	err := m.linkManager.DelLink(wireguardNetworkInterface)
	if err != nil {
		if !strings.Contains(err.Error(), "no such network interface") {
			return fmt.Errorf("failed to delete interface '%s': %w", wireguardNetworkInterface, err)
		}
	}

	br, err := netlink.LinkByName(bridgeNetworkInterface)
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("failed to retrieve interface '%s': %w", bridgeNetworkInterface, err)
		}
		return nil
	}
	err = netlink.LinkDel(br)
	if err != nil {
		return fmt.Errorf("failed to retrieve interface '%s': %w", bridgeNetworkInterface, err)
	}

	return nil
}

func (m *Manager) ConfigurePeers(instances []cloud.InstanceInfo, devices []auth.UserDevice) error {

	log.Debug("Configuring network")
	lnk, err := m.linkManager.GetLink(wireguardNetworkInterface)
	if err != nil {
		return fmt.Errorf("failed to configure interface '%s': %w", wireguardNetworkInterface, err)
	}

	// create the peer and routes lists. At the moment these are all the devices that a user has
	newRoutes := []netlink.Route{}
	peers := []wgtypes.PeerConfig{}
	if len(devices) == 0 {
		return fmt.Errorf("failed to configure interface because 0 user devices were provided")
	}

	// build instances peer list
	for _, instance := range instances {
		if len(instance.PublicKey) == 0 || m.network.String() == instance.Network {
			continue
		}

		publicKey, err := ssh.ConvertPublicEd25519ToCurve25519(instance.PublicKey)
		if err != nil {
			return fmt.Errorf("failed to configure network (%s): %w", instance.Name, err)
		}

		_, instanceNetwork, err := net.ParseCIDR(instance.Network)
		if err != nil {
			return fmt.Errorf("failed to parse network for instance '%s': %w", instance.Name, err)
		}

		newRoutes = append(newRoutes, netlink.Route{Dst: instanceNetwork, Src: m.gateway})
		peers = append(peers, wgtypes.PeerConfig{PublicKey: publicKey, ReplaceAllowedIPs: true, AllowedIPs: []net.IPNet{*instanceNetwork}})
	}

	// build devices peer list
	for _, userDevice := range devices {
		log.Debugf("Using route '%s' for device '%s(%s)'", userDevice.Network, userDevice.Name, userDevice.MachineID)
		publicKeyBytes, err := base64.StdEncoding.DecodeString(userDevice.PublicKey)
		if err != nil {
			return fmt.Errorf("failed to decode base64 encoded key for device '%s': %w", userDevice.Name, err)
		}
		_, deviceNetwork, err := net.ParseCIDR(userDevice.Network)
		if err != nil {
			return fmt.Errorf("failed to parse network for device '%s': %w", userDevice.Name, err)
		}
		newRoutes = append(newRoutes, netlink.Route{Dst: deviceNetwork, Src: m.gateway})
		var publicKey wgtypes.Key
		copy(publicKey[:], publicKeyBytes)

		peers = append(peers, wgtypes.PeerConfig{PublicKey: publicKey, ReplaceAllowedIPs: true, AllowedIPs: []net.IPNet{*deviceNetwork}})
	}

	// create the wireguard interface
	wgcfg := wgtypes.Config{
		ReplacePeers: true,
		ListenPort:   &wgPort,
		Peers:        peers,
		PrivateKey:   &m.privateKey,
	}
	err = lnk.ConfigureWG(wgcfg)
	if err != nil {
		return fmt.Errorf("failed to configure interface '%s': %w", wireguardNetworkInterface, err)
	}

	netlinkWG, err := netlink.LinkByName(wireguardNetworkInterface)
	if err != nil {
		return fmt.Errorf("failed to retrieve interface: %w", err)
	}

	existingRoutes, err := netlink.RouteList(netlinkWG, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("failed to retrieve routes: %w", err)
	}

	delRoutes, addRoutes := diffRoutes(existingRoutes, newRoutes)

	// add the new routes to the wireguard interface
	for _, route := range addRoutes {
		route.LinkIndex = netlinkWG.Attrs().Index
		err = netlink.RouteAdd(&route)
		if err != nil {
			return fmt.Errorf("failed to add route: %w", err)
		}
	}

	// delete old routes from the wireguard interface
	for _, route := range delRoutes {
		err = netlink.RouteDel(&route)
		if err != nil {
			return fmt.Errorf("failed to delete route: %w", err)
		}
	}
	return nil
}

func (m *Manager) AllocateIP(networkNamespaces []string) (net.IP, error) {

	usedIPs := map[string]bool{}
	for _, netNSPath := range networkNamespaces {
		ip, err := getNetNSInterfaceIP(netNSPath, m.network)
		if err != nil {
			log.Errorf("Failed to retrieve IP for network namespace '%s': %s", netNSPath, err.Error())
		}
		if ip != nil {
			usedIPs[ip.String()] = true
		}
	}

	allIPs := []net.IP{}
	for ip := m.network.IP.Mask(m.network.Mask); m.network.Contains(ip); incIP(ip) {
		newIP := make(net.IP, len(ip))
		copy(newIP, ip)
		allIPs = append(allIPs, newIP)
	}

	// starting from the 4th position in the slice to avoid allocating the network IP, WG and bridge interface IPs
	for _, ip := range allIPs[3 : len(allIPs)-1] {
		if _, found := usedIPs[ip.String()]; !found {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("failed to allocate IP. No IP's left")
}

func (m *Manager) CreateNamespacedInterface(netNSpath string, IP net.IP) error {
	netns, err := ns.GetNS(netNSpath)
	if err != nil {
		return fmt.Errorf("failed to open netns '%s': %v", netNSpath, err)
	}
	defer netns.Close()

	contIface := &current.Interface{}
	hostIface := &current.Interface{}
	err = netns.Do(func(hostNS ns.NetNS) error {
		// create the veth pair in the container and move host end into host netns
		name := "prts0"
		hostVeth, containerVeth, err := ip.SetupVeth(name, netBridge.MTU, hostNS)
		if err != nil {
			return err
		}
		contIface.Name = containerVeth.Name
		contIface.Mac = containerVeth.HardwareAddr.String()
		contIface.Sandbox = netns.Path()
		hostIface.Name = hostVeth.Name

		link, err := netlink.LinkByName(name)
		if err != nil {
			return fmt.Errorf("failed to find interface %q: %v", name, err)
		}

		if err := netlink.LinkSetUp(link); err != nil {
			return fmt.Errorf("failed to bring interface %q UP: %v", name, err)
		}

		addr := &netlink.Addr{IPNet: &net.IPNet{Mask: m.network.Mask, IP: IP}, Label: ""}
		if err = netlink.AddrAdd(link, addr); err != nil {
			return fmt.Errorf("failed to configure IP address '%s' on interface: %v", IP.String(), err)
		}

		_, networkALL, _ := net.ParseCIDR("0.0.0.0/0")
		route := netlink.Route{
			LinkIndex: link.Attrs().Index,
			Gw:        m.gateway,
			Dst:       networkALL,
		}
		err = netlink.RouteAdd(&route)
		if err != nil {
			return fmt.Errorf("failed to add route on interface: %v", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create veth pair: %v", err)
	}

	hostVeth, err := netlink.LinkByName(hostIface.Name)
	if err != nil {
		return fmt.Errorf("failed to find host interface '%s': %v", hostIface.Name, err)
	}
	hostIface.Mac = hostVeth.Attrs().HardwareAddr.String()

	if err := netlink.LinkSetMaster(hostVeth, netBridge); err != nil {
		return fmt.Errorf("failed to connect %q to bridge %v: %v", hostVeth.Attrs().Name, netBridge.Attrs().Name, err)
	}
	return nil
}