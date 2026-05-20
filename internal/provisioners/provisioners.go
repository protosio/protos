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

	manager := &Manager{db: db, um: um, sm: sm, p2p: p2p, provisioners: newProvisionerRegistry()}
	for _, provisioner := range provisioners {
		manager.RegisterProvisioner(provisioner)
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

func (cm *Manager) saveInstanceSSHKey(instanceID string, key *pcrypto.Key) error {
	if key == nil {
		return fmt.Errorf("instance SSH key is nil")
	}
	keyPath, err := instanceSSHKeyPath(instanceID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(path.Dir(keyPath), 0700); err != nil {
		return fmt.Errorf("create instance SSH key directory: %w", err)
	}
	if err := os.WriteFile(keyPath, []byte(key.EncodePrivateKeytoPEM()), 0600); err != nil {
		return fmt.Errorf("write instance SSH key: %w", err)
	}
	return nil
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
	if err != nil {
		return "", fmt.Errorf("read SSH key for instance '%s': %w", instance.Name, err)
	}
	return string(key), nil
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

// GetProvider returns a cloud provider instance from the db.
func (cm *Manager) GetProvider(id string) (ProviderClient, error) {
	return cm.GetProvisioner(id)
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
func (cm *Manager) DeployInstance(instanceName string, cloudName string, cloudLocation string, release release.Release, machineType string) (InstanceInfo, error) {
	// init cloud
	provider, err := cm.GetProvider(cloudName)
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
	if instanceInfo.ProviderResourceID == "" {
		instanceInfo.ProviderResourceID = vmID
	}
	if instanceInfo.Kind == "" {
		instanceInfo.Kind = KindCloudVM
	}
	if instanceInfo.KindID == "" {
		instanceInfo.KindID = provider.NameStr()
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
	if instanceUpdate.Status != "" {
		instanceInfo.Status = instanceUpdate.Status
	}

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
	defer sshCon.Close()

	// retrieve instance public key via SSH
	publicKeyPEM, err := waitForRemoteFile(sshCon, protosPublicKey, 2*time.Minute)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to retrieve public key from instance: %w", err)
	}
	publicKey, err := pcrypto.CreatePublicKeyFromPEM(publicKeyPEM)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to decode public key from instance: %w", err)
	}
	instanceInfo.PublicKey = publicKey.PublicKey()
	instanceInfo.ID = publicKey.GetID()
	if err := cm.saveInstanceSSHKey(instanceInfo.ID, instanceSSHKey); err != nil {
		log.Warnf("failed to persist SSH key for instance '%s': %s", instanceInfo.Name, err.Error())
	}

	p2pClient, err := cm.p2p.AddPeer(instanceInfo)
	if err != nil {
		_ = cm.p2p.RemovePeer(instanceInfo)
		return InstanceInfo{}, fmt.Errorf("failed to initialize instance: %w", err)
	}
	if p2pClient == nil {
		_ = cm.p2p.RemovePeer(instanceInfo)
		return InstanceInfo{}, errors.New("failed to initialize instance: p2p client is nil")
	}

	originSwarmionAddrs, closeBootstrapTunnel, err := cm.originSwarmionBootstrapAddrsViaSSH(sshCon, thisDevice.GetPublicKey(), instanceInfo.PublicIP)
	if err != nil {
		_ = cm.p2p.RemovePeer(instanceInfo)
		return InstanceInfo{}, fmt.Errorf("failed to create bootstrap tunnel: %w", err)
	}
	defer closeBootstrapTunnel()

	// do the initialization
	log.Infof("Initializing instance '%s'", instanceName)
	resp, err := p2pClient.Init(context.TODO(), &proto.InitRequest{
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
		PublicIP:      ipString,
		Name:          instanceName,
		Kind:          kind,
		KindID:        kindID,
		DesiredStatus: ServerStateRunning,
		Location:      locationName,
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
	defer sshCon.Close()

	// retrieve instance public key via SSH
	publicKeyPEM, err := waitForRemoteFile(sshCon, path.Join("/var/lib/protos/", pcrypto.PublicKeyFileName), 2*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to retrieve public key from instance: %w", err)
	}
	publicKey, err := pcrypto.CreatePublicKeyFromPEM(publicKeyPEM)
	if err != nil {
		return fmt.Errorf("failed to decode public key from instance: %w", err)
	}
	instanceInfo.PublicKey = publicKey.PublicKey()
	instanceInfo.ID = publicKey.GetID()

	p2pClient, err := cm.p2p.AddPeer(instanceInfo)
	if err != nil {
		return fmt.Errorf("failed to initialize instance: %w", err)
	}
	if p2pClient == nil {
		return errors.New("failed to initialize instance: p2p client is nil")
	}

	originSwarmionAddrs, closeBootstrapTunnel, err := cm.originSwarmionBootstrapAddrsViaSSH(sshCon, thisDevice.GetPublicKey(), instanceInfo.PublicIP)
	if err != nil {
		return fmt.Errorf("failed to create bootstrap tunnel: %w", err)
	}
	defer closeBootstrapTunnel()

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

	machineMapper, machineMetadataMapper := createInstanceInsertMapper(instanceInfo)
	err = db.Insert(cm.db, machineMapper, machineMetadataMapper, db.CreatePeerInsertMapper(instanceInfo.ID))
	if err != nil {
		return fmt.Errorf("failed to save instance '%s': %w", instanceName, err)
	}

	log.Infof("Instance '%s'(%s) initialized", instanceName, ipString)

	return nil
}

func waitForRemoteFile(sshCon *ssh.Client, remotePath string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	var lastOutput string

	for time.Now().Before(deadline) {
		output, err := pcrypto.ExecuteCommand(fmt.Sprintf("cat %s", remotePath), sshCon)
		if err == nil && strings.TrimSpace(output) != "" {
			return strings.TrimSpace(output), nil
		}
		lastErr = err
		lastOutput = strings.TrimSpace(output)
		time.Sleep(2 * time.Second)
	}

	diagnostics, _ := pcrypto.ExecuteCommand(
		"ls -la /var/lib /var/lib/protos /var/log 2>&1; cat /var/log/protos.log 2>&1 || true; ctr -n services.linuxkit tasks ls 2>&1 || true; ctr -n services.linuxkit containers ls 2>&1 || true",
		sshCon,
	)
	if lastErr != nil {
		return "", fmt.Errorf("timed out waiting for %s; last output: %q; last error: %w; diagnostics: %s", remotePath, lastOutput, lastErr, diagnostics)
	}
	return "", fmt.Errorf("timed out waiting for %s; last output: %q; diagnostics: %s", remotePath, lastOutput, diagnostics)
}

func (cm *Manager) originSwarmionBootstrapAddrs(originPublicKey string, peerPublicIP string) []string {
	ips := originBootstrapIPs(originPublicKey, peerPublicIP)
	addrs := cm.db.DialableListenMultiaddrs(ips)
	if len(addrs) == 0 {
		return cm.db.ListenMultiaddrs()
	}
	return addrs
}

func (cm *Manager) originSwarmionBootstrapAddrsViaSSH(sshCon *ssh.Client, originPublicKey string, peerPublicIP string) ([]string, func(), error) {
	addrs := cm.originSwarmionBootstrapAddrs(originPublicKey, peerPublicIP)
	noop := func() {}
	if sshCon == nil {
		return addrs, noop, nil
	}
	if config.Get().P2PPort <= 0 {
		return addrs, noop, nil
	}
	originKey, err := pcrypto.CreatePublicKeyFromBase64(originPublicKey)
	if err != nil {
		return addrs, noop, err
	}

	target := fmt.Sprintf("127.0.0.1:%d", config.Get().P2PPort+1)
	tunnel := pcrypto.NewReverseTunnel(sshCon, target)
	remotePort, err := tunnel.Start("127.0.0.1:0")
	if err != nil {
		return addrs, noop, err
	}

	tunnelAddr := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", remotePort, originKey.GetID())
	log.Debugf("Created bootstrap reverse SSH tunnel for Swarmion at '%s'", tunnelAddr)
	return append([]string{tunnelAddr}, addrs...), func() {
		if err := tunnel.Close(); err != nil {
			log.Warnf("Failed to close bootstrap reverse SSH tunnel: %s", err)
		}
	}, nil
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
			if strings.Contains(err.Error(), "not found") {
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
					return fmt.Errorf("could not stop instance '%s': %w", id, err)
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

	err = cm.p2p.RemovePeer(instance)
	if err != nil {
		return fmt.Errorf("failed to remove peer: %w", err)
	}

	im, cmmd := createInstanceDeleteMapper(instance.ID)
	err = db.Delete(cm.db, db.CreatePeerDeleteMapper(instance.ID), im, cmmd)
	if err != nil {
		return fmt.Errorf("failed to delete instance '%s': %w", id, err)
	}
	if err := cm.deleteInstanceSSHKey(instance.ID); err != nil {
		log.Warnf("failed to delete SSH key for instance '%s': %s", instance.Name, err.Error())
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

	computeProvider, err := requireComputeProvisioner(provider)
	if err != nil {
		return err
	}
	if err := provider.Init(); err != nil {
		return fmt.Errorf("could not init cloud '%s': %w", id, err)
	}

	_, providerInstanceID, err := getProviderInstanceInfo(computeProvider, instance)
	if err != nil {
		return fmt.Errorf("failed to get details for instance '%s': %w", id, err)
	}

	log.Infof("Starting instance '%s' (%s)", instance.Name, providerInstanceID)
	err = computeProvider.StartInstance(providerInstanceID, instance.Location)
	if err != nil {
		return fmt.Errorf("could not start instance '%s': %w", id, err)
	}

	// IP can change if an instance is stopped and started so a refresh is required
	info, err := computeProvider.GetInstanceInfo(providerInstanceID, instance.Location)
	if err != nil {
		return fmt.Errorf("could not retrieve instance info for '%s': %w", id, err)
	}

	instance.PublicIP = info.PublicIP
	instance.Volumes = info.Volumes
	instance.DesiredStatus = ServerStateRunning
	if instance.ProviderResourceID == "" {
		instance.ProviderResourceID = providerInstanceID
	}
	if err := cm.p2p.RequestReconnect(instance); err != nil {
		log.Debugf("failed to request p2p reconnect for instance '%s': %v", instance.Name, err)
	}

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

	computeProvider, err := requireComputeProvisioner(provider)
	if err != nil {
		return err
	}
	if err := provider.Init(); err != nil {
		return fmt.Errorf("could not init cloud '%s': %w", id, err)
	}

	_, providerInstanceID, err := getProviderInstanceInfo(computeProvider, instance)
	if err != nil {
		return fmt.Errorf("failed to get details for instance '%s': %w", id, err)
	}

	log.Infof("Stopping instance '%s' (%s)", instance.Name, providerInstanceID)
	err = computeProvider.StopInstance(providerInstanceID, instance.Location)
	if err != nil {
		return fmt.Errorf("could not stop instance '%s': %w", id, err)
	}
	instance.DesiredStatus = ServerStateStopped
	im, cmm := createInstanceUpdateMapper(instance)
	if err := db.Update(cm.db, im, cmm); err != nil {
		return fmt.Errorf("failed to save instance '%s': %w", id, err)
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
		instanceByName, nameErr := db.SelectOne(cm.db, createInstanceQueryByNameMapper(id))
		if nameErr != nil {
			return instance, fmt.Errorf("failed to retrieve instance: %w", err)
		}
		instance = instanceByName
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
		desiredStatus := normalizeDesiredInstanceStatus(instance.DesiredStatus)
		if desiredStatus == "" || instance.Kind == KindLocalVM {
			continue
		}

		provisioner, err := cm.GetProvisioner(instance.KindID)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		reconciler, ok := provisioner.(InstanceReconciler)
		if !ok {
			continue
		}
		if err := provisioner.Init(); err != nil {
			failures = append(failures, fmt.Sprintf("init provisioner %s: %v", instance.KindID, err))
			continue
		}

		instance.DesiredStatus = desiredStatus
		updated, err := reconciler.ReconcileInstance(instance)
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
