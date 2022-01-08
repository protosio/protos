package network

import (
	"net"
)

type DNSManager struct {
}

func (m *DNSManager) AddDomainServer(domain string, server net.IP, port int) error {
	return nil
}

func (m *DNSManager) DelDomainServer(domain string) error {
	return nil
}

// NewDNS returns a new DNS manager on MacOS
func NewDNSManager() (*DNSManager, error) {
	return &DNSManager{}, nil
}
