package platform

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/core"
	"github.com/protosio/protos/internal/util"
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
)

type internalIPSetter interface {
	SetInternalIP(ip string)
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

// Initialize checks if the Protos network exists
func Initialize(runtime string, runtimeUnixSocket string, appStoreHost string, inContainer bool, ipSetter internalIPSetter) core.RuntimePlatform {
	var dp core.RuntimePlatform
	switch runtime {
	case dockerRuntime:
		dp = createDockerRuntimePlatform(runtimeUnixSocket, appStoreHost, inContainer)
	case containerdRuntime:
		dp = createContainerdRuntimePlatform(runtimeUnixSocket, appStoreHost, inContainer, "192.168.40.29")
	}
	internalIP, err := dp.Init()
	if err != nil {
		log.Fatalf("Can't connect to runtime: %s", err)
	}
	ipSetter.SetInternalIP(internalIP)

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
