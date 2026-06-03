//go:build darwin

package localmacos

import (
	"net"
	"testing"
)

func TestAllocateLocalMacOSStaticIPUsesObservedNATNetwork(t *testing.T) {
	_, network, err := net.ParseCIDR("192.168.138.0/23")
	if err != nil {
		t.Fatal(err)
	}
	gateway := net.ParseIP("192.168.139.3")
	ip, err := allocateLocalMacOSStaticIP(network, gateway, "vm-test", nil)
	if err != nil {
		t.Fatal(err)
	}
	parsed := net.ParseIP(ip)
	if parsed == nil || !network.Contains(parsed) {
		t.Fatalf("allocated IP %q outside network %s", ip, network.String())
	}
	if ip == gateway.String() {
		t.Fatalf("allocated gateway IP %q", ip)
	}
	if ip == "192.168.64.192" {
		t.Fatalf("allocated old hard-coded subnet IP %q", ip)
	}
}

func TestAllocateLocalMacOSStaticIPSkipsUsedAddresses(t *testing.T) {
	_, network, err := net.ParseCIDR("192.168.64.0/30")
	if err != nil {
		t.Fatal(err)
	}
	ip, err := allocateLocalMacOSStaticIP(network, net.ParseIP("192.168.64.1"), "vm-test", map[string]struct{}{
		"192.168.64.2": {},
	})
	if err == nil {
		t.Fatalf("allocated %q from exhausted network", ip)
	}
}
