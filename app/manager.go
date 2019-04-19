package app

import (
	"encoding/gob"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/protosio/protos/capability"
	"github.com/protosio/protos/core"
	"github.com/protosio/protos/database"
	"github.com/protosio/protos/installer"
	"github.com/protosio/protos/platform"
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

// func add(app *App) {
// 	am.apps.put(app.ID, app)
// 	saveApp(app)
// }

// Manager keeps track of all the apps
type Manager struct {
	apps Map
	rm   core.ResourceManager
	tm   core.TaskManager
	m    core.Meta
}

//
// Public methods
//

// CreateManager returns a Manager, which implements the core.AppManager interface
func CreateManager(rm core.ResourceManager, tm core.TaskManager) core.AppManager {
	log.Debug("Retrieving applications from DB")
	gob.Register(&App{})
	gob.Register(&platform.DockerContainer{})

	dbapps := []*App{}
	err := database.All(&dbapps)
	if err != nil {
		log.Fatal("Could not retrieve applications from database: ", err)
	}

	apps := Map{access: &sync.Mutex{}, apps: map[string]*App{}}
	for _, app := range dbapps {
		tmp := app
		tmp.access = &sync.Mutex{}
		apps.put(tmp.ID, tmp)
	}
	return &Manager{apps: apps, rm: rm, tm: tm}
}

// GetCopy returns a copy of an application based on its id
func (am *Manager) GetCopy(id string) (App, error) {
	log.Debug("Copying application ", id)
	app, err := am.apps.get(id)
	app.access.Lock()
	capp := *app
	app.access.Unlock()
	return capp, err
}

// CopyAll returns a copy of all the applications
func (am *Manager) CopyAll() map[string]App {
	return am.apps.copy()
}

// Read returns an application based on its id
func (am *Manager) Read(id string) (core.App, error) {
	return am.apps.get(id)
}

// Select takes a function and applies it to all the apps in the map. The ones that return true are returned
func (am *Manager) Select(filter func(*App) bool) map[string]*App {
	apps := map[string]*App{}
	am.apps.access.Lock()
	for k, v := range am.apps.apps {
		app := v
		app.access.Lock()
		if filter(app) {
			apps[k] = app
		}
		app.access.Unlock()
	}
	am.apps.access.Unlock()
	return apps
}

// ReadByIP searches and returns an application based on it's IP address
// ToDo: refresh IP data for all applications?
func (am *Manager) ReadByIP(appIP string) (*App, error) {
	am.apps.access.Lock()
	defer am.apps.access.Unlock()
	for _, app := range am.apps.apps {
		if app.IP == appIP {
			return app, nil
		}
	}
	return nil, errors.New("Could not find any application with IP " + appIP)
}

// CreateAsync creates, runs and returns a task of type CreateAppTask
func (am *Manager) CreateAsync(installerID string, installerVersion string, appName string, installerMetadata *installer.Metadata, installerParams map[string]string, startOnCreation bool) core.CustomTask {
	createApp := CreateAppTask{
		InstallerID:       installerID,
		InstallerVersion:  installerVersion,
		AppName:           appName,
		InstallerMetadata: installerMetadata,
		InstallerParams:   installerParams,
		StartOnCreation:   startOnCreation,
	}
	return &createApp
}

// Create takes an image and creates an application, without starting it
func (am *Manager) Create(installerID string, installerVersion string, name string, installerParams map[string]string, installerMetadata *installer.Metadata, taskID string) (*App, error) {

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
		rc := am.rm.(core.ResourceCreator)
		rsc, err := rc.CreateDNS(app.ID, app.Name, "A", am.m.GetPublicIP(), 300)
		if err != nil {
			return app, err
		}
		app.Resources = append(app.Resources, rsc.GetID())
	}

	am.apps.put(app.ID, app)
	saveApp(app)

	log.Debug("Created application ", name, "[", guid.String(), "]")
	return app, nil
}

// dnsResource is only used locally to retrieve the Name of a DNS record
type dnsResource interface {
	GetName() string
	GetValue() string
}

// GetServices returns a list of services performed by apps
func (am *Manager) GetServices() []util.Service {
	services := []util.Service{}
	apps := am.apps.copy()

	resourceFilter := func(rsc core.Resource) bool {
		if rsc.GetType() == core.DNS {
			return true
		}
		return false
	}
	rscs := am.rm.Select(resourceFilter)

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
			dnsrsc := rsc.GetValue().(dnsResource)
			if rsc.GetAppID() == app.ID && dnsrsc.GetName() == app.Name {
				service.Domain = dnsrsc.GetName() + "." + am.m.GetDomain()
				service.IP = dnsrsc.GetValue()
				break
			}
		}
		services = append(services, service)
	}
	return services
}

// Remove removes an application based on the provided id
func (am *Manager) Remove(appID string) error {
	app, err := am.apps.get(appID)
	if err != nil {
		return errors.Wrapf(err, "Can't remove application %s", app.ID)
	}

	err = app.remove()
	if err != nil {
		return errors.Wrapf(err, "Can't remove application %s(%s)", app.Name, app.ID)
	}
	return nil
}

// RemoveAsync asynchronously removes an applications and returns a task
func (am *Manager) RemoveAsync(appID string) core.Task {
	return am.tm.New(&RemoveAppTask{am: am, appID: appID})
}

//
// Dev related methods
//

// CreateDevApp creates an application (DEV mode). It only creates the database entry and leaves the rest to the user
func (am *Manager) CreateDevApp(installerID string, installerVersion string, appName string, installerMetadata *installer.Metadata, installerParams map[string]string) (*App, error) {

	// app creation (dev purposes)
	log.Info("Creating application using local installer (DEV)")

	app, err := am.Create(installerID, installerVersion, appName, installerParams, installerMetadata, "sync")
	if err != nil {
		return nil, errors.Wrapf(err, "Could not create application %s", appName)
	}

	app.SetStatus(statusUnknown)
	return app, nil
}
