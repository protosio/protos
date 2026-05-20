//go:build darwin

package wireguard

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"

	networkmodule "github.com/protosio/protos/internal/network/module"
	wgconn "golang.zx2c4.com/wireguard/conn"
	wgdevice "golang.zx2c4.com/wireguard/device"
	wgtun "golang.zx2c4.com/wireguard/tun"
)

const (
	ifconfigPath = "/sbin/ifconfig"
	routePath    = "/sbin/route"
	sysctlPath   = "/usr/sbin/sysctl"

	wireguardPort             = 10999
	domainDNSServer           = "127.0.0.1"
	domainDNSServerPort       = 10053
	peerKeepaliveSeconds      = 25
	defaultWireGuardInterface = "utun"
	ipv6ForwardingSysctl      = "net.inet6.ip6.forwarding"
)

type platformState struct {
	tunDevice              wgtun.Device
	hairpinDevice          *hairpinTun
	wgDevice               *wgdevice.Device
	interfaceName          string
	address                string
	privateKeyHex          string
	listenPort             int
	routes                 map[string]struct{}
	peers                  map[string]struct{}
	previousIPv6Forwarding string
	forwardingConfigured   bool
}

type declarativePeer struct {
	publicKeyHex               string
	allowedIPs                 []string
	endpoint                   string
	persistentKeepaliveSeconds int
}

func (m *Module) closePlatform() error {
	return m.Down()
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
	for route := range m.routes {
		if deleteErr := m.deleteRouteLocked(route); deleteErr != nil {
			err = errors.Join(err, deleteErr)
		}
		delete(m.routes, route)
	}

	if m.domain != "" {
		if deleteErr := m.delDomain(m.domain); deleteErr != nil {
			err = errors.Join(err, deleteErr)
		}
		m.domain = ""
	}

	if restoreErr := m.restoreIPv6ForwardingLocked(); restoreErr != nil {
		err = errors.Join(err, restoreErr)
	}

	if m.address != "" && m.interfaceName != "" {
		if deleteErr := runCommand(ifconfigPath, m.interfaceName, "inet6", m.address, "-alias"); deleteErr != nil {
			log.Debugf("Failed to remove WireGuard address %s from %s: %v", m.address, m.interfaceName, deleteErr)
		}
		m.address = ""
	}

	if m.wgDevice != nil {
		if downErr := m.wgDevice.Down(); downErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to stop WireGuard device: %w", downErr))
		}
		m.wgDevice.Close()
		m.wgDevice = nil
		m.tunDevice = nil
		m.hairpinDevice = nil
	} else if m.tunDevice != nil {
		if closeErr := m.tunDevice.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close utun device: %w", closeErr))
		}
		m.tunDevice = nil
		m.hairpinDevice = nil
	}

	m.interfaceName = ""
	m.privateKeyHex = ""
	m.listenPort = 0
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

	peers, routes, err := m.declarativePeers(peerSet)
	if err != nil {
		return err
	}

	log.Debugf("Applying WireGuard peer set with %d peers and %d routes", len(peers), len(routes))
	logDeclarativePeerState(peers)
	if m.hairpinDevice != nil {
		m.hairpinDevice.setHairpinRoutes(routes)
	}
	if err := m.applyPeerConfigLocked(peers); err != nil {
		return fmt.Errorf("failed to configure embedded WireGuard device: %w", err)
	}

	if err := m.syncRoutesLocked(routes); err != nil {
		return fmt.Errorf("failed to reconcile utun routes: %w", err)
	}
	if m.hairpinDevice != nil {
		stats := m.hairpinDevice.stats()
		log.Debugf("WireGuard hairpin state: routes=%d hairpinned_packets=%d dropped_packets=%d", stats.HairpinRoutes, stats.HairpinnedPackets, stats.DroppedPackets)
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
	hairpinDevice := newHairpinTun(tunDevice)
	wgDevice := wgdevice.NewDevice(hairpinDevice, wgconn.NewDefaultBind(), wgLogger)

	m.tunDevice = hairpinDevice
	m.hairpinDevice = hairpinDevice
	m.wgDevice = wgDevice
	m.interfaceName = interfaceName
	if m.routes == nil {
		m.routes = map[string]struct{}{}
	}

	log.Debugf("Created embedded WireGuard device on %s", interfaceName)
	return nil
}

func (m *Module) applyBaseConfigLocked(config networkmodule.Config) error {
	privateKeyHex, err := privateWireGuardKeyHex(config.WireGuardPrivateKey)
	if err != nil {
		return err
	}

	var b strings.Builder
	if m.privateKeyHex != privateKeyHex {
		fmt.Fprintf(&b, "private_key=%s\n", privateKeyHex)
	}
	if m.listenPort != wireguardPort {
		fmt.Fprintf(&b, "listen_port=%d\n", wireguardPort)
	}
	if b.Len() == 0 {
		return nil
	}
	b.WriteString("\n")

	if err := m.wgDevice.IpcSet(b.String()); err != nil {
		return err
	}

	m.privateKeyHex = privateKeyHex
	m.listenPort = wireguardPort
	return nil
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
		if err := runCommand(ifconfigPath, m.interfaceName, "inet6", m.address, "-alias"); err != nil {
			log.Debugf("Failed to remove previous WireGuard address %s from %s: %v", m.address, m.interfaceName, err)
		}
		m.address = ""
	}

	if err := runCommand(ifconfigPath, m.interfaceName, "inet6", address, "prefixlen", "128", "alias"); err != nil {
		if !isCommandError(err, "File exists", "already exists") {
			return fmt.Errorf("failed to assign IPv6 address %s to %s: %w", address, m.interfaceName, err)
		}
	}
	if err := runCommand(ifconfigPath, m.interfaceName, "up"); err != nil {
		return fmt.Errorf("failed to bring %s up: %w", m.interfaceName, err)
	}

	m.address = address
	if m.hairpinDevice != nil {
		m.hairpinDevice.setLocalAddress(address)
	}
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

func (m *Module) declarativePeers(peerSet networkmodule.Peers) ([]declarativePeer, map[string]struct{}, error) {
	peers := make([]declarativePeer, 0, len(peerSet.Instances)+len(peerSet.Devices))
	routes := map[string]struct{}{}

	for _, instance := range peerSet.Instances {
		if instance.PublicKey == "" || instance.PublicIP == "" || instance.Name == "" {
			log.Warnf("Skipping instance %s: missing public key, public IP, or name", instance.ID)
			continue
		}

		pubKeyAddr, err := ipv6AddressFromPublicKeyBase64(instance.PublicKey)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to configure network (%s): %w", instance.Name, err)
		}

		allowedIP := createIPv6Net(pubKeyAddr)
		if allowedIP == nil {
			return nil, nil, fmt.Errorf("failed to configure network (%s): public key did not map to an IPv6 route", instance.Name)
		}

		publicKeyHex, err := publicWireGuardKeyHex(instance.PublicKey)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to configure network (%s): %w", instance.Name, err)
		}

		instancePublicIP := net.ParseIP(instance.PublicIP)
		if instancePublicIP == nil {
			return nil, nil, fmt.Errorf("failed to parse public IP while configuring VPN interface '%s': bad IP %s", instance.Name, instance.PublicIP)
		}

		route := allowedIP.String()
		routes[route] = struct{}{}
		peers = append(peers, declarativePeer{
			publicKeyHex:               publicKeyHex,
			allowedIPs:                 []string{route},
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
			return nil, nil, fmt.Errorf("failed to decode base64 encoded key for device '%s': %w", device.Name, err)
		}

		allowedIP := createIPv6Net(pubKeyAddr)
		if allowedIP == nil {
			return nil, nil, fmt.Errorf("failed to configure network (%s): public key did not map to an IPv6 route", device.Name)
		}

		publicKeyHex, err := publicWireGuardKeyHex(device.PublicKey)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to decode base64 encoded key for device '%s': %w", device.Name, err)
		}

		route := allowedIP.String()
		routes[route] = struct{}{}
		peers = append(peers, declarativePeer{
			publicKeyHex: publicKeyHex,
			allowedIPs:   []string{route},
		})
	}

	return peers, routes, nil
}

func (m *Module) syncRoutesLocked(desiredRoutes map[string]struct{}) error {
	deleted := 0
	for route := range m.routes {
		if _, ok := desiredRoutes[route]; ok {
			continue
		}
		if err := m.deleteRouteLocked(route); err != nil {
			return err
		}
		delete(m.routes, route)
		deleted++
	}

	added := 0
	for route := range desiredRoutes {
		if _, ok := m.routes[route]; ok {
			continue
		}
		if err := m.addRouteLocked(route); err != nil {
			return err
		}
		m.routes[route] = struct{}{}
		added++
	}
	if added > 0 || deleted > 0 {
		log.Debugf("Reconciled WireGuard routes on %s: added=%d deleted=%d active=%d", m.interfaceName, added, deleted, len(m.routes))
	}

	return nil
}

func (m *Module) addRouteLocked(route string) error {
	err := runCommand(routePath, "-n", "add", "-inet6", route, "-interface", m.interfaceName)
	if err != nil && !isCommandError(err, "File exists", "already in table") {
		return fmt.Errorf("failed to add route %s through %s: %w", route, m.interfaceName, err)
	}
	log.Debugf("Added WireGuard route %s through %s", route, m.interfaceName)
	return nil
}

func (m *Module) deleteRouteLocked(route string) error {
	err := runCommand(routePath, "-n", "delete", "-inet6", route, "-interface", m.interfaceName)
	if err != nil && !isCommandError(err, "not in table", "not found", "No such process") {
		return fmt.Errorf("failed to delete route %s through %s: %w", route, m.interfaceName, err)
	}
	log.Debugf("Deleted WireGuard route %s through %s", route, m.interfaceName)
	return nil
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

func runCommand(name string, args ...string) error {
	output, err := exec.Command(name, args...).CombinedOutput()
	if err == nil {
		return nil
	}

	outputText := strings.TrimSpace(string(output))
	if outputText == "" {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, outputText)
}

func sysctlValue(name string) (string, error) {
	output, err := exec.Command(sysctlPath, "-n", name).CombinedOutput()
	if err != nil {
		outputText := strings.TrimSpace(string(output))
		if outputText == "" {
			return "", fmt.Errorf("%s -n %s: %w", sysctlPath, name, err)
		}
		return "", fmt.Errorf("%s -n %s: %w: %s", sysctlPath, name, err, outputText)
	}
	return strings.TrimSpace(string(output)), nil
}

func setSysctl(name string, value string) error {
	return runCommand(sysctlPath, "-w", fmt.Sprintf("%s=%s", name, value))
}

func isCommandError(err error, needles ...string) bool {
	if err == nil {
		return false
	}
	errText := err.Error()
	for _, needle := range needles {
		if strings.Contains(errText, needle) {
			return true
		}
	}
	return false
}
