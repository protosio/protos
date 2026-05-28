package wireguard

import (
	"fmt"
	"net"
	"net/netip"
	"strings"

	networkmodule "github.com/protosio/protos/internal/network/module"
)

const (
	exitTunnelIPv4CIDR = "100.64.0.0/10"
	exitTunnelIPv6CIDR = "0200::/8"
)

var (
	defaultExitRouteCIDRs = []string{
		"0.0.0.0/0",
		"::/0",
	}
	ipv4DefaultRoutePrefixes = []string{"0.0.0.0/1", "128.0.0.0/1"}
	ipv6DefaultRoutePrefixes = []string{"::/1", "8000::/1"}
)

func localExitRoute(config networkmodule.Config, peerSet networkmodule.Peers) (networkmodule.ExitRoute, bool) {
	if config.LocalPeerID == "" {
		return networkmodule.ExitRoute{}, false
	}
	for _, route := range peerSet.ExitRoutes {
		if route.DeviceID == config.LocalPeerID && route.InstanceID != "" {
			return route, true
		}
	}
	return networkmodule.ExitRoute{}, false
}

func exitGatewayRouteCIDRsByDevice(config networkmodule.Config, peerSet networkmodule.Peers) (map[string][]string, error) {
	routesByDevice := map[string][]string{}
	if config.LocalPeerID == "" {
		return routesByDevice, nil
	}
	for _, route := range peerSet.ExitRoutes {
		if route.InstanceID != config.LocalPeerID || route.DeviceID == "" {
			continue
		}
		cidrs, err := normalizedExitRouteCIDRs(route)
		if err != nil {
			return nil, err
		}
		routesByDevice[route.DeviceID] = appendUniqueStrings(routesByDevice[route.DeviceID], cidrs...)
	}
	return routesByDevice, nil
}

func normalizedExitRouteCIDRs(route networkmodule.ExitRoute) ([]string, error) {
	if len(route.CIDRs) == 0 {
		return append([]string(nil), defaultExitRouteCIDRs...), nil
	}
	cidrs := make([]string, 0, len(route.CIDRs))
	seen := map[string]struct{}{}
	for _, cidr := range route.CIDRs {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			return nil, fmt.Errorf("exit route CIDR must not be empty")
		}
		prefix, err := netip.ParsePrefix(cidr)
		if err != nil {
			return nil, fmt.Errorf("invalid exit route CIDR %q: %w", cidr, err)
		}
		prefix = prefix.Masked()
		normalized := prefix.String()
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		cidrs = append(cidrs, normalized)
	}
	if len(cidrs) == 0 {
		return append([]string(nil), defaultExitRouteCIDRs...), nil
	}
	return cidrs, nil
}

func exitRouteCIDRsUseFullTunnel(cidrs []string) bool {
	for _, cidr := range cidrs {
		prefix, err := netip.ParsePrefix(cidr)
		if err != nil {
			continue
		}
		if prefix.Bits() == 0 && (prefix.Addr().Is4() || prefix.Addr().Is6()) {
			return true
		}
	}
	return false
}

func exitRouteCIDRsNeedIPv4(cidrs []string) bool {
	for _, cidr := range cidrs {
		prefix, err := netip.ParsePrefix(cidr)
		if err == nil && prefix.Addr().Is4() {
			return true
		}
	}
	return false
}

func exitRouteCIDRsContainEndpoint(cidrs []string, endpoint net.IP) bool {
	addr, ok := netipAddrFromIP(endpoint)
	if !ok {
		return false
	}
	for _, cidr := range cidrs {
		prefix, err := netip.ParsePrefix(cidr)
		if err == nil && prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func netipAddrFromIP(ip net.IP) (netip.Addr, bool) {
	if ip == nil {
		return netip.Addr{}, false
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		return netip.AddrFrom4([4]byte{ipv4[0], ipv4[1], ipv4[2], ipv4[3]}), true
	}
	ipv6 := ip.To16()
	if ipv6 == nil {
		return netip.Addr{}, false
	}
	var octets [16]byte
	copy(octets[:], ipv6)
	return netip.AddrFrom16(octets), true
}

func ipNetFromAddr(addr netip.Addr, bits int) *net.IPNet {
	if !addr.IsValid() {
		return nil
	}
	ip := net.IP(addr.AsSlice())
	return &net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(bits, bits),
	}
}

func routePrefix(value string) (*net.IPNet, error) {
	ip, network, err := net.ParseCIDR(value)
	if err != nil {
		return nil, fmt.Errorf("invalid route prefix %q: %w", value, err)
	}
	network.IP = ip
	return network, nil
}

func appendUniqueStrings(values []string, additions ...string) []string {
	seen := make(map[string]struct{}, len(values)+len(additions))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	for _, value := range additions {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values
}
