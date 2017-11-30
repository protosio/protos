package daemon

import (
	"context"
	"errors"
	"protos/database"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
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
	Name            string            `json:"name"`
	ID              string            `json:"id"`
	InstallerID     string            `json:"installer-id"`
	ContainerID     string            `json:"container-id"`
	Status          string            `json:"status"`
	Actions         []AppAction       `json:"actions"`
	IP              string            `json:"ip"`
	PublicPorts     string            `json:"publicports"`
	InstallerParams map[string]string `json:"installer-params"`
}

// Apps maintains a map of all the applications
var Apps = make(map[string]*App)

// combineEnv takes a map of environment variables and transforms them into a list of environment variables
func combineEnv(params map[string]string) []string {
	var env []string
	for id, val := range params {
		env = append(env, id+"="+val)
	}
	return env
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

	var publicports []string
	for _, v := range strings.Split(ports, ",") {
		publicports = append(publicports, "0.0.0.0:"+v+":"+v+"/tcp")
	}
	exposedPorts, portBindings, err := nat.ParsePortSpecs(publicports)
	if err != nil {
		return App{}, err
	}

	containerConfig := &container.Config{
		Image:        installerID,
		ExposedPorts: exposedPorts,
		Env:          combineEnv(installerParams),
	}
	hostConfig := &container.HostConfig{
		Links:        []string{"protos"},
		PortBindings: portBindings,
	}

	cnt, err := dockerClient.ContainerCreate(context.Background(), containerConfig, hostConfig, nil, name)
	if err != nil {
		return App{}, err
	}
	log.Debug("Created application ", name, "[", guid.String(), "]")

	app := App{Name: name, ID: guid.String(), InstallerID: installerID, ContainerID: cnt.ID}
	err = database.Save(&app)
	if err != nil {
		return App{}, err
	}

	return app, nil

}

// AddAction performs an action on an application
func (app *App) AddAction(action AppAction) error {
	log.Info("Performing action [", action.Name, "] on application ", app.Name, "[", app.ID, "]")

	switch action.Name {
	case "start":
		app.Start()
	case "stop":
		app.Stop()
	default:
		return errors.New("Action not supported")
	}
	return nil
}

// Save - persists application data to database
func (app *App) Save() error {
	err := database.Save(app)
	if err != nil {
		return err
	}
	return nil
}

// Start starts an application
func (app *App) Start() error {
	log.Info("Starting application ", app.Name, "[", app.ID, "]")

	err := dockerClient.ContainerStart(context.Background(), app.ID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}
	return nil
}

// Stop stops an application
func (app *App) Stop() error {
	log.Info("Stoping application ", app.Name, "[", app.ID, "]")

	stopTimeout := time.Duration(10) * time.Second
	err := dockerClient.ContainerStop(context.Background(), app.ID, &stopTimeout)
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

	container, err := dockerClient.ContainerInspect(context.Background(), app.ContainerID)
	if err != nil {
		app.Status = statusMissingContainer
		return app, errors.New("Error retrieving container for application " + app.ID + ": " + err.Error())
	}

	app.Status = container.State.Status
	app.IP = container.NetworkSettings.Networks["bridge"].IPAddress
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

	err := dockerClient.ContainerRemove(context.Background(), app.ID, types.ContainerRemoveOptions{})
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

// LoadApps connects to the Docker daemon and refreshes the internal application list
func LoadApps() {
	var apps []App
	log.Info("Retrieving applications")

	err := database.All(&apps)
	if err != nil {
		log.Error("Could not retrieve applications from the database: ", err)
		return
	}

	containers, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		log.Fatal(err)
	}

	for _, app := range apps {
		matched := false
		for _, container := range containers {
			if container.ID == app.ContainerID {
				app.Status = container.State
				matched = true
			}
		}
		if matched != true {
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
