package daemon

import (
	"context"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"

	"github.com/docker/docker/api/types/strslice"
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

// App represents the application state
type App struct {
	Name    string `json:"name"`
	ID      string `json:"id"`
	ImageID string `json:"imageid"`
	Status  string `json:"status"`
}

// Installer represents a Docker image
type Installer struct {
	Name string
	ID   string
}

// Apps maintains a map of all the applications
var Apps map[string]*App

// CreateApp takes an image and creates an application, without starting it
func CreateApp(imageID string, name string) (App, error) {
	client := Gconfig.DockerClient

	log.Debug("Creating container")
	cnt, err := client.ContainerCreate(context.Background(), &container.Config{Image: imageID, Cmd: strslice.StrSlice{"sleep", "600"}}, nil, nil, name)
	if err != nil {
		log.Error(err)
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

// Start starts an application
func (app *App) Start() error {
	log.Info("Starting application ", app.Name, "[", app.ID, "]")
	client := Gconfig.DockerClient

	err := client.ContainerStart(context.Background(), app.ID, types.ContainerStartOptions{})
	if err != nil {
		log.Error(err)
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
		log.Error(err)
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
		log.Error(err)
		return App{}, err
	}

	app := App{Name: container.Name, ID: container.ID, ImageID: container.Image, Status: container.State.Status}
	return app, nil
}

// Remove removes an application container
func (app *App) Remove() error {
	log.Info("Stoping application ", app.Name, "[", app.ID, "]")
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
		panic(err)
	}

	for _, container := range containers {
		app := App{Name: strings.Replace(container.Names[0], "/", "", 1), ID: container.ID, ImageID: container.ImageID, Status: container.State}
		apps[app.Name] = &app
	}

	Apps = apps
}

// GetApps refreshes the application list and returns it
func GetApps() map[string]*App {
	LoadApps()
	return Apps
}

// GetInstallers gets all the local images and returns them
func GetInstallers() []Installer {
	client := Gconfig.DockerClient
	var installers []Installer
	log.Info("Retrieving installers")
	images, err := client.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		log.Warn(err)
		return nil
	}

	for _, image := range images {
		var name string
		if len(image.RepoTags) > 0 {
			name = image.RepoTags[0]
		} else {
			name = "n/a"
		}
		installer := Installer{Name: name, ID: image.ID}
		installers = append(installers, installer)
	}

	return installers
}
