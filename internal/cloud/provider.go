package cloud

import (
	"time"

	"github.com/pkg/errors"
)

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
