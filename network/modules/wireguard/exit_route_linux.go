//go:build linux

package wireguard

import (
	"net"
	"net/netip"

	networkmodule "github.com/protosio/protos/internal/network/module"
)

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
