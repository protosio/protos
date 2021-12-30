package platform

import (
	"github.com/protosio/protos/internal/network"
	"github.com/protosio/protos/internal/util"
)

const (
	protosNamespace string = "protos"
)

type containerdPlatform struct{}

func createContainerdRuntimePlatform(networkManager *network.Manager, runtimeUnixSocket string, appStoreHost string, inContainer bool, logsPath string) *containerdPlatform {
	return &containerdPlatform{}
}

func (cdp *containerdPlatform) Init() error {
	return nil
}

func (cdp *containerdPlatform) NewSandbox(name string, appID string, imageID string, volumeID string, volumeMountPath string, publicPorts []util.Port, installerParams map[string]string) (PlatformRuntimeUnit, error) {
	return nil, nil
}

func (cdp *containerdPlatform) GetImage(id string) (PlatformImage, error) {
	return nil, nil
}

func (cdp *containerdPlatform) GetAllImages() (map[string]PlatformImage, error) {
	images := map[string]PlatformImage{}
	return images, nil
}

func (cdp *containerdPlatform) GetSandbox(id string) (PlatformRuntimeUnit, error) {
	return nil, nil
}

func (cdp *containerdPlatform) GetAllSandboxes() (map[string]PlatformRuntimeUnit, error) {
	return map[string]PlatformRuntimeUnit{}, nil
}

func (cdp *containerdPlatform) GetHWStats() (HardwareStats, error) {
	return HardwareStats{}, nil
}

func (cdp *containerdPlatform) PullImage(id string, name string, version string) error {
	return nil
}

func (cdp *containerdPlatform) RemoveImage(id string) error {
	return nil
}

func (cdp *containerdPlatform) GetOrCreateVolume(id string, path string) (string, error) {
	return "", nil
}

func (cdp *containerdPlatform) RemoveVolume(id string) error {
	return nil
}

func (cdp *containerdPlatform) ImageExistsLocally(id string) (bool, error) {
	return false, nil
}

//
// struct and methods that satisfy PlatformRuntimeUnit
//

// containerdSandbox represents a container
type containerdSandbox struct{}

// Update reads the container and updates the struct fields
func (cnt *containerdSandbox) Update() error {
	return nil
}

// Start starts a containerd sandbox
func (cnt *containerdSandbox) Start() error {
	return nil
}

// Stop stops a containerd sandbox
func (cnt *containerdSandbox) Stop() error {
	return nil
}

// Remove removes a containerd sandbox
func (cnt *containerdSandbox) Remove() error {
	return nil
}

// GetID returns the ID of the container, as a string
func (cnt *containerdSandbox) GetID() string {
	return ""
}

// GetIP returns the IP of the container, as a string
func (cnt *containerdSandbox) GetIP() string {
	return ""
}

// GetStatus returns the status of the container, as a string
func (cnt *containerdSandbox) GetStatus() string {
	return ""
}

// GetExitCode returns the exit code of the container, as an int
func (cnt *containerdSandbox) GetExitCode() int {
	return 0
}
