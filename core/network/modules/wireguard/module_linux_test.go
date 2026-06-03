//go:build linux

package wireguard

import (
	"bytes"
	"net"
	"testing"

	"github.com/google/nftables"
	"github.com/google/nftables/expr"
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

	delRoutes, addRoutes := diffManagedRoutes(existing, desired, localIP, nil)
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

func TestIsDefaultRouteDstAcceptsNilAndZeroPrefix(t *testing.T) {
	if !isDefaultRouteDst(nil) {
		t.Fatal("nil route destination should be treated as default")
	}
	if !isDefaultRouteDst(mustCIDR(t, "0.0.0.0/0")) {
		t.Fatal("0.0.0.0/0 should be treated as default")
	}
	if !isDefaultRouteDst(mustCIDR(t, "::/0")) {
		t.Fatal("::/0 should be treated as default")
	}
	if isDefaultRouteDst(mustCIDR(t, "192.0.2.0/24")) {
		t.Fatal("non-default IPv4 route was treated as default")
	}
	if isDefaultRouteDst(mustCIDR(t, "2001:db8::/32")) {
		t.Fatal("non-default IPv6 route was treated as default")
	}
}

func TestParseProcDefaultRoute(t *testing.T) {
	ipv4 := []byte("Iface\tDestination\tGateway\tFlags\tRefCnt\tUse\tMetric\tMask\tMTU\tWindow\tIRTT\n" +
		"protosWG\t00000000\t0100000A\t0003\t0\t0\t0\t00000000\t0\t0\t0\n" +
		"eth0\t00000000\t0100000A\t0003\t0\t0\t100\t00000000\t0\t0\t0\n")
	if got := parseProcDefaultRoute(ipv4, false); got != "eth0" {
		t.Fatalf("IPv4 default interface = %q, want eth0", got)
	}

	ipv6 := []byte(
		"00000000000000000000000000000000 00000000 00000000000000000000000000000000 00000000 fe800000000000000000000000000001 00000000 00000001 00000000 00200200 protosWG\n" +
			"00000000000000000000000000000000 00000000 00000000000000000000000000000000 00000000 fe800000000000000000000000000001 00000000 00000001 00000000 00200200 eth0\n")
	if got := parseProcDefaultRoute(ipv6, true); got != "eth0" {
		t.Fatalf("IPv6 default interface = %q, want eth0", got)
	}
}

func TestDesiredExitGatewayTablesUsesNativeNFTablesRules(t *testing.T) {
	routes := []exitGatewayRoute{
		{
			sourceIPv4CIDR:   "100.65.169.124/32",
			sourceIPv6CIDR:   "0200::123/128",
			destinationCIDRs: []string{"34.160.111.145/32", "2001:db8::45/128"},
		},
	}
	specs, err := desiredExitGatewayTables(routes, "eth0", "eth0")
	if err != nil {
		t.Fatalf("desired exit gateway tables: %v", err)
	}
	if len(specs) != 2 {
		t.Fatalf("table specs = %d, want IPv4 + IPv6 exit tables", len(specs))
	}

	ipv4 := findExitTableSpec(t, specs, nftables.TableFamilyIPv4)
	ipv4Forward := rulesForChain(ipv4, exitNftForwardChain)
	if got := len(ipv4Forward); got != 4 {
		t.Fatalf("IPv4 forward rules = %d, want allow + return + outbound drop + inbound drop", got)
	}
	if !hasVerdict(ipv4Forward[0], expr.VerdictAccept) {
		t.Fatal("forward rule must explicitly accept exit-route forwarding")
	}
	assertCIDRMatch(t, ipv4Forward[0], 4, ipv4SourceOffset, "100.65.169.124", []byte{0xff, 0xff, 0xff, 0xff})
	assertCIDRMatch(t, ipv4Forward[0], 7, ipv4DestinationOffset, "34.160.111.145", []byte{0xff, 0xff, 0xff, 0xff})
	if !hasConntrackEstablishedMatch(ipv4Forward[1]) {
		t.Fatal("return-path filter rule must be scoped to established/related flows")
	}
	if !hasVerdict(ipv4Forward[2], expr.VerdictDrop) || !hasVerdict(ipv4Forward[3], expr.VerdictDrop) {
		t.Fatal("forward table must drop unapproved exit traffic in both directions")
	}

	ipv4NAT := rulesForChain(ipv4, exitNftPostroutingChain)
	if len(ipv4NAT) != 1 {
		t.Fatalf("IPv4 NAT rules = %d, want one", len(ipv4NAT))
	}
	assertMasqueradeRule(t, ipv4NAT[0], "100.65.169.124", []byte{0xff, 0xff, 0xff, 0xff}, "34.160.111.145", []byte{0xff, 0xff, 0xff, 0xff}, "eth0")

	ipv6 := findExitTableSpec(t, specs, nftables.TableFamilyIPv6)
	ipv6NAT := rulesForChain(ipv6, exitNftPostroutingChain)
	if len(ipv6NAT) != 1 {
		t.Fatalf("IPv6 NAT rules = %d, want one", len(ipv6NAT))
	}
	ipv6Mask := bytes.Repeat([]byte{0xff}, 16)
	assertMasqueradeRule(t, ipv6NAT[0], "0200::123", ipv6Mask, "2001:db8::45", ipv6Mask, "eth0")
}

func TestDesiredExitGatewayTablesFullTunnelStaysSourceScoped(t *testing.T) {
	routes := []exitGatewayRoute{
		{
			sourceIPv4CIDR:   "100.65.169.124/32",
			destinationCIDRs: []string{"0.0.0.0/0"},
		},
	}
	specs, err := desiredExitGatewayTables(routes, "eth0", "")
	if err != nil {
		t.Fatalf("desired exit gateway tables: %v", err)
	}

	ipv4 := findExitTableSpec(t, specs, nftables.TableFamilyIPv4)
	ipv4Forward := rulesForChain(ipv4, exitNftForwardChain)
	if len(ipv4Forward[0].Exprs) != 8 {
		t.Fatalf("full-tunnel forward expression count = %d, want no destination CIDR match", len(ipv4Forward[0].Exprs))
	}
	assertCIDRMatch(t, ipv4Forward[0], 4, ipv4SourceOffset, "100.65.169.124", []byte{0xff, 0xff, 0xff, 0xff})

	ipv4NAT := rulesForChain(ipv4, exitNftPostroutingChain)
	if len(ipv4NAT[0].Exprs) != 6 {
		t.Fatalf("full-tunnel NAT expression count = %d, want source + oif + masquerade only", len(ipv4NAT[0].Exprs))
	}
	assertMasqueradeRule(t, ipv4NAT[0], "100.65.169.124", []byte{0xff, 0xff, 0xff, 0xff}, "", nil, "eth0")
}

func TestNFTIfNameDataMatchesLinuxIFNAMSIZ(t *testing.T) {
	got, err := nftIfNameData("eth0")
	if err != nil {
		t.Fatalf("nft ifname data: %v", err)
	}
	if len(got) != 16 {
		t.Fatalf("ifname data length = %d, want 16", len(got))
	}
	if !bytes.Equal(got[:5], []byte{'e', 't', 'h', '0', 0}) {
		t.Fatalf("ifname data prefix = %v, want eth0 NUL", got[:5])
	}
	if _, err := nftIfNameData("1234567890123456"); err == nil {
		t.Fatal("expected overlong interface name to fail")
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

func findExitTableSpec(t *testing.T, specs []exitGatewayTableSpec, family nftables.TableFamily) exitGatewayTableSpec {
	t.Helper()
	for _, spec := range specs {
		if spec.table.Family == family {
			return spec
		}
	}
	t.Fatalf("missing table family %v", family)
	return exitGatewayTableSpec{}
}

func rulesForChain(spec exitGatewayTableSpec, chain string) []*nftables.Rule {
	rules := []*nftables.Rule{}
	for _, rule := range spec.rules {
		if rule.Chain != nil && rule.Chain.Name == chain {
			rules = append(rules, rule)
		}
	}
	return rules
}

func hasVerdict(rule *nftables.Rule, kind expr.VerdictKind) bool {
	for _, ruleExpr := range rule.Exprs {
		verdict, ok := ruleExpr.(*expr.Verdict)
		if ok && verdict.Kind == kind {
			return true
		}
	}
	return false
}

func hasConntrackEstablishedMatch(rule *nftables.Rule) bool {
	for _, ruleExpr := range rule.Exprs {
		if _, ok := ruleExpr.(*expr.Ct); ok {
			return true
		}
	}
	return false
}

func assertMasqueradeRule(t *testing.T, rule *nftables.Rule, source string, sourceMask []byte, destination string, destinationMask []byte, outIface string) {
	t.Helper()
	wantExprs := 6
	if destination != "" {
		wantExprs = 9
	}
	if len(rule.Exprs) != wantExprs {
		t.Fatalf("masquerade rule expression count = %d, want %d", len(rule.Exprs), wantExprs)
	}
	assertCIDRMatch(t, rule, 0, payloadOffsetForMask(sourceMask, true), source, sourceMask)
	if destination != "" {
		assertCIDRMatch(t, rule, 3, payloadOffsetForMask(destinationMask, false), destination, destinationMask)
	}

	ifnameCmpIndex := wantExprs - 2
	ifnameCmp, ok := rule.Exprs[ifnameCmpIndex].(*expr.Cmp)
	if !ok {
		t.Fatalf("expr[%d] = %T, want *expr.Cmp", ifnameCmpIndex, rule.Exprs[ifnameCmpIndex])
	}
	wantIface, err := nftIfNameData(outIface)
	if err != nil {
		t.Fatalf("nft ifname data: %v", err)
	}
	if !bytes.Equal(ifnameCmp.Data, wantIface) {
		t.Fatalf("oifname compare = %v, want %v", ifnameCmp.Data, wantIface)
	}
	if _, ok := rule.Exprs[wantExprs-1].(*expr.Masq); !ok {
		t.Fatalf("expr[%d] = %T, want *expr.Masq", wantExprs-1, rule.Exprs[wantExprs-1])
	}
}

func assertCIDRMatch(t *testing.T, rule *nftables.Rule, start int, offset uint32, ip string, mask []byte) {
	t.Helper()
	if len(rule.Exprs) < start+3 {
		t.Fatalf("rule has %d expressions, need at least %d", len(rule.Exprs), start+3)
	}
	payload, ok := rule.Exprs[start].(*expr.Payload)
	if !ok {
		t.Fatalf("expr[%d] = %T, want *expr.Payload", start, rule.Exprs[start])
	}
	if payload.Offset != offset {
		t.Fatalf("payload offset = %d, want %d", payload.Offset, offset)
	}
	bitwise, ok := rule.Exprs[start+1].(*expr.Bitwise)
	if !ok {
		t.Fatalf("expr[%d] = %T, want *expr.Bitwise", start+1, rule.Exprs[start+1])
	}
	if !bytes.Equal(bitwise.Mask, mask) {
		t.Fatalf("CIDR mask = %v, want %v", bitwise.Mask, mask)
	}
	cmp, ok := rule.Exprs[start+2].(*expr.Cmp)
	if !ok {
		t.Fatalf("expr[%d] = %T, want *expr.Cmp", start+2, rule.Exprs[start+2])
	}
	wantIP := net.ParseIP(ip)
	if len(mask) == net.IPv4len {
		wantIP = wantIP.To4()
	} else {
		wantIP = wantIP.To16()
	}
	if !bytes.Equal(cmp.Data, wantIP) {
		t.Fatalf("CIDR compare = %v, want %v", cmp.Data, wantIP)
	}
}

func payloadOffsetForMask(mask []byte, source bool) uint32 {
	if len(mask) == net.IPv4len {
		if source {
			return ipv4SourceOffset
		}
		return ipv4DestinationOffset
	}
	if source {
		return ipv6SourceOffset
	}
	return ipv6DestinationOffset
}
