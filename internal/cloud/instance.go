package cloud

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"text/tabwriter"

	"github.com/bokwoon95/sq"
	"github.com/protosio/protos/internal/auth"
	"github.com/protosio/protos/internal/db"
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

	protosPublicKey = "/var/lib/protos/protos_key.pub"
)

func createInstanceInsertMapper(instance InstanceInfo) func() (sq.Table, func(*sq.Column)) {
	return func() (sq.Table, func(*sq.Column)) {
		i := sq.New[db.INSTANCE]("")
		return i, func(col *sq.Column) {
			col.SetString(i.VM_ID, instance.VMID)
			col.SetString(i.NAME, instance.Name)
			col.SetString(i.SSH_KEY_SEED, instance.SSHKeySeed)
			col.SetString(i.PUBLIC_KEY, instance.PublicKey)
			col.SetString(i.PUBLIC_IP, instance.PublicIP)
			col.SetString(i.INTERNAL_IP, instance.InternalIP)
			col.SetString(i.CLOUD_TYPE, instance.CloudType)
			col.SetString(i.CLOUD_NAME, instance.CloudName)
			col.SetString(i.LOCATION, instance.Location)
			col.SetString(i.NETWORK, instance.Network)
			col.SetString(i.PROTOS_VERSION, instance.ProtosVersion)
			col.SetString(i.ARCHITECTURE, instance.Architecture)
		}
	}
}

func createInstanceUpdateMapper(instance InstanceInfo) func() (sq.Table, func(*sq.Column), []sq.Predicate) {
	return func() (sq.Table, func(*sq.Column), []sq.Predicate) {
		i := sq.New[db.INSTANCE]("")
		predicates := []sq.Predicate{i.VM_ID.EqString(instance.VMID)}
		return i, func(col *sq.Column) {
			col.SetString(i.NAME, instance.Name)
			col.SetString(i.SSH_KEY_SEED, instance.SSHKeySeed)
			col.SetString(i.PUBLIC_KEY, instance.PublicKey)
			col.SetString(i.PUBLIC_IP, instance.PublicIP)
			col.SetString(i.INTERNAL_IP, instance.InternalIP)
			col.SetString(i.CLOUD_TYPE, instance.CloudType)
			col.SetString(i.CLOUD_NAME, instance.CloudName)
			col.SetString(i.LOCATION, instance.Location)
			col.SetString(i.NETWORK, instance.Network)
			col.SetString(i.PROTOS_VERSION, instance.ProtosVersion)
			col.SetString(i.ARCHITECTURE, instance.Architecture)
		}, predicates
	}
}

func createInstanceQueryMapper(i db.INSTANCE, predicates []sq.Predicate) func() (sq.Table, func(row *sq.Row) InstanceInfo, []sq.Predicate) {
	return func() (sq.Table, func(row *sq.Row) InstanceInfo, []sq.Predicate) {
		mapper := func(row *sq.Row) InstanceInfo {
			return InstanceInfo{
				VMID:          row.StringField(i.VM_ID),
				Name:          row.StringField(i.NAME),
				SSHKeySeed:    row.StringField(i.SSH_KEY_SEED),
				PublicKey:     row.StringField(i.PUBLIC_KEY),
				PublicIP:      row.StringField(i.PUBLIC_IP),
				InternalIP:    row.StringField(i.INTERNAL_IP),
				CloudType:     row.StringField(i.CLOUD_TYPE),
				CloudName:     row.StringField(i.CLOUD_NAME),
				Location:      row.StringField(i.LOCATION),
				Network:       row.StringField(i.NETWORK),
				ProtosVersion: row.StringField(i.PROTOS_VERSION),
				Architecture:  row.StringField(i.ARCHITECTURE),
			}
		}
		return i, mapper, predicates
	}
}

func createInstanceDeleteByNameQuery(name string) func() (sq.Table, []sq.Predicate) {
	return func() (sq.Table, []sq.Predicate) {
		i := sq.New[db.INSTANCE]("")
		return i, []sq.Predicate{i.NAME.EqString(name)}
	}
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

// InstanceInfo holds information about a cloud instance
type InstanceInfo struct {
	VMID          string
	Name          string
	SSHKeySeed    string // private SSH key stored only on the client
	PublicKey     string // ed25519 public key
	PublicIP      string // this can be a public or private IP, depending on where the device is located
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

func (i InstanceInfo) GetPublicKey() string {
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
