package protosc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/app"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/dns"
	"github.com/protosio/protos/internal/membership"
	"github.com/protosio/protos/internal/network"
	"github.com/protosio/protos/internal/p2p"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/provisioners"
	"github.com/protosio/protos/internal/release"
	"github.com/protosio/protos/internal/runtime"
	"github.com/protosio/protos/internal/user"
	"github.com/protosio/protos/internal/util"
	"github.com/protosio/protos/provisioners/local_macos"
	"github.com/protosio/protos/provisioners/scaleway"

	"github.com/Masterminds/semver"
)

var log = util.GetLogger("protosc")

type ProtosClient struct {
	stoppers map[string]func() error
	cfg      config.Config
	version  string
	wg       sync.WaitGroup
	notifyMu sync.Mutex
	localKey *pcrypto.Key

	DB             *db.DB
	Manager        *user.Manager
	KeyManager     *pcrypto.Manager
	AppManager     *app.Manager
	NetworkManager *network.Manager
	CloudManager   *provisioners.Manager
	P2PManager     *p2p.P2P
}

func New(dataPath string, version *semver.Version) (*ProtosClient, error) {

	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve home directory: %w", err)
	}

	if dataPath == "~" {
		dataPath = homedir
	} else if strings.HasPrefix(dataPath, "~/") {
		dataPath = filepath.Join(homedir, dataPath[2:])
	}

	protosClient := &ProtosClient{
		stoppers: map[string]func() error{},
		version:  version.String(),
		wg:       sync.WaitGroup{},
		cfg:      config.New(dataPath, version),
	}

	// create protos dir
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		err := os.Mkdir(dataPath, 0755)
		if err != nil {
			return nil, fmt.Errorf("failed to create Protos directory '%s': %w", dataPath, err)
		}
	}

	// get local key
	lkey, err := pcrypto.GetLocalKey(dataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get local key: %w", err)
	}
	protosClient.localKey = lkey

	// open db
	protosClient.DB, err = db.Open(dataPath, config.DBName, lkey)
	if err != nil {
		return nil, fmt.Errorf("failed to open db during configuration: %w", err)
	}

	// create various managers
	keyManager := pcrypto.CreateManager(protosClient.DB)
	Manager := user.CreateManager(protosClient.DB, keyManager)

	protosClient.Manager = Manager
	protosClient.KeyManager = keyManager

	return protosClient, nil
}

func networkUp(internalDomain string) (*network.Manager, error) {
	key, err := pcrypto.GetLocalKey(config.Get().WorkDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get device key while setting up network: %w", err)
	}

	networkManager, err := network.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to configure network: %w", err)
	}

	err = networkManager.Init(key, internalDomain)
	if err != nil {
		return nil, fmt.Errorf("failed to configure network: %w", err)
	}

	return networkManager, nil
}

//
// public methods
//

func (pc *ProtosClient) Init(username string, name string, organization string) error {
	log.Debugf("Performing initialization")

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to init. Could not retrieve hostname: %w", err)
	}

	err = pc.DB.Init()
	if err != nil {
		return fmt.Errorf("failed to init. Error while initializing db: %w", err)
	}

	adminUser, err := pc.Manager.CreateUser(username, name, true)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	err = pc.Manager.AddDevice(adminUser.Username, hostname, pc.localKey)
	if err != nil {
		return fmt.Errorf("failed to add user. Error while creating user device: %w", err)
	}

	// saving the key to disk
	pc.SetInitialized()

	return nil
}

func (pc *ProtosClient) StartUp() error {
	skipNetwork := os.Getenv("PROTOS_SKIP_NETWORK") == "1"

	var networkManager *network.Manager
	if skipNetwork {
		log.Warn("skipping host network setup")
	} else {
		var err error
		networkManager, err = networkUp(pc.cfg.InternalDomain)
		if err != nil {
			return fmt.Errorf("failed to create network manager: %s", err.Error())
		}
	}

	appRuntime := runtime.Create(networkManager, pc.cfg.RuntimeEndpoint)
	appManager := app.CreateManager(app.TypeProtosc, appRuntime, pc.DB)

	p2pManager, err := p2p.NewManager(pc.localKey, appManager, pc.DB, pc.cfg.P2PPort)
	if err != nil {
		return fmt.Errorf("failed to create p2p manager: %s", err.Error())
	}
	pc.P2PManager = p2pManager

	p2pStopper, err := p2pManager.StartServer()
	if err != nil {
		return fmt.Errorf("failed to start p2p server: %s", err.Error())
	}
	pc.stoppers["p2p"] = p2pStopper

	cloudManager, err := provisioners.CreateManager(
		pc.DB,
		pc.Manager,
		pc.KeyManager,
		p2pManager,
		scaleway.NewFactory(),
		localmacos.NewFactory(),
	)
	if err != nil {
		return fmt.Errorf("failed to create cloud manager: %s", err.Error())
	}

	if !skipNetwork {
		dnsStopper := dns.StartServer(pc.localKey, config.LocalDNSPort, "", pc.cfg.InternalDomain, appManager)
		pc.stoppers["dns"] = dnsStopper
	}
	pc.AppManager = appManager
	pc.CloudManager = cloudManager
	pc.NetworkManager = networkManager

	for _, model := range []any{
		db.CLOUD_MACHINE_METADATA{},
		db.MACHINE{},
		db.PEER{},
		db.USER{},
		db.USER_DEVICE_METADATA{},
	} {
		if err := pc.DB.RegisterNotifier(model, pc); err != nil {
			return fmt.Errorf("failed to register database notifier: %w", err)
		}
	}

	if pc.DB.Initialized() {
		pc.Notify()
	}
	pc.stoppers["db-reconcile"] = db.StartPeriodicNotifier(pc, 5*time.Second)

	return nil

}

func (pc *ProtosClient) Notify() {
	pc.notifyMu.Lock()
	defer pc.notifyMu.Unlock()

	if pc.CloudManager == nil || pc.Manager == nil || pc.P2PManager == nil {
		log.Debug("Protos client not ready yet. Skipping refresh")
		return
	}

	peerIDs, err := db.GetPeerIDs(pc.DB)
	if err != nil {
		log.Errorf("failed to get peer membership: %s", err.Error())
		return
	}

	instances, err := pc.CloudManager.GetInstances(false)
	if err != nil {
		log.Errorf("failed to get instances: %s", err.Error())
		return
	}
	instances = membership.FilterInstances(instances, peerIDs)

	userDevices, err := pc.Manager.GetAllDevices(true)
	if err != nil {
		log.Errorf("failed to get user devices: %s", err.Error())
		return
	}
	userDevices = membership.FilterDevices(userDevices, peerIDs)

	if pc.NetworkManager != nil {
		err = pc.NetworkManager.ConfigurePeers(instances, userDevices)
		if err != nil {
			log.Errorf("failed to configure network peers: %s", err.Error())
			return
		}
	}

	err = pc.P2PManager.ConfigurePeers(membership.Machines(instances, userDevices))
	if err != nil {
		log.Errorf("failed to configure p2p peers: %s", err.Error())
		return
	}
}

func (pc *ProtosClient) IsInitialized() bool {
	_, err := pc.Manager.GetAdmin()
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
	resp, err := http.Get(config.ReleasesURL)
	if err != nil {
		return releases, errors.Wrapf(err, "Failed to retrieve releases from '%s'", config.ReleasesURL)
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&releases)
	if err != nil {
		return releases, errors.Wrap(err, "Failed to JSON decode the releases response")
	}

	if len(releases.Releases) == 0 {
		return releases, errors.Errorf("Something went wrong. Parsed 0 releases from '%s'", config.ReleasesURL)
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
