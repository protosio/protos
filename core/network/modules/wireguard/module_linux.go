//go:build linux

package wireguard

import (
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

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
	namespacedGatewayIPv6         = "fe80::7072:6f74:6f73"
)

var wgPort int = 10999
var netBridge *netlink.Bridge

var _ networkmodule.NamespacedInterfaceModule = (*Module)(nil)

type platformState struct {
	linkManager wglink.Manager
	exitGateway bool
}

func (m *Module) closePlatform() error {
	if m.dnsManager != nil {
		if err := m.dnsManager.Close(); err != nil {
			return err
		}
	}
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

func diffManagedRoutes(existingRoutes []netlink.Route, desiredRoutes []netlink.Route, localIPv6 net.IP, localIPv4 net.IP) ([]netlink.Route, []netlink.Route) {
	return diffRoutes(managedWireGuardRoutes(existingRoutes, localIPv6, localIPv4), desiredRoutes)
}

func managedWireGuardRoutes(routes []netlink.Route, localIPv6 net.IP, localIPv4 net.IP) []netlink.Route {
	managed := make([]netlink.Route, 0, len(routes))
	for _, route := range routes {
		if route.Dst == nil || route.Dst.IP == nil {
			continue
		}
		ip := route.Dst.IP
		if localIPv6 != nil && ip.Equal(localIPv6) {
			continue
		}
		if localIPv4 != nil && ip.Equal(localIPv4) {
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
	if err := configureBridgeGateway(bridge); err != nil {
		return err
	}
	return nil
}

func configureBridgeGateway(bridge netlink.Link) error {
	gatewayNet, err := namespacedGatewayNet()
	if err != nil {
		return err
	}
	if err := netlink.AddrReplace(bridge, &netlink.Addr{IPNet: gatewayNet}); err != nil {
		return fmt.Errorf("failed to configure bridge gateway '%s' on %s: %w", gatewayNet.String(), bridge.Attrs().Name, err)
	}
	return nil
}

func (m *Module) Down() error {
	if err := m.syncExitGateway(nil); err != nil {
		log.Warnf("failed to disable exit gateway rules: %v", err)
	}
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
	exitGatewayRoutesByDevice, err := exitGatewayRouteCIDRsByDevice(config, peerSet)
	if err != nil {
		return fmt.Errorf("failed to resolve exit gateway routes: %w", err)
	}
	exitGatewayRoutes := []exitGatewayRoute{}
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
		if pubKeyAddr == config.IPv6Address {
			log.Debugf("Skipping local instance peer %s (%s)", instance.Name, pubKeyAddr.String())
			continue
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
		allowedIPs := []net.IPNet{instanceInternalNet}
		newRoutes = append(newRoutes, netlink.Route{Dst: &instanceInternalNet, LinkIndex: lnk.Index()})
		for _, routeAddr := range instance.Routes {
			if routeAddr == pubKeyAddr || routeAddr == config.IPv6Address {
				continue
			}
			routeNet := *createIPv6Net(routeAddr)
			allowedIPs = append(allowedIPs, routeNet)
			newRoutes = append(newRoutes, netlink.Route{Dst: &routeNet, LinkIndex: lnk.Index()})
		}
		keepalive := peerKeepaliveInterval()
		peerConf := wgtypes.PeerConfig{
			PublicKey:                   pubKeyWG,
			ReplaceAllowedIPs:           true,
			Endpoint:                    &net.UDPAddr{IP: instancePublicIP, Port: wgPort},
			AllowedIPs:                  allowedIPs,
			PersistentKeepaliveInterval: &keepalive,
		}

		wgPeers = append(wgPeers, peerConf)
	}

	for _, userDevice := range peerSet.Devices {
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
		if exitCIDRs, ok := exitGatewayRoutesByDevice[userDevice.ID]; ok {
			gatewayRoute := exitGatewayRoute{
				deviceID:         userDevice.ID,
				deviceName:       userDevice.Name,
				sourceIPv6CIDR:   instanceInternalNet.String(),
				destinationCIDRs: exitCIDRs,
			}
			if exitRouteCIDRsNeedIPv4(exitCIDRs) {
				ipv4Net := ipNetFromAddr(userDevice.IPv4Address, net.IPv4len*8)
				if ipv4Net == nil {
					return fmt.Errorf("failed to configure exit routing for device '%s': missing tunnel IPv4 address", userDevice.Name)
				}
				gatewayRoute.sourceIPv4CIDR = ipv4Net.String()
				allowedIPs = append(allowedIPs, *ipv4Net)
				newRoutes = append(newRoutes, netlink.Route{Dst: ipv4Net, LinkIndex: lnk.Index()})
			}
			if gatewayRoute.sourceIPv4CIDR == "" && gatewayRoute.sourceIPv6CIDR == "" {
				return fmt.Errorf("failed to configure exit routing for device '%s': missing tunnel address", userDevice.Name)
			}
			exitGatewayRoutes = append(exitGatewayRoutes, gatewayRoute)
			log.Debugf("Allowing device %s to use this VM as an exit gateway for %s", userDevice.Name, strings.Join(exitCIDRs, ","))
		}
		var endpoint *net.UDPAddr
		var keepalive *time.Duration
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
	log.Debugf("Applying WireGuard peer set on %s with peers=%d desired_routes=%d", wireguardNetworkInterfaceName, len(wgPeers), len(newRoutes))
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

	existingRoutes, err := netlink.RouteList(netlinkWG, netlink.FAMILY_ALL)
	if err != nil {
		return fmt.Errorf("failed to retrieve routes: %w", err)
	}

	var localIPv4 net.IP
	if config.IPv4Address.IsValid() {
		localIPv4 = config.IPv4Address.AsSlice()
	}
	delRoutes, addRoutes := diffManagedRoutes(existingRoutes, newRoutes, config.IPv6Address.AsSlice(), localIPv4)

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
	if err := m.syncExitGateway(exitGatewayRoutes); err != nil {
		return err
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

func (m *Module) syncExitGateway(routes []exitGatewayRoute) error {
	if len(routes) > 0 {
		if err := ensureExitGatewayRules(routes); err != nil {
			return err
		}
		m.exitGateway = true
		return nil
	}
	if err := deleteExitGatewayRules(); err != nil {
		return err
	}
	m.exitGateway = false
	return nil
}

func ensureExitGatewayRules(routes []exitGatewayRoute) error {
	v4Iface, _ := defaultRouteInterface(netlink.FAMILY_V4)
	v6Iface, _ := defaultRouteInterface(netlink.FAMILY_V6)
	if v4Iface == "" && v6Iface == "" {
		return fmt.Errorf("failed to find a default route interface for exit gateway")
	}
	return replaceExitGatewayRules(routes, v4Iface, v6Iface)
}

func deleteExitGatewayRules() error {
	return clearExitGatewayRules()
}

func defaultRouteInterface(family int) (string, error) {
	routes, err := netlink.RouteList(nil, family)
	if err == nil {
		if name := defaultRouteInterfaceFromRoutes(routes); name != "" {
			return name, nil
		}
	}
	if name := defaultRouteInterfaceFromProc(family); name != "" {
		return name, nil
	}
	if err != nil {
		return "", err
	}
	return "", fmt.Errorf("default route not found")
}

func defaultRouteInterfaceFromRoutes(routes []netlink.Route) string {
	for _, route := range routes {
		if !isDefaultRouteDst(route.Dst) {
			continue
		}
		link, err := netlink.LinkByIndex(route.LinkIndex)
		if err != nil {
			continue
		}
		if link.Attrs().Name == wireguardNetworkInterfaceName {
			continue
		}
		return link.Attrs().Name
	}
	return ""
}

func isDefaultRouteDst(dst *net.IPNet) bool {
	if dst == nil {
		return true
	}
	ones, _ := dst.Mask.Size()
	return ones == 0
}

func defaultRouteInterfaceFromProc(family int) string {
	switch family {
	case netlink.FAMILY_V4:
		data, err := os.ReadFile("/proc/net/route")
		if err != nil {
			return ""
		}
		return parseProcDefaultRoute(data, false)
	case netlink.FAMILY_V6:
		data, err := os.ReadFile("/proc/net/ipv6_route")
		if err != nil {
			return ""
		}
		return parseProcDefaultRoute(data, true)
	default:
		return ""
	}
}

func parseProcDefaultRoute(data []byte, ipv6 bool) string {
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] == "Iface" {
			continue
		}
		if ipv6 {
			if len(fields) < 10 {
				continue
			}
			if fields[0] != strings.Repeat("0", 32) || fields[1] != "00000000" {
				continue
			}
			if fields[9] != wireguardNetworkInterfaceName {
				return fields[9]
			}
			continue
		}
		if len(fields) < 2 {
			continue
		}
		if fields[1] != "00000000" {
			continue
		}
		if fields[0] != wireguardNetworkInterfaceName {
			return fields[0]
		}
	}
	return ""
}

func shortWireGuardKey(key wgtypes.Key) string {
	value := key.String()
	if len(value) <= 12 {
		return value
	}
	return value[:12]
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
	hostIfaceIndex := 0
	err = netns.Do(func(hostNS ns.NetNS) error {
		name := "prts0"
		link, err := netlink.LinkByName(name)
		if err == nil {
			hostIfaceIndex = link.Attrs().ParentIndex
			return configureNamespacedLink(link, config, IP)
		}
		if !strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("failed to inspect interface %q: %w", name, err)
		}

		hostVethName, containerVethName, err := setupVeth(name, netBridge.MTU, hostNS)
		if err != nil {
			return err
		}
		hostIfaceName = hostVethName

		link, err = netlink.LinkByName(containerVethName)
		if err != nil {
			return fmt.Errorf("failed to find interface %q: %w", containerVethName, err)
		}

		return configureNamespacedLink(link, config, IP)
	})
	if err != nil {
		return fmt.Errorf("failed to create veth pair: %w", err)
	}
	if err := ensureNamespacedHostRoute(IP); err != nil {
		return err
	}
	if hostIfaceName == "" && hostIfaceIndex == 0 {
		return nil
	}

	var hostVeth netlink.Link
	if hostIfaceName != "" {
		hostVeth, err = netlink.LinkByName(hostIfaceName)
		if err != nil {
			return fmt.Errorf("failed to find host interface '%s': %w", hostIfaceName, err)
		}
	} else {
		hostVeth, err = netlink.LinkByIndex(hostIfaceIndex)
		if err != nil {
			return fmt.Errorf("failed to find host interface index '%d': %w", hostIfaceIndex, err)
		}
	}

	if err := netlink.LinkSetUp(hostVeth); err != nil {
		return fmt.Errorf("failed to bring host interface %q UP: %w", hostVeth.Attrs().Name, err)
	}
	if err := netlink.LinkSetMaster(hostVeth, netBridge); err != nil {
		return fmt.Errorf("failed to connect %q to bridge %v: %w", hostVeth.Attrs().Name, netBridge.Attrs().Name, err)
	}
	return nil
}

func setupVeth(containerIfaceName string, mtu int, hostNS ns.NetNS) (string, string, error) {
	var lastErr error
	for i := 0; i < 10; i++ {
		hostIfaceName, err := randomVethName()
		if err != nil {
			return "", "", err
		}
		linkAttrs := netlink.NewLinkAttrs()
		linkAttrs.Name = containerIfaceName
		linkAttrs.MTU = mtu
		veth := &netlink.Veth{
			LinkAttrs:     linkAttrs,
			PeerName:      hostIfaceName,
			PeerNamespace: netlink.NsFd(int(hostNS.Fd())),
		}
		if err := netlink.LinkAdd(veth); err != nil {
			lastErr = err
			if os.IsExist(err) {
				continue
			}
			return "", "", fmt.Errorf("failed to create veth pair %s/%s: %w", containerIfaceName, hostIfaceName, err)
		}

		containerVeth, err := netlink.LinkByName(containerIfaceName)
		if err != nil {
			_ = netlink.LinkDel(veth)
			return "", "", fmt.Errorf("failed to find container veth %q: %w", containerIfaceName, err)
		}
		if err := hostNS.Do(func(_ ns.NetNS) error {
			hostVeth, err := netlink.LinkByName(hostIfaceName)
			if err != nil {
				return fmt.Errorf("failed to find host veth %q: %w", hostIfaceName, err)
			}
			if err := netlink.LinkSetUp(hostVeth); err != nil {
				return fmt.Errorf("failed to bring host veth %q up: %w", hostIfaceName, err)
			}
			_, _ = sysctl.Sysctl(fmt.Sprintf("net/ipv6/conf/%s/accept_ra", hostIfaceName), "0")
			return nil
		}); err != nil {
			_ = netlink.LinkDel(containerVeth)
			return "", "", err
		}
		return hostIfaceName, containerVeth.Attrs().Name, nil
	}
	return "", "", fmt.Errorf("failed to find a unique veth name: %w", lastErr)
}

func randomVethName() (string, error) {
	entropy := make([]byte, 4)
	if _, err := rand.Read(entropy); err != nil {
		return "", fmt.Errorf("failed to generate veth name: %w", err)
	}
	return fmt.Sprintf("veth%x", entropy), nil
}

func configureNamespacedLink(link netlink.Link, config networkmodule.Config, IP net.IP) error {
	if IP == nil || IP.To16() == nil || IP.To4() != nil {
		return fmt.Errorf("invalid IPv6 address for namespaced interface: %v", IP)
	}
	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("failed to bring interface %q UP: %w", link.Attrs().Name, err)
	}

	network := createIPv6Net(config.IPv6Address)
	addr := &netlink.Addr{IPNet: &net.IPNet{Mask: network.Mask, IP: IP}, Label: ""}
	if err := netlink.AddrReplace(link, addr); err != nil {
		return fmt.Errorf("failed to configure IP address '%s' on interface: %w", IP.String(), err)
	}

	gateway, err := namespacedGatewayIP()
	if err != nil {
		return err
	}
	_, networkALL, _ := net.ParseCIDR("::/0")
	route := netlink.Route{
		LinkIndex: link.Attrs().Index,
		Dst:       networkALL,
		Gw:        gateway,
	}
	if err := netlink.RouteReplace(&route); err != nil {
		return fmt.Errorf("failed to configure route on interface: %w", err)
	}
	return nil
}

func ensureNamespacedHostRoute(IP net.IP) error {
	if netBridge == nil {
		return fmt.Errorf("bridge is nil")
	}
	if IP == nil || IP.To16() == nil || IP.To4() != nil {
		return fmt.Errorf("invalid IPv6 address for namespaced interface: %v", IP)
	}
	routeNet := &net.IPNet{IP: IP, Mask: net.CIDRMask(net.IPv6len*8, net.IPv6len*8)}
	route := netlink.Route{
		LinkIndex: netBridge.Attrs().Index,
		Dst:       routeNet,
	}
	if err := netlink.RouteReplace(&route); err != nil {
		return fmt.Errorf("failed to configure host route for namespaced interface '%s': %w", IP.String(), err)
	}
	return nil
}

func namespacedGatewayIP() (net.IP, error) {
	ip := net.ParseIP(namespacedGatewayIPv6)
	if ip == nil {
		return nil, fmt.Errorf("invalid namespaced gateway IP %q", namespacedGatewayIPv6)
	}
	return ip, nil
}

func namespacedGatewayNet() (*net.IPNet, error) {
	ip, network, err := net.ParseCIDR(namespacedGatewayIPv6 + "/64")
	if err != nil {
		return nil, fmt.Errorf("invalid namespaced gateway network: %w", err)
	}
	network.IP = ip
	return network, nil
}
