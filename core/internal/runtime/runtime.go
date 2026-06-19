package runtime

import (
	"context"
	"errors"
	"net"

	"github.com/protosio/protos/internal/imageregistry"
	"github.com/protosio/protos/internal/network"
)

var ErrSandboxNotFound = errors.New("sandbox not found")

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
	NewSandbox(name string, appID string, imageID string, persistence bool) (RuntimeSandbox, error)
	GetHWStats() (HardwareStats, error)
}

type ImageLoader interface {
	LoadImageArchive(ctx context.Context, archivePath string, imageRef string) (imageregistry.LoadedImage, error)
}

// Create initializes the run time platform
func Create(networkManager *network.Manager, runtimeUnixSocket string) RuntimePlatform {
	return createContainerdRuntimePlatform(networkManager, runtimeUnixSocket)
}
