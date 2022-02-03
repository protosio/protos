package protosc

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

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
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/release"
	"github.com/protosio/protos/internal/resource"
	"github.com/protosio/protos/internal/runtime"
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
	stoppers          map[string]func() error
	db                db.DB
	cfg               *config.Config
	version           string
	wg                sync.WaitGroup
	capabilityManager *capability.Manager

	UserManager    *auth.UserManager
	KeyManager     *pcrypto.Manager
	AppManager     *app.Manager
	NetworkManager *network.Manager
	AppStore       *installer.AppStore
	CloudManager   *cloud.Manager
	P2PManager     *p2p.P2P
	Meta           *meta.Meta
}

func New(dataPath string, version string) (*ProtosClient, error) {

	protosClient := &ProtosClient{
		stoppers: map[string]func() error{},
		version:  version,
		wg:       sync.WaitGroup{},
		cfg:      config.Get(),
	}

	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve home directory: %w", err)
	}

	if dataPath == "~" {
		dataPath = homedir
	} else if strings.HasPrefix(dataPath, "~/") {
		dataPath = filepath.Join(homedir, dataPath[2:])
	}
	protosClient.cfg.WorkDir = dataPath

	// create protos dir
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		err := os.Mkdir(dataPath, 0755)
		if err != nil {
			return nil, fmt.Errorf("failed to create Protos directory '%s': %w", dataPath, err)
		}
	}

	// open db
	protosDB := "protos.db"
	protosClient.db, err = db.Open(dataPath, protosDB)
	if err != nil {
		return nil, fmt.Errorf("failed to open db during configuration: %w", err)
	}

	// create various managers
	keyManager := pcrypto.CreateManager(protosClient.db)
	metaClient := meta.Setup(protosClient.db, keyManager, version)
	capabilityManager := capability.CreateManager()
	userManager := auth.CreateUserManager(protosClient.db, keyManager, capabilityManager, protosClient)

	protosClient.UserManager = userManager
	protosClient.KeyManager = keyManager
	protosClient.capabilityManager = capabilityManager
	protosClient.Meta = metaClient

	return protosClient, nil

}

func networkUp(userManager *auth.UserManager, internalDomain string) (*network.Manager, error) {
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

	err = networkManager.Init(*netp, internalIP, key.PrivateWG(), internalDomain)
	if err != nil {
		return nil, fmt.Errorf("failed to configure network: %w", err)
	}

	return networkManager, nil
}

//
// public methods
//

func (pc *ProtosClient) FinishInit() error {

	// create publisher
	pub := &publisher{pubchan: make(chan interface{}, 100)}

	resourceManager := resource.CreateManager(pc.db)

	taskManager := task.CreateManager(pc.db, pub)
	networkManager, err := networkUp(pc.UserManager, pc.cfg.InternalDomain)
	if err != nil {
		log.Fatalf("Failed to create network manager: %s", err.Error())
	}

	appRuntime := runtime.Create(networkManager, pc.cfg.RuntimeEndpoint, pc.cfg.InContainer, "")
	appStore := installer.CreateAppStore(appRuntime, taskManager, pc.capabilityManager)
	appManager := app.CreateManager(app.TypeProtosc, resourceManager, taskManager, appRuntime, pc.db, pc.Meta, pub, appStore, pc.capabilityManager)

	// get device key
	key, err := pc.Meta.GetPrivateKey()
	if err != nil {
		log.Fatalf("Failed to retrieve key during configuration: %s", err.Error())
	}

	p2pManager, err := p2p.NewManager(key, pc.db, false)
	if err != nil {
		log.Fatalf("Failed to create p2p manager: %s", err.Error())
	}
	pc.P2PManager = p2pManager
	pc.db.AddPublisher(p2pManager)

	p2pStopper, err := p2pManager.StartServer(pc.Meta, pc.db.GetChunkStore())
	if err != nil {
		log.Fatalf("Failed to start p2p server: %s", err.Error())
	}
	pc.stoppers["p2p"] = p2pStopper

	admin, err := pc.UserManager.GetAdmin()
	if err != nil {
		log.Fatalf("Failed to retrieve admin user: %s", err.Error())
	}

	currentDevice, err := admin.GetCurrentDevice()
	if err != nil {
		log.Fatalf("Failed to get current device: %s", err.Error())
	}

	cloudManager, err := cloud.CreateManager(pc.db, pc.UserManager, pc.KeyManager, p2pManager, pc, currentDevice.Name)
	if err != nil {
		log.Fatalf("Failed to create cloud manager: %s", err.Error())
	}

	dnsStopper := dns.StartServer(localDNSAddress, localDNSPort, "", pc.cfg.InternalDomain, appManager)
	pc.stoppers["dns"] = dnsStopper
	pc.AppManager = appManager
	pc.AppStore = appStore
	pc.CloudManager = cloudManager
	pc.NetworkManager = networkManager

	pc.Refresh()

	return nil

}

func (pc *ProtosClient) Refresh() error {

	if pc.CloudManager == nil || pc.UserManager == nil || pc.P2PManager == nil {
		log.Debug("Protos client not ready yet. Skipping refresh")
		return nil
	}

	instances, err := pc.CloudManager.GetInstances()
	if err != nil {
		return fmt.Errorf("failed to retrieve instances: %w", err)
	}

	peers := []p2p.Machine{}
	for _, instance := range instances {
		peers = append(peers, instance)
	}

	admin, err := pc.UserManager.GetAdmin()
	if err == nil {
		userDevices := admin.GetDevices()
		err = pc.NetworkManager.ConfigurePeers(instances, userDevices)
		if err != nil {
			return fmt.Errorf("failed to configure network peers: %w", err)
		}
		for _, device := range userDevices {
			peers = append(peers, &device)
		}
	}
	if err != nil {
		if !strings.Contains(err.Error(), "could not find admin user") {
			return fmt.Errorf("failed to retrieve admin user: %w", err)
		}
	}

	err = pc.P2PManager.ConfigurePeers(peers)
	if err != nil {
		return fmt.Errorf("failed to configure network peers: %w", err)
	}

	return nil
}

func (pc *ProtosClient) IsInitialized() bool {
	_, err := pc.UserManager.GetAdmin()
	if err != nil {
		pc.wg.Add(1)
		return false
	}
	return true
}

func (pc *ProtosClient) SetInitialized() {
	pc.wg.Done()
}

func (pc *ProtosClient) WaitForInitialization() {
	log.Info("Waiting for initialization")
	pc.wg.Wait()
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
