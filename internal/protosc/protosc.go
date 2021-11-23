package protosc

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/protosio/protos/internal/auth"
	"github.com/protosio/protos/internal/capability"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/ssh"
	"github.com/protosio/protos/internal/util"
)

var log = util.GetLogger("protosc")

type publisher struct {
	pubchan chan interface{}
}

// GetWSPublishChannel returns the channel that can be used to publish messages to the available websockets
func (pub *publisher) GetWSPublishChannel() chan interface{} {
	return pub.pubchan
}

type ProtosClient struct {
	UserManager *auth.UserManager
	KeyManager  *ssh.Manager
}

func New(dataPath string, version string) (*ProtosClient, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	if dataPath == "~" {
		dataPath = homedir
	} else if strings.HasPrefix(dataPath, "~/") {
		dataPath = filepath.Join(homedir, dataPath[2:])
	}

	// create protos dir
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		err := os.Mkdir(dataPath, 0755)
		if err != nil {
			log.Fatalf("Failed to create protos dir '%s': %s", dataPath, err.Error())
		}
	}

	// open db
	protosDB := "protos.db"
	dbi, err := db.Open(dataPath, protosDB)
	if err != nil {
		log.Fatalf("Failed to open db during configuration: %v", err)
	}

	// create various managers
	keyManager := ssh.CreateManager(dbi)
	capabilityManager := capability.CreateManager()
	userManager := auth.CreateUserManager(dbi, keyManager, capabilityManager)

	protosClient := ProtosClient{
		UserManager: userManager,
		KeyManager:  keyManager,
	}

	return &protosClient, nil

}
