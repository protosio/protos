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
	"github.com/protosio/protos/internal/database"
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
	db := database.GetDatabase()
	db.Open()
	defer db.Close()

	// create all the managers
	p := platform.Initialize(cfg.Runtime, cfg.RuntimeEndpoint, cfg.AppStoreHost, cfg.InContainer, cfg, cfg.InternalInterface)
	cm := capability.CreateManager()
	um := auth.CreateUserManager(db, cm)
	tm := task.CreateManager(db, cfg)
	as := installer.CreateAppStore(p, tm, cm)
	rm := resource.CreateManager(db)
	m := meta.Setup(rm, db, version.String())
	am := app.CreateManager(rm, tm, p, db, m, cfg, as, cm)
	pm := provider.CreateManager(rm, am, db)

	// check init and dev mode

	cfg.InitMode = m.InitMode() || init
	if cfg.InitMode {
		log.Info("Starting up in init mode")
	}
	cfg.DevMode = devmode
	meta.PrintBanner()

	httpAPI := api.New(devmode, cfg.StaticAssets, cfg.InternalIP, cfg.WSPublish, cfg.HTTPport, cfg.HTTPSport, m, am, rm, tm, pm, as, um, p, cm)

	// start ws connection manager
	err := httpAPI.StartWSManager()
	if err != nil {
		log.Fatal(err)
	}

	// start DNS server
	dns.StartServer(cfg.InternalIP, cfg.ExternalDNS)

	// start insecure webserver
	err = httpAPI.StartInsecureWebServer(cfg.InitMode)
	if err != nil {
		log.Fatal(err)
	}

	// start secure webserver only if not in init mode
	if !cfg.InitMode {
		err = httpAPI.StartSecureWebServer()
		if err != nil {
			log.Fatal(err)
		}
	}

	log.Info("Started all servers successfully")
	wg.Wait()
	log.Info("Terminating all servers...")

	// stop all servers
	err = httpAPI.StopWSManager()
	if err != nil {
		log.Error(err)
	}
	err = httpAPI.StopInsecureWebServer()
	if err != nil {
		log.Error(err)
	}
	err = httpAPI.StopSecureWebServer()
	if err != nil {
		log.Error(err)
	}
	dns.StopServer()
}
