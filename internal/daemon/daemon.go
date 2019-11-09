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
	"github.com/protosio/protos/internal/installer"
	"github.com/protosio/protos/internal/meta"
	"github.com/protosio/protos/internal/platform"
	"github.com/protosio/protos/internal/provider"
	"github.com/protosio/protos/internal/resource"
	"github.com/protosio/protos/internal/task"
	"github.com/protosio/protos/internal/util"
)

var log = util.GetLogger("daemon")

func catchSignals(sigs chan os.Signal, cfg *config.Config) {
	sig := <-sigs
	log.Infof("Received OS signal %s. Terminating", sig.String())
	cfg.ProcsQuit.Range(func(k, v interface{}) bool {
		quitChan := v.(chan bool)
		quitChan <- true
		return true
	})
}

// StartUp triggers a sequence of steps required to start the application
func StartUp(configFile string, init bool, version *semver.Version, devmode bool) {
	// Load config and print banner
	cfg := config.Load(configFile, version)

	// Handle OS signals
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go catchSignals(sigs, cfg)

	// open databse
	db := database.CreateDatabase()
	db.Open()
	defer db.Close()

	log.Info("Starting up...")
	var err error
	var wg sync.WaitGroup
	cfg.InitMode = (db.Exists() == false) || init
	cfg.DevMode = devmode
	meta.PrintBanner()

	// Generate secret key used for JWT
	log.Info("Generating secret for JWT")
	cfg.Secret, err = util.GenerateRandomBytes(32)
	if err != nil {
		log.Fatal(err)
	}

	p := platform.Initialize(cfg.Runtime, cfg.InContainer) // required to connect to the Docker daemon
	cm := capability.CreateManager()
	um := auth.CreateUserManager(db, cm)
	tm := task.CreateManager(db, cfg)
	as := installer.CreateAppStore(p, tm, cm)
	rm := resource.CreateManager(db)
	m := meta.Setup(rm, db)
	am := app.CreateManager(rm, tm, p, db, m, cfg, as, cm)
	pm := provider.CreateManager(rm, am, db)

	// start ws connection manager
	wg.Add(1)
	wsmanagerQuit := make(chan bool, 1)
	cfg.ProcsQuit.Store("wsmanager", wsmanagerQuit)
	go func() {
		api.WSManager(am, wsmanagerQuit)
		wg.Done()
	}()

	var initInterrupted bool
	if cfg.InitMode {
		// run the init webserver in blocking mode
		initwebserverQuit := make(chan bool, 1)
		cfg.ProcsQuit.Store("initwebserver", initwebserverQuit)
		wg.Add(1)
		initInterrupted = api.WebsrvInit(initwebserverQuit, devmode, m, am, rm, tm, pm, as, as, um, p, cm)
		wg.Done()
		cm.ClearAll()
	}

	if initInterrupted == false {
		log.Info("Finished initialisation. Resuming normal operations")
		cfg.InitMode = false

		m.InitCheck()
		// start tls web server
		wg.Add(1)
		webserverQuit := make(chan bool, 1)
		cfg.ProcsQuit.Store("webserver", webserverQuit)
		go func() {
			api.Websrv(webserverQuit, devmode, m, am, rm, tm, pm, as, as, um, p, cm)
			wg.Done()
		}()
	}

	wg.Wait()
	log.Info("Terminating...")

}
