//go:build darwin

package wireguard

import (
	"encoding/hex"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	networkmodule "github.com/protosio/protos/internal/network/module"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func (m *Module) State() (networkmodule.State, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state := networkmodule.State{
		Module:        m.Name(),
		Up:            m.wgDevice != nil,
		InterfaceName: m.interfaceName,
	}

	if m.interfaceName != "" {
		state.Interfaces = append(state.Interfaces, darwinInterfaceState(m.interfaceName, m.wgDevice != nil))
		state.Addresses = append(state.Addresses, darwinInterfaceAddresses(m.interfaceName)...)
	}
	if m.address != "" && !hasAddressState(state.Addresses, m.interfaceName, m.address) {
		state.Addresses = append(state.Addresses, networkmodule.AddressState{
			InterfaceName: m.interfaceName,
			CIDR:          m.address + "/128",
			Scope:         "global",
		})
	}
	if m.ipv4Address != "" && !hasAddressState(state.Addresses, m.interfaceName, m.ipv4Address) {
		state.Addresses = append(state.Addresses, networkmodule.AddressState{
			InterfaceName: m.interfaceName,
			CIDR:          m.ipv4Address + "/32",
			Scope:         "global",
		})
	}

	for _, route := range sortedRouteSpecs(m.routes, false) {
		state.Routes = append(state.Routes, networkmodule.RouteState{
			InterfaceName: route.iface,
			Destination:   route.destination,
			Gateway:       route.gateway,
			Family:        routeFamilyName(route.family),
			Priority:      strconv.Itoa(route.priority),
			Kind:          "protos-managed",
		})
	}

	if m.wgDevice != nil {
		ipcState, err := m.wgDevice.IpcGet()
		if err != nil {
			state.Messages = append(state.Messages, fmt.Sprintf("failed to read WireGuard peer state: %v", err))
		} else {
			state.WireGuardPeers = parseWireGuardIPCState(ipcState)
		}
	}

	if m.domain != "" {
		state.DNS = append(state.DNS, networkmodule.DNSState{
			Scope:   "domain",
			Domain:  m.domain,
			Servers: []string{domainDNSServer},
			Port:    domainDNSServerPort,
			Active:  true,
			Source:  resolverPath,
		})
	}
	if m.exitDNSConfigured || (m.dnsManager != nil && m.dnsManager.HasGlobalServerBackup()) {
		state.DNS = append(state.DNS, networkmodule.DNSState{
			Scope:   "global",
			Servers: []string{domainDNSServer},
			Port:    domainDNSServerPort,
			Active:  m.exitDNSConfigured,
			Source:  globalDNSKey,
		})
	}

	return state, nil
}

func darwinInterfaceState(interfaceName string, active bool) networkmodule.InterfaceState {
	state := networkmodule.InterfaceState{
		Name: interfaceName,
		Type: "wireguard-userspace",
		Up:   active,
		Kind: "protos-managed",
	}
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return state
	}
	state.Index = iface.Index
	state.MTU = iface.MTU
	state.Up = iface.Flags&net.FlagUp != 0
	if iface.HardwareAddr != nil {
		state.MacAddress = iface.HardwareAddr.String()
	}
	return state
}

func darwinInterfaceAddresses(interfaceName string) []networkmodule.AddressState {
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return []networkmodule.AddressState{{
			InterfaceName: interfaceName,
			Scope:         fmt.Sprintf("error: %v", err),
		}}
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return []networkmodule.AddressState{{
			InterfaceName: interfaceName,
			Scope:         fmt.Sprintf("error: %v", err),
		}}
	}
	out := make([]networkmodule.AddressState, 0, len(addrs))
	for _, addr := range addrs {
		scope := "global"
		if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.IsLinkLocalUnicast() {
			scope = "link"
		}
		out = append(out, networkmodule.AddressState{
			InterfaceName: interfaceName,
			CIDR:          addr.String(),
			Scope:         scope,
		})
	}
	return out
}

func hasAddressState(values []networkmodule.AddressState, iface string, address string) bool {
	for _, value := range values {
		if value.InterfaceName == iface && strings.HasPrefix(value.CIDR, address+"/") {
			return true
		}
	}
	return false
}

func routeFamilyName(family string) string {
	switch family {
	case inetRoute:
		return "ipv4"
	case inet6Route:
		return "ipv6"
	default:
		return family
	}
}

func parseWireGuardIPCState(raw string) []networkmodule.WireGuardPeerState {
	var peers []networkmodule.WireGuardPeerState
	var current *networkmodule.WireGuardPeerState
	var handshakeSec int64
	var handshakeNsec int64

	flush := func() {
		if current == nil {
			return
		}
		if handshakeSec > 0 || handshakeNsec > 0 {
			current.LatestHandshake = time.Unix(handshakeSec, handshakeNsec).Format(time.RFC3339)
		}
		peers = append(peers, *current)
		current = nil
		handshakeSec = 0
		handshakeNsec = 0
	}

	for _, line := range strings.Split(raw, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch key {
		case "public_key":
			flush()
			current = &networkmodule.WireGuardPeerState{PublicKey: wireGuardKeyString(value)}
		case "endpoint":
			if current != nil {
				current.Endpoint = value
			}
		case "allowed_ip":
			if current != nil {
				current.AllowedIPs = append(current.AllowedIPs, value)
			}
		case "last_handshake_time_sec":
			handshakeSec, _ = strconv.ParseInt(value, 10, 64)
		case "last_handshake_time_nsec":
			handshakeNsec, _ = strconv.ParseInt(value, 10, 64)
		case "rx_bytes":
			if current != nil {
				current.RxBytes, _ = strconv.ParseUint(value, 10, 64)
			}
		case "tx_bytes":
			if current != nil {
				current.TxBytes, _ = strconv.ParseUint(value, 10, 64)
			}
		}
	}
	flush()
	for i := range peers {
		sort.Strings(peers[i].AllowedIPs)
	}
	return peers
}

func wireGuardKeyString(hexValue string) string {
	keyBytes, err := hex.DecodeString(hexValue)
	if err != nil || len(keyBytes) != 32 {
		return hexValue
	}
	var key wgtypes.Key
	copy(key[:], keyBytes)
	return key.String()
}
