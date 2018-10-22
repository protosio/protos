package platform

import (
	"github.com/pkg/errors"
	"github.com/protosio/protos/config"
	"github.com/protosio/protos/util"
)

var gconfig = config.Get()
var log = util.GetLogger("platform")

type platform struct {
	ID          string
	NetworkID   string
	NetworkName string
}

// RuntimeUnit represents the abstract concept of a running program: it can be a container, VM or process.
type RuntimeUnit interface {
	Start() error
	Stop() error
	Update() error
	Remove() error
	GetID() string
	GetIP() string
	GetStatus() string
}

// Initialize checks if the Protos network exists
func Initialize(inContainer bool) {
	ConnectDocker()
	if inContainer {
		// if running in container the user needs to take care that the correct protos network is created
		return
	}
	protosNet, err := GetDockerNetwork(protosNetwork)
	if err != nil {
		if util.IsErrorType(err, ErrDockerNetworkNotFound) {
			// if network is not found it should be created
			protosNet, err = CreateDockerNetwork(protosNetwork)
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
	protosIP = netConfig.Gateway
}
