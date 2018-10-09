package app

import (
	"encoding/gob"
	"fmt"

	"github.com/pkg/errors"

	"github.com/protosio/protos/installer"
	"github.com/protosio/protos/meta"

	"github.com/protosio/protos/capability"
	"github.com/protosio/protos/database"
	"github.com/protosio/protos/platform"
	"github.com/protosio/protos/resource"
	"github.com/protosio/protos/util"

	"github.com/rs/xid"
)

var log = util.GetLogger("app")

// Defines structure for config parameters
// specific to each application
const (
	Running                = "Running"
	Stopped                = "Stopped"
	appBucket              = "app"
	statusMissingContainer = "missing container"
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
	Name              string               `json:"name"`
	ID                string               `json:"id"`
	InstallerID       string               `json:"installer-id"`
	InstallerVersion  string               `json:"installer-version"`
	InstallerMetadata installer.Metadata   `json:"-"`
	ContainerID       string               `json:"container-id"`
	VolumeID          string               `json:"volumeid"`
	Rtu               platform.RuntimeUnit `json:"-"`
	Status            string               `json:"status"`
	Actions           []Action             `json:"actions"`
	IP                string               `json:"ip"`
	PublicPorts       []util.Port          `json:"publicports"`
	InstallerParams   map[string]string    `json:"installer-params"`
	Capabilities      []string             `json:"capabilities"`
	Tasks             []string             `json:"tasks"`
	Resources         []string             `json:"resources"`
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

func createCapabilities(installerCapabilities []*capability.Capability) []string {
	caps := []string{}
	for _, cap := range installerCapabilities {
		caps = append(caps, cap.Name)
	}
	return caps
}

// ToDo: do app refresh caching in the platform code
func refreshAppsPlatform(apps map[string]App) map[string]App {
	for _, app := range apps {
		tmp := &app
		tmp.RefreshPlatform()
		apps[tmp.ID] = *tmp
	}
	return apps
}

//
// Methods for application instance
//

// AddAction performs an action on an application
func (app *App) AddAction(action Action) error {
	log.Info("Performing action [", action.Name, "] on application ", app.Name, "[", app.ID, "]")

	switch action.Name {
	case "start":
		return app.Start()
	case "stop":
		return app.Stop()
	default:
		return errors.New("Action not supported")
	}
}

// Save - persists application data to database
func (app *App) Save() {
	err := database.Save(app)
	if err != nil {
		log.Panic(errors.Wrap(err, "Could not save app to database"))
	}
}

// reateContainer create the underlying Docker container
func (app *App) createContainer() error {
	var volume *platform.DockerVolume
	if app.InstallerMetadata.PersistancePath != "" {
		volume, err := platform.GetOrCreateDockerVolume(app.VolumeID, app.InstallerMetadata.PersistancePath)
		if err != nil {
			return errors.New("Failed to create volume for app " + app.ID + ":" + err.Error())
		}
		app.VolumeID = volume.ID
	}

	cnt, err := platform.NewDockerContainer(app.Name, app.ID, app.InstallerMetadata.PlatformID, volume, app.PublicPorts, app.InstallerParams)
	if err != nil {
		return errors.Wrap(err, "Failed to create container")
	}
	app.Rtu = cnt
	app.ContainerID = app.Rtu.GetID()
	app.IP = app.Rtu.GetIP()
	app.Status = app.Rtu.GetStatus()
	app.Save()
	return nil
}

func (app *App) containerMissing() bool {
	if app.Rtu != nil && app.Rtu.Update() == nil {
		return false
	}
	return true
}

// RefreshPlatform updates the information about the underlying application container
func (app *App) RefreshPlatform() {
	if app.Rtu == nil {
		cnt, err := platform.GetDockerContainer(app.ContainerID)
		if err != nil {
			log.Warnf("Application %s(%s) has no container: %s", app.ID, app.Name, err.Error())
			app.Status = statusMissingContainer
			return
		}
		app.Rtu = cnt
	} else {
		err := app.Rtu.Update()
		if err != nil {
			log.Warnf("Application %s(%s) has no container: %s", app.ID, app.Name, err.Error())
			app.Status = statusMissingContainer
			return
		}
	}
	app.IP = app.Rtu.GetIP()
	app.Status = app.Rtu.GetStatus()
}

// Start starts an application
func (app *App) Start() error {
	log.Info("Starting application ", app.Name, "[", app.ID, "]")

	if app.containerMissing() {
		err := app.createContainer()
		if err != nil {
			return err
		}
	}

	err := app.Rtu.Start()
	if err != nil {
		return errors.Wrap(err, "Failed to start application "+app.ID)
	}
	return nil
}

// Stop stops an application
func (app *App) Stop() error {
	log.Info("Stoping application ", app.Name, "[", app.ID, "]")

	if app.containerMissing() {
		return errors.New("Can't stop application " + app.ID + ". Container is missing.")
	}

	err := app.Rtu.Stop()
	if err != nil {
		return err
	}
	return nil
}

// Remove App removes an application container
func (app *App) Remove() error {
	log.Info("Removing application ", app.Name, "[", app.ID, "]")

	if app.containerMissing() {
		log.Warn("App ", app.ID, " does not have a container.")
	} else {
		err := app.Rtu.Remove()
		if err != nil {
			return err
		}
	}

	if app.VolumeID != "" {
		err := platform.RemoveDockerVolume(app.VolumeID)
		if err != nil {
			log.Error(err)
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

	err := database.Remove(app)
	if err != nil {
		return err
	}

	ra := removeAppReq{id: app.ID, resp: make(chan error)}
	removeAppQueue <- ra
	err = <-ra.resp
	if err != nil {
		return err
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

// Create takes an image and creates an application, without starting it
func Create(installerID string, installerVersion string, name string, installerParams map[string]string, installerMetadata installer.Metadata) (App, error) {

	var app App
	if name == "" {
		return app, fmt.Errorf("Application name cannot be empty")
	}

	err := validateInstallerParams(installerParams, installerMetadata.Params)
	if err != nil {
		return app, err
	}

	guid := xid.New()
	log.Debugf("Creating application %s(%s), based on installer %s", guid.String(), name, installerID)
	app = App{Name: name, ID: guid.String(), InstallerID: installerID, InstallerVersion: installerVersion, PublicPorts: installerMetadata.PublicPorts, InstallerParams: installerParams, InstallerMetadata: installerMetadata}

	app.Capabilities = createCapabilities(installerMetadata.Capabilities)
	if app.ValidateCapability(capability.PublicDNS) == nil {
		rsc, err := resource.Create(resource.DNS, &resource.DNSResource{Host: app.Name, Value: meta.GetPublicIP(), Type: "A", TTL: 300})
		if err != nil {
			return app, err
		}
		app.Resources = append(app.Resources, rsc.ID)
	}
	app.Save()
	addAppQueue <- app

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
	app.RefreshPlatform()
	return *app, resp.err
}

// ReadByIP searches and returns an application based on it's IP address
// ToDo: refresh IP data for all applications?
func ReadByIP(appIP string) (App, error) {
	log.Debug("Reading application with IP ", appIP)

	apps := GetApps()
	for _, app := range apps {
		if app.IP == appIP {
			log.Debug("Found application '", app.Name, "' for IP ", appIP)
			return app, nil
		}
	}
	return App{}, errors.New("Could not find any application with IP " + appIP)

}

// LoadAppsDB connects to the Docker daemon and refreshes the internal application list
func LoadAppsDB() {
	log.Info("Retrieving applications from DB")
	gob.Register(&App{})
	gob.Register(&platform.DockerContainer{})

	apps := []App{}
	err := database.All(&apps)
	if err != nil {
		log.Error("Could not retrieve applications from the database: ", err)
		return
	}

	for _, app := range apps {
		addAppQueue <- app
	}
}

// GetApps refreshes the application list and returns it
func GetApps() map[string]App {
	resp := make(chan map[string]App)
	readAllQueue <- resp
	apps := refreshAppsPlatform(<-resp)
	return apps
}