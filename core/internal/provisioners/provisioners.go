package provisioners

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bokwoon95/sq"
	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/p2p"
	"github.com/protosio/protos/internal/p2p/proto"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/release"
	"github.com/protosio/protos/internal/tasks"
	"github.com/protosio/protos/internal/user"
	"github.com/protosio/protos/internal/util"
	"golang.org/x/crypto/ssh"
)

var log = util.GetLogger("provisioners")

// Type represents a specific cloud (AWS, GCP, DigitalOcean etc.)
type Type string

func (ct Type) String() string {
	return string(ct)
}

// CreateManager creates and returns a cloud manager.
//
// Concrete provisioners are passed in by the caller so internal/provisioners owns the
// orchestration contract without importing implementation packages.
func CreateManager(db *db.DB, um *user.Manager, sm *pcrypto.Manager, p2p *p2p.P2P, taskManager *tasks.Manager, provisioners ...ProvisionerFactory) (*Manager, error) {
	if db == nil || um == nil || sm == nil || p2p == nil || taskManager == nil {
		return nil, fmt.Errorf("failed to create cloud manager: none of the inputs can be nil")
	}

	manager := &Manager{
		db:           db,
		um:           um,
		sm:           sm,
		p2p:          p2p,
		provisioners: newProvisionerRegistry(),
		tasks:        taskManager,
		lifecycleSig: map[string]string{},
	}
	for _, provisioner := range provisioners {
		manager.RegisterProvisioner(provisioner)
	}
	if err := manager.registerTaskStreams(); err != nil {
		return nil, err
	}

	return manager, nil
}

// Manager manages provisioners and instances.
type Manager struct {
	db           *db.DB
	um           *user.Manager
	sm           *pcrypto.Manager
	p2p          *p2p.P2P
	provisioners *provisionerRegistry
	tasks        *tasks.Manager
	lifecycleMu  sync.Mutex
	lifecycleSig map[string]string
}

const instanceSSHKeysDir = "instance-ssh-keys"

func instanceSSHKeyPath(instanceID string) (string, error) {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return "", fmt.Errorf("instance id is empty")
	}
	if strings.ContainsAny(instanceID, `/\`) {
		return "", fmt.Errorf("invalid instance id %q", instanceID)
	}
	return path.Join(config.Get().WorkDir, instanceSSHKeysDir, instanceID), nil
}

// GetInstanceSSHKey returns the locally persisted private SSH key used for
// provisioning an instance.
func (cm *Manager) GetInstanceSSHKey(id string) (string, error) {
	instance, err := cm.getInstanceRecord(id)
	if err != nil {
		return "", err
	}
	keyPath, err := instanceSSHKeyPath(instance.ID)
	if err != nil {
		return "", err
	}
	key, err := os.ReadFile(keyPath)
	if err == nil {
		return string(key), nil
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("read SSH key for instance '%s': %w", instance.Name, err)
	}
	localKey, localErr := cm.sm.GetLocalKey()
	if localErr != nil {
		return "", fmt.Errorf("read SSH key for instance '%s': %w; failed to load local SSH key: %w", instance.Name, err, localErr)
	}
	return localKey.EncodePrivateKeytoPEM(), nil
}

func (cm *Manager) deleteInstanceSSHKey(instanceID string) error {
	keyPath, err := instanceSSHKeyPath(instanceID)
	if err != nil {
		return err
	}
	if err := os.Remove(keyPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

//
// Cloud manager methods
//

// RegisterProvisioner adds a concrete provisioner implementation to the manager.
func (cm *Manager) RegisterProvisioner(factory ProvisionerFactory) {
	cm.provisioners.register(factory)
}

// AddProvisioner validates and saves a provisioner configuration.
func (cm *Manager) AddProvisioner(name string, provisionerType string, auth map[string]string) error {
	record := newProvisionerRecord(name, Type(provisionerType), auth)
	provisioner, err := cm.newProvisioner(record)
	if err != nil {
		return fmt.Errorf("failed to create provisioner: %w", err)
	}
	if err := provisioner.Init(); err != nil {
		return fmt.Errorf("failed to initialize provisioner: %w", err)
	}
	if err := cm.saveProviderRecord(provisioner.ProviderRecord()); err != nil {
		return fmt.Errorf("failed to save provisioner: %w", err)
	}
	return nil
}

// AddProvider validates and saves a cloud provider configuration.
func (cm *Manager) AddProvider(cloudName string, cloud string, auth map[string]string) error {
	return cm.AddProvisioner(cloudName, cloud, auth)
}

// SupportedProvisioners returns a list of supported provisioners.
func (cm *Manager) SupportedProvisioners() []string {
	provisionerTypes := cm.provisioners.types()
	supportedProvisioners := make([]string, 0, len(provisionerTypes))
	for _, provisionerType := range provisionerTypes {
		supportedProvisioners = append(supportedProvisioners, provisionerType.String())
	}
	return supportedProvisioners
}

// SupportedProviders returns a list of supported cloud providers.
func (cm *Manager) SupportedProviders() []string {
	return cm.SupportedProvisioners()
}

// ProvisionerAuthFields returns the authentication fields required by a provisioner type.
func (cm *Manager) ProvisionerAuthFields(provisionerType string) ([]string, error) {
	factory, found := cm.provisioners.factory(Type(provisionerType))
	if !found {
		return nil, fmt.Errorf("provisioner '%s' not supported", provisionerType)
	}
	return factory.AuthFields(), nil
}

// ProviderAuthFields returns the authentication fields required by a provider type.
func (cm *Manager) ProviderAuthFields(cloud string) ([]string, error) {
	return cm.ProvisionerAuthFields(cloud)
}

// GetProvisioner returns a provisioner instance from the db.
func (cm *Manager) GetProvisioner(id string) (Provisioner, error) {
	record, found, err := cm.findProviderRecord(id)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve provisioner '%s': %w", id, err)
	}
	if !found {
		return nil, fmt.Errorf("could not find provisioner '%s'", id)
	}
	provisioner, err := cm.newProvisioner(record)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve provisioner '%s': %w", id, err)
	}
	return provisioner, nil
}

// GetProvisionerOrDefault returns a configured provisioner or a transient
// zero-auth built-in provisioner such as local_macos.
func (cm *Manager) GetProvisionerOrDefault(id string) (Provisioner, error) {
	provisioner, err := cm.GetProvisioner(id)
	if err == nil {
		return provisioner, nil
	}
	defaultProvisioner, defaultErr := cm.newDefaultProvisioner(id)
	if defaultErr != nil {
		return nil, err
	}
	return defaultProvisioner, nil
}

// GetProvider returns a cloud provider instance from the db.
func (cm *Manager) GetProvider(id string) (ProviderClient, error) {
	return cm.GetProvisioner(id)
}

func (cm *Manager) newDefaultProvisioner(id string) (Provisioner, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("provisioner id is empty")
	}
	factory, found := cm.provisioners.factory(Type(id))
	if !found {
		return nil, fmt.Errorf("provisioner '%s' not supported", id)
	}
	if len(factory.AuthFields()) > 0 {
		return nil, fmt.Errorf("provisioner '%s' requires credentials", id)
	}
	return cm.newProvisioner(newProvisionerRecord(id, Type(id), nil))
}

func (cm *Manager) ensureProviderForDeployment(id string) (Provisioner, error) {
	provisioner, err := cm.GetProvider(id)
	if err == nil {
		return provisioner, nil
	}
	defaultProvisioner, defaultErr := cm.newDefaultProvisioner(id)
	if defaultErr != nil {
		return nil, err
	}
	if err := defaultProvisioner.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize default provisioner '%s': %w", id, err)
	}
	if err := cm.saveProviderRecord(defaultProvisioner.ProviderRecord()); err != nil {
		return nil, fmt.Errorf("failed to save default provisioner '%s': %w", id, err)
	}
	return defaultProvisioner, nil
}

// DeleteProvisioner deletes a provisioner from the db.
func (cm *Manager) DeleteProvisioner(name string) error {
	record, found, err := cm.findProviderRecord(name)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("could not find provisioner '%s'", name)
	}

	err = db.Delete(cm.db, createCloudProviderDeleteMapper(record.ID))
	if err != nil {
		return fmt.Errorf("failed to delete provisioner '%s': %w", name, err)
	}

	return nil
}

// DeleteProvider deletes a cloud provider from the db.
func (cm *Manager) DeleteProvider(name string) error {
	return cm.DeleteProvisioner(name)
}

// GetProvisioners returns all configured provisioners from the db.
func (cm *Manager) GetProvisioners() ([]Provisioner, error) {
	provisioners := []Provisioner{}
	records, err := db.SelectMultiple(cm.db, createCloudProviderQueryMapper(nil))
	if err != nil {
		return provisioners, fmt.Errorf("failed to retrieve provisioners: %w", err)
	}

	for _, record := range records {
		client, err := cm.newProvisioner(record)
		if err != nil {
			return provisioners, err
		}
		provisioners = append(provisioners, client)
	}

	return provisioners, nil
}

// GetProviders returns all the cloud providers from the db.
func (cm *Manager) GetProviders() ([]ProviderClient, error) {
	return cm.GetProvisioners()
}

func (cm *Manager) provisionerDeps() ProvisionerDeps {
	return ProvisionerDeps{SecretManager: cm.sm, WorkDir: config.Get().WorkDir}
}

func (cm *Manager) newProvisioner(record ProvisionerRecord) (Provisioner, error) {
	record = record.normalized()
	if record.Name == "" {
		return nil, fmt.Errorf("cloud provider name is empty")
	}
	factory, found := cm.provisioners.factory(record.Type)
	if !found {
		return nil, fmt.Errorf("cloud '%s' not supported", record.Type.String())
	}
	return factory.NewClient(record, cm.provisionerDeps())
}

func (cm *Manager) saveProviderRecord(record ProviderRecord) error {
	record = record.normalized()
	if record.Name == "" {
		return fmt.Errorf("cloud provider name is empty")
	}

	cpModel := sq.New[db.CLOUD_PROVIDER]("")
	records, err := db.SelectMultiple(cm.db, createCloudProviderQueryMapper([]sq.Predicate{cpModel.NAME.EqString(record.Name)}))
	if err != nil {
		return err
	}
	if len(records) > 1 {
		return fmt.Errorf("found multiple cloud providers named '%s'", record.Name)
	}
	if len(records) == 1 {
		record.ID = records[0].ID
		return db.Update(cm.db, createCloudProviderUpdateMapper(record))
	}

	if strings.TrimSpace(record.ID) == "" {
		record.ID = db.MustNewUUIDv7()
	} else if _, err := db.UUIDBytes(record.ID); err != nil {
		return fmt.Errorf("cloud provider id must be a UUID: %w", err)
	}
	return db.Insert(cm.db, createCloudProviderInsertMapper(record))
}

func (cm *Manager) findProviderRecord(ref string) (ProviderRecord, bool, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ProviderRecord{}, false, fmt.Errorf("cloud provider name is empty")
	}

	if _, parseErr := db.UUIDBytes(ref); parseErr == nil {
		records, err := cm.findProviderRecordsByID(ref)
		if err != nil {
			return ProviderRecord{}, false, err
		}
		if len(records) == 1 {
			return records[0], true, nil
		}
		if len(records) > 1 {
			return ProviderRecord{}, false, fmt.Errorf("found multiple cloud providers with id '%s'", ref)
		}
	}

	cpModel := sq.New[db.CLOUD_PROVIDER]("")
	records, err := db.SelectMultiple(cm.db, createCloudProviderQueryMapper([]sq.Predicate{cpModel.NAME.EqString(ref)}))
	if err != nil {
		return ProviderRecord{}, false, err
	}
	if len(records) == 0 {
		return ProviderRecord{}, false, nil
	}
	if len(records) > 1 {
		return ProviderRecord{}, false, fmt.Errorf("found multiple cloud providers named '%s'", ref)
	}
	return records[0], true, nil
}

func (cm *Manager) findProviderRecordsByID(id string) ([]ProviderRecord, error) {
	cpModel := sq.New[db.CLOUD_PROVIDER]("")
	return db.SelectMultiple(cm.db, createCloudProviderQueryMapper([]sq.Predicate{db.UUIDEq(cpModel.ID, id)}))
}

//
// Instance related methods
//

// DeployInstance deploys an instance on the provided cloud
func (cm *Manager) DeployInstance(instanceName string, cloudName string, cloudLocation string, release release.Release, machineType string) (result InstanceInfo, err error) {
	provider, err := cm.ensureProviderForDeployment(cloudName)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("could not retrieve cloud '%s': %w", cloudName, err)
	}
	if _, err := requireComputeProvisioner(provider); err != nil {
		return InstanceInfo{}, err
	}
	if _, err := requireImageProvisioner(provider); err != nil {
		return InstanceInfo{}, err
	}
	if _, err := requireVolumeProvisioner(provider); err != nil {
		return InstanceInfo{}, err
	}
	if existing, existingErr := db.SelectOne(cm.db, createInstanceQueryByNameMapper(instanceName)); existingErr == nil && existing.ID != "" {
		return InstanceInfo{}, fmt.Errorf("instance '%s' already exists", instanceName)
	}

	pendingID := newPendingInstanceID()
	instance := InstanceInfo{
		ID:                  pendingID,
		Name:                instanceName,
		Kind:                KindCloudVM,
		KindID:              provider.NameStr(),
		DesiredStatus:       ServerStateRunning,
		ReplicationPriority: db.DefaultReplicationPriorityForMachine(KindCloudVM, provider.NameStr()),
		Location:            cloudLocation,
		Status:              ServerStateChanging,
	}
	mm, cmm := createInstanceInsertMapper(instance)
	if err := db.Insert(cm.db, mm, cmm); err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to save desired instance '%s': %w", instanceName, err)
	}

	task, err := tasks.Enqueue(cm.tasks, tasks.EnqueueOptions[deployInstanceTaskPayload]{
		Stream:      InstanceDeploymentTaskStream,
		SubjectType: taskSubjectInstance,
		SubjectID:   pendingID,
		Title:       fmt.Sprintf("Deploy instance %s", instanceName),
		Message:     "queued",
		Payload: deployInstanceTaskPayload{
			PendingInstanceID: pendingID,
			InstanceName:      instanceName,
			CloudName:         cloudName,
			CloudLocation:     cloudLocation,
			Release:           release,
			MachineType:       machineType,
		},
	})
	if err != nil {
		im, cmmd := createInstanceDeleteMapper(pendingID)
		_ = db.Delete(cm.db, im, cmmd)
		return InstanceInfo{}, fmt.Errorf("failed to queue deployment for instance '%s': %w", instanceName, err)
	}
	instance.Status = fmt.Sprintf("%s: %s", task.Status, task.Message)
	log.Infof("Queued deployment task '%s' for desired instance '%s'", task.ID, instanceName)
	return instance, nil
}

func (cm *Manager) deployInstanceImperative(ctx context.Context, progress func(int, string, any) error, pendingInstanceID string, instanceName string, cloudName string, cloudLocation string, release release.Release, machineType string) (result InstanceInfo, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if progress == nil {
		progress = func(int, string, any) error { return nil }
	}
	// init cloud
	_ = progress(5, "initializing provisioner", nil)
	provider, err := cm.ensureProviderForDeployment(cloudName)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("could not retrieve cloud '%s': %w", cloudName, err)
	}
	computeProvider, err := requireComputeProvisioner(provider)
	if err != nil {
		return InstanceInfo{}, err
	}
	imageProvider, err := requireImageProvisioner(provider)
	if err != nil {
		return InstanceInfo{}, err
	}
	volumeProvider, err := requireVolumeProvisioner(provider)
	if err != nil {
		return InstanceInfo{}, err
	}
	if err := provider.Init(); err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to init cloud provider '%s'(%s) API: %w", cloudName, provider.TypeStr(), err)
	}

	var vmID string
	var volumeID string
	var instanceInfo InstanceInfo
	defer func() {
		if err == nil || vmID == "" {
			return
		}
		log.Warnf("Cleaning up failed instance deployment '%s' (%s): %s", instanceName, vmID, err.Error())
		if instanceInfo.ID != "" {
			_ = cm.p2p.RemovePeer(instanceInfo)
			_ = cm.deleteInstanceSSHKey(instanceInfo.ID)
		}
		if stopErr := computeProvider.StopInstance(vmID, cloudLocation); stopErr != nil {
			log.Debugf("failed to stop partially deployed instance '%s' (%s): %s", instanceName, vmID, stopErr.Error())
		}
		if volumeID != "" {
			if deleteErr := volumeProvider.DeleteVolume(volumeID, cloudLocation); deleteErr != nil {
				log.Warnf("failed to delete partially deployed instance volume '%s' for '%s': %s", volumeID, instanceName, deleteErr.Error())
			}
		}
		if deleteErr := computeProvider.DeleteInstance(vmID, cloudLocation); deleteErr != nil {
			log.Warnf("failed to delete partially deployed instance '%s' (%s): %s", instanceName, vmID, deleteErr.Error())
		}
	}()

	// validate machine type
	_ = progress(10, "validating machine type", nil)
	supportedMachineTypes, err := computeProvider.SupportedMachines(cloudLocation)
	if err != nil {
		return InstanceInfo{}, err
	}
	if _, found := supportedMachineTypes[machineType]; !found {
		return InstanceInfo{}, errors.Errorf("Machine type '%s' is not valid for cloud provider '%s'. The following types are supported: \n%s", machineType, provider.TypeStr(), createMachineTypesString(supportedMachineTypes))
	}

	// add image
	_ = progress(20, "resolving image", nil)
	imageID := ""
	images, err := imageProvider.GetImages()
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to deploy Protos instance: %w", err)
	}
	if selectedID, selectedImage, found := SelectProtosImageForRef(images, cloudLocation, release.Version); found {
		imageID = selectedID
		log.Infof("selected Protos image '%s' (%s) for version '%s'", selectedImage.Name, selectedID, release.Version)
	}
	if imageID != "" {
		log.Infof("found Protos image version '%s' in your cloud account", release.Version)
	} else {
		// upload protos image
		if image, found := release.CloudImages[provider.TypeStr()]; found {
			log.Infof("Protos image version '%s' not in your infra cloud account. Adding it.", release.Version)
			_ = progress(25, "uploading image", map[string]string{"version": release.Version})
			imageID, err = imageProvider.AddImage(image.URL, image.Digest, release.Version, cloudLocation)
			if err != nil {
				return InstanceInfo{}, fmt.Errorf("failed to deploy Protos instance: %w", err)
			}
		} else {
			return InstanceInfo{}, errors.Errorf("could not find a Protos version '%s' release for cloud '%s'", release.Version, provider.TypeStr())
		}
	}

	thisDevice, err := cm.um.GetCurrentDevice()
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to get current device : %w", err)
	}

	// deploy a protos instance
	log.Infof("Deploying instance '%s' of type '%s', using Protos version '%s' (image id '%s')", instanceName, machineType, release.Version, imageID)
	_ = progress(35, "creating VM", map[string]string{"image_id": imageID})
	vmID, err = computeProvider.NewInstance(instanceName, imageID, thisDevice.GetPublicKey(), machineType, cloudLocation)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to deploy Protos instance: %w", err)
	}
	log.Infof("Instance with ID '%s' deployed", vmID)

	// get instance info
	instanceInfo, err = computeProvider.GetInstanceInfo(vmID, cloudLocation)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to get Protos instance info: %w", err)
	}
	if instanceInfo.ProviderResourceID == "" {
		instanceInfo.ProviderResourceID = vmID
	}
	if instanceInfo.Kind == "" {
		instanceInfo.Kind = KindCloudVM
	}
	if instanceInfo.KindID == "" {
		instanceInfo.KindID = provider.NameStr()
	}
	instanceInfo.ReplicationPriority = db.DefaultReplicationPriorityForMachine(instanceInfo.Kind, instanceInfo.KindID)
	if pendingInstanceID != "" {
		pendingUpdate := instanceInfo
		pendingUpdate.ID = pendingInstanceID
		pendingUpdate.PublicKey = ""
		pendingUpdate.Architecture = ""
		pendingUpdate.DesiredStatus = ServerStateRunning
		pendingUpdate.ReplicationPriority = db.DefaultReplicationPriorityForMachine(pendingUpdate.Kind, pendingUpdate.KindID)
		if updateErr := cm.updateDeploymentPlaceholder(pendingUpdate); updateErr != nil {
			return InstanceInfo{}, updateErr
		}
	}

	// create protos data volume
	log.Infof("creating data volume for Protos instance '%s'", instanceName)
	_ = progress(45, "creating data volume", nil)
	volumeID, err = volumeProvider.NewVolume(instanceName, 30000, cloudLocation)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to create data volume: %w", err)
	}

	// attach volume to instance
	_ = progress(55, "attaching data volume", map[string]string{"volume_id": volumeID})
	err = volumeProvider.AttachVolume(volumeID, vmID, cloudLocation)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to attach volume to instance '%s': %w", instanceName, err)
	}

	// start protos instance
	log.Infof("Starting instance '%s'", instanceName)
	_ = progress(65, "starting VM", nil)
	err = computeProvider.StartInstance(vmID, cloudLocation)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to start instance: %w", err)
	}

	// get instance info again
	instanceUpdate, err := computeProvider.GetInstanceInfo(vmID, cloudLocation)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to get instance info: %w", err)
	}
	instanceInfo.PublicIP = instanceUpdate.PublicIP
	instanceInfo.Volumes = instanceUpdate.Volumes
	if instanceUpdate.Status != "" {
		instanceInfo.Status = instanceUpdate.Status
	}
	if pendingInstanceID != "" {
		pendingUpdate := instanceInfo
		pendingUpdate.ID = pendingInstanceID
		pendingUpdate.PublicKey = ""
		pendingUpdate.Architecture = ""
		pendingUpdate.DesiredStatus = ServerStateRunning
		if updateErr := cm.updateDeploymentPlaceholder(pendingUpdate); updateErr != nil {
			return InstanceInfo{}, updateErr
		}
	}

	_ = progress(75, "discovering VM identity", map[string]string{"public_ip": instanceInfo.PublicIP})
	discoverCtx, discoverCancel := context.WithTimeout(ctx, 5*time.Minute)
	discoveredPeer, err := cm.p2p.DiscoverPeer(discoverCtx, instanceInfo.PublicIP)
	discoverCancel()
	if err != nil {
		return InstanceInfo{}, deploymentFailureError(computeProvider, vmID, cloudLocation, fmt.Errorf("failed to discover instance peer over libp2p: %w", err))
	}
	instanceInfo.PublicKey = discoveredPeer.PublicKey
	if pendingInstanceID != "" {
		instanceInfo.ID = pendingInstanceID
	} else if _, parseErr := db.UUIDBytes(instanceInfo.ID); parseErr != nil {
		instanceInfo.ID = db.MustNewUUIDv7()
	}
	log.Infof("Discovered instance peer '%s' at '%s' with fingerprint '%s'", discoveredPeer.ID, discoveredPeer.Address, discoveredPeer.Fingerprint)

	_ = progress(85, "initializing VM", map[string]string{"peer_id": discoveredPeer.ID})
	p2pClient, err := cm.p2p.AddPeer(instanceInfo)
	if err != nil {
		_ = cm.p2p.RemovePeer(instanceInfo)
		return InstanceInfo{}, fmt.Errorf("failed to initialize instance: %w", err)
	}
	if p2pClient == nil {
		_ = cm.p2p.RemovePeer(instanceInfo)
		return InstanceInfo{}, errors.New("failed to initialize instance: p2p client is nil")
	}

	originSwarmionAddrs := cm.originSwarmionBootstrapAddrs(thisDevice.GetPublicKey(), instanceInfo.PublicIP)

	// do the initialization
	log.Infof("Initializing instance '%s'", instanceName)
	resp, err := p2pClient.Init(ctx, &proto.InitRequest{
		OriginDevice:          thisDevice.GetName(),
		OriginDevicePublicKey: thisDevice.GetPublicKey(),
		OriginSwarmionAddrs:   originSwarmionAddrs,
		InstanceName:          instanceName,
	})
	if err != nil {
		_ = cm.p2p.RemovePeer(instanceInfo)
		return InstanceInfo{}, fmt.Errorf("failed to initialize instance: %w", err)
	}

	// removing peer after initialization is done, so that the target peer has time to re-create the grpc server
	err = cm.p2p.RemovePeer(instanceInfo)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to remove peer: %w", err)
	}

	instanceUpdate, err = computeProvider.GetInstanceInfo(vmID, cloudLocation)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to get instance info: %w", err)
	}

	// final save instance info
	instanceInfo.Architecture = resp.Architecture
	instanceInfo.Status = instanceUpdate.Status
	instanceInfo.DesiredStatus = ServerStateRunning
	instanceInfo.ReplicationPriority = db.DefaultReplicationPriorityForMachine(instanceInfo.Kind, instanceInfo.KindID)

	_ = progress(95, "saving VM identity", map[string]string{"peer_id": discoveredPeer.ID})
	if pendingInstanceID != "" {
		if err := cm.completeDeploymentInstance(pendingInstanceID, instanceInfo); err != nil {
			return InstanceInfo{}, fmt.Errorf("failed to save instance '%s': %w", instanceName, err)
		}
	}

	log.Infof("Instance '%s' at '%s' is ready", instanceName, instanceInfo.PublicIP)

	return instanceInfo, nil
}

func deploymentFailureError(provider ComputeProvisioner, id string, location string, cause error) error {
	diagnosticsProvider, ok := provider.(DeploymentDiagnosticsProvider)
	if !ok {
		return cause
	}
	diagnostics, err := diagnosticsProvider.DeploymentDiagnostics(id, location)
	if err != nil {
		log.Debugf("failed to collect deployment diagnostics for '%s': %s", id, err.Error())
		return cause
	}
	diagnostics = strings.TrimSpace(diagnostics)
	if diagnostics == "" {
		return cause
	}
	return fmt.Errorf("%w\n\nVM diagnostics:\n%s", cause, diagnostics)
}

func (cm *Manager) InitInstance(instanceName string, kind string, kindID string, locationName string, ipString string) (err error) {
	instanceInfo := InstanceInfo{
		ID:                  db.MustNewUUIDv7(),
		PublicIP:            ipString,
		Name:                instanceName,
		Kind:                kind,
		KindID:              kindID,
		DesiredStatus:       ServerStateRunning,
		ReplicationPriority: db.DefaultReplicationPriorityForMachine(kind, kindID),
		Location:            locationName,
	}

	ip := net.ParseIP(ipString)
	if ip == nil {
		return fmt.Errorf("String '%s' is not a valid IP address", ipString)
	}

	thisDevice, err := cm.um.GetCurrentDevice()
	if err != nil {
		return fmt.Errorf("failed to get current device : %w", err)
	}

	discoverCtx, discoverCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	discoveredPeer, err := cm.p2p.DiscoverPeer(discoverCtx, instanceInfo.PublicIP)
	discoverCancel()
	if err != nil {
		return fmt.Errorf("failed to discover instance peer over libp2p: %w", err)
	}
	instanceInfo.PublicKey = discoveredPeer.PublicKey
	log.Infof("Discovered instance peer '%s' at '%s' with fingerprint '%s'", discoveredPeer.ID, discoveredPeer.Address, discoveredPeer.Fingerprint)

	machineMapper, machineMetadataMapper := createInstanceInsertMapper(instanceInfo)
	insertedInstance := false
	if err := db.Insert(cm.db, machineMapper, machineMetadataMapper, db.CreatePeerInsertMapper(instanceInfo.PublicKey)); err != nil {
		return fmt.Errorf("failed to save instance '%s': %w", instanceName, err)
	}
	insertedInstance = true
	defer func() {
		if err == nil || !insertedInstance {
			return
		}
		im, cmmd := createInstanceDeleteMapper(instanceInfo.ID)
		_ = db.Delete(cm.db, db.CreatePeerDeleteMapper(instanceInfo.PublicKey), im, cmmd)
	}()

	p2pClient, err := cm.p2p.AddPeer(instanceInfo)
	if err != nil {
		return fmt.Errorf("failed to initialize instance: %w", err)
	}
	if p2pClient == nil {
		return errors.New("failed to initialize instance: p2p client is nil")
	}

	originSwarmionAddrs := cm.originSwarmionBootstrapAddrs(thisDevice.GetPublicKey(), instanceInfo.PublicIP)

	// do the initialization
	log.Infof("Initializing instance '%s'", instanceName)
	resp, err := p2pClient.Init(context.TODO(), &proto.InitRequest{
		OriginDevice:          thisDevice.GetName(),
		OriginDevicePublicKey: thisDevice.GetPublicKey(),
		OriginSwarmionAddrs:   originSwarmionAddrs,
		InstanceName:          instanceName,
	})
	if err != nil {
		return fmt.Errorf("failed to init instance: %w", err)
	}

	instanceInfo.Architecture = resp.Architecture

	// removing peer after initialization is done, so that the target peer has time to re-create the grpc server
	err = cm.p2p.RemovePeer(instanceInfo)
	if err != nil {
		return fmt.Errorf("failed to remove peer: %w", err)
	}

	log.Infof("Instance '%s'(%s) initialized", instanceName, ipString)

	return nil
}

func (cm *Manager) originSwarmionBootstrapAddrs(originPublicKey string, peerPublicIP string) []string {
	ips := originBootstrapIPs(originPublicKey, peerPublicIP)
	addrs := cm.db.DialableListenMultiaddrs(ips)
	if len(addrs) == 0 {
		return cm.db.ListenMultiaddrs()
	}
	return addrs
}

func originBootstrapIPs(originPublicKey string, peerPublicIP string) []string {
	var ips []string
	if key, err := pcrypto.CreatePublicKeyFromBase64(originPublicKey); err == nil {
		ips = append(ips, key.IPv6Address().String())
	}
	if gateway := localMacOSNATGateway(peerPublicIP); gateway != "" {
		ips = append(ips, gateway)
	}
	return dedupeIPs(ips)
}

func localMacOSNATGateway(peerPublicIP string) string {
	ip := net.ParseIP(strings.TrimSpace(peerPublicIP))
	if ip == nil {
		return ""
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return ""
	}
	if ip4[0] != 192 || ip4[1] != 168 || ip4[2] != 64 {
		return ""
	}
	return net.IPv4(ip4[0], ip4[1], ip4[2], 1).String()
}

func dedupeIPs(ips []string) []string {
	seen := map[string]struct{}{}
	deduped := make([]string, 0, len(ips))
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			continue
		}
		if _, found := seen[ip]; found {
			continue
		}
		seen[ip] = struct{}{}
		deduped = append(deduped, ip)
	}
	return deduped
}

func getProviderInstanceInfo(provider ComputeProvisioner, instance InstanceInfo) (InstanceInfo, string, error) {
	refs := []string{instance.ProviderResourceID, instance.ID}
	bestRef := firstNonEmptyString(refs...)
	var firstErr error
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		info, err := provider.GetInstanceInfo(ref, instance.Location)
		if err == nil {
			if info.ProviderResourceID == "" {
				info.ProviderResourceID = ref
			}
			return info, ref, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	if strings.TrimSpace(instance.Name) == "" || instance.Name == instance.ID {
		return InstanceInfo{}, bestRef, firstErr
	}
	byName, nameErr := provider.GetInstanceInfo(instance.Name, instance.Location)
	if nameErr == nil {
		providerID := byName.ProviderResourceID
		if providerID == "" {
			providerID = byName.ID
		}
		byName.ProviderResourceID = providerID
		return byName, providerID, nil
	}
	if firstErr != nil {
		return InstanceInfo{}, bestRef, firstErr
	}
	return InstanceInfo{}, bestRef, nameErr
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

// UpdateInstance updates an instance
func (cm *Manager) UpdateInstance(id string, ip string) error {
	instance, err := cm.getInstanceRecord(id)
	if err != nil {
		return fmt.Errorf("could not retrieve instance '%s': %w", id, err)
	}

	instance.PublicIP = ip
	im, cmm := createInstanceUpdateMapper(instance)
	err = db.Update(cm.db, im, cmm)
	if err != nil {
		return fmt.Errorf("failed to save instance '%s': %w", id, err)
	}

	return nil

}

// DeleteInstance queues deletion of an instance.
func (cm *Manager) DeleteInstance(ctx context.Context, id string) (tasks.Record, error) {
	return cm.QueueDeleteInstance(ctx, id)
}

// DeleteInstanceLocal deletes only local database and peer state for an instance.
func (cm *Manager) DeleteInstanceLocal(ctx context.Context, id string) (tasks.Record, error) {
	return cm.QueueDeleteInstanceLocal(ctx, id)
}

func (cm *Manager) deleteInstanceImperative(ctx context.Context, progress func(int, string, any) error, id string, localOnly bool) error {
	ctx, cancel := instanceDeleteContext(ctx)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return err
	}
	if progress == nil {
		progress = func(int, string, any) error { return nil }
	}

	instance, err := cm.getInstanceRecord(id)
	if err != nil {
		return fmt.Errorf("could not retrieve instance '%s': %w", id, err)
	}
	if strings.TrimSpace(instance.PublicKey) == "" && cm.tasks != nil {
		if task, found, taskErr := cm.tasks.LatestForSubject(InstanceDeploymentTaskStream, taskSubjectInstance, instance.ID); taskErr == nil && found {
			if cancelErr := cm.tasks.Cancel(task.ID, "instance removed"); cancelErr != nil {
				log.Warnf("failed to cancel deployment task for instance '%s': %s", instance.Name, cancelErr.Error())
			}
		}
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	if err := progress(10, "marking instance deleting", map[string]string{"instance_id": instance.ID, "instance_name": instance.Name}); err != nil {
		return err
	}
	if err := cm.markInstanceDeleting(ctx, instance); err != nil {
		return err
	}
	instance.DesiredStatus = ServerStateDeleting

	if err := progress(20, "removing instance apps", map[string]string{"instance_id": instance.ID}); err != nil {
		return err
	}
	if err := cm.deleteAppsForInstance(ctx, instance.ID); err != nil {
		return err
	}

	if strings.TrimSpace(instance.PublicKey) != "" {
		if cm.p2p != nil {
			if err := progress(30, "removing p2p peer", map[string]string{"instance_id": instance.ID}); err != nil {
				return err
			}
			err = cm.p2p.RemovePeer(instance)
			if err != nil {
				return fmt.Errorf("failed to remove peer: %w", err)
			}
		}
		if err := progress(40, "waiting for durable peer removal", map[string]string{"instance_id": instance.ID}); err != nil {
			return err
		}
		if err := cm.waitForInstancePeerDurableRemovalReady(ctx, instance); err != nil {
			return err
		}
	}

	if !localOnly && (instance.Kind == KindCloudVM || instance.Kind == KindLocalVM) {
		if err := ctx.Err(); err != nil {
			return err
		}
		provider, err := cm.GetProvider(instance.KindID)
		if err != nil {
			return fmt.Errorf("could not retrieve cloud '%s': %w", id, err)
		}

		computeProvider, err := requireComputeProvisioner(provider)
		if err != nil {
			return err
		}
		volumeProvider, err := requireVolumeProvisioner(provider)
		if err != nil {
			return err
		}
		if err := provider.Init(); err != nil {
			return fmt.Errorf("could not init cloud '%s': %w", id, err)
		}

		if err := progress(50, "loading provider instance", map[string]string{"instance_id": instance.ID, "provisioner": instance.KindID}); err != nil {
			return err
		}
		found := true
		vmInfo, providerInstanceID, err := getProviderInstanceInfo(computeProvider, instance)
		if err != nil {
			if instance.Kind == KindLocalVM && errors.Is(err, os.ErrNotExist) {
				vmInfo = InstanceInfo{ProviderResourceID: providerInstanceID}
			} else if strings.Contains(strings.ToLower(err.Error()), "not found") {
				found = false
			} else {
				return fmt.Errorf("failed to get details for instance '%s': %w", id, err)
			}
		}

		// only delete cloud instance if found. Otherwise we proceed with removing it from local db
		if found {
			if vmInfo.Status == ServerStateRunning {
				log.Infof("Stopping instance '%s' (%s)", instance.Name, providerInstanceID)
				if err := ctx.Err(); err != nil {
					return err
				}
				if err := progress(60, "stopping provider instance", map[string]string{"instance_id": instance.ID, "provider_instance_id": providerInstanceID}); err != nil {
					return err
				}
				err = computeProvider.StopInstance(providerInstanceID, instance.Location)
				if err != nil {
					log.Warnf("failed to stop instance '%s' before delete; attempting provider delete anyway: %s", id, err.Error())
				}
			}
			for _, vol := range vmInfo.Volumes {
				log.Infof("Deleting volume '%s' (%s) for instance '%s'", vol.Name, vol.VolumeID, id)
				if err := ctx.Err(); err != nil {
					return err
				}
				if err := progress(70, "deleting provider volume", map[string]string{"instance_id": instance.ID, "volume_id": vol.VolumeID}); err != nil {
					return err
				}
				err = volumeProvider.DeleteVolume(vol.VolumeID, instance.Location)
				if err != nil {
					return fmt.Errorf("could not delete volume '%s' for instance '%s': %w", vol.VolumeID, id, err)
				}
			}
			log.Infof("Deleting instance '%s' (%s)", instance.Name, providerInstanceID)
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := progress(80, "deleting provider instance", map[string]string{"instance_id": instance.ID, "provider_instance_id": providerInstanceID}); err != nil {
				return err
			}
			err = computeProvider.DeleteInstance(providerInstanceID, instance.Location)
			if err != nil {
				return fmt.Errorf("could not delete instance '%s': %w", id, err)
			}
		}
	}

	if err := progress(90, "deleting instance records", map[string]string{"instance_id": instance.ID}); err != nil {
		return err
	}
	if err := cm.deleteInstanceRecords(ctx, instance); err != nil {
		return fmt.Errorf("failed to delete instance '%s': %w", id, err)
	}
	if err := cm.deleteInstanceSSHKey(instance.ID); err != nil {
		log.Warnf("failed to delete SSH key for instance '%s': %s", instance.Name, err.Error())
	}

	return nil
}

func instanceDeleteContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, 10*time.Minute)
}

func (cm *Manager) deleteInstanceRecords(ctx context.Context, instance InstanceInfo) error {
	if cm == nil || cm.db == nil {
		return fmt.Errorf("provisioner manager database is not configured")
	}
	if strings.TrimSpace(instance.ID) == "" {
		return fmt.Errorf("instance ID is empty")
	}

	var lastErr error
	for attempt := 1; ; attempt++ {
		if err := ctx.Err(); err != nil {
			if lastErr != nil {
				return fmt.Errorf("%w: %w", err, lastErr)
			}
			return err
		}
		im, cmmd := createInstanceDeleteMapper(instance.ID)
		err := db.Delete(cm.db, db.CreatePeerDeleteMapper(instance.PublicKey), createAppDeleteByInstanceMapper(instance.ID), im, cmmd)
		peerID := ""
		if strings.TrimSpace(instance.PublicKey) != "" {
			peerID, _ = db.PeerIDFromPublicKeyString(instance.PublicKey)
		}
		if err == nil {
			err = cm.assertInstancePeerRemoved(ctx, peerID)
		}
		if err == nil {
			if verifyErr := cm.waitForInstanceDeleteCheckpoint(ctx, peerID); verifyErr == nil {
				return nil
			} else {
				err = verifyErr
			}
		}
		lastErr = err
		sleep := time.Duration(attempt*2) * time.Second
		if sleep > 15*time.Second {
			sleep = 15 * time.Second
		}
		select {
		case <-ctx.Done():
		case <-time.After(sleep):
		}
	}
}

func (cm *Manager) waitForInstanceDeleteCheckpoint(ctx context.Context, peerID string) error {
	var lastErr error
	for {
		if err := ctx.Err(); err != nil {
			if lastErr != nil {
				return fmt.Errorf("%w: %w", err, lastErr)
			}
			return err
		}
		attemptCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		catchUpErr := cm.db.CatchUpCheckpointStrict(attemptCtx, "verify instance delete checkpoint")
		cancel()
		if catchUpErr != nil {
			lastErr = catchUpErr
			if !db.IsRetryableCheckpointCatchUp(catchUpErr) {
				return catchUpErr
			}
			select {
			case <-ctx.Done():
			case <-time.After(5 * time.Second):
			}
			continue
		}

		status, ok := cm.db.SwarmionStatus()
		if !ok {
			if err := cm.assertInstancePeerRemoved(ctx, peerID); err != nil {
				return err
			}
			return nil
		}
		checkpoint := status.CheckpointRootHash.String()
		durable := status.DurableMainRootHash.String()
		if checkpoint == "" ||
			status.TentativeRootHash.String() != checkpoint ||
			(durable != "" && durable != checkpoint) {
			select {
			case <-ctx.Done():
			case <-time.After(5 * time.Second):
			}
			continue
		}
		if err := cm.assertInstancePeerRemoved(ctx, peerID); err != nil {
			return err
		}
		return nil
	}
}

func (cm *Manager) markInstanceDeleting(ctx context.Context, instance InstanceInfo) error {
	if cm == nil || cm.db == nil || strings.TrimSpace(instance.ID) == "" || IsDeletingInstance(instance) {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	instance.DesiredStatus = ServerStateDeleting
	im, _ := createInstanceUpdateMapper(instance)
	if err := db.Update(cm.db, im); err != nil {
		return fmt.Errorf("failed to mark instance '%s' as deleting: %w", instance.Name, err)
	}
	return nil
}

func (cm *Manager) deleteAppsForInstance(ctx context.Context, instanceID string) error {
	instanceID = strings.TrimSpace(instanceID)
	if cm == nil || cm.db == nil || instanceID == "" {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := db.Delete(cm.db, createAppDeleteByInstanceMapper(instanceID)); err != nil {
		return fmt.Errorf("failed to delete apps for instance '%s': %w", instanceID, err)
	}
	return nil
}

func (cm *Manager) assertInstancePeerRemoved(ctx context.Context, peerID string) error {
	peerID = strings.TrimSpace(peerID)
	if cm == nil || cm.db == nil || peerID == "" {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	peerIDs, err := db.GetPeerIDs(cm.db)
	if err != nil {
		return err
	}
	if _, found := peerIDs[peerID]; found {
		return fmt.Errorf("peer table still contains %s", peerID)
	}
	if _, ok := cm.db.SwarmionStatus(); !ok {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	readiness, err := cm.db.SwarmionPeerRemovalReadiness(ctx, peerID)
	if err != nil {
		return fmt.Errorf("read swarmion peer removal readiness: %w", err)
	}
	log.Debugf("swarmion peer removal readiness after local row removal for %s: %s", peerID, db.PeerRemovalReadinessSummary(readiness))
	if err := db.PeerRemovalReadinessError(readiness); err != nil {
		return fmt.Errorf("swarmion peer removal readiness blocks %s: %w", peerID, err)
	}
	return nil
}

func (cm *Manager) waitForInstancePeerDurableRemovalReady(ctx context.Context, instance InstanceInfo) error {
	if cm == nil || cm.db == nil {
		return nil
	}
	peerID, err := instance.GetPeerID()
	if err != nil {
		return fmt.Errorf("derive peer id for instance '%s': %w", instance.Name, err)
	}
	candidates, err := cm.replicationCandidatesExcluding(peerID)
	if err != nil {
		return fmt.Errorf("build remaining replication candidates for instance '%s': %w", instance.Name, err)
	}
	if err := cm.db.RemoveReplicationPeerState(ctx, peerID, candidates); err != nil {
		return fmt.Errorf("wait for swarmion peer removal readiness for instance '%s': %w", instance.Name, err)
	}
	log.Debugf("swarmion peer %s is ready for provider resource cleanup for instance '%s'", peerID, instance.Name)
	return nil
}

func (cm *Manager) replicationCandidatesExcluding(peerID string) ([]db.ReplicationCandidate, error) {
	peerID = strings.TrimSpace(peerID)
	instances, err := cm.GetInstances(false)
	if err != nil {
		return nil, err
	}
	candidates := make([]db.ReplicationCandidate, 0, len(instances)+1)
	for _, instance := range instances {
		if strings.TrimSpace(instance.ID) == "" || strings.TrimSpace(instance.PublicKey) == "" || !IsActiveInstance(instance) {
			continue
		}
		instancePeerID, err := instance.GetPeerID()
		if err != nil || instancePeerID == peerID {
			continue
		}
		candidates = append(candidates, db.ReplicationCandidate{
			PeerID:      instancePeerID,
			DeviceClass: db.ReplicationDeviceClassForMachine(instance.Kind, instance.KindID),
			Priority:    instance.ReplicationPriority,
		})
	}
	if cm.um == nil {
		return candidates, nil
	}
	devices, err := cm.um.GetAllDevices(false)
	if err != nil {
		return nil, err
	}
	for _, device := range devices {
		if strings.TrimSpace(device.ID) == "" || strings.TrimSpace(device.PublicKey) == "" {
			continue
		}
		devicePeerID, err := db.PeerIDFromPublicKeyString(device.PublicKey)
		if err != nil || devicePeerID == peerID {
			continue
		}
		candidates = append(candidates, db.ReplicationCandidate{
			PeerID:      devicePeerID,
			DeviceClass: db.ReplicationDeviceClassForUserDeviceName(device.Name),
			Priority:    device.ReplicationPriority,
		})
	}
	return candidates, nil
}

// StartInstance starts an instance
func (cm *Manager) StartInstance(id string) (tasks.Record, error) {
	return cm.QueueStartInstance(id)
}

// StopInstance stops an instance
func (cm *Manager) StopInstance(id string) (tasks.Record, error) {
	return cm.QueueStopInstance(id)
}

// TunnelInstance creates and SSH tunnel to the instance
func (cm *Manager) TunnelInstance(id string) error {
	instanceInfo, err := cm.GetInstance(id)
	if err != nil {
		return fmt.Errorf("could not retrieve instance '%s': %w", id, err)
	}

	localKey, err := cm.sm.GetLocalKey()
	if err != nil {
		return err
	}

	log.Infof("creating SSH tunnel to instance '%s', using ip '%s'", instanceInfo.Name, instanceInfo.PublicIP)
	tunnel := pcrypto.NewTunnel(instanceInfo.PublicIP+":22", "root", localKey.SSHAuth(), "localhost:8080")
	localPort, err := tunnel.Start()
	if err != nil {
		return fmt.Errorf("error while creating the SSH tunnel: %w", err)
	}

	quit := make(chan interface{}, 1)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go catchSignals(sigs, quit)

	log.Infof("SSH tunnel ready. Use 'http://localhost:%d/' to access the instance dashboard. Once finished, press CTRL+C to terminate the SSH tunnel", localPort)

	// waiting for a SIGTERM or SIGINT
	<-quit

	log.Info("CTRL+C received. Terminating the SSH tunnel")
	err = tunnel.Close()
	if err != nil {
		return fmt.Errorf("error while terminating the SSH tunnel: %w", err)
	}
	log.Info("SSH tunnel terminated successfully")
	return nil
}

// LogsRemoteInstance retrieves the Protos logs from an instance, via SSH
func (cm *Manager) LogsRemoteInstance(id string) (string, error) {
	instanceInfo, err := cm.GetInstance(id)
	if err != nil {
		return "", err
	}

	auth, err := cm.sshAuthForInstance(instanceInfo)
	if err != nil {
		return "", err
	}

	sshCon, err := pcrypto.NewConnection(instanceInfo.PublicIP, "root", auth, 10)
	if err != nil {
		return "", err
	}
	defer sshCon.Close()

	output, err := pcrypto.ExecuteCommand("cat /var/log/protos.log", sshCon)
	if err != nil {
		return "", err
	}
	return output, nil
}

func (cm *Manager) sshAuthForInstance(instance InstanceInfo) (ssh.AuthMethod, error) {
	keyPath, err := instanceSSHKeyPath(instance.ID)
	if err != nil {
		return nil, err
	}
	auth, err := cm.sm.NewAuthFromKeyFile(keyPath)
	if err == nil {
		return auth, nil
	}
	if !os.IsNotExist(err) {
		log.Warnf("failed to load SSH key for instance '%s': %s", instance.Name, err.Error())
	}

	localKey, localErr := cm.sm.GetLocalKey()
	if localErr != nil {
		return nil, fmt.Errorf("failed to load instance SSH key: %w; failed to load local SSH key: %w", err, localErr)
	}
	return localKey.SSHAuth(), nil
}

// GetInstance retrieves an instance from the db and returns it
func (cm *Manager) GetInstance(id string) (InstanceInfo, error) {

	instance, err := cm.getInstanceRecord(id)
	if err != nil {
		return instance, err
	}

	if IsDeletingInstance(instance) {
		instance.Status = ServerStateDeleting
		return instance, nil
	}

	if strings.TrimSpace(instance.PublicKey) == "" {
		instance.Status = cm.pendingInstanceStatus(instance)
		return instance, nil
	}

	status, err := cm.retrieveInstanceStatus(instance)
	if err != nil {
		log.Errorf("failed to retrieve status from instance '%s': %s", instance.Name, err.Error())
		instance.Status = "n/a"
	} else {
		instance.Status = status
	}
	return instance, nil
}

func (cm *Manager) getInstanceRecord(id string) (InstanceInfo, error) {
	iqm := createInstanceQueryMapper(id)
	instance, err := db.SelectOne(cm.db, iqm)
	if err == nil {
		return instance, nil
	}
	instanceByName, nameErr := db.SelectOne(cm.db, createInstanceQueryByNameMapper(id))
	if nameErr != nil {
		return instance, fmt.Errorf("failed to retrieve instance: %w", err)
	}
	return instanceByName, nil
}

// GetInstances returns all the instances from the db
func (cm *Manager) GetInstances(excludeLocalInstance bool) ([]InstanceInfo, error) {

	publicKey := ""
	if excludeLocalInstance {
		key, err := cm.sm.GetLocalKey()
		if err != nil {
			return nil, fmt.Errorf("could not retrieve local key: %w", err)
		}
		publicKey = key.PublicString()
	}

	instances, err := db.SelectMultiple(cm.db, createInstanceQueryAllMapper(publicKey))
	if err != nil {
		return instances, fmt.Errorf("failed to retrieve instances: %w", err)
	}

	return instances, nil
}

func (cm *Manager) retrieveInstanceStatus(instance InstanceInfo) (string, error) {
	provider, err := cm.GetProvider(instance.KindID)
	if err != nil && instance.Kind == KindLocalVM {
		provider, err = cm.GetProvisionerOrDefault(instance.KindID)
	}
	if err != nil {
		return "", fmt.Errorf("failed to retrieve status for instance '%s': %s", instance.Name, err.Error())
	}
	computeProvider, err := requireComputeProvisioner(provider)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve status for instance '%s': %s", instance.Name, err.Error())
	}
	if err := provider.Init(); err != nil {
		return "", fmt.Errorf("failed to retrieve status for instance '%s': %s", instance.Name, err.Error())
	}
	instanceInfo, _, err := getProviderInstanceInfo(computeProvider, instance)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve status for instance '%s': %s", instance.Name, err.Error())

	}

	return instanceInfo.Status, nil
}

// GetInstances returns all the instances from the db
func (cm *Manager) GetInstancesWithUpdatedStatus() ([]InstanceInfo, error) {
	instances, err := db.SelectMultiple(cm.db, createInstanceQueryAllMapper(""))
	if err != nil {
		return instances, fmt.Errorf("failed to retrieve instances: %w", err)
	}

	for i, instance := range instances {
		if IsDeletingInstance(instance) {
			instances[i].Status = ServerStateDeleting
			continue
		}
		if strings.TrimSpace(instance.PublicKey) == "" {
			instances[i].Status = cm.pendingInstanceStatus(instance)
			continue
		}
		status, err := cm.retrieveInstanceStatus(instance)
		if err != nil {
			log.Warn(err.Error())
			instances[i].Status = "n/a"
		} else {
			instances[i].Status = status
		}
	}
	return instances, nil
}

func (cm *Manager) reconcileDesiredInstance(ctx context.Context, progress func(int, string, any) error, id string) (bool, InstanceInfo, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if progress == nil {
		progress = func(int, string, any) error { return nil }
	}
	instance, err := cm.getInstanceRecord(id)
	if err != nil {
		return false, InstanceInfo{}, fmt.Errorf("could not retrieve instance '%s': %w", id, err)
	}
	if IsDeletingInstance(instance) {
		return false, instance, nil
	}
	desiredStatus := normalizeDesiredInstanceStatus(instance.DesiredStatus)
	if desiredStatus == "" || strings.TrimSpace(instance.PublicKey) == "" {
		return false, instance, nil
	}
	if err := ctx.Err(); err != nil {
		return false, instance, err
	}
	if err := progress(20, "loading provisioner", map[string]string{"instance_id": instance.ID, "provisioner": instance.KindID}); err != nil {
		return false, instance, err
	}
	provisioner, err := cm.GetProvisioner(instance.KindID)
	if err != nil {
		return false, instance, err
	}
	if err := provisioner.Init(); err != nil {
		return false, instance, fmt.Errorf("init provisioner %s: %w", instance.KindID, err)
	}
	if err := ctx.Err(); err != nil {
		return false, instance, err
	}
	instance.DesiredStatus = desiredStatus
	if err := progress(50, "applying desired instance state", map[string]string{
		"instance_id":    instance.ID,
		"desired_status": desiredStatus,
	}); err != nil {
		return false, instance, err
	}
	updated, err := reconcileProvisionerInstance(provisioner, instance)
	if err != nil {
		return false, instance, fmt.Errorf("reconcile instance %s: %w", instance.Name, err)
	}
	updated = mergedReconciledInstance(instance, updated)
	if persistentInstanceEqual(instance, updated) {
		return false, updated, nil
	}
	if err := progress(85, "saving observed instance state", map[string]string{"instance_id": instance.ID}); err != nil {
		return false, updated, err
	}
	im, cmm := createInstanceUpdateMapper(updated)
	if err := db.Update(cm.db, im, cmm); err != nil {
		return false, updated, fmt.Errorf("save reconciled instance %s: %w", instance.Name, err)
	}
	return true, updated, nil
}

func (cm *Manager) pendingInstanceStatus(instance InstanceInfo) string {
	if cm == nil || cm.tasks == nil {
		return ServerStateChanging
	}
	task, found, err := cm.tasks.LatestForSubject(InstanceDeploymentTaskStream, taskSubjectInstance, instance.ID)
	if err != nil {
		log.Debugf("failed to load deployment task for pending instance '%s': %s", instance.Name, err.Error())
		return ServerStateChanging
	}
	if !found {
		return ServerStateChanging
	}
	status := string(task.Status)
	if task.Message != "" {
		status += ": " + task.Message
	}
	if task.ErrorMessage != "" {
		status += ": " + task.ErrorMessage
	}
	return status
}

func reconcileProvisionerInstance(provisioner Provisioner, instance InstanceInfo) (InstanceInfo, error) {
	if reconciler, ok := provisioner.(InstanceReconciler); ok {
		return reconciler.ReconcileInstance(instance)
	}
	computeProvider, err := requireComputeProvisioner(provisioner)
	if err != nil {
		return InstanceInfo{}, err
	}
	return reconcileComputeInstance(computeProvider, instance)
}

func reconcileComputeInstance(provider ComputeProvisioner, instance InstanceInfo) (InstanceInfo, error) {
	current, providerInstanceID, err := getProviderInstanceInfo(provider, instance)
	if err != nil {
		return InstanceInfo{}, err
	}
	if current.Status == ServerStateChanging {
		current.DesiredStatus = instance.DesiredStatus
		return current, nil
	}
	switch strings.ToLower(strings.TrimSpace(instance.DesiredStatus)) {
	case ServerStateRunning:
		if current.Status != ServerStateRunning {
			if err := provider.StartInstance(providerInstanceID, instance.Location); err != nil {
				return InstanceInfo{}, err
			}
			current, err = provider.GetInstanceInfo(providerInstanceID, instance.Location)
			if err != nil {
				return InstanceInfo{}, err
			}
		}
	case ServerStateStopped:
		if current.Status != ServerStateStopped {
			if err := provider.StopInstance(providerInstanceID, instance.Location); err != nil {
				return InstanceInfo{}, err
			}
			current, err = provider.GetInstanceInfo(providerInstanceID, instance.Location)
			if err != nil {
				return InstanceInfo{}, err
			}
		}
	}
	current.DesiredStatus = instance.DesiredStatus
	return current, nil
}

func normalizeDesiredInstanceStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case ServerStateRunning:
		return ServerStateRunning
	case ServerStateStopped:
		return ServerStateStopped
	default:
		return ""
	}
}

func lifecycleDesiredSignature(instance InstanceInfo, desiredStatus string) string {
	return strings.Join([]string{
		strings.TrimSpace(instance.ID),
		strings.TrimSpace(instance.Kind),
		strings.TrimSpace(instance.KindID),
		strings.TrimSpace(instance.ProviderResourceID),
		strings.TrimSpace(instance.PublicKey),
		strings.TrimSpace(instance.Location),
		strings.TrimSpace(desiredStatus),
	}, "|")
}

func (cm *Manager) lifecycleSignatureCurrent(instanceID string, sig string) bool {
	if cm == nil {
		return false
	}
	cm.lifecycleMu.Lock()
	defer cm.lifecycleMu.Unlock()
	if cm.lifecycleSig == nil {
		cm.lifecycleSig = map[string]string{}
	}
	return cm.lifecycleSig[instanceID] == sig
}

func (cm *Manager) setLifecycleSignature(instanceID string, sig string) {
	if cm == nil || strings.TrimSpace(instanceID) == "" || strings.TrimSpace(sig) == "" {
		return
	}
	cm.lifecycleMu.Lock()
	defer cm.lifecycleMu.Unlock()
	if cm.lifecycleSig == nil {
		cm.lifecycleSig = map[string]string{}
	}
	cm.lifecycleSig[instanceID] = sig
}

func (cm *Manager) clearLifecycleSignature(instanceID string) {
	if cm == nil || strings.TrimSpace(instanceID) == "" {
		return
	}
	cm.lifecycleMu.Lock()
	defer cm.lifecycleMu.Unlock()
	delete(cm.lifecycleSig, instanceID)
}

func mergedReconciledInstance(current InstanceInfo, observed InstanceInfo) InstanceInfo {
	if observed.ProviderResourceID == "" && observed.ID != "" && observed.ID != current.ID {
		observed.ProviderResourceID = observed.ID
	}
	observed.ID = current.ID
	if observed.Name == "" {
		observed.Name = current.Name
	}
	if observed.PublicKey == "" {
		observed.PublicKey = current.PublicKey
	}
	if observed.Kind == "" {
		observed.Kind = current.Kind
	}
	if observed.KindID == "" {
		observed.KindID = current.KindID
	}
	if observed.ProviderResourceID == "" {
		observed.ProviderResourceID = current.ProviderResourceID
	}
	if observed.DesiredStatus == "" {
		observed.DesiredStatus = current.DesiredStatus
	}
	if observed.ReplicationPriority <= 0 {
		observed.ReplicationPriority = current.ReplicationPriority
	}
	if observed.Location == "" {
		observed.Location = current.Location
	}
	if observed.Architecture == "" {
		observed.Architecture = current.Architecture
	}
	return observed
}

func persistentInstanceEqual(a InstanceInfo, b InstanceInfo) bool {
	return a.ID == b.ID &&
		a.Name == b.Name &&
		a.Kind == b.Kind &&
		a.KindID == b.KindID &&
		a.ProviderResourceID == b.ProviderResourceID &&
		a.DesiredStatus == b.DesiredStatus &&
		a.ReplicationPriority == b.ReplicationPriority &&
		a.PublicIP == b.PublicIP &&
		a.Location == b.Location &&
		a.Architecture == b.Architecture &&
		a.PublicKey == b.PublicKey
}

func (cm *Manager) uploadLocalImageImperative(ctx context.Context, progress func(int, string, any, bool) error, imagePath string, imageName string, cloudName string, cloudLocation string, timeout time.Duration) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if progress == nil {
		progress = func(int, string, any, bool) error { return nil }
	}
	errMsg := fmt.Sprintf("failed to upload local image '%s' to cloud '%s'", imagePath, cloudName)

	if err := progress(5, "validating image", map[string]string{
		"image_path":  imagePath,
		"image_name":  imageName,
		"provisioner": cloudName,
		"location":    cloudLocation,
	}, true); err != nil {
		return "", err
	}
	// check local image file
	finfo, err := os.Stat(imagePath)
	if err != nil {
		return "", fmt.Errorf("%s: %w", errMsg, err)
	}
	if !finfo.IsDir() && finfo.Size() == 0 {
		return "", fmt.Errorf("%s: Image '%s' has 0 bytes", errMsg, imagePath)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	if err := progress(15, "loading provisioner", map[string]string{
		"provisioner": cloudName,
		"location":    cloudLocation,
	}, true); err != nil {
		return "", err
	}
	provider, err := cm.GetProvider(cloudName)
	if err != nil {
		return "", fmt.Errorf("%s: %w", errMsg, err)
	}

	imageProvider, err := requireImageProvisioner(provider)
	if err != nil {
		return "", fmt.Errorf("%s: %w", errMsg, err)
	}
	if err := provider.Init(); err != nil {
		return "", fmt.Errorf("%s: %w", errMsg, err)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	if err := progress(30, "checking existing images", map[string]string{
		"image_name": imageName,
		"location":   cloudLocation,
	}, true); err != nil {
		return "", err
	}
	// find image
	images, err := imageProvider.GetImages()
	if err != nil {
		return "", fmt.Errorf("%s: %w", errMsg, err)
	}
	for _, img := range images {
		if img.Location == cloudLocation && img.Name == imageName {
			return "", fmt.Errorf("%s: Found an image with the same name", errMsg)
		}
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	uploadDetails := map[string]string{
		"image_path":  imagePath,
		"image_name":  imageName,
		"provisioner": cloudName,
		"location":    cloudLocation,
	}
	if err := progress(60, "uploading image", uploadDetails, true); err != nil {
		return "", err
	}
	if err := progress(60, "upload in progress", uploadDetails, false); err != nil {
		return "", err
	}
	// upload image
	id, err := imageProvider.UploadLocalImage(ctx, imagePath, imageName, cloudLocation, timeout, uploadTaskProgressSink(progress, uploadDetails))
	if err != nil {
		return "", fmt.Errorf("%s: %w", errMsg, err)
	}
	if err := progress(95, "image uploaded", map[string]string{
		"image_id":    id,
		"image_name":  imageName,
		"provisioner": cloudName,
		"location":    cloudLocation,
	}, true); err != nil {
		return "", err
	}
	return id, nil
}

func uploadTaskProgressSink(progress func(int, string, any, bool) error, baseDetails map[string]string) UploadProgressFunc {
	lastProgress := -1
	lastAt := time.Time{}
	return func(update UploadProgress) error {
		total := update.TotalBytes
		transferred := update.BytesTransferred
		if transferred < 0 {
			transferred = 0
		}
		if total > 0 && transferred > total {
			transferred = total
		}
		taskProgress := 60
		percent := int64(0)
		if total > 0 {
			percent = transferred * 100 / total
			taskProgress = 60 + int(transferred*30/total)
			if taskProgress > 90 {
				taskProgress = 90
			}
		}
		if taskProgress == lastProgress && time.Since(lastAt) < 2*time.Second {
			return nil
		}
		lastProgress = taskProgress
		lastAt = time.Now()
		details := map[string]any{
			"bytes_uploaded":     transferred,
			"archive_size_bytes": total,
			"percent":            percent,
			"phase":              update.Phase,
		}
		for key, value := range baseDetails {
			details[key] = value
		}
		message := strings.TrimSpace(update.Message)
		if message == "" {
			message = "upload in progress"
		}
		return progress(taskProgress, message, details, false)
	}
}
