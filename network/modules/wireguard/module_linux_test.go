//go:build linux

package wireguard

import (
	"net"
	"testing"

	"github.com/vishvananda/netlink"
)

func TestDiffManagedRoutesDeletesStaleIPv6PeerRoutes(t *testing.T) {
	localIP := net.ParseIP("fd00::1")
	stalePeer := mustCIDR(t, "fd00::2/128")
	keptPeer := mustCIDR(t, "fd00::3/128")
	localRoute := mustCIDR(t, "fd00::1/128")

	existing := []netlink.Route{
		{Dst: stalePeer},
		{Dst: keptPeer, Src: localIP},
		{Dst: localRoute},
	}
	desired := []netlink.Route{
		{Dst: keptPeer},
	}

	delRoutes, addRoutes := diffManagedRoutes(existing, desired, localIP)
	if len(delRoutes) != 1 || delRoutes[0].Dst.String() != stalePeer.String() {
		t.Fatalf("deleted routes = %v, want only %s", delRoutes, stalePeer)
	}
	if len(addRoutes) != 0 {
		t.Fatalf("added routes = %v, want none", addRoutes)
	}
}

func mustCIDR(t *testing.T, value string) *net.IPNet {
	t.Helper()
	_, ipNet, err := net.ParseCIDR(value)
	if err != nil {
		t.Fatalf("parse CIDR %s: %v", value, err)
	}
	return ipNet
}
