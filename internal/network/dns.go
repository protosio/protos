package network

import "net"

// DNSManager allows for addition and deletion of DNS servers
type DNSManager interface {
	AddDomainServer(domain string, server net.IP) error
	DelDomainServer(domain string) error
	AddServer(server net.IP) error
	DelServer(server net.IP) error
}
