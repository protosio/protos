package protosd

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/Masterminds/semver"

	"github.com/protosio/protos/internal/app"
	"github.com/protosio/protos/internal/banner"
	"github.com/protosio/protos/internal/cloud"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/dns"
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
	dbcli, err := db.Open(cfg.WorkDir, config.DBName, lkey)
	if err != nil {
		log.Fatal(err)
	}
	defer dbcli.Close()

	// create all the managers
	sm := pcrypto.CreateManager(dbcli)

	networkManager, err := network.NewManager()
	if err != nil {
		log.Fatal(err)
	}

	appRuntime := runtime.Create(networkManager, cfg.RuntimeEndpoint)
	userManager := user.CreateManager(dbcli, sm)
	appManager := app.CreateManager(app.TypeProtosd, appRuntime, dbcli)

	p2pManager, err := p2p.NewManager(lkey, appManager, dbcli, cfg.P2PPort)
	if err != nil {
		log.Fatal(err)
	}

	cloudManager, err := cloud.CreateManager(dbcli, userManager, sm, p2pManager)
	if err != nil {
		log.Fatal(err)
	}

	p2pStopper, err := p2pManager.StartServer()
	if err != nil {
		log.Fatal(err)
	}
	stoppers["p2p"] = p2pStopper

	banner.PrintBanner(cfg)

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

	dbNotifier := &DBNotifier{cm: cloudManager, um: userManager, nm: networkManager, p2pm: p2pManager}
	dbcli.RegisterNotifier(db.CLOUD_MACHINE_METADATA{}, dbNotifier)
	dbcli.RegisterNotifier(db.MACHINE{}, dbNotifier)
	dbcli.RegisterNotifier(db.USER{}, dbNotifier)
	dbcli.RegisterNotifier(db.USER_DEVICE_METADATA{}, dbNotifier)
	dbcli.RegisterNotifier(db.APP{}, appManager)

	log.Info("Started all servers successfully")

	if dbcli.Initialized() {
		dbNotifier.Notify()
	} else {
		log.Info("DB not initialized. Waiting for remote init")
	}

	wg.Wait()
	log.Info("Shutdown completed")

}

type DBNotifier struct {
	cm   *cloud.Manager
	um   *user.Manager
	nm   *network.Manager
	p2pm *p2p.P2P
}

func (dbn *DBNotifier) Notify() {

	instances, err := dbn.cm.GetInstances(true)
	if err != nil {
		log.Error(fmt.Errorf("failed to retrieve instances: %w", err))
		return
	}

	peers := []p2p.Machine{}
	for _, instance := range instances {
		peers = append(peers, instance)
	}

	userDevices, err := dbn.um.GetAllDevices(false)
	if err != nil {
		log.Error(fmt.Errorf("failed to retrieve user devices: %w", err))
		return
	}

	// configure peers without user devices
	err = dbn.nm.ConfigurePeers(instances, userDevices)
	if err != nil {
		log.Error(fmt.Errorf("failed to configure network peers: %w", err))
		return
	}

	// finally add user devices to peer list
	for _, device := range userDevices {
		peers = append(peers, &device)
	}

	err = dbn.p2pm.ConfigurePeers(peers)
	if err != nil {
		log.Error(fmt.Errorf("failed to configure p2p peers: %w", err))
		return
	}
}
