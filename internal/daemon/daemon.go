package daemon

import (
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
	"github.com/protosio/protos/internal/platform"
	"github.com/protosio/protos/internal/provider"
	"github.com/protosio/protos/internal/resource"
	"github.com/protosio/protos/internal/task"
	"github.com/protosio/protos/internal/util"
)

var log = util.GetLogger("daemon")

func catchSignals(sigs chan os.Signal, wg *sync.WaitGroup) {
	sig := <-sigs
	log.Infof("Received OS signal %s. Terminating", sig.String())
	wg.Done()
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

	// open databse
	dbcli, err := db.Open(cfg.WorkDir, "db")
	if err != nil {
		log.Fatal(err)
	}
	defer dbcli.Close()

	// create all the managers
	rm := resource.CreateManager(dbcli)
	m := meta.Setup(rm, dbcli, version.String())
	p := platform.Create(cfg.Runtime, cfg.RuntimeEndpoint, cfg.AppStoreHost, cfg.InContainer, m.GetKey())
	cm := capability.CreateManager()
	um := auth.CreateUserManager(dbcli, cm)
	tm := task.CreateManager(dbcli, cfg)
	as := installer.CreateAppStore(p, tm, cm)
	am := app.CreateManager(rm, tm, p, dbcli, m, cfg, as, cm)
	pm := provider.CreateManager(rm, am, dbcli)

	// check init and dev mode

	cfg.InitMode = m.InitMode() || init
	if cfg.InitMode {
		log.Info("Starting up in init mode")
	}
	cfg.DevMode = devmode
	meta.PrintBanner()

	httpAPI := api.New(devmode, cfg.StaticAssets, cfg.WSPublish, cfg.HTTPport, cfg.HTTPSport, m, am, rm, tm, pm, as, um, p, cm)

	// start ws connection manager
	err = httpAPI.StartWSManager()
	if err != nil {
		log.Fatal(err)
	}

	// start loopback webserver
	err = httpAPI.StartLoopbackWebServer(cfg.InitMode)
	if err != nil {
		log.Fatal(err)
	}

	// if starting for the first time, this will block until remote init is done
	ip, network, domain, adminUser := m.WaitForInit()
	usr, err := um.GetUser(adminUser)
	if err != nil {
		log.Fatal(err)
	}

	// perform the runtime initialization (network + container runtime)
	err = p.Init(network, usr.GetDevices())
	if err != nil {
		log.Fatal(err)
	}

	dns.StartServer(ip.String(), cfg.ExternalDNS, domain)

	err = httpAPI.StartInternalWebServer(cfg.InitMode, ip.String())
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Started all servers successfully")
	wg.Wait()
	log.Info("Terminating all servers...")

	// stop all servers
	err = httpAPI.StopWSManager()
	if err != nil {
		log.Error(err)
	}
	err = httpAPI.StopLoopbackWebServer()
	if err != nil {
		log.Error(err)
	}
	err = httpAPI.StopInternalWebServer()
	if err != nil {
		log.Error(err)
	}
	dns.StopServer()
}
