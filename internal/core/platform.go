package core

import (
	"github.com/protosio/protos/internal/util"
)

const (
	// ErrImageNotFound means the requested docker image is not found locally
	ErrImageNotFound = 101
	// ErrNetworkNotFound means the requested docker network is not found locally
	ErrNetworkNotFound = 102
	// ErrContainerNotFound means the requested docker container is not found locally
	ErrContainerNotFound = 103
)

// RuntimePlatform represents the platform that manages the PlatformRuntimeUnits. For now Docker.
type RuntimePlatform interface {
	Init() error
	GetSandbox(id string) (PlatformRuntimeUnit, error)
	GetAllSandboxes() (map[string]PlatformRuntimeUnit, error)
	GetImage(id string) (PlatformImage, error)
	GetAllImages() (map[string]PlatformImage, error)
	PullImage(task Task, id string, name string, version string) error
	RemoveImage(id string) error
	GetOrCreateVolume(id string, path string) (string, error)
	RemoveVolume(id string) error
	CleanUpSandbox(id string) error
	NewSandbox(name string, appID string, imageID string, volumeID string, volumeMountPath string, publicPorts []util.Port, installerParams map[string]string) (PlatformRuntimeUnit, error)
	GetHWStats() (HardwareStats, error)
}

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

type HardwareStats interface {
}
