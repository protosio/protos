package daemon

import (
	"errors"

	"github.com/nustiueudinastea/protos/capability"
	"github.com/nustiueudinastea/protos/database"
	"github.com/nustiueudinastea/protos/platform"
	"github.com/nustiueudinastea/protos/resource"
	"github.com/nustiueudinastea/protos/util"

	"github.com/rs/xid"
)

// Defines structure for config parameters
// specific to each application
const (
	Running                = "Running"
	Stopped                = "Stopped"
	appBucket              = "app"
	statusMissingContainer = "missing container"
)

// AppConfig the application config
type AppConfig struct {
	Description string
	Image       string
	Ports       map[string]string
	Data        string
}

// AppAction represents an action that can be executed on
// an application
type AppAction struct {
	Name string
}

// App represents the application state
type App struct {
	Name            string               `json:"name"`
	ID              string               `json:"id"`
	InstallerID     string               `json:"installer-id"`
	ContainerID     string               `json:"container-id"`
	Rtu             platform.RuntimeUnit `json:"-"`
	Status          string               `json:"status"`
	Actions         []AppAction          `json:"actions"`
	IP              string               `json:"ip"`
	PublicPorts     string               `json:"publicports"`
	InstallerParams map[string]string    `json:"installer-params"`
	Capabilities    []string             `json:"capabilities"`
	Resources       []string             `json:"resources"`
	Provider        provider             `json:"-"`
}

// Apps maintains a map of all the applications
var Apps = make(map[string]*App)

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

// CreateApp takes an image and creates an application, without starting it
func CreateApp(installerID string, name string, ports string, installerParams map[string]string) (App, error) {

	installer, err := ReadInstaller(installerID)
	if err != nil {
		return App{}, err
	}

	err = validateInstallerParams(installerParams, installer.Metadata.Params)
	if err != nil {
		return App{}, err
	}

	guid := xid.New()
	log.Debugf("Creating application %s(%s), based on installer %s", guid.String(), name, installerID)
	app := App{Name: name, ID: guid.String(), InstallerID: installerID, PublicPorts: ports, InstallerParams: installerParams}
	err = app.createContainer()
	if err != nil {
		return App{}, err
	}
	app.Capabilities = createCapabilities(installer.Metadata.Capabilities)
	app.Save()

	log.Debug("Created application ", name, "[", guid.String(), "]")

	return app, nil

}

// AddAction performs an action on an application
func (app *App) AddAction(action AppAction) error {
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
func (app *App) Save() error {
	err := database.Save(app)
	if err != nil {
		return err
	}
	return nil
}

// reateContainer create the underlying Docker container
func (app *App) createContainer() error {
	cnt, err := platform.NewDockerContainer(app.Name, app.ID, app.InstallerID, app.PublicPorts, app.InstallerParams)
	if err != nil {
		return err
	}
	app.ContainerID = cnt.ID
	app.Status = cnt.Status
	app.IP = cnt.IP
	app.Rtu = cnt
	return app.Save()
}

func (app *App) containerMissing() bool {
	if app.Rtu != nil && app.Rtu.Update() == nil {
		return false
	}
	return true
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
		return err
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

// ReadApp reads a fresh copy of the application
func ReadApp(appID string) (*App, error) {
	log.Info("Reading application ", appID)

	var app *App
	if a, ok := Apps[appID]; ok {
		app = a
	} else {
		err := errors.New("Can't find app " + appID)
		log.Debug(err)
		return app, err
	}

	err := database.One("ID", appID, app)
	if err != nil {
		log.Debugf("Can't find app %s (%s)", appID, err)
		return app, err
	}

	cnt, err := platform.GetDockerContainer(app.ContainerID)
	if err != nil {
		app.Status = statusMissingContainer
		return app, nil
	}

	app.Status = cnt.Status
	app.IP = cnt.IP
	app.Rtu = cnt
	return app, nil
}

// ReadAppByIP searches and returns an application based on it's IP address
func ReadAppByIP(appIP string) (*App, error) {
	log.Debug("Reading application with IP ", appIP)
	LoadApps()

	for _, app := range Apps {
		if app.IP == appIP {
			log.Debug("Found application '", app.Name, "' for IP ", appIP)
			return app, nil
		}
	}
	return &App{}, errors.New("Could not find any application with IP " + appIP)

}

// Remove App removes an application container
func (app *App) Remove() error {
	log.Info("Removing application ", app.Name, "[", app.ID, "]")

	if app.containerMissing() {
		log.Warn("App ", app.ID, " does not have a container.")
	} else {
		app.Rtu.Remove()
	}

	err := database.Remove(app)
	if err != nil {
		return err
	}
	delete(Apps, app.ID)
	return nil
}

// LoadApps connects to the Docker daemon and refreshes the internal application list
func LoadApps() {
	apps := []App{}
	log.Info("Retrieving applications")

	err := database.All(&apps)
	if err != nil {
		log.Error("Could not retrieve applications from the database: ", err)
		return
	}

	cnts, err := platform.GetAllDockerContainers()
	if err != nil {
		log.Error("Could not retrieve containers from Docker: ", err)
		return
	}

	for idx, app := range apps {
		if cnt, ok := cnts[app.ContainerID]; ok {
			app.Status = cnt.Status
			app.IP = cnt.IP
		} else {
			log.Errorf("Application %s is missing container %s", app.ID, app.ContainerID)
			app.Status = statusMissingContainer
		}

		if app.Save() != nil {
			log.Panicf("Failed to persist app %s to db", app.ID)
		}
		Apps[app.ID] = &apps[idx]
	}

}

// GetApps refreshes the application list and returns it
func GetApps() map[string]*App {
	LoadApps()
	return Apps
}

//
// Resource related methods
//

//CreateResource adds a resource to the internal resources map.
func (app *App) CreateResource(appJSON []byte) (*resource.Resource, error) {

	rsc, err := resource.Create(appJSON)
	if err != nil {
		return &resource.Resource{}, err
	}
	app.Resources = append(app.Resources, rsc.ID)

	return rsc, nil
}

//DeleteResource deletes a resource
func (app *App) DeleteResource(resourceID string) error {
	if v, index := util.StringInSlice(resourceID, app.Resources); v {
		log.Info("Deleting resource " + resourceID + " belonging to application " + app.ID)
		err := resource.Delete(resourceID)
		if err != nil {
			return err
		}
		app.Resources = util.RemoveStringFromSlice(app.Resources, index)
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

//
// Provider related methods
//

type provider interface {
	TypeName() string
	GetResources() map[string]*resource.Resource
	GetResource(string) *resource.Resource
}

// SetProvider makes an application a resource provider
func (app *App) SetProvider(provider provider) {
	log.Debugf("Making application %s a provider for resource %s", app.ID, provider.TypeName())
	app.Provider = provider
}

// IsProvider checks if an application is a provider
func (app *App) IsProvider() bool {
	if app.Provider != nil {
		return true
	}
	return false
}
