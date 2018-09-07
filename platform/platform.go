package platform

import (
	"github.com/protosio/protos/config"
	"github.com/protosio/protos/util"
)

var gconfig = config.Get()
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

// Initialize checks if the Protos network exists
func Initialize() {
	ConnectDocker()
}
