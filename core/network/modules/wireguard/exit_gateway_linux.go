//go:build linux

package wireguard

import (
	"fmt"
	"net"

	"github.com/google/nftables"
	"github.com/google/nftables/binaryutil"
	"github.com/google/nftables/expr"
)

const (
	exitNftTableName        = "protos_exit"
	exitNftForwardChain     = "forward"
	exitNftPostroutingChain = "postrouting"
	linuxInterfaceNameMax   = 15
	ipv4SourceOffset        = 12
	ipv4DestinationOffset   = 16
	ipv6SourceOffset        = 8
	ipv6DestinationOffset   = 24
)

type exitGatewayRoute struct {
	deviceID         string
	deviceName       string
	sourceIPv4CIDR   string
	sourceIPv6CIDR   string
	destinationCIDRs []string
}

type exitGatewayTableSpec struct {
	table  *nftables.Table
	chains []*nftables.Chain
	rules  []*nftables.Rule
}

func replaceExitGatewayRules(routes []exitGatewayRoute, ipv4OutIface string, ipv6OutIface string) error {
	specs, err := desiredExitGatewayTables(routes, ipv4OutIface, ipv6OutIface)
	if err != nil {
		return err
	}

	if err := applyExitGatewayTables(specs); err != nil {
		return err
	}
	log.Debugf("Configured nftables exit gateway rules: route_entries=%d ipv4_iface=%s ipv6_iface=%s", len(routes), ipv4OutIface, ipv6OutIface)
	return nil
}

func clearExitGatewayRules() error {
	if err := applyExitGatewayTables(nil); err != nil {
		return err
	}
	log.Debug("Cleared nftables exit gateway rules")
	return nil
}

func applyExitGatewayTables(specs []exitGatewayTableSpec) error {
	conn, err := nftables.New()
	if err != nil {
		return fmt.Errorf("failed to initialize nftables netlink client: %w", err)
	}
	if err := deleteManagedExitGatewayTables(conn); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return fmt.Errorf("failed to remove stale nftables exit gateway rules: %w", err)
	}

	if len(specs) == 0 {
		return nil
	}
	for _, spec := range specs {
		conn.AddTable(spec.table)
		for _, chain := range spec.chains {
			conn.AddChain(chain)
		}
		for _, rule := range spec.rules {
			conn.AddRule(rule)
		}
	}
	if err := conn.Flush(); err != nil {
		return fmt.Errorf("failed to apply nftables exit gateway rules: %w", err)
	}
	return nil
}

func deleteManagedExitGatewayTables(conn *nftables.Conn) error {
	for _, managed := range managedExitGatewayTables() {
		exists, err := nftTableExists(conn, managed)
		if err != nil {
			return fmt.Errorf("failed to inspect nftables table %v/%s: %w", managed.Family, managed.Name, err)
		}
		if exists {
			conn.DelTable(managed)
		}
	}
	return nil
}

func nftTableExists(conn *nftables.Conn, table *nftables.Table) (bool, error) {
	tables, err := conn.ListTablesOfFamily(table.Family)
	if err != nil {
		return false, err
	}
	for _, existing := range tables {
		if existing.Name == table.Name {
			return true, nil
		}
	}
	return false, nil
}

func managedExitGatewayTables() []*nftables.Table {
	return []*nftables.Table{
		{Name: exitNftTableName, Family: nftables.TableFamilyINet},
		{Name: exitNftTableName, Family: nftables.TableFamilyIPv4},
		{Name: exitNftTableName, Family: nftables.TableFamilyIPv6},
	}
}

func desiredExitGatewayTables(routes []exitGatewayRoute, ipv4OutIface string, ipv6OutIface string) ([]exitGatewayTableSpec, error) {
	specs := make([]exitGatewayTableSpec, 0, 2)
	if ipv4OutIface != "" {
		spec, ok, err := exitGatewayIPTable(nftables.TableFamilyIPv4, routes, ipv4OutIface)
		if err != nil {
			return nil, err
		}
		if ok {
			specs = append(specs, spec)
		}
	}
	if ipv6OutIface != "" {
		spec, ok, err := exitGatewayIPTable(nftables.TableFamilyIPv6, routes, ipv6OutIface)
		if err != nil {
			return nil, err
		}
		if ok {
			specs = append(specs, spec)
		}
	}
	return specs, nil
}

func exitGatewayIPTable(family nftables.TableFamily, routes []exitGatewayRoute, outIface string) (exitGatewayTableSpec, bool, error) {
	table := &nftables.Table{Name: exitNftTableName, Family: family}
	forward := &nftables.Chain{
		Name:     exitNftForwardChain,
		Table:    table,
		Type:     nftables.ChainTypeFilter,
		Hooknum:  nftables.ChainHookForward,
		Priority: nftables.ChainPriorityFilter,
		Policy:   chainPolicyRef(nftables.ChainPolicyAccept),
	}
	postrouting := &nftables.Chain{
		Name:     exitNftPostroutingChain,
		Table:    table,
		Type:     nftables.ChainTypeNAT,
		Hooknum:  nftables.ChainHookPostrouting,
		Priority: nftables.ChainPriorityNATSource,
	}

	filterRules := []*nftables.Rule{}
	natRules := []*nftables.Rule{}
	for _, route := range routes {
		sourceCIDR := route.sourceCIDR(family)
		if sourceCIDR == "" {
			continue
		}
		for _, destinationCIDR := range route.destinationCIDRs {
			matches, err := cidrMatchesFamily(destinationCIDR, family)
			if err != nil {
				return exitGatewayTableSpec{}, false, err
			}
			if !matches {
				continue
			}
			forwardRule, err := exitGatewayForwardRule(table, forward, sourceCIDR, destinationCIDR, outIface)
			if err != nil {
				return exitGatewayTableSpec{}, false, err
			}
			natRule, err := masqueradeRule(table, postrouting, sourceCIDR, destinationCIDR, outIface)
			if err != nil {
				return exitGatewayTableSpec{}, false, err
			}
			filterRules = append(filterRules, forwardRule)
			natRules = append(natRules, natRule)
		}
	}
	if len(filterRules) == 0 {
		return exitGatewayTableSpec{}, false, nil
	}

	returnRule, err := interfaceVerdictRule(table, forward, outIface, wireguardNetworkInterfaceName, true, expr.VerdictAccept)
	if err != nil {
		return exitGatewayTableSpec{}, false, err
	}
	dropOutbound, err := interfaceVerdictRule(table, forward, wireguardNetworkInterfaceName, outIface, false, expr.VerdictDrop)
	if err != nil {
		return exitGatewayTableSpec{}, false, err
	}
	dropInbound, err := interfaceVerdictRule(table, forward, outIface, wireguardNetworkInterfaceName, false, expr.VerdictDrop)
	if err != nil {
		return exitGatewayTableSpec{}, false, err
	}

	rules := make([]*nftables.Rule, 0, len(filterRules)+len(natRules)+3)
	rules = append(rules, filterRules...)
	rules = append(rules, returnRule, dropOutbound, dropInbound)
	rules = append(rules, natRules...)
	return exitGatewayTableSpec{
		table:  table,
		chains: []*nftables.Chain{forward, postrouting},
		rules:  rules,
	}, true, nil
}

func (r exitGatewayRoute) sourceCIDR(family nftables.TableFamily) string {
	switch family {
	case nftables.TableFamilyIPv4:
		return r.sourceIPv4CIDR
	case nftables.TableFamilyIPv6:
		return r.sourceIPv6CIDR
	default:
		return ""
	}
}

func exitGatewayForwardRule(table *nftables.Table, chain *nftables.Chain, sourceCIDR string, destinationCIDR string, outIface string) (*nftables.Rule, error) {
	exprs, err := interfaceMatchExprs(expr.MetaKeyIIFNAME, wireguardNetworkInterfaceName)
	if err != nil {
		return nil, err
	}
	outExprs, err := interfaceMatchExprs(expr.MetaKeyOIFNAME, outIface)
	if err != nil {
		return nil, err
	}
	sourceExprs, err := sourceCIDRMatchExprs(sourceCIDR)
	if err != nil {
		return nil, err
	}
	destinationExprs, err := destinationCIDRMatchExprs(destinationCIDR)
	if err != nil {
		return nil, err
	}
	exprs = append(exprs, outExprs...)
	exprs = append(exprs, sourceExprs...)
	if !isDefaultCIDR(destinationCIDR) {
		exprs = append(exprs, destinationExprs...)
	}
	exprs = append(exprs, &expr.Verdict{Kind: expr.VerdictAccept})
	return &nftables.Rule{
		Table: table,
		Chain: chain,
		Exprs: exprs,
	}, nil
}

func interfaceVerdictRule(table *nftables.Table, chain *nftables.Chain, inIface string, outIface string, establishedOnly bool, verdict expr.VerdictKind) (*nftables.Rule, error) {
	exprs, err := interfaceMatchExprs(expr.MetaKeyIIFNAME, inIface)
	if err != nil {
		return nil, err
	}
	outExprs, err := interfaceMatchExprs(expr.MetaKeyOIFNAME, outIface)
	if err != nil {
		return nil, err
	}
	exprs = append(exprs, outExprs...)
	if establishedOnly {
		exprs = append(exprs, establishedRelatedExprs()...)
	}
	exprs = append(exprs, &expr.Verdict{Kind: verdict})
	return &nftables.Rule{
		Table: table,
		Chain: chain,
		Exprs: exprs,
	}, nil
}

func masqueradeRule(table *nftables.Table, chain *nftables.Chain, sourceCIDR string, destinationCIDR string, outIface string) (*nftables.Rule, error) {
	exprs, err := sourceCIDRMatchExprs(sourceCIDR)
	if err != nil {
		return nil, err
	}
	if !isDefaultCIDR(destinationCIDR) {
		destinationExprs, err := destinationCIDRMatchExprs(destinationCIDR)
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, destinationExprs...)
	}
	outExprs, err := interfaceMatchExprs(expr.MetaKeyOIFNAME, outIface)
	if err != nil {
		return nil, err
	}
	exprs = append(exprs, outExprs...)
	exprs = append(exprs, &expr.Masq{})
	return &nftables.Rule{
		Table: table,
		Chain: chain,
		Exprs: exprs,
	}, nil
}

func sourceCIDRMatchExprs(cidr string) ([]expr.Any, error) {
	offset, length, err := cidrPayloadLocation(cidr, true)
	if err != nil {
		return nil, err
	}
	return payloadCIDRMatchExprs(cidr, offset, length, "source")
}

func destinationCIDRMatchExprs(cidr string) ([]expr.Any, error) {
	offset, length, err := cidrPayloadLocation(cidr, false)
	if err != nil {
		return nil, err
	}
	return payloadCIDRMatchExprs(cidr, offset, length, "destination")
}

func payloadCIDRMatchExprs(cidr string, offset uint32, length uint32, label string) ([]expr.Any, error) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid exit route %s prefix %q: %w", label, cidr, err)
	}
	var matchIP net.IP
	switch length {
	case net.IPv4len:
		matchIP = ip.To4()
	case net.IPv6len:
		matchIP = ip.To16()
	default:
		return nil, fmt.Errorf("unsupported %s prefix length %d", label, length)
	}
	if matchIP == nil || len(ipNet.Mask) != int(length) {
		return nil, fmt.Errorf("%s prefix %q does not match expected address length %d", label, cidr, length)
	}
	matchIP = matchIP.Mask(ipNet.Mask)
	return []expr.Any{
		&expr.Payload{
			DestRegister: 1,
			Base:         expr.PayloadBaseNetworkHeader,
			Offset:       offset,
			Len:          length,
		},
		&expr.Bitwise{
			SourceRegister: 1,
			DestRegister:   1,
			Len:            length,
			Mask:           append([]byte(nil), ipNet.Mask...),
			Xor:            make([]byte, length),
		},
		&expr.Cmp{
			Op:       expr.CmpOpEq,
			Register: 1,
			Data:     append([]byte(nil), matchIP...),
		},
	}, nil
}

func cidrPayloadLocation(cidr string, source bool) (uint32, uint32, error) {
	family, err := cidrFamily(cidr)
	if err != nil {
		return 0, 0, err
	}
	switch family {
	case nftables.TableFamilyIPv4:
		if source {
			return ipv4SourceOffset, net.IPv4len, nil
		}
		return ipv4DestinationOffset, net.IPv4len, nil
	case nftables.TableFamilyIPv6:
		if source {
			return ipv6SourceOffset, net.IPv6len, nil
		}
		return ipv6DestinationOffset, net.IPv6len, nil
	default:
		return 0, 0, fmt.Errorf("unsupported CIDR family for %q", cidr)
	}
}

func cidrMatchesFamily(cidr string, family nftables.TableFamily) (bool, error) {
	cidrFamily, err := cidrFamily(cidr)
	if err != nil {
		return false, err
	}
	return cidrFamily == family, nil
}

func cidrFamily(cidr string) (nftables.TableFamily, error) {
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return 0, fmt.Errorf("invalid exit route prefix %q: %w", cidr, err)
	}
	if ip.To4() != nil {
		return nftables.TableFamilyIPv4, nil
	}
	if ip.To16() != nil {
		return nftables.TableFamilyIPv6, nil
	}
	return 0, fmt.Errorf("exit route prefix %q is not an IPv4 or IPv6 prefix", cidr)
}

func isDefaultCIDR(cidr string) bool {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	ones, _ := ipNet.Mask.Size()
	return ones == 0
}

func interfaceMatchExprs(key expr.MetaKey, ifname string) ([]expr.Any, error) {
	ifNameData, err := nftIfNameData(ifname)
	if err != nil {
		return nil, err
	}
	return []expr.Any{
		&expr.Meta{Key: key, Register: 1},
		&expr.Cmp{
			Op:       expr.CmpOpEq,
			Register: 1,
			Data:     ifNameData,
		},
	}, nil
}

func establishedRelatedExprs() []expr.Any {
	return []expr.Any{
		&expr.Ct{Register: 1, SourceRegister: false, Key: expr.CtKeySTATE},
		&expr.Bitwise{
			SourceRegister: 1,
			DestRegister:   1,
			Len:            4,
			Mask:           binaryutil.NativeEndian.PutUint32(expr.CtStateBitESTABLISHED | expr.CtStateBitRELATED),
			Xor:            binaryutil.NativeEndian.PutUint32(0),
		},
		&expr.Cmp{
			Op:       expr.CmpOpNeq,
			Register: 1,
			Data:     []byte{0, 0, 0, 0},
		},
	}
}

func nftIfNameData(ifname string) ([]byte, error) {
	if ifname == "" {
		return nil, fmt.Errorf("empty interface name")
	}
	if len(ifname) > linuxInterfaceNameMax {
		return nil, fmt.Errorf("interface name %q exceeds Linux IFNAMSIZ limit", ifname)
	}
	data := make([]byte, linuxInterfaceNameMax+1)
	copy(data, []byte(ifname+"\x00"))
	return data, nil
}

func chainPolicyRef(policy nftables.ChainPolicy) *nftables.ChainPolicy {
	return &policy
}

func uniqueNonEmptyStrings(values ...string) []string {
	seen := map[string]struct{}{}
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}
