package protosd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/Masterminds/semver"

	"github.com/protosio/protos/internal/api"
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
	"github.com/protosio/protos/internal/provider"
	"github.com/protosio/protos/internal/resource"
	"github.com/protosio/protos/internal/runtime"
	"github.com/protosio/protos/internal/task"
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

type publisher struct {
	pubchan chan interface{}
}

// GetWSPublishChannel returns the channel that can be used to publish messages to the available websockets
func (pub *publisher) GetWSPublishChannel() chan interface{} {
	return pub.pubchan
}

// StartUp triggers a sequence of steps required to start the application
func StartUp(configFile string, version *semver.Version, devmode bool) {
	// Load config and print banner
	cfg := config.Load(configFile, version)

	// create workdir
	if _, err := os.Stat(cfg.WorkDir); os.IsNotExist(err) {
		err := os.Mkdir(cfg.WorkDir, 0755)
		if err != nil {
			log.Fatalf("Failed to create Protos directory '%s': %w", cfg.WorkDir, err)
		}
	}

	// Handle OS signals
	var wg sync.WaitGroup
	wg.Add(1)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go catchSignals(sigs, &wg)

	// create publisher
	pub := &publisher{pubchan: make(chan interface{}, 100)}

	// open databse
	dbcli, err := db.Open(cfg.WorkDir, "db")
	if err != nil {
		log.Fatal(err)
	}
	defer dbcli.Close()

	// create all the managers
	rm := resource.CreateManager(dbcli)
	sm := pcrypto.CreateManager(dbcli)
	m := meta.Setup(dbcli, sm, version.String())
	key, err := m.GetPrivateKey()
	if err != nil {
		log.Fatal(err)
	}

	networkManager, err := network.NewManager()
	if err != nil {
		log.Fatal(err)
	}

	peerConfigurator := &PeerConfigurator{NetworkManager: networkManager}

	appRuntime := runtime.Create(networkManager, cfg.RuntimeEndpoint, cfg.InContainer)
	cm := capability.CreateManager()
	um := auth.CreateUserManager(dbcli, sm, cm, peerConfigurator)
	peerConfigurator.UserManager = um
	tm := task.CreateManager(dbcli, pub)
	as := installer.CreateAppStore(appRuntime, tm, cm)
	appManager := app.CreateManager(app.TypeProtosd, rm, tm, appRuntime, dbcli, m, pub, as, cm)
	pm := provider.CreateManager(rm, appManager, dbcli)

	p2pManager, err := p2p.NewManager(key, dbcli, appManager, m.InitMode())
	if err != nil {
		log.Fatal(err)
	}
	dbcli.AddPublisher(p2pManager)
	peerConfigurator.P2PManager = p2pManager

	cloudManager, err := cloud.CreateManager(dbcli, um, sm, p2pManager, peerConfigurator, m.InstanceName)
	if err != nil {
		log.Fatal(err)
	}
	peerConfigurator.CloudManager = cloudManager

	p2pStopper, err := p2pManager.StartServer(m, dbcli.GetChunkStore())
	if err != nil {
		log.Fatal(err)
	}
	stoppers["p2p"] = p2pStopper

	// check init and dev mode
	cfg.InitMode = m.InitMode()
	if cfg.InitMode {
		log.Info("Starting up in init mode")
	}
	cfg.DevMode = devmode
	meta.PrintBanner()

	httpAPI := api.New(devmode, cfg.StaticAssets, pub.GetWSPublishChannel(), cfg.HTTPport, cfg.HTTPSport, m, appManager, rm, tm, pm, as, um, appRuntime, cm)

	// if starting for the first time, this will block until remote init is done
	ctx, cancel := context.WithCancel(context.Background())

	canceled := false
	ctxStopper := func() error {
		cancel()
		canceled = true
		return nil
	}
	stoppers["wfi"] = ctxStopper

	// perform runtime initialization (container runtime)
	err = appRuntime.Init()
	if err != nil {
		log.Fatal(err)
	}

	internalIP, network := m.WaitForInit(ctx)

	if canceled {
		wg.Wait()
		log.Info("Shutdown completed")
		return
	}

	// perform network initialization
	err = networkManager.Init(network, internalIP, key.PrivateWG(), cfg.InternalDomain)
	if err != nil {
		log.Fatal(err)
	}

	dnsStopper := dns.StartServer(internalIP.String(), DNSPort, cfg.ExternalDNS, cfg.InternalDomain, appManager)
	stoppers["dns"] = dnsStopper

	iwsStopper, err := httpAPI.StartInternalWebServer(cfg.InitMode, internalIP.String())
	if err != nil {
		log.Fatal(err)
	}
	stoppers["iws"] = iwsStopper

	log.Info("Started all servers successfully")
	peerConfigurator.Refresh()
	appManager.Refresh()
	p2pManager.BroadcastRequestHead()
	wg.Wait()
	log.Info("Shutdown completed")

}

type PeerConfigurator struct {
	UserManager    *auth.UserManager
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
