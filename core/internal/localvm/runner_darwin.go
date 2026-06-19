//go:build darwin

package localvm

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/tmc/apple/foundation"
	vz "github.com/tmc/apple/virtualization"
	vzvm "github.com/tmc/apple/x/vzkit/vm"
)

const consoleLog = "console.log"

var debugConsoleFiles []*os.File

type attachedVolume struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	SizeMiB int    `json:"size_mib"`
}

type instanceManifest struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	ImageID      string           `json:"image_id"`
	Location     string           `json:"location"`
	MachineType  string           `json:"machine_type"`
	Cores        uint32           `json:"cores"`
	MemoryMiB    uint32           `json:"memory_mib"`
	PublicKey    string           `json:"public_key"`
	PublicIP     string           `json:"public_ip"`
	MACAddress   string           `json:"mac_address"`
	Status       string           `json:"status"`
	PID          int              `json:"pid,omitempty"`
	KernelPath   string           `json:"kernel_path"`
	InitrdPath   string           `json:"initrd_path"`
	CmdlinePath  string           `json:"cmdline_path"`
	RootDiskPath string           `json:"root_disk_path,omitempty"`
	BootISOPath  string           `json:"boot_iso_path,omitempty"`
	MetadataISO  string           `json:"metadata_iso,omitempty"`
	Volumes      []attachedVolume `json:"volumes,omitempty"`
}

// Run starts the VM described by manifestPath and blocks until it exits.
func Run(manifestPath string) error {
	manifest, err := readManifest(manifestPath)
	if err != nil {
		return err
	}
	instanceDir := filepath.Dir(manifestPath)
	config, netCleanup, err := buildVMConfig(instanceDir, manifest)
	if err != nil {
		return err
	}
	if netCleanup != nil {
		defer netCleanup()
	}
	if err := vzvm.Validate(config); err != nil {
		return fmt.Errorf("VM configuration validation failed: %w", err)
	}

	queue := vzvm.NewQueue("protos.hostagent." + manifest.ID)
	vm := vzvm.Create(config, queue)
	if vm.ID == 0 {
		return fmt.Errorf("failed to create VM")
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(signalCh)

	if err := startVM(queue, vm, 2*time.Minute); err != nil {
		return fmt.Errorf("failed to start VM: %w", err)
	}

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	var forceStopAt time.Time
	for {
		select {
		case <-signalCh:
			if forceStopAt.IsZero() {
				if err := requestStop(queue, vm); err != nil {
					if stopErr := stopVM(queue, vm, 30*time.Second); stopErr != nil {
						return fmt.Errorf("failed to stop VM after stop request failed: %w", stopErr)
					}
					return fmt.Errorf("failed to request VM stop: %w", err)
				}
				forceStopAt = time.Now().Add(2 * time.Minute)
			} else {
				if err := stopVM(queue, vm, 30*time.Second); err != nil {
					return fmt.Errorf("failed to force stop VM: %w", err)
				}
				return nil
			}
		case <-ticker.C:
			vzvm.RunLoopOnce()
			state := vzvm.State(queue, vm)
			switch state {
			case vz.VZVirtualMachineStateStopped:
				return nil
			case vz.VZVirtualMachineStateError:
				return fmt.Errorf("VM entered error state")
			}
			if !forceStopAt.IsZero() && time.Now().After(forceStopAt) {
				if err := stopVM(queue, vm, 30*time.Second); err != nil {
					return fmt.Errorf("failed to force stop VM after graceful shutdown timeout: %w", err)
				}
				return nil
			}
		}
	}
}

func readManifest(path string) (instanceManifest, error) {
	var manifest instanceManifest
	data, err := os.ReadFile(path)
	if err != nil {
		return manifest, fmt.Errorf("failed to read VM manifest: %w", err)
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return manifest, fmt.Errorf("failed to decode VM manifest: %w", err)
	}
	return manifest, nil
}

// buildVMConfig builds the VM configuration and returns a network cleanup func
// (nil unless the shared-vmnet datapath is active) that the caller must invoke
// when the VM exits.
func buildVMConfig(instanceDir string, manifest instanceManifest) (vz.VZVirtualMachineConfiguration, func(), error) {
	config := vz.NewVZVirtualMachineConfiguration()
	if config.ID == 0 {
		return config, nil, fmt.Errorf("failed to create VM configuration")
	}

	platformConfig := vz.NewVZGenericPlatformConfiguration()
	if platformConfig.ID == 0 {
		return config, nil, fmt.Errorf("failed to create generic platform configuration")
	}
	machineID, err := loadOrCreateMachineID(instanceDir)
	if err != nil {
		return config, nil, err
	}
	platformConfig.SetMachineIdentifier(&machineID)
	config.SetPlatform(&platformConfig.VZPlatformConfiguration)

	if strings.TrimSpace(manifest.BootISOPath) != "" {
		if err := configureEFIBootLoader(config, instanceDir); err != nil {
			return config, nil, err
		}
	} else {
		if err := configureLinuxBootLoader(config, instanceDir, manifest); err != nil {
			return config, nil, err
		}
	}
	config.SetCPUCount(uint(manifest.Cores))
	config.SetMemorySize(uint64(manifest.MemoryMiB) * 1048576)

	if err := configureConsole(config, instanceDir); err != nil {
		return config, nil, err
	}
	netCleanup, err := configureSharedVMNetNetwork(config)
	if err != nil {
		return config, nil, err
	}
	if err := configureStorage(config, manifest); err != nil {
		if netCleanup != nil {
			netCleanup()
		}
		return config, nil, err
	}
	if err := configureDevices(config); err != nil {
		if netCleanup != nil {
			netCleanup()
		}
		return config, nil, err
	}
	return config, netCleanup, nil
}

func configureLinuxBootLoader(config vz.VZVirtualMachineConfiguration, instanceDir string, manifest instanceManifest) error {
	kernelPath, err := usableKernel(instanceDir, manifest.KernelPath)
	if err != nil {
		return err
	}
	cmdline, err := commandLine(manifest.CmdlinePath)
	if err != nil {
		return err
	}

	kernelURL := foundation.NewURLFileURLWithPath(kernelPath)
	bootLoader := vz.NewLinuxBootLoaderWithKernelURL(kernelURL)
	if bootLoader.ID == 0 {
		return fmt.Errorf("failed to create Linux bootloader")
	}
	if strings.TrimSpace(manifest.InitrdPath) != "" {
		initrdURL := foundation.NewURLFileURLWithPath(manifest.InitrdPath)
		bootLoader.SetInitialRamdiskURL(initrdURL)
	}
	bootLoader.SetCommandLine(cmdline)
	config.SetBootLoader(&bootLoader.VZBootLoader)
	return nil
}

func configureEFIBootLoader(config vz.VZVirtualMachineConfiguration, instanceDir string) error {
	bootLoader := vz.NewVZEFIBootLoader()
	if bootLoader.ID == 0 {
		return fmt.Errorf("failed to create EFI bootloader")
	}

	efiStorePath := filepath.Join(instanceDir, "efi.nvram")
	efiStoreURL := foundation.NewURLFileURLWithPath(efiStorePath)
	var efiStore vz.VZEFIVariableStore
	if _, err := os.Stat(efiStorePath); os.IsNotExist(err) {
		var createErr error
		efiStore, createErr = vz.NewEFIVariableStoreCreatingVariableStoreAtURLOptionsError(
			efiStoreURL,
			vz.VZEFIVariableStoreInitializationOptionAllowOverwrite,
		)
		if createErr != nil {
			return fmt.Errorf("failed to create EFI variable store: %w", createErr)
		}
	} else {
		efiStore = vz.NewEFIVariableStoreWithURL(efiStoreURL)
	}
	if efiStore.ID == 0 {
		return fmt.Errorf("failed to open EFI variable store")
	}
	efiStore.Retain()
	bootLoader.SetVariableStore(efiStore)
	config.SetBootLoader(&bootLoader.VZBootLoader)
	return nil
}

func configureConsole(config vz.VZVirtualMachineConfiguration, instanceDir string) error {
	if os.Getenv("PROTOS_HOSTAGENT_CONSOLE_STDIO") == "1" || os.Getenv("PROTOS_VM_RUNNER_CONSOLE_STDIO") == "1" {
		toGuestRead, toGuestWrite, err := os.Pipe()
		if err != nil {
			return fmt.Errorf("failed to create console input pipe: %w", err)
		}
		fromGuestRead, fromGuestWrite, err := os.Pipe()
		if err != nil {
			return fmt.Errorf("failed to create console output pipe: %w", err)
		}
		debugConsoleFiles = append(debugConsoleFiles, toGuestRead, toGuestWrite, fromGuestRead, fromGuestWrite)

		go func() { _, _ = io.Copy(toGuestWrite, os.Stdin) }()
		go func() { _, _ = io.Copy(os.Stdout, fromGuestRead) }()

		readHandle := foundation.NewFileHandleWithFileDescriptorCloseOnDealloc(int(toGuestRead.Fd()), false)
		writeHandle := foundation.NewFileHandleWithFileDescriptorCloseOnDealloc(int(fromGuestWrite.Fd()), false)
		serialPortAttachment := vz.NewFileHandleSerialPortAttachmentWithFileHandleForReadingFileHandleForWriting(readHandle, writeHandle)
		if serialPortAttachment.ID == 0 {
			return fmt.Errorf("failed to create file handle serial port attachment")
		}
		serialPortAttachment.Retain()

		consoleConfig := vz.NewVZVirtioConsoleDeviceSerialPortConfiguration()
		if consoleConfig.ID == 0 {
			return fmt.Errorf("failed to create console configuration")
		}
		consoleConfig.SetAttachment(serialPortAttachment)
		config.SetSerialPorts([]vz.VZSerialPortConfiguration{consoleConfig.VZSerialPortConfiguration})
		return nil
	}

	consolePath := filepath.Join(instanceDir, consoleLog)
	output, err := os.OpenFile(consolePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	if err := output.Close(); err != nil {
		return err
	}

	consoleURL := foundation.NewURLFileURLWithPath(consolePath)
	serialPortAttachment, err := vz.NewFileSerialPortAttachmentWithURLAppendError(consoleURL, true)
	if err != nil {
		return fmt.Errorf("failed to create serial port attachment: %w", err)
	}
	if serialPortAttachment.ID == 0 {
		return fmt.Errorf("failed to create serial port attachment")
	}
	serialPortAttachment.Retain()

	consoleConfig := vz.NewVZVirtioConsoleDeviceSerialPortConfiguration()
	if consoleConfig.ID == 0 {
		return fmt.Errorf("failed to create console configuration")
	}
	consoleConfig.SetAttachment(serialPortAttachment)
	config.SetSerialPorts([]vz.VZSerialPortConfiguration{consoleConfig.VZSerialPortConfiguration})
	return nil
}

func configureStorage(config vz.VZVirtualMachineConfiguration, manifest instanceManifest) error {
	storageDevices := []vz.VZStorageDeviceConfiguration{}
	addDisk := func(path string, readOnly bool) error {
		if strings.TrimSpace(path) == "" {
			return nil
		}
		diskURL := foundation.NewURLFileURLWithPath(path)
		attachment, err := vz.NewDiskImageStorageDeviceAttachmentWithURLReadOnlyError(diskURL, readOnly)
		if err != nil {
			return fmt.Errorf("failed to attach disk '%s': %w", path, err)
		}
		if attachment.ID == 0 {
			return fmt.Errorf("failed to attach disk '%s'", path)
		}
		attachment.Retain()
		device := vz.NewVirtioBlockDeviceConfigurationWithAttachment(attachment)
		if device.ID == 0 {
			return fmt.Errorf("failed to create block device for '%s'", path)
		}
		device.Retain()
		storageDevices = append(storageDevices, device.VZStorageDeviceConfiguration)
		return nil
	}
	if err := addDisk(manifest.RootDiskPath, false); err != nil {
		return err
	}
	if err := addDisk(manifest.BootISOPath, true); err != nil {
		return err
	}
	for _, volume := range manifest.Volumes {
		if err := addDisk(volume.Path, false); err != nil {
			return err
		}
	}
	if err := addDisk(manifest.MetadataISO, true); err != nil {
		return err
	}
	config.SetStorageDevices(storageDevices)
	return nil
}

func configureDevices(config vz.VZVirtualMachineConfiguration) error {
	entropyConfig := vz.NewVZVirtioEntropyDeviceConfiguration()
	if entropyConfig.ID == 0 {
		return fmt.Errorf("failed to create entropy device")
	}
	config.SetEntropyDevices([]vz.VZEntropyDeviceConfiguration{entropyConfig.VZEntropyDeviceConfiguration})

	memoryBalloonDeviceConfiguration := vz.NewVZVirtioTraditionalMemoryBalloonDeviceConfiguration()
	if memoryBalloonDeviceConfiguration.ID == 0 {
		return fmt.Errorf("failed to create memory balloon device")
	}
	config.SetMemoryBalloonDevices([]vz.VZMemoryBalloonDeviceConfiguration{memoryBalloonDeviceConfiguration.VZMemoryBalloonDeviceConfiguration})

	socketDeviceConfiguration := vz.NewVZVirtioSocketDeviceConfiguration()
	if socketDeviceConfiguration.ID == 0 {
		return fmt.Errorf("failed to create socket device")
	}
	config.SetSocketDevices([]vz.VZSocketDeviceConfiguration{socketDeviceConfiguration.VZSocketDeviceConfiguration})
	return nil
}

func loadOrCreateMachineID(instanceDir string) (vz.VZGenericMachineIdentifier, error) {
	machineIDPath := filepath.Join(instanceDir, "linux-machine.id")
	if data, err := os.ReadFile(machineIDPath); err == nil && len(data) > 0 {
		nsData := foundation.NewDataWithBytesLength(data)
		if nsData.ID != 0 {
			machineID := vz.NewGenericMachineIdentifierWithDataRepresentation(&nsData)
			if machineID.ID != 0 {
				return machineID, nil
			}
		}
	}

	machineID := vz.NewVZGenericMachineIdentifier()
	if machineID.ID == 0 {
		return machineID, fmt.Errorf("failed to create VM machine identifier")
	}

	dataRepresentation := machineID.DataRepresentation()
	if dataRepresentation.GetID() == 0 {
		return machineID, nil
	}
	data := foundation.NSDataFromID(dataRepresentation.GetID()).GoBytes()
	if len(data) == 0 {
		return machineID, nil
	}
	if err := os.WriteFile(machineIDPath, data, 0644); err != nil {
		return machineID, fmt.Errorf("failed to persist VM machine identifier: %w", err)
	}
	return machineID, nil
}

func startVM(queue *vzvm.Queue, vm vz.VZVirtualMachine, timeout time.Duration) error {
	return runVMOperation(timeout, func(completion func(error)) {
		vzvm.Start(queue, vm, completion)
	})
}

func stopVM(queue *vzvm.Queue, vm vz.VZVirtualMachine, timeout time.Duration) error {
	if !vzvm.CanStop(queue, vm) {
		return nil
	}
	return runVMOperation(timeout, func(completion func(error)) {
		vzvm.Stop(queue, vm, completion)
	})
}

func requestStop(queue *vzvm.Queue, vm vz.VZVirtualMachine) error {
	var (
		ok  bool
		err error
	)
	queue.Sync(func() {
		if !vm.CanRequestStop() {
			err = fmt.Errorf("guest stop request is not available in state %s", vm.State())
			return
		}
		ok, err = vm.RequestStopWithError()
	})
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("guest stop request was rejected")
	}
	return nil
}

func runVMOperation(timeout time.Duration, operation func(func(error))) error {
	done := make(chan error, 1)
	operation(func(err error) {
		done <- err
	})

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case err := <-done:
			return err
		case <-timer.C:
			return fmt.Errorf("operation timed out after %s", timeout)
		default:
			vzvm.RunLoopOnce()
		}
	}
}

func commandLine(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read kernel cmdline: %w", err)
	}
	args := strings.Fields(string(data))
	for _, arg := range args {
		if arg == "console=hvc0" {
			return strings.Join(args, " "), nil
		}
	}
	args = append(args, "console=hvc0")
	return strings.Join(args, " "), nil
}

func usableKernel(instanceDir string, kernelPath string) (string, error) {
	file, err := os.Open(kernelPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	var header [2]byte
	n, err := file.Read(header[:])
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	if _, err := file.Seek(0, 0); err != nil {
		return "", err
	}
	if n != 2 || header[0] != 0x1f || header[1] != 0x8b {
		return kernelPath, nil
	}
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzipReader.Close()
	uncompressedPath := filepath.Join(instanceDir, "kernel-uncompressed")
	output, err := os.OpenFile(uncompressedPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(output, gzipReader); err != nil {
		_ = output.Close()
		return "", err
	}
	if err := output.Close(); err != nil {
		return "", err
	}
	return uncompressedPath, nil
}
