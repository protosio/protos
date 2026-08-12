package provisioners

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/pcrypto"
)

const (
	ServerStateRunning  = "running"
	ServerStateStopped  = "stopped"
	ServerStateOther    = "other"
	ServerStateChanging = "changing"
	ServerStateDeleting = "deleting"

	KindLocalVM = "local_vm"
	KindCloudVM = "cloud_vm"
)

// VolumeInfo holds information about a data volume
type VolumeInfo struct {
	VolumeID string
	Name     string
	Size     uint64
}

// ImageInfo holds information about a cloud image used for deploying an instance
type ImageInfo struct {
	ID          string
	Name        string
	LogicalName string
	DateSuffix  string
	Location    string
	UpdatedAt   time.Time
	Canonical   bool
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
	ID                   string
	Name                 string
	PublicKey            string // ed25519 public key
	PublicIP             string // this can be a public or private IP, depending on where the device is located
	Kind                 string // type of instance: local_vm, cloud_vm
	KindID               string // ID of the cloud provider or device ID for local VM
	ProviderResourceID   string // ID used by the provisioner API for lifecycle operations
	LifecycleOwnerPeerID string // immutable peer authorized to perform provider lifecycle operations
	DesiredStatus        string
	ReplicationPriority  int
	Location             string
	Status               string
	Architecture         string
	Volumes              []VolumeInfo
}

func (i InstanceInfo) GetID() string {
	return i.ID
}

func (i InstanceInfo) GetPublicKey() string {
	return i.PublicKey
}

func (i InstanceInfo) GetPeerID() (string, error) {
	return db.PeerIDFromPublicKeyString(i.PublicKey)
}

func (i InstanceInfo) GetPublicIP() string {
	return i.PublicIP
}

func (i InstanceInfo) GetInternalIP() string {
	key, err := pcrypto.CreatePublicKeyFromBase64(i.PublicKey)
	if err != nil {
		return ""
	}
	return key.IPv6Address().String()
}

func (i InstanceInfo) GetName() string {
	return i.Name
}

func IsDeletingInstance(instance InstanceInfo) bool {
	return strings.EqualFold(strings.TrimSpace(instance.DesiredStatus), ServerStateDeleting)
}

func IsStoppedInstance(instance InstanceInfo) bool {
	return strings.EqualFold(strings.TrimSpace(instance.DesiredStatus), ServerStateStopped)
}

func IsActiveInstance(instance InstanceInfo) bool {
	return !IsDeletingInstance(instance) && !IsStoppedInstance(instance)
}

func ActiveInstances(instances []InstanceInfo) []InstanceInfo {
	active := make([]InstanceInfo, 0, len(instances))
	for _, instance := range instances {
		if !IsActiveInstance(instance) {
			continue
		}
		active = append(active, instance)
	}
	return active
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
