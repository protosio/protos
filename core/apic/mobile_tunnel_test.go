package apic

import (
	"net"
	"net/netip"
	"slices"
	"testing"

	"github.com/protosio/protos/internal/network"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/provisioners"
)

func TestBuildMobileTunnelConfigAppliesRouteIntent(t *testing.T) {
	t.Parallel()

	localKey := generateTestKey(t)
	peerKey := generateTestKey(t)
	instance := provisioners.InstanceInfo{
		ID:        "instance-1",
		Name:      "exit-node",
		PublicIP:  "8.8.8.8",
		PublicKey: peerKey.PublicString(),
	}

	config, err := buildMobileTunnelConfig(
		"device-1",
		instance,
		localKey,
		"[2606:4700:4700::1111]:5353",
		[]string{"10.4.2.9/16", "::/0", "10.4.0.0/16"},
	)
	if err != nil {
		t.Fatalf("buildMobileTunnelConfig: %v", err)
	}

	includedRoutes := []string{"10.4.0.0/16", "::/0"}
	normalizedDNS := net.JoinHostPort("2606:4700:4700::1111", "5353")
	peerWireGuardKey, err := pcrypto.ConvertPublicEd25519ToCurve25519(peerKey.PublicString())
	if err != nil {
		t.Fatalf("derive peer WireGuard key: %v", err)
	}
	peerIdentity, err := pcrypto.CreatePublicKeyFromBase64(peerKey.PublicString())
	if err != nil {
		t.Fatalf("derive peer identity: %v", err)
	}
	wantPeerIPv6 := netip.PrefixFrom(peerIdentity.IPv6Address(), 128).String()
	wantLocalIPv6 := netip.PrefixFrom(localKey.IPv6Address(), 128).String()
	wantLocalIPv4 := netip.PrefixFrom(network.TunnelIPv4ForPublicKey(localKey.PublicString()), 32).String()

	if config.GetConfigId() != mobileTunnelConfigID("device-1", "instance-1", normalizedDNS, includedRoutes) {
		t.Fatalf("config id = %q, want deterministic id for device/instance/dns/routes", config.GetConfigId())
	}
	if config.GetGeneratedAtUnix() <= 0 {
		t.Fatalf("generated timestamp = %d, want positive unix timestamp", config.GetGeneratedAtUnix())
	}
	if config.GetInstanceId() != "instance-1" || config.GetInstanceName() != "exit-node" {
		t.Fatalf("instance fields = %q/%q, want instance-1/exit-node", config.GetInstanceId(), config.GetInstanceName())
	}
	if config.GetPeerPublicKey() != peerWireGuardKey.String() {
		t.Fatalf("peer public key = %q, want %q", config.GetPeerPublicKey(), peerWireGuardKey.String())
	}
	if config.GetPeerEndpoint() != "8.8.8.8:10999" {
		t.Fatalf("peer endpoint = %q, want 8.8.8.8:10999", config.GetPeerEndpoint())
	}
	if got, want := config.GetInterfaceAddresses(), []string{wantLocalIPv6, wantLocalIPv4}; !slices.Equal(got, want) {
		t.Fatalf("interface addresses = %#v, want %#v", got, want)
	}
	if got, want := config.GetDnsServers(), []string{"2606:4700:4700::1111"}; !slices.Equal(got, want) {
		t.Fatalf("dns servers = %#v, want %#v", got, want)
	}
	if got := config.GetIncludedRoutes(); !slices.Equal(got, includedRoutes) {
		t.Fatalf("included routes = %#v, want %#v", got, includedRoutes)
	}
	if got, want := config.GetExcludedRoutes(), []string{"8.8.8.8/32"}; !slices.Equal(got, want) {
		t.Fatalf("excluded routes = %#v, want %#v", got, want)
	}
	if config.GetMtu() != mobileTunnelMTU {
		t.Fatalf("mtu = %d, want %d", config.GetMtu(), mobileTunnelMTU)
	}
	if got, want := config.GetAllowedIps(), []string{wantPeerIPv6, "10.4.0.0/16", "::/0"}; !slices.Equal(got, want) {
		t.Fatalf("allowed ips = %#v, want %#v", got, want)
	}
	if config.GetPersistentKeepaliveSeconds() != mobileTunnelKeepaliveSeconds {
		t.Fatalf("keepalive = %d, want %d", config.GetPersistentKeepaliveSeconds(), mobileTunnelKeepaliveSeconds)
	}
	if config.GetKeychainAccount() != mobileTunnelKeychainAccount {
		t.Fatalf("keychain account = %q, want %q", config.GetKeychainAccount(), mobileTunnelKeychainAccount)
	}
	if config.GetWireguardPrivateKey() != localKey.PrivateWG().String() {
		t.Fatal("wireguard private key does not match local key")
	}
}

func TestShouldReportHostAgent(t *testing.T) {
	t.Parallel()

	if shouldReportHostAgent("ios") {
		t.Fatal("iOS should not report host-agent status")
	}
	if !shouldReportHostAgent("darwin") {
		t.Fatal("macOS should report host-agent status")
	}
}

func generateTestKey(t *testing.T) *pcrypto.Key {
	t.Helper()

	key, err := pcrypto.CreateManager(nil).GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return key
}
