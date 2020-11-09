package cloud

import (
	"time"

	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/release"
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

type CloudManager interface {
	// provider methods
	SupportedProviders() []string
	GetProvider(name string) (CloudProvider, error)
	GetProviders() ([]CloudProvider, error)
	NewProvider(cloudName string, cloud string) (CloudProvider, error)
	DeleteProvider(name string) error

	// instance methods
	DeployInstance(instanceName string, cloudName string, cloudLocation string, release release.Release, machineType string) (InstanceInfo, error)
	GetInstance(name string) (InstanceInfo, error)
	GetInstances() ([]InstanceInfo, error)
	DeleteInstance(name string, localOnly bool) error
	StartInstance(name string) error
	StopInstance(name string) error
	TunnelInstance(name string) error
	LogsInstance(name string) (string, error)
	InitDevInstance(instanceName string, cloudName string, locationName string, keyFile string, ipString string) error
}

type CloudProviderBase interface {
	Save() error     // saves the instance of the cloud provider (name and credentials) in the db
	NameStr() string // returns the name of the cloud provider
	TypeStr() string // returns the string formatted cloud type
}

// CloudProviderImplementation allows interactions with cloud instances and images
type CloudProviderImplementation interface {
	// Config methods
	SupportedLocations() (locations []string)                          // returns the supported locations for a specific cloud provider
	AuthFields() (fields []string)                                     // returns the fields that are required to authenticate for a specific cloud provider
	SetAuth(auth map[string]string) error                              // sets the credentials for a cloud provider
	Init() error                                                       // a cloud provider always needs to have Init called to configure it and test the credentials. If auth fails, Init should return an error
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
	UploadLocalImage(imagePath string, imageName string, location string, timeout time.Duration) (id string, err error)
	RemoveImage(name string, location string) error
	// Volume methods
	// - size should by provided in megabytes
	NewVolume(name string, size int, location string) (id string, err error)
	DeleteVolume(id string, location string) error
	AttachVolume(volumeID string, instanceID string, location string) error
	DettachVolume(volumeID string, instanceID string, location string) error
}

type CloudProvider interface {
	CloudProviderBase
	CloudProviderImplementation
}

// ProviderInfo stores information about a cloud provider
type ProviderInfo struct {
	CloudProviderImplementation `noms:"-"`
	cm                          *Manager `noms:"-"`

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
func (pi ProviderInfo) getCloudProvider() (CloudProvider, error) {
	var client CloudProviderImplementation
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
