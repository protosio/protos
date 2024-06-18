package cloud

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"text/tabwriter"

	"github.com/bokwoon95/sq"
	"github.com/protosio/protos/internal/db"
)

const (
	ServerStateRunning  = "running"
	ServerStateStopped  = "stopped"
	ServerStateOther    = "other"
	ServerStateChanging = "changing"

	protosPublicKey = "/var/lib/protos/protos.pub"
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
			col.SetString(i.CLOUD_TYPE, instance.CloudType)
			col.SetString(i.CLOUD_NAME, instance.CloudName)
			col.SetString(i.LOCATION, instance.Location)
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
			col.SetString(i.CLOUD_TYPE, instance.CloudType)
			col.SetString(i.CLOUD_NAME, instance.CloudName)
			col.SetString(i.LOCATION, instance.Location)
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
				CloudType:     row.StringField(i.CLOUD_TYPE),
				CloudName:     row.StringField(i.CLOUD_NAME),
				Location:      row.StringField(i.LOCATION),
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
	CloudType     string
	CloudName     string
	Location      string
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
