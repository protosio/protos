package app

import (
	"encoding/gob"
	"fmt"
	"sync"

	"github.com/protosio/protos/internal/capability"
	"github.com/protosio/protos/internal/core"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/installer"
	"github.com/protosio/protos/internal/meta"
	"github.com/protosio/protos/internal/task"
	"github.com/protosio/protos/internal/util"

	"github.com/pkg/errors"
	"github.com/rs/xid"
)

const (
	appDS = "app"
)

// WSPublisher returns a channel that can be used to publish WS messages to the frontend
type WSPublisher interface {
	GetWSPublishChannel() chan interface{}
}

type appStore interface {
	GetInstaller(id string) (*installer.Installer, error)
}

// dnsResource is only used locally to retrieve the Name of a DNS record
type dnsResource interface {
	GetName() string
	GetValue() string
	Update(value core.ResourceValue)
	Sanitize() core.ResourceValue
}

// Map is a thread safe application map
type Map struct {
	access *sync.Mutex
	apps   map[string]*App
	db     db.DB
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
	err := am.db.RemoveFromMap(appDS, app.ID)
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

// Manager keeps track of all the apps
type Manager struct {
	apps        Map
	store       appStore
	rm          core.ResourceManager
	tm          *task.Manager
	m           *meta.Meta
	db          db.DB
	cm          *capability.Manager
	platform    core.RuntimePlatform
	wspublisher WSPublisher
}

//
// Public methods
//

// CreateManager returns a Manager, which implements the *AppManager interface
func CreateManager(rm core.ResourceManager, tm *task.Manager, platform core.RuntimePlatform, db db.DB, meta *meta.Meta, wspublisher WSPublisher, appStore appStore, cm *capability.Manager) *Manager {

	if rm == nil || tm == nil || platform == nil || db == nil || meta == nil || wspublisher == nil || appStore == nil || cm == nil {
		log.Panic("Failed to create app manager: none of the inputs can be nil")
	}

	log.Debug("Retrieving applications from DB")
	gob.Register(&App{})
	gob.Register(&installer.InstallerMetadata{})
	err := db.InitMap(appDS, true)
	if err != nil {
		log.Fatal("Failed to initialize app dataset: ", err)
	}

	dbapps := map[string]App{}
	err = db.GetMap(appDS, &dbapps)
	if err != nil {
		log.Fatal("Could not retrieve applications from database: ", err)
	}

	manager := &Manager{rm: rm, tm: tm, db: db, m: meta, platform: platform, wspublisher: wspublisher, store: appStore, cm: cm}
	apps := Map{access: &sync.Mutex{}, apps: map[string]*App{}, db: db}
	for _, app := range dbapps {
		tmp := app
		tmp.access = &sync.Mutex{}
		tmp.parent = manager
		apps.put(tmp.ID, &tmp)
	}
	manager.apps = apps
	return manager
}

// methods to satisfy local interfaces

func (am *Manager) getPlatform() core.RuntimePlatform {
	return am.platform
}

func (am *Manager) getResourceManager() core.ResourceManager {
	return am.rm
}

func (am *Manager) getTaskManager() *task.Manager {
	return am.tm
}

func (am *Manager) getAppStore() appStore {
	return am.store
}

func (am *Manager) getCapabilityManager() *capability.Manager {
	return am.cm
}

func (am *Manager) createAppForTask(installerID string, installerVersion string, name string, installerParams map[string]string, installerMetadata installer.InstallerMetadata, taskID string) (app, error) {
	newApp, err := am.Create(installerID, installerVersion, name, installerParams, installerMetadata)
	if err != nil {
		return nil, err
	}

	return newApp, nil
}

// GetCopy returns a copy of an application based on its id
func (am *Manager) GetCopy(id string) (*App, error) {
	log.Trace("Copying application ", id)
	app, err := am.apps.get(id)
	if err != nil {
		return nil, err
	}
	app.access.Lock()
	capp := *app
	app.access.Unlock()
	return &capp, err
}

// Get returns a copy of an application based on its name
func (am *Manager) Get(name string) (*App, error) {
	for _, app := range am.apps.copy() {
		if app.Name == name {
			return &app, nil
		}
	}
	return nil, fmt.Errorf("Could not find application '%s'", name)
}

// CopyAll returns a copy of all the applications
func (am *Manager) CopyAll() map[string]*App {
	apps := map[string]*App{}
	for id, app := range am.apps.copy() {
		apps[id] = &app
	}
	return apps
}

// Read returns an application based on its id
func (am *Manager) Read(id string) (*App, error) {
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

// CreateAsync creates, runs and returns a task of type CreateAppTask
func (am *Manager) CreateAsync(installerID string, installerVersion string, appName string, installerParams map[string]string, startOnCreation bool) *task.Base {
	if installerID == "" || appName == "" {
		log.Panic("CreateAsync doesn't have all the required parameters")
	}
	createApp := CreateAppTask{
		am:               am,
		InstallerID:      installerID,
		InstallerVersion: installerVersion,
		AppName:          appName,
		InstallerParams:  installerParams,
		StartOnCreation:  startOnCreation,
	}
	return am.tm.New("Create application", &createApp)
}

// Create takes an image and creates an application, without starting it
func (am *Manager) Create(installerID string, installerVersion string, name string, installerParams map[string]string, installerMetadata installer.InstallerMetadata) (*App, error) {

	var app *App
	if name == "" || installerID == "" || installerVersion == "" {
		return app, fmt.Errorf("Application name, installer ID or installer version cannot be empty")
	}

	err := validateInstallerParams(installerParams, installerMetadata.Params)
	if err != nil {
		return app, err
	}

	for _, app := range am.apps.copy() {
		if app.Name == name {
			return nil, fmt.Errorf("Could not create application '%s': another application exists with the same name", name)
		}
	}

	guid := xid.New()
	log.Debugf("Creating application %s(%s), based on installer %s", guid.String(), name, installerID)
	app = &App{access: &sync.Mutex{}, Name: name, ID: guid.String(), InstallerID: installerID, InstallerVersion: installerVersion,
		PublicPorts: installerMetadata.PublicPorts, InstallerParams: installerParams,
		InstallerMetadata: installerMetadata, Tasks: []string{}, Status: statusCreating, parent: am}

	app.Capabilities = createCapabilities(am.cm, installerMetadata.Capabilities)
	publicDNSCapability, err := am.cm.GetByName("PublicDNS")
	if err != nil {
		return nil, errors.Wrapf(err, "Could not create application '%s'", name)
	}
	if app.ValidateCapability(publicDNSCapability) == nil {
		rsc, err := am.rm.CreateDNS(app.ID, app.Name, "A", am.m.GetPublicIP(), 300)
		if err != nil {
			return app, err
		}
		app.Resources = append(app.Resources, rsc.GetID())
	}

	am.apps.put(app.ID, app)
	am.saveApp(app)

	log.Debug("Created application ", name, "[", guid.String(), "]")
	return app, nil
}

// Delete sets the status of the app to WillDelete, which triggers the deletion of the app
func (am *Manager) Delete(name string) error {
	app, err := am.Get(name)
	if err != nil {
		return err
	}

	app.SetStatus(statusWillDelete)
	return nil
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
		return errors.Wrapf(err, "Can't remove application %s", appID)
	}

	err = app.remove()
	if err != nil {
		return errors.Wrapf(err, "Can't remove application %s(%s)", app.Name, app.ID)
	}
	app.SetStatus(statusDeleted)
	err = am.apps.remove(app.ID)
	if err != nil {
		return errors.Wrapf(err, "Failed to remove application %s(%s) from database", app.Name, app.ID)
	}
	return nil
}

// RemoveAsync asynchronously removes an applications and returns a task
func (am *Manager) RemoveAsync(appID string) *task.Base {
	return am.tm.New("Remove application", &RemoveAppTask{am: am, appID: appID})
}

func (am *Manager) saveApp(app *App) {
	app.access.Lock()
	papp := *app
	app.access.Unlock()
	papp.access = nil
	am.wspublisher.GetWSPublishChannel() <- util.WSMessage{MsgType: util.WSMsgTypeUpdate, PayloadType: util.WSPayloadTypeApp, PayloadValue: papp.Public()}
	err := am.db.InsertInMap(appDS, papp.ID, papp)
	if err != nil {
		log.Panic(errors.Wrap(err, "Could not save app to database"))
	}
}

//
// Dev related methods
//

// CreateDevApp creates an application (DEV mode). It only creates the database entry and leaves the rest to the user
func (am *Manager) CreateDevApp(appName string, installerMetadata installer.InstallerMetadata, installerParams map[string]string) (*App, error) {

	// app creation (dev purposes)
	log.Info("Creating application using local installer (DEV)")

	newApp, err := am.Create("dev", "0.0.0-dev", appName, installerParams, installerMetadata)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not create application %s", appName)
	}

	newApp.SetStatus(statusUnknown)
	return newApp, nil
}
