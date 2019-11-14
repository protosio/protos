package platform

import (
	"github.com/protosio/protos/internal/core"
	"github.com/protosio/protos/internal/util"
)

var log = util.GetLogger("platform")

const (
	dockerRuntime     = "docker"
	containerdRuntime = "containerd"
)

type internalIPSetter interface {
	SetInternalIP(ip string)
}

// Initialize checks if the Protos network exists
func Initialize(runtime string, runtimeUnixSocket string, appStoreHost string, inContainer bool, ipSetter internalIPSetter) core.RuntimePlatform {
	var dp core.RuntimePlatform
	switch runtime {
	case dockerRuntime:
		dp = createDockerRuntimePlatform(runtimeUnixSocket, appStoreHost, inContainer)
	case containerdRuntime:
		dp = createContainerdRuntimePlatform(runtimeUnixSocket, appStoreHost, inContainer)
	}
	internalIP, err := dp.Init()
	if err != nil {
		log.Fatalf("Can't connect to runtime: %s", err)
	}
	ipSetter.SetInternalIP(internalIP)

	return dp
}
