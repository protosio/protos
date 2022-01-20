package cloud

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"text/tabwriter"

	"github.com/protosio/protos/internal/auth"
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
	Architecture  string
	Volumes       []VolumeInfo
}

func (i InstanceInfo) GetPublicKey() []byte {
	return i.PublicKey
}

func (i InstanceInfo) GetPublicIP() string {
	return i.PublicIP
}

func (i InstanceInfo) GetName() string {
	return i.Name
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
