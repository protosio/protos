//go:build linux

package wireguard

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	networkmodule "github.com/protosio/protos/internal/network/module"
	wglink "github.com/protosio/protos/internal/wireguard"
	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const (
	wireguardNetworkInterfaceName = "protosWG"
	bridgeNetworkInterface        = "protosBR"
	bridgeMTU                     = 1500
	localMacOSNATGatewayHost      = 1
)

var wgPort int = 10999
var netBridge *netlink.Bridge

var _ networkmodule.NamespacedInterfaceModule = (*Module)(nil)

type platformState struct {
	linkManager wglink.Manager
}

func (m *Module) closePlatform() error {
	if m.linkManager == nil {
		return nil
	}
	return m.linkManager.Close()
}

func compareRoutes(a netlink.Route, b netlink.Route) bool {
	if a.Dst == nil || b.Dst == nil {
		return a.Dst == nil && b.Dst == nil
	}
	return a.Dst.String() == b.Dst.String()
}

func diffRoutes(a []netlink.Route, b []netlink.Route) ([]netlink.Route, []netlink.Route) {
	extraA := []netlink.Route{}
	for _, ar := range a {
		matched := false
		for _, br := range b {
			if compareRoutes(ar, br) {
				matched = true
			}
		}
		if !matched {
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

func diffManagedRoutes(existingRoutes []netlink.Route, desiredRoutes []netlink.Route, localIP net.IP) ([]netlink.Route, []netlink.Route) {
	return diffRoutes(managedWireGuardRoutes(existingRoutes, localIP), desiredRoutes)
}

func managedWireGuardRoutes(routes []netlink.Route, localIP net.IP) []netlink.Route {
	managed := make([]netlink.Route, 0, len(routes))
	for _, route := range routes {
		if route.Dst == nil || route.Dst.IP == nil {
			continue
		}
		ip := route.Dst.IP
		if ip.Equal(localIP) || ip.To4() != nil || !ip.IsGlobalUnicast() {
			continue
		}
		ones, bits := route.Dst.Mask.Size()
		if bits != net.IPv6len*8 || ones != bits {
			continue
		}
		managed = append(managed, route)
	}
	return managed
}

func (m *Module) Up(config networkmodule.Config) error {
	if err := validateConfig(config); err != nil {
		return err
	}
	m.domain = config.Domain

	_, err := sysctl.Sysctl("net.ipv4.ip_forward", "1")
	if err != nil {
		return fmt.Errorf("failed to set IPv4 forwarding while initializing network: %w", err)
	}
	if _, err := sysctl.Sysctl("net.ipv6.conf.all.forwarding", "1"); err != nil {
		return fmt.Errorf("failed to set IPv6 forwarding while initializing network: %w", err)
	}
	if _, err := sysctl.Sysctl("net.ipv6.conf.default.forwarding", "1"); err != nil {
		return fmt.Errorf("failed to set default IPv6 forwarding while initializing network: %w", err)
	}
	if err := m.ensureBridge(); err != nil {
		return err
	}

	network := createIPv6Net(config.IPv6Address)
	linkAddrs := []wglink.Address{
		{
			IPNet: net.IPNet{
				IP:   network.IP,
				Mask: network.Mask,
			},
			Scope: wglink.ScopeGlobal,
		},
	}

	wgKey, err := privateWireGuardKey(config.WireGuardPrivateKey)
	if err != nil {
		return fmt.Errorf("failed to initialize network: %w", err)
	}
	wgConfig := wgtypes.Config{
		ReplacePeers: false,
		ListenPort:   &wgPort,
		PrivateKey:   &wgKey,
	}

	_, _, err = wglink.CreateWG(m.linkManager, wireguardNetworkInterfaceName, wgConfig, linkAddrs)
	if err != nil {
		return fmt.Errorf("failed to create WireGuard interface during network initialization: %w", err)
	}

	log.Debugf("Waiting for link '%s' to come up", wireguardNetworkInterfaceName)
	time.Sleep(2 * time.Second)

	return nil
}

func (m *Module) ensureBridge() error {
	if netBridge != nil {
		if _, err := netlink.LinkByName(netBridge.Attrs().Name); err == nil {
			return setBridgeUp(netBridge)
		}
		netBridge = nil
	}

	link, err := netlink.LinkByName(bridgeNetworkInterface)
	if err == nil {
		bridge, ok := link.(*netlink.Bridge)
		if !ok {
			return fmt.Errorf("interface '%s' exists but is %T, not a bridge", bridgeNetworkInterface, link)
		}
		netBridge = bridge
		return setBridgeUp(bridge)
	}
	if !strings.Contains(err.Error(), "not found") {
		return fmt.Errorf("failed to retrieve interface '%s': %w", bridgeNetworkInterface, err)
	}

	bridge := &netlink.Bridge{LinkAttrs: netlink.LinkAttrs{
		Name: bridgeNetworkInterface,
		MTU:  bridgeMTU,
	}}
	if err := netlink.LinkAdd(bridge); err != nil {
		return fmt.Errorf("failed to create bridge '%s': %w", bridgeNetworkInterface, err)
	}
	created, err := netlink.LinkByName(bridgeNetworkInterface)
	if err != nil {
		return fmt.Errorf("failed to retrieve created bridge '%s': %w", bridgeNetworkInterface, err)
	}
	createdBridge, ok := created.(*netlink.Bridge)
	if !ok {
		return fmt.Errorf("created interface '%s' is %T, not a bridge", bridgeNetworkInterface, created)
	}
	netBridge = createdBridge
	return setBridgeUp(createdBridge)
}

func setBridgeUp(bridge *netlink.Bridge) error {
	if bridge == nil {
		return fmt.Errorf("bridge is nil")
	}
	if err := netlink.LinkSetUp(bridge); err != nil {
		return fmt.Errorf("failed to bring bridge '%s' up: %w", bridge.Attrs().Name, err)
	}
	return nil
}

func (m *Module) Down() error {
	err := m.linkManager.DelLink(wireguardNetworkInterfaceName)
	if err != nil {
		if !strings.Contains(err.Error(), "no such network interface") {
			return fmt.Errorf("failed to delete interface '%s': %w", wireguardNetworkInterfaceName, err)
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
	netBridge = nil

	return nil
}

func (m *Module) ConfigurePeers(config networkmodule.Config, peerSet networkmodule.Peers) error {
	if err := validateConfig(config); err != nil {
		return err
	}

	if config.Domain == "" {
		log.Debugf("Skipping peer configuration because the network is not configured yet")
		return nil
	}

	log.Debug("Refreshing network configuration for peers")
	lnk, err := m.linkManager.GetLink(wireguardNetworkInterfaceName)
	if err != nil {
		return fmt.Errorf("failed to configure interface '%s': %w", wireguardNetworkInterfaceName, err)
	}

	newRoutes := []netlink.Route{}
	wgPeers := []wgtypes.PeerConfig{}
	activeDevice := activeWireGuardDevice(lnk)
	existingPeerEndpoints := peerEndpoints(activeDevice)
	relayAllowedIPs := []net.IPNet{}
	var relayEndpoint *net.UDPAddr
	relayLocalMacOSPeers := localMacOSNATAttached()
	if len(peerSet.Devices) == 0 {
		log.Debug("Configuring WireGuard with zero user device peers; stale peers and routes will be removed")
	}

	for _, instance := range peerSet.Instances {
		if len(instance.PublicKey) == 0 {
			continue
		}

		pubKeyAddr, err := ipv6AddressFromPublicKeyBase64(instance.PublicKey)
		if err != nil {
			return fmt.Errorf("failed to configure network (%s): %w", instance.Name, err)
		}

		pubKeyWG, err := publicEd25519ToWireGuard(instance.PublicKey)
		if err != nil {
			return fmt.Errorf("failed to configure network (%s): %w", instance.Name, err)
		}

		instancePublicIP := net.ParseIP(instance.PublicIP)
		if instancePublicIP == nil {
			return fmt.Errorf("failed to parse CIDR while configuring VPN interface '%s': bad IP %s", instance.Name, "instance.PublicIP")
		}

		instanceInternalNet := *createIPv6Net(pubKeyAddr)
		newRoutes = append(newRoutes, netlink.Route{Dst: &instanceInternalNet, LinkIndex: lnk.Index()})
		if isLocalMacOSNATIP(instancePublicIP) {
			if relayLocalMacOSPeers {
				relayAllowedIPs = append(relayAllowedIPs, instanceInternalNet)
				if relayEndpoint == nil {
					relayEndpoint = &net.UDPAddr{IP: localMacOSNATGateway(instancePublicIP), Port: wgPort}
				}
				log.Debugf("Routing local macOS VM peer %s (%s) through host relay endpoint %s", instance.Name, instanceInternalNet.String(), relayEndpoint.String())
				continue
			}
			endpoint := existingPeerEndpoints[pubKeyWG]
			var keepalive *time.Duration
			if endpoint != nil {
				interval := peerKeepaliveInterval()
				keepalive = &interval
				log.Debugf("Preserving learned endpoint %s for roaming local macOS VM peer %s (%s)", endpoint.String(), instance.Name, instanceInternalNet.String())
			} else {
				log.Debugf("Routing local macOS VM peer %s (%s) without a fixed endpoint; waiting for WireGuard roaming", instance.Name, instanceInternalNet.String())
			}
			wgPeers = append(wgPeers, wgtypes.PeerConfig{
				PublicKey:                   pubKeyWG,
				ReplaceAllowedIPs:           true,
				AllowedIPs:                  []net.IPNet{instanceInternalNet},
				Endpoint:                    endpoint,
				PersistentKeepaliveInterval: keepalive,
			})
			continue
		}

		keepalive := peerKeepaliveInterval()
		peerConf := wgtypes.PeerConfig{
			PublicKey:                   pubKeyWG,
			ReplaceAllowedIPs:           true,
			Endpoint:                    &net.UDPAddr{IP: instancePublicIP, Port: wgPort},
			AllowedIPs:                  []net.IPNet{instanceInternalNet},
			PersistentKeepaliveInterval: &keepalive,
		}

		wgPeers = append(wgPeers, peerConf)
	}

	if len(relayAllowedIPs) > 0 && len(peerSet.Devices) == 0 {
		log.Warnf("Local macOS relay routes are desired but no user device peer is available to relay %d routes", len(relayAllowedIPs))
	}
	for deviceIndex, userDevice := range peerSet.Devices {
		pubKeyAddr, err := ipv6AddressFromPublicKeyBase64(userDevice.PublicKey)
		if err != nil {
			return fmt.Errorf("failed to decode base64 encoded key for device '%s': %w", userDevice.Name, err)
		}

		publicKeyWG, err := publicEd25519ToWireGuard(userDevice.PublicKey)
		if err != nil {
			return fmt.Errorf("failed to decode base64 encoded key for device '%s': %w", userDevice.Name, err)
		}

		instanceInternalNet := *createIPv6Net(pubKeyAddr)
		allowedIPs := []net.IPNet{instanceInternalNet}
		var endpoint *net.UDPAddr
		var keepalive *time.Duration
		if deviceIndex == 0 && len(relayAllowedIPs) > 0 {
			allowedIPs = append(allowedIPs, relayAllowedIPs...)
			endpoint = relayEndpoint
			interval := peerKeepaliveInterval()
			keepalive = &interval
		}
		peerConf := wgtypes.PeerConfig{
			PublicKey:                   publicKeyWG,
			ReplaceAllowedIPs:           true,
			AllowedIPs:                  allowedIPs,
			Endpoint:                    endpoint,
			PersistentKeepaliveInterval: keepalive,
		}
		wgPeers = append(wgPeers, peerConf)
		newRoutes = append(newRoutes, netlink.Route{Dst: &instanceInternalNet, LinkIndex: lnk.Index()})
	}
	wgPeers = appendStalePeerRemovals(wgPeers, activeDevice)
	log.Debugf("Applying WireGuard peer set on %s with peers=%d desired_routes=%d relay_routes=%d", wireguardNetworkInterfaceName, len(wgPeers), len(newRoutes), len(relayAllowedIPs))
	logLinuxPeerConfigs(wgPeers)

	wgKey, err := privateWireGuardKey(config.WireGuardPrivateKey)
	if err != nil {
		return fmt.Errorf("failed to configure interface '%s': %w", wireguardNetworkInterfaceName, err)
	}
	wgConfig := wgtypes.Config{
		ReplacePeers: false,
		ListenPort:   &wgPort,
		Peers:        wgPeers,
		PrivateKey:   &wgKey,
	}
	err = lnk.ConfigureWG(wgConfig)
	if err != nil {
		return fmt.Errorf("failed to configure interface '%s': %w", wireguardNetworkInterfaceName, err)
	}
	logActiveWireGuardPeers(lnk)

	netlinkWG, err := netlink.LinkByName(wireguardNetworkInterfaceName)
	if err != nil {
		return fmt.Errorf("failed to retrieve interface: %w", err)
	}

	existingRoutes, err := netlink.RouteList(netlinkWG, netlink.FAMILY_V6)
	if err != nil {
		return fmt.Errorf("failed to retrieve routes: %w", err)
	}

	delRoutes, addRoutes := diffManagedRoutes(existingRoutes, newRoutes, config.IPv6Address.AsSlice())

	for _, route := range addRoutes {
		route.LinkIndex = netlinkWG.Attrs().Index
		err = netlink.RouteAdd(&route)
		if err != nil {
			return fmt.Errorf("failed to add route: %w", err)
		}
		log.Debugf("Added WireGuard route %s through %s", routeString(route), wireguardNetworkInterfaceName)
	}

	for _, route := range delRoutes {
		err = netlink.RouteDel(&route)
		if err != nil {
			return fmt.Errorf("failed to delete route: %w", err)
		}
		log.Debugf("Deleted stale WireGuard route %s through %s", routeString(route), wireguardNetworkInterfaceName)
	}
	if len(addRoutes) > 0 || len(delRoutes) > 0 {
		log.Debugf("Reconciled WireGuard routes on %s: added=%d deleted=%d desired=%d", wireguardNetworkInterfaceName, len(addRoutes), len(delRoutes), len(newRoutes))
	}
	return nil
}

func peerKeepaliveInterval() time.Duration {
	return 25 * time.Second
}

func activeWireGuardDevice(lnk wglink.Link) *wgtypes.Device {
	device, err := lnk.WGConfig()
	if err != nil {
		log.Debugf("Failed to read active WireGuard peer state from %s: %v", lnk.Name(), err)
		return nil
	}
	return device
}

func peerEndpoints(device *wgtypes.Device) map[wgtypes.Key]*net.UDPAddr {
	endpoints := map[wgtypes.Key]*net.UDPAddr{}
	if device == nil {
		return endpoints
	}
	for _, peer := range device.Peers {
		if peer.Endpoint == nil {
			continue
		}
		endpoints[peer.PublicKey] = copyUDPAddr(peer.Endpoint)
	}
	return endpoints
}

func appendStalePeerRemovals(desired []wgtypes.PeerConfig, activeDevice *wgtypes.Device) []wgtypes.PeerConfig {
	if activeDevice == nil {
		return desired
	}
	desiredKeys := make(map[wgtypes.Key]struct{}, len(desired))
	for _, peer := range desired {
		desiredKeys[peer.PublicKey] = struct{}{}
	}
	for _, peer := range activeDevice.Peers {
		if _, found := desiredKeys[peer.PublicKey]; found {
			continue
		}
		desired = append(desired, wgtypes.PeerConfig{
			PublicKey: peer.PublicKey,
			Remove:    true,
		})
	}
	return desired
}

func copyUDPAddr(addr *net.UDPAddr) *net.UDPAddr {
	if addr == nil {
		return nil
	}
	return &net.UDPAddr{
		IP:   append(net.IP(nil), addr.IP...),
		Port: addr.Port,
		Zone: addr.Zone,
	}
}

func logLinuxPeerConfigs(peers []wgtypes.PeerConfig) {
	for _, peer := range peers {
		if peer.Remove {
			log.Debugf("WireGuard peer desired: public_key=%s remove=true", shortWireGuardKey(peer.PublicKey))
			continue
		}
		endpoint := "<none>"
		if peer.Endpoint != nil {
			endpoint = peer.Endpoint.String()
		}
		keepalive := time.Duration(0)
		if peer.PersistentKeepaliveInterval != nil {
			keepalive = *peer.PersistentKeepaliveInterval
		}
		log.Debugf(
			"WireGuard peer desired: public_key=%s endpoint=%s allowed_ips=%s keepalive=%s",
			shortWireGuardKey(peer.PublicKey),
			endpoint,
			strings.Join(ipNetStrings(peer.AllowedIPs), ","),
			keepalive,
		)
	}
}

func logActiveWireGuardPeers(lnk wglink.Link) {
	device, err := lnk.WGConfig()
	if err != nil {
		log.Debugf("Failed to read active WireGuard peer state from %s: %v", lnk.Name(), err)
		return
	}
	log.Debugf("WireGuard active state on %s: peers=%d", lnk.Name(), len(device.Peers))
	for _, peer := range device.Peers {
		endpoint := "<none>"
		if peer.Endpoint != nil {
			endpoint = peer.Endpoint.String()
		}
		log.Debugf(
			"WireGuard peer active: public_key=%s endpoint=%s allowed_ips=%s latest_handshake=%s tx=%d rx=%d",
			shortWireGuardKey(peer.PublicKey),
			endpoint,
			strings.Join(ipNetStrings(peer.AllowedIPs), ","),
			peer.LastHandshakeTime.Format(time.RFC3339),
			peer.TransmitBytes,
			peer.ReceiveBytes,
		)
	}
}

func ipNetStrings(nets []net.IPNet) []string {
	values := make([]string, 0, len(nets))
	for _, ipNet := range nets {
		values = append(values, ipNet.String())
	}
	return values
}

func routeString(route netlink.Route) string {
	if route.Dst == nil {
		return "<default>"
	}
	return route.Dst.String()
}

func shortWireGuardKey(key wgtypes.Key) string {
	value := key.String()
	if len(value) <= 12 {
		return value
	}
	return value[:12]
}

func isLocalMacOSNATIP(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	return ip4[0] == 192 && ip4[1] == 168 && ip4[2] == 64
}

func localMacOSNATGateway(ip net.IP) net.IP {
	ip4 := ip.To4()
	if ip4 == nil {
		return nil
	}
	gateway := append(net.IP(nil), ip4...)
	gateway[3] = localMacOSNATGatewayHost
	return gateway
}

func localMacOSNATAttached() bool {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Debugf("failed to inspect interfaces for local macOS NAT attachment: %v", err)
		return false
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			log.Debugf("failed to inspect addresses for interface %s: %v", iface.Name, err)
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch value := addr.(type) {
			case *net.IPNet:
				ip = value.IP
			case *net.IPAddr:
				ip = value.IP
			}
			if ip != nil && isLocalMacOSNATIP(ip) {
				return true
			}
		}
	}
	return false
}

func (m *Module) CreateNamespacedInterface(config networkmodule.Config, netNSpath string, IP net.IP) error {
	if err := validateConfig(config); err != nil {
		return err
	}
	if err := m.ensureBridge(); err != nil {
		return err
	}

	netns, err := ns.GetNS(netNSpath)
	if err != nil {
		return fmt.Errorf("failed to open netns '%s': %w", netNSpath, err)
	}
	defer netns.Close()

	hostIfaceName := ""
	err = netns.Do(func(hostNS ns.NetNS) error {
		name := "prts0"
		link, err := netlink.LinkByName(name)
		if err == nil {
			return configureNamespacedLink(link, config, IP)
		}
		if !strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("failed to inspect interface %q: %w", name, err)
		}

		hostVeth, containerVeth, err := ip.SetupVeth(name, netBridge.MTU, "", hostNS)
		if err != nil {
			return err
		}
		hostIfaceName = hostVeth.Name

		link, err = netlink.LinkByName(containerVeth.Name)
		if err != nil {
			return fmt.Errorf("failed to find interface %q: %w", containerVeth.Name, err)
		}

		return configureNamespacedLink(link, config, IP)
	})
	if err != nil {
		return fmt.Errorf("failed to create veth pair: %w", err)
	}
	if hostIfaceName == "" {
		return nil
	}

	hostVeth, err := netlink.LinkByName(hostIfaceName)
	if err != nil {
		return fmt.Errorf("failed to find host interface '%s': %w", hostIfaceName, err)
	}

	if err := netlink.LinkSetMaster(hostVeth, netBridge); err != nil {
		return fmt.Errorf("failed to connect %q to bridge %v: %w", hostVeth.Attrs().Name, netBridge.Attrs().Name, err)
	}
	return nil
}

func configureNamespacedLink(link netlink.Link, config networkmodule.Config, IP net.IP) error {
	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("failed to bring interface %q UP: %w", link.Attrs().Name, err)
	}

	network := createIPv6Net(config.IPv6Address)
	addr := &netlink.Addr{IPNet: &net.IPNet{Mask: network.Mask, IP: IP}, Label: ""}
	if err := netlink.AddrReplace(link, addr); err != nil {
		return fmt.Errorf("failed to configure IP address '%s' on interface: %w", IP.String(), err)
	}

	_, networkALL, _ := net.ParseCIDR("0.0.0.0/0")
	route := netlink.Route{
		LinkIndex: link.Attrs().Index,
		Dst:       networkALL,
	}
	if err := netlink.RouteReplace(&route); err != nil {
		return fmt.Errorf("failed to configure route on interface: %w", err)
	}
	return nil
}
