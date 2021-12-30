package protosc

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/app"
	"github.com/protosio/protos/internal/auth"
	"github.com/protosio/protos/internal/capability"
	"github.com/protosio/protos/internal/cloud"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/dns"
	"github.com/protosio/protos/internal/installer"
	"github.com/protosio/protos/internal/meta"
	"github.com/protosio/protos/internal/network"
	"github.com/protosio/protos/internal/p2p"
	"github.com/protosio/protos/internal/platform"
	"github.com/protosio/protos/internal/release"
	"github.com/protosio/protos/internal/resource"
	"github.com/protosio/protos/internal/ssh"
	"github.com/protosio/protos/internal/task"
	"github.com/protosio/protos/internal/util"
)

var log = util.GetLogger("protosc")

const (
	releasesURL     = "https://releases.protos.io/releases.json"
	localDNSPort    = 10053
	localDNSAddress = "127.0.0.1"
)

type publisher struct {
	pubchan chan interface{}
}

// GetWSPublishChannel returns the channel that can be used to publish messages to the available websockets
func (pub *publisher) GetWSPublishChannel() chan interface{} {
	return pub.pubchan
}

type ProtosClient struct {
	stoppers map[string]func() error

	UserManager    *auth.UserManager
	KeyManager     *ssh.Manager
	AppManager     *app.Manager
	NetworkManager *network.Manager
	AppStore       *installer.AppStore
	CloudManager   *cloud.Manager
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
		log.Fatalf("Failed to open db during configuration: %s", err.Error())
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

	networkManager, err := networkUp(userManager)
	if err != nil {
		log.Fatalf("Failed to create network manager: %s", err.Error())
	}

	runtimePlatform := platform.Create(networkManager, cfg.RuntimeEndpoint, cfg.AppStoreHost, cfg.InContainer, "")
	metaClient := meta.SetupForClient(resourceManager, dbi, keyManager, version)
	appStore := installer.CreateAppStore(runtimePlatform, taskManager, capabilityManager)
	appManager := app.CreateManager(resourceManager, taskManager, runtimePlatform, dbi, metaClient, pub, appStore, capabilityManager)

	// get device key
	key, err := metaClient.GetPrivateKey()
	if err != nil {
		log.Fatalf("Failed to retrieve key during configuration: %s", err.Error())
	}

	p2pManager, err := p2p.NewManager(10500, key)
	if err != nil {
		log.Fatalf("Failed to create p2p manager: %s", err.Error())
	}
	cloudManager, err := cloud.CreateManager(dbi, userManager, keyManager, p2pManager)
	if err != nil {
		log.Fatalf("Failed to create cloud manager: %s", err.Error())
	}

	instances, err := cloudManager.GetInstances()
	if err != nil {
		log.Fatalf("Failed to retrieve instances: %s", err.Error())
	}

	admin, err := userManager.GetAdmin()
	if err != nil {
		log.Fatalf("Failed to retrieve admin user: %s", err.Error())
	}

	err = networkManager.ConfigurePeers(instances, admin.GetDevices())
	if err != nil {
		log.Fatalf("Failed to configure network peers: %s", err.Error())
	}

	dnsStopper := dns.StartServer(localDNSAddress, localDNSPort, "", admin.GetInfo().Domain, appManager)

	protosClient := &ProtosClient{
		stoppers: map[string]func() error{
			"dns": dnsStopper,
		},

		UserManager:    userManager,
		KeyManager:     keyManager,
		AppManager:     appManager,
		AppStore:       appStore,
		CloudManager:   cloudManager,
		NetworkManager: networkManager,
	}

	dbi.AddRefresher(protosClient)

	return protosClient, nil

}

func networkUp(userManager *auth.UserManager) (*network.Manager, error) {
	usr, err := userManager.GetAdmin()
	if err != nil {
		return nil, fmt.Errorf("failed to get admin while setting up network: %w", err)
	}

	dev, err := usr.GetCurrentDevice()
	if err != nil {
		return nil, fmt.Errorf("failed to get current device while setting up network: %w", err)
	}

	ip, netp, err := net.ParseCIDR(dev.Network)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CIDR while setting up network: %w", err)
	}
	netp.IP = ip
	internalIP := netp.IP.Mask(netp.Mask)
	internalIP[3]++

	key, err := usr.GetKeyCurrentDevice()
	if err != nil {
		return nil, fmt.Errorf("failed to get device key while setting up network: %w", err)
	}

	networkManager, err := network.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to configure network: %w", err)
	}

	err = networkManager.Init(*netp, internalIP, key.PrivateWG(), usr.GetInfo().Domain)
	if err != nil {
		return nil, fmt.Errorf("failed to configure network: %w", err)
	}

	return networkManager, nil
}

//
// public methods
//

func (pc *ProtosClient) Refresh() error {

	instances, err := pc.CloudManager.GetInstances()
	if err != nil {
		return fmt.Errorf("failed to retrieve instances: %w", err)
	}

	admin, err := pc.UserManager.GetAdmin()
	if err != nil {
		return fmt.Errorf("failed to retrieve admin user: %w", err)
	}

	err = pc.NetworkManager.ConfigurePeers(instances, admin.GetDevices())
	if err != nil {
		return fmt.Errorf("failed to configure network peers: %w", err)
	}

	return nil
}

func (pc *ProtosClient) GetProtosAvailableReleases() (release.Releases, error) {
	var releases release.Releases
	resp, err := http.Get(releasesURL)
	if err != nil {
		return releases, errors.Wrapf(err, "Failed to retrieve releases from '%s'", releasesURL)
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&releases)
	if err != nil {
		return releases, errors.Wrap(err, "Failed to JSON decode the releases response")
	}

	if len(releases.Releases) == 0 {
		return releases, errors.Errorf("Something went wrong. Parsed 0 releases from '%s'", releasesURL)
	}

	return releases, nil
}

func (pc *ProtosClient) Stop() error {
	for _, stopper := range pc.stoppers {
		err := stopper()
		if err != nil {
			log.Error(err)
		}
	}
	return nil
}
