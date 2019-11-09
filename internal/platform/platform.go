package platform

import (
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/core"
	"github.com/protosio/protos/internal/util"
)

var gconfig = config.Get()
var log = util.GetLogger("platform")

const (
	dockerRuntime     = "docker"
	containerdRuntime = "containerd"
)

type platform struct {
	ID          string
	NetworkID   string
	NetworkName string
}

// Initialize checks if the Protos network exists
func Initialize(runtime string, inContainer bool) core.RuntimePlatform {
	var dp core.RuntimePlatform
	switch runtime {
	case dockerRuntime:
		dp = createDockerRuntimePlatform()
	case containerdRuntime:
		dp = createContainerdRuntimePlatform()
	}
	ip, err := dp.Init(inContainer)
	if err != nil {
		log.Panic(err)
	}
	gconfig.InternalIP = ip

	return dp
}
