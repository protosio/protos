package cloud

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/core"
	"github.com/protosio/protos/internal/release"
	"github.com/protosio/protos/internal/ssh"
	pclient "github.com/protosio/protos/pkg/client"
	"github.com/protosio/protos/pkg/types"
)

const (
	instanceDS = "instance"
	cloudDS    = "cloud"
	netSpace   = "10.100.0.0/16"
)

func catchSignals(sigs chan os.Signal, quit chan interface{}) {
	<-sigs
	quit <- true
}

func createMachineTypesString(machineTypes map[string]core.MachineSpec) string {
	var machineTypesStr bytes.Buffer
	w := new(tabwriter.Writer)
	w.Init(&machineTypesStr, 8, 8, 0, ' ', 0)
	for instanceID, instanceSpec := range machineTypes {
		fmt.Fprintf(w, "    %s\t -  Nr of CPUs: %d,\t Memory: %d MiB,\t Storage: %d GB\t\n", instanceID, instanceSpec.Cores, instanceSpec.Memory, instanceSpec.DefaultStorage)
	}
	w.Flush()
	return machineTypesStr.String()
}

// AllocateNetwork allocates an unused network for an instance
func AllocateNetwork(instances []core.InstanceInfo, devices []types.UserDevice) (net.IPNet, error) {
	// create list of existing networks
	usedNetworks := []net.IPNet{}
	for _, inst := range instances {
		_, inet, err := net.ParseCIDR(inst.Network)
		if err != nil {
			panic(err)
		}
		usedNetworks = append(usedNetworks, *inet)
	}
	for _, dev := range devices {
		_, inet, err := net.ParseCIDR(dev.Network)
		if err != nil {
			panic(err)
		}
		usedNetworks = append(usedNetworks, *inet)
	}

	// figure out which is the first network that is not currently used
	_, netspace, _ := net.ParseCIDR(netSpace)
	for i := 0; i <= 255; i++ {
		newNet := *netspace
		newNet.IP[2] = byte(i)
		newNet.Mask[2] = byte(255)
		for _, usedNet := range usedNetworks {
			if !newNet.Contains(usedNet.IP) {
				return newNet, nil
			}
		}
	}

	return net.IPNet{}, fmt.Errorf("Failed to allocate network")
}

// Manager manages cloud providers and instances
type Manager struct {
	db core.DB
	um core.UserManager
	sm core.SSHManager
}

//
// Cloud related methods
//

// SupportedProviders returns a list of supported cloud providers
func (cm *Manager) SupportedProviders() []string {
	return []string{Scaleway.String()}
}

// GetProvider returns a cloud provider instance from the db
func (cm *Manager) GetProvider(name string) (core.CloudProvider, error) {
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
	return nil, fmt.Errorf("Could not find cloud provider '%s'", name)
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
func (cm *Manager) GetProviders() ([]core.CloudProvider, error) {
	cloudProviders := []core.CloudProvider{}
	clouds := map[string]ProviderInfo{}
	err := cm.db.GetMap(cloudDS, &clouds)
	if err != nil {
		return cloudProviders, fmt.Errorf("Failed to retrieve cloud provides")
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
func (cm *Manager) NewProvider(cloudName string, cloud string) (core.CloudProvider, error) {
	cloudType := Type(cloud)
	cld := ProviderInfo{Name: cloudName, Type: cloudType, cm: cm}
	return cld.getCloudProvider()
}

//
// Instance related methods
//

// DeployInstance deploys an instance on the provided cloud
func (cm *Manager) DeployInstance(instanceName string, cloudName string, cloudLocation string, release release.Release, machineType string) (core.InstanceInfo, error) {
	usr, err := cm.um.GetAdmin()
	if err != nil {
		return core.InstanceInfo{}, err
	}

	// init cloud
	provider, err := cm.GetProvider(cloudName)
	if err != nil {
		return core.InstanceInfo{}, errors.Wrapf(err, "Could not retrieve cloud '%s'", cloudName)
	}
	err = provider.Init()
	if err != nil {
		return core.InstanceInfo{}, errors.Wrapf(err, "Failed to init cloud provider '%s'(%s) API", cloudName, provider.TypeStr())
	}

	// validate machine type
	supportedMachineTypes, err := provider.SupportedMachines(cloudLocation)
	if err != nil {
		return core.InstanceInfo{}, err
	}
	if _, found := supportedMachineTypes[machineType]; !found {
		return core.InstanceInfo{}, errors.Errorf("Machine type '%s' is not valid for cloud provider '%s'. The following types are supported: \n%s", machineType, provider.TypeStr(), createMachineTypesString(supportedMachineTypes))
	}

	// add image
	imageID := ""
	images, err := provider.GetImages()
	if err != nil {
		return core.InstanceInfo{}, errors.Wrap(err, "Failed to deploy Protos instance")
	}
	for id, img := range images {
		if img.Location == cloudLocation && img.Name == release.Version {
			imageID = id
			break
		}
	}
	if imageID != "" {
		log.Infof("Found Protos image version '%s' in your cloud account", release.Version)
	} else {
		// upload protos image
		if image, found := release.CloudImages[provider.TypeStr()]; found {
			log.Infof("Protos image version '%s' not in your infra cloud account. Adding it.", release.Version)
			imageID, err = provider.AddImage(image.URL, image.Digest, release.Version, cloudLocation)
			if err != nil {
				return core.InstanceInfo{}, errors.Wrap(err, "Failed to deploy Protos instance")
			}
		} else {
			return core.InstanceInfo{}, errors.Errorf("Could not find a Protos version '%s' release for cloud '%s'", release.Version, provider.TypeStr())
		}
	}

	// create SSH key used for instance
	log.Info("Generating SSH key for the new VM instance")
	instanceSSHKey, err := cm.sm.GenerateKey()
	if err != nil {
		return core.InstanceInfo{}, errors.Wrap(err, "Failed to deploy Protos instance")
	}

	// deploy a protos instance
	log.Infof("Deploying instance '%s' of type '%s', using Protos version '%s' (image id '%s')", instanceName, machineType, release.Version, imageID)
	vmID, err := provider.NewInstance(instanceName, imageID, instanceSSHKey.AuthorizedKey(), machineType, cloudLocation)
	if err != nil {
		return core.InstanceInfo{}, errors.Wrap(err, "Failed to deploy Protos instance")
	}
	log.Infof("Instance with ID '%s' deployed", vmID)

	// get instance info
	instanceInfo, err := provider.GetInstanceInfo(vmID, cloudLocation)
	if err != nil {
		return core.InstanceInfo{}, errors.Wrap(err, "Failed to get Protos instance info")
	}

	// allocate network
	instances, err := cm.GetInstances()
	if err != nil {
		return core.InstanceInfo{}, fmt.Errorf("Failed to allocate network for instance '%s': %w", instanceInfo.Name, err)
	}
	network, err := AllocateNetwork(instances, usr.GetDevices())
	if err != nil {
		return core.InstanceInfo{}, fmt.Errorf("Failed to allocate network for instance '%s': %w", instanceInfo.Name, err)
	}

	// save instance information
	instanceInfo.KeySeed = instanceSSHKey.Seed()
	instanceInfo.ProtosVersion = release.Version
	instanceInfo.Network = network.String()
	err = cm.db.InsertInMap(instanceDS, instanceInfo.Name, instanceInfo)
	if err != nil {
		return core.InstanceInfo{}, errors.Wrapf(err, "Failed to save instance '%s'", instanceName)
	}

	// create protos data volume
	log.Infof("Creating data volume for Protos instance '%s'", instanceName)
	volumeID, err := provider.NewVolume(instanceName, 30000, cloudLocation)
	if err != nil {
		return core.InstanceInfo{}, errors.Wrap(err, "Failed to create data volume")
	}

	// attach volume to instance
	err = provider.AttachVolume(volumeID, vmID, cloudLocation)
	if err != nil {
		return core.InstanceInfo{}, errors.Wrapf(err, "Failed to attach volume to instance '%s'", instanceName)
	}

	// start protos instance
	log.Infof("Starting Protos instance '%s'", instanceName)
	err = provider.StartInstance(vmID, cloudLocation)
	if err != nil {
		return core.InstanceInfo{}, errors.Wrap(err, "Failed to start Protos instance")
	}

	// get instance info again
	instanceUpdate, err := provider.GetInstanceInfo(vmID, cloudLocation)
	if err != nil {
		return core.InstanceInfo{}, errors.Wrap(err, "Failed to get Protos instance info")
	}
	instanceInfo.PublicIP = instanceUpdate.PublicIP
	instanceInfo.Volumes = instanceUpdate.Volumes
	// second save of the instance information
	err = cm.db.InsertInMap(instanceDS, instanceInfo.Name, instanceInfo)
	if err != nil {
		return core.InstanceInfo{}, errors.Wrapf(err, "Failed to save instance '%s'", instanceName)
	}

	// wait for port 22 to be open
	err = WaitForPort(instanceInfo.PublicIP, "22", 20)
	if err != nil {
		return core.InstanceInfo{}, errors.Wrap(err, "Failed to deploy instance")
	}

	// allow some time for Protosd to start up, or else the tunnel might fail
	time.Sleep(5 * time.Second)

	log.Infof("Creating SSH tunnel to instance '%s'", instanceName)
	tunnel := ssh.NewTunnel(instanceInfo.PublicIP+":22", "root", instanceSSHKey.SSHAuth(), "localhost:8080")
	localPort, err := tunnel.Start()
	if err != nil {
		return core.InstanceInfo{}, errors.Wrap(err, "Error while creating the SSH tunnel")
	}

	// wait for the API to be up
	err = WaitForHTTP(fmt.Sprintf("http://127.0.0.1:%d/ui/", localPort), 20)
	if err != nil {
		return core.InstanceInfo{}, errors.Wrap(err, "Failed to deploy instance")
	}
	log.Infof("Tunnel to '%s' ready", instanceName)

	// do the initialization
	log.Infof("Initializing instance '%s'", instanceName)
	protos := pclient.NewInitClient(fmt.Sprintf("127.0.0.1:%d", localPort), usr.GetUsername(), usr.GetPassword())
	dev, err := usr.GetCurrentDevice()
	if err != nil {
		return core.InstanceInfo{}, err
	}

	ip, pubKey, err := protos.InitInstance(usr.GetInfo().Name, instanceInfo.Network, usr.GetInfo().Domain, []types.UserDevice{dev})
	if err != nil {
		return core.InstanceInfo{}, errors.Wrap(err, "Error while doing the instance initialization")
	}

	// final save instance info
	instanceInfo.InternalIP = ip.String()
	instanceInfo.PublicKey = pubKey
	err = cm.db.InsertInMap(instanceDS, instanceInfo.Name, instanceInfo)
	if err != nil {
		return core.InstanceInfo{}, errors.Wrapf(err, "Failed to save instance '%s'", instanceName)
	}

	// close the SSH tunnel
	err = tunnel.Close()
	if err != nil {
		return core.InstanceInfo{}, errors.Wrap(err, "Error while terminating the SSH tunnel")
	}
	log.Infof("Instance '%s' is ready", instanceName)

	return instanceInfo, nil
}

// InitDevInstance initializes an existing instance, without deploying one. Used for development purposes
func (cm *Manager) InitDevInstance(instanceName string, cloudName string, locationName string, keyFile string, ipString string) error {
	instanceInfo := core.InstanceInfo{
		VMID:          instanceName,
		PublicIP:      ipString,
		Name:          instanceName,
		CloudType:     Hyperkit.String(),
		CloudName:     cloudName,
		Location:      locationName,
		ProtosVersion: "dev",
	}

	ip := net.ParseIP(ipString)
	if ip == nil {
		return fmt.Errorf("String '%s' is not a valid IP address", ipString)
	}

	auth, err := cm.sm.NewAuthFromKeyFile(keyFile)
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
	developmentNetwork, err := AllocateNetwork(instances, usr.GetDevices())
	if err != nil {
		return fmt.Errorf("Failed to allocate network for instance '%s': %w", "dev", err)
	}

	log.Infof("Creating SSH tunnel to dev instance IP '%s'", ipString)
	tunnel := ssh.NewTunnel(ip.String()+":22", "root", auth, "localhost:8080")
	localPort, err := tunnel.Start()
	if err != nil {
		return errors.Wrap(err, "Error while creating the SSH tunnel")
	}

	// wait for the API to be up
	err = WaitForHTTP(fmt.Sprintf("http://127.0.0.1:%d/ui/", localPort), 20)
	if err != nil {
		return errors.Wrap(err, "Failed to deploy instance")
	}
	log.Infof("Tunnel to '%s' ready", ipString)

	// do the initialization
	log.Infof("Initializing instance at '%s'", ipString)
	protos := pclient.NewInitClient(fmt.Sprintf("127.0.0.1:%d", localPort), usr.GetUsername(), usr.GetPassword())
	dev, err := usr.GetCurrentDevice()
	if err != nil {
		return err
	}

	// Doing the instance initialization which returns the internal wireguard IP and the public key created using the wireguard library.
	instanceIP, instancePublicKey, err := protos.InitInstance(usr.GetInfo().Name, developmentNetwork.String(), usr.GetInfo().Domain, []types.UserDevice{dev})
	if err != nil {
		return errors.Wrap(err, "Error while doing the instance initialization")
	}
	instanceInfo.InternalIP = instanceIP.String()
	instanceInfo.PublicKey = instancePublicKey
	instanceInfo.Network = developmentNetwork.String()

	err = cm.db.InsertInMap(instanceDS, instanceInfo.Name, instanceInfo)
	if err != nil {
		return errors.Wrapf(err, "Failed to save dev instance '%s'", instanceName)
	}

	// close the SSH tunnel
	err = tunnel.Close()
	if err != nil {
		return errors.Wrap(err, "Error while terminating the SSH tunnel")
	}
	log.Infof("Instance at '%s' is ready", ipString)

	return nil
}

// DeleteInstance deletes an instance
func (cm *Manager) DeleteInstance(name string, localOnly bool) error {
	instance, err := cm.GetInstance(name)
	if err != nil {
		return errors.Wrapf(err, "Could not retrieve instance '%s'", name)
	}

	// if local only, ignore any cloud resources
	if !localOnly {
		provider, err := cm.GetProvider(instance.CloudName)
		if err != nil {
			return errors.Wrapf(err, "Could not retrieve cloud '%s'", name)
		}

		err = provider.Init()
		if err != nil {
			return errors.Wrapf(err, "Could not init cloud '%s'", name)
		}

		log.Infof("Stopping instance '%s' (%s)", instance.Name, instance.VMID)
		err = provider.StopInstance(instance.VMID, instance.Location)
		if err != nil {
			return errors.Wrapf(err, "Could not stop instance '%s'", name)
		}
		vmInfo, err := provider.GetInstanceInfo(instance.VMID, instance.Location)
		if err != nil {
			return errors.Wrapf(err, "Failed to get details for instance '%s'", name)
		}
		log.Infof("Deleting instance '%s' (%s)", instance.Name, instance.VMID)
		err = provider.DeleteInstance(instance.VMID, instance.Location)
		if err != nil {
			return errors.Wrapf(err, "Could not delete instance '%s'", name)
		}
		for _, vol := range vmInfo.Volumes {
			log.Infof("Deleting volume '%s' (%s) for instance '%s'", vol.Name, vol.VolumeID, name)
			err = provider.DeleteVolume(vol.VolumeID, instance.Location)
			if err != nil {
				log.Errorf("Failed to delete volume '%s': %s", vol.Name, err.Error())
			}
		}
	}
	return cm.db.RemoveFromMap(instanceDS, instance.Name)
}

// StartInstance starts an instance
func (cm *Manager) StartInstance(name string) error {
	instance, err := cm.GetInstance(name)
	if err != nil {
		return errors.Wrapf(err, "Could not retrieve instance '%s'", name)
	}
	provider, err := cm.GetProvider(instance.CloudName)
	if err != nil {
		return errors.Wrapf(err, "Could not retrieve cloud '%s'", name)
	}

	err = provider.Init()
	if err != nil {
		return errors.Wrapf(err, "Could not init cloud '%s'", name)
	}

	log.Infof("Starting instance '%s' (%s)", instance.Name, instance.VMID)
	err = provider.StartInstance(instance.VMID, instance.Location)
	if err != nil {
		return errors.Wrapf(err, "Could not start instance '%s'", name)
	}
	return nil
}

// StopInstance stops an instance
func (cm *Manager) StopInstance(name string) error {
	instance, err := cm.GetInstance(name)
	if err != nil {
		return errors.Wrapf(err, "Could not retrieve instance '%s'", name)
	}
	provider, err := cm.GetProvider(instance.CloudName)
	if err != nil {
		return errors.Wrapf(err, "Could not retrieve cloud '%s'", name)
	}

	err = provider.Init()
	if err != nil {
		return errors.Wrapf(err, "Could not init cloud '%s'", name)
	}

	log.Infof("Stopping instance '%s' (%s)", instance.Name, instance.VMID)
	err = provider.StopInstance(instance.VMID, instance.Location)
	if err != nil {
		return errors.Wrapf(err, "Could not stop instance '%s'", name)
	}
	return nil
}

// TunnelInstance creates and SSH tunnel to the instance
func (cm *Manager) TunnelInstance(name string) error {
	instanceInfo, err := cm.GetInstance(name)
	if err != nil {
		return errors.Wrapf(err, "Could not retrieve instance '%s'", name)
	}
	if len(instanceInfo.KeySeed) == 0 {
		return errors.Errorf("Instance '%s' is missing its SSH key", name)
	}
	key, err := cm.sm.NewKeyFromSeed(instanceInfo.KeySeed)
	if err != nil {
		return errors.Wrapf(err, "Instance '%s' has an invalid SSH key", name)
	}

	log.Infof("Creating SSH tunnel to instance '%s', using ip '%s'", instanceInfo.Name, instanceInfo.PublicIP)
	tunnel := ssh.NewTunnel(instanceInfo.PublicIP+":22", "root", key.SSHAuth(), "localhost:8080")
	localPort, err := tunnel.Start()
	if err != nil {
		return errors.Wrap(err, "Error while creating the SSH tunnel")
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
		return errors.Wrap(err, "Error while terminating the SSH tunnel")
	}
	log.Info("SSH tunnel terminated successfully")
	return nil
}

// GetInstance retrieves an instance from the db and returns it
func (cm *Manager) GetInstance(name string) (core.InstanceInfo, error) {
	instances := map[string]core.InstanceInfo{}
	err := cm.db.GetMap(instanceDS, &instances)
	if err != nil {
		return core.InstanceInfo{}, err
	}
	for _, instance := range instances {
		if instance.Name == name {
			return instance, nil
		}
	}
	return core.InstanceInfo{}, fmt.Errorf("Could not find instance '%s'", name)
}

// GetInstances returns all the instances from the db
func (cm *Manager) GetInstances() ([]core.InstanceInfo, error) {
	instanceMap := map[string]core.InstanceInfo{}
	var instances []core.InstanceInfo
	err := cm.db.GetMap(instanceDS, &instanceMap)
	if err != nil {
		return instances, err
	}
	for _, instance := range instanceMap {
		instances = append(instances, instance)
	}
	return instances, nil
}

// CreateManager creates and returns a cloud manager
func CreateManager(db core.DB, um core.UserManager, sm core.SSHManager) *Manager {
	if db == nil || um == nil || sm == nil {
		log.Panic("Failed to create cloud manager: none of the inputs can be nil")
	}
	return &Manager{db: db, um: um, sm: sm}
}
