//go:build linux

package wireguard

import (
	"encoding/hex"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	networkmodule "github.com/protosio/protos/internal/network/module"
	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func (m *Module) State() (networkmodule.State, error) {
	state := networkmodule.State{
		Module:        m.Name(),
		InterfaceName: wireguardNetworkInterfaceName,
	}

	wgLink, err := netlink.LinkByName(wireguardNetworkInterfaceName)
	if err != nil {
		state.Messages = append(state.Messages, fmt.Sprintf("wireguard interface %s is not present: %v", wireguardNetworkInterfaceName, err))
	} else {
		state.Up = wgLink.Attrs().Flags&net.FlagUp != 0
		state.Interfaces = append(state.Interfaces, linuxInterfaceState(wgLink, "protos-managed"))
		state.Addresses = append(state.Addresses, linuxAddressState(wgLink)...)
	}
	if brLink, err := netlink.LinkByName(bridgeNetworkInterface); err == nil {
		state.Interfaces = append(state.Interfaces, linuxInterfaceState(brLink, "protos-managed"))
		state.Interfaces = append(state.Interfaces, linuxBridgeSlaveStates(brLink)...)
		state.Addresses = append(state.Addresses, linuxAddressState(brLink)...)
	}

	routes, err := netlink.RouteList(nil, netlink.FAMILY_ALL)
	if err != nil {
		state.Messages = append(state.Messages, fmt.Sprintf("failed to list kernel routes: %v", err))
	} else {
		state.Routes = linuxRouteState(routes)
	}

	if m.linkManager != nil {
		link, err := m.linkManager.GetLink(wireguardNetworkInterfaceName)
		if err != nil {
			state.Messages = append(state.Messages, fmt.Sprintf("failed to read WireGuard link: %v", err))
		} else if device, err := link.WGConfig(); err != nil {
			state.Messages = append(state.Messages, fmt.Sprintf("failed to read WireGuard peer state: %v", err))
		} else {
			state.WireGuardPeers = wireGuardPeerState(device.Peers)
		}
	}

	tables, err := inspectExitGatewayTables()
	if err != nil {
		state.Messages = append(state.Messages, fmt.Sprintf("failed to inspect exit gateway nftables state: %v", err))
	} else {
		state.FirewallTables = tables
	}

	return state, nil
}

func linuxInterfaceState(link netlink.Link, kind string) networkmodule.InterfaceState {
	attrs := link.Attrs()
	state := networkmodule.InterfaceState{
		Name:  attrs.Name,
		Type:  link.Type(),
		Index: attrs.Index,
		MTU:   attrs.MTU,
		Up:    attrs.Flags&net.FlagUp != 0,
		Kind:  kind,
	}
	if attrs.HardwareAddr != nil {
		state.MacAddress = attrs.HardwareAddr.String()
	}
	if attrs.MasterIndex > 0 {
		if master, err := netlink.LinkByIndex(attrs.MasterIndex); err == nil {
			state.Master = master.Attrs().Name
		}
	}
	return state
}

func linuxBridgeSlaveStates(bridge netlink.Link) []networkmodule.InterfaceState {
	bridgeIndex := bridge.Attrs().Index
	links, err := netlink.LinkList()
	if err != nil {
		return nil
	}
	out := []networkmodule.InterfaceState{}
	for _, link := range links {
		if link.Attrs().MasterIndex != bridgeIndex {
			continue
		}
		out = append(out, linuxInterfaceState(link, "protos-managed-app"))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func linuxAddressState(link netlink.Link) []networkmodule.AddressState {
	addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return []networkmodule.AddressState{{
			InterfaceName: link.Attrs().Name,
			Scope:         fmt.Sprintf("error: %v", err),
		}}
	}
	out := make([]networkmodule.AddressState, 0, len(addrs))
	for _, addr := range addrs {
		scope := "global"
		if addr.Scope != int(netlink.SCOPE_UNIVERSE) {
			scope = strconv.Itoa(addr.Scope)
		}
		out = append(out, networkmodule.AddressState{
			InterfaceName: link.Attrs().Name,
			CIDR:          addr.IPNet.String(),
			Scope:         scope,
		})
	}
	return out
}

func linuxRouteState(routes []netlink.Route) []networkmodule.RouteState {
	out := make([]networkmodule.RouteState, 0, len(routes))
	for _, route := range routes {
		iface := ""
		if route.LinkIndex > 0 {
			if link, err := netlink.LinkByIndex(route.LinkIndex); err == nil {
				iface = link.Attrs().Name
			}
		}
		family := "unknown"
		if route.Family == netlink.FAMILY_V4 {
			family = "ipv4"
		} else if route.Family == netlink.FAMILY_V6 {
			family = "ipv6"
		}
		kind := "kernel"
		if iface == wireguardNetworkInterfaceName || iface == bridgeNetworkInterface {
			kind = "protos-managed"
		}
		out = append(out, networkmodule.RouteState{
			InterfaceName: iface,
			Destination:   routeDestinationString(route),
			Gateway:       ipString(route.Gw),
			Source:        ipString(route.Src),
			Family:        family,
			Table:         strconv.Itoa(route.Table),
			Protocol:      strconv.Itoa(int(route.Protocol)),
			Scope:         strconv.Itoa(int(route.Scope)),
			Priority:      routePriorityString(route.Priority),
			Kind:          kind,
		})
	}
	return out
}

func routeDestinationString(route netlink.Route) string {
	if route.Dst == nil {
		return "default"
	}
	return route.Dst.String()
}

func routePriorityString(priority int) string {
	if priority == 0 {
		return ""
	}
	return strconv.Itoa(priority)
}

func ipString(ip net.IP) string {
	if ip == nil {
		return ""
	}
	return ip.String()
}

func wireGuardPeerState(peers []wgtypes.Peer) []networkmodule.WireGuardPeerState {
	out := make([]networkmodule.WireGuardPeerState, 0, len(peers))
	for _, peer := range peers {
		allowed := make([]string, 0, len(peer.AllowedIPs))
		for _, allowedIP := range peer.AllowedIPs {
			allowed = append(allowed, allowedIP.String())
		}
		handshake := ""
		if !peer.LastHandshakeTime.IsZero() {
			handshake = peer.LastHandshakeTime.Format(time.RFC3339)
		}
		endpoint := ""
		if peer.Endpoint != nil {
			endpoint = peer.Endpoint.String()
		}
		out = append(out, networkmodule.WireGuardPeerState{
			PublicKey:       peer.PublicKey.String(),
			Endpoint:        endpoint,
			AllowedIPs:      allowed,
			LatestHandshake: handshake,
			RxBytes:         uint64(peer.ReceiveBytes),
			TxBytes:         uint64(peer.TransmitBytes),
		})
	}
	return out
}

func inspectExitGatewayTables() ([]networkmodule.FirewallTableState, error) {
	conn, err := nftables.New()
	if err != nil {
		return nil, err
	}

	out := []networkmodule.FirewallTableState{}
	for _, tableRef := range managedExitGatewayTables() {
		exists, err := nftTableExists(conn, tableRef)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		tableState := networkmodule.FirewallTableState{
			Family: nftFamilyString(tableRef.Family),
			Name:   tableRef.Name,
		}
		chains, err := conn.ListChainsOfTableFamily(tableRef.Family)
		if err != nil {
			return nil, err
		}
		for _, chain := range chains {
			if chain.Table == nil || chain.Table.Name != tableRef.Name {
				continue
			}
			rules, err := conn.GetRules(tableRef, chain)
			if err != nil {
				return nil, err
			}
			chainState := networkmodule.FirewallChainState{
				Name:     chain.Name,
				Type:     string(chain.Type),
				Hook:     fmt.Sprint(chain.Hooknum),
				Priority: fmt.Sprint(chain.Priority),
				Rules:    make([]networkmodule.FirewallRuleState, 0, len(rules)),
			}
			for _, rule := range rules {
				chainState.Rules = append(chainState.Rules, firewallRuleState(rule))
			}
			tableState.Chains = append(tableState.Chains, chainState)
		}
		out = append(out, tableState)
	}
	return out, nil
}

func firewallRuleState(rule *nftables.Rule) networkmodule.FirewallRuleState {
	state := networkmodule.FirewallRuleState{Expressions: make([]string, 0, len(rule.Exprs))}
	for _, e := range rule.Exprs {
		if counter, ok := e.(*expr.Counter); ok {
			state.Packets = counter.Packets
			state.Bytes = counter.Bytes
		}
		state.Expressions = append(state.Expressions, describeNFTExpr(e))
	}
	return state
}

func describeNFTExpr(e expr.Any) string {
	switch v := e.(type) {
	case *expr.Meta:
		return fmt.Sprintf("meta key=%d register=%d", v.Key, v.Register)
	case *expr.Payload:
		return fmt.Sprintf("payload base=%d offset=%d length=%d register=%d", v.Base, v.Offset, v.Len, v.DestRegister)
	case *expr.Bitwise:
		return fmt.Sprintf("bitwise length=%d mask=%s xor=%s", v.Len, hex.EncodeToString(v.Mask), hex.EncodeToString(v.Xor))
	case *expr.Cmp:
		return fmt.Sprintf("cmp op=%d register=%d data=%s", v.Op, v.Register, describeNFTData(v.Data))
	case *expr.Ct:
		return fmt.Sprintf("ct key=%d register=%d", v.Key, v.Register)
	case *expr.Verdict:
		return fmt.Sprintf("verdict=%d", v.Kind)
	case *expr.Masq:
		return "masquerade"
	case *expr.Counter:
		return fmt.Sprintf("counter packets=%d bytes=%d", v.Packets, v.Bytes)
	default:
		return fmt.Sprintf("%T", e)
	}
}

func describeNFTData(data []byte) string {
	if len(data) == net.IPv4len || len(data) == net.IPv6len {
		return net.IP(data).String()
	}
	trimmed := strings.TrimRight(string(data), "\x00")
	if isPrintableASCII(trimmed) {
		return strconv.Quote(trimmed)
	}
	return hex.EncodeToString(data)
}

func isPrintableASCII(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < 32 || r > 126 {
			return false
		}
	}
	return true
}

func nftFamilyString(family nftables.TableFamily) string {
	switch family {
	case nftables.TableFamilyIPv4:
		return "ipv4"
	case nftables.TableFamilyIPv6:
		return "ipv6"
	case nftables.TableFamilyINet:
		return "inet"
	default:
		return fmt.Sprint(family)
	}
}
