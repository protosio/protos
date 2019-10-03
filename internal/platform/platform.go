package platform

import (
	"protos/internal/config"
	"protos/internal/core"
	"protos/internal/util"

	"github.com/pkg/errors"
)

var gconfig = config.Get()
var log = util.GetLogger("platform")

type platform struct {
	ID          string
	NetworkID   string
	NetworkName string
}

// Initialize checks if the Protos network exists
func Initialize(inContainer bool) core.RuntimePlatform {
	dp := &dockerPlatform{}
	dp.Connect()
	if inContainer {
		// if running in container the user needs to take care that the correct protos network is created
		return nil
	}
	protosNet, err := dp.GetNetwork(protosNetwork)
	if err != nil {
		if util.IsErrorType(err, core.ErrNetworkNotFound) {
			// if network is not found it should be created
			protosNet, err = dp.CreateNetwork(protosNetwork)
			if err != nil {
				log.Fatal(errors.Wrap(err, "Failed to initialize Docker platform"))
			}
		} else {
			log.Fatal(errors.Wrap(err, "Failed to initialize Docker platform"))
		}
	}
	if len(protosNet.IPAM.Config) == 0 {
		log.Fatalf("Failed to initialize Docker platform: no network config for network %s(%s)", protosNet.Name, protosNet.ID)
	}
	netConfig := protosNet.IPAM.Config[0]
	log.Debugf("Running using internal Docker network %s(%s), gateway %s in subnet %s", protosNet.Name, protosNet.ID, netConfig.Gateway, netConfig.Subnet)
	gconfig.InternalIP = netConfig.Gateway

	return dp
}
