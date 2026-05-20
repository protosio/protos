package wireguard

import (
	"net"
	"net/netip"
)

func createIPv6Net(addr netip.Addr) *net.IPNet {
	if !addr.Is6() {
		return nil
	}

	ip := net.IP(addr.AsSlice())
	mask := net.CIDRMask(128, 128)
	return &net.IPNet{
		IP:   ip,
		Mask: mask,
	}
}
