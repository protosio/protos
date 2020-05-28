package vpn

import (
	"net"
)

type dnsManager struct {
}

func (m *dnsManager) AddDomainServer(domain string, server net.IP) error {
	return nil
}

func (m *dnsManager) DelDomainServer(domain string) error {
	return nil
}

func (m *dnsManager) AddServer(server net.IP) error {
	return nil
}

func (m *dnsManager) DelServer(server net.IP) error {
	return nil
}

// NewDNS returns a new DNS manager on MacOS
func NewDNS() (DNSManager, error) {
	return &dnsManager{}, nil
}
