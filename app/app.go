package app

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/protosio/protos/config"
	"github.com/protosio/protos/installer"
	"github.com/protosio/protos/meta"
	"github.com/protosio/protos/task"

	"github.com/protosio/protos/capability"
	"github.com/protosio/protos/platform"
	"github.com/protosio/protos/resource"
	"github.com/protosio/protos/util"

	"github.com/rs/xid"
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
		return task.Task{}, errors.New("Action not supported")
	}
}

// AddTask adds a task owned by the applications
func (app *App) AddTask(id string) {
	app.Tasks = append(app.Tasks, id)
	app.Save()
}

// Save - sends update to the app manager which persists application data to database
func (app *App) Save() {
	addAppQueue <- *app
}

// reateContainer create the underlying Docker container
func (app *App) createContainer() (platform.RuntimeUnit, error) {
	var volume *platform.DockerVolume
	if app.InstallerMetadata.PersistancePath != "" {
		volume, err := platform.GetOrCreateDockerVolume(app.VolumeID, app.InstallerMetadata.PersistancePath)
		if err != nil {
			return nil, errors.New("Failed to create volume for app " + app.ID + ":" + err.Error())
		}
		app.VolumeID = volume.ID
	}

	cnt, err := platform.NewDockerContainer(app.Name, app.ID, app.InstallerMetadata.PlatformID, volume, app.PublicPorts, app.InstallerParams)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create container")
	}
	app.ContainerID = cnt.GetID()
	app.IP = cnt.GetIP()
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
func (app App) StartAsync() task.Task {
	return task.New(StartAppTask{app: app})
}

// Start starts an application
func (app *App) Start() error {
	log.Info("Starting application ", app.Name, "[", app.ID, "]")

	cnt, err := app.getOrCreateContainer()
	if err != nil {
		app.Status = statusFailed
		return errors.Wrap(err, "Failed to start application "+app.ID)
	}

	err = cnt.Start()
	if err != nil {
		app.Status = statusFailed
		return errors.Wrap(err, "Failed to start application "+app.ID)
	}
	app.Status = statusRunning
	app.Save()
	return nil
}

// StopAsync asynchronously stops an application and returns a task
func (app App) StopAsync() task.Task {
	return task.New(StopAppTask{app: app})
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
	app.Status = statusStopped
	app.Save()
	return nil
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

	ra := removeAppReq{id: app.ID, resp: make(chan error)}
	removeAppQueue <- ra
	err = <-ra.resp
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
	app.Resources = append(app.Resources, rsc.ID)
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
		app.Resources = util.RemoveStringFromSlice(app.Resources, index)
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
func (app App) ValidateCapability(cap *capability.Capability) error {
	for _, appcap := range app.Capabilities {
		if capability.Validate(cap, appcap) {
			return nil
		}
	}
	return errors.New("Method capability " + cap.Name + " not satisfied by application " + app.ID)
}

//
// Package public methods
//

// CreateAsync creates, runs and returns a task of type CreateAppTask
func CreateAsync(installerID string, installerVersion string, appName string, installerParams map[string]string, startOnCreation bool) task.Task {
	taskType := CreateAppTask{
		InstallerID:      installerID,
		InstallerVersion: installerVersion,
		AppName:          appName,
		InstallerParams:  installerParams,
		StartOnCreation:  startOnCreation,
	}
	return task.New(taskType)
}

// Create takes an image and creates an application, without starting it
func Create(installerID string, installerVersion string, name string, installerParams map[string]string, installerMetadata installer.Metadata, taskID string) (*App, error) {

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
	app = &App{Name: name, ID: guid.String(), InstallerID: installerID, InstallerVersion: installerVersion,
		PublicPorts: installerMetadata.PublicPorts, InstallerParams: installerParams,
		InstallerMetadata: installerMetadata, Tasks: []string{taskID}, Status: statusCreating}

	app.Capabilities = createCapabilities(installerMetadata.Capabilities)
	if app.ValidateCapability(capability.PublicDNS) == nil {
		rsc, err := resource.Create(resource.DNS, &resource.DNSResource{Host: app.Name, Value: meta.GetPublicIP(), Type: "A", TTL: 300})
		if err != nil {
			return app, err
		}
		app.Resources = append(app.Resources, rsc.ID)
	}
	app.Save()

	log.Debug("Created application ", name, "[", guid.String(), "]")
	return app, nil
}

// Read reads a fresh copy of the application
func Read(id string) (App, error) {
	log.Info("Reading application ", id)

	ra := readAppReq{id: id, resp: make(chan readAppResp)}
	readAppQueue <- ra
	resp := <-ra.resp
	app := &resp.app
	app.enrichAppData()
	return *app, resp.err
}

// ReadByIP searches and returns an application based on it's IP address
// ToDo: refresh IP data for all applications?
func ReadByIP(appIP string) (App, error) {
	log.Debug("Reading application with IP ", appIP)

	apps := GetAll()
	for _, app := range apps {
		if app.IP == appIP {
			log.Debug("Found application '", app.Name, "' for IP ", appIP)
			return app, nil
		}
	}
	return App{}, errors.New("Could not find any application with IP " + appIP)

}

// GetAll returns all applications
func GetAll() map[string]App {
	resp := make(chan map[string]App)
	readAllQueue <- resp
	return <-resp
}
