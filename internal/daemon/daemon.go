package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/Masterminds/semver"

	"github.com/protosio/protos/internal/api"
	"github.com/protosio/protos/internal/app"
	"github.com/protosio/protos/internal/auth"
	"github.com/protosio/protos/internal/capability"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/dns"
	"github.com/protosio/protos/internal/installer"
	"github.com/protosio/protos/internal/meta"
	"github.com/protosio/protos/internal/p2p"
	"github.com/protosio/protos/internal/platform"
	"github.com/protosio/protos/internal/provider"
	"github.com/protosio/protos/internal/resource"
	"github.com/protosio/protos/internal/ssh"
	"github.com/protosio/protos/internal/task"
	"github.com/protosio/protos/internal/util"
)

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
func StartUp(configFile string, init bool, version *semver.Version, devmode bool) {
	// Load config and print banner
	cfg := config.Load(configFile, version)

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
	sm := ssh.CreateManager(dbcli)
	m := meta.Setup(rm, dbcli, sm, version.String())
	key, err := m.GetKey()
	if err != nil {
		log.Fatal(err)
	}
	p := platform.Create(cfg.Runtime, cfg.RuntimeEndpoint, cfg.AppStoreHost, cfg.InContainer, key.PrivateWG())
	cm := capability.CreateManager()
	um := auth.CreateUserManager(dbcli, sm, cm)
	tm := task.CreateManager(dbcli, pub)
	as := installer.CreateAppStore(p, tm, cm)
	appManager := app.CreateManager(rm, tm, p, dbcli, m, pub, as, cm)
	pm := provider.CreateManager(rm, appManager, dbcli)

	p2pManager, err := p2p.NewManager(10500, key)
	if err != nil {
		log.Fatal(err)
	}

	p2pStopper, err := p2pManager.StartServer(m, um, dbcli.GetChunkStore(), appManager)
	if err != nil {
		log.Fatal(err)
	}
	stoppers["p2p"] = p2pStopper

	// check init and dev mode

	cfg.InitMode = m.InitMode() || init
	if cfg.InitMode {
		log.Info("Starting up in init mode")
	}
	cfg.DevMode = devmode
	meta.PrintBanner()

	httpAPI := api.New(devmode, cfg.StaticAssets, pub.GetWSPublishChannel(), cfg.HTTPport, cfg.HTTPSport, m, appManager, rm, tm, pm, as, um, p, cm)

	// if starting for the first time, this will block until remote init is done
	ctx, cancel := context.WithCancel(context.Background())

	canceled := false
	ctxStopper := func() error {
		cancel()
		canceled = true
		return nil
	}
	stoppers["wfi"] = ctxStopper

	internalIP, network, domain, adminUser := m.WaitForInit(ctx)

	if canceled {
		wg.Wait()
		log.Info("Shutdown completed")
		return
	}

	usr, err := um.GetUser(adminUser)
	if err != nil {
		log.Fatal(err)
	}

	// perform the runtime initialization (network + container runtime)
	err = p.Init(network, usr.GetDevices())
	if err != nil {
		log.Fatal(err)
	}

	dnsStopper := dns.StartServer(internalIP.String(), cfg.ExternalDNS, domain)
	stoppers["dns"] = dnsStopper

	iwsStopper, err := httpAPI.StartInternalWebServer(cfg.InitMode, internalIP.String())
	if err != nil {
		log.Fatal(err)
	}
	stoppers["iws"] = iwsStopper

	fmt.Println(appManager.GetAll())

	log.Info("Started all servers successfully")
	wg.Wait()
	log.Info("Shutdown completed")

}
