//go:build linux

package wireguard

import (
	"net"
	"testing"

	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
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

func TestPeerEndpointsCopiesLearnedRoamingEndpoints(t *testing.T) {
	keyWithEndpoint := mustWireGuardPublicKey(t)
	keyWithoutEndpoint := mustWireGuardPublicKey(t)
	endpointIP := net.ParseIP("192.0.2.44")
	device := &wgtypes.Device{
		Peers: []wgtypes.Peer{
			{
				PublicKey: keyWithEndpoint,
				Endpoint:  &net.UDPAddr{IP: endpointIP, Port: 51820},
			},
			{PublicKey: keyWithoutEndpoint},
		},
	}

	endpoints := peerEndpoints(device)
	endpoint := endpoints[keyWithEndpoint]
	if endpoint == nil || endpoint.String() != "192.0.2.44:51820" {
		t.Fatalf("endpoint for peer = %v, want 192.0.2.44:51820", endpoint)
	}
	if _, found := endpoints[keyWithoutEndpoint]; found {
		t.Fatal("peer without endpoint should not have a preserved endpoint")
	}

	endpointIP[15] = 99
	if endpoint.String() != "192.0.2.44:51820" {
		t.Fatalf("endpoint was not copied defensively: %s", endpoint.String())
	}
}

func TestAppendStalePeerRemovalsPreservesDesiredPeers(t *testing.T) {
	keptKey := mustWireGuardPublicKey(t)
	staleKey := mustWireGuardPublicKey(t)
	desired := []wgtypes.PeerConfig{
		{
			PublicKey: keptKey,
			AllowedIPs: []net.IPNet{
				*mustCIDR(t, "fd00::2/128"),
			},
		},
	}
	active := &wgtypes.Device{
		Peers: []wgtypes.Peer{
			{PublicKey: keptKey},
			{PublicKey: staleKey},
		},
	}

	got := appendStalePeerRemovals(desired, active)
	if len(got) != 2 {
		t.Fatalf("peer configs = %d, want 2", len(got))
	}
	if got[0].Remove {
		t.Fatal("desired peer was marked for removal")
	}
	if !got[1].Remove || got[1].PublicKey != staleKey {
		t.Fatalf("stale removal = %+v, want remove for stale key", got[1])
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

func mustWireGuardPublicKey(t *testing.T) wgtypes.Key {
	t.Helper()
	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		t.Fatalf("generate WireGuard key: %v", err)
	}
	return key.PublicKey()
}
