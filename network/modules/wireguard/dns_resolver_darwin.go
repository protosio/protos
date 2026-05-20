//go:build darwin

package wireguard

import (
	"fmt"
	"net"
	"os"
	"path"
)

const (
	resolverPath = "/etc/resolver"
)

type DNSManager struct {
}

func (m *DNSManager) AddDomainServer(domain string, server net.IP, port int) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}
	if server == nil {
		return fmt.Errorf("server cannot be empty")
	}

	if err := os.MkdirAll(resolverPath, 0755); err != nil {
		return fmt.Errorf("could not create DNS resolver directory '%s': %w", resolverPath, err)
	}
	resolverFile := path.Join(resolverPath, domain)

	dnsData := fmt.Sprintf("domain %s\nport %d\nnameserver %s\n", domain, port, server.String())
	if err := os.WriteFile(resolverFile, []byte(dnsData), 0644); err != nil {
		return fmt.Errorf("could not add DNS server for domains '%s': %w", domain, err)
	}

	return nil
}

func (m *DNSManager) DelDomainServer(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	resolverFile := path.Join(resolverPath, domain)
	if err := os.Remove(resolverFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete dns resolver file '%s': %w", resolverFile, err)
	}

	return nil
}

func NewDNSManager() (*DNSManager, error) {
	return &DNSManager{}, nil
}
