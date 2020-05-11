package cloud

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

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

// SupportedProviders returns a list of supported cloud providers
func SupportedProviders() []string {
	return []string{Scaleway.String()}
}

// ProviderInfo stores information about a cloud provider
type ProviderInfo struct {
	Name string `storm:"id"`
	Type Type
	Auth map[string]string
}

// Client returns a cloud provider client that can be used to run all the operations exposed by the Provider interface
func (pi ProviderInfo) Client() Provider {
	client, err := NewProvider(pi.Name, pi.Type.String())
	if err != nil {
		log.Fatal(err)
	}
	return client
}

// InstanceInfo holds information about a cloud instance
type InstanceInfo struct {
	VMID          string
	Name          string `storm:"id"`
	KeySeed       []byte // private SSH key stored only on the client
	PublicKey     []byte // public key used for wireguard connection
	PublicIP      string
	InternalIP    string
	CloudType     Type
	CloudName     string
	Location      string
	Network       string
	ProtosVersion string
	Volumes       []VolumeInfo
}

// VolumeInfo holds information about a data volume
type VolumeInfo struct {
	VolumeID string
	Name     string
	Size     uint64
}

// MachineSpec holds information about the hardware characteristics of vm or baremetal instance
type MachineSpec struct {
	Cores                uint32  // Nr of cores
	Memory               uint32  // MiB
	DefaultStorage       uint32  // GB
	Bandwidth            uint32  // Mbit
	IncludedDataTransfer uint32  // GB. 0 for unlimited
	Baremetal            bool    // true if machine is bare metal
	PriceMonthly         float32 // no currency conversion at the moment. Each cloud reports this differently
}

// ImageInfo holds information about a cloud image used for deploying an instance
type ImageInfo struct {
	ID       string
	Name     string
	Location string
}

// Provider allows interactions with cloud instances and images
type Provider interface {
	// Config methods
	AuthFields() (fields []string)                                     // returns the fields that are required to authenticate for a specific cloud provider
	SupportedLocations() (locations []string)                          // returns the supported locations for a specific cloud provider
	Init(auth map[string]string) error                                 // a cloud provider always needs to have Init called to configure it and test the credentials. If auth fails, Init should return an error
	GetInfo() ProviderInfo                                             // returns information that can be stored in the database and allows for re-creation of the provider
	SupportedMachines(location string) (map[string]MachineSpec, error) // returns a map of machine ids and their hardware specifications. A user will choose the machines for their instance

	// Instance methods
	NewInstance(name string, image string, pubKey string, machineType string, location string) (id string, err error)
	DeleteInstance(id string, location string) error
	StartInstance(id string, location string) error
	StopInstance(id string, location string) error
	GetInstanceInfo(id string, location string) (InstanceInfo, error)
	// Image methods
	GetImages() (images map[string]ImageInfo, err error)
	GetProtosImages() (images map[string]ImageInfo, err error)
	AddImage(url string, hash string, version string, location string) (id string, err error)
	UploadLocalImage(imagePath string, imageName string, location string) (id string, err error)
	RemoveImage(name string, location string) error
	// Volume methods
	// - size should by provided in megabytes
	NewVolume(name string, size int, location string) (id string, err error)
	DeleteVolume(id string, location string) error
	AttachVolume(volumeID string, instanceID string, location string) error
	DettachVolume(volumeID string, instanceID string, location string) error
}

// NewProvider creates a new cloud provider client
func NewProvider(cloudName string, cloud string) (Provider, error) {
	var client Provider
	var err error
	cloudType := Type(cloud)
	switch cloudType {
	// case DigitalOcean:
	// 	client, err = newDigitalOceanClient()
	case Scaleway:
		client = newScalewayClient(cloudName)
	default:
		err = errors.Errorf("Cloud '%s' not supported", cloud)
	}
	if err != nil {
		return nil, err
	}
	return client, nil
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
