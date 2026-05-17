package cloud

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
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/p2p"
	"github.com/protosio/protos/internal/p2p/proto"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/release"
	"github.com/protosio/protos/internal/user"
	"github.com/protosio/protos/internal/util"
)

var log = util.GetLogger("cloud")

// Type represents a specific cloud (AWS, GCP, DigitalOcean etc.)
type Type string

func (ct Type) String() string {
	return string(ct)
}

// CreateManager creates and returns a cloud manager
func CreateManager(db *db.DB, um *user.Manager, sm *pcrypto.Manager, p2p *p2p.P2P) (*Manager, error) {
	if db == nil || um == nil || sm == nil || p2p == nil {
		return nil, fmt.Errorf("failed to create cloud manager: none of the inputs can be nil")
	}

	manager := &Manager{db: db, um: um, sm: sm, p2p: p2p, providers: defaultProviderRegistry()}

	return manager, nil
}

// Manager manages cloud providers and instances
type Manager struct {
	db        *db.DB
	um        *user.Manager
	sm        *pcrypto.Manager
	p2p       *p2p.P2P
	providers *providerRegistry
}

//
// Cloud manager methods
//

// AddProvider validates and saves a cloud provider configuration.
func (cm *Manager) AddProvider(cloudName string, cloud string, auth map[string]string) error {
	record := newProviderRecord(cloudName, Type(cloud), auth)
	provider, err := cm.newProviderClient(record)
	if err != nil {
		return fmt.Errorf("failed to create cloud provider: %w", err)
	}
	if err := provider.Init(); err != nil {
		return fmt.Errorf("failed to initialize cloud provider: %w", err)
	}
	if err := cm.saveProviderRecord(provider.ProviderRecord()); err != nil {
		return fmt.Errorf("failed to save cloud provider: %w", err)
	}
	return nil
}

// SupportedProviders returns a list of supported cloud providers
func (cm *Manager) SupportedProviders() []string {
	providerTypes := cm.providers.types()
	supportedProviders := make([]string, 0, len(providerTypes))
	for _, providerType := range providerTypes {
		supportedProviders = append(supportedProviders, providerType.String())
	}
	return supportedProviders
}

// ProviderAuthFields returns the authentication fields required by a provider type.
func (cm *Manager) ProviderAuthFields(cloud string) ([]string, error) {
	factory, found := cm.providers.factory(Type(cloud))
	if !found {
		return nil, fmt.Errorf("cloud '%s' not supported", cloud)
	}
	return factory.AuthFields(), nil
}

// GetProvider returns a cloud provider instance from the db
func (cm *Manager) GetProvider(id string) (ProviderClient, error) {
	record, found, err := cm.findProviderRecord(id)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve cloud provider '%s': %w", id, err)
	}
	if !found {
		return nil, fmt.Errorf("could not find cloud provider '%s'", id)
	}
	provider, err := cm.newProviderClient(record)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve cloud provider '%s': %w", id, err)
	}
	return provider, nil
}

// DeleteProvider deletes a cloud provider from the db
func (cm *Manager) DeleteProvider(name string) error {
	record, found, err := cm.findProviderRecord(name)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("could not find cloud provider '%s'", name)
	}

	err = db.Delete(cm.db, createCloudProviderDeleteMapper(record.ID))
	if err != nil {
		return fmt.Errorf("failed to delete instance '%s': %w", name, err)
	}

	return nil
}

// GetProviders returns all the cloud providers from the db
func (cm *Manager) GetProviders() ([]ProviderClient, error) {
	cloudProviders := []ProviderClient{}
	clouds, err := db.SelectMultiple(cm.db, createCloudProviderQueryMapper(nil))
	if err != nil {
		return cloudProviders, fmt.Errorf("failed to retrieve cloud providers: %w", err)
	}

	for _, cloud := range clouds {
		client, err := cm.newProviderClient(cloud)
		if err != nil {
			return cloudProviders, err
		}
		cloudProviders = append(cloudProviders, client)
	}

	return cloudProviders, nil
}

func (cm *Manager) providerDeps() ProviderDeps {
	return ProviderDeps{SecretManager: cm.sm}
}

func (cm *Manager) newProviderClient(record ProviderRecord) (ProviderClient, error) {
	record = record.normalized()
	if record.ID == "" {
		return nil, fmt.Errorf("cloud provider id is empty")
	}
	if record.Name == "" {
		return nil, fmt.Errorf("cloud provider name is empty")
	}
	factory, found := cm.providers.factory(record.Type)
	if !found {
		return nil, fmt.Errorf("cloud '%s' not supported", record.Type.String())
	}
	return factory.NewClient(record, cm.providerDeps())
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
func (cm *Manager) DeployInstance(instanceName string, cloudName string, cloudLocation string, release release.Release, machineType string) (InstanceInfo, error) {
	// init cloud
	provider, err := cm.GetProvider(cloudName)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("could not retrieve cloud '%s': %w", cloudName, err)
	}
	computeProvider, err := requireComputeProvider(provider)
	if err != nil {
		return InstanceInfo{}, err
	}
	imageProvider, err := requireImageProvider(provider)
	if err != nil {
		return InstanceInfo{}, err
	}
	volumeProvider, err := requireVolumeProvider(provider)
	if err != nil {
		return InstanceInfo{}, err
	}
	if err := provider.Init(); err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to init cloud provider '%s'(%s) API: %w", cloudName, provider.TypeStr(), err)
	}

	// validate machine type
	supportedMachineTypes, err := computeProvider.SupportedMachines(cloudLocation)
	if err != nil {
		return InstanceInfo{}, err
	}
	if _, found := supportedMachineTypes[machineType]; !found {
		return InstanceInfo{}, errors.Errorf("Machine type '%s' is not valid for cloud provider '%s'. The following types are supported: \n%s", machineType, provider.TypeStr(), createMachineTypesString(supportedMachineTypes))
	}

	// add image
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
			imageID, err = imageProvider.AddImage(image.URL, image.Digest, release.Version, cloudLocation)
			if err != nil {
				return InstanceInfo{}, fmt.Errorf("failed to deploy Protos instance: %w", err)
			}
		} else {
			return InstanceInfo{}, errors.Errorf("could not find a Protos version '%s' release for cloud '%s'", release.Version, provider.TypeStr())
		}
	}

	// create SSH key used for instance
	log.Info("Generating SSH key for the new VM instance")
	instanceSSHKey, err := cm.sm.GenerateKey()
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to deploy Protos instance: %w", err)
	}

	// deploy a protos instance
	log.Infof("Deploying instance '%s' of type '%s', using Protos version '%s' (image id '%s')", instanceName, machineType, release.Version, imageID)
	vmID, err := computeProvider.NewInstance(instanceName, imageID, instanceSSHKey.AuthorizedKey(), machineType, cloudLocation)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to deploy Protos instance: %w", err)
	}
	log.Infof("Instance with ID '%s' deployed", vmID)

	// get instance info
	instanceInfo, err := computeProvider.GetInstanceInfo(vmID, cloudLocation)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to get Protos instance info: %w", err)
	}

	thisDevice, err := cm.um.GetCurrentDevice()
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to get current device : %w", err)
	}

	// create protos data volume
	log.Infof("creating data volume for Protos instance '%s'", instanceName)
	volumeID, err := volumeProvider.NewVolume(instanceName, 30000, cloudLocation)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to create data volume: %w", err)
	}

	// attach volume to instance
	err = volumeProvider.AttachVolume(volumeID, vmID, cloudLocation)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to attach volume to instance '%s': %w", instanceName, err)
	}

	// start protos instance
	log.Infof("Starting instance '%s'", instanceName)
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

	// wait for port 22 to be open
	err = util.WaitForPort(instanceInfo.PublicIP, "22", 20)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to deploy instance: %w", err)
	}

	// connect via SSH
	sshCon, err := pcrypto.NewConnection(instanceInfo.PublicIP, "root", instanceSSHKey.SSHAuth(), 10)
	if err != nil {
		return InstanceInfo{}, err
	}

	// retrieve instance public key via SSH
	instanceInfo.PublicKey, err = pcrypto.ExecuteCommand(fmt.Sprintf("cat %s", protosPublicKey), sshCon)
	if err != nil {
		return InstanceInfo{}, err
	}

	// close SSH connection
	sshCon.Close()

	p2pClient, err := cm.p2p.AddPeer(instanceInfo)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to initialize instance: %w", err)
	}
	if p2pClient == nil {
		return InstanceInfo{}, errors.New("failed to initialize instance: p2p client is nil")
	}

	// do the initialization
	log.Infof("Initializing instance '%s'", instanceName)
	resp, err := p2pClient.Init(context.TODO(), &proto.InitRequest{
		OriginDevice:          thisDevice.GetName(),
		OriginDevicePublicKey: thisDevice.GetPublicKey(),
		OriginSwarmionAddrs:   cm.db.ListenMultiaddrs(),
		InstanceName:          instanceName,
	})
	if err != nil {
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

	mm, cmm := createInstanceInsertMapper(instanceInfo)

	err = db.Insert(cm.db, mm, cmm, db.CreatePeerInsertMapper(instanceInfo.ID))
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to save instance '%s': %w", instanceName, err)
	}

	log.Infof("Instance '%s' at '%s' is ready", instanceName, instanceInfo.PublicIP)

	return instanceInfo, nil
}

func (cm *Manager) InitInstance(instanceName string, kind string, kindID string, locationName string, ipString string) error {
	instanceInfo := InstanceInfo{
		PublicIP: ipString,
		Name:     instanceName,
		Kind:     kind,
		KindID:   kindID,
		Location: locationName,
	}

	ip := net.ParseIP(ipString)
	if ip == nil {
		return fmt.Errorf("String '%s' is not a valid IP address", ipString)
	}

	localKey, err := cm.sm.GetLocalKey()
	if err != nil {
		return err
	}

	thisDevice, err := cm.um.GetCurrentDevice()
	if err != nil {
		return fmt.Errorf("failed to get current device : %w", err)
	}

	// wait for port 22 to be open
	err = util.WaitForPort(instanceInfo.PublicIP, "22", 20)
	if err != nil {
		return fmt.Errorf("failure while waiting for port: %w", err)
	}

	// connect via SSH
	sshCon, err := pcrypto.NewConnection(instanceInfo.PublicIP, "root", localKey.SSHAuth(), 10)
	if err != nil {
		return fmt.Errorf("failed to connect to dev instance over SSH: %w", err)
	}

	// retrieve instance public key via SSH
	publicKeyPEM, err := pcrypto.ExecuteCommand(fmt.Sprintf("cat %s", path.Join("/var/lib/protos/", pcrypto.PublicKeyFileName)), sshCon)
	if err != nil {
		return fmt.Errorf("failed to retrieve public key from instance: %w", err)
	}
	publicKey, err := pcrypto.CreatePublicKeyFromPEM(publicKeyPEM)
	if err != nil {
		return fmt.Errorf("failed to decode public key from instance: %w", err)
	}
	instanceInfo.PublicKey = publicKey.PublicKey()
	instanceInfo.ID = publicKey.GetID()

	// close SSH connection
	sshCon.Close()

	p2pClient, err := cm.p2p.AddPeer(instanceInfo)
	if err != nil {
		return fmt.Errorf("failed to initialize instance: %w", err)
	}
	if p2pClient == nil {
		return errors.New("failed to initialize instance: p2p client is nil")
	}

	// do the initialization
	log.Infof("Initializing instance '%s'", instanceName)
	resp, err := p2pClient.Init(context.TODO(), &proto.InitRequest{
		OriginDevice:          thisDevice.GetName(),
		OriginDevicePublicKey: thisDevice.GetPublicKey(),
		OriginSwarmionAddrs:   cm.db.ListenMultiaddrs(),
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

	machineMapper, machineMetadataMapper := createInstanceInsertMapper(instanceInfo)
	err = db.Insert(cm.db, machineMapper, machineMetadataMapper, db.CreatePeerInsertMapper(instanceInfo.ID))
	if err != nil {
		return fmt.Errorf("failed to save instance '%s': %w", instanceName, err)
	}

	log.Infof("Instance '%s'(%s) initialized", instanceName, ipString)

	return nil
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
	instance, err := cm.GetInstance(id)
	if err != nil {
		return fmt.Errorf("could not retrieve instance '%s': %w", id, err)
	}

	// if local only, ignore any cloud resources
	if instance.Kind == KindCloudVM {
		provider, err := cm.GetProvider(instance.KindID)
		if err != nil {
			return fmt.Errorf("could not retrieve cloud '%s': %w", id, err)
		}

		computeProvider, err := requireComputeProvider(provider)
		if err != nil {
			return err
		}
		volumeProvider, err := requireVolumeProvider(provider)
		if err != nil {
			return err
		}
		if err := provider.Init(); err != nil {
			return fmt.Errorf("could not init cloud '%s': %w", id, err)
		}

		found := true
		vmInfo, err := computeProvider.GetInstanceInfo(instance.ID, instance.Location)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				found = false
			} else {
				return fmt.Errorf("failed to get details for instance '%s': %w", id, err)
			}
		}

		// only delete cloud instance if found. Otherwise we proceed with removing it from local db
		if found {
			if vmInfo.Status == ServerStateRunning {
				log.Infof("Stopping instance '%s' (%s)", instance.Name, instance.ID)
				err = computeProvider.StopInstance(instance.ID, instance.Location)
				if err != nil {
					return fmt.Errorf("could not stop instance '%s': %w", id, err)
				}
			}
			log.Infof("Deleting instance '%s' (%s)", instance.Name, instance.ID)
			err = computeProvider.DeleteInstance(instance.ID, instance.Location)
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

	err = cm.p2p.RemovePeer(instance)
	if err != nil {
		return fmt.Errorf("failed to remove peer: %w", err)
	}

	im, cmmd := createInstanceDeleteMapper(instance.ID)
	err = db.Delete(cm.db, db.CreatePeerDeleteMapper(instance.ID), im, cmmd)
	if err != nil {
		return fmt.Errorf("failed to delete instance '%s': %w", id, err)
	}

	return nil
}

// StartInstance starts an instance
func (cm *Manager) StartInstance(id string) error {
	instance, err := cm.GetInstance(id)
	if err != nil {
		return fmt.Errorf("could not retrieve instance '%s': %w", id, err)
	}
	provider, err := cm.GetProvider(instance.KindID)
	if err != nil {
		return fmt.Errorf("could not retrieve cloud '%s': %w", id, err)
	}

	computeProvider, err := requireComputeProvider(provider)
	if err != nil {
		return err
	}
	if err := provider.Init(); err != nil {
		return fmt.Errorf("could not init cloud '%s': %w", id, err)
	}

	log.Infof("Starting instance '%s' (%s)", instance.Name, instance.ID)
	err = computeProvider.StartInstance(instance.ID, instance.Location)
	if err != nil {
		return fmt.Errorf("could not start instance '%s': %w", id, err)
	}

	// IP can change if an instance is stopped and started so a refresh is required
	info, err := computeProvider.GetInstanceInfo(instance.ID, instance.Location)
	if err != nil {
		return fmt.Errorf("could not retrieve instance info for '%s': %w", id, err)
	}

	instance.PublicIP = info.PublicIP
	instance.Volumes = info.Volumes

	im, cmm := createInstanceUpdateMapper(instance)
	err = db.Update(cm.db, im)
	if err != nil {
		return fmt.Errorf("failed to save instance '%s': %w", id, err)
	}

	err = db.Update(cm.db, cmm)
	if err != nil {
		return fmt.Errorf("failed to save instance metadata '%s': %w", id, err)
	}

	return nil
}

// StopInstance stops an instance
func (cm *Manager) StopInstance(id string) error {
	instance, err := cm.GetInstance(id)
	if err != nil {
		return fmt.Errorf("could not retrieve instance '%s': %w", id, err)
	}
	provider, err := cm.GetProvider(instance.KindID)
	if err != nil {
		return fmt.Errorf("could not retrieve cloud '%s': %w", id, err)
	}

	computeProvider, err := requireComputeProvider(provider)
	if err != nil {
		return err
	}
	if err := provider.Init(); err != nil {
		return fmt.Errorf("could not init cloud '%s': %w", id, err)
	}

	log.Infof("Stopping instance '%s' (%s)", instance.Name, instance.ID)
	err = computeProvider.StopInstance(instance.ID, instance.Location)
	if err != nil {
		return fmt.Errorf("could not stop instance '%s': %w", id, err)
	}
	return nil
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

	localKey, err := cm.sm.GetLocalKey()
	if err != nil {
		return "", err
	}

	sshCon, err := pcrypto.NewConnection(instanceInfo.PublicIP, "root", localKey.SSHAuth(), 10)
	if err != nil {
		return "", err
	}
	output, err := pcrypto.ExecuteCommand("cat /var/log/protos.log", sshCon)
	if err != nil {
		return "", err
	}
	return output, nil
}

// GetInstance retrieves an instance from the db and returns it
func (cm *Manager) GetInstance(id string) (InstanceInfo, error) {

	iqm := createInstanceQueryMapper(id)
	instance, err := db.SelectOne(cm.db, iqm)
	if err != nil {
		return instance, fmt.Errorf("failed to retrieve instance: %w", err)
	}

	// if not local, we update the instance status
	if instance.Kind != KindLocalVM {
		provider, err := cm.GetProvider(instance.KindID)
		if err != nil {
			return InstanceInfo{}, err
		}
		computeProvider, err := requireComputeProvider(provider)
		if err != nil {
			return InstanceInfo{}, err
		}
		if err := provider.Init(); err != nil {
			return InstanceInfo{}, err
		}
		instanceInfo, err := computeProvider.GetInstanceInfo(instance.ID, instance.Location)
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
		computeProvider, err := requireComputeProvider(provider)
		if err != nil {
			return "", fmt.Errorf("failed to retrieve status for instance '%s': %s", instance.Name, err.Error())
		}
		if err := provider.Init(); err != nil {
			return "", fmt.Errorf("failed to retrieve status for instance '%s': %s", instance.Name, err.Error())
		}
		instanceInfo, err := computeProvider.GetInstanceInfo(instance.ID, instance.Location)
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

// UploadLocalImage uploads a local Protosd image to a specific cloud
func (cm *Manager) UploadLocalImage(imagePath string, imageName string, cloudName string, cloudLocation string, timeout time.Duration) error {
	errMsg := fmt.Sprintf("failed to upload local image '%s' to cloud '%s'", imagePath, cloudName)
	// check local image file
	finfo, err := os.Stat(imagePath)
	if err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}
	if finfo.IsDir() {
		return fmt.Errorf("%s: Path '%s' is a directory", errMsg, imagePath)
	}
	if finfo.Size() == 0 {
		return fmt.Errorf("%s: Image '%s' has 0 bytes", errMsg, imagePath)
	}

	provider, err := cm.GetProvider(cloudName)
	if err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}

	imageProvider, err := requireImageProvider(provider)
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
