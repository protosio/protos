package cloud

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"text/tabwriter"
)

const (
	ServerStateRunning  = "running"
	ServerStateStopped  = "stopped"
	ServerStateOther    = "other"
	ServerStateChanging = "changing"

	KindLocalVM = "local_vm"
	KindCloudVM = "cloud_vm"
	// Scaleway cloud provider
	Scaleway = Type("scaleway")

	protosPublicKey = "/var/lib/protos/protos.pub"
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
	ID           string
	Name         string
	PublicKey    string // ed25519 public key
	PublicIP     string // this can be a public or private IP, depending on where the device is located
	Kind         string // type of instance: local_vm, cloud_vm
	KindID       string // ID of the cloud provider or device ID for local VM
	Location     string
	Status       string
	Architecture string
	Volumes      []VolumeInfo
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
