package protosd

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/Masterminds/semver"

	"github.com/protosio/protos/internal/app"
	"github.com/protosio/protos/internal/cloud"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/dns"
	"github.com/protosio/protos/internal/meta"
	"github.com/protosio/protos/internal/network"
	"github.com/protosio/protos/internal/p2p"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/runtime"
	"github.com/protosio/protos/internal/user"
	"github.com/protosio/protos/internal/util"
)

const DNSPort = 53

var log = util.GetLogger("daemon")

var stoppers = map[string]func() error{}

func catchSignals(sigs chan os.Signal, wg *sync.WaitGroup) {
	sig := <-sigs
	log.Infof("Received OS signal %s. Terminating", sig.String())
	for _, stopper := range stoppers {
		err := stopper()
		if err != nil {
			log.Error(err)
		}
	}
	wg.Done()
}

// StartUp triggers a sequence of steps required to start the application
func StartUp(configFile string, version *semver.Version, devmode bool) {
	// Load config and print banner
	cfg := config.Load(configFile, version)

	// create workdir
	if _, err := os.Stat(cfg.WorkDir); os.IsNotExist(err) {
		err := os.Mkdir(cfg.WorkDir, 0755)
		if err != nil {
			log.Fatalf("Failed to create Protos directory '%s': %s", cfg.WorkDir, err.Error())
		}
	}

	// Handle OS signals
	var wg sync.WaitGroup
	wg.Add(1)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go catchSignals(sigs, &wg)

	// retrieve local key
	lkey, err := pcrypto.GetLocalKey(cfg.WorkDir)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to get local key: %w", err))
	}

	// open databse
	dbcli, err := db.Open(cfg.WorkDir, "db", lkey)
	if err != nil {
		log.Fatal(err)
	}
	defer dbcli.Close()

	// create all the managers
	sm := pcrypto.CreateManager(dbcli)
	m := meta.Setup(dbcli, sm, version.String())

	networkManager, err := network.NewManager()
	if err != nil {
		log.Fatal(err)
	}

	peerConfigurator := &PeerConfigurator{NetworkManager: networkManager}

	appRuntime := runtime.Create(networkManager, cfg.RuntimeEndpoint)
	um := user.CreateManager(dbcli, sm, peerConfigurator)
	peerConfigurator.UserManager = um
	appManager := app.CreateManager(app.TypeProtosd, appRuntime, dbcli, m)

	p2pManager, err := p2p.NewManager(lkey, appManager, dbcli, cfg.P2PPort)
	if err != nil {
		log.Fatal(err)
	}
	peerConfigurator.P2PManager = p2pManager

	cloudManager, err := cloud.CreateManager(dbcli, um, sm, p2pManager, peerConfigurator)
	if err != nil {
		log.Fatal(err)
	}
	peerConfigurator.CloudManager = cloudManager

	p2pStopper, err := p2pManager.StartServer()
	if err != nil {
		log.Fatal(err)
	}
	stoppers["p2p"] = p2pStopper

	// check init and dev mode
	if !dbcli.Initialized() {
		log.Info("DB not initialized. Waiting for remote init")
	}

	meta.PrintBanner()

	canceled := false
	ctxStopper := func() error {
		canceled = true
		return nil
	}
	stoppers["wfi"] = ctxStopper

	// perform runtime initialization (container runtime)
	err = appRuntime.Init()
	if err != nil {
		log.Fatal(err)
	}

	if canceled {
		wg.Wait()
		log.Info("Shutdown completed")
		return
	}

	// perform network initialization
	err = networkManager.Init(lkey, cfg.InternalDomain)
	if err != nil {
		log.Fatal(err)
	}

	dnsStopper := dns.StartServer(lkey, DNSPort, cfg.ExternalDNS, cfg.InternalDomain, appManager)
	stoppers["dns"] = dnsStopper

	log.Info("Started all servers successfully")
	peerConfigurator.Refresh()
	appManager.Refresh()
	wg.Wait()
	log.Info("Shutdown completed")

}

type PeerConfigurator struct {
	UserManager    *user.UserManager
	NetworkManager *network.Manager
	CloudManager   *cloud.Manager
	P2PManager     *p2p.P2P
}

func (pc *PeerConfigurator) Refresh() error {

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
