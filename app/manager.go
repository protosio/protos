package app

import (
	"encoding/gob"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/protosio/protos/capability"
	"github.com/protosio/protos/database"
	"github.com/protosio/protos/installer"
	"github.com/protosio/protos/meta"
	"github.com/protosio/protos/platform"
	"github.com/protosio/protos/resource"
	"github.com/protosio/protos/task"
	"github.com/protosio/protos/util"
	"github.com/rs/xid"
)

// Map is a thread safe application map
type Map struct {
	access *sync.Mutex
	apps   map[string]*App
}

// put saves an application into the application map
func (am Map) put(id string, app *App) {
	am.access.Lock()
	am.apps[id] = app
	am.access.Unlock()
}

// get retrieves an application from the application map
func (am Map) get(id string) (*App, error) {
	am.access.Lock()
	app, found := am.apps[id]
	am.access.Unlock()
	if found {
		return app, nil
	}
	return nil, fmt.Errorf("Could not find app %s", id)
}

func (am Map) remove(id string) error {
	am.access.Lock()
	defer am.access.Unlock()
	app, found := am.apps[id]
	if found == false {
		return fmt.Errorf("Could not find app %s", id)
	}
	err := database.Remove(app)
	if err != nil {
		log.Panicf("Failed to remove app from db: %s", err.Error())
	}
	delete(am.apps, id)
	return nil
}

// copy returns a copy of the applications map
func (am Map) copy() map[string]App {
	apps := map[string]App{}
	am.access.Lock()
	for k, v := range am.apps {
		v.access.Lock()
		app := *v
		v.access.Unlock()
		apps[k] = app
	}
	am.access.Unlock()
	return apps
}

// mapps maintains the main application map
var mapps Map

// initDB runs when Protos starts and loads all apps from the DB in memory
func initDB() {
	log.Debug("Retrieving applications from DB")
	gob.Register(&App{})
	gob.Register(&platform.DockerContainer{})

	dbapps := []*App{}
	err := database.All(&dbapps)
	if err != nil {
		log.Fatal("Could not retrieve applications from database: ", err)
	}

	mapps = Map{access: &sync.Mutex{}, apps: map[string]*App{}}
	for _, app := range dbapps {
		tmp := app
		tmp.access = &sync.Mutex{}
		mapps.put(tmp.ID, tmp)
	}
}

func saveApp(app *App) {
	app.access.Lock()
	papp := *app
	app.access.Unlock()
	papp.access = nil
	gconfig.WSPublish <- util.WSMessage{MsgType: util.WSMsgTypeUpdate, PayloadType: util.WSPayloadTypeApp, PayloadValue: papp.Public()}
	err := database.Save(&papp)
	if err != nil {
		log.Panic(errors.Wrap(err, "Could not save app to database"))
	}
}

func add(app *App) {
	mapps.put(app.ID, app)
	saveApp(app)
}

//
// Public methods
//

// Init initializes the app package by loading all the applications from the database
func Init() {
	log.Info("Initializing application manager")
	initDB()
}

// GetCopy returns a copy of an application based on its id
func GetCopy(id string) (App, error) {
	log.Debug("Copying application ", id)
	app, err := mapps.get(id)
	app.access.Lock()
	capp := *app
	app.access.Unlock()
	return capp, err
}

// CopyAll returns a copy of all the applications
func CopyAll() map[string]App {
	return mapps.copy()
}

// Read returns an application based on its id
func Read(id string) (*App, error) {
	log.Debug("Reading application ", id)
	return mapps.get(id)
}

// Select takes a function and applies it to all the apps in the map. The ones that return true are returned
func Select(filter func(*App) bool) map[string]*App {
	apps := map[string]*App{}
	mapps.access.Lock()
	for k, v := range mapps.apps {
		app := v
		app.access.Lock()
		if filter(app) {
			apps[k] = app
		}
		app.access.Unlock()
	}
	mapps.access.Unlock()
	return apps
}

// ReadByIP searches and returns an application based on it's IP address
// ToDo: refresh IP data for all applications?
func ReadByIP(appIP string) (*App, error) {
	log.Debug("Reading application with IP ", appIP)

	mapps.access.Lock()
	defer mapps.access.Unlock()
	for _, app := range mapps.apps {
		if app.IP == appIP {
			log.Debug("Found application '", app.Name, "' for IP ", appIP)
			return app, nil
		}
	}
	return nil, errors.New("Could not find any application with IP " + appIP)
}

// CreateAsync creates, runs and returns a task of type CreateAppTask
func CreateAsync(installerID string, installerVersion string, appName string, installerMetadata *installer.Metadata, installerParams map[string]string, startOnCreation bool) task.Task {
	createApp := CreateAppTask{
		InstallerID:      installerID,
		InstallerVersion: installerVersion,
		AppName:          appName,
		InstallerMedata:  installerMetadata,
		InstallerParams:  installerParams,
		StartOnCreation:  startOnCreation,
	}
	tsk := task.New(&createApp)
	return tsk
}

// Create takes an image and creates an application, without starting it
func Create(installerID string, installerVersion string, name string, installerParams map[string]string, installerMetadata *installer.Metadata, taskID string) (*App, error) {

	var app *App
	if name == "" {
		return app, fmt.Errorf("Application name cannot be empty")
	}

	err := validateInstallerParams(installerParams, installerMetadata.Params)
	if err != nil {
		return app, err
	}

	guid := xid.New()
	log.Debugf("Creating application %s(%s), based on installer %s", guid.String(), name, installerID)
	app = &App{access: &sync.Mutex{}, Name: name, ID: guid.String(), InstallerID: installerID, InstallerVersion: installerVersion,
		PublicPorts: installerMetadata.PublicPorts, InstallerParams: installerParams,
		InstallerMetadata: installerMetadata, Tasks: []string{taskID}, Status: statusCreating}

	app.Capabilities = createCapabilities(installerMetadata.Capabilities)
	if app.ValidateCapability(capability.PublicDNS) == nil {
		rsc, err := resource.Create(resource.DNS, &resource.DNSResource{Host: app.Name, Value: meta.GetPublicIP(), Type: "A", TTL: 300}, app.ID)
		if err != nil {
			return app, err
		}
		app.Resources = append(app.Resources, rsc.ID)
	}

	log.Debug("Created application ", name, "[", guid.String(), "]")
	return app, nil
}

// GetServices returns a list of services performed by apps
func GetServices() []util.Service {
	services := []util.Service{}
	apps := mapps.copy()

	resourceFilter := func(rsc *resource.Resource) bool {
		if rsc.Type == resource.DNS {
			return true
		}
		return false
	}
	rscs := resource.Select(resourceFilter)

	for _, app := range apps {
		if len(app.PublicPorts) == 0 {
			continue
		}
		service := util.Service{
			Name:  app.Name,
			Ports: app.PublicPorts,
		}

		if app.Status == statusRunning {
			service.Status = util.StatusActive
		} else {
			service.Status = util.StatusInactive
		}

		for _, rsc := range rscs {
			dnsrsc := rsc.Value.(*resource.DNSResource)
			if rsc.App == app.ID && dnsrsc.Host == app.Name {
				service.Domain = dnsrsc.Host + "." + meta.GetDomain()
				service.IP = dnsrsc.Value
				break
			}
		}
		services = append(services, service)
	}
	return services
}
