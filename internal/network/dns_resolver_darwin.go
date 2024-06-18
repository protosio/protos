package network

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
)

const (
	networksetupPath = "/usr/sbin/networksetup"
	resolverPath     = "/etc/resolver"
	sudoPath         = "/usr/bin/sudo"
	touchPath        = "/usr/bin/touch"
	chownPath        = "/usr/sbin/chown"
	rmPath           = "/bin/rm"
)

type DNSManager struct {
}

func (m *DNSManager) AddDomainServer(domain string, server net.IP, port int) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	// check if the file exists
	resolverFile := path.Join(resolverPath, domain)

	if _, err := os.Stat(resolverFile); err != nil {
		if os.IsNotExist(err) {
			// create the file
			cmd := exec.Command(sudoPath, touchPath, resolverFile)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to create dns resolver file: \n---- output ----\n%s-------------------", string(output))
			}

			// set ownership
			cmd = exec.Command(sudoPath, chownPath, "al3x", resolverFile)
			output, err = cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to set ownership for dns resolver file: \n---- output ----\n%s-------------------", string(output))
			}
		} else {
			return fmt.Errorf("could not check if resolver file exists '%s': %w", resolverFile, err)
		}
	}

	// write file
	dnsData := fmt.Sprintf("domain %s\nport %d\nnameserver %s.%d\n", domain, port, server.String(), port)
	err := os.WriteFile(resolverFile, []byte(dnsData), 0744)
	if err != nil {
		return fmt.Errorf("could not add DNS server for domains '%s': %w", domain, err)
	}

	return nil
}

func (m *DNSManager) DelDomainServer(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	// check if the file exists
	resolverFile := resolverPath + "/" + domain
	// delete the file
	cmd := exec.Command(sudoPath, rmPath, "-f", resolverFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete dns resolver file: \n---- output ----\n%s-------------------", string(output))
	}

	return nil
}

// NewDNS returns a new DNS manager on MacOS
func NewDNSManager() (*DNSManager, error) {
	return &DNSManager{}, nil
}
