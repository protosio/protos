package daemon

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/Masterminds/semver"
	"github.com/protosio/protos/api"
	"github.com/protosio/protos/app"
	"github.com/protosio/protos/provider"
	"github.com/protosio/protos/resource"
	"github.com/protosio/protos/task"

	"github.com/protosio/protos/capability"
	"github.com/protosio/protos/config"
	"github.com/protosio/protos/database"
	"github.com/protosio/protos/meta"
	"github.com/protosio/protos/platform"
	"github.com/protosio/protos/util"
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

	// Load config and print banner
	config.Load(configFile, version)
	log.Info("Starting up...")
	var err error
	var wg sync.WaitGroup
	gconfig.InitMode = (database.Exists() == false) || init
	gconfig.DevMode = devmode
	meta.PrintBanner()

	// Generate secret key used for JWT
	log.Info("Generating secret for JWT")
	gconfig.Secret, err = util.GenerateRandomBytes(32)
	if err != nil {
		log.Fatal(err)
	}

	// open databse
	database.Open()
	defer database.Close()

	// If db does not exist or init is set to true, need to run in init mode and create the db
	if gconfig.InitMode {
		log.Info("Database file doesn't exists or init mode requested. Running in web init mode")
		meta.Setup()
	}

	capability.Initialize()
	platform.Initialize(incontainer) // required to connect to the Docker daemon
	resource.Init()                  // required to register the resource structs with the DB
	rm := resource.CreateManager()
	pm := provider.CreateManager() // required to register the provider structs with the DB
	tm := task.CreateManager()
	am := app.CreateManager(rm, tm)

	// Init app package
	app.Init()
	// Init task manager
	task.Init()
	// start ws connection manager
	wg.Add(1)
	wsmanagerQuit := make(chan bool, 1)
	gconfig.ProcsQuit.Store("wsmanager", wsmanagerQuit)
	go func() {
		api.WSManager(wsmanagerQuit)
		wg.Done()
	}()

	var initInterrupted bool
	if gconfig.InitMode {
		// run the init webserver in blocking mode
		initwebserverQuit := make(chan bool, 1)
		gconfig.ProcsQuit.Store("initwebserver", initwebserverQuit)
		wg.Add(1)
		initInterrupted = api.WebsrvInit(initwebserverQuit, devmode)
		wg.Done()
	}

	if initInterrupted == false {
		log.Info("Finished initialisation. Resuming normal operations")
		gconfig.InitMode = false

		meta.InitCheck()
		// start tls web server
		wg.Add(1)
		webserverQuit := make(chan bool, 1)
		gconfig.ProcsQuit.Store("webserver", webserverQuit)
		go func() {
			api.Websrv(webserverQuit, devmode)
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
