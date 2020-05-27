package core

import (
	"time"

	"github.com/protosio/protos/internal/release"
)

// import "github.com/protosio/protos/internal/cloud"

// InstanceInfo holds information about a cloud instance
type InstanceInfo struct {
	VMID          string
	Name          string
	KeySeed       []byte // private SSH key stored only on the client
	PublicKey     []byte // public key used for wireguard connection
	PublicIP      string
	InternalIP    string
	CloudType     string
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

// ImageInfo holds information about a cloud image used for deploying an instance
type ImageInfo struct {
	ID       string
	Name     string
	Location string
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
	InitDevInstance(instanceName string, cloudName string, locationName string, keyFile string, ipString string) error
}

type CloudProviderInstance interface {
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
	CloudProviderInstance
	CloudProviderImplementation
}
