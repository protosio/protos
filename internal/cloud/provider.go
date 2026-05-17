package cloud

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/protosio/protos/internal/pcrypto"
)

// ProviderRecord is the persisted cloud provider configuration.
type ProviderRecord struct {
	ID   string
	Name string
	Type Type
	Auth map[string]string
}

func newProviderRecord(name string, providerType Type, auth map[string]string) ProviderRecord {
	name = strings.TrimSpace(name)
	return ProviderRecord{
		ID:   name,
		Name: name,
		Type: providerType,
		Auth: copyStringMap(auth),
	}
}

func (record ProviderRecord) normalized() ProviderRecord {
	record.ID = strings.TrimSpace(record.ID)
	record.Name = strings.TrimSpace(record.Name)
	if record.ID == "" {
		record.ID = record.Name
	}
	if record.Name == "" {
		record.Name = record.ID
	}
	record.Auth = copyStringMap(record.Auth)
	return record
}

func (record ProviderRecord) Clone() ProviderRecord {
	return record.normalized()
}

type providerMetadata struct {
	record     ProviderRecord
	authFields []string
}

func newProviderMetadata(record ProviderRecord, authFields []string) providerMetadata {
	return providerMetadata{
		record:     record.normalized(),
		authFields: copyStringSlice(authFields),
	}
}

func (metadata providerMetadata) ProviderRecord() ProviderRecord {
	return metadata.record.Clone()
}

func (metadata providerMetadata) NameStr() string {
	return metadata.record.Name
}

func (metadata providerMetadata) TypeStr() string {
	return metadata.record.Type.String()
}

func (metadata providerMetadata) AuthFields() []string {
	return copyStringSlice(metadata.authFields)
}

// ProviderClient is a configured cloud provider client.
type ProviderClient interface {
	ProviderRecord() ProviderRecord
	NameStr() string
	TypeStr() string
	AuthFields() []string
	Init() error
}

// ComputeProvider provisions and manages VM instances.
type ComputeProvider interface {
	SupportedLocations() []string
	SupportedMachines(location string) (map[string]MachineSpec, error)
	NewInstance(name string, image string, pubKey string, machineType string, location string) (id string, err error)
	DeleteInstance(id string, location string) error
	StartInstance(id string, location string) error
	StopInstance(id string, location string) error
	GetInstanceInfo(id string, location string) (InstanceInfo, error)
}

// ImageProvider manages cloud VM images.
type ImageProvider interface {
	GetImages() (images map[string]ImageInfo, err error)
	GetProtosImages() (images map[string]ImageInfo, err error)
	AddImage(url string, hash string, version string, location string) (id string, err error)
	UploadLocalImage(imagePath string, imageName string, location string, timeout time.Duration) (id string, err error)
	RemoveImage(name string, location string) error
}

// VolumeProvider manages cloud block volumes.
type VolumeProvider interface {
	NewVolume(name string, size int, location string) (id string, err error)
	DeleteVolume(id string, location string) error
	AttachVolume(volumeID string, instanceID string, location string) error
	DettachVolume(volumeID string, instanceID string, location string) error
}

type ProviderDeps struct {
	SecretManager *pcrypto.Manager
}

type ProviderFactory interface {
	Type() Type
	AuthFields() []string
	NewClient(record ProviderRecord, deps ProviderDeps) (ProviderClient, error)
}

type typedProviderFactory[C any] struct {
	providerType Type
	authFields   []string
	decodeAuth   func(map[string]string) (C, error)
	newClient    func(providerMetadata, ProviderDeps, C) (ProviderClient, error)
}

func (factory typedProviderFactory[C]) Type() Type {
	return factory.providerType
}

func (factory typedProviderFactory[C]) AuthFields() []string {
	return copyStringSlice(factory.authFields)
}

func (factory typedProviderFactory[C]) NewClient(record ProviderRecord, deps ProviderDeps) (ProviderClient, error) {
	record = record.normalized()
	if record.Type == "" {
		record.Type = factory.providerType
	}
	if record.Type != factory.providerType {
		return nil, fmt.Errorf("provider factory for %q cannot create client for %q", factory.providerType, record.Type)
	}
	credentials, err := factory.decodeAuth(record.Auth)
	if err != nil {
		return nil, err
	}
	return factory.newClient(newProviderMetadata(record, factory.authFields), deps, credentials)
}

type providerRegistry struct {
	factories map[Type]ProviderFactory
}

func newProviderRegistry(factories ...ProviderFactory) *providerRegistry {
	registry := &providerRegistry{factories: map[Type]ProviderFactory{}}
	for _, factory := range factories {
		if factory == nil {
			continue
		}
		registry.factories[factory.Type()] = factory
	}
	return registry
}

func defaultProviderRegistry() *providerRegistry {
	return newProviderRegistry(newScalewayFactory())
}

func (registry *providerRegistry) factory(providerType Type) (ProviderFactory, bool) {
	if registry == nil {
		return nil, false
	}
	factory, found := registry.factories[providerType]
	return factory, found
}

func (registry *providerRegistry) types() []Type {
	if registry == nil {
		return nil
	}
	providerTypes := make([]Type, 0, len(registry.factories))
	for providerType := range registry.factories {
		providerTypes = append(providerTypes, providerType)
	}
	sort.Slice(providerTypes, func(i int, j int) bool {
		return providerTypes[i].String() < providerTypes[j].String()
	})
	return providerTypes
}

func requireComputeProvider(provider ProviderClient) (ComputeProvider, error) {
	computeProvider, ok := provider.(ComputeProvider)
	if !ok {
		return nil, unsupportedProviderCapability(provider, "compute")
	}
	return computeProvider, nil
}

func requireImageProvider(provider ProviderClient) (ImageProvider, error) {
	imageProvider, ok := provider.(ImageProvider)
	if !ok {
		return nil, unsupportedProviderCapability(provider, "image")
	}
	return imageProvider, nil
}

func requireVolumeProvider(provider ProviderClient) (VolumeProvider, error) {
	volumeProvider, ok := provider.(VolumeProvider)
	if !ok {
		return nil, unsupportedProviderCapability(provider, "volume")
	}
	return volumeProvider, nil
}

func unsupportedProviderCapability(provider ProviderClient, capability string) error {
	if provider == nil {
		return fmt.Errorf("cloud provider is nil and cannot support %s operations", capability)
	}
	return fmt.Errorf("cloud provider %q (%s) does not support %s operations", provider.NameStr(), provider.TypeStr(), capability)
}

func copyStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func copyStringSlice(in []string) []string {
	if in == nil {
		return nil
	}
	return append([]string(nil), in...)
}
