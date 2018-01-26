package daemon

import (
	"os"
	"protos/config"
	"protos/platform"
	"protos/util"
)

var gconfig = config.Gconfig
var log = util.Log

// StartUp triggers a sequence of steps required to start the application
func StartUp() {
	log.Info("Starting up...")
	var err error

	// Generate secret key used for JWT
	log.Info("Generating secret for JWT")
	gconfig.Secret, err = util.GenerateRandomBytes(32)
	if err != nil {
		log.Fatal(err)
	}

	platform.ConnectDocker()

}

// Setup creates the Protos work directory
func Setup() {

	// create the workdir if it does not exist
	if _, err := os.Stat(gconfig.WorkDir); err != nil {
		if os.IsNotExist(err) {
			log.Info("Creating working directory [", gconfig.WorkDir, "]")
			err = os.Mkdir(gconfig.WorkDir, 0755)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal(err)
		}
	}

}
