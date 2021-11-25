package protosc

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/protosio/protos/internal/app"
	"github.com/protosio/protos/internal/auth"
	"github.com/protosio/protos/internal/capability"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/installer"
	"github.com/protosio/protos/internal/meta"
	"github.com/protosio/protos/internal/platform"
	"github.com/protosio/protos/internal/resource"
	"github.com/protosio/protos/internal/ssh"
	"github.com/protosio/protos/internal/task"
	"github.com/protosio/protos/internal/util"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
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
	AppManager  *app.Manager
	AppStore    *installer.AppStore
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

	// get default cfg
	cfg := config.Get()

	// create publisher
	pub := &publisher{pubchan: make(chan interface{}, 100)}

	// create various managers
	keyManager := ssh.CreateManager(dbi)
	capabilityManager := capability.CreateManager()
	userManager := auth.CreateUserManager(dbi, keyManager, capabilityManager)
	resourceManager := resource.CreateManager(dbi)
	taskManager := task.CreateManager(dbi, pub)
	runtimePlatform := platform.Create(cfg.Runtime, cfg.RuntimeEndpoint, cfg.AppStoreHost, cfg.InContainer, wgtypes.Key{}, "")
	metaClient := meta.SetupForClient(resourceManager, dbi, keyManager, version)
	appStore := installer.CreateAppStore(runtimePlatform, taskManager, capabilityManager)
	appManager := app.CreateManager(resourceManager, taskManager, runtimePlatform, dbi, metaClient, pub, appStore, capabilityManager)

	protosClient := ProtosClient{
		UserManager: userManager,
		KeyManager:  keyManager,
		AppManager:  appManager,
		AppStore:    appStore,
	}

	return &protosClient, nil

}
