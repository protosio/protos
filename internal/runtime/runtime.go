package runtime

import (
	"errors"
	"net"

	"github.com/protosio/protos/internal/network"
	"github.com/protosio/protos/internal/util"
)

var log = util.GetLogger("platform")
var ErrSandboxNotFound = errors.New("sandbox not found")

const (
	// ErrImageNotFound means the requested docker image is not found locally
	ErrImageNotFound = 101
	// ErrNetworkNotFound means the requested docker network is not found locally
	ErrNetworkNotFound = 102
	// ErrContainerNotFound means the requested docker container is not found locally
	ErrContainerNotFound = 103
)

// RuntimeSandbox represents the abstract concept of a running program: it can be a container, VM or process.
type RuntimeSandbox interface {
	Start(ip net.IP) error
	Stop() error
	Update() error
	Remove() error
	GetID() string
	GetStatus() string
	GetLogs() ([]byte, error)
	GetExitCode() int
}

type PlatformImage interface {
	GetID() string
	GetDataPath() string
	GetRepoTags() []string
	GetLabels() map[string]string
}

// RuntimePlatform represents the platform that manages the RuntimeSandboxs. For now Docker.
type RuntimePlatform interface {
	Init() error
	GetSandbox(id string) (RuntimeSandbox, error)
	GetAllSandboxes() (map[string]RuntimeSandbox, error)
	GetImage(id string) (PlatformImage, error)
	ImageExistsLocally(id string) (bool, error)
	GetAllImages() (map[string]PlatformImage, error)
	PullImage(imageRef string) error
	RemoveImage(id string) error
	GetOrCreateVolume(path string) (string, error)
	RemoveVolume(id string) error
	NewSandbox(name string, appID string, imageID string, volumeMountPath string, installerParams map[string]string) (RuntimeSandbox, error)
	GetHWStats() (HardwareStats, error)
}

// Create initializes the run time platform
func Create(networkManager *network.Manager, runtimeUnixSocket string, inContainer bool, logsPath string) RuntimePlatform {
	return createContainerdRuntimePlatform(networkManager, runtimeUnixSocket, inContainer, logsPath)
}

type platformImage struct {
	id              string
	localID         string
	persistencePath string
	repoTags        []string
	labels          map[string]string
}

func (pi *platformImage) GetID() string {
	return pi.id
}

func (pi *platformImage) GetDataPath() string {
	return pi.persistencePath
}

func (pi *platformImage) GetRepoTags() []string {
	return pi.repoTags
}

func (pi *platformImage) GetLabels() map[string]string {
	return pi.labels
}
