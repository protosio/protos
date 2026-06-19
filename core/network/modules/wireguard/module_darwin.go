//go:build darwin

package wireguard

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"sort"
	"strconv"
	"strings"

	networkmodule "github.com/protosio/protos/internal/network/module"
	"golang.org/x/sys/unix"
	wgconn "golang.zx2c4.com/wireguard/conn"
	wgdevice "golang.zx2c4.com/wireguard/device"
	wgtun "golang.zx2c4.com/wireguard/tun"
)

const (
	wireguardPort             = 10999
	wireguardListenPortEnv    = "PROTOS_WIREGUARD_LISTEN_PORT"
	domainDNSServer           = "127.0.0.1"
	domainDNSServerPort       = 10053
	peerKeepaliveSeconds      = 25
	defaultWireGuardInterface = "utun"
	ipv6ForwardingSysctl      = "net.inet6.ip6.forwarding"
)

type platformState struct {
	tunDevice              wgtun.Device
	wgDevice               *wgdevice.Device
	interfaceName          string
	address                string
	ipv4Address            string
	privateKeyHex          string
	listenPort             int
	routes                 map[string]routeSpec
	peers                  map[string]struct{}
	previousIPv6Forwarding string
	forwardingConfigured   bool
	exitDNSConfigured      bool
}

type declarativePeer struct {
	publicKeyHex               string
	allowedIPs                 []string
	endpoint                   string
	persistentKeepaliveSeconds int
}

type routeSpec struct {
	family      string
	destination string
	gateway     string
	iface       string
	priority    int
}

func (r routeSpec) key() string {
	return r.family + "|" + r.destination
}

func (m *Module) closePlatform() error {
	err := m.Down()
	if m.dnsManager != nil {
		err = errors.Join(err, m.dnsManager.Close())
	}
	return err
}

func (m *Module) Up(config networkmodule.Config) error {
	if err := validateConfig(config); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return m.upLocked(config)
}

func (m *Module) Down() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var err error
	for key, route := range m.routes {
		if deleteErr := m.deleteRouteLocked(route); deleteErr != nil {
			err = errors.Join(err, deleteErr)
		}
		delete(m.routes, key)
	}

	if m.domain != "" {
		if deleteErr := m.delDomain(m.domain); deleteErr != nil {
			err = errors.Join(err, deleteErr)
		}
		m.domain = ""
	}
	if deleteErr := m.syncExitDNSLocked(false); deleteErr != nil {
		err = errors.Join(err, deleteErr)
	}

	if restoreErr := m.restoreIPv6ForwardingLocked(); restoreErr != nil {
		err = errors.Join(err, restoreErr)
	}

	if m.address != "" && m.interfaceName != "" {
		if deleteErr := deleteInterfaceIPv6Address(m.interfaceName, m.address); deleteErr != nil {
			log.Debugf("Failed to remove WireGuard address %s from %s: %v", m.address, m.interfaceName, deleteErr)
		}
		m.address = ""
	}
	if m.ipv4Address != "" && m.interfaceName != "" {
		if deleteErr := deleteInterfaceIPv4Address(m.interfaceName, m.ipv4Address); deleteErr != nil {
			log.Debugf("Failed to remove WireGuard IPv4 address %s from %s: %v", m.ipv4Address, m.interfaceName, deleteErr)
		}
		m.ipv4Address = ""
	}

	if m.wgDevice != nil {
		if downErr := m.wgDevice.Down(); downErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to stop WireGuard device: %w", downErr))
		}
		m.wgDevice.Close()
		m.wgDevice = nil
		m.tunDevice = nil
	} else if m.tunDevice != nil {
		if closeErr := m.tunDevice.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close utun device: %w", closeErr))
		}
		m.tunDevice = nil
	}

	m.interfaceName = ""
	m.privateKeyHex = ""
	m.listenPort = 0
	m.routes = map[string]routeSpec{}
	m.peers = map[string]struct{}{}

	return err
}

func (m *Module) ConfigurePeers(config networkmodule.Config, peerSet networkmodule.Peers) error {
	if err := validateConfig(config); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.upLocked(config); err != nil {
		return err
	}

	peers, routes, localExitNeedsIPv4, localExitDNSActive, err := m.declarativePeers(config, peerSet)
	if err != nil {
		return err
	}
	if err := m.configureIPv4AddressLocked(config.IPv4Address, localExitNeedsIPv4); err != nil {
		return err
	}

	log.Debugf("Applying WireGuard peer set with %d peers and %d routes", len(peers), len(routes))
	logDeclarativePeerState(peers)
	if err := m.applyPeerConfigLocked(peers); err != nil {
		return fmt.Errorf("failed to configure embedded WireGuard device: %w", err)
	}

	if err := m.syncRoutesLocked(routes); err != nil {
		return fmt.Errorf("failed to reconcile utun routes: %w", err)
	}
	if err := m.syncExitDNSLocked(localExitDNSActive); err != nil {
		return fmt.Errorf("failed to reconcile exit DNS: %w", err)
	}

	return nil
}

func (m *Module) upLocked(config networkmodule.Config) error {
	if err := m.ensureDeviceLocked(); err != nil {
		return err
	}

	if err := m.applyBaseConfigLocked(config); err != nil {
		return fmt.Errorf("failed to configure embedded WireGuard device: %w", err)
	}

	if err := m.configureAddressLocked(config.IPv6Address.String()); err != nil {
		return err
	}

	if err := m.enableIPv6ForwardingLocked(); err != nil {
		return err
	}

	if err := m.syncDomainLocked(config.Domain); err != nil {
		return err
	}

	if err := m.wgDevice.Up(); err != nil {
		return fmt.Errorf("failed to start embedded WireGuard device: %w", err)
	}

	return nil
}

func (m *Module) ensureDeviceLocked() error {
	if m.wgDevice != nil {
		return nil
	}

	tunDevice, err := wgtun.CreateTUN(defaultWireGuardInterface, wgdevice.DefaultMTU)
	if err != nil {
		return fmt.Errorf("failed to create utun device: %w", err)
	}

	interfaceName, err := tunDevice.Name()
	if err != nil {
		_ = tunDevice.Close()
		return fmt.Errorf("failed to resolve utun device name: %w", err)
	}

	wgLogger := &wgdevice.Logger{
		Verbosef: log.Debugf,
		Errorf:   log.Errorf,
	}
	wgDevice := wgdevice.NewDevice(tunDevice, wgconn.NewDefaultBind(), wgLogger)

	m.tunDevice = tunDevice
	m.wgDevice = wgDevice
	m.interfaceName = interfaceName
	if m.routes == nil {
		m.routes = map[string]routeSpec{}
	}

	log.Debugf("Created embedded WireGuard device on %s", interfaceName)
	return nil
}

func (m *Module) applyBaseConfigLocked(config networkmodule.Config) error {
	privateKeyHex, err := privateWireGuardKeyHex(config.WireGuardPrivateKey)
	if err != nil {
		return err
	}
	listenPort := wireGuardListenPort()

	var b strings.Builder
	if m.privateKeyHex != privateKeyHex {
		fmt.Fprintf(&b, "private_key=%s\n", privateKeyHex)
	}
	if m.listenPort != listenPort {
		fmt.Fprintf(&b, "listen_port=%d\n", listenPort)
	}
	if b.Len() == 0 {
		return nil
	}
	b.WriteString("\n")

	if err := m.wgDevice.IpcSet(b.String()); err != nil {
		return err
	}

	m.privateKeyHex = privateKeyHex
	m.listenPort = listenPort
	return nil
}

func wireGuardListenPort() int {
	value := strings.TrimSpace(os.Getenv(wireguardListenPortEnv))
	if value == "" {
		return wireguardPort
	}
	port, err := strconv.Atoi(value)
	if err != nil || port < 0 || port > 65535 {
		log.Warnf("Ignoring invalid %s=%q; using %d", wireguardListenPortEnv, value, wireguardPort)
		return wireguardPort
	}
	return port
}

func (m *Module) applyPeerConfigLocked(peers []declarativePeer) error {
	var b strings.Builder
	desired := make(map[string]struct{}, len(peers))
	for _, peer := range peers {
		desired[peer.publicKeyHex] = struct{}{}
	}
	for publicKeyHex := range m.peers {
		if _, found := desired[publicKeyHex]; found {
			continue
		}
		fmt.Fprintf(&b, "public_key=%s\n", publicKeyHex)
		b.WriteString("remove=true\n")
	}
	for _, peer := range peers {
		fmt.Fprintf(&b, "public_key=%s\n", peer.publicKeyHex)
		b.WriteString("replace_allowed_ips=true\n")
		for _, allowedIP := range peer.allowedIPs {
			fmt.Fprintf(&b, "allowed_ip=%s\n", allowedIP)
		}
		if peer.endpoint != "" {
			fmt.Fprintf(&b, "endpoint=%s\n", peer.endpoint)
		}
		if peer.persistentKeepaliveSeconds > 0 {
			fmt.Fprintf(&b, "persistent_keepalive_interval=%d\n", peer.persistentKeepaliveSeconds)
		}
		b.WriteString("protocol_version=1\n")
	}
	b.WriteString("\n")

	if err := m.wgDevice.IpcSet(b.String()); err != nil {
		return err
	}
	m.peers = desired
	return nil
}

func (m *Module) configureAddressLocked(address string) error {
	if m.address == address {
		return nil
	}

	if m.address != "" {
		if err := deleteInterfaceIPv6Address(m.interfaceName, m.address); err != nil {
			log.Debugf("Failed to remove previous WireGuard address %s from %s: %v", m.address, m.interfaceName, err)
		}
		m.address = ""
	}

	if err := addInterfaceIPv6Address(m.interfaceName, address); err != nil {
		if !errors.Is(err, unix.EEXIST) {
			return fmt.Errorf("failed to assign IPv6 address %s to %s: %w", address, m.interfaceName, err)
		}
	}
	if err := setInterfaceUp(m.interfaceName); err != nil {
		return fmt.Errorf("failed to bring %s up: %w", m.interfaceName, err)
	}

	m.address = address
	return nil
}

func (m *Module) configureIPv4AddressLocked(address netip.Addr, enabled bool) error {
	if !enabled {
		if m.ipv4Address != "" {
			if err := deleteInterfaceIPv4Address(m.interfaceName, m.ipv4Address); err != nil {
				log.Debugf("Failed to remove WireGuard IPv4 address %s from %s: %v", m.ipv4Address, m.interfaceName, err)
			}
			m.ipv4Address = ""
		}
		return nil
	}
	if !address.IsValid() || !address.Is4() {
		return fmt.Errorf("network IPv4 address is required for exit routing")
	}

	addr := address.String()
	if m.ipv4Address == addr {
		return nil
	}
	if m.ipv4Address != "" {
		if err := deleteInterfaceIPv4Address(m.interfaceName, m.ipv4Address); err != nil {
			log.Debugf("Failed to remove previous WireGuard IPv4 address %s from %s: %v", m.ipv4Address, m.interfaceName, err)
		}
		m.ipv4Address = ""
	}
	if err := addInterfaceIPv4Address(m.interfaceName, address); err != nil {
		if !errors.Is(err, unix.EEXIST) {
			return fmt.Errorf("failed to assign IPv4 address %s to %s: %w", addr, m.interfaceName, err)
		}
	}
	m.ipv4Address = addr
	return nil
}

func (m *Module) enableIPv6ForwardingLocked() error {
	if m.forwardingConfigured {
		return nil
	}

	previous, err := sysctlValue(ipv6ForwardingSysctl)
	if err != nil {
		return fmt.Errorf("failed to read IPv6 forwarding state: %w", err)
	}

	if previous != "1" {
		if err := setSysctl(ipv6ForwardingSysctl, "1"); err != nil {
			return fmt.Errorf("failed to enable IPv6 forwarding: %w", err)
		}
		log.Debug("Enabled IPv6 forwarding for WireGuard peer routing")
	}

	m.previousIPv6Forwarding = previous
	m.forwardingConfigured = true
	return nil
}

func (m *Module) restoreIPv6ForwardingLocked() error {
	if !m.forwardingConfigured {
		return nil
	}
	previous := m.previousIPv6Forwarding
	if previous == "" {
		m.forwardingConfigured = false
		return nil
	}
	if err := setSysctl(ipv6ForwardingSysctl, previous); err != nil {
		return fmt.Errorf("failed to restore IPv6 forwarding state: %w", err)
	}
	m.previousIPv6Forwarding = ""
	m.forwardingConfigured = false
	return nil
}

func (m *Module) syncDomainLocked(domain string) error {
	if domain == m.domain {
		return nil
	}

	if m.domain != "" {
		if err := m.delDomain(m.domain); err != nil {
			return err
		}
		m.domain = ""
	}

	if domain == "" {
		return nil
	}

	if err := m.addDomain(domain, domainDNSServer); err != nil {
		return err
	}
	m.domain = domain
	return nil
}

func (m *Module) syncExitDNSLocked(enabled bool) error {
	if enabled {
		dnsServerIP := net.ParseIP(domainDNSServer)
		if dnsServerIP == nil {
			return fmt.Errorf("failed to parse DNS server IP '%s'", domainDNSServer)
		}
		if err := m.dnsManager.SetGlobalServer(dnsServerIP, domainDNSServerPort); err != nil {
			return err
		}
		if m.exitDNSConfigured {
			return nil
		}
		m.exitDNSConfigured = true
		log.Debugf("Configured macOS global DNS resolver through Protos at %s:%d", domainDNSServer, domainDNSServerPort)
		return nil
	}
	if err := m.dnsManager.DelGlobalServer(); err != nil {
		return err
	}
	m.exitDNSConfigured = false
	log.Debug("Restored macOS global DNS resolver after Protos exit route")
	return nil
}

func (m *Module) declarativePeers(config networkmodule.Config, peerSet networkmodule.Peers) ([]declarativePeer, map[string]routeSpec, bool, bool, error) {
	peers := make([]declarativePeer, 0, len(peerSet.Instances)+len(peerSet.Devices))
	routes := map[string]routeSpec{}
	exitRoute, localExitActive := localExitRoute(config, peerSet)
	localExitNeedsIPv4 := false
	localExitDNSActive := false

	for _, instance := range peerSet.Instances {
		if instance.PublicKey == "" || instance.PublicIP == "" || instance.Name == "" {
			log.Warnf("Skipping instance %s: missing public key, public IP, or name", instance.ID)
			continue
		}

		pubKeyAddr, err := ipv6AddressFromPublicKeyBase64(instance.PublicKey)
		if err != nil {
			return nil, nil, false, false, fmt.Errorf("failed to configure network (%s): %w", instance.Name, err)
		}
		if pubKeyAddr == config.IPv6Address {
			log.Debugf("Skipping local instance peer %s (%s)", instance.Name, pubKeyAddr.String())
			continue
		}

		allowedIP := createIPv6Net(pubKeyAddr)
		if allowedIP == nil {
			return nil, nil, false, false, fmt.Errorf("failed to configure network (%s): public key did not map to an IPv6 route", instance.Name)
		}

		publicKeyHex, err := publicWireGuardKeyHex(instance.PublicKey)
		if err != nil {
			return nil, nil, false, false, fmt.Errorf("failed to configure network (%s): %w", instance.Name, err)
		}

		instancePublicIP := net.ParseIP(instance.PublicIP)
		if instancePublicIP == nil {
			return nil, nil, false, false, fmt.Errorf("failed to parse public IP while configuring VPN interface '%s': bad IP %s", instance.Name, instance.PublicIP)
		}

		route := allowedIP.String()
		allowedIPs := []string{route}
		addInterfaceRoute(routes, inet6Route, route, m.interfaceName, 20)
		for _, routeAddr := range instance.Routes {
			if routeAddr == pubKeyAddr || routeAddr == config.IPv6Address {
				continue
			}
			appRoute := createIPv6Net(routeAddr)
			if appRoute == nil {
				continue
			}
			route := appRoute.String()
			addInterfaceRoute(routes, inet6Route, route, m.interfaceName, 20)
			allowedIPs = append(allowedIPs, route)
		}
		if localExitActive && exitRoute.InstanceID == instance.ID {
			exitCIDRs, err := normalizedExitRouteCIDRs(exitRoute)
			if err != nil {
				return nil, nil, false, false, fmt.Errorf("failed to configure exit route through %s: %w", instance.Name, err)
			}
			allowedIPs = append(allowedIPs, exitCIDRs...)
			if err := addExitCIDRRoutes(routes, exitCIDRs, instancePublicIP, m.interfaceName); err != nil {
				return nil, nil, false, false, fmt.Errorf("failed to configure exit route through %s: %w", instance.Name, err)
			}
			localExitNeedsIPv4 = exitRouteCIDRsNeedIPv4(exitCIDRs)
			localExitDNSActive = exitRouteCIDRsUseFullTunnel(exitCIDRs)
			log.Debugf("Routing %s through exit instance %s (%s)", strings.Join(exitCIDRs, ","), instance.Name, instance.PublicIP)
		}
		peers = append(peers, declarativePeer{
			publicKeyHex:               publicKeyHex,
			allowedIPs:                 allowedIPs,
			endpoint:                   net.JoinHostPort(instancePublicIP.String(), strconv.Itoa(wireguardPort)),
			persistentKeepaliveSeconds: peerKeepaliveSeconds,
		})
	}

	for _, device := range peerSet.Devices {
		if device.PublicKey == "" {
			log.Warnf("Skipping device %s: missing public key", device.Name)
			continue
		}

		pubKeyAddr, err := ipv6AddressFromPublicKeyBase64(device.PublicKey)
		if err != nil {
			return nil, nil, false, false, fmt.Errorf("failed to decode base64 encoded key for device '%s': %w", device.Name, err)
		}

		allowedIP := createIPv6Net(pubKeyAddr)
		if allowedIP == nil {
			return nil, nil, false, false, fmt.Errorf("failed to configure network (%s): public key did not map to an IPv6 route", device.Name)
		}

		publicKeyHex, err := publicWireGuardKeyHex(device.PublicKey)
		if err != nil {
			return nil, nil, false, false, fmt.Errorf("failed to decode base64 encoded key for device '%s': %w", device.Name, err)
		}

		route := allowedIP.String()
		addInterfaceRoute(routes, inet6Route, route, m.interfaceName, 20)
		peers = append(peers, declarativePeer{
			publicKeyHex: publicKeyHex,
			allowedIPs:   []string{route},
		})
	}

	return peers, routes, localExitNeedsIPv4, localExitDNSActive, nil
}

const (
	inetRoute  = "-inet"
	inet6Route = "-inet6"
)

func addInterfaceRoute(routes map[string]routeSpec, family string, destination string, iface string, priority int) {
	route := routeSpec{family: family, destination: destination, iface: iface, priority: priority}
	routes[route.key()] = route
}

func addGatewayRoute(routes map[string]routeSpec, family string, destination string, gateway string, iface string, priority int) {
	route := routeSpec{family: family, destination: destination, gateway: gateway, iface: iface, priority: priority}
	routes[route.key()] = route
}

func addExitCIDRRoutes(routes map[string]routeSpec, cidrs []string, endpoint net.IP, iface string) error {
	for _, cidr := range cidrs {
		prefix, err := netip.ParsePrefix(cidr)
		if err != nil {
			return err
		}
		prefix = prefix.Masked()
		family := inetRoute
		if prefix.Addr().Is6() {
			family = inet6Route
		}

		routePrefixes := []string{prefix.String()}
		if prefix.Bits() == 0 {
			if prefix.Addr().Is4() {
				routePrefixes = ipv4DefaultRoutePrefixes
			} else {
				routePrefixes = ipv6DefaultRoutePrefixes
			}
		}
		for _, route := range routePrefixes {
			addInterfaceRoute(routes, family, route, iface, 100)
		}
	}

	if !exitRouteCIDRsContainEndpoint(cidrs, endpoint) {
		return nil
	}
	family := inetRoute
	destination := endpoint.String() + "/32"
	if endpoint.To4() == nil {
		family = inet6Route
		destination = endpoint.String() + "/128"
	}
	target, err := defaultRouteTarget(family)
	if err != nil {
		return err
	}
	addGatewayRoute(routes, family, destination, target.gateway, target.iface, 0)
	return nil
}

func (m *Module) syncRoutesLocked(desiredRoutes map[string]routeSpec) error {
	deleted := 0
	for _, route := range sortedRouteSpecs(m.routes, true) {
		key := route.key()
		if desired, ok := desiredRoutes[key]; ok && desired == route {
			continue
		}
		if err := m.deleteRouteLocked(route); err != nil {
			return err
		}
		delete(m.routes, key)
		deleted++
	}

	added := 0
	for _, route := range sortedRouteSpecs(desiredRoutes, false) {
		key := route.key()
		if _, ok := m.routes[key]; ok {
			continue
		}
		if err := m.addRouteLocked(route); err != nil {
			return err
		}
		m.routes[key] = route
		added++
	}
	if added > 0 || deleted > 0 {
		log.Debugf("Reconciled WireGuard routes on %s: added=%d deleted=%d active=%d", m.interfaceName, added, deleted, len(m.routes))
	}

	return nil
}

func sortedRouteSpecs(routes map[string]routeSpec, reverse bool) []routeSpec {
	values := make([]routeSpec, 0, len(routes))
	for _, route := range routes {
		values = append(values, route)
	}
	sort.Slice(values, func(i int, j int) bool {
		if values[i].priority != values[j].priority {
			if reverse {
				return values[i].priority > values[j].priority
			}
			return values[i].priority < values[j].priority
		}
		return values[i].key() < values[j].key()
	})
	return values
}

func (m *Module) addRouteLocked(route routeSpec) error {
	err := addDarwinRoute(route)
	if err != nil && !errors.Is(err, unix.EEXIST) {
		return fmt.Errorf("failed to add route %s: %w", route.destination, err)
	}
	log.Debugf("Added WireGuard route %s", routeString(route))
	return nil
}

func (m *Module) deleteRouteLocked(route routeSpec) error {
	err := deleteDarwinRoute(route)
	if err != nil && !errors.Is(err, unix.ESRCH) {
		return fmt.Errorf("failed to delete route %s: %w", route.destination, err)
	}
	log.Debugf("Deleted WireGuard route %s", routeString(route))
	return nil
}

func routeString(route routeSpec) string {
	if route.gateway != "" {
		return fmt.Sprintf("%s via %s", route.destination, route.gateway)
	}
	if route.iface != "" {
		return fmt.Sprintf("%s dev %s", route.destination, route.iface)
	}
	return route.destination
}

type routeTarget struct {
	gateway string
	iface   string
}

func logDeclarativePeerState(peers []declarativePeer) {
	for _, peer := range peers {
		endpoint := peer.endpoint
		if endpoint == "" {
			endpoint = "<none>"
		}
		log.Debugf(
			"WireGuard peer desired: public_key=%s endpoint=%s allowed_ips=%s keepalive_seconds=%d",
			shortWireGuardKeyHex(peer.publicKeyHex),
			endpoint,
			strings.Join(peer.allowedIPs, ","),
			peer.persistentKeepaliveSeconds,
		)
	}
}

func shortWireGuardKeyHex(key string) string {
	if len(key) <= 12 {
		return key
	}
	return key[:12]
}

func privateWireGuardKeyHex(privateKey string) (string, error) {
	key, err := privateWireGuardKeyBytes(privateKey)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(key[:]), nil
}

func publicWireGuardKeyHex(publicKey string) (string, error) {
	key, err := publicEd25519ToWireGuardBytes(publicKey)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(key[:]), nil
}

func (m *Module) addDomain(domain string, dnsServer string) error {
	dnsServerIP := net.ParseIP(dnsServer)
	if dnsServerIP == nil {
		return fmt.Errorf("failed to parse DNS server IP '%s'", dnsServer)
	}

	if err := m.dnsManager.AddDomainServer(domain, dnsServerIP, domainDNSServerPort); err != nil {
		return fmt.Errorf("failed to add domain: %w", err)
	}
	return nil
}

func (m *Module) delDomain(domain string) error {
	if err := m.dnsManager.DelDomainServer(domain); err != nil {
		return fmt.Errorf("failed to delete DNS server for domain '%s': %w", domain, err)
	}
	return nil
}

func sysctlValue(name string) (string, error) {
	value, err := sysctlUint32(name)
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(uint64(value), 10), nil
}

func setSysctl(name string, value string) error {
	parsed, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid sysctl value %q for %s: %w", value, name, err)
	}
	return setSysctlUint32(name, uint32(parsed))
}
