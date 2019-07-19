package core

import (
	"github.com/docker/docker/api/types"
	"github.com/protosio/protos/util"
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
	GetDockerContainer(string) (PlatformRuntimeUnit, error)
	GetAllDockerContainers() (map[string]PlatformRuntimeUnit, error)
	GetDockerImage(string) (types.ImageInspect, error)
	GetAllDockerImages() (map[string]types.ImageSummary, error)
	GetDockerImageDataPath(types.ImageInspect) (string, error)
	PullDockerImage(Task, string, string, string) error
	RemoveDockerImage(id string) error
	GetOrCreateVolume(string, string) (string, error)
	RemoveVolume(string) error
	NewContainer(string, string, string, string, string, []util.Port, map[string]string) (PlatformRuntimeUnit, error)
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
