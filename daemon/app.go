package daemon

import (
	"errors"
	"protos/capability"
	"protos/database"
	"protos/platform"

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
	Capabilities    []string             `json:"tokens"`
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

func createDefaultCapabilities() []string {
	caps := []string{}
	caps = append(caps, capability.RC.Name)
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
	app.Capabilities = createDefaultCapabilities()
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
func ReadApp(appID string) (App, error) {
	log.Info("Reading application ", appID)

	var app App
	err := database.One("ID", appID, &app)
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
	Apps[app.ID] = &app
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

	for _, app := range apps {
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
		Apps[app.ID] = &app
	}

}

// GetApps refreshes the application list and returns it
func GetApps() map[string]*App {
	LoadApps()
	return Apps
}
