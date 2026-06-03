package network

import (
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"

	"github.com/bokwoon95/sq"
	"github.com/protosio/protos/internal/db"
)

const ExitRouteStatusActive = "active"

var DefaultExitRouteCIDRs = []string{"0.0.0.0/0", "::/0"}

type ExitRoute struct {
	ID            string
	DeviceID      string
	InstanceID    string
	DesiredStatus string
	DNSServer     string
	CIDRs         []string
	cidrsJSON     string
}

func SetExitRoute(database *db.DB, deviceID string, instanceID string, dnsServer string, cidrs []string) (ExitRoute, error) {
	deviceID = strings.TrimSpace(deviceID)
	instanceID = strings.TrimSpace(instanceID)
	if deviceID == "" {
		return ExitRoute{}, fmt.Errorf("device id is required")
	}
	if instanceID == "" {
		return ExitRoute{}, fmt.Errorf("instance id is required")
	}
	normalizedDNSServer, err := NormalizeDNSServer(dnsServer)
	if err != nil {
		return ExitRoute{}, err
	}
	normalizedCIDRs, err := NormalizeExitRouteCIDRs(cidrs)
	if err != nil {
		return ExitRoute{}, err
	}

	route := ExitRoute{
		ID:            exitRouteID(deviceID),
		DeviceID:      deviceID,
		InstanceID:    instanceID,
		DesiredStatus: ExitRouteStatusActive,
		DNSServer:     normalizedDNSServer,
		CIDRs:         normalizedCIDRs,
	}

	existing, err := GetExitRoutesForDevice(database, deviceID)
	if err != nil {
		return ExitRoute{}, err
	}
	if len(existing) == 0 {
		if err := db.Insert(database, createExitRouteInsertMapper(route)); err != nil {
			return ExitRoute{}, err
		}
		return route, nil
	}
	if err := db.Update(database, createExitRouteUpdateMapper(route)); err != nil {
		return ExitRoute{}, err
	}
	return route, nil
}

func ClearExitRoute(database *db.DB, deviceID string) error {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return fmt.Errorf("device id is required")
	}
	existing, err := GetExitRoutesForDevice(database, deviceID)
	if err != nil {
		return err
	}
	if len(existing) == 0 {
		return nil
	}
	return db.Delete(database, createExitRouteDeleteMapper(exitRouteID(deviceID)))
}

func GetExitRoutes(database *db.DB) ([]ExitRoute, error) {
	routes, err := db.SelectMultiple(database, createExitRouteQueryMapper(nil))
	if err != nil {
		return nil, fmt.Errorf("retrieve exit routes: %w", err)
	}
	return normalizeExitRouteRows(routes)
}

func GetExitRoutesForDevice(database *db.DB, deviceID string) ([]ExitRoute, error) {
	deviceID = strings.TrimSpace(deviceID)
	model := sq.New[db.EXIT_ROUTE]("")
	routes, err := db.SelectMultiple(database, createExitRouteQueryMapper([]sq.Predicate{model.DEVICE_ID.EqString(deviceID)}))
	if err != nil {
		return nil, fmt.Errorf("retrieve exit route for device %s: %w", deviceID, err)
	}
	return normalizeExitRouteRows(routes)
}

func exitRouteID(deviceID string) string {
	return strings.TrimSpace(deviceID)
}

func NormalizeExitRouteStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return ExitRouteStatusActive
	}
	return status
}

func NormalizeExitRouteCIDRs(cidrs []string) ([]string, error) {
	if len(cidrs) == 0 {
		return append([]string(nil), DefaultExitRouteCIDRs...), nil
	}

	out := make([]string, 0, len(cidrs))
	seen := map[string]struct{}{}
	for _, cidr := range cidrs {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			return nil, fmt.Errorf("exit route CIDR must not be empty")
		}
		prefix, err := netip.ParsePrefix(cidr)
		if err != nil {
			return nil, fmt.Errorf("invalid exit route CIDR %q: %w", cidr, err)
		}
		prefix = prefix.Masked()
		if !prefix.IsValid() {
			return nil, fmt.Errorf("invalid exit route CIDR %q", cidr)
		}
		normalized := prefix.String()
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	if len(out) == 0 {
		return append([]string(nil), DefaultExitRouteCIDRs...), nil
	}
	return out, nil
}

func ExitRouteUsesFullTunnel(route ExitRoute) bool {
	return RouteCIDRsUseFullTunnel(route.CIDRs)
}

func RouteCIDRsUseFullTunnel(cidrs []string) bool {
	normalized, err := NormalizeExitRouteCIDRs(cidrs)
	if err != nil {
		return false
	}
	for _, cidr := range normalized {
		prefix, err := netip.ParsePrefix(cidr)
		if err != nil {
			continue
		}
		if prefix.Bits() == 0 && (prefix.Addr().Is4() || prefix.Addr().Is6()) {
			return true
		}
	}
	return false
}

func normalizeExitRouteRows(routes []ExitRoute) ([]ExitRoute, error) {
	for i := range routes {
		cidrs, err := parseStoredExitRouteCIDRs(routes[i].cidrsJSON)
		if err != nil {
			return nil, fmt.Errorf("parse exit route %s CIDRs: %w", routes[i].ID, err)
		}
		routes[i].CIDRs = cidrs
	}
	return routes, nil
}

func parseStoredExitRouteCIDRs(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return NormalizeExitRouteCIDRs(nil)
	}
	var cidrs []string
	if err := json.Unmarshal([]byte(value), &cidrs); err != nil {
		return nil, err
	}
	return NormalizeExitRouteCIDRs(cidrs)
}

func encodeExitRouteCIDRs(cidrs []string) string {
	normalized, err := NormalizeExitRouteCIDRs(cidrs)
	if err != nil {
		normalized = DefaultExitRouteCIDRs
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return `["0.0.0.0/0","::/0"]`
	}
	return string(encoded)
}

func createExitRouteInsertMapper(route ExitRoute) db.InsertMapper {
	return func() sq.InsertQuery {
		m := sq.New[db.EXIT_ROUTE]("")
		mapper := func(col *sq.Column) {
			col.SetString(m.ID, route.ID)
			col.SetString(m.DEVICE_ID, route.DeviceID)
			col.SetString(m.INSTANCE_ID, route.InstanceID)
			col.SetString(m.DESIRED_STATUS, NormalizeExitRouteStatus(route.DesiredStatus))
			col.SetString(m.DNS_SERVER, route.DNSServer)
			col.SetString(m.CIDRS, encodeExitRouteCIDRs(route.CIDRs))
		}
		return sq.InsertInto(m).ColumnValues(mapper)
	}
}

func createExitRouteUpdateMapper(route ExitRoute) db.UpdateMapper {
	return func() sq.UpdateQuery {
		m := sq.New[db.EXIT_ROUTE]("")
		mapper := func(col *sq.Column) {
			col.SetString(m.DEVICE_ID, route.DeviceID)
			col.SetString(m.INSTANCE_ID, route.InstanceID)
			col.SetString(m.DESIRED_STATUS, NormalizeExitRouteStatus(route.DesiredStatus))
			col.SetString(m.DNS_SERVER, route.DNSServer)
			col.SetString(m.CIDRS, encodeExitRouteCIDRs(route.CIDRs))
		}
		return sq.Update(m).SetFunc(mapper).Where(m.ID.EqString(route.ID))
	}
}

func createExitRouteDeleteMapper(id string) db.DeleteMapper {
	return func() sq.DeleteQuery {
		m := sq.New[db.EXIT_ROUTE]("")
		return sq.DeleteFrom(m).Where(m.ID.EqString(id))
	}
}

func createExitRouteQueryMapper(predicates []sq.Predicate) db.QueryMapper[ExitRoute] {
	m := sq.New[db.EXIT_ROUTE]("")
	query := sq.From(m)
	if len(predicates) > 0 {
		query = query.Where(predicates...)
	}

	return func() (sq.SelectQuery, func(row *sq.Row) ExitRoute) {
		mapper := func(row *sq.Row) ExitRoute {
			return ExitRoute{
				ID:            row.StringField(m.ID),
				DeviceID:      row.StringField(m.DEVICE_ID),
				InstanceID:    row.StringField(m.INSTANCE_ID),
				DesiredStatus: row.StringField(m.DESIRED_STATUS),
				DNSServer:     row.StringField(m.DNS_SERVER),
				cidrsJSON:     row.StringField(m.CIDRS),
			}
		}
		return query, mapper
	}
}

func NormalizeDNSServer(server string) (string, error) {
	server = strings.TrimSpace(server)
	if server == "" {
		return "", nil
	}

	if addr, err := netip.ParseAddr(server); err == nil {
		return net.JoinHostPort(addr.String(), "53"), nil
	}

	host, portString, err := net.SplitHostPort(server)
	if err != nil {
		return "", fmt.Errorf("DNS server must be an IP address with optional port: %w", err)
	}
	host = strings.Trim(host, "[]")
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return "", fmt.Errorf("DNS server host must be an IP address: %w", err)
	}
	port, err := strconv.Atoi(portString)
	if err != nil || port < 1 || port > 65535 {
		return "", fmt.Errorf("DNS server port must be between 1 and 65535")
	}
	return net.JoinHostPort(addr.String(), strconv.Itoa(port)), nil
}
