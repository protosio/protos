package app

import (
	"sync"

	"github.com/pkg/errors"

	"github.com/protosio/protos/internal/capability"
	"github.com/protosio/protos/internal/installer"
	"github.com/protosio/protos/internal/platform"
	"github.com/protosio/protos/internal/resource"

	"github.com/protosio/protos/internal/util"
)

var log = util.GetLogger("app")

// Defines structure for config parameters
// specific to each application
const (
	// app states
	statusRunning    = "running"
	statusStopped    = "stopped"
	statusCreating   = "creating"
	statusFailed     = "failed"
	statusUnknown    = "unknown"
	statusDeleted    = "deleted"
	statusWillDelete = "willdelete"

	appBucket = "app"
)

// WSConnection is a websocket connection via which messages can be sent to the app, if the connection is active
type WSConnection struct {
	Send  chan interface{}
	Close chan bool
}

// Config the application config
type Config struct {
	Description string
	Image       string
	Ports       map[string]string
	Data        string
}

// App represents the application state
type App struct {
	access *sync.Mutex   `noms:"-"`
	msgq   *WSConnection `noms:"-"`
	mgr    *Manager      `noms:"-"`

	// Public members
	Name              string                      `json:"name"`
	ID                string                      `json:"id"`
	InstallerID       string                      `json:"installer-id"`
	InstallerVersion  string                      `json:"installer-version"`
	InstallerMetadata installer.InstallerMetadata `json:"installer-metadata"`
	InstanceName      string                      `json:"instance-id"`
	VolumeID          string                      `json:"volumeid"`
	DesiredStatus     string                      `json:"desired-status"`
	Actions           []string                    `json:"actions"`
	IP                string                      `json:"ip"`
	PublicPorts       []util.Port                 `json:"publicports"`
	InstallerParams   map[string]string           `json:"installer-params"`
	Capabilities      []string                    `json:"capabilities"`
	Resources         []string                    `json:"resources"`
	Tasks             []string                    `json:"tasks"`
}

//
// Utilities
//

// validateInstallerParams makes sure that the params passed at app creation match what is requested by the installer
func validateInstallerParams(paramsProvided map[string]string, paramsExpected []string) error {
	for _, param := range paramsExpected {
		if val, ok := paramsProvided[param]; ok && val != "" {
			continue
		} else {
			return errors.New("Installer parameter " + param + " should not be empty")
		}
	}
	return nil
}

func createCapabilities(cm *capability.Manager, installerCapabilities []string) []string {
	caps := []string{}
	for _, cap := range installerCapabilities {
		cap, err := cm.GetByName(cap)
		if err != nil {
			log.Error(err)
		} else {
			caps = append(caps, cap.GetName())
		}
	}
	return caps
}

// createSandbox create the underlying container
func (app *App) createSandbox() (platform.PlatformRuntimeUnit, error) {

	// normal app creation, using the app store
	inst, err := app.mgr.store.GetInstaller(app.InstallerID)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not create application '%s'", app.Name)
	}

	var version string
	if app.InstallerVersion == "" {
		version = inst.GetLastVersion()
		log.Infof("Creating application using latest version (%s) of installer '%s'", version, app.InstallerID)
	} else {
		version = app.InstallerVersion
	}

	metadata, err := inst.GetMetadata(version)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not create application '%s'", app.Name)
	}

	// var err error
	var volumeID string
	if metadata.PersistancePath != "" {
		volumeID, err = app.mgr.getPlatform().GetOrCreateVolume(app.VolumeID, metadata.PersistancePath)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create volume for app '%s'", app.ID)
		}
	}

	available, err := inst.IsPlatformImageAvailable(version)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not create application '%s'", app.Name)
	}
	if available != true {
		log.Infof("Downloading image '%s' for installer '%s'(%s) version '%s'", metadata.PlatformID, inst.Name, inst.ID, version)
		err = app.mgr.getPlatform().PullImage(metadata.PlatformID, inst.Name, version)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to download installer '%s' version '%s'", inst.ID, version)
		}
	} else {
		log.Debugf("Container image for installer %s(%s) found locally", app.InstallerID, version)
	}

	log.Infof("Creating sandbox for app '%s'[%s]", app.Name, app.ID)
	cnt, err := app.mgr.getPlatform().NewSandbox(app.Name, app.ID, metadata.PlatformID, app.VolumeID, metadata.PersistancePath, app.PublicPorts, app.InstallerParams)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create sandbox for app '%s'", app.ID)
	}
	app.access.Lock()
	app.VolumeID = volumeID
	app.IP = cnt.GetIP()
	app.access.Unlock()
	err = app.mgr.saveApp(app)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create sandbox for app '%s'", app.ID)
	}
	return cnt, nil
}

func (app *App) getOrcreateSandbox() (platform.PlatformRuntimeUnit, error) {
	cnt, err := app.mgr.getPlatform().GetSandbox(app.ID)
	if err != nil {
		if util.IsErrorType(err, platform.ErrContainerNotFound) {
			cnt, err := app.createSandbox()
			if err != nil {
				return nil, err
			}
			return cnt, nil
		}
		return nil, errors.Wrapf(err, "Failed to retrieve container for app '%s'", app.ID)
	}
	return cnt, nil
}

func (app *App) getSandbox() (platform.PlatformRuntimeUnit, error) {
	cnt, err := app.mgr.getPlatform().GetSandbox(app.ID)
	if err != nil {
		if util.IsErrorType(err, platform.ErrContainerNotFound) {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "Failed to retrieve container for app '%s'", app.ID)
	}
	return cnt, nil
}

//
// Methods for application instance
//

// GetID returns the id of the application
func (app *App) GetID() string {
	return app.ID
}

// GetName returns the id of the application
func (app *App) GetName() string {
	return app.Name
}

// SetDesiredStatus sets the status of an application
func (app *App) SetDesiredStatus(status string) error {
	app.access.Lock()
	app.DesiredStatus = status
	app.access.Unlock()
	return app.mgr.saveApp(app)
}

// GetStatus returns the status of an application
func (app *App) GetStatus() string {
	cnt, err := app.mgr.getPlatform().GetSandbox(app.ID)
	if err != nil {
		if util.IsErrorType(err, platform.ErrContainerNotFound) == false {
			log.Warnf("Failed to retrieve app (%s) sandbox: %s", app.ID, err.Error())
		}
		return statusStopped
	}

	return cnt.GetStatus()
}

// GetVersion returns the version of an application
func (app *App) GetVersion() string {
	return app.InstallerVersion
}

// AddTask adds a task owned by the applications
func (app *App) AddTask(id string) {
	app.access.Lock()
	app.Tasks = append(app.Tasks, id)
	app.access.Unlock()
	log.Debugf("Added task '%s' to app '%s'", id, app.ID)
	err := app.mgr.saveApp(app)
	if err != nil {
		log.Panic(errors.Wrapf(err, "Failed to add task for app '%s'", app.ID))
	}
}

// Start starts an application
func (app *App) Start() error {
	log.Infof("Starting application '%s'[%s]", app.Name, app.ID)

	cnt, err := app.getOrcreateSandbox()
	if err != nil {
		return errors.Wrapf(err, "Failed to start application '%s'", app.ID)
	}

	err = cnt.Start()
	if err != nil {
		return errors.Wrapf(err, "Failed to start application '%s'", app.ID)
	}
	return nil
}

// Stop stops an application
func (app *App) Stop() error {
	log.Infof("Stopping application '%s'[%s]", app.Name, app.ID)

	cnt, err := app.mgr.getPlatform().GetSandbox(app.ID)
	if err != nil {
		if util.IsErrorType(err, platform.ErrContainerNotFound) == false {
			return err
		}
		log.Warnf("Application '%s'(%s) has no sandbox to stop", app.Name, app.ID)
		return nil
	}

	err = cnt.Stop()
	if err != nil {
		return errors.Wrapf(err, "Failed to stop application '%s'(%s)", app.Name, app.ID)
	}

	err = cnt.Remove()
	if err != nil {
		return errors.Wrapf(err, "Failed to remove application '%s'(%s)", app.Name, app.ID)
	}

	return nil
}

// ReplaceContainer replaces the container of the app with the one provided. Used during development
func (app *App) ReplaceContainer(id string) error {
	log.Infof("Using container %s for app %s", id, app.Name)
	cnt, err := app.mgr.getPlatform().GetSandbox(id)
	if err != nil {
		return errors.Wrapf(err, "Failed to replace container for app '%s'", app.ID)
	}

	app.access.Lock()
	app.IP = cnt.GetIP()
	app.access.Unlock()
	if err != nil {
		return errors.Wrapf(err, "Failed to create sandbox for app '%s'", app.ID)
	}
	return nil
}

// GetIP returns the ip address of the app
func (app *App) GetIP() string {
	return app.IP
}

//
// WS connection related methods
//

// SetMsgQ sets the channel that can be used to send WS messages to the app
func (app *App) SetMsgQ(msgq *WSConnection) {
	app.access.Lock()
	app.msgq = msgq
	id := app.ID
	app.access.Unlock()
	log.Debugf("New WS connection available for app '%s'", id)
}

// CloseMsgQ closes and removes the WS connection to the application
func (app *App) CloseMsgQ() {
	app.access.Lock()
	msgq := app.msgq
	app.msgq = nil
	id := app.ID
	app.access.Unlock()
	if msgq == nil {
		return
	}
	log.Debugf("Closing WS connection for app '%s'", id)
	msgq.Close <- true
}

// SendMsg sends a message to the app via the active WS connection. Returns error if no WS connection is active
func (app *App) SendMsg(msg interface{}) error {
	app.access.Lock()
	msgq := app.msgq
	id := app.ID
	app.access.Unlock()
	if msgq == nil {
		return errors.Errorf("Application '%s' does not have a WS connection open", id)
	}
	msgq.Send <- msg
	return nil
}

//
// Resource related methods
//

//CreateResource adds a resource to the internal resources map.
func (app *App) CreateResource(appJSON []byte) (*resource.Resource, error) {

	app.access.Lock()
	rsc, err := app.mgr.getResourceManager().CreateFromJSON(appJSON, app.ID)
	if err != nil {
		app.access.Unlock()
		return nil, errors.Wrapf(err, "Failed to create resource for app '%s'", app.ID)
	}
	app.Resources = append(app.Resources, rsc.GetID())
	app.access.Unlock()
	err = app.mgr.saveApp(app)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create resource for app '%s'", app.ID)
	}

	return rsc, nil
}

//DeleteResource deletes a resource
func (app *App) DeleteResource(resourceID string) error {
	if v, index := util.StringInSlice(resourceID, app.Resources); v {
		err := app.mgr.getResourceManager().Delete(resourceID)
		if err != nil {
			return errors.Wrap(err, "Failed to delete resource for app "+app.ID)
		}
		app.access.Lock()
		app.Resources = util.RemoveStringFromSlice(app.Resources, index)
		app.access.Unlock()
		err = app.mgr.saveApp(app)
		if err != nil {
			return errors.Wrapf(err, "Failed to delete resource for app '%s'", app.ID)
		}

		return nil
	}

	return errors.Errorf("Resource '%s' not owned by application '%s'", resourceID, app.ID)
}

// GetResources retrieves all the resources that belong to an application
func (app *App) GetResources() map[string]*resource.Resource {
	resources := make(map[string]*resource.Resource)
	rm := app.mgr.getResourceManager()
	for _, rscid := range app.Resources {
		rsc, err := rm.Get(rscid)
		if err != nil {
			log.Error(errors.Wrapf(err, "Failed to get resource for app '%s'", app.ID))
			continue
		}
		resources[rscid] = rsc
	}
	return resources
}

// GetResource returns resource with provided ID, if it belongs to this app
func (app *App) GetResource(resourceID string) (*resource.Resource, error) {
	if found, _ := util.StringInSlice(resourceID, app.Resources); found {
		rsc, err := app.mgr.getResourceManager().Get(resourceID)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to get resource %s for app %s", resourceID, app.ID)
		}
		return rsc, nil

	}
	return nil, errors.Errorf("Resource '%s' not owned by application '%s'", resourceID, app.ID)
}

// ValidateCapability implements the capability checker interface
func (app *App) ValidateCapability(cap *capability.Capability) error {
	for _, capName := range app.Capabilities {
		if app.mgr.getCapabilityManager().Validate(cap, capName) {
			return nil
		}
	}
	return errors.Errorf("Method capability '%s' not satisfied by application '%s'", cap.GetName(), app.ID)
}

// Provides returns true if the application is a provider for a specific type of resource
func (app *App) Provides(rscType string) bool {
	if prov, _ := util.StringInSlice(rscType, app.InstallerMetadata.Provides); prov {
		return true
	}
	return false
}
