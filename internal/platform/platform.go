package platform

import (
	"net"
	"strings"

	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/auth"
	"github.com/protosio/protos/internal/util"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var log = util.GetLogger("platform")

const (
	// app states
	statusRunning  = "running"
	statusStopped  = "stopped"
	statusCreating = "creating"
	statusFailed   = "failed"
	statusUnknown  = "unknown"

	dockerRuntime     = "docker"
	containerdRuntime = "containerd"

	// ErrImageNotFound means the requested docker image is not found locally
	ErrImageNotFound = 101
	// ErrNetworkNotFound means the requested docker network is not found locally
	ErrNetworkNotFound = 102
	// ErrContainerNotFound means the requested docker container is not found locally
	ErrContainerNotFound = 103
)

// PlatformRuntimeUnit represents the abstract concept of a running program: it can be a container, VM or process.
type PlatformRuntimeUnit interface {
	Start() error
	Stop() error
	Update() error
	Remove() error
	GetID() string
	GetIP() string
	GetStatus() string
	GetExitCode() int
}

type PlatformImage interface {
	GetID() string
	GetDataPath() string
	GetRepoTags() []string
	GetLabels() map[string]string
}

// RuntimePlatform represents the platform that manages the PlatformRuntimeUnits. For now Docker.
type RuntimePlatform interface {
	Init(network net.IPNet, devices []auth.UserDevice) error
	GetSandbox(id string) (PlatformRuntimeUnit, error)
	GetAllSandboxes() (map[string]PlatformRuntimeUnit, error)
	GetImage(id string) (PlatformImage, error)
	GetAllImages() (map[string]PlatformImage, error)
	PullImage(id string, name string, version string) error
	RemoveImage(id string) error
	GetOrCreateVolume(id string, path string) (string, error)
	RemoveVolume(id string) error
	NewSandbox(name string, appID string, imageID string, volumeID string, volumeMountPath string, publicPorts []util.Port, installerParams map[string]string) (PlatformRuntimeUnit, error)
	GetHWStats() (HardwareStats, error)
}

func normalizeRepoDigest(repoDigests []string) (string, string, error) {
	if len(repoDigests) == 0 {
		return "<none>", "<none>", errors.New("image has no repo digests")
	}
	repoDigestPair := strings.Split(repoDigests[0], "@")
	if len(repoDigestPair) != 2 {
		return "errorName", "errorRepoDigest", errors.Errorf("image repo digest has an invalid format: '%s'", repoDigests[0])
	}
	return repoDigestPair[0], repoDigestPair[1], nil
}

// Create initializes the run time platform
func Create(runtime string, runtimeUnixSocket string, appStoreHost string, inContainer bool, key wgtypes.Key, logsPath string) RuntimePlatform {

	var dp RuntimePlatform
	switch runtime {
	case containerdRuntime:
		dp = createContainerdRuntimePlatform(runtimeUnixSocket, appStoreHost, inContainer, key, logsPath)
	default:
		log.Fatalf("Runtime '%s' is not supported", runtime)
	}

	return dp
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
