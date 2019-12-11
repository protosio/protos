package app

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"

	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/core"

	"github.com/protosio/protos/internal/util"
)

var log = util.GetLogger("app")
var gconfig = config.Get()

// Defines structure for config parameters
// specific to each application
const (
	// app states
	statusRunning  = "running"
	statusStopped  = "stopped"
	statusCreating = "creating"
	statusFailed   = "failed"
	statusUnknown  = "unknown"

	appBucket = "app"
)

type appParent interface {
	Remove(appID string) error
	saveApp(app *App)
	getPlatform() core.RuntimePlatform
	getTaskManager() core.TaskManager
	getResourceManager() core.ResourceManager
	getCapabilityManager() core.CapabilityManager
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
	access *sync.Mutex
	parent appParent

	// Public members
	Name              string                 `json:"name"`
	ID                string                 `json:"id"`
	InstallerID       string                 `json:"installer-id"`
	InstallerVersion  string                 `json:"installer-version"`
	InstallerMetadata core.InstallerMetadata `json:"installer-metadata"`
	ContainerID       string                 `json:"container-id"`
	VolumeID          string                 `json:"volumeid"`
	Status            string                 `json:"status"`
	Actions           []string               `json:"actions"`
	IP                string                 `json:"ip"`
	PublicPorts       []util.Port            `json:"publicports"`
	InstallerParams   map[string]string      `json:"installer-params"`
	Capabilities      []string               `json:"capabilities"`
	Resources         []string               `json:"resources"`
	Tasks             []string               `json:"-"`
	msgq              *core.WSConnection
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

func createCapabilities(cm core.CapabilityManager, installerCapabilities []string) []string {
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

// SetStatus is used to set the status of an application
func (app *App) SetStatus(status string) {
	app.access.Lock()
	app.Status = status
	app.access.Unlock()
	app.Save()
}

// AddAction performs an action on an application
func (app *App) AddAction(action string) (core.Task, error) {
	log.Info("Performing action [", action, "] on application ", app.Name, "[", app.ID, "]")

	switch action {
	case "start":
		tsk := app.StartAsync()
		return tsk, nil
	case "stop":
		tsk := app.StopAsync()
		return tsk, nil
	default:
		return nil, fmt.Errorf("Action '%s' not supported", action)
	}
}

// AddTask adds a task owned by the applications
func (app *App) AddTask(id string) {
	app.access.Lock()
	app.Tasks = append(app.Tasks, id)
	app.access.Unlock()
	app.Save()
}

// Save - sends update to the app manager which persists application data to database
func (app *App) Save() {
	app.parent.saveApp(app)
}

// reateContainer create the underlying Docker container
func (app *App) createSandbox() (core.PlatformRuntimeUnit, error) {
	// var volume *platform.DockerVolume
	var err error
	var volumeID string
	if app.InstallerMetadata.PersistancePath != "" {
		volumeID, err = app.parent.getPlatform().GetOrCreateVolume(app.VolumeID, app.InstallerMetadata.PersistancePath)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create volume for app '%s'", app.ID)
		}
	}

	log.Infof("Creating sandbox for app '%s'[%s]", app.Name, app.ID)
	cnt, err := app.parent.getPlatform().NewSandbox(app.Name, app.ID, app.InstallerMetadata.PlatformID, app.VolumeID, app.InstallerMetadata.PersistancePath, app.PublicPorts, app.InstallerParams)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create container for app '%s'", app.ID)
	}
	app.access.Lock()
	app.VolumeID = volumeID
	app.ContainerID = cnt.GetID()
	app.IP = cnt.GetIP()
	app.access.Unlock()
	app.Save()
	return cnt, nil
}

func (app *App) getOrcreateSandbox() (core.PlatformRuntimeUnit, error) {
	cnt, err := app.parent.getPlatform().GetSandbox(app.ContainerID)
	if err != nil {
		if util.IsErrorType(err, core.ErrContainerNotFound) {
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

// enrichAppData updates the information about the underlying application
func (app *App) enrichAppData() {

	if app.Status == statusCreating || app.Status == statusFailed {
		// not refreshing the platform until the app creation process is done
		return
	}

	cnt, err := app.parent.getPlatform().GetSandbox(app.ContainerID)
	if err != nil {
		if util.IsErrorType(err, core.ErrContainerNotFound) {
			log.Warnf("Application '%s'(%s) has no container: %s", app.Name, app.ID, err.Error())
			app.Status = statusStopped
			return
		}
		log.Errorf("Failed to enrich app data: %s", err.Error())
		app.Status = statusUnknown
		return
	}

	app.Status = cnt.GetStatus()
}

// StartAsync asynchronously starts an application and returns a task
func (app *App) StartAsync() core.Task {
	return app.parent.getTaskManager().New("Start application", &StartAppTask{app: app})
}

// Start starts an application
func (app *App) Start() error {
	log.Infof("Starting application '%s'[%s]", app.Name, app.ID)

	cnt, err := app.getOrcreateSandbox()
	if err != nil {
		app.SetStatus(statusFailed)
		return errors.Wrapf(err, "Failed to start application '%s'", app.ID)
	}

	err = cnt.Start()
	if err != nil {
		app.SetStatus(statusFailed)
		return errors.Wrapf(err, "Failed to start application '%s'", app.ID)
	}
	app.SetStatus(statusRunning)
	return nil
}

// StopAsync asynchronously stops an application and returns a task
func (app *App) StopAsync() core.Task {
	return app.parent.getTaskManager().New("Stop application", &StopAppTask{app: app})
}

// Stop stops an application
func (app *App) Stop() error {
	log.Infof("Stopping application '%s'[%s]", app.Name, app.ID)

	cnt, err := app.parent.getPlatform().GetSandbox(app.ContainerID)
	if err != nil {
		if util.IsErrorType(err, core.ErrContainerNotFound) == false {
			app.SetStatus(statusUnknown)
			return err
		}
		log.Warnf("Application '%s'(%s) has no container to stop", app.Name, app.ID)
		app.SetStatus(statusStopped)
		return nil
	}

	err = cnt.Stop()
	if err != nil {
		app.SetStatus(statusUnknown)
		return errors.Wrapf(err, "Failed to stop application '%s'(%s)", app.Name, app.ID)
	}
	app.SetStatus(statusStopped)
	return nil
}

// remove App removes an application container
func (app *App) remove() error {
	log.Debugf("Removing application '%s'[%s]", app.Name, app.ID)

	cnt, err := app.parent.getPlatform().GetSandbox(app.ContainerID)
	if err != nil {
		if util.IsErrorType(err, core.ErrContainerNotFound) == false {
			return errors.Wrapf(err, "Failed to remove application '%s'(%s)", app.Name, app.ID)
		}
		log.Warnf("Application %s(%s) has no container to remove", app.Name, app.ID)
	} else {
		err := cnt.Remove()
		if err != nil {
			return errors.Wrapf(err, "Failed to remove application '%s'(%s)", app.Name, app.ID)
		}
	}

	// perform CleanUpSandbox for the sandbox
	err = app.parent.getPlatform().CleanUpSandbox(app.ContainerID)
	if err != nil {
		log.Warnf("Failed to perform CleanUpSandbox for sandbox '%s': %s", app.ContainerID, err.Error())
	}

	if app.VolumeID != "" {
		err := app.parent.getPlatform().RemoveVolume(app.VolumeID)
		if err != nil {
			return errors.Wrapf(err, "Failed to remove application '%s'(%s)", app.Name, app.ID)
		}
	}

	// Removing resources requested by this app
	for _, rscID := range app.Resources {
		_, err := app.parent.getResourceManager().Get(rscID)
		if err != nil {
			log.Error(err)
			continue
		}
		err = app.parent.getResourceManager().Delete(rscID)
		if err != nil {
			log.Error(err)
			continue
		}
	}

	return nil
}

// ReplaceContainer replaces the container of the app with the one provided. Used during development
func (app *App) ReplaceContainer(id string) error {
	log.Infof("Using container %s for app %s", id, app.Name)
	cnt, err := app.parent.getPlatform().GetSandbox(id)
	if err != nil {
		return errors.Wrapf(err, "Failed to replace container for app '%s'", app.ID)
	}

	app.access.Lock()
	app.ContainerID = id
	app.IP = cnt.GetIP()
	app.access.Unlock()
	app.Save()
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
func (app *App) SetMsgQ(msgq *core.WSConnection) {
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
func (app *App) CreateResource(appJSON []byte) (core.Resource, error) {

	app.access.Lock()
	rsc, err := app.parent.getResourceManager().CreateFromJSON(appJSON, app.ID)
	if err != nil {
		app.access.Unlock()
		return nil, errors.Wrapf(err, "Failed to create resource for app '%s'", app.ID)
	}
	app.Resources = append(app.Resources, rsc.GetID())
	app.access.Unlock()
	app.Save()

	return rsc, nil
}

//DeleteResource deletes a resource
func (app *App) DeleteResource(resourceID string) error {
	if v, index := util.StringInSlice(resourceID, app.Resources); v {
		err := app.parent.getResourceManager().Delete(resourceID)
		if err != nil {
			return errors.Wrap(err, "Failed to delete resource for app "+app.ID)
		}
		app.access.Lock()
		app.Resources = util.RemoveStringFromSlice(app.Resources, index)
		app.access.Unlock()
		app.Save()

		return nil
	}

	return errors.Errorf("Resource '%s' not owned by application '%s'", resourceID, app.ID)
}

// GetResources retrieves all the resources that belong to an application
func (app *App) GetResources() map[string]core.Resource {
	resources := make(map[string]core.Resource)
	rm := app.parent.getResourceManager()
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
func (app *App) GetResource(resourceID string) (core.Resource, error) {
	if found, _ := util.StringInSlice(resourceID, app.Resources); found {
		rsc, err := app.parent.getResourceManager().Get(resourceID)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to get resource %s for app %s", resourceID, app.ID)
		}
		return rsc, nil

	}
	return nil, errors.Errorf("Resource '%s' not owned by application '%s'", resourceID, app.ID)
}

// ValidateCapability implements the capability checker interface
func (app *App) ValidateCapability(cap core.Capability) error {
	for _, capName := range app.Capabilities {
		if app.parent.getCapabilityManager().Validate(cap, capName) {
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
