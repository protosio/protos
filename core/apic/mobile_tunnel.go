package apic

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"time"

	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/protosio/protos/internal/network"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/provisioners"
)

const (
	mobileTunnelMTU              = 1280
	mobileTunnelWireGuardPort    = 10999
	mobileTunnelKeepaliveSeconds = 25
	mobileTunnelKeychainAccount  = "protos.mobile-tunnel.wireguard-private-key"
)

func (b *Backend) GetMobileTunnelConfig(ctx context.Context, in *pbApic.GetMobileTunnelConfigRequest) (*pbApic.GetMobileTunnelConfigResponse, error) {
	if b.protosClient == nil || b.protosClient.ProvisionerManager == nil || b.protosClient.Manager == nil || b.protosClient.KeyManager == nil {
		return nil, fmt.Errorf("mobile tunnel config is not available")
	}

	deviceID := strings.TrimSpace(in.GetDeviceId())
	if deviceID == "" {
		currentDevice, err := b.protosClient.Manager.GetCurrentDevice()
		if err != nil {
			return nil, fmt.Errorf("failed to resolve current device: %w", err)
		}
		deviceID = currentDevice.ID
	}

	instanceRef := strings.TrimSpace(in.GetInstance())
	dnsServer := strings.TrimSpace(in.GetDnsServer())
	cidrs := append([]string(nil), in.GetCidrs()...)
	if instanceRef == "" {
		route, found, err := b.mobileTunnelRouteForDevice(deviceID)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("instance is required when no exit route is configured for this device")
		}
		instanceRef = route.InstanceID
		if dnsServer == "" {
			dnsServer = route.DNSServer
		}
		if len(cidrs) == 0 {
			cidrs = append([]string(nil), route.CIDRs...)
		}
	}

	instance, err := b.protosClient.ProvisionerManager.GetInstance(instanceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve tunnel instance %q: %w", instanceRef, err)
	}
	localKey, err := b.protosClient.KeyManager.GetLocalKey()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve local tunnel key: %w", err)
	}
	config, err := buildMobileTunnelConfig(deviceID, instance, localKey, dnsServer, cidrs)
	if err != nil {
		return nil, err
	}
	return &pbApic.GetMobileTunnelConfigResponse{Config: config}, nil
}

func buildMobileTunnelConfig(deviceID string, instance provisioners.InstanceInfo, localKey *pcrypto.Key, dnsServer string, cidrs []string) (*pbApic.MobileTunnelConfig, error) {
	if localKey == nil {
		return nil, fmt.Errorf("local tunnel key is required")
	}
	if strings.TrimSpace(instance.PublicKey) == "" {
		return nil, fmt.Errorf("instance %q does not have a public key", instance.Name)
	}
	publicAddr, err := parsePublicEndpoint(instance.PublicIP)
	if err != nil {
		return nil, fmt.Errorf("instance %q does not have a usable public endpoint: %w", instance.Name, err)
	}
	if !isPublicExitIP(publicAddr.String()) {
		return nil, fmt.Errorf("instance %q does not have a routable public IP", instance.Name)
	}

	peerPublicKey, err := pcrypto.ConvertPublicEd25519ToCurve25519(instance.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive instance WireGuard public key: %w", err)
	}
	peerIdentity, err := pcrypto.CreatePublicKeyFromBase64(instance.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive instance tunnel address: %w", err)
	}
	normalizedCIDRs, err := network.NormalizeExitRouteCIDRs(cidrs)
	if err != nil {
		return nil, err
	}
	normalizedDNSServer, err := network.NormalizeDNSServer(dnsServer)
	if err != nil {
		return nil, err
	}

	localIPv6 := netip.PrefixFrom(localKey.IPv6Address(), 128).String()
	localIPv4 := netip.PrefixFrom(network.TunnelIPv4ForPublicKey(localKey.PublicString()), 32).String()
	peerIPv6 := netip.PrefixFrom(peerIdentity.IPv6Address(), 128).String()
	dnsServers := dnsServersForMobileConfig(normalizedDNSServer)
	excludedRoutes := endpointExcludedRoutes(publicAddr)
	allowedIPs := append([]string{peerIPv6}, normalizedCIDRs...)

	config := &pbApic.MobileTunnelConfig{
		ConfigId:                   mobileTunnelConfigID(deviceID, instance.ID, normalizedDNSServer, normalizedCIDRs),
		GeneratedAtUnix:            time.Now().Unix(),
		InstanceId:                 instance.ID,
		InstanceName:               instance.Name,
		PeerPublicKey:              peerPublicKey.String(),
		PeerEndpoint:               net.JoinHostPort(publicAddr.String(), fmt.Sprintf("%d", mobileTunnelWireGuardPort)),
		InterfaceAddresses:         []string{localIPv6, localIPv4},
		DnsServers:                 dnsServers,
		IncludedRoutes:             normalizedCIDRs,
		ExcludedRoutes:             excludedRoutes,
		Mtu:                        mobileTunnelMTU,
		AllowedIps:                 allowedIPs,
		PersistentKeepaliveSeconds: mobileTunnelKeepaliveSeconds,
		KeychainAccount:            mobileTunnelKeychainAccount,
		WireguardPrivateKey:        localKey.PrivateWG().String(),
	}
	return config, nil
}

func (b *Backend) mobileTunnelRouteForDevice(deviceID string) (network.ExitRoute, bool, error) {
	if b.protosClient == nil || b.protosClient.DB == nil {
		return network.ExitRoute{}, false, fmt.Errorf("database is not configured")
	}
	routes, err := network.GetExitRoutesForDevice(b.protosClient.DB, deviceID)
	if err != nil {
		return network.ExitRoute{}, false, err
	}
	for _, route := range routes {
		if network.NormalizeExitRouteStatus(route.DesiredStatus) == network.ExitRouteStatusActive {
			return route, true, nil
		}
	}
	return network.ExitRoute{}, false, nil
}

func parsePublicEndpoint(value string) (netip.Addr, error) {
	addr, err := netip.ParseAddr(strings.TrimSpace(value))
	if err != nil {
		return netip.Addr{}, err
	}
	return addr.Unmap(), nil
}

func endpointExcludedRoutes(addr netip.Addr) []string {
	if !addr.IsValid() {
		return nil
	}
	if addr.Is4() {
		return []string{netip.PrefixFrom(addr, 32).String()}
	}
	return []string{netip.PrefixFrom(addr, 128).String()}
}

func dnsServersForMobileConfig(server string) []string {
	server = strings.TrimSpace(server)
	if server == "" {
		return nil
	}
	host, _, err := net.SplitHostPort(server)
	if err != nil {
		return []string{server}
	}
	return []string{strings.Trim(host, "[]")}
}

func mobileTunnelConfigID(deviceID string, instanceID string, dnsServer string, cidrs []string) string {
	sum := sha256.Sum256([]byte(strings.Join(append([]string{deviceID, instanceID, dnsServer}, cidrs...), "\x00")))
	return fmt.Sprintf("%x", sum[:])
}
