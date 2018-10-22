package daemon

import (
	"os"
	"sync"

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

func run(wg *sync.WaitGroup, manager func(chan bool), quit chan bool) {
	go func() {
		manager(quit)
		wg.Done()
	}()
}

// StartUp triggers a sequence of steps required to start the application
func StartUp(configFile string, init bool, version *semver.Version, incontainer bool) {
	config.Load(configFile, version)
	log.Info("Starting up...")
	var err error
	var wg sync.WaitGroup
	gconfig.InitMode = (database.Exists() == false) || init
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
	resource.LoadResourcesDB()       // required to register the resource structs with the DB
	provider.LoadProvidersDB()       // required to register the provider structs with the DB

	wg.Add(3)
	// start app manager
	gconfig.ProcsQuit["appmanager"] = make(chan bool)
	run(&wg, app.Manager, gconfig.ProcsQuit["appmanager"])
	// start task scheduler
	gconfig.ProcsQuit["taskscheduler"] = make(chan bool)
	run(&wg, task.Scheduler, gconfig.ProcsQuit["taskscheduler"])

	if gconfig.InitMode {
		// run the init webserver in blocking mode
		gconfig.ProcsQuit["initwebserver"] = make(chan bool)
		api.WebsrvInit(gconfig.ProcsQuit["initwebserver"])
		log.Info("Finished initialisation. Resuming normal operations")
	}
	gconfig.InitMode = false

	meta.InitCheck()
	// start tls web server
	gconfig.ProcsQuit["webserver"] = make(chan bool)
	run(&wg, api.Websrv, gconfig.ProcsQuit["webserver"])
	wg.Wait()

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
