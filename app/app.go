package app

import (
	"sync"

	"github.com/pkg/errors"

	"github.com/protosio/protos/config"
	"github.com/protosio/protos/installer"
	"github.com/protosio/protos/task"

	"github.com/protosio/protos/capability"
	"github.com/protosio/protos/platform"
	"github.com/protosio/protos/resource"
	"github.com/protosio/protos/util"
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

// Config the application config
type Config struct {
	Description string
	Image       string
	Ports       map[string]string
	Data        string
}

// Action represents an action that can be executed on
// an application
type Action struct {
	Name string
}

// App represents the application state
type App struct {
	access *sync.Mutex

	// Public members
	Name              string             `json:"name"`
	ID                string             `json:"id"`
	InstallerID       string             `json:"installer-id"`
	InstallerVersion  string             `json:"installer-version"`
	InstallerMetadata installer.Metadata `json:"-"`
	ContainerID       string             `json:"container-id"`
	VolumeID          string             `json:"volumeid"`
	Status            string             `json:"status"`
	Actions           []Action           `json:"actions"`
	IP                string             `json:"ip"`
	PublicPorts       []util.Port        `json:"publicports"`
	InstallerParams   map[string]string  `json:"installer-params"`
	Capabilities      []string           `json:"capabilities"`
	Resources         []string           `json:"resources"`
	Tasks             []string           `json:"-"`
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

// SetStatus is used to set the status of an application
func (app *App) SetStatus(status string) {
	app.access.Lock()
	app.Status = status
	app.access.Unlock()
	app.Save()
}

// AddAction performs an action on an application
func (app *App) AddAction(action Action) (task.Task, error) {
	log.Info("Performing action [", action.Name, "] on application ", app.Name, "[", app.ID, "]")

	switch action.Name {
	case "start":
		tsk := app.StartAsync()
		return tsk, nil
	case "stop":
		tsk := app.StopAsync()
		return tsk, nil
	default:
		return nil, errors.New("Action not supported")
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
	saveApp(app)
}

// reateContainer create the underlying Docker container
func (app *App) createContainer() (platform.RuntimeUnit, error) {
	var volume *platform.DockerVolume
	var err error
	if app.InstallerMetadata.PersistancePath != "" {
		volume, err = platform.GetOrCreateDockerVolume(app.VolumeID, app.InstallerMetadata.PersistancePath)
		if err != nil {
			return nil, errors.New("Failed to create volume for app " + app.ID + ":" + err.Error())
		}
	}

	cnt, err := platform.NewDockerContainer(app.Name, app.ID, app.InstallerMetadata.PlatformID, volume, app.PublicPorts, app.InstallerParams)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create container")
	}
	app.access.Lock()
	if volume != nil {
		app.VolumeID = volume.ID
	}
	app.ContainerID = cnt.GetID()
	app.IP = cnt.GetIP()
	app.access.Unlock()
	app.Save()
	return cnt, nil
}

func (app *App) getOrCreateContainer() (platform.RuntimeUnit, error) {
	cnt, err := platform.GetDockerContainer(app.ContainerID)
	if err != nil {
		if util.IsErrorType(err, platform.ErrDockerContainerNotFound) {
			cnt, err := app.createContainer()
			if err != nil {
				return nil, err
			}
			return cnt, nil
		}
		return nil, err
	}
	return cnt, nil
}

// enrichAppData updates the information about the underlying application
func (app *App) enrichAppData() {
	if app.Status == statusCreating || app.Status == statusFailed {
		// not refreshing the platform until the app creation process is done
		return
	}

	cnt, err := platform.GetDockerContainer(app.ContainerID)
	if err != nil {
		if util.IsErrorType(err, platform.ErrDockerContainerNotFound) {
			log.Warnf("Application %s(%s) has no container: %s", app.Name, app.ID, err.Error())
			app.Status = statusStopped
			return
		}
		app.Status = statusUnknown
		return
	}

	app.Status = containerToAppStatus(cnt.Status, cnt.ExitCode)
}

// StartAsync asynchronously starts an application and returns a task
func (app *App) StartAsync() task.Task {
	tsk := task.New(&StartAppTask{app: app})
	return tsk
}

// Start starts an application
func (app *App) Start() error {
	log.Info("Starting application ", app.Name, "[", app.ID, "]")

	cnt, err := app.getOrCreateContainer()
	if err != nil {
		app.access.Lock()
		app.Status = statusFailed
		app.access.Unlock()
		return errors.Wrap(err, "Failed to start application "+app.ID)
	}

	err = cnt.Start()
	if err != nil {
		app.access.Lock()
		app.Status = statusFailed
		app.access.Unlock()
		return errors.Wrap(err, "Failed to start application "+app.ID)
	}
	app.access.Lock()
	app.Status = statusRunning
	app.access.Unlock()
	app.Save()
	return nil
}

// StopAsync asynchronously stops an application and returns a task
func (app *App) StopAsync() task.Task {
	tsk := task.New(&StopAppTask{app: app})
	return tsk
}

// Stop stops an application
func (app *App) Stop() error {
	log.Info("Stoping application ", app.Name, "[", app.ID, "]")

	cnt, err := platform.GetDockerContainer(app.ContainerID)
	if err != nil {
		if util.IsErrorType(err, platform.ErrDockerContainerNotFound) == false {
			return err
		}
		log.Warnf("Application %s(%s) has no container to stop", app.Name, app.ID)
		return nil
	}

	err = cnt.Stop()
	if err != nil {
		return errors.Wrapf(err, "Can't stop application %s(%s)", app.Name, app.ID)
	}
	app.access.Lock()
	app.Status = statusStopped
	app.access.Unlock()
	app.Save()
	return nil
}

// RemoveAsync asynchronously removes an applications and returns a task
func (app *App) RemoveAsync() task.Task {
	tsk := task.New(&RemoveAppTask{app: app})
	return tsk
}

// Remove App removes an application container
func (app *App) Remove() error {
	log.Info("Removing application ", app.Name, "[", app.ID, "]")

	cnt, err := platform.GetDockerContainer(app.ContainerID)
	if err != nil {
		if util.IsErrorType(err, platform.ErrDockerContainerNotFound) == false {
			return err
		}
		log.Warnf("Application %s(%s) has no container to remove", app.Name, app.ID)
	} else {
		err := cnt.Remove()
		if err != nil {
			return err
		}
	}

	if app.VolumeID != "" {
		err := platform.RemoveDockerVolume(app.VolumeID)
		if err != nil {
			return errors.Wrapf(err, "Can't remove application %s(%s)", app.Name, app.ID)
		}
	}

	// Removing resources requested by this app
	for _, rscID := range app.Resources {
		rsc, err := resource.Get(rscID)
		if err != nil {
			log.Error(err)
			continue
		}
		err = rsc.Delete()
		if err != nil {
			log.Error(err)
			continue
		}
	}

	err = mapps.remove(app.ID)
	if err != nil {
		return errors.Wrapf(err, "Can't remove application %s(%s)", app.Name, app.ID)
	}

	return nil
}

// Resource related methods

//CreateResource adds a resource to the internal resources map.
func (app *App) CreateResource(appJSON []byte) (*resource.Resource, error) {

	rsc, err := resource.CreateFromJSON(appJSON)
	if err != nil {
		return &resource.Resource{}, err
	}
	app.access.Lock()
	app.Resources = append(app.Resources, rsc.ID)
	app.access.Unlock()
	app.Save()

	return rsc, nil
}

//DeleteResource deletes a resource
func (app *App) DeleteResource(resourceID string) error {
	if v, index := util.StringInSlice(resourceID, app.Resources); v {
		rsc, err := resource.Get(resourceID)
		if err != nil {
			return err
		}
		err = rsc.Delete()
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
func (app *App) GetResources() map[string]*resource.Resource {
	resources := make(map[string]*resource.Resource)
	for _, rscid := range app.Resources {
		rsc, err := resource.Get(rscid)
		if err != nil {
			log.Error(err)
			continue
		}
		resources[rscid] = rsc
	}
	return resources
}

// GetResource returns resource with provided ID, if it belongs to this app
func (app *App) GetResource(resourceID string) *resource.Resource {
	for _, rscid := range app.Resources {
		if rscid == resourceID {
			rsc, err := resource.Get(rscid)
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
