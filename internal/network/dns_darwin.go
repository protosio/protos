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

type dnsManager struct {
}

func (m *dnsManager) AddDomainServer(domain string, server net.IP) error {
	if domain == "" {
		return fmt.Errorf("Domain cannot be empty")
	}

	// check if the file exists
	resolverFile := resolverPath + "/" + domain
	_, err := os.Stat(resolverFile)
	if err == nil {
		return fmt.Errorf("Could not add DNS server for domain '%s': file '%s' already exists", domain, resolverFile)
	}

	// write file
	dnsData := fmt.Sprintf("nameserver %s\n", server.String())
	err = ioutil.WriteFile(resolverFile, []byte(dnsData), 0644)
	if err != nil {
		return fmt.Errorf("Could not add DNS server for domain '%s': %w", domain, err)
	}

	return nil
}

func (m *dnsManager) DelDomainServer(domain string) error {
	if domain == "" {
		return fmt.Errorf("Domain cannot be empty")
	}

	// check if the file exists
	resolverFile := resolverPath + "/" + domain
	err := os.Remove(resolverFile)
	if err != nil {
		return fmt.Errorf("Could not delete DNS server for domain '%s': %w", domain, err)
	}

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
