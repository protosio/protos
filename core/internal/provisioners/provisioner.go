package provisioners

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/protosio/protos/internal/pcrypto"
)

// ProvisionerRecord is the persisted provisioner configuration.
//
// The backing database and CLI still use "cloud provider" terminology for
// compatibility, but the runtime abstraction is a provisioner.
type ProvisionerRecord struct {
	ID   string
	Name string
	Type Type
	Auth map[string]string
}

// ProviderRecord is kept as a compatibility alias for existing cloud APIs.
type ProviderRecord = ProvisionerRecord

func newProvisionerRecord(name string, provisionerType Type, auth map[string]string) ProvisionerRecord {
	name = strings.TrimSpace(name)
	return ProvisionerRecord{
		Name: name,
		Type: provisionerType,
		Auth: copyStringMap(auth),
	}
}

func (record ProvisionerRecord) normalized() ProvisionerRecord {
	record.ID = strings.TrimSpace(record.ID)
	record.Name = strings.TrimSpace(record.Name)
	record.Auth = copyStringMap(record.Auth)
	return record
}

func (record ProvisionerRecord) Clone() ProvisionerRecord {
	return record.normalized()
}

// ProvisionerMetadata provides common metadata methods for concrete provisioners.
type ProvisionerMetadata struct {
	record     ProvisionerRecord
	authFields []string
}

func newProvisionerMetadata(record ProvisionerRecord, authFields []string) ProvisionerMetadata {
	return ProvisionerMetadata{
		record:     record.normalized(),
		authFields: copyStringSlice(authFields),
	}
}

func (metadata ProvisionerMetadata) ProviderRecord() ProviderRecord {
	return metadata.record.Clone()
}

func (metadata ProvisionerMetadata) NameStr() string {
	return metadata.record.Name
}

func (metadata ProvisionerMetadata) TypeStr() string {
	return metadata.record.Type.String()
}

func (metadata ProvisionerMetadata) AuthFields() []string {
	return copyStringSlice(metadata.authFields)
}

// Provisioner is a configured infrastructure provisioner.
type Provisioner interface {
	ProviderRecord() ProviderRecord
	NameStr() string
	TypeStr() string
	AuthFields() []string
	Init() error
}

// ProviderClient is kept as a compatibility alias for existing cloud APIs.
type ProviderClient = Provisioner

// ComputeProvisioner provisions and manages VM instances.
type ComputeProvisioner interface {
	SupportedLocations() []string
	SupportedMachines(location string) (map[string]MachineSpec, error)
	NewInstance(name string, image string, originPublicKey string, machineType string, location string) (id string, err error)
	DeleteInstance(id string, location string) error
	StartInstance(id string, location string) error
	StopInstance(id string, location string) error
	GetInstanceInfo(id string, location string) (InstanceInfo, error)
}

// ComputeProvider is kept as a compatibility alias for existing cloud APIs.
type ComputeProvider = ComputeProvisioner

// InstanceReconciler applies persisted desired instance state to provider resources.
type InstanceReconciler interface {
	ReconcileInstance(InstanceInfo) (InstanceInfo, error)
}

// DeploymentDiagnosticsProvider can add provider-specific details to failed deploys.
type DeploymentDiagnosticsProvider interface {
	DeploymentDiagnostics(id string, location string) (string, error)
}

// ImageProvisioner manages VM images.
type ImageProvisioner interface {
	GetImages() (images map[string]ImageInfo, err error)
	GetProtosImages() (images map[string]ImageInfo, err error)
	AddImage(url string, hash string, version string, location string) (id string, err error)
	UploadLocalImage(imagePath string, imageName string, location string, timeout time.Duration) (id string, err error)
	RemoveImage(name string, location string) error
}

// ImageProvider is kept as a compatibility alias for existing cloud APIs.
type ImageProvider = ImageProvisioner

// VolumeProvisioner manages block volumes.
type VolumeProvisioner interface {
	NewVolume(name string, size int, location string) (id string, err error)
	DeleteVolume(id string, location string) error
	AttachVolume(volumeID string, instanceID string, location string) error
	DettachVolume(volumeID string, instanceID string, location string) error
}

// VolumeProvider is kept as a compatibility alias for existing cloud APIs.
type VolumeProvider = VolumeProvisioner

type ProvisionerDeps struct {
	SecretManager *pcrypto.Manager
	WorkDir       string
}

// ProviderDeps is kept as a compatibility alias for existing cloud APIs.
type ProviderDeps = ProvisionerDeps

type ProvisionerFactory interface {
	Type() Type
	AuthFields() []string
	NewClient(record ProvisionerRecord, deps ProvisionerDeps) (Provisioner, error)
}

// ProviderFactory is kept as a compatibility alias for existing cloud APIs.
type ProviderFactory = ProvisionerFactory

type typedProvisionerFactory[C any] struct {
	provisionerType Type
	authFields      []string
	decodeAuth      func(map[string]string) (C, error)
	newClient       func(ProvisionerMetadata, ProvisionerDeps, C) (Provisioner, error)
}

func NewTypedProvisionerFactory[C any](
	provisionerType Type,
	authFields []string,
	decodeAuth func(map[string]string) (C, error),
	newClient func(ProvisionerMetadata, ProvisionerDeps, C) (Provisioner, error),
) ProvisionerFactory {
	return typedProvisionerFactory[C]{
		provisionerType: provisionerType,
		authFields:      authFields,
		decodeAuth:      decodeAuth,
		newClient:       newClient,
	}
}

func (factory typedProvisionerFactory[C]) Type() Type {
	return factory.provisionerType
}

func (factory typedProvisionerFactory[C]) AuthFields() []string {
	return copyStringSlice(factory.authFields)
}

func (factory typedProvisionerFactory[C]) NewClient(record ProvisionerRecord, deps ProvisionerDeps) (Provisioner, error) {
	record = record.normalized()
	if record.Type == "" {
		record.Type = factory.provisionerType
	}
	if record.Type != factory.provisionerType {
		return nil, fmt.Errorf("provisioner factory for %q cannot create client for %q", factory.provisionerType, record.Type)
	}
	credentials, err := factory.decodeAuth(record.Auth)
	if err != nil {
		return nil, err
	}
	return factory.newClient(newProvisionerMetadata(record, factory.authFields), deps, credentials)
}

type provisionerRegistry struct {
	factories map[Type]ProvisionerFactory
}

func newProvisionerRegistry(factories ...ProvisionerFactory) *provisionerRegistry {
	registry := &provisionerRegistry{factories: map[Type]ProvisionerFactory{}}
	for _, factory := range factories {
		registry.register(factory)
	}
	return registry
}

func (registry *provisionerRegistry) register(factory ProvisionerFactory) {
	if registry == nil || factory == nil {
		return
	}
	registry.factories[factory.Type()] = factory
}

func (registry *provisionerRegistry) factory(provisionerType Type) (ProvisionerFactory, bool) {
	if registry == nil {
		return nil, false
	}
	factory, found := registry.factories[provisionerType]
	return factory, found
}

func (registry *provisionerRegistry) types() []Type {
	if registry == nil {
		return nil
	}
	provisionerTypes := make([]Type, 0, len(registry.factories))
	for provisionerType := range registry.factories {
		provisionerTypes = append(provisionerTypes, provisionerType)
	}
	sort.Slice(provisionerTypes, func(i int, j int) bool {
		return provisionerTypes[i].String() < provisionerTypes[j].String()
	})
	return provisionerTypes
}

func requireComputeProvisioner(provisioner Provisioner) (ComputeProvisioner, error) {
	computeProvisioner, ok := provisioner.(ComputeProvisioner)
	if !ok {
		return nil, unsupportedProvisionerCapability(provisioner, "compute")
	}
	return computeProvisioner, nil
}

func requireImageProvisioner(provisioner Provisioner) (ImageProvisioner, error) {
	imageProvisioner, ok := provisioner.(ImageProvisioner)
	if !ok {
		return nil, unsupportedProvisionerCapability(provisioner, "image")
	}
	return imageProvisioner, nil
}

func requireVolumeProvisioner(provisioner Provisioner) (VolumeProvisioner, error) {
	volumeProvisioner, ok := provisioner.(VolumeProvisioner)
	if !ok {
		return nil, unsupportedProvisionerCapability(provisioner, "volume")
	}
	return volumeProvisioner, nil
}

func unsupportedProvisionerCapability(provisioner Provisioner, capability string) error {
	if provisioner == nil {
		return fmt.Errorf("provisioner is nil and cannot support %s operations", capability)
	}
	return fmt.Errorf("provisioner %q (%s) does not support %s operations", provisioner.NameStr(), provisioner.TypeStr(), capability)
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
