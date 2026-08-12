//go:build darwin

package localmacos

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
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
	"sort"
	"strings"
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
	localMacOSMetadataFile  = "metadata.json"
	localMacOSDiagMaxBytes  = 8192

	localMacOSNetworkInterface    = "eth0"
	localMacOSFallbackNetworkCIDR = "192.168.64.0/24"
	localMacOSFallbackGateway     = "192.168.64.1"
)

var localMacOSAuthFields = []string{}
var log = util.GetLogger("provisioner.localmacos")

const Type = provisioners.Type("local_macos")

type localMacOSCredentials struct {
	vmDir string
}

type localMacOS struct {
	provisioners.ProvisionerMetadata

	vmDir              string
	newHostAgentClient func() (localMacOSHostAgent, error)
}

type localMacOSHostAgent interface {
	Close() error
	ApplyVM(id string, rootDir string, desiredState string, config *hostagentpb.VMConfig) (*hostagentpb.VMObservedState, error)
	VMStatus(id string, name string, rootDir string) (*hostagentpb.VMObservedState, error)
	ListVMs(rootDir string) ([]*hostagentpb.VMObservedState, error)
	VMLogs(id string, name string, rootDir string) (string, error)
}

func newLocalMacOSHostAgentClient() (localMacOSHostAgent, error) {
	return hostagentclient.New()
}

type localMacOSImageMetadata struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Location     string `json:"location"`
	KernelPath   string `json:"kernel_path"`
	InitrdPath   string `json:"initrd_path"`
	CmdlinePath  string `json:"cmdline_path"`
	RootDiskPath string `json:"root_disk_path,omitempty"`
	BootISOPath  string `json:"boot_iso_path,omitempty"`
}

type localMacOSVolumeMetadata struct {
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

type localMacOSNetworkConfig struct {
	Interface    string   `json:"interface"`
	IPAddress    string   `json:"ip_address"`
	PrefixLength int      `json:"prefix_length"`
	Gateway      string   `json:"gateway"`
	DNSServers   []string `json:"dns_servers,omitempty"`
}

type localMacOSVMConfig struct {
	ID                  string                     `json:"id"`
	Name                string                     `json:"name"`
	ArtifactDir         string                     `json:"-"`
	ImageID             string                     `json:"image_id"`
	Location            string                     `json:"location"`
	MachineType         string                     `json:"machine_type"`
	Cores               uint32                     `json:"cores"`
	MemoryMiB           uint32                     `json:"memory_mib"`
	InitOriginPublicKey string                     `json:"init_origin_public_key"`
	PublicIP            string                     `json:"public_ip"`
	MACAddress          string                     `json:"mac_address"`
	KernelPath          string                     `json:"kernel_path"`
	InitrdPath          string                     `json:"initrd_path"`
	CmdlinePath         string                     `json:"cmdline_path"`
	RootDiskPath        string                     `json:"root_disk_path,omitempty"`
	BootISOPath         string                     `json:"boot_iso_path,omitempty"`
	MetadataISO         string                     `json:"metadata_iso,omitempty"`
	Network             localMacOSNetworkConfig    `json:"network,omitempty"`
	Volumes             []localMacOSAttachedVolume `json:"volumes,omitempty"`
}

type localMacOSNATNetwork struct {
	Gateway      net.IP
	Network      *net.IPNet
	PrefixLength int
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
		newHostAgentClient:  newLocalMacOSHostAgentClient,
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

func (lm *localMacOS) NewInstance(name string, imageID string, originPublicKey string, machineType string, location string) (string, error) {
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

	image, err := lm.readImageMetadata(imageID)
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
	network, err := lm.newNetworkConfig(id)
	if err != nil {
		return "", err
	}

	vmConfig := localMacOSVMConfig{
		ID:                  id,
		Name:                name,
		ArtifactDir:         instanceDir,
		ImageID:             imageID,
		Location:            localMacOSLocation,
		MachineType:         machineType,
		Cores:               machine.Cores,
		MemoryMiB:           machine.Memory,
		InitOriginPublicKey: strings.TrimSpace(originPublicKey),
		PublicIP:            network.IPAddress,
		MACAddress:          macAddress,
		KernelPath:          kernelPath,
		InitrdPath:          initrdPath,
		CmdlinePath:         cmdlinePath,
		RootDiskPath:        rootDiskPath,
		BootISOPath:         bootISOPath,
		Network:             network,
	}
	if _, err := lm.applyHostAgentVM(id, "configured", &vmConfig); err != nil {
		return "", err
	}
	return id, nil
}

func (lm *localMacOS) DeleteInstance(id string, location string) error {
	if err := validateLocalMacOSLocation(location); err != nil {
		return err
	}
	state, err := lm.hostAgentVMStateByIDOrName(id)
	if err != nil {
		if errors.Is(err, provisioners.ErrInstanceNotFound) {
			return lm.removeInstanceArtifacts(id)
		}
		return fmt.Errorf("failed to inspect local macOS VM '%s' before deletion: %w", id, err)
	}
	config, err := lm.vmConfigFromState(state)
	if err != nil {
		return fmt.Errorf("failed to load local macOS VM '%s' before deletion: %w", id, err)
	}
	instanceID, err := sanitizeLocalMacOSName(config.ID)
	if err != nil {
		return fmt.Errorf("invalid local macOS VM identity returned by host agent: %w", err)
	}
	if _, err := lm.applyHostAgentVM(instanceID, "deleted", nil); err != nil {
		return fmt.Errorf("failed to delete local macOS VM '%s' through host agent: %w", instanceID, err)
	}
	return lm.removeInstanceArtifacts(instanceID)
}

func (lm *localMacOS) removeInstanceArtifacts(id string) error {
	instanceID, err := sanitizeLocalMacOSName(id)
	if err != nil {
		return err
	}
	if err := removeLocalMacOSDir(lm.instanceDir(instanceID)); err != nil {
		return fmt.Errorf("failed to remove local macOS VM '%s': %w", id, err)
	}
	return nil
}

func (lm *localMacOS) StartInstance(id string, location string) error {
	if err := validateLocalMacOSLocation(location); err != nil {
		return err
	}
	state, err := lm.hostAgentVMStateByIDOrName(id)
	if err != nil {
		return err
	}
	if strings.EqualFold(state.GetStatus(), provisioners.ServerStateRunning) && state.GetPid() != 0 {
		return nil
	}
	vmConfig, err := lm.vmConfigFromState(state)
	if err != nil {
		return err
	}
	if err := lm.ensureNetworkConfig(&vmConfig); err != nil {
		return err
	}
	vmConfig.PublicIP = vmConfig.Network.IPAddress
	vmConfig.MetadataISO = filepath.Join(lm.instanceDir(vmConfig.ID), localMacOSMetadataISO)
	if err := writeLocalMacOSMetadataISO(vmConfig.MetadataISO, vmConfig.Name, vmConfig.InitOriginPublicKey, vmConfig.Network); err != nil {
		return fmt.Errorf("failed to write metadata ISO: %w", err)
	}

	if _, err := lm.applyHostAgentVM(vmConfig.ID, provisioners.ServerStateRunning, &vmConfig); err != nil {
		return err
	}
	return nil
}

func (lm *localMacOS) StopInstance(id string, location string) error {
	if err := validateLocalMacOSLocation(location); err != nil {
		return err
	}
	state, err := lm.hostAgentVMStateByIDOrName(id)
	if err != nil {
		return err
	}
	vmConfig, err := lm.vmConfigFromState(state)
	if err != nil {
		return err
	}
	_, err = lm.applyHostAgentVM(vmConfig.ID, provisioners.ServerStateStopped, nil)
	return err
}

func (lm *localMacOS) GetInstanceInfo(id string, location string) (provisioners.InstanceInfo, error) {
	if err := validateLocalMacOSLocation(location); err != nil {
		return provisioners.InstanceInfo{}, err
	}
	state, err := lm.hostAgentVMStateByIDOrName(id)
	if err != nil {
		return provisioners.InstanceInfo{}, err
	}
	vmConfig, err := lm.vmConfigFromState(state)
	if err != nil {
		return provisioners.InstanceInfo{}, err
	}

	info := provisioners.InstanceInfo{
		ID:                 vmConfig.ID,
		Name:               vmConfig.Name,
		PublicIP:           firstNonEmptyLocalMacOSString(state.GetPublicIp(), vmConfig.PublicIP, vmConfig.Network.IPAddress),
		Kind:               provisioners.KindLocalVM,
		KindID:             lm.NameStr(),
		ProviderResourceID: vmConfig.ID,
		Location:           localMacOSLocation,
		Status:             state.GetStatus(),
	}
	if info.Status == "" {
		info.Status = provisioners.ServerStateStopped
	}
	for _, volume := range vmConfig.Volumes {
		info.Volumes = append(info.Volumes, provisioners.VolumeInfo{
			VolumeID: volume.ID,
			Name:     volume.Name,
			Size:     uint64(volume.SizeMiB) * 1048576,
		})
	}
	return info, nil
}

func (lm *localMacOS) DeploymentDiagnostics(id string, location string) (string, error) {
	if err := validateLocalMacOSLocation(location); err != nil {
		return "", err
	}
	state, err := lm.hostAgentVMStateByIDOrName(id)
	if err != nil {
		return "", err
	}
	vmConfig, err := lm.vmConfigFromState(state)
	if err != nil {
		return "", err
	}
	lines := []string{
		fmt.Sprintf("image: %s", firstNonEmptyLocalMacOSString(vmConfig.ImageID, "unknown")),
		fmt.Sprintf("network: %s/%d via %s", firstNonEmptyLocalMacOSString(vmConfig.Network.IPAddress, "unknown"), vmConfig.Network.PrefixLength, firstNonEmptyLocalMacOSString(vmConfig.Network.Gateway, "unknown")),
		fmt.Sprintf("mac: %s", firstNonEmptyLocalMacOSString(vmConfig.MACAddress, "unknown")),
	}
	logs, err := lm.hostAgentVMLogs(vmConfig.ID, vmConfig.Name)
	if err != nil {
		lines = append(lines, fmt.Sprintf("console: unavailable: %s", err.Error()))
		return strings.Join(lines, "\n"), nil
	}
	lines = append(lines, "host-agent console output:", tailLocalMacOSDiagnostics([]byte(logs), localMacOSDiagMaxBytes))
	return strings.Join(lines, "\n"), nil
}

func (lm *localMacOS) InstanceLogs(id string, location string) (string, error) {
	return lm.DeploymentDiagnostics(id, location)
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

	return lm.UploadLocalImage(context.Background(), imagePath, version, location, 0, nil)
}

func (lm *localMacOS) UploadLocalImage(ctx context.Context, imagePath string, imageName string, location string, timeout time.Duration, progress provisioners.UploadProgressFunc) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
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
	if _, err := lm.readImageMetadata(imageID); err == nil {
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
		if err := copyLocalMacOSFileWithProgress(ctx, source.kernel, kernelPath, "kernel", progress); err != nil {
			return "", err
		}
		if err := copyLocalMacOSFileWithProgress(ctx, source.initrd, initrdPath, "initrd", progress); err != nil {
			return "", err
		}
		if source.cmdline != "" {
			if err := copyLocalMacOSFileWithProgress(ctx, source.cmdline, cmdlinePath, "cmdline", progress); err != nil {
				return "", err
			}
		} else if err := os.WriteFile(cmdlinePath, []byte("console=hvc0\n"), 0644); err != nil {
			return "", fmt.Errorf("failed to write default cmdline: %w", err)
		}
	}

	bootISOPath := ""
	if source.bootISO != "" {
		bootISOPath = filepath.Join(imageDir, localMacOSImageBootISO)
		if err := copyLocalMacOSFileWithProgress(ctx, source.bootISO, bootISOPath, "boot_iso", progress); err != nil {
			return "", err
		}
	}

	rootDiskPath := ""
	if source.rootDisk != "" {
		rootDiskPath = filepath.Join(imageDir, localMacOSImageRootDisk)
		if err := copyLocalMacOSFileWithProgress(ctx, source.rootDisk, rootDiskPath, "root_disk", progress); err != nil {
			return "", err
		}
	}

	metadata := localMacOSImageMetadata{
		ID:           imageID,
		Name:         imageName,
		Location:     localMacOSLocation,
		KernelPath:   kernelPath,
		InitrdPath:   initrdPath,
		CmdlinePath:  cmdlinePath,
		RootDiskPath: rootDiskPath,
		BootISOPath:  bootISOPath,
	}
	if err := writeJSONFile(filepath.Join(imageDir, localMacOSMetadataFile), metadata); err != nil {
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
	if _, err := lm.readImageMetadata(imageID); err != nil {
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
	metadata := localMacOSVolumeMetadata{
		ID:       id,
		Name:     name,
		Path:     path,
		SizeMiB:  size,
		Location: localMacOSLocation,
	}
	if err := writeJSONFile(lm.volumeMetadataPath(id), metadata); err != nil {
		return "", err
	}
	return id, nil
}

func (lm *localMacOS) DeleteVolume(id string, location string) error {
	if err := validateLocalMacOSLocation(location); err != nil {
		return err
	}
	metadata, err := lm.readVolumeMetadata(id)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		volumeID, sanitizeErr := sanitizeLocalMacOSName(id)
		if sanitizeErr != nil {
			return sanitizeErr
		}
		// Volume metadata can be removed before a task process crashes. The
		// canonical raw path remains derivable from the volume ID, including
		// while a stopped VM manifest still references that volume.
		metadata = localMacOSVolumeMetadata{
			ID:   volumeID,
			Path: filepath.Join(lm.volumesDir(), volumeID+".raw"),
		}
	}
	if err := os.Remove(metadata.Path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete local macOS volume '%s': %w", id, err)
	}
	if err := os.Remove(lm.volumeMetadataPath(id)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete local macOS volume metadata '%s': %w", id, err)
	}
	return nil
}

func (lm *localMacOS) AttachVolume(volumeID string, instanceID string, location string) error {
	if err := validateLocalMacOSLocation(location); err != nil {
		return err
	}
	state, err := lm.hostAgentVMStateByIDOrName(instanceID)
	if err != nil {
		return err
	}
	if strings.EqualFold(state.GetStatus(), provisioners.ServerStateRunning) || state.GetPid() != 0 {
		return fmt.Errorf("cannot attach local macOS volume '%s' to running VM '%s'", volumeID, instanceID)
	}
	instance, err := lm.vmConfigFromState(state)
	if err != nil {
		return err
	}
	volume, err := lm.readVolumeMetadata(volumeID)
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
	_, err = lm.applyHostAgentVM(instance.ID, "configured", &instance)
	return err
}

func (lm *localMacOS) DettachVolume(volumeID string, instanceID string, location string) error {
	if err := validateLocalMacOSLocation(location); err != nil {
		return err
	}
	state, err := lm.hostAgentVMStateByIDOrName(instanceID)
	if err != nil {
		return err
	}
	if strings.EqualFold(state.GetStatus(), provisioners.ServerStateRunning) || state.GetPid() != 0 {
		return fmt.Errorf("cannot detach local macOS volume '%s' from running VM '%s'", volumeID, instanceID)
	}
	instance, err := lm.vmConfigFromState(state)
	if err != nil {
		return err
	}
	volumes := instance.Volumes[:0]
	for _, volume := range instance.Volumes {
		if volume.ID != volumeID {
			volumes = append(volumes, volume)
		}
	}
	instance.Volumes = volumes
	_, err = lm.applyHostAgentVM(instance.ID, "configured", &instance)
	return err
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

func tailLocalMacOSDiagnostics(data []byte, limit int) string {
	if limit <= 0 || len(data) <= limit {
		return string(data)
	}
	data = data[len(data)-limit:]
	if idx := bytes.IndexByte(data, '\n'); idx >= 0 && idx+1 < len(data) {
		data = data[idx+1:]
	}
	return fmt.Sprintf("[last %d bytes]\n%s", limit, string(data))
}

func (lm *localMacOS) volumeMetadataPath(id string) string {
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
		metadata, err := lm.readImageMetadata(entry.Name())
		if err != nil {
			log.Warnf("failed to read local macOS image '%s': %s", entry.Name(), err.Error())
			continue
		}
		var updatedAt time.Time
		if info, err := os.Stat(filepath.Join(lm.imageDir(entry.Name()), localMacOSMetadataFile)); err == nil {
			updatedAt = info.ModTime()
		}
		images[metadata.ID] = provisioners.ImageInfo{
			ID:        metadata.ID,
			Name:      metadata.Name,
			Location:  metadata.Location,
			UpdatedAt: updatedAt,
		}
	}
	return images, nil
}

func (lm *localMacOS) readImageMetadata(id string) (localMacOSImageMetadata, error) {
	id, err := sanitizeLocalMacOSName(id)
	if err != nil {
		return localMacOSImageMetadata{}, err
	}
	var metadata localMacOSImageMetadata
	if err := readJSONFile(filepath.Join(lm.imageDir(id), localMacOSMetadataFile), &metadata); err != nil {
		return localMacOSImageMetadata{}, err
	}
	return metadata, nil
}

func (lm *localMacOS) ensureNetworkConfig(config *localMacOSVMConfig) error {
	if config == nil {
		return fmt.Errorf("local macOS VM config is nil")
	}
	natNetwork, err := localMacOSCurrentNATNetwork()
	if err != nil {
		return err
	}
	ip := net.ParseIP(strings.TrimSpace(config.Network.IPAddress))
	if ip == nil || !natNetwork.Network.Contains(ip) {
		network, err := lm.newNetworkConfig(config.ID)
		if err != nil {
			return err
		}
		config.Network = network
	} else {
		config.Network.PrefixLength = natNetwork.PrefixLength
		config.Network.Gateway = natNetwork.Gateway.String()
	}
	if config.Network.Interface == "" {
		config.Network.Interface = localMacOSNetworkInterface
	}
	if len(config.Network.DNSServers) == 0 {
		config.Network.DNSServers = defaultLocalMacOSDNSServers()
	}
	config.PublicIP = config.Network.IPAddress
	return nil
}

func (lm *localMacOS) newNetworkConfig(id string) (localMacOSNetworkConfig, error) {
	natNetwork, err := localMacOSCurrentNATNetwork()
	if err != nil {
		return localMacOSNetworkConfig{}, err
	}
	ipAddress, err := lm.allocateStaticIPAddress(id, natNetwork)
	if err != nil {
		return localMacOSNetworkConfig{}, err
	}
	return localMacOSNetworkConfig{
		Interface:    localMacOSNetworkInterface,
		IPAddress:    ipAddress,
		PrefixLength: natNetwork.PrefixLength,
		Gateway:      natNetwork.Gateway.String(),
		DNSServers:   defaultLocalMacOSDNSServers(),
	}, nil
}

func defaultLocalMacOSDNSServers() []string {
	return []string{"8.8.8.8", "1.1.1.1"}
}

func (lm *localMacOS) readVolumeMetadata(id string) (localMacOSVolumeMetadata, error) {
	id, err := sanitizeLocalMacOSName(id)
	if err != nil {
		return localMacOSVolumeMetadata{}, err
	}
	var metadata localMacOSVolumeMetadata
	if err := readJSONFile(lm.volumeMetadataPath(id), &metadata); err != nil {
		return localMacOSVolumeMetadata{}, err
	}
	return metadata, nil
}

func (lm *localMacOS) findInstanceByName(name string) (localMacOSVMConfig, error) {
	states, err := lm.hostAgentVMs()
	if err != nil {
		return localMacOSVMConfig{}, err
	}
	for _, state := range states {
		config, err := lm.vmConfigFromState(state)
		if err != nil {
			continue
		}
		if config.Name == name {
			return config, nil
		}
	}
	return localMacOSVMConfig{}, nil
}

func localMacOSCurrentNATNetwork() (localMacOSNATNetwork, error) {
	if observed, err := localMacOSPreferredVZNATNetwork(); err == nil {
		return observed, nil
	} else {
		log.Debugf("failed to inspect preferred local macOS VZ NAT network: %s", err.Error())
	}
	return localMacOSFallbackNATNetwork()
}

func localMacOSPreferredVZNATNetwork() (localMacOSNATNetwork, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return localMacOSNATNetwork{}, err
	}
	names := make([]string, 0, len(interfaces))
	for _, iface := range interfaces {
		if strings.HasPrefix(iface.Name, "bridge") {
			names = append(names, iface.Name)
		}
	}
	sort.Strings(names)

	fallbackGateway := net.ParseIP(localMacOSFallbackGateway).To4()
	if fallbackGateway == nil {
		return localMacOSNATNetwork{}, fmt.Errorf("fallback gateway %q is not IPv4", localMacOSFallbackGateway)
	}
	for _, name := range names {
		network, err := localMacOSBridgeNATNetwork(name)
		if err != nil {
			continue
		}
		if network.Gateway.Equal(fallbackGateway) || network.Network.Contains(fallbackGateway) {
			return network, nil
		}
	}
	return localMacOSNATNetwork{}, fmt.Errorf("no bridge interface matched preferred VZ NAT gateway %s", fallbackGateway.String())
}

func localMacOSBridgeNATNetwork(name string) (localMacOSNATNetwork, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return localMacOSNATNetwork{}, err
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return localMacOSNATNetwork{}, err
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet == nil {
			continue
		}
		gateway := ipNet.IP.To4()
		if gateway == nil {
			continue
		}
		prefix, bits := ipNet.Mask.Size()
		if bits != 32 || prefix <= 0 || prefix >= 31 {
			continue
		}
		networkIP := append(net.IP(nil), gateway...)
		networkIP = networkIP.Mask(ipNet.Mask)
		return localMacOSNATNetwork{
			Gateway:      append(net.IP(nil), gateway...),
			Network:      &net.IPNet{IP: networkIP, Mask: append(net.IPMask(nil), ipNet.Mask...)},
			PrefixLength: prefix,
		}, nil
	}
	return localMacOSNATNetwork{}, fmt.Errorf("interface has no usable IPv4 address")
}

func localMacOSFallbackNATNetwork() (localMacOSNATNetwork, error) {
	_, network, err := net.ParseCIDR(localMacOSFallbackNetworkCIDR)
	if err != nil {
		return localMacOSNATNetwork{}, err
	}
	gateway := net.ParseIP(localMacOSFallbackGateway).To4()
	if gateway == nil {
		return localMacOSNATNetwork{}, fmt.Errorf("fallback gateway %q is not IPv4", localMacOSFallbackGateway)
	}
	prefix, bits := network.Mask.Size()
	if bits != 32 {
		return localMacOSNATNetwork{}, fmt.Errorf("fallback network %q is not IPv4", localMacOSFallbackNetworkCIDR)
	}
	return localMacOSNATNetwork{
		Gateway:      gateway,
		Network:      network,
		PrefixLength: prefix,
	}, nil
}

func (lm *localMacOS) allocateStaticIPAddress(id string, natNetwork localMacOSNATNetwork) (string, error) {
	used := map[string]struct{}{
		natNetwork.Gateway.String(): {},
	}
	for ip := range lm.localMacOSAssignedIPs(id) {
		used[ip] = struct{}{}
	}
	for ip := range localMacOSDHCPLeaseIPs() {
		used[ip] = struct{}{}
	}
	for ip := range localMacOSARPIPs() {
		used[ip] = struct{}{}
	}
	return allocateLocalMacOSStaticIP(natNetwork.Network, natNetwork.Gateway, id, used)
}

func allocateLocalMacOSStaticIP(network *net.IPNet, gateway net.IP, id string, used map[string]struct{}) (string, error) {
	if network == nil {
		return "", fmt.Errorf("local macOS NAT network is nil")
	}
	base := network.IP.To4()
	if base == nil {
		return "", fmt.Errorf("local macOS NAT network %s is not IPv4", network.String())
	}
	prefix, bits := network.Mask.Size()
	if bits != 32 || prefix >= 31 {
		return "", fmt.Errorf("local macOS NAT network %s has no usable static host addresses", network.String())
	}
	networkUint := binary.BigEndian.Uint32(base)
	maskUint := binary.BigEndian.Uint32([]byte(network.Mask))
	broadcastUint := networkUint | ^maskUint
	firstUint := networkUint + 1
	lastUint := broadcastUint - 1
	if lastUint < firstUint {
		return "", fmt.Errorf("local macOS NAT network %s has no usable static host addresses", network.String())
	}

	width := uint64(lastUint-firstUint) + 1
	startOffset := (width * 3) / 4
	if id != "" {
		hash := sha256.Sum256([]byte(id))
		startOffset = (startOffset + uint64(binary.BigEndian.Uint16(hash[:2]))) % width
	}
	gatewayIP := gateway.To4()
	for offset := uint64(0); offset < width; offset++ {
		ipUint := firstUint + uint32((startOffset+offset)%width)
		ipBytes := make(net.IP, net.IPv4len)
		binary.BigEndian.PutUint32(ipBytes, ipUint)
		ip := ipBytes.String()
		if _, found := used[ip]; found {
			continue
		}
		if gatewayIP != nil && ipBytes.Equal(gatewayIP) {
			continue
		}
		return ip, nil
	}
	return "", fmt.Errorf("no free local macOS static IPs in %s", network.String())
}

func (lm *localMacOS) localMacOSAssignedIPs(excludeID string) map[string]struct{} {
	used := map[string]struct{}{}
	states, err := lm.hostAgentVMs()
	if err != nil {
		return used
	}
	for _, state := range states {
		config, err := lm.vmConfigFromState(state)
		if err != nil || config.ID == excludeID {
			continue
		}
		for _, ip := range []string{config.Network.IPAddress, config.PublicIP} {
			ip = strings.TrimSpace(ip)
			if ip == "" {
				continue
			}
			used[ip] = struct{}{}
		}
	}
	return used
}

func (lm *localMacOS) applyHostAgentVM(id string, desiredState string, config *localMacOSVMConfig) (*hostagentpb.VMObservedState, error) {
	client, err := lm.hostAgentClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	state, err := client.ApplyVM(id, lm.vmDir, desiredState, vmConfigToProto(config))
	if err != nil {
		return nil, fmt.Errorf("host agent is unavailable; start it through the Protos StartHostAgent API: %w", err)
	}
	if state.GetStatus() == "error" {
		return nil, fmt.Errorf("host agent failed to apply state '%s' for VM '%s': %s", desiredState, id, state.GetMessage())
	}
	return state, nil
}

func (lm *localMacOS) hostAgentVMStateByIDOrName(idOrName string) (*hostagentpb.VMObservedState, error) {
	state, err := lm.hostAgentVMStatus(idOrName, "")
	if err == nil && state.GetConfig() != nil {
		return state, nil
	}
	state, nameErr := lm.hostAgentVMStatus("", idOrName)
	if nameErr == nil && state.GetConfig() != nil {
		return state, nil
	}
	if err != nil {
		return nil, err
	}
	if nameErr != nil {
		return nil, nameErr
	}
	return nil, fmt.Errorf("%w: local macOS VM %q", provisioners.ErrInstanceNotFound, idOrName)
}

func (lm *localMacOS) hostAgentVMStatus(id string, name string) (*hostagentpb.VMObservedState, error) {
	client, err := lm.hostAgentClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	state, err := client.VMStatus(id, name, lm.vmDir)
	if err != nil {
		return nil, fmt.Errorf("host agent is unavailable; start it through the Protos StartHostAgent API: %w", err)
	}
	if strings.EqualFold(state.GetStatus(), "error") {
		return nil, fmt.Errorf("host agent failed to inspect local macOS VM '%s': %s", firstNonEmptyLocalMacOSString(id, name), state.GetMessage())
	}
	return state, nil
}

func (lm *localMacOS) hostAgentVMs() ([]*hostagentpb.VMObservedState, error) {
	client, err := lm.hostAgentClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	states, err := client.ListVMs(lm.vmDir)
	if err != nil {
		return nil, fmt.Errorf("host agent is unavailable; start it through the Protos StartHostAgent API: %w", err)
	}
	return states, nil
}

func (lm *localMacOS) hostAgentVMLogs(id string, name string) (string, error) {
	client, err := lm.hostAgentClient()
	if err != nil {
		return "", err
	}
	defer client.Close()

	logs, err := client.VMLogs(id, name, lm.vmDir)
	if err != nil {
		return "", fmt.Errorf("host agent is unavailable; start it through the Protos StartHostAgent API: %w", err)
	}
	return logs, nil
}

func (lm *localMacOS) hostAgentClient() (localMacOSHostAgent, error) {
	newClient := lm.newHostAgentClient
	if newClient == nil {
		newClient = newLocalMacOSHostAgentClient
	}
	return newClient()
}

func (lm *localMacOS) vmConfigFromState(state *hostagentpb.VMObservedState) (localMacOSVMConfig, error) {
	if state == nil || state.GetConfig() == nil {
		return localMacOSVMConfig{}, os.ErrNotExist
	}
	return vmConfigFromProto(state.GetConfig()), nil
}

func vmConfigToProto(config *localMacOSVMConfig) *hostagentpb.VMConfig {
	if config == nil {
		return nil
	}
	out := &hostagentpb.VMConfig{
		Id:                  config.ID,
		Name:                config.Name,
		ImageId:             config.ImageID,
		Location:            config.Location,
		MachineType:         config.MachineType,
		Cores:               config.Cores,
		MemoryMib:           config.MemoryMiB,
		InitOriginPublicKey: config.InitOriginPublicKey,
		PublicIp:            config.PublicIP,
		MacAddress:          config.MACAddress,
		KernelPath:          config.KernelPath,
		InitrdPath:          config.InitrdPath,
		CmdlinePath:         config.CmdlinePath,
		RootDiskPath:        config.RootDiskPath,
		BootIsoPath:         config.BootISOPath,
		MetadataIso:         config.MetadataISO,
		ArtifactDir:         config.ArtifactDir,
	}
	out.Network = &hostagentpb.VMNetworkConfig{
		Interface:    config.Network.Interface,
		IpAddress:    config.Network.IPAddress,
		PrefixLength: int32(config.Network.PrefixLength),
		Gateway:      config.Network.Gateway,
		DnsServers:   append([]string(nil), config.Network.DNSServers...),
	}
	for _, volume := range config.Volumes {
		out.Volumes = append(out.Volumes, &hostagentpb.VMVolume{
			Id:      volume.ID,
			Name:    volume.Name,
			Path:    volume.Path,
			SizeMib: int32(volume.SizeMiB),
		})
	}
	return out
}

func vmConfigFromProto(config *hostagentpb.VMConfig) localMacOSVMConfig {
	if config == nil {
		return localMacOSVMConfig{}
	}
	out := localMacOSVMConfig{
		ID:                  config.GetId(),
		Name:                config.GetName(),
		ArtifactDir:         config.GetArtifactDir(),
		ImageID:             config.GetImageId(),
		Location:            config.GetLocation(),
		MachineType:         config.GetMachineType(),
		Cores:               config.GetCores(),
		MemoryMiB:           config.GetMemoryMib(),
		InitOriginPublicKey: config.GetInitOriginPublicKey(),
		PublicIP:            config.GetPublicIp(),
		MACAddress:          config.GetMacAddress(),
		KernelPath:          config.GetKernelPath(),
		InitrdPath:          config.GetInitrdPath(),
		CmdlinePath:         config.GetCmdlinePath(),
		RootDiskPath:        config.GetRootDiskPath(),
		BootISOPath:         config.GetBootIsoPath(),
		MetadataISO:         config.GetMetadataIso(),
	}
	if network := config.GetNetwork(); network != nil {
		out.Network = localMacOSNetworkConfig{
			Interface:    network.GetInterface(),
			IPAddress:    network.GetIpAddress(),
			PrefixLength: int(network.GetPrefixLength()),
			Gateway:      network.GetGateway(),
			DNSServers:   append([]string(nil), network.GetDnsServers()...),
		}
	}
	for _, volume := range config.GetVolumes() {
		if volume == nil {
			continue
		}
		out.Volumes = append(out.Volumes, localMacOSAttachedVolume{
			ID:      volume.GetId(),
			Name:    volume.GetName(),
			Path:    volume.GetPath(),
			SizeMiB: int(volume.GetSizeMib()),
		})
	}
	return out
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

func writeLocalMacOSMetadataISO(path string, hostname string, initOriginPublicKey string, network localMacOSNetworkConfig) error {
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
		"protos": {
			Entries: map[string]localMacOSMetadataEntry{
				"init_origin_public_key": {
					Content: strings.TrimSpace(initOriginPublicKey) + "\n",
					Perm:    "0600",
				},
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

	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if info, err := os.Stat(path); err == nil {
		if err := os.Chmod(tmpPath, info.Mode().Perm()); err != nil {
			_ = os.Remove(tmpPath)
			return err
		}
	} else if os.IsNotExist(err) {
		if err := os.Chmod(tmpPath, 0644); err != nil {
			_ = os.Remove(tmpPath)
			return err
		}
	} else {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func readJSONFile(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}

func copyLocalMacOSFile(src string, dst string) error {
	return copyLocalMacOSFileWithProgress(context.Background(), src, dst, "", nil)
}

func copyLocalMacOSFileWithProgress(ctx context.Context, src string, dst string, phase string, progress provisioners.UploadProgressFunc) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := cloneLocalMacOSFile(src, dst); err == nil {
		if progress != nil {
			if info, statErr := os.Stat(dst); statErr == nil {
				if err := progress(provisioners.UploadProgress{
					Phase:            phase,
					Message:          "upload in progress",
					BytesTransferred: info.Size(),
					TotalBytes:       info.Size(),
				}); err != nil {
					return err
				}
			}
		}
		return nil
	} else {
		log.Debugf("failed to clone local macOS file %s to %s, falling back to stream copy: %v", src, dst, err)
	}
	if err := ctx.Err(); err != nil {
		return err
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
	reader := provisioners.NewUploadProgressReader(srcFile, info.Size(), phase, progress)
	if _, err := io.Copy(dstFile, reader); err != nil {
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
