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

	"github.com/denisbrodbeck/machineid"
	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/app"
	"github.com/protosio/protos/internal/auth"
	"github.com/protosio/protos/internal/cloud"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/dns"
	"github.com/protosio/protos/internal/meta"
	"github.com/protosio/protos/internal/network"
	"github.com/protosio/protos/internal/p2p"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/release"
	"github.com/protosio/protos/internal/runtime"

	"github.com/protosio/protos/internal/util"
)

var log = util.GetLogger("protosc")

const (
	releasesURL     = "https://releases.protos.io/releases.json"
	localDNSPort    = 10053
	localDNSAddress = "127.0.0.1"
	dbName          = "protos"
)

type ProtosClient struct {
	stoppers map[string]func() error
	db       *db.DB
	cfg      *config.Config
	version  string
	wg       sync.WaitGroup
	localKey *pcrypto.Key

	AuthManager    *auth.AuthManager
	KeyManager     *pcrypto.Manager
	AppManager     *app.Manager
	NetworkManager *network.Manager
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

	// get local key
	lkey, err := pcrypto.GetLocalKey(dataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get local key: %w", err)
	}
	protosClient.localKey = lkey

	// open db
	protosClient.db, err = db.Open(dataPath, dbName, lkey)
	if err != nil {
		return nil, fmt.Errorf("failed to open db during configuration: %w", err)
	}

	// create various managers
	keyManager := pcrypto.CreateManager(protosClient.db)
	metaClient := meta.Setup(protosClient.db, keyManager, version)
	AuthManager := auth.CreateAuthManager(protosClient.db, keyManager, protosClient)

	protosClient.AuthManager = AuthManager
	protosClient.KeyManager = keyManager
	protosClient.Meta = metaClient

	return protosClient, nil
}

func networkUp(authMgr *auth.AuthManager, internalDomain string) (*network.Manager, error) {
	currentDevice, err := authMgr.GetCurrentDevice()
	if err != nil {
		return nil, fmt.Errorf("failed to get current device while setting up network: %w", err)
	}

	ip, netp, err := net.ParseCIDR(currentDevice.Network)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CIDR while setting up network: %w", err)
	}
	netp.IP = ip
	internalIP := netp.IP.Mask(netp.Mask)
	internalIP[3]++

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

	host, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to init. Could not retrieve hostname: %w", err)
	}

	err = pc.db.Init()
	if err != nil {
		return fmt.Errorf("failed to init. Error while initializing db: %w", err)
	}

	adminUser, err := pc.AuthManager.CreateUser(username, name, true)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	machineID, err := machineid.ProtectedID("protos")
	if err != nil {
		return fmt.Errorf("failed to add user. Error while generating machine id: %w", err)
	}

	err = pc.AuthManager.AddDevice(adminUser.Username, machineID, host, pc.localKey.PublicString(), "10.100.0.1/24")
	if err != nil {
		return fmt.Errorf("failed to add user. Error while creating user device: %w", err)
	}

	// saving the key to disk
	pc.SetInitialized()

	return nil
}

func (pc *ProtosClient) FinishInit() error {

	networkManager, err := networkUp(pc.AuthManager, pc.cfg.InternalDomain)
	if err != nil {
		return fmt.Errorf("failed to create network manager: %s", err.Error())
	}

	appRuntime := runtime.Create(networkManager, pc.cfg.RuntimeEndpoint)
	appManager := app.CreateManager(app.TypeProtosc, appRuntime, pc.db, pc.Meta)

	p2pManager, err := p2p.NewManager(pc.localKey, appManager, pc.db, pc.cfg.P2PPort)
	if err != nil {
		return fmt.Errorf("failed to create p2p manager: %s", err.Error())
	}
	pc.P2PManager = p2pManager

	p2pStopper, err := p2pManager.StartServer(pc.Meta)
	if err != nil {
		return fmt.Errorf("failed to start p2p server: %s", err.Error())
	}
	pc.stoppers["p2p"] = p2pStopper

	currentDevice, err := pc.AuthManager.GetCurrentDevice()
	if err != nil {
		return fmt.Errorf("failed to get current device: %s", err.Error())
	}

	cloudManager, err := cloud.CreateManager(pc.db, pc.AuthManager, pc.KeyManager, p2pManager, pc, currentDevice.Name)
	if err != nil {
		return fmt.Errorf("failed to create cloud manager: %s", err.Error())
	}

	dnsStopper := dns.StartServer(pc.localKey, localDNSPort, "", pc.cfg.InternalDomain, appManager)
	pc.stoppers["dns"] = dnsStopper
	pc.AppManager = appManager
	pc.CloudManager = cloudManager
	pc.NetworkManager = networkManager

	err = pc.Refresh()
	if err != nil {
		return fmt.Errorf("failed to refresh state: %s", err.Error())
	}

	return nil

}

func (pc *ProtosClient) Refresh() error {

	if pc.CloudManager == nil || pc.AuthManager == nil || pc.P2PManager == nil {
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

	admin, err := pc.AuthManager.GetAdmin()
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
	_, err := pc.AuthManager.GetAdmin()
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
