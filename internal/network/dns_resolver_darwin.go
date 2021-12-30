package network

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
)

const (
	networksetupPath = "/usr/sbin/networksetup"
	resolverPath     = "/etc/resolver"
)

type DNSManager struct {
}

func (m *DNSManager) AddDomainServer(domain string, server net.IP, port int) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	// check if the file exists
	resolverFile := resolverPath + "/" + domain
	// write file
	dnsData := fmt.Sprintf("domain %s\nport %d\nnameserver %s.%d\n", domain, port, server.String(), port)
	err := ioutil.WriteFile(resolverFile, []byte(dnsData), 0644)
	if err != nil {
		return fmt.Errorf("could not add DNS server for domainss '%s': %w", domain, err)
	}

	return nil
}

func (m *DNSManager) DelDomainServer(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	// check if the file exists
	resolverFile := resolverPath + "/" + domain
	err := os.Remove(resolverFile)
	if err != nil {
		return fmt.Errorf("could not delete DNS server for domain '%s': %w", domain, err)
	}

	return nil
}

// NewDNS returns a new DNS manager on MacOS
func NewDNSManager() (*DNSManager, error) {
	return &DNSManager{}, nil
}
