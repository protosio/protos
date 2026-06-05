//go:build !ios

package mdns

import (
	"context"
	"fmt"
	"time"

	"github.com/grandcat/zeroconf"

	"github.com/protosio/protos/internal/invitations"
)

type zeroconfAdvertisement struct {
	server *zeroconf.Server
}

func startBonjourAdvertisement(instanceName string, service string, domain string, port int, txt []string) (bonjourAdvertisement, error) {
	server, err := zeroconf.Register(instanceName, service, domain, port, txt, nil)
	if err != nil {
		return nil, err
	}
	return &zeroconfAdvertisement{server: server}, nil
}

func (a *zeroconfAdvertisement) Shutdown() {
	if a == nil || a.server == nil {
		return
	}
	a.server.Shutdown()
}

func browseBonjour(ctx context.Context, timeout time.Duration) ([]invitations.NearbyInvite, error) {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	resolver, err := zeroconf.NewResolver()
	if err != nil {
		return nil, fmt.Errorf("create Bonjour resolver: %w", err)
	}
	return browseBonjourWithTimeout(ctx, timeout, func(browseCtx context.Context) ([]invitations.NearbyInvite, error) {
		entries := make(chan *zeroconf.ServiceEntry)
		if err := resolver.Browse(browseCtx, ServiceTCP, ServiceDomain, entries); err != nil {
			return nil, fmt.Errorf("browse Bonjour services: %w", err)
		}
		var items []invitations.NearbyInvite
		for entry := range entries {
			item, ok := parseEntry(entry)
			if ok {
				items = append(items, item)
			}
		}
		return items, nil
	})
}

func parseEntry(entry *zeroconf.ServiceEntry) (invitations.NearbyInvite, bool) {
	if entry == nil {
		return invitations.NearbyInvite{}, false
	}
	return parseTXTEntry(entry.Text, entry.HostName, entry.Port, entryIPs(entry))
}

func entryIPs(entry *zeroconf.ServiceEntry) []string {
	var ips []string
	for _, ip := range entry.AddrIPv4 {
		ips = append(ips, ip.String())
	}
	for _, ip := range entry.AddrIPv6 {
		ips = append(ips, ip.String())
	}
	return dedupeStrings(ips)
}
