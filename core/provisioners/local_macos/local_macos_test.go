//go:build darwin

package localmacos

import (
	"net"
	"os"
	"path/filepath"
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

func TestWriteJSONFileReplacesExistingFileAtomically(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metadata.json")

	if err := writeJSONFile(path, map[string]any{"status": "running"}); err != nil {
		t.Fatalf("write initial JSON: %v", err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatalf("chmod initial JSON: %v", err)
	}

	if err := writeJSONFile(path, map[string]any{"status": "stopped"}); err != nil {
		t.Fatalf("replace JSON: %v", err)
	}

	var got struct {
		Status string `json:"status"`
	}
	if err := readJSONFile(path, &got); err != nil {
		t.Fatalf("read replaced JSON: %v", err)
	}
	if got.Status != "stopped" {
		t.Fatalf("status = %q, want stopped", got.Status)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat replaced JSON: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0600 {
		t.Fatalf("mode = %v, want 0600", mode)
	}
}
