//go:build darwin

package localmacos

import (
	"archive/tar"
	"compress/gzip"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	hostagentclient "github.com/protosio/protos/internal/hostagent/client"
	hostagentpb "github.com/protosio/protos/internal/hostagent/proto"
	"github.com/protosio/protos/internal/provisioners"
	"github.com/protosio/protos/internal/util"
	"golang.org/x/sys/unix"
)

const (
	localMacOSLocation      = "local"
	localMacOSAuthVMDir     = "VM_DIR"
	localMacOSImageKernel   = "kernel"
	localMacOSImageInitrd   = "initrd.img"
	localMacOSImageCmdline  = "cmdline"
	localMacOSImageRootDisk = "root.raw"
	localMacOSImageBootISO  = "boot.iso"
	localMacOSMetadataISO   = "metadata.iso"
	localMacOSManifestFile  = "manifest.json"
	localMacOSConsoleLog    = "console.log"

	localMacOSNetworkInterface  = "eth0"
	localMacOSNetworkCIDR       = "192.168.64.0/24"
	localMacOSNetworkGateway    = "192.168.64.1"
	localMacOSNetworkPrefix     = 24
	localMacOSStaticIPRangeFrom = 128
	localMacOSStaticIPRangeTo   = 254
)

var localMacOSAuthFields = []string{}
var log = util.GetLogger("provisioner.localmacos")

const Type = provisioners.Type("local_macos")

type localMacOSCredentials struct {
	vmDir string
}

type localMacOS struct {
	provisioners.ProvisionerMetadata

	vmDir string
}

type localMacOSImageManifest struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Location     string `json:"location"`
	KernelPath   string `json:"kernel_path"`
	InitrdPath   string `json:"initrd_path"`
	CmdlinePath  string `json:"cmdline_path"`
	RootDiskPath string `json:"root_disk_path,omitempty"`
	BootISOPath  string `json:"boot_iso_path,omitempty"`
}

type localMacOSVolumeManifest struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	SizeMiB  int    `json:"size_mib"`
	Location string `json:"location"`
}

type localMacOSAttachedVolume struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	SizeMiB int    `json:"size_mib"`
}

type localMacOSNetworkManifest struct {
	Interface    string   `json:"interface"`
	IPAddress    string   `json:"ip_address"`
	PrefixLength int      `json:"prefix_length"`
	Gateway      string   `json:"gateway"`
	DNSServers   []string `json:"dns_servers,omitempty"`
}

type localMacOSInstanceManifest struct {
	ID           string                     `json:"id"`
	Name         string                     `json:"name"`
	ImageID      string                     `json:"image_id"`
	Location     string                     `json:"location"`
	MachineType  string                     `json:"machine_type"`
	Cores        uint32                     `json:"cores"`
	MemoryMiB    uint32                     `json:"memory_mib"`
	PublicKey    string                     `json:"public_key"`
	PublicIP     string                     `json:"public_ip"`
	MACAddress   string                     `json:"mac_address"`
	Status       string                     `json:"status"`
	PID          int                        `json:"pid,omitempty"`
	KernelPath   string                     `json:"kernel_path"`
	InitrdPath   string                     `json:"initrd_path"`
	CmdlinePath  string                     `json:"cmdline_path"`
	RootDiskPath string                     `json:"root_disk_path,omitempty"`
	BootISOPath  string                     `json:"boot_iso_path,omitempty"`
	MetadataISO  string                     `json:"metadata_iso,omitempty"`
	Network      localMacOSNetworkManifest  `json:"network,omitempty"`
	Volumes      []localMacOSAttachedVolume `json:"volumes,omitempty"`
}

type localMacOSImageSource struct {
	kernel   string
	initrd   string
	cmdline  string
	rootDisk string
	bootISO  string
}

type localMacOSMetadataEntry struct {
	Content string                             `json:"content,omitempty"`
	Entries map[string]localMacOSMetadataEntry `json:"entries,omitempty"`
	Perm    string                             `json:"perm,omitempty"`
}

func NewFactory() provisioners.ProvisionerFactory {
	return provisioners.NewTypedProvisionerFactory(Type, localMacOSAuthFields, decodeLocalMacOSCredentials, newLocalMacOSClient)
}

func decodeLocalMacOSCredentials(auth map[string]string) (localMacOSCredentials, error) {
	credentials := localMacOSCredentials{}
	for key, value := range auth {
		if strings.TrimSpace(value) == "" {
			continue
		}
		switch key {
		case localMacOSAuthVMDir:
			credentials.vmDir = value
		default:
			return localMacOSCredentials{}, fmt.Errorf("credentials field '%s' not supported by local macOS cloud provider", key)
		}
	}
	return credentials, nil
}

func newLocalMacOSClient(metadata provisioners.ProvisionerMetadata, deps provisioners.ProvisionerDeps, credentials localMacOSCredentials) (provisioners.Provisioner, error) {
	workDir := deps.WorkDir
	if strings.TrimSpace(workDir) == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to resolve home directory: %w", err)
		}
		workDir = filepath.Join(homeDir, ".protos")
	}
	workDir, err := expandLocalMacOSPath(workDir)
	if err != nil {
		return nil, err
	}

	vmDir := credentials.vmDir
	if strings.TrimSpace(vmDir) == "" {
		vmDir = filepath.Join(workDir, "local-macos-vms")
	}
	vmDir, err = expandLocalMacOSPath(vmDir)
	if err != nil {
		return nil, err
	}

	return &localMacOS{
		ProvisionerMetadata: metadata,
		vmDir:               vmDir,
	}, nil
}

func (lm *localMacOS) Init() error {
	for _, dir := range []string{lm.imagesDir(), lm.instancesDir(), lm.volumesDir()} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create local macOS VM directory '%s': %w", dir, err)
		}
	}
	return nil
}

func (lm *localMacOS) SupportedLocations() []string {
	return []string{localMacOSLocation}
}

func (lm *localMacOS) SupportedMachines(location string) (map[string]provisioners.MachineSpec, error) {
	if err := validateLocalMacOSLocation(location); err != nil {
		return nil, err
	}
	return map[string]provisioners.MachineSpec{
		"vz-1c-1g": {
			Cores:          1,
			Memory:         1024,
			DefaultStorage: 30,
			Baremetal:      false,
			PriceMonthly:   0,
		},
		"vz-2c-2g": {
			Cores:          2,
			Memory:         2048,
			DefaultStorage: 30,
			Baremetal:      false,
			PriceMonthly:   0,
		},
		"vz-4c-4g": {
			Cores:          4,
			Memory:         4096,
			DefaultStorage: 30,
			Baremetal:      false,
			PriceMonthly:   0,
		},
	}, nil
}

func (lm *localMacOS) NewInstance(name string, imageID string, pubKey string, machineType string, location string) (string, error) {
	if err := validateLocalMacOSLocation(location); err != nil {
		return "", err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("instance name is empty")
	}
	if existing, err := lm.findInstanceByName(name); err != nil {
		return "", err
	} else if existing.ID != "" {
		return "", fmt.Errorf("there is already a local macOS VM named '%s'", name)
	}

	machines, err := lm.SupportedMachines(location)
	if err != nil {
		return "", err
	}
	machine, found := machines[machineType]
	if !found {
		return "", fmt.Errorf("machine type '%s' is not valid for local macOS", machineType)
	}

	image, err := lm.readImageManifest(imageID)
	if err != nil {
		return "", err
	}

	id, err := newLocalMacOSID("vm")
	if err != nil {
		return "", err
	}
	instanceDir := lm.instanceDir(id)
	if err := os.MkdirAll(instanceDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create instance directory: %w", err)
	}

	kernelPath := ""
	initrdPath := ""
	cmdlinePath := ""
	if image.KernelPath != "" {
		kernelPath = filepath.Join(instanceDir, localMacOSImageKernel)
		initrdPath = filepath.Join(instanceDir, localMacOSImageInitrd)
		cmdlinePath = filepath.Join(instanceDir, localMacOSImageCmdline)
		if err := copyLocalMacOSFile(image.KernelPath, kernelPath); err != nil {
			return "", err
		}
		if err := copyLocalMacOSFile(image.InitrdPath, initrdPath); err != nil {
			return "", err
		}
		if err := copyLocalMacOSFile(image.CmdlinePath, cmdlinePath); err != nil {
			return "", err
		}
	}

	bootISOPath := ""
	if image.BootISOPath != "" {
		bootISOPath = filepath.Join(instanceDir, localMacOSImageBootISO)
		if err := copyLocalMacOSFile(image.BootISOPath, bootISOPath); err != nil {
			return "", err
		}
	}

	rootDiskPath := ""
	if image.RootDiskPath != "" {
		rootDiskPath = filepath.Join(instanceDir, localMacOSImageRootDisk)
		if err := copyLocalMacOSFile(image.RootDiskPath, rootDiskPath); err != nil {
			return "", err
		}
	}

	macAddress, err := newLocalMacOSMACAddress()
	if err != nil {
		return "", err
	}
	network, err := lm.newNetworkManifest(id)
	if err != nil {
		return "", err
	}

	manifest := localMacOSInstanceManifest{
		ID:           id,
		Name:         name,
		ImageID:      imageID,
		Location:     localMacOSLocation,
		MachineType:  machineType,
		Cores:        machine.Cores,
		MemoryMiB:    machine.Memory,
		PublicKey:    strings.TrimSpace(pubKey),
		PublicIP:     network.IPAddress,
		MACAddress:   macAddress,
		Status:       provisioners.ServerStateStopped,
		KernelPath:   kernelPath,
		InitrdPath:   initrdPath,
		CmdlinePath:  cmdlinePath,
		RootDiskPath: rootDiskPath,
		BootISOPath:  bootISOPath,
		Network:      network,
	}
	if err := lm.writeInstanceManifest(manifest); err != nil {
		return "", err
	}
	return id, nil
}

func (lm *localMacOS) DeleteInstance(id string, location string) error {
	if err := validateLocalMacOSLocation(location); err != nil {
		return err
	}
	manifest, err := lm.readInstanceManifestByIDOrName(id)
	if err != nil {
		return err
	}
	if _, err := lm.applyHostAgentVM(manifest.ID, "deleted"); err != nil {
		if manifest.PID != 0 && localMacOSProcessAlive(manifest.PID) {
			return err
		}
		log.Debugf("Skipping host agent delete for stopped local macOS VM '%s': %v", manifest.ID, err)
	}
	if err := removeLocalMacOSDir(lm.instanceDir(manifest.ID)); err != nil {
		return fmt.Errorf("failed to remove local macOS VM '%s': %w", id, err)
	}
	return nil
}

func (lm *localMacOS) StartInstance(id string, location string) error {
	if err := validateLocalMacOSLocation(location); err != nil {
		return err
	}
	manifest, err := lm.readInstanceManifestByIDOrName(id)
	if err != nil {
		return err
	}
	if manifest.PID != 0 && localMacOSProcessAlive(manifest.PID) {
		return nil
	}
	if err := lm.ensureNetworkManifest(&manifest); err != nil {
		return err
	}
	manifest.Status = provisioners.ServerStateChanging
	manifest.PID = 0
	manifest.PublicIP = manifest.Network.IPAddress
	manifest.MetadataISO = filepath.Join(lm.instanceDir(manifest.ID), localMacOSMetadataISO)
	if err := writeLocalMacOSMetadataISO(manifest.MetadataISO, manifest.Name, manifest.PublicKey, manifest.Network); err != nil {
		return fmt.Errorf("failed to write metadata ISO: %w", err)
	}
	if err := lm.writeInstanceManifest(manifest); err != nil {
		return err
	}

	state, err := lm.applyHostAgentVM(manifest.ID, provisioners.ServerStateRunning)
	if err != nil {
		manifest.Status = provisioners.ServerStateStopped
		_ = lm.writeInstanceManifest(manifest)
		return err
	}
	manifest.PID = int(state.GetPid())
	if state.GetStatus() != "" {
		manifest.Status = state.GetStatus()
	}
	if err := lm.writeInstanceManifest(manifest); err != nil {
		return err
	}
	return nil
}

func (lm *localMacOS) StopInstance(id string, location string) error {
	if err := validateLocalMacOSLocation(location); err != nil {
		return err
	}
	manifest, err := lm.readInstanceManifestByIDOrName(id)
	if err != nil {
		return err
	}
	manifest.Status = provisioners.ServerStateChanging
	if err := lm.writeInstanceManifest(manifest); err != nil {
		return err
	}

	state, err := lm.applyHostAgentVM(manifest.ID, provisioners.ServerStateStopped)
	if err != nil {
		return err
	}
	manifest.PID = int(state.GetPid())
	manifest.Status = state.GetStatus()
	if manifest.Status == "" {
		manifest.Status = provisioners.ServerStateStopped
	}
	return lm.writeInstanceManifest(manifest)
}

func (lm *localMacOS) GetInstanceInfo(id string, location string) (provisioners.InstanceInfo, error) {
	if err := validateLocalMacOSLocation(location); err != nil {
		return provisioners.InstanceInfo{}, err
	}
	manifest, err := lm.readInstanceManifestByIDOrName(id)
	if err != nil {
		return provisioners.InstanceInfo{}, err
	}
	updated := false
	if manifest.Network.IPAddress == "" {
		if err := lm.ensureNetworkManifest(&manifest); err != nil {
			return provisioners.InstanceInfo{}, err
		}
		updated = true
	}
	if manifest.PublicIP != manifest.Network.IPAddress {
		manifest.PublicIP = manifest.Network.IPAddress
		updated = true
	}
	if state, err := lm.hostAgentVMStatus(manifest.ID); err == nil {
		if state.GetPid() != int32(manifest.PID) {
			manifest.PID = int(state.GetPid())
			updated = true
		}
		if state.GetStatus() != "" && state.GetStatus() != manifest.Status {
			manifest.Status = state.GetStatus()
			updated = true
		}
	} else if manifest.PID != 0 && localMacOSProcessAlive(manifest.PID) {
		if manifest.Status != provisioners.ServerStateRunning {
			manifest.Status = provisioners.ServerStateRunning
			updated = true
		}
	} else if manifest.Status == provisioners.ServerStateRunning || manifest.Status == provisioners.ServerStateChanging {
		manifest.PID = 0
		manifest.Status = provisioners.ServerStateStopped
		updated = true
	}
	if updated {
		if err := lm.writeInstanceManifest(manifest); err != nil {
			return provisioners.InstanceInfo{}, err
		}
	}

	info := provisioners.InstanceInfo{
		ID:                 manifest.ID,
		Name:               manifest.Name,
		PublicIP:           manifest.PublicIP,
		Kind:               provisioners.KindCloudVM,
		KindID:             lm.NameStr(),
		ProviderResourceID: manifest.ID,
		Location:           localMacOSLocation,
		Status:             manifest.Status,
	}
	for _, volume := range manifest.Volumes {
		info.Volumes = append(info.Volumes, provisioners.VolumeInfo{
			VolumeID: volume.ID,
			Name:     volume.Name,
			Size:     uint64(volume.SizeMiB) * 1048576,
		})
	}
	return info, nil
}

func (lm *localMacOS) ReconcileInstance(instance provisioners.InstanceInfo) (provisioners.InstanceInfo, error) {
	ref := firstNonEmptyLocalMacOSString(instance.ProviderResourceID, instance.Name, instance.ID)
	if ref == "" {
		return instance, fmt.Errorf("local macOS VM reference is empty")
	}

	current, err := lm.GetInstanceInfo(ref, instance.Location)
	if err != nil {
		return instance, err
	}
	switch strings.ToLower(strings.TrimSpace(instance.DesiredStatus)) {
	case provisioners.ServerStateRunning:
		if current.Status != provisioners.ServerStateRunning {
			if err := lm.StartInstance(ref, instance.Location); err != nil {
				return instance, err
			}
			current, err = lm.GetInstanceInfo(ref, instance.Location)
			if err != nil {
				return instance, err
			}
		}
	case provisioners.ServerStateStopped:
		if current.Status != provisioners.ServerStateStopped {
			if err := lm.StopInstance(ref, instance.Location); err != nil {
				return instance, err
			}
			current, err = lm.GetInstanceInfo(ref, instance.Location)
			if err != nil {
				return instance, err
			}
		}
	}
	current.DesiredStatus = instance.DesiredStatus
	return current, nil
}

func (lm *localMacOS) GetImages() (map[string]provisioners.ImageInfo, error) {
	return lm.getImages(false)
}

func (lm *localMacOS) GetProtosImages() (map[string]provisioners.ImageInfo, error) {
	return lm.getImages(true)
}

func (lm *localMacOS) AddImage(url string, hash string, version string, location string) (string, error) {
	if err := validateLocalMacOSLocation(location); err != nil {
		return "", err
	}
	if strings.TrimSpace(url) == "" {
		return "", fmt.Errorf("local macOS image URL is empty")
	}

	tmpDir, err := os.MkdirTemp("", "protos-local-macos-image-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	downloadPath := filepath.Join(tmpDir, "image")
	if err := downloadLocalMacOSImage(url, downloadPath); err != nil {
		return "", err
	}
	if strings.TrimSpace(hash) != "" {
		digest, err := sha256LocalMacOSFile(downloadPath)
		if err != nil {
			return "", err
		}
		if !strings.EqualFold(digest, hash) {
			return "", fmt.Errorf("downloaded image digest %s does not match expected digest %s", digest, hash)
		}
	}

	imagePath := downloadPath
	if isLocalMacOSArchive(url) {
		archiveDir := filepath.Join(tmpDir, "bundle")
		if err := os.MkdirAll(archiveDir, 0755); err != nil {
			return "", err
		}
		if err := extractLocalMacOSArchive(downloadPath, archiveDir, url); err != nil {
			return "", err
		}
		imagePath = archiveDir
	}

	return lm.UploadLocalImage(imagePath, version, location, 0)
}

func (lm *localMacOS) UploadLocalImage(imagePath string, imageName string, location string, timeout time.Duration) (string, error) {
	if err := validateLocalMacOSLocation(location); err != nil {
		return "", err
	}
	imageName = strings.TrimSpace(imageName)
	if imageName == "" {
		return "", fmt.Errorf("image name is empty")
	}
	imageID, err := sanitizeLocalMacOSName(imageName)
	if err != nil {
		return "", err
	}
	if _, err := lm.readImageManifest(imageID); err == nil {
		return "", fmt.Errorf("found an image with the same name")
	} else if !os.IsNotExist(err) {
		return "", err
	}

	source, err := resolveLocalMacOSImageSource(imagePath)
	if err != nil {
		return "", err
	}

	imageDir := lm.imageDir(imageID)
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create local image directory: %w", err)
	}
	kernelPath := ""
	initrdPath := ""
	cmdlinePath := ""
	if source.kernel != "" {
		kernelPath = filepath.Join(imageDir, localMacOSImageKernel)
		initrdPath = filepath.Join(imageDir, localMacOSImageInitrd)
		cmdlinePath = filepath.Join(imageDir, localMacOSImageCmdline)
		if err := copyLocalMacOSFile(source.kernel, kernelPath); err != nil {
			return "", err
		}
		if err := copyLocalMacOSFile(source.initrd, initrdPath); err != nil {
			return "", err
		}
		if source.cmdline != "" {
			if err := copyLocalMacOSFile(source.cmdline, cmdlinePath); err != nil {
				return "", err
			}
		} else if err := os.WriteFile(cmdlinePath, []byte("console=hvc0\n"), 0644); err != nil {
			return "", fmt.Errorf("failed to write default cmdline: %w", err)
		}
	}

	bootISOPath := ""
	if source.bootISO != "" {
		bootISOPath = filepath.Join(imageDir, localMacOSImageBootISO)
		if err := copyLocalMacOSFile(source.bootISO, bootISOPath); err != nil {
			return "", err
		}
	}

	rootDiskPath := ""
	if source.rootDisk != "" {
		rootDiskPath = filepath.Join(imageDir, localMacOSImageRootDisk)
		if err := copyLocalMacOSFile(source.rootDisk, rootDiskPath); err != nil {
			return "", err
		}
	}

	manifest := localMacOSImageManifest{
		ID:           imageID,
		Name:         imageName,
		Location:     localMacOSLocation,
		KernelPath:   kernelPath,
		InitrdPath:   initrdPath,
		CmdlinePath:  cmdlinePath,
		RootDiskPath: rootDiskPath,
		BootISOPath:  bootISOPath,
	}
	if err := writeJSONFile(filepath.Join(imageDir, localMacOSManifestFile), manifest); err != nil {
		return "", err
	}
	return imageID, nil
}

func (lm *localMacOS) RemoveImage(name string, location string) error {
	if err := validateLocalMacOSLocation(location); err != nil {
		return err
	}
	imageID, err := sanitizeLocalMacOSName(name)
	if err != nil {
		return err
	}
	if _, err := lm.readImageManifest(imageID); err != nil {
		return err
	}
	return os.RemoveAll(lm.imageDir(imageID))
}

func (lm *localMacOS) NewVolume(name string, size int, location string) (string, error) {
	if err := validateLocalMacOSLocation(location); err != nil {
		return "", err
	}
	if size <= 0 {
		return "", fmt.Errorf("volume size must be greater than 0 MiB")
	}
	id, err := newLocalMacOSID("vol")
	if err != nil {
		return "", err
	}
	path := filepath.Join(lm.volumesDir(), id+".raw")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create local macOS volume: %w", err)
	}
	if err := file.Truncate(int64(size) * 1048576); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("failed to size local macOS volume: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	manifest := localMacOSVolumeManifest{
		ID:       id,
		Name:     name,
		Path:     path,
		SizeMiB:  size,
		Location: localMacOSLocation,
	}
	if err := writeJSONFile(lm.volumeManifestPath(id), manifest); err != nil {
		return "", err
	}
	return id, nil
}

func (lm *localMacOS) DeleteVolume(id string, location string) error {
	if err := validateLocalMacOSLocation(location); err != nil {
		return err
	}
	manifest, err := lm.readVolumeManifest(id)
	if err != nil {
		return err
	}
	if err := os.Remove(manifest.Path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete local macOS volume '%s': %w", id, err)
	}
	if err := os.Remove(lm.volumeManifestPath(id)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete local macOS volume metadata '%s': %w", id, err)
	}
	return nil
}

func (lm *localMacOS) AttachVolume(volumeID string, instanceID string, location string) error {
	if err := validateLocalMacOSLocation(location); err != nil {
		return err
	}
	instance, err := lm.readInstanceManifest(instanceID)
	if err != nil {
		return err
	}
	if instance.PID != 0 && localMacOSProcessAlive(instance.PID) {
		return fmt.Errorf("cannot attach local macOS volume '%s' to running VM '%s'", volumeID, instanceID)
	}
	volume, err := lm.readVolumeManifest(volumeID)
	if err != nil {
		return err
	}
	for _, attached := range instance.Volumes {
		if attached.ID == volumeID {
			return nil
		}
	}
	instance.Volumes = append(instance.Volumes, localMacOSAttachedVolume{
		ID:      volume.ID,
		Name:    volume.Name,
		Path:    volume.Path,
		SizeMiB: volume.SizeMiB,
	})
	return lm.writeInstanceManifest(instance)
}

func (lm *localMacOS) DettachVolume(volumeID string, instanceID string, location string) error {
	if err := validateLocalMacOSLocation(location); err != nil {
		return err
	}
	instance, err := lm.readInstanceManifest(instanceID)
	if err != nil {
		return err
	}
	if instance.PID != 0 && localMacOSProcessAlive(instance.PID) {
		return fmt.Errorf("cannot detach local macOS volume '%s' from running VM '%s'", volumeID, instanceID)
	}
	volumes := instance.Volumes[:0]
	for _, volume := range instance.Volumes {
		if volume.ID != volumeID {
			volumes = append(volumes, volume)
		}
	}
	instance.Volumes = volumes
	return lm.writeInstanceManifest(instance)
}

func (lm *localMacOS) imagesDir() string {
	return filepath.Join(lm.vmDir, "images")
}

func (lm *localMacOS) instancesDir() string {
	return filepath.Join(lm.vmDir, "instances")
}

func (lm *localMacOS) volumesDir() string {
	return filepath.Join(lm.vmDir, "volumes")
}

func (lm *localMacOS) imageDir(id string) string {
	return filepath.Join(lm.imagesDir(), id)
}

func (lm *localMacOS) instanceDir(id string) string {
	return filepath.Join(lm.instancesDir(), id)
}

func (lm *localMacOS) instanceManifestPath(id string) string {
	return filepath.Join(lm.instanceDir(id), localMacOSManifestFile)
}

func removeLocalMacOSDir(path string) error {
	var lastErr error
	for attempt := 0; attempt < 20; attempt++ {
		if err := os.RemoveAll(path); err != nil {
			lastErr = err
		} else if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil
		} else if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("directory still exists")
		}
		time.Sleep(250 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("directory still exists")
	}
	if entries, err := os.ReadDir(path); err == nil && len(entries) > 0 {
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		return fmt.Errorf("%w; remaining entries: %s", lastErr, strings.Join(names, ", "))
	}
	return lastErr
}

func (lm *localMacOS) volumeManifestPath(id string) string {
	return filepath.Join(lm.volumesDir(), id+".json")
}

func (lm *localMacOS) getImages(protosOnly bool) (map[string]provisioners.ImageInfo, error) {
	images := map[string]provisioners.ImageInfo{}
	entries, err := os.ReadDir(lm.imagesDir())
	if err != nil {
		if os.IsNotExist(err) {
			return images, nil
		}
		return images, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifest, err := lm.readImageManifest(entry.Name())
		if err != nil {
			log.Warnf("failed to read local macOS image '%s': %s", entry.Name(), err.Error())
			continue
		}
		images[manifest.ID] = provisioners.ImageInfo{ID: manifest.ID, Name: manifest.Name, Location: manifest.Location}
	}
	return images, nil
}

func (lm *localMacOS) readImageManifest(id string) (localMacOSImageManifest, error) {
	id, err := sanitizeLocalMacOSName(id)
	if err != nil {
		return localMacOSImageManifest{}, err
	}
	var manifest localMacOSImageManifest
	if err := readJSONFile(filepath.Join(lm.imageDir(id), localMacOSManifestFile), &manifest); err != nil {
		return localMacOSImageManifest{}, err
	}
	return manifest, nil
}

func (lm *localMacOS) readInstanceManifest(id string) (localMacOSInstanceManifest, error) {
	id, err := sanitizeLocalMacOSName(id)
	if err != nil {
		return localMacOSInstanceManifest{}, err
	}
	var manifest localMacOSInstanceManifest
	if err := readJSONFile(lm.instanceManifestPath(id), &manifest); err != nil {
		return localMacOSInstanceManifest{}, err
	}
	return manifest, nil
}

func (lm *localMacOS) readInstanceManifestByIDOrName(id string) (localMacOSInstanceManifest, error) {
	manifest, err := lm.readInstanceManifest(id)
	if err == nil {
		return manifest, nil
	}
	directErr := err
	manifest, err = lm.findInstanceByName(id)
	if err != nil {
		return localMacOSInstanceManifest{}, err
	}
	if manifest.ID != "" {
		return manifest, nil
	}
	return localMacOSInstanceManifest{}, directErr
}

func (lm *localMacOS) ensureNetworkManifest(manifest *localMacOSInstanceManifest) error {
	if manifest == nil {
		return fmt.Errorf("local macOS VM manifest is nil")
	}
	if manifest.Network.IPAddress == "" {
		network, err := lm.newNetworkManifest(manifest.ID)
		if err != nil {
			return err
		}
		manifest.Network = network
	}
	if manifest.Network.Interface == "" {
		manifest.Network.Interface = localMacOSNetworkInterface
	}
	if manifest.Network.PrefixLength == 0 {
		manifest.Network.PrefixLength = localMacOSNetworkPrefix
	}
	if manifest.Network.Gateway == "" {
		manifest.Network.Gateway = localMacOSNetworkGateway
	}
	if len(manifest.Network.DNSServers) == 0 {
		manifest.Network.DNSServers = defaultLocalMacOSDNSServers()
	}
	if manifest.PublicIP == "" {
		manifest.PublicIP = manifest.Network.IPAddress
	}
	return nil
}

func (lm *localMacOS) newNetworkManifest(id string) (localMacOSNetworkManifest, error) {
	ipAddress, err := lm.allocateStaticIPAddress(id)
	if err != nil {
		return localMacOSNetworkManifest{}, err
	}
	return localMacOSNetworkManifest{
		Interface:    localMacOSNetworkInterface,
		IPAddress:    ipAddress,
		PrefixLength: localMacOSNetworkPrefix,
		Gateway:      localMacOSNetworkGateway,
		DNSServers:   defaultLocalMacOSDNSServers(),
	}, nil
}

func defaultLocalMacOSDNSServers() []string {
	return []string{"8.8.8.8", "1.1.1.1"}
}

func (lm *localMacOS) writeInstanceManifest(manifest localMacOSInstanceManifest) error {
	if err := os.MkdirAll(lm.instanceDir(manifest.ID), 0755); err != nil {
		return err
	}
	return writeJSONFile(lm.instanceManifestPath(manifest.ID), manifest)
}

func (lm *localMacOS) readVolumeManifest(id string) (localMacOSVolumeManifest, error) {
	id, err := sanitizeLocalMacOSName(id)
	if err != nil {
		return localMacOSVolumeManifest{}, err
	}
	var manifest localMacOSVolumeManifest
	if err := readJSONFile(lm.volumeManifestPath(id), &manifest); err != nil {
		return localMacOSVolumeManifest{}, err
	}
	return manifest, nil
}

func (lm *localMacOS) findInstanceByName(name string) (localMacOSInstanceManifest, error) {
	entries, err := os.ReadDir(lm.instancesDir())
	if err != nil {
		if os.IsNotExist(err) {
			return localMacOSInstanceManifest{}, nil
		}
		return localMacOSInstanceManifest{}, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifest, err := lm.readInstanceManifest(entry.Name())
		if err != nil {
			log.Warnf("failed to read local macOS VM '%s': %s", entry.Name(), err.Error())
			continue
		}
		if manifest.Name == name {
			return manifest, nil
		}
	}
	return localMacOSInstanceManifest{}, nil
}

func (lm *localMacOS) allocateStaticIPAddress(id string) (string, error) {
	_, network, err := net.ParseCIDR(localMacOSNetworkCIDR)
	if err != nil {
		return "", err
	}
	base := network.IP.To4()
	if base == nil {
		return "", fmt.Errorf("local macOS network %s is not IPv4", localMacOSNetworkCIDR)
	}

	used := map[string]struct{}{
		localMacOSNetworkGateway: {},
	}
	for ip := range lm.localMacOSManifestIPs(id) {
		used[ip] = struct{}{}
	}
	for ip := range localMacOSDHCPLeaseIPs() {
		used[ip] = struct{}{}
	}
	for ip := range localMacOSARPIPs() {
		used[ip] = struct{}{}
	}

	width := localMacOSStaticIPRangeTo - localMacOSStaticIPRangeFrom + 1
	start := localMacOSStaticIPRangeFrom
	if id != "" {
		hash := sha256.Sum256([]byte(id))
		start += int(hash[0]) % width
	}
	for offset := 0; offset < width; offset++ {
		host := localMacOSStaticIPRangeFrom + ((start - localMacOSStaticIPRangeFrom + offset) % width)
		ip := net.IPv4(base[0], base[1], base[2], byte(host)).String()
		if _, found := used[ip]; found {
			continue
		}
		return ip, nil
	}
	return "", fmt.Errorf("no free local macOS static IPs in %s host range %d-%d", localMacOSNetworkCIDR, localMacOSStaticIPRangeFrom, localMacOSStaticIPRangeTo)
}

func (lm *localMacOS) localMacOSManifestIPs(excludeID string) map[string]struct{} {
	used := map[string]struct{}{}
	entries, err := os.ReadDir(lm.instancesDir())
	if err != nil {
		return used
	}
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == excludeID {
			continue
		}
		manifest, err := lm.readInstanceManifest(entry.Name())
		if err != nil {
			continue
		}
		for _, ip := range []string{manifest.Network.IPAddress, manifest.PublicIP} {
			ip = strings.TrimSpace(ip)
			if ip == "" {
				continue
			}
			used[ip] = struct{}{}
		}
	}
	return used
}

func (lm *localMacOS) applyHostAgentVM(id string, desiredState string) (*hostagentpb.VMObservedState, error) {
	client, err := hostagentclient.New()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	state, err := client.ApplyVM(id, lm.instanceManifestPath(id), desiredState)
	if err != nil {
		return nil, fmt.Errorf("host agent is unavailable; start it with sudo protos-hostagent: %w", err)
	}
	if state.GetStatus() == "error" {
		return nil, fmt.Errorf("host agent failed to apply state '%s' for VM '%s': %s", desiredState, id, state.GetMessage())
	}
	return state, nil
}

func (lm *localMacOS) hostAgentVMStatus(id string) (*hostagentpb.VMObservedState, error) {
	client, err := hostagentclient.New()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	state, err := client.VMStatus(id, lm.instanceManifestPath(id))
	if err != nil {
		return nil, fmt.Errorf("host agent is unavailable; start it with sudo protos-hostagent: %w", err)
	}
	return state, nil
}

func validateLocalMacOSLocation(location string) error {
	if location == "" || location == localMacOSLocation {
		return nil
	}
	return fmt.Errorf("local macOS provider only supports location '%s'", localMacOSLocation)
}

func resolveLocalMacOSImageSource(imagePath string) (localMacOSImageSource, error) {
	imagePath, err := expandLocalMacOSPath(imagePath)
	if err != nil {
		return localMacOSImageSource{}, err
	}

	if stat, err := os.Stat(imagePath); err == nil && stat.IsDir() {
		return resolveLocalMacOSImageSourceInDir(imagePath)
	}

	if strings.HasSuffix(strings.ToLower(imagePath), ".iso") {
		source := localMacOSImageSource{
			bootISO:  imagePath,
			rootDisk: firstExistingLocalMacOSPath(strings.TrimSuffix(imagePath, ".iso")+"-disk.img", strings.TrimSuffix(imagePath, ".iso")+"-root.raw"),
		}
		if err := validateLocalMacOSImageSource(source); err != nil {
			return localMacOSImageSource{}, err
		}
		return source, nil
	}

	for _, suffix := range []string{"-kernel", "-initrd.img", "-cmdline"} {
		if strings.HasSuffix(imagePath, suffix) {
			imagePath = strings.TrimSuffix(imagePath, suffix)
			break
		}
	}
	source := localMacOSImageSource{
		kernel:  imagePath + "-kernel",
		initrd:  imagePath + "-initrd.img",
		cmdline: imagePath + "-cmdline",
	}
	if err := validateLocalMacOSImageSource(source); err == nil {
		source.rootDisk = firstExistingLocalMacOSPath(imagePath+"-disk.img", imagePath+"-root.raw")
		return source, nil
	}
	return localMacOSImageSource{}, fmt.Errorf("could not find LinuxKit kernel/initrd files for image path '%s'", imagePath)
}

func resolveLocalMacOSImageSourceInDir(dir string) (localMacOSImageSource, error) {
	base := filepath.Join(dir, filepath.Base(dir))
	source := localMacOSImageSource{
		bootISO: firstExistingLocalMacOSPath(
			filepath.Join(dir, localMacOSImageBootISO),
			base+"-efi-initrd.iso",
			base+"-efi.iso",
			base+".iso",
		),
		kernel: firstExistingLocalMacOSPath(
			filepath.Join(dir, localMacOSImageKernel),
			filepath.Join(dir, "vmlinux"),
			base+"-kernel",
		),
		initrd: firstExistingLocalMacOSPath(
			filepath.Join(dir, localMacOSImageInitrd),
			filepath.Join(dir, "initrd"),
			base+"-initrd.img",
		),
		cmdline: firstExistingLocalMacOSPath(
			filepath.Join(dir, localMacOSImageCmdline),
			base+"-cmdline",
		),
		rootDisk: firstExistingLocalMacOSPath(
			filepath.Join(dir, localMacOSImageRootDisk),
			filepath.Join(dir, "disk.img"),
			base+"-disk.img",
			base+"-root.raw",
		),
	}
	if err := validateLocalMacOSImageSource(source); err != nil {
		return localMacOSImageSource{}, err
	}
	return source, nil
}

func validateLocalMacOSImageSource(source localMacOSImageSource) error {
	if source.bootISO == "" {
		if source.kernel == "" {
			return fmt.Errorf("missing kernel file")
		}
		if source.initrd == "" {
			return fmt.Errorf("missing initrd file")
		}
	}
	for _, path := range []string{source.bootISO, source.kernel, source.initrd} {
		if path == "" {
			continue
		}
		if stat, err := os.Stat(path); err != nil {
			return err
		} else if stat.IsDir() {
			return fmt.Errorf("image component '%s' is a directory", path)
		}
	}
	if source.cmdline != "" {
		if stat, err := os.Stat(source.cmdline); err != nil {
			return err
		} else if stat.IsDir() {
			return fmt.Errorf("cmdline component '%s' is a directory", source.cmdline)
		}
	}
	if source.rootDisk != "" {
		if stat, err := os.Stat(source.rootDisk); err != nil {
			return err
		} else if stat.IsDir() {
			return fmt.Errorf("root disk component '%s' is a directory", source.rootDisk)
		}
	}
	return nil
}

func firstExistingLocalMacOSPath(paths ...string) string {
	for _, path := range paths {
		if path == "" {
			continue
		}
		if stat, err := os.Stat(path); err == nil && !stat.IsDir() {
			return path
		}
	}
	return ""
}

func firstNonEmptyLocalMacOSString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func writeLocalMacOSMetadataISO(path string, hostname string, authorizedKey string, network localMacOSNetworkManifest) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmpDir, err := os.MkdirTemp("", "protos-local-macos-metadata-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	userData := map[string]localMacOSMetadataEntry{
		"hostname": {
			Content: hostname + "\n",
			Perm:    "0644",
		},
		"ssh": {
			Entries: map[string]localMacOSMetadataEntry{
				"authorized_keys": {
					Content: strings.TrimSpace(authorizedKey) + "\n",
					Perm:    "0600",
				},
			},
		},
		"protos": {
			Entries: map[string]localMacOSMetadataEntry{
				"network": {
					Entries: map[string]localMacOSMetadataEntry{
						"interface": {
							Content: network.Interface + "\n",
							Perm:    "0644",
						},
						"address": {
							Content: fmt.Sprintf("%s/%d\n", network.IPAddress, network.PrefixLength),
							Perm:    "0644",
						},
						"gateway": {
							Content: network.Gateway + "\n",
							Perm:    "0644",
						},
						"dns": {
							Content: strings.Join(network.DNSServers, "\n") + "\n",
							Perm:    "0644",
						},
					},
				},
			},
		},
	}
	data, err := json.Marshal(userData)
	if err != nil {
		return err
	}

	configPath := filepath.Join(tmpDir, "config")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	if mkisofs, err := exec.LookPath("mkisofs"); err == nil {
		cmd := exec.Command(mkisofs, "-quiet", "-J", "-r", "-V", "CIDATA", "-o", path, tmpDir)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to write metadata ISO with mkisofs: %w: %s", err, strings.TrimSpace(string(output)))
		}
		return nil
	}
	cmd := exec.Command("hdiutil", "makehybrid", "-quiet", "-iso", "-joliet", "-default-volume-name", "CIDATA", "-o", path, tmpDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to write metadata ISO with hdiutil: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func writeJSONFile(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

func readJSONFile(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}

func copyLocalMacOSFile(src string, dst string) error {
	if err := cloneLocalMacOSFile(src, dst); err == nil {
		return nil
	} else {
		log.Debugf("failed to clone local macOS file %s to %s, falling back to stream copy: %v", src, dst, err)
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open '%s': %w", src, err)
	}
	defer srcFile.Close()
	info, err := srcFile.Stat()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("failed to create '%s': %w", dst, err)
	}
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		_ = dstFile.Close()
		return err
	}
	return dstFile.Close()
}

func cloneLocalMacOSFile(src string, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		return err
	}
	return unix.Clonefile(src, dst, 0)
}

func expandLocalMacOSPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			path = homeDir
		} else {
			path = filepath.Join(homeDir, path[2:])
		}
	}
	return filepath.Abs(path)
}

func sanitizeLocalMacOSName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("name is empty")
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	clean := strings.Trim(b.String(), ".-")
	if clean == "" || clean == "." || clean == ".." {
		return "", fmt.Errorf("name '%s' is not usable as a local macOS VM identifier", name)
	}
	return clean, nil
}

func newLocalMacOSID(prefix string) (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(b[:])), nil
}

func newLocalMacOSMACAddress() (string, error) {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[0] = (b[0] | 0x02) & 0xfe
	return net.HardwareAddr(b[:]).String(), nil
}

func localMacOSProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}

func localMacOSDHCPLeaseIPs() map[string]struct{} {
	ips := map[string]struct{}{}
	data, err := os.ReadFile("/var/db/dhcpd_leases")
	if err != nil {
		return ips
	}
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if !strings.HasPrefix(line, "ip_address=") {
			continue
		}
		ip := strings.TrimPrefix(line, "ip_address=")
		if parsed := net.ParseIP(ip); parsed != nil {
			ips[ip] = struct{}{}
		}
	}
	return ips
}

func localMacOSARPIPs() map[string]struct{} {
	ips := map[string]struct{}{}
	output, err := exec.Command("arp", "-an").Output()
	if err != nil {
		return ips
	}
	for _, line := range strings.Split(string(output), "\n") {
		start := strings.Index(line, "(")
		end := strings.Index(line, ")")
		if start < 0 || end <= start {
			continue
		}
		ip := line[start+1 : end]
		if parsed := net.ParseIP(ip); parsed != nil {
			ips[ip] = struct{}{}
		}
	}
	return ips
}

func downloadLocalMacOSImage(url string, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected HTTP status %s", resp.Status)
	}
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func sha256LocalMacOSFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func isLocalMacOSArchive(path string) bool {
	path = strings.ToLower(path)
	return strings.HasSuffix(path, ".tar") || strings.HasSuffix(path, ".tar.gz") || strings.HasSuffix(path, ".tgz")
}

func extractLocalMacOSArchive(archivePath string, dstDir string, originalName string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var reader io.Reader = file
	if strings.HasSuffix(strings.ToLower(originalName), ".tar.gz") || strings.HasSuffix(strings.ToLower(originalName), ".tgz") {
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		cleanName := filepath.Clean(header.Name)
		if strings.HasPrefix(cleanName, "..") || filepath.IsAbs(cleanName) {
			return fmt.Errorf("archive contains unsafe path '%s'", header.Name)
		}
		target := filepath.Join(dstDir, cleanName)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode).Perm()); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode).Perm())
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tarReader); err != nil {
				_ = out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		}
	}
}
