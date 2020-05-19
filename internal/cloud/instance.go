package cloud

import (
	"bytes"
	"encoding/base64"
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
	"github.com/protosio/protos/internal/user"
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

// AllocateNetwork allocates an unused network for an instance
func AllocateNetwork(instances []InstanceInfo) (net.IPNet, error) {
	_, userNet, err := net.ParseCIDR(user.UserNetwork)
	if err != nil {
		panic(err)
	}
	// create list of existing networks
	usedNetworks := []net.IPNet{*userNet}
	for _, inst := range instances {
		_, inet, err := net.ParseCIDR(inst.Network)
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

type CloudManager struct {
	db core.DB
}

//
// Cloud related methods
//

// SupportedProviders returns a list of supported cloud providers
func (cm *CloudManager) SupportedProviders() []string {
	return []string{Scaleway.String()}
}

func (cm *CloudManager) GetCloudProvider(name string) (Provider, error) {
	clouds := []ProviderInfo{}
	err := cm.db.GetSet(cloudDS, &clouds)
	if err != nil {
		return nil, err
	}
	for _, cld := range clouds {
		if cld.Name == name {
			cld.cm = cm
			return cld.getClient()
		}
	}
	return nil, fmt.Errorf("Could not find cloud provider '%s'", name)
}

func (cm *CloudManager) DeleteCloudProvider(name string) error {
	cld, err := cm.GetCloudProvider(name)
	if err != nil {
		return err
	}
	err = cm.db.RemoveFromSet(cloudDS, cld)
	if err != nil {
		return err
	}
	return nil
}

func (cm *CloudManager) GetCloudProviders() ([]ProviderInfo, error) {
	clouds := []ProviderInfo{}
	err := cm.db.GetSet(cloudDS, &clouds)
	if err != nil {
		return clouds, fmt.Errorf("Failed to retrieve cloud provides")
	}

	return clouds, nil
}

func (cm *CloudManager) NewProvider(cloudName string, cloud string) (Provider, error) {
	cloudType := Type(cloud)
	cld := ProviderInfo{Name: cloudName, Type: cloudType, cm: cm}
	return cld.getClient()
}

//
// Instance related methods
//

func (cm *CloudManager) DeployInstance(instanceName string, cloudName string, cloudLocation string, release release.Release, machineType string) (InstanceInfo, error) {
	usr, err := user.Get(cm.db)
	if err != nil {
		return InstanceInfo{}, err
	}

	// init cloud
	provider, err := cm.GetCloudProvider(cloudName)
	if err != nil {
		return InstanceInfo{}, errors.Wrapf(err, "Could not retrieve cloud '%s'", cloudName)
	}
	err = provider.Init()
	if err != nil {
		return InstanceInfo{}, errors.Wrapf(err, "Failed to init cloud provider '%s'(%s) API", cloudName, provider.TypeStr())
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
		return InstanceInfo{}, errors.Wrap(err, "Failed to deploy Protos instance")
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
				return InstanceInfo{}, errors.Wrap(err, "Failed to deploy Protos instance")
			}
		} else {
			return InstanceInfo{}, errors.Errorf("Could not find a Protos version '%s' release for cloud '%s'", release.Version, provider.TypeStr())
		}
	}

	// create SSH key used for instance
	log.Info("Generating SSH key for the new VM instance")
	instanceSSHKey, err := ssh.GenerateKey()
	if err != nil {
		return InstanceInfo{}, errors.Wrap(err, "Failed to deploy Protos instance")
	}

	// deploy a protos instance
	log.Infof("Deploying instance '%s' of type '%s', using Protos version '%s' (image id '%s')", instanceName, machineType, release.Version, imageID)
	vmID, err := provider.NewInstance(instanceName, imageID, instanceSSHKey.AuthorizedKey(), machineType, cloudLocation)
	if err != nil {
		return InstanceInfo{}, errors.Wrap(err, "Failed to deploy Protos instance")
	}
	log.Infof("Instance with ID '%s' deployed", vmID)

	// get instance info
	instanceInfo, err := provider.GetInstanceInfo(vmID, cloudLocation)
	if err != nil {
		return InstanceInfo{}, errors.Wrap(err, "Failed to get Protos instance info")
	}

	// allocate network
	instances, err := cm.GetInstances()
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("Failed to allocate network for instance '%s': %w", instanceInfo.Name, err)
	}
	network, err := AllocateNetwork(instances)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("Failed to allocate network for instance '%s': %w", instanceInfo.Name, err)
	}

	// save instance information
	instanceInfo.KeySeed = instanceSSHKey.Seed()
	instanceInfo.ProtosVersion = release.Version
	instanceInfo.Network = network.String()
	err = cm.db.InsertInSet(instanceDS, instanceInfo)
	if err != nil {
		return InstanceInfo{}, errors.Wrapf(err, "Failed to save instance '%s'", instanceName)
	}

	// create protos data volume
	log.Infof("Creating data volume for Protos instance '%s'", instanceName)
	volumeID, err := provider.NewVolume(instanceName, 30000, cloudLocation)
	if err != nil {
		return InstanceInfo{}, errors.Wrap(err, "Failed to create data volume")
	}

	// attach volume to instance
	err = provider.AttachVolume(volumeID, vmID, cloudLocation)
	if err != nil {
		return InstanceInfo{}, errors.Wrapf(err, "Failed to attach volume to instance '%s'", instanceName)
	}

	// start protos instance
	log.Infof("Starting Protos instance '%s'", instanceName)
	err = provider.StartInstance(vmID, cloudLocation)
	if err != nil {
		return InstanceInfo{}, errors.Wrap(err, "Failed to start Protos instance")
	}

	// get instance info again
	instanceInfo, err = provider.GetInstanceInfo(vmID, cloudLocation)
	if err != nil {
		return InstanceInfo{}, errors.Wrap(err, "Failed to get Protos instance info")
	}
	// second save of the instance information
	err = cm.db.InsertInSet(instanceDS, instanceInfo)
	if err != nil {
		return InstanceInfo{}, errors.Wrapf(err, "Failed to save instance '%s'", instanceName)
	}

	// wait for port 22 to be open
	err = WaitForPort(instanceInfo.PublicIP, "22", 20)
	if err != nil {
		return InstanceInfo{}, errors.Wrap(err, "Failed to deploy instance")
	}

	// allow some time for Protosd to start up, or else the tunnel might fail
	time.Sleep(5 * time.Second)

	log.Infof("Creating SSH tunnel to instance '%s'", instanceName)
	tunnel := ssh.NewTunnel(instanceInfo.PublicIP+":22", "root", instanceSSHKey.SSHAuth(), "localhost:8080")
	localPort, err := tunnel.Start()
	if err != nil {
		return InstanceInfo{}, errors.Wrap(err, "Error while creating the SSH tunnel")
	}

	// wait for the API to be up
	err = WaitForHTTP(fmt.Sprintf("http://127.0.0.1:%d/ui/", localPort), 20)
	if err != nil {
		return InstanceInfo{}, errors.Wrap(err, "Failed to deploy instance")
	}
	log.Infof("Tunnel to '%s' ready", instanceName)

	// do the initialization
	log.Infof("Initializing instance '%s'", instanceName)
	protos := pclient.NewInitClient(fmt.Sprintf("127.0.0.1:%d", localPort), usr.Username, usr.Password)
	userDeviceKey, err := ssh.NewKeyFromSeed(usr.Device.KeySeed)
	if err != nil {
		panic(err)
	}
	keyEncoded := base64.StdEncoding.EncodeToString(userDeviceKey.Public())
	usrDev := types.UserDevice{
		Name:      usr.Device.Name,
		PublicKey: keyEncoded,
		Network:   usr.Device.Network,
	}
	ip, pubKey, err := protos.InitInstance(usr.Name, instanceInfo.Network, usr.Domain, []types.UserDevice{usrDev})
	if err != nil {
		return InstanceInfo{}, errors.Wrap(err, "Error while doing the instance initialization")
	}

	// final save instance info
	instanceInfo.InternalIP = ip.String()
	instanceInfo.PublicKey = pubKey
	err = cm.db.InsertInSet(instanceDS, instanceInfo)
	if err != nil {
		return InstanceInfo{}, errors.Wrapf(err, "Failed to save instance '%s'", instanceName)
	}

	// close the SSH tunnel
	err = tunnel.Close()
	if err != nil {
		return InstanceInfo{}, errors.Wrap(err, "Error while terminating the SSH tunnel")
	}
	log.Infof("Instance '%s' is ready", instanceName)

	return instanceInfo, nil
}

func (cm *CloudManager) DeleteInstance(name string, localOnly bool) error {
	instance, err := cm.GetInstance(name)
	if err != nil {
		return errors.Wrapf(err, "Could not retrieve instance '%s'", name)
	}

	// if local only, ignore any cloud resources
	if !localOnly {
		provider, err := cm.GetCloudProvider(instance.CloudName)
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
	return cm.db.RemoveFromSet(instanceDS, instance)
}

func (cm *CloudManager) StartInstance(name string) error {
	instance, err := cm.GetInstance(name)
	if err != nil {
		return errors.Wrapf(err, "Could not retrieve instance '%s'", name)
	}
	provider, err := cm.GetCloudProvider(instance.CloudName)
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

func (cm *CloudManager) StopInstance(name string) error {
	instance, err := cm.GetInstance(name)
	if err != nil {
		return errors.Wrapf(err, "Could not retrieve instance '%s'", name)
	}
	provider, err := cm.GetCloudProvider(instance.CloudName)
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

func (cm *CloudManager) TunnelInstance(name string) error {
	instanceInfo, err := cm.GetInstance(name)
	if err != nil {
		return errors.Wrapf(err, "Could not retrieve instance '%s'", name)
	}
	if len(instanceInfo.KeySeed) == 0 {
		return errors.Errorf("Instance '%s' is missing its SSH key", name)
	}
	key, err := ssh.NewKeyFromSeed(instanceInfo.KeySeed)
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

func (im *CloudManager) GetInstance(name string) (InstanceInfo, error) {
	instances := []InstanceInfo{}
	err := im.db.GetSet(instanceDS, &instances)
	if err != nil {
		return InstanceInfo{}, err
	}
	for _, instance := range instances {
		if instance.Name == name {
			return instance, nil
		}
	}
	return InstanceInfo{}, fmt.Errorf("Could not find instance '%s'", name)
}

func (im *CloudManager) GetInstances() ([]InstanceInfo, error) {
	var instances []InstanceInfo
	err := im.db.GetSet(instanceDS, &instances)
	if err != nil {
		return instances, err
	}
	return instances, nil
}

func NewCloudManager(db core.DB) (CloudManager, error) {
	return CloudManager{db: db}, nil
}
