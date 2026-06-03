package provisioners

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path"
	"strings"
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
func CreateManager(db *db.DB, um *user.Manager, sm *pcrypto.Manager, p2p *p2p.P2P, provisioners ...ProvisionerFactory) (*Manager, error) {
	if db == nil || um == nil || sm == nil || p2p == nil {
		return nil, fmt.Errorf("failed to create cloud manager: none of the inputs can be nil")
	}

	manager := &Manager{db: db, um: um, sm: sm, p2p: p2p, provisioners: newProvisionerRegistry(), tasks: tasks.NewManager(db)}
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
	instance, err := cm.GetInstance(id)
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
	if record.ID == "" {
		return nil, fmt.Errorf("cloud provider id is empty")
	}
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
	records, err := cm.findProviderRecordsByID(record.ID)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return db.Insert(cm.db, createCloudProviderInsertMapper(record))
	}
	return db.Update(cm.db, createCloudProviderUpdateMapper(record))
}

func (cm *Manager) findProviderRecord(ref string) (ProviderRecord, bool, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ProviderRecord{}, false, fmt.Errorf("cloud provider name is empty")
	}

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

	cpModel := sq.New[db.CLOUD_PROVIDER]("")
	records, err = db.SelectMultiple(cm.db, createCloudProviderQueryMapper([]sq.Predicate{cpModel.NAME.EqString(ref)}))
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
	return db.SelectMultiple(cm.db, createCloudProviderQueryMapper([]sq.Predicate{cpModel.ID.EqString(id)}))
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
		ID:            pendingID,
		Name:          instanceName,
		Kind:          KindCloudVM,
		KindID:        provider.NameStr(),
		DesiredStatus: ServerStateRunning,
		WitnessRank:   db.DefaultWitnessRankForMachine(KindCloudVM, provider.NameStr()),
		Location:      cloudLocation,
		Status:        ServerStateChanging,
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
	for id, img := range images {
		if img.Location == cloudLocation && img.Name == release.Version {
			imageID = id
			break
		}
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
	instanceInfo.WitnessRank = db.DefaultWitnessRankForMachine(instanceInfo.Kind, instanceInfo.KindID)
	if pendingInstanceID != "" {
		pendingUpdate := instanceInfo
		pendingUpdate.ID = pendingInstanceID
		pendingUpdate.PublicKey = ""
		pendingUpdate.Architecture = ""
		pendingUpdate.DesiredStatus = ServerStateRunning
		pendingUpdate.WitnessRank = db.DefaultWitnessRankForMachine(pendingUpdate.Kind, pendingUpdate.KindID)
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
	instanceInfo.ID = discoveredPeer.ID
	log.Infof("Discovered instance peer '%s' at '%s' with fingerprint '%s'", discoveredPeer.ID, discoveredPeer.Address, discoveredPeer.Fingerprint)

	_ = progress(85, "initializing VM", map[string]string{"peer_id": instanceInfo.ID})
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
	instanceInfo.WitnessRank = db.DefaultWitnessRankForMachine(instanceInfo.Kind, instanceInfo.KindID)

	_ = progress(95, "saving VM identity", map[string]string{"peer_id": instanceInfo.ID})
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
		PublicIP:      ipString,
		Name:          instanceName,
		Kind:          kind,
		KindID:        kindID,
		DesiredStatus: ServerStateRunning,
		WitnessRank:   db.DefaultWitnessRankForMachine(kind, kindID),
		Location:      locationName,
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
	instanceInfo.ID = discoveredPeer.ID
	log.Infof("Discovered instance peer '%s' at '%s' with fingerprint '%s'", discoveredPeer.ID, discoveredPeer.Address, discoveredPeer.Fingerprint)

	machineMapper, machineMetadataMapper := createInstanceInsertMapper(instanceInfo)
	insertedInstance := false
	if err := db.Insert(cm.db, machineMapper, machineMetadataMapper, db.CreatePeerInsertMapper(instanceInfo.ID)); err != nil {
		return fmt.Errorf("failed to save instance '%s': %w", instanceName, err)
	}
	insertedInstance = true
	defer func() {
		if err == nil || !insertedInstance {
			return
		}
		im, cmmd := createInstanceDeleteMapper(instanceInfo.ID)
		_ = db.Delete(cm.db, db.CreatePeerDeleteMapper(instanceInfo.ID), im, cmmd)
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
	instance, err := cm.GetInstance(id)
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

// DeleteInstance deletes an instance
func (cm *Manager) DeleteInstance(id string) error {
	return cm.deleteInstance(id, false)
}

// DeleteInstanceLocal deletes only local database and peer state for an instance.
func (cm *Manager) DeleteInstanceLocal(id string) error {
	return cm.deleteInstance(id, true)
}

func (cm *Manager) deleteInstance(id string, localOnly bool) error {
	instance, err := cm.GetInstance(id)
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

	if err := cm.markInstanceDeleting(instance); err != nil {
		return err
	}
	instance.DesiredStatus = ServerStateDeleting

	if err := cm.deleteAppsForInstance(instance.ID); err != nil {
		return err
	}

	if strings.TrimSpace(instance.PublicKey) != "" {
		if err := cm.removeInstanceWitnessEligibility(instance); err != nil {
			return err
		}
	}

	if !localOnly && (instance.Kind == KindCloudVM || instance.Kind == KindLocalVM) {
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
				err = computeProvider.StopInstance(providerInstanceID, instance.Location)
				if err != nil {
					log.Warnf("failed to stop instance '%s' before delete; attempting provider delete anyway: %s", id, err.Error())
				}
			}
			log.Infof("Deleting instance '%s' (%s)", instance.Name, providerInstanceID)
			err = computeProvider.DeleteInstance(providerInstanceID, instance.Location)
			if err != nil {
				return fmt.Errorf("could not delete instance '%s': %w", id, err)
			}
			for _, vol := range vmInfo.Volumes {
				log.Infof("Deleting volume '%s' (%s) for instance '%s'", vol.Name, vol.VolumeID, id)
				err = volumeProvider.DeleteVolume(vol.VolumeID, instance.Location)
				if err != nil {
					log.Errorf("failed to delete volume '%s': %s", vol.Name, err.Error())
				}
			}
		}
	}

	if strings.TrimSpace(instance.PublicKey) != "" {
		err = cm.p2p.RemovePeer(instance)
		if err != nil {
			return fmt.Errorf("failed to remove peer: %w", err)
		}
		if err := cm.removeInstanceWitnessEligibility(instance); err != nil {
			return err
		}
	}

	if err := cm.deleteInstanceRecords(instance); err != nil {
		return fmt.Errorf("failed to delete instance '%s': %w", id, err)
	}
	if err := cm.deleteInstanceSSHKey(instance.ID); err != nil {
		log.Warnf("failed to delete SSH key for instance '%s': %s", instance.Name, err.Error())
	}

	return nil
}

func (cm *Manager) deleteInstanceRecords(instance InstanceInfo) error {
	if cm == nil || cm.db == nil {
		return fmt.Errorf("provisioner manager database is not configured")
	}
	if strings.TrimSpace(instance.ID) == "" {
		return fmt.Errorf("instance ID is empty")
	}

	var lastErr error
	deadline := time.Now().Add(10 * time.Minute)
	for attempt := 1; time.Now().Before(deadline); attempt++ {
		im, cmmd := createInstanceDeleteMapper(instance.ID)
		err := db.Delete(cm.db, db.CreatePeerDeleteMapper(instance.ID), createAppDeleteByInstanceMapper(instance.ID), im, cmmd)
		if err == nil {
			err = cm.assertInstancePeerRemoved(instance.ID)
		}
		if err == nil {
			if verifyErr := cm.waitForInstanceDeleteFinalized(instance.ID); verifyErr == nil {
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
		time.Sleep(sleep)
	}

	return fmt.Errorf("instance '%s' removed but residual peer state remains: %w", instance.Name, lastErr)
}

func (cm *Manager) waitForInstanceDeleteFinalized(peerID string) error {
	deadline := time.Now().Add(2 * time.Minute)
	var lastErr error
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		catchUpErr := cm.db.CatchUpFinalized(ctx, "verify instance delete finalized")
		cancel()
		if catchUpErr != nil {
			lastErr = catchUpErr
			time.Sleep(5 * time.Second)
			continue
		}

		status, ok := cm.db.SwarmionStatus()
		if !ok {
			if err := cm.assertInstancePeerRemoved(peerID); err != nil {
				return err
			}
			return nil
		}
		finalized := status.FinalizedRootHash.String()
		durable := status.DurableMainRootHash.String()
		if finalized == "" ||
			status.TentativeRootHash.String() != finalized ||
			(durable != "" && durable != finalized) {
			time.Sleep(5 * time.Second)
			continue
		}
		if err := cm.assertInstancePeerRemoved(peerID); err != nil {
			return err
		}
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("timed out waiting for finalized delete of peer %s", peerID)
}

func (cm *Manager) markInstanceDeleting(instance InstanceInfo) error {
	if cm == nil || cm.db == nil || strings.TrimSpace(instance.ID) == "" || IsDeletingInstance(instance) {
		return nil
	}
	instance.DesiredStatus = ServerStateDeleting
	im, _ := createInstanceUpdateMapper(instance)
	if err := db.Update(cm.db, im); err != nil {
		return fmt.Errorf("failed to mark instance '%s' as deleting: %w", instance.Name, err)
	}
	return nil
}

func (cm *Manager) deleteAppsForInstance(instanceID string) error {
	instanceID = strings.TrimSpace(instanceID)
	if cm == nil || cm.db == nil || instanceID == "" {
		return nil
	}
	if err := db.Delete(cm.db, createAppDeleteByInstanceMapper(instanceID)); err != nil {
		return fmt.Errorf("failed to delete apps for instance '%s': %w", instanceID, err)
	}
	return nil
}

func (cm *Manager) assertInstancePeerRemoved(peerID string) error {
	peerID = strings.TrimSpace(peerID)
	if cm == nil || cm.db == nil || peerID == "" {
		return nil
	}
	peerIDs, err := db.GetPeerIDs(cm.db)
	if err != nil {
		return err
	}
	if _, found := peerIDs[peerID]; found {
		return fmt.Errorf("peer table still contains %s", peerID)
	}
	status, ok := cm.db.SwarmionStatus()
	if !ok {
		return nil
	}
	if containsString(status.ActiveWitnessIDs, peerID) {
		return fmt.Errorf("swarmion active witnesses still contain %s", peerID)
	}
	if containsString(status.EligibleWitnessIDs, peerID) {
		return fmt.Errorf("swarmion eligible witnesses still contain %s", peerID)
	}
	if rank, found := status.EligibleWitnessRanks[peerID]; found && rank > 0 {
		return fmt.Errorf("swarmion eligible witness rank for %s is still %d", peerID, rank)
	}
	if containsString(status.StateProviders, peerID) {
		return fmt.Errorf("swarmion state providers still contain %s", peerID)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	peerStatuses, err := cm.db.SwarmionPeerStatus(ctx)
	if err != nil {
		return fmt.Errorf("read swarmion peer status: %w", err)
	}
	for _, peerStatus := range peerStatuses {
		if strings.TrimSpace(peerStatus.PeerID) != peerID {
			continue
		}
		if peerStatus.Witness || peerStatus.EligibleWitness || peerStatus.StateProvider {
			return fmt.Errorf(
				"swarmion peer status still marks %s as witness=%t eligible=%t state_provider=%t",
				peerID,
				peerStatus.Witness,
				peerStatus.EligibleWitness,
				peerStatus.StateProvider,
			)
		}
	}
	return nil
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func (cm *Manager) removeInstanceWitnessEligibility(instance InstanceInfo) error {
	if cm == nil || cm.db == nil {
		return nil
	}
	candidates, err := cm.witnessCandidatesExcluding(instance.ID)
	if err != nil {
		return fmt.Errorf("build remaining witness candidates for instance '%s': %w", instance.Name, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if err := cm.db.RemoveWitnessEligibility(ctx, instance.ID, candidates); err != nil {
		return fmt.Errorf("remove swarmion witness eligibility for instance '%s': %w", instance.Name, err)
	}
	return nil
}

func (cm *Manager) witnessCandidatesExcluding(peerID string) ([]db.WitnessCandidate, error) {
	peerID = strings.TrimSpace(peerID)
	instances, err := cm.GetInstances(false)
	if err != nil {
		return nil, err
	}
	candidates := make([]db.WitnessCandidate, 0, len(instances)+1)
	for _, instance := range instances {
		if strings.TrimSpace(instance.ID) == "" || instance.ID == peerID || strings.TrimSpace(instance.PublicKey) == "" || IsDeletingInstance(instance) {
			continue
		}
		candidates = append(candidates, db.WitnessCandidate{
			PeerID:     instance.ID,
			DeviceType: db.WitnessDeviceTypeForMachine(instance.Kind, instance.KindID),
			Rank:       instance.WitnessRank,
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
		if strings.TrimSpace(device.ID) == "" || device.ID == peerID {
			continue
		}
		candidates = append(candidates, db.WitnessCandidate{
			PeerID:     device.ID,
			DeviceType: db.WitnessDeviceTypeForUserDeviceName(device.Name),
			Rank:       device.WitnessRank,
		})
	}
	return candidates, nil
}

// StartInstance starts an instance
func (cm *Manager) StartInstance(id string) error {
	return cm.setInstanceDesiredStatus(id, ServerStateRunning)
}

// StopInstance stops an instance
func (cm *Manager) StopInstance(id string) error {
	return cm.setInstanceDesiredStatus(id, ServerStateStopped)
}

func (cm *Manager) setInstanceDesiredStatus(id string, desiredStatus string) error {
	instance, err := cm.GetInstance(id)
	if err != nil {
		return fmt.Errorf("could not retrieve instance '%s': %w", id, err)
	}
	instance.DesiredStatus = desiredStatus
	im, _ := createInstanceUpdateMapper(instance)
	if err := db.Update(cm.db, im); err != nil {
		return fmt.Errorf("failed to save instance '%s': %w", id, err)
	}
	log.Infof("Set desired status for instance '%s' to '%s'", instance.Name, desiredStatus)
	return cm.ReconcileDesiredInstances()
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

	iqm := createInstanceQueryMapper(id)
	instance, err := db.SelectOne(cm.db, iqm)
	if err != nil {
		instanceByName, nameErr := db.SelectOne(cm.db, createInstanceQueryByNameMapper(id))
		if nameErr != nil {
			return instance, fmt.Errorf("failed to retrieve instance: %w", err)
		}
		instance = instanceByName
	}

	if IsDeletingInstance(instance) {
		instance.Status = ServerStateDeleting
		return instance, nil
	}

	if strings.TrimSpace(instance.PublicKey) == "" {
		instance.Status = cm.pendingInstanceStatus(instance)
		return instance, nil
	}

	// if not local, we update the instance status
	if instance.Kind != KindLocalVM {
		provider, err := cm.GetProvider(instance.KindID)
		if err != nil {
			return InstanceInfo{}, err
		}
		computeProvider, err := requireComputeProvisioner(provider)
		if err != nil {
			return InstanceInfo{}, err
		}
		if err := provider.Init(); err != nil {
			return InstanceInfo{}, err
		}
		instanceInfo, _, err := getProviderInstanceInfo(computeProvider, instance)
		if err != nil {
			log.Errorf("failed to retrieve remote status from instance '%s': %s", instance.Name, err.Error())
			instance.Status = "n/a"
		} else {
			instance.Status = instanceInfo.Status
		}
	} else {
		instance.Status = "n/a"
	}
	return instance, nil
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

	if instance.Kind != KindLocalVM {
		provider, err := cm.GetProvider(instance.KindID)
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

	return "n/a", nil
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

// ReconcileDesiredInstances applies persisted desired machine state through
// provisioners that expose a declarative reconcile hook.
func (cm *Manager) ReconcileDesiredInstances() error {
	instances, err := cm.GetInstances(false)
	if err != nil {
		return err
	}

	var failures []string
	for _, instance := range instances {
		if IsDeletingInstance(instance) {
			continue
		}
		desiredStatus := normalizeDesiredInstanceStatus(instance.DesiredStatus)
		if desiredStatus == "" || instance.Kind == KindLocalVM || strings.TrimSpace(instance.PublicKey) == "" {
			continue
		}

		provisioner, err := cm.GetProvisioner(instance.KindID)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		if err := provisioner.Init(); err != nil {
			failures = append(failures, fmt.Sprintf("init provisioner %s: %v", instance.KindID, err))
			continue
		}

		instance.DesiredStatus = desiredStatus
		updated, err := reconcileProvisionerInstance(provisioner, instance)
		if err != nil {
			failures = append(failures, fmt.Sprintf("reconcile instance %s: %v", instance.Name, err))
			continue
		}
		updated = mergedReconciledInstance(instance, updated)
		if persistentInstanceEqual(instance, updated) {
			continue
		}
		im, cmm := createInstanceUpdateMapper(updated)
		if err := db.Update(cm.db, im, cmm); err != nil {
			failures = append(failures, fmt.Sprintf("save reconciled instance %s: %v", instance.Name, err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("%s", strings.Join(failures, "; "))
	}
	return nil
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
	if observed.WitnessRank <= 0 {
		observed.WitnessRank = current.WitnessRank
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
		a.WitnessRank == b.WitnessRank &&
		a.PublicIP == b.PublicIP &&
		a.Location == b.Location &&
		a.Architecture == b.Architecture &&
		a.PublicKey == b.PublicKey
}

// UploadLocalImage uploads a local Protosd image to a specific cloud
func (cm *Manager) UploadLocalImage(imagePath string, imageName string, cloudName string, cloudLocation string, timeout time.Duration) error {
	errMsg := fmt.Sprintf("failed to upload local image '%s' to cloud '%s'", imagePath, cloudName)
	// check local image file
	finfo, err := os.Stat(imagePath)
	if err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}
	if !finfo.IsDir() && finfo.Size() == 0 {
		return fmt.Errorf("%s: Image '%s' has 0 bytes", errMsg, imagePath)
	}

	provider, err := cm.GetProvider(cloudName)
	if err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}

	imageProvider, err := requireImageProvisioner(provider)
	if err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}
	if err := provider.Init(); err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}

	// find image
	images, err := imageProvider.GetImages()
	if err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}
	for _, img := range images {
		if img.Location == cloudLocation && img.Name == imageName {
			return fmt.Errorf("%s: Found an image with the same name", errMsg)
		}
	}

	// upload image
	_, err = imageProvider.UploadLocalImage(imagePath, imageName, cloudLocation, timeout)
	if err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}
	return nil
}
