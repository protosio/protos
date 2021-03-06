package app

import (
	"encoding/gob"
	"fmt"
	"sync"

	"github.com/protosio/protos/internal/capability"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/installer"
	"github.com/protosio/protos/internal/meta"
	"github.com/protosio/protos/internal/platform"
	"github.com/protosio/protos/internal/resource"
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
	Update(value resource.ResourceValue)
	Sanitize() resource.ResourceValue
}

// Manager keeps track of all the apps
type Manager struct {
	store       appStore
	rm          *resource.Manager
	tm          *task.Manager
	m           *meta.Meta
	db          db.DB
	cm          *capability.Manager
	platform    platform.RuntimePlatform
	wspublisher WSPublisher
}

//
// Public methods
//

// CreateManager returns a Manager, which implements the *AppManager interface
func CreateManager(rm *resource.Manager, tm *task.Manager, platform platform.RuntimePlatform, db db.DB, meta *meta.Meta, wspublisher WSPublisher, appStore appStore, cm *capability.Manager) *Manager {

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

	return &Manager{rm: rm, tm: tm, db: db, m: meta, platform: platform, wspublisher: wspublisher, store: appStore, cm: cm}
}

// methods to satisfy local interfaces

func (am *Manager) getPlatform() platform.RuntimePlatform {
	return am.platform
}

func (am *Manager) getResourceManager() *resource.Manager {
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

//
// Client methods
//

// Create takes an image and creates an application, without starting it
func (am *Manager) Create(installerID string, installerVersion string, name string, instanceName string, installerParams map[string]string, installerMetadata installer.InstallerMetadata) (*App, error) {

	var app *App
	if name == "" || installerID == "" || installerVersion == "" || instanceName == "" {
		return app, fmt.Errorf("Application name, installer ID, installer version or instance ID cannot be empty")
	}

	err := validateInstallerParams(installerParams, installerMetadata.Params)
	if err != nil {
		return app, err
	}

	apps := map[string]App{}
	err = am.db.GetMap(appDS, &apps)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not create application '%s'", name)
	}

	for _, app := range apps {
		if app.Name == name {
			return nil, fmt.Errorf("Could not create application '%s': another application exists with the same name", name)
		}
	}

	guid := xid.New()
	log.Debugf("Creating application %s(%s), based on installer %s", guid.String(), name, installerID)
	app = &App{
		access: &sync.Mutex{},
		mgr:    am,

		Name:              name,
		ID:                guid.String(),
		InstallerID:       installerID,
		InstallerVersion:  installerVersion,
		InstanceName:      instanceName,
		PublicPorts:       installerMetadata.PublicPorts,
		InstallerParams:   installerParams,
		InstallerMetadata: installerMetadata,
		Tasks:             []string{},
		Status:            statusCreating,
	}

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

	err = am.saveApp(app)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not create application '%s'", name)
	}

	log.Debug("Created application ", name, "[", guid.String(), "]")
	return app, nil
}

//
// Instance methods
//

// GetByID returns an application based on its id
func (am *Manager) GetByID(id string) (*App, error) {
	apps := map[string]App{}
	err := am.db.GetMap(appDS, &apps)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not retrieve application '%s'", id)
	}

	for _, app := range apps {
		if app.ID == id {
			app.mgr = am
			app.access = &sync.Mutex{}
			return &app, nil
		}
	}

	return nil, errors.Wrapf(err, "Could not find application '%s'", id)
}

// Get returns a copy of an application based on its name
func (am *Manager) Get(name string) (*App, error) {
	apps := map[string]App{}
	err := am.db.GetMap(appDS, &apps)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not retrieve application '%s'", name)
	}

	for _, app := range apps {
		if app.Name == name {
			app.mgr = am
			app.access = &sync.Mutex{}
			return &app, nil
		}
	}

	return nil, errors.Wrapf(err, "Could not find application '%s'", name)
}

// GetAll returns a copy of all the applications
func (am *Manager) GetAll() (map[string]App, error) {
	apps := map[string]App{}
	err := am.db.GetMap(appDS, &apps)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not retrieve applications")
	}

	return apps, nil
}

// ReSync checks the db for new apps and deploys them if they belong to the current instance
func (am *Manager) ReSync() {
	log.Debug("Syncing apps")
	dbapps := map[string]App{}
	err := am.db.GetMap(appDS, &dbapps)
	if err != nil {
		log.Fatal("Could not retrieve applications from database: ", err)
	}

	for _, app := range dbapps {
		if app.InstanceName == am.m.GetInstanceName() {
			app.mgr = am
			app.access = &sync.Mutex{}
			log.Infof("App '%s' status: '%s'", app.Name, app.Status)
			sandBox, err := app.getSandbox()
			if err != nil {
				log.Error("Failed to retrieve sandbox for app '%s': '%s'", app.Name, err.Error())
				continue
			}
			if app.Status == statusCreating && sandBox == nil {
				sandBox, err = app.createSandbox()
				if err != nil {
					log.Errorf("Failed to create sandbox for app '%s': '%s'", app.Name, err.Error())
					continue
				}

				app.Status = statusRunning
				err = app.mgr.saveApp(&app)
				if err != nil {
					log.Errorf("Failed to save app '%s': '%s'", app.Name, err.Error())
					continue
				}
			} else if app.Status == statusWillDelete && sandBox != nil {
				err = app.removeSandbox()
				if err != nil {
					log.Errorf("Failed to delete sandbox for app '%s': '%s'", app.Name, err.Error())
					continue
				}
			}
		}
	}
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
func (am *Manager) GetServices() ([]util.Service, error) {
	services := []util.Service{}
	apps, err := am.GetAll()
	if err != nil {
		return services, errors.Wrap(err, "Could not retrieve services")
	}

	resourceFilter := func(rsc *resource.Resource) bool {
		if rsc.GetType() == resource.DNS {
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
	return services, nil
}

// Remove removes an application based on the provided id
func (am *Manager) Remove(appID string) error {
	app, err := am.GetByID(appID)
	if err != nil {
		return errors.Wrapf(err, "Can't remove application %s", appID)
	}

	err = app.removeSandbox()
	if err != nil {
		return errors.Wrapf(err, "Can't remove application %s(%s)", app.Name, app.ID)
	}
	app.SetStatus(statusDeleted)

	return nil
}

func (am *Manager) saveApp(app *App) error {
	app.access.Lock()
	papp := *app
	app.access.Unlock()
	papp.access = nil
	am.wspublisher.GetWSPublishChannel() <- util.WSMessage{MsgType: util.WSMsgTypeUpdate, PayloadType: util.WSPayloadTypeApp, PayloadValue: papp.Public()}
	err := am.db.InsertInMap(appDS, papp.ID, papp)
	if err != nil {
		return errors.Wrap(err, "Could not save app to database")
	}
	return nil
}

//
// Dev related methods
//

// CreateDevApp creates an application (DEV mode). It only creates the database entry and leaves the rest to the user
func (am *Manager) CreateDevApp(appName string, installerMetadata installer.InstallerMetadata, installerParams map[string]string) (*App, error) {

	// app creation (dev purposes)
	log.Info("Creating application using local installer (DEV)")

	newApp, err := am.Create("dev", "0.0.0-dev", appName, "", installerParams, installerMetadata)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not create application %s", appName)
	}

	newApp.SetStatus(statusUnknown)
	return newApp, nil
}
