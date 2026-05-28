//go:build linux

package wireguard

import "net"

type DNSManager struct {
}

func (m *DNSManager) AddDomainServer(domain string, server net.IP, port int) error {
	return nil
}

func (m *DNSManager) DelDomainServer(domain string) error {
	return nil
}

func (m *DNSManager) SetGlobalServer(server net.IP, port int) error {
	return nil
}

func (m *DNSManager) DelGlobalServer() error {
	return nil
}

func (m *DNSManager) HasGlobalServerBackup() bool {
	return false
}

func (m *DNSManager) Close() error {
	return nil
}

func NewDNSManager() (*DNSManager, error) {
	return &DNSManager{}, nil
}
