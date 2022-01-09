package cloud

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/auth"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/p2p"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/release"
	"github.com/protosio/protos/internal/util"
)

const (
	instanceDS = "instance"
	cloudDS    = "cloud"
	netSpace   = "10.100.0.0/16"
)

const (
	ServerStateRunning  = "running"
	ServerStateStopped  = "stopped"
	ServerStateOther    = "other"
	ServerStateChanging = "changing"

	protosPublicKey = "/var/protos/protos_key.pub"
)

type NetworkConfigurator interface {
	ConfigurePeers(instances []InstanceInfo, devices []auth.UserDevice) error
}

// InstanceInfo holds information about a cloud instance
type InstanceInfo struct {
	VMID          string
	Name          string
	SSHKeySeed    []byte // private SSH key stored only on the client
	PublicKey     []byte // ed25519 public key
	PublicIP      string
	InternalIP    string
	CloudType     string
	CloudName     string
	Location      string
	Network       string
	ProtosVersion string
	Status        string
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

func catchSignals(sigs chan os.Signal, quit chan interface{}) {
	<-sigs
	quit <- true
}

func createMachineTypesString(machineTypes map[string]MachineSpec) string {
	var machineTypesStr bytes.Buffer
	w := new(tabwriter.Writer)
	w.Init(&machineTypesStr, 8, 8, 0, ' ', 0)
	for instanceID, instanceSpec := range machineTypes {
		fmt.Fprintf(w, "    %s\t -  Nr of CPUs: %d,\t Memory: %d MiB,\t Storage: %d GB\t\n", instanceID, instanceSpec.Cores, instanceSpec.Memory, instanceSpec.DefaultStorage)
	}
	w.Flush()
	return machineTypesStr.String()
}

func removeNetworkFromSlice(s []net.IPNet, i int) []net.IPNet {
	networks := make([]net.IPNet, 0)
	networks = append(networks, s[:i]...)
	return append(networks, s[i+1:]...)
}

func copyIP(ip net.IP) net.IP {
	ipCopy := make(net.IP, len(ip))
	copy(ipCopy, ip)
	return ipCopy
}

func copyMask(mask net.IPMask) net.IPMask {
	maskCopy := make(net.IPMask, len(mask))
	copy(maskCopy, mask)
	return maskCopy
}

// allocateNetwork allocates an unused network for an instance
func allocateNetwork(instances []InstanceInfo, devices []auth.UserDevice) (net.IPNet, error) {
	// create list of existing networks
	usedNetworks := []net.IPNet{}
	for _, inst := range instances {
		_, inet, err := net.ParseCIDR(inst.Network)
		if err != nil {
			return net.IPNet{}, err
		}
		usedNetworks = append(usedNetworks, *inet)
	}
	for _, dev := range devices {
		_, inet, err := net.ParseCIDR(dev.Network)
		if err != nil {
			return net.IPNet{}, err
		}
		usedNetworks = append(usedNetworks, *inet)
	}

	// figure out which is the first network that is not currently used
	allNetworks := []net.IPNet{}
	_, netspace, _ := net.ParseCIDR(netSpace)
	for i := 0; i <= 255; i++ {
		newNet := net.IPNet{}
		newNet.IP = copyIP(netspace.IP)
		newNet.Mask = copyMask(netspace.Mask)
		newNet.IP[2] = byte(i)
		newNet.Mask[2] = byte(255)
		allNetworks = append(allNetworks, newNet)
	}
	for _, usedNet := range usedNetworks {
		for i, network := range allNetworks {
			if usedNet.IP.String() == network.IP.String() && usedNet.Mask.String() == network.Mask.String() {
				allNetworks = removeNetworkFromSlice(allNetworks, i)
				break
			}
		}
	}
	if len(allNetworks) == 0 {
		return net.IPNet{}, fmt.Errorf("failed to allocate network. Maximum number of networks allocated")
	}

	return allNetworks[0], nil
}

// Manager manages cloud providers and instances
type Manager struct {
	db                  db.DB
	um                  *auth.UserManager
	sm                  *pcrypto.Manager
	p2p                 *p2p.P2P
	networkConfigurator NetworkConfigurator
}

//
// Cloud related methods
//

// SupportedProviders returns a list of supported cloud providers
func (cm *Manager) SupportedProviders() []string {
	return []string{Scaleway.String()}
}

// GetProvider returns a cloud provider instance from the db
func (cm *Manager) GetProvider(name string) (CloudProvider, error) {
	clouds := map[string]ProviderInfo{}
	err := cm.db.GetMap(cloudDS, &clouds)
	if err != nil {
		return nil, err
	}
	for _, cld := range clouds {
		if cld.Name == name {
			cld.cm = cm
			return cld.getCloudProvider()
		}
	}
	return nil, fmt.Errorf("could not find cloud provider '%s'", name)
}

// DeleteProvider deletes a cloud provider from the db
func (cm *Manager) DeleteProvider(name string) error {
	cld, err := cm.GetProvider(name)
	if err != nil {
		return err
	}

	providerInfo, ok := cld.(*ProviderInfo)
	if !ok {
		panic("Failed type assertion in delete provider")
	}

	err = cm.db.RemoveFromMap(cloudDS, providerInfo.Name)
	if err != nil {
		return err
	}
	return nil
}

// GetProviders returns all the cloud providers from the db
func (cm *Manager) GetProviders() ([]CloudProvider, error) {
	cloudProviders := []CloudProvider{}
	clouds := map[string]ProviderInfo{}
	err := cm.db.GetMap(cloudDS, &clouds)
	if err != nil {
		return cloudProviders, fmt.Errorf("failed to retrieve cloud provides")
	}

	for _, cloud := range clouds {
		client, err := cloud.getCloudProvider()
		if err != nil {
			return cloudProviders, err
		}
		cloudProviders = append(cloudProviders, client)
	}

	return cloudProviders, nil
}

// NewProvider creates and returns a cloud provider. At this point it is not saved in the db
func (cm *Manager) NewProvider(cloudName string, cloud string) (CloudProvider, error) {
	cloudType := Type(cloud)
	cld := ProviderInfo{Name: cloudName, Type: cloudType, cm: cm}
	return cld.getCloudProvider()
}

//
// Instance related methods
//

// DeployInstance deploys an instance on the provided cloud
func (cm *Manager) DeployInstance(instanceName string, cloudName string, cloudLocation string, release release.Release, machineType string) (InstanceInfo, error) {
	usr, err := cm.um.GetAdmin()
	if err != nil {
		return InstanceInfo{}, err
	}

	// init cloud
	provider, err := cm.GetProvider(cloudName)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("could not retrieve cloud '%s': %w", cloudName, err)
	}
	err = provider.Init()
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to init cloud provider '%s'(%s) API: %w", cloudName, provider.TypeStr(), err)
	}

	// validate machine type
	supportedMachineTypes, err := provider.SupportedMachines(cloudLocation)
	if err != nil {
		return InstanceInfo{}, err
	}
	if _, found := supportedMachineTypes[machineType]; !found {
		return InstanceInfo{}, errors.Errorf("Machine type '%s' is not valid for cloud provider '%s'. The following types are supported: \n%s", machineType, provider.TypeStr(), createMachineTypesString(supportedMachineTypes))
	}

	// add image
	imageID := ""
	images, err := provider.GetImages()
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
			imageID, err = provider.AddImage(image.URL, image.Digest, release.Version, cloudLocation)
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
	vmID, err := provider.NewInstance(instanceName, imageID, instanceSSHKey.AuthorizedKey(), machineType, cloudLocation)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to deploy Protos instance: %w", err)
	}
	log.Infof("Instance with ID '%s' deployed", vmID)

	// get instance info
	instanceInfo, err := provider.GetInstanceInfo(vmID, cloudLocation)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to get Protos instance info: %w", err)
	}

	// allocate network
	instances, err := cm.GetInstances()
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to allocate network for instance '%s': %w", instanceInfo.Name, err)
	}
	network, err := allocateNetwork(instances, usr.GetDevices())
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to allocate network for instance '%s': %w", instanceInfo.Name, err)
	}

	// save instance information
	instanceInfo.SSHKeySeed = instanceSSHKey.Seed()
	instanceInfo.ProtosVersion = release.Version
	instanceInfo.Network = network.String()
	err = cm.db.InsertInMap(instanceDS, instanceInfo.Name, instanceInfo)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to save instance '%s': %w", instanceName, err)
	}

	// create protos data volume
	log.Infof("creating data volume for Protos instance '%s'", instanceName)
	volumeID, err := provider.NewVolume(instanceName, 30000, cloudLocation)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to create data volume: %w", err)
	}

	// attach volume to instance
	err = provider.AttachVolume(volumeID, vmID, cloudLocation)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to attach volume to instance '%s': %w", instanceName, err)
	}

	// start protos instance
	log.Infof("Starting instance '%s'", instanceName)
	err = provider.StartInstance(vmID, cloudLocation)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to start instance: %w", err)
	}

	// get instance info again
	instanceUpdate, err := provider.GetInstanceInfo(vmID, cloudLocation)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to get instance info: %w", err)
	}
	instanceInfo.PublicIP = instanceUpdate.PublicIP
	instanceInfo.Volumes = instanceUpdate.Volumes
	// second save of the instance information
	err = cm.db.InsertInMap(instanceDS, instanceInfo.Name, instanceInfo)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to save instance '%s': %w", instanceName, err)
	}

	// wait for port 22 to be open
	err = util.WaitForPort(instanceInfo.PublicIP, "22", 20)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to deploy instance: %w", err)
	}

	key, err := cm.sm.NewKeyFromSeed(instanceInfo.SSHKeySeed)
	if err != nil {
		return InstanceInfo{}, err
	}

	// connect via SSH
	sshCon, err := pcrypto.NewConnection(instanceInfo.PublicIP, "root", key.SSHAuth(), 10)
	if err != nil {
		return InstanceInfo{}, err
	}

	// retrieve instance public key via SSH
	pubKeyStr, err := pcrypto.ExecuteCommand(fmt.Sprintf("cat %s", protosPublicKey), sshCon)
	if err != nil {
		return InstanceInfo{}, err
	}

	// close SSH connection
	sshCon.Close()

	var pubKey ed25519.PublicKey
	pubKey, err = base64.StdEncoding.DecodeString(pubKeyStr)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to decode public key: %w", err)
	}
	instanceInfo.PublicKey = pubKey

	dev, err := usr.GetCurrentDevice()
	if err != nil {
		return InstanceInfo{}, err
	}

	p2pClient, err := cm.p2p.AddPeer(pubKey, instanceInfo.PublicIP)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to add peer: %w", err)
	}

	// do the initialization
	log.Infof("Initializing instance '%s'", instanceName)
	ip, pubKey, err := p2pClient.Init(usr.GetUsername(), usr.GetPassword(), usr.GetInfo().Name, instanceName, instanceInfo.Network, []auth.UserDevice{dev})
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to initialize instance: %w", err)
	}

	// final save instance info
	instanceInfo.InternalIP = ip.String()
	instanceInfo.PublicKey = pubKey
	err = cm.db.InsertInMap(instanceDS, instanceInfo.Name, instanceInfo)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to save instance '%s': %w", instanceName, err)
	}

	log.Infof("Instance '%s' at '%s' is ready", instanceName, instanceInfo.PublicIP)

	return instanceInfo, nil
}

// InitDevInstance initializes an existing instance, without deploying one. Used for development purposes
func (cm *Manager) InitDevInstance(instanceName string, cloudName string, locationName string, keyFile string, ipString string) error {
	instanceInfo := InstanceInfo{
		VMID:          instanceName,
		PublicIP:      ipString,
		Name:          instanceName,
		CloudType:     Local.String(),
		CloudName:     cloudName,
		Location:      locationName,
		ProtosVersion: "dev",
	}

	ip := net.ParseIP(ipString)
	if ip == nil {
		return fmt.Errorf("String '%s' is not a valid IP address", ipString)
	}

	// we use a local key because we don't have a dedicated SSH key for the dev instance
	sshAuth, err := cm.sm.NewAuthFromKeyFile(keyFile)
	if err != nil {
		return err
	}

	// allocate network for dev instance
	instances, err := cm.GetInstances()
	if err != nil {
		return err
	}
	usr, err := cm.um.GetAdmin()
	if err != nil {
		return err
	}
	developmentNetwork, err := allocateNetwork(instances, usr.GetDevices())
	if err != nil {
		return fmt.Errorf("failed to allocate network for instance '%s': %w", "dev", err)
	}

	// wait for port 22 to be open
	err = util.WaitForPort(instanceInfo.PublicIP, "22", 20)
	if err != nil {
		return fmt.Errorf("failure while waiting for port: %w", err)
	}

	// connect via SSH
	sshCon, err := pcrypto.NewConnection(instanceInfo.PublicIP, "root", sshAuth, 10)
	if err != nil {
		return fmt.Errorf("failed to connect to dev instance over SSH: %w", err)
	}

	// retrieve instance public key via SSH
	pubKeyStr, err := pcrypto.ExecuteCommand(fmt.Sprintf("cat %s", protosPublicKey), sshCon)
	if err != nil {
		return fmt.Errorf("failed to retrieve public key from dev instance: %w", err)
	}

	// close SSH connection
	sshCon.Close()

	var pubKey ed25519.PublicKey
	pubKey, err = base64.StdEncoding.DecodeString(pubKeyStr)
	if err != nil {
		return fmt.Errorf("failed to decode public key: %w", err)
	}
	instanceInfo.PublicKey = pubKey

	dev, err := usr.GetCurrentDevice()
	if err != nil {
		return err
	}

	p2pClient, err := cm.p2p.AddPeer(pubKey, instanceInfo.PublicIP)
	if err != nil {
		return fmt.Errorf("failed to add peer: %w", err)
	}

	// do the initialization
	log.Infof("Initializing instance '%s'", instanceName)
	ip, _, err = p2pClient.Init(usr.GetUsername(), usr.GetPassword(), usr.GetInfo().Name, instanceName, developmentNetwork.String(), []auth.UserDevice{dev})
	if err != nil {
		return fmt.Errorf("failed to init dev instance: %w", err)
	}

	instanceInfo.InternalIP = ip.String()
	instanceInfo.PublicKey = pubKey
	instanceInfo.Network = developmentNetwork.String()
	err = cm.db.InsertInMap(instanceDS, instanceInfo.Name, instanceInfo)
	if err != nil {
		return fmt.Errorf("failed to save dev instance '%s': %w", instanceName, err)
	}

	log.Infof("Dev instance '%s' at '%s' is ready", instanceName, ipString)

	return nil
}

// DeleteInstance deletes an instance
func (cm *Manager) DeleteInstance(name string) error {
	instance, err := cm.GetInstance(name)
	if err != nil {
		return fmt.Errorf("could not retrieve instance '%s': %w", name, err)
	}

	// if local only, ignore any cloud resources
	if instance.CloudType != string(Local) {
		provider, err := cm.GetProvider(instance.CloudName)
		if err != nil {
			return fmt.Errorf("could not retrieve cloud '%s': %w", name, err)
		}

		err = provider.Init()
		if err != nil {
			return fmt.Errorf("could not init cloud '%s': %w", name, err)
		}

		found := true
		vmInfo, err := provider.GetInstanceInfo(instance.VMID, instance.Location)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				found = false
			} else {
				return fmt.Errorf("failed to get details for instance '%s': %w", name, err)
			}
		}

		// only delete cloud instance if found. Otherwise we proceed with removing it from local db
		if found {
			if vmInfo.Status == ServerStateRunning {
				log.Infof("Stopping instance '%s' (%s)", instance.Name, instance.VMID)
				err = provider.StopInstance(instance.VMID, instance.Location)
				if err != nil {
					return fmt.Errorf("could not stop instance '%s': %w", name, err)
				}
			}
			log.Infof("Deleting instance '%s' (%s)", instance.Name, instance.VMID)
			err = provider.DeleteInstance(instance.VMID, instance.Location)
			if err != nil {
				return fmt.Errorf("could not delete instance '%s': %w", name, err)
			}
			for _, vol := range vmInfo.Volumes {
				log.Infof("Deleting volume '%s' (%s) for instance '%s'", vol.Name, vol.VolumeID, name)
				err = provider.DeleteVolume(vol.VolumeID, instance.Location)
				if err != nil {
					log.Errorf("failed to delete volume '%s': %s", vol.Name, err.Error())
				}
			}
		}
	}

	err = cm.p2p.RemovePeer(instance.PublicKey)
	if err != nil {
		log.Warnf("Failed to remove peer for instance '%s': %s", instance.Name, err.Error())
	}

	return cm.db.RemoveFromMap(instanceDS, instance.Name)
}

// StartInstance starts an instance
func (cm *Manager) StartInstance(name string) error {
	instance, err := cm.GetInstance(name)
	if err != nil {
		return fmt.Errorf("could not retrieve instance '%s': %w", name, err)
	}
	provider, err := cm.GetProvider(instance.CloudName)
	if err != nil {
		return fmt.Errorf("could not retrieve cloud '%s': %w", name, err)
	}

	err = provider.Init()
	if err != nil {
		return fmt.Errorf("could not init cloud '%s': %w", name, err)
	}

	log.Infof("Starting instance '%s' (%s)", instance.Name, instance.VMID)
	err = provider.StartInstance(instance.VMID, instance.Location)
	if err != nil {
		return fmt.Errorf("could not start instance '%s': %w", name, err)
	}

	// IP can change if an instance is stopped and started so a refresh is required
	info, err := provider.GetInstanceInfo(instance.VMID, instance.Location)
	if err != nil {
		return fmt.Errorf("could not retrieve instance info for '%s': %w", name, err)
	}

	instance.PublicIP = info.PublicIP
	instance.Volumes = info.Volumes

	err = cm.db.InsertInMap(instanceDS, instance.Name, instance)
	if err != nil {
		return fmt.Errorf("failed to save instance '%s': %w", name, err)
	}

	return nil
}

// StopInstance stops an instance
func (cm *Manager) StopInstance(name string) error {
	instance, err := cm.GetInstance(name)
	if err != nil {
		return fmt.Errorf("could not retrieve instance '%s': %w", name, err)
	}
	provider, err := cm.GetProvider(instance.CloudName)
	if err != nil {
		return fmt.Errorf("could not retrieve cloud '%s': %w", name, err)
	}

	err = provider.Init()
	if err != nil {
		return fmt.Errorf("could not init cloud '%s': %w", name, err)
	}

	log.Infof("Stopping instance '%s' (%s)", instance.Name, instance.VMID)
	err = provider.StopInstance(instance.VMID, instance.Location)
	if err != nil {
		return fmt.Errorf("could not stop instance '%s': %w", name, err)
	}
	return nil
}

// TunnelInstance creates and SSH tunnel to the instance
func (cm *Manager) TunnelInstance(name string) error {
	instanceInfo, err := cm.GetInstance(name)
	if err != nil {
		return fmt.Errorf("could not retrieve instance '%s': %w", name, err)
	}
	if len(instanceInfo.SSHKeySeed) == 0 {
		return errors.Errorf("Instance '%s' is missing its SSH key", name)
	}
	key, err := cm.sm.NewKeyFromSeed(instanceInfo.SSHKeySeed)
	if err != nil {
		return fmt.Errorf("instance '%s' has an invalid SSH key: %w", name, err)
	}

	log.Infof("creating SSH tunnel to instance '%s', using ip '%s'", instanceInfo.Name, instanceInfo.PublicIP)
	tunnel := pcrypto.NewTunnel(instanceInfo.PublicIP+":22", "root", key.SSHAuth(), "localhost:8080")
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

// LogsInstance retrieves the Protos logs from an instance
func (cm *Manager) LogsInstance(name string) (string, error) {
	instanceInfo, err := cm.GetInstance(name)
	if err != nil {
		return "", err
	}
	if len(instanceInfo.SSHKeySeed) == 0 {
		return "", err
	}
	key, err := cm.sm.NewKeyFromSeed(instanceInfo.SSHKeySeed)
	if err != nil {
		return "", err
	}

	sshCon, err := pcrypto.NewConnection(instanceInfo.PublicIP, "root", key.SSHAuth(), 10)
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
func (cm *Manager) GetInstance(name string) (InstanceInfo, error) {
	instances := map[string]InstanceInfo{}
	err := cm.db.GetMap(instanceDS, &instances)
	if err != nil {
		return InstanceInfo{}, err
	}

	for _, instance := range instances {
		if instance.Name == name {
			// if not local, we update the instance status
			if instance.CloudName != Local.String() {
				provider, err := cm.GetProvider(instance.CloudName)
				if err != nil {
					return InstanceInfo{}, err
				}
				err = provider.Init()
				if err != nil {
					return InstanceInfo{}, err
				}
				instanceInfo, err := provider.GetInstanceInfo(instance.VMID, instance.Location)
				if err != nil {
					log.Warn(err.Error())
					instance.Status = "n/a"
				} else {
					instance.Status = instanceInfo.Status
				}
			}
			return instance, nil
		}
	}
	return InstanceInfo{}, fmt.Errorf("could not find instance '%s'", name)
}

// GetInstances returns all the instances from the db
func (cm *Manager) GetInstances() ([]InstanceInfo, error) {
	instanceMap := map[string]InstanceInfo{}
	var instances []InstanceInfo
	err := cm.db.GetMap(instanceDS, &instanceMap)
	if err != nil {
		return instances, err
	}
	for _, instance := range instanceMap {
		instances = append(instances, instance)
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

	err = provider.Init()
	if err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}

	// find image
	images, err := provider.GetImages()
	if err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}
	for _, img := range images {
		if img.Location == cloudLocation && img.Name == imageName {
			return fmt.Errorf("%s: Found an image with the same name", errMsg)
		}
	}

	// upload image
	_, err = provider.UploadLocalImage(imagePath, imageName, cloudLocation, timeout)
	if err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}
	return nil
}

func (cm *Manager) Refresh() error {

	instances, err := cm.GetInstances()
	if err != nil {
		return fmt.Errorf("failed to retrieve instances: %w", err)
	}

	admin, err := cm.um.GetAdmin()
	if err != nil {
		return fmt.Errorf("failed to retrieve admin user: %w", err)
	}

	err = cm.networkConfigurator.ConfigurePeers(instances, admin.GetDevices())
	if err != nil {
		return fmt.Errorf("failed to configure network peers: %w", err)
	}

	return nil
}

// CreateManager creates and returns a cloud manager
func CreateManager(db db.DB, um *auth.UserManager, sm *pcrypto.Manager, p2p *p2p.P2P, networkConfiguration NetworkConfigurator, selfName string) (*Manager, error) {
	if db == nil || um == nil || sm == nil || p2p == nil {
		return nil, fmt.Errorf("failed to create cloud manager: none of the inputs can be nil")
	}

	manager := &Manager{db: db, um: um, sm: sm, p2p: p2p, networkConfigurator: networkConfiguration}

	err := db.InitDataset(instanceDS, manager)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize instance dataset: %w", err)
	}

	err = db.InitDataset(cloudDS, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cloud dataset: %w", err)
	}

	instances, err := manager.GetInstances()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve instances: %w", err)
	}
	for _, instance := range instances {
		if instance.Name == selfName {
			continue
		}
		log.Debugf("Connecting to instance '%s'(cloud: %s, IP: %s)", instance.Name, instance.CloudName, instance.PublicIP)
		_, err := manager.p2p.AddPeer(instance.PublicKey, instance.PublicIP)
		if err != nil {
			log.Errorf("Failed to add peer: %s", err.Error())
			continue
		}
	}

	// manager.db.SyncAll()
	return manager, nil
}
