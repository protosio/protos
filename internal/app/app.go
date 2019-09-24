package app

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"

	"protos/internal/config"
	"protos/internal/core"

	"protos/internal/capability"
	"protos/internal/resource"
	"protos/internal/util"
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

type parent interface {
	saveApp(app *App)
	getPlatform() core.RuntimePlatform
	getTaskManager() core.TaskManager
	getResourceManager() core.ResourceManager
}

// Config the application config
type Config struct {
	Description string
	Image       string
	Ports       map[string]string
	Data        string
}

// WSConnection is a websocket connection via which messages can be sent to the app, if the connection is active
type WSConnection struct {
	Send  chan interface{}
	Close chan bool
}

// App represents the application state
type App struct {
	access *sync.Mutex
	parent parent

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
	msgq              *WSConnection
}

//
// Utilities
//

func containerToAppStatus(status string, exitCode int) string {
	switch status {
	case "created":
		return statusStopped
	case "container missing":
		return statusStopped
	case "restarting":
		return statusStopped
	case "paused":
		return statusStopped
	case "exited":
		if exitCode == 0 {
			return statusStopped
		}
		return statusFailed
	case "dead":
		return statusFailed
	case "removing":
		return statusRunning
	case "running":
		return statusRunning
	default:
		return statusUnknown
	}
}

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

func createCapabilities(installerCapabilities []*capability.Capability) []string {
	caps := []string{}
	for _, cap := range installerCapabilities {
		caps = append(caps, cap.Name)
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
func (app *App) createContainer() (core.PlatformRuntimeUnit, error) {
	// var volume *platform.DockerVolume
	var err error
	var volumeID string
	if app.InstallerMetadata.PersistancePath != "" {
		volumeID, err = app.parent.getPlatform().GetOrCreateVolume(app.VolumeID, app.InstallerMetadata.PersistancePath)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create volume for app '%s'", app.ID)
		}
	}

	cnt, err := app.parent.getPlatform().NewContainer(app.Name, app.ID, app.InstallerMetadata.PlatformID, app.VolumeID, app.InstallerMetadata.PersistancePath, app.PublicPorts, app.InstallerParams)
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

func (app *App) getOrCreateContainer() (core.PlatformRuntimeUnit, error) {
	cnt, err := app.parent.getPlatform().GetDockerContainer(app.ContainerID)
	if err != nil {
		if util.IsErrorType(err, core.ErrContainerNotFound) {
			cnt, err := app.createContainer()
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

	cnt, err := app.parent.getPlatform().GetDockerContainer(app.ContainerID)
	if err != nil {
		if util.IsErrorType(err, core.ErrContainerNotFound) {
			log.Warnf("Application %s(%s) has no container: %s", app.Name, app.ID, err.Error())
			app.Status = statusStopped
			return
		}
		app.Status = statusUnknown
		return
	}

	app.Status = containerToAppStatus(cnt.GetStatus(), cnt.GetExitCode())
}

// StartAsync asynchronously starts an application and returns a task
func (app *App) StartAsync() core.Task {
	return app.parent.getTaskManager().New(&StartAppTask{app: app})
}

// Start starts an application
func (app *App) Start() error {
	log.Info("Starting application ", app.Name, "[", app.ID, "]")

	cnt, err := app.getOrCreateContainer()
	if err != nil {
		app.SetStatus(statusFailed)
		return errors.Wrap(err, "Failed to start application "+app.ID)
	}

	err = cnt.Start()
	if err != nil {
		app.SetStatus(statusFailed)
		return errors.Wrap(err, "Failed to start application "+app.ID)
	}
	app.SetStatus(statusRunning)
	return nil
}

// StopAsync asynchronously stops an application and returns a task
func (app *App) StopAsync() core.Task {
	return app.parent.getTaskManager().New(&StopAppTask{app: app})
}

// Stop stops an application
func (app *App) Stop() error {
	log.Info("Stoping application ", app.Name, "[", app.ID, "]")

	cnt, err := app.parent.getPlatform().GetDockerContainer(app.ContainerID)
	if err != nil {
		if util.IsErrorType(err, core.ErrContainerNotFound) == false {
			app.SetStatus(statusUnknown)
			return err
		}
		log.Warnf("Application %s(%s) has no container to stop", app.Name, app.ID)
		app.SetStatus(statusStopped)
		return nil
	}

	err = cnt.Stop()
	if err != nil {
		app.SetStatus(statusUnknown)
		return errors.Wrapf(err, "Can't stop application %s(%s)", app.Name, app.ID)
	}
	app.SetStatus(statusStopped)
	return nil
}

// remove App removes an application container
func (app *App) remove() error {
	log.Debug("Removing application ", app.Name, "[", app.ID, "]")

	cnt, err := app.parent.getPlatform().GetDockerContainer(app.ContainerID)
	if err != nil {
		if util.IsErrorType(err, core.ErrContainerNotFound) == false {
			return errors.Wrapf(err, "Failed to remove application %s(%s)", app.Name, app.ID)
		}
		log.Warnf("Application %s(%s) has no container to remove", app.Name, app.ID)
	} else {
		err := cnt.Remove()
		if err != nil {
			return errors.Wrapf(err, "Failed to remove application %s(%s)", app.Name, app.ID)
		}
	}

	if app.VolumeID != "" {
		err := app.parent.getPlatform().RemoveVolume(app.VolumeID)
		if err != nil {
			return errors.Wrapf(err, "Failed to remove application %s(%s)", app.Name, app.ID)
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
	cnt, err := app.parent.getPlatform().GetDockerContainer(id)
	if err != nil {
		return errors.Wrap(err, "Failed to replace container for app "+app.ID)
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
func (app *App) SetMsgQ(msgq *WSConnection) {
	app.access.Lock()
	app.msgq = msgq
	id := app.ID
	app.access.Unlock()
	log.Debug("New WS connection established for app ", id)
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
	log.Debug("Closing WS connection for app ", id)
	msgq.Close <- true
}

// SendMsg sends a message to the app via the active WS connection. Returns error if no WS connection is active
func (app *App) SendMsg(msg interface{}) error {
	app.access.Lock()
	msgq := app.msgq
	id := app.ID
	app.access.Unlock()
	if msgq == nil {
		return errors.Errorf("Application %s does not have a WS connection open", id)
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
		return &resource.Resource{}, err
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
			return err
		}
		app.access.Lock()
		app.Resources = util.RemoveStringFromSlice(app.Resources, index)
		app.access.Unlock()
		app.Save()

		return nil
	}

	return errors.New("Resource " + resourceID + " not owned by application " + app.ID)
}

// GetResources retrieves all the resources that belong to an application
func (app *App) GetResources() map[string]core.Resource {
	resources := make(map[string]core.Resource)
	for _, rscid := range app.Resources {
		rsc, err := app.parent.getResourceManager().Get(rscid)
		if err != nil {
			log.Error(err)
			continue
		}
		resources[rscid] = rsc
	}
	return resources
}

// GetResource returns resource with provided ID, if it belongs to this app
func (app *App) GetResource(resourceID string) core.Resource {
	for _, rscid := range app.Resources {
		if rscid == resourceID {
			rsc, err := app.parent.getResourceManager().Get(rscid)
			if err != nil {
				log.Error(err)
			}
			return rsc
		}
	}
	return nil
}

// ValidateCapability implements the capability checker interface
func (app *App) ValidateCapability(cap *capability.Capability) error {
	for _, appcap := range app.Capabilities {
		if capability.Validate(cap, appcap) {
			return nil
		}
	}
	return errors.New("Method capability " + cap.Name + " not satisfied by application " + app.ID)
}

// Provides returns true if the application is a provider for a specific type of resource
func (app *App) Provides(rscType string) bool {
	if prov, _ := util.StringInSlice(rscType, app.InstallerMetadata.Provides); prov {
		return true
	}
	return false
}
