package daemon

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"
)

// Defines structure for config parameters
// specific to each application
const (
	Running = "Running"
	Stopped = "Stopped"
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
	ImageID         string            `json:"imageid"`
	Status          string            `json:"status"`
	Command         string            `json:"command"`
	Actions         []AppAction       `json:"actions"`
	IP              string            `json:"ip"`
	PublicPorts     string            `json:"publicports"`
	InstallerParams map[string]string `json:"installer-params"`
}

// Apps maintains a map of all the applications
var Apps map[string]*App

func combineEnv(params map[string]string) []string {
	var env []string
	for id, val := range params {
		env = append(env, id+"="+val)
	}
	return env
}

// CreateApp takes an image and creates an application, without starting it
func CreateApp(imageID string, name string, commandstr string, ports string, installerParams map[string]string) (App, error) {
	client := Gconfig.DockerClient

	log.Debugf("Creating container: %s %s {%s}", imageID, name, commandstr)
	command := strslice.StrSlice{}
	if len(commandstr) > 0 {
		command = strings.Split(commandstr, " ")
	}

	var publicports []string
	for _, v := range strings.Split(ports, ",") {
		publicports = append(publicports, "0.0.0.0:"+v+":"+v+"/tcp")
	}
	exposedPorts, portBindings, _ := nat.ParsePortSpecs(publicports)

	containerConfig := &container.Config{
		Image:        imageID,
		Cmd:          command,
		ExposedPorts: exposedPorts,
		Env:          combineEnv(installerParams),
	}
	hostConfig := &container.HostConfig{
		Links:        []string{"protos"},
		PortBindings: portBindings,
	}

	cnt, err := client.ContainerCreate(context.Background(), containerConfig, hostConfig, nil, name)
	if err != nil {
		return App{}, err
	}
	log.Debug("Created application ", name, "[", cnt.ID, "]")

	container, err := client.ContainerInspect(context.Background(), cnt.ID)
	if err != nil {
		log.Error(err)
		return App{}, err
	}

	app := App{Name: container.Name, ID: container.ID, ImageID: container.Image, Status: container.State.Status}

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

// Start starts an application
func (app *App) Start() error {
	log.Info("Starting application ", app.Name, "[", app.ID, "]")
	client := Gconfig.DockerClient

	err := client.ContainerStart(context.Background(), app.ID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}
	return nil
}

// Stop stops an application
func (app *App) Stop() error {
	log.Info("Stoping application ", app.Name, "[", app.ID, "]")
	client := Gconfig.DockerClient

	stopTimeout := time.Duration(10) * time.Second
	err := client.ContainerStop(context.Background(), app.ID, &stopTimeout)
	if err != nil {
		return err
	}
	return nil
}

// ReadApp reads a fresh copy of the application
func ReadApp(appID string) (App, error) {
	log.Info("Reading application ", appID)
	client := Gconfig.DockerClient

	container, err := client.ContainerInspect(context.Background(), appID)
	if err != nil {
		return App{}, err
	}

	app := App{Name: container.Name, ID: container.ID, ImageID: container.Image, Status: container.State.Status, Command: strings.Join(container.Config.Cmd, " ")}
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
	client := Gconfig.DockerClient

	err := client.ContainerRemove(context.Background(), app.ID, types.ContainerRemoveOptions{})
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}

// LoadApps connects to the Docker daemon and refreshes the internal application list
func LoadApps() {
	client := Gconfig.DockerClient
	apps := make(map[string]*App)
	log.Info("Retrieving applications")

	containers, err := client.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		log.Fatal(err)
	}

	//FixMe: names for the protos container has werid names
	for _, container := range containers {
		app := App{Name: strings.Replace(container.Names[0], "/", "", 1), ID: container.ID, ImageID: container.ImageID, Status: container.State, Command: container.Command, IP: container.NetworkSettings.Networks["bridge"].IPAddress}
		apps[app.ID] = &app
	}

	Apps = apps
}

// GetApps refreshes the application list and returns it
func GetApps() map[string]*App {
	LoadApps()
	return Apps
}
