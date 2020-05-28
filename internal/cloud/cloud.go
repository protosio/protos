package cloud

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/core"
	"github.com/protosio/protos/internal/util"
)

var log = util.GetLogger("cloud")

// Type represents a specific cloud (AWS, GCP, DigitalOcean etc.)
type Type string

func (ct Type) String() string {
	return string(ct)
}

const (
	// DigitalOcean cloud provider
	DigitalOcean = Type("digitalocean")
	// Scaleway cloud provider
	Scaleway = Type("scaleway")
	// Hyperkit is a MacOS hypervisor based on xhyve
	Hyperkit = Type("hyperkit")
)

// ProviderInfo stores information about a cloud provider
type ProviderInfo struct {
	core.CloudProviderImplementation `noms:"-"`
	cm                               *Manager `noms:"-"`

	Name string
	Type Type
	Auth map[string]string
}

// Save saves the provider information to disk
func (pi ProviderInfo) Save() error {
	err := pi.cm.db.InsertInMap(cloudDS, pi.Name, pi)
	if err != nil {
		return errors.Wrap(err, "Failed to save cloud provider info")
	}
	return nil
}

// NameStr returns the name of the cloud provider instance
func (pi ProviderInfo) NameStr() string {
	return pi.Name
}

// TypeStr returns the cloud type formatted as string
func (pi ProviderInfo) TypeStr() string {
	return pi.Type.String()
}

// GetInfo returns the ProviderInfo struct. Seems redundant but it's used via the Provider interface
func (pi ProviderInfo) GetInfo() ProviderInfo {
	return pi
}

// getClient creates a new cloud provider client based on the info in ProviderInfo
func (pi ProviderInfo) getCloudProvider() (core.CloudProvider, error) {
	var client core.CloudProviderImplementation
	var err error
	switch pi.Type {
	// case DigitalOcean:
	// 	client, err = newDigitalOceanClient()
	case Scaleway:
		client = newScalewayClient(&pi, pi.cm)
	default:
		err = errors.Errorf("Cloud '%s' not supported", pi.Type.String())
	}
	if err != nil {
		return nil, err
	}
	// if we have auth data, add it to the client
	if len(pi.Auth) > 0 {
		err := client.SetAuth(pi.Auth)
		if err != nil {
			return nil, err
		}
	}
	pi.CloudProviderImplementation = client
	return &pi, nil
}

func findInSlice(slice []string, value string) (int, bool) {
	for i, item := range slice {
		if item == value {
			return i, true
		}
	}
	return -1, false
}

// WaitForPort is a utility method that waits until a specific port is open on a specific host
func WaitForPort(host string, port string, maxTries int) error {
	tries := 0
	for {
		timeout := time.Second
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
		if err == nil && conn != nil {
			conn.Close()
			return nil
		}
		time.Sleep(3 * time.Second)
		tries++
		if tries == maxTries {
			return fmt.Errorf("Failed to connect to '%s:%s' after %d tries", host, port, maxTries)
		}
	}
}

// WaitForHTTP is a utility method that waits until a specific URL returns a succesful response
func WaitForHTTP(url string, maxTries int) error {
	client := http.Client{
		Timeout: 3 * time.Second,
	}
	tries := 0
	for {
		resp, err := client.Get(url)
		if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
			return nil
		}
		time.Sleep(3 * time.Second)
		tries++
		if tries == maxTries {
			return fmt.Errorf("Failed to do HTTP req to '%s' after %d tries", url, maxTries)
		}
	}
}
