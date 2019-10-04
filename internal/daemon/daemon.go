package daemon

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/Masterminds/semver"

	"protos/internal/api"
	"protos/internal/app"
	"protos/internal/auth"
	"protos/internal/capability"
	"protos/internal/config"
	"protos/internal/database"
	"protos/internal/installer"
	"protos/internal/meta"
	"protos/internal/platform"
	"protos/internal/provider"
	"protos/internal/resource"
	"protos/internal/task"
	"protos/internal/util"
)

var gconfig = config.Get()
var log = util.GetLogger("daemon")

func catchSignals(sigs chan os.Signal) {
	sig := <-sigs
	log.Infof("Received OS signal %s. Terminating", sig.String())
	gconfig.ProcsQuit.Range(func(k, v interface{}) bool {
		quitChan := v.(chan bool)
		quitChan <- true
		return true
	})
}

// StartUp triggers a sequence of steps required to start the application
func StartUp(configFile string, init bool, version *semver.Version, incontainer bool, devmode bool) {
	// Handle OS signals
	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go catchSignals(sigs)

	// open databse
	db := database.CreateDatabase()
	db.Open()
	defer db.Close()

	// Load config and print banner
	config.Load(configFile, version)
	log.Info("Starting up...")
	var err error
	var wg sync.WaitGroup
	gconfig.InitMode = (db.Exists() == false) || init
	gconfig.DevMode = devmode
	meta.PrintBanner()

	// Generate secret key used for JWT
	log.Info("Generating secret for JWT")
	gconfig.Secret, err = util.GenerateRandomBytes(32)
	if err != nil {
		log.Fatal(err)
	}

	p := platform.Initialize(incontainer) // required to connect to the Docker daemon
	cm := capability.CreateManager()
	um := auth.CreateUserManager(db, cm)
	tm := task.CreateManager(db, gconfig)
	as := installer.CreateAppStore(p, tm, cm)
	rm := resource.CreateManager(db)
	m := meta.Setup(rm, db)
	am := app.CreateManager(rm, tm, p, db, m, gconfig, as, cm)
	pm := provider.CreateManager(rm, am, db)

	// start ws connection manager
	wg.Add(1)
	wsmanagerQuit := make(chan bool, 1)
	gconfig.ProcsQuit.Store("wsmanager", wsmanagerQuit)
	go func() {
		api.WSManager(am, wsmanagerQuit)
		wg.Done()
	}()

	var initInterrupted bool
	if gconfig.InitMode {
		// run the init webserver in blocking mode
		initwebserverQuit := make(chan bool, 1)
		gconfig.ProcsQuit.Store("initwebserver", initwebserverQuit)
		wg.Add(1)
		initInterrupted = api.WebsrvInit(initwebserverQuit, devmode, m, am, rm, tm, pm, as, as, um, p, cm)
		wg.Done()
	}

	if initInterrupted == false {
		log.Info("Finished initialisation. Resuming normal operations")
		gconfig.InitMode = false

		m.InitCheck()
		// start tls web server
		wg.Add(1)
		webserverQuit := make(chan bool, 1)
		gconfig.ProcsQuit.Store("webserver", webserverQuit)
		go func() {
			api.Websrv(webserverQuit, devmode, m, am, rm, tm, pm, as, as, um, p, cm)
			wg.Done()
		}()
	}

	wg.Wait()
	log.Info("Terminating...")

}

// Setup creates the Protos work directory
func Setup() {

	// create the workdir if it does not exist
	if _, err := os.Stat(gconfig.WorkDir); err != nil {
		if os.IsNotExist(err) {
			log.Info("Creating working directory [", gconfig.WorkDir, "]")
			err = os.Mkdir(gconfig.WorkDir, 0755)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal(err)
		}
	}

}
