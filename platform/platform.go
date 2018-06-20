package platform

import (
	"github.com/nustiueudinastea/protos/database"
	"github.com/nustiueudinastea/protos/util"
)

var log = util.Log

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

// Setup creates the Protos network through which applications communicate
func Setup() {
	ConnectDocker()
	log.Debug("Creating protosnet network")
	networkID, err := CreateDockerNetwork("protosnet")
	if err != nil {
		log.Fatal(err)
	}

	platform := platform{ID: "docker", NetworkID: networkID, NetworkName: "protosnet"}
	err = database.Save(&platform)
	if err != nil {
		log.Fatal(err)
	}
}

// Initialize checks if the Protos network exists
func Initialize() {
	log.Debug("Reading platform information from database")
	var platform platform
	err := database.One("ID", "docker", &platform)
	if err != nil {
		log.Fatalf("Can't load platform information from database(%s). Please run init", err.Error())
	}
	ConnectDocker()

	if platform.NetworkID == "" || DockerNetworkExists(platform.NetworkID) == false {
		log.Fatal("Protos network does not exist. Please run init")
	}
}
