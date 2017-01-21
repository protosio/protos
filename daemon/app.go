package daemon

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/fsouza/go-dockerclient"
	"gopkg.in/yaml.v2"
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

// AppStatus is the application status
type AppStatus struct {
	Running bool
}

// App represents the application state
type App struct {
	Name       string
	ImageID    string
	Containers []string
	Status     AppStatus
	Config     AppConfig
}

// Apps maintains a map of all the applications
var Apps map[string]*App

func (app *App) loadCfg() {
	log.Info("Reading config for [", app.Name, "]")
	filename, _ := filepath.Abs(fmt.Sprintf("%s/%s/app.yaml", Gconfig.AppsPath, app.Name))
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		log.Warn(err)
		return
	}

	err = yaml.Unmarshal(yamlFile, &app.Config)
	if err != nil {
		log.Warn(err)
		return
	}
}

func (app *App) getImage() *docker.Image {
	log.Debug("Reading image ", app.ImageID)
	client := Gconfig.DockerClient
	image, err := client.InspectImage(app.ImageID)
	if err != nil {
		log.Error(err)
	}
	return image
}

// Start starts an application
func (app *App) Start() {
	log.Info("Starting application [", app.Name, "]")
	client := Gconfig.DockerClient

	app.loadCfg()
	image := app.getImage()
	if len(image.Config.Entrypoint) == 0 {
		log.Error("Image [", image.ID, "|", app.Name, "] has no entrypoint. Aborting")
		return
	}

	////Configure
	//volumes := make(map[string]struct{})
	//var tmp struct{}
	//volumes["/data"] = tmp

	// Create container
	config := docker.Config{Image: app.Name}
	createOptions := docker.CreateContainerOptions{Name: "protos." + app.Name, Config: &config}
	container, err := client.CreateContainer(createOptions)
	if err != nil {
		log.Error("Could not create container: ", err)
		return
	}
	log.Debug("Created container ", app.Name, container)

	//removeOptions := docker.RemoveContainerOptions{ID: container.ID, RemoveVolumes: true}
	//_ = client.RemoveContainer(removeOptions)

	//// Configure ports
	//portsWrapper := make(map[docker.Port][]docker.PortBinding)
	//for key, value := range app.Config.Ports {
	//	ports := []docker.PortBinding{docker.PortBinding{HostIP: "0.0.0.0", HostPort: value}}
	//	port_host := docker.Port(key)
	//	portsWrapper[port_host] = ports
	//}

	//// Bind volumes
	////binds := []string{fmt.Sprintf("%s/%s:%s:rw", Gconfig.DataPath, app.Name, app.Config.Data)}

	// Start container
	hostConfig := docker.HostConfig{} //PortBindings: portsWrapper} //, Binds: binds}
	err2 := client.StartContainer(container.ID, &hostConfig)
	if err2 != nil {
		log.Error("Could not start container: ", err2)
		return
	}
	log.Debug("Started container ", app.Name)

	containerInstance, _ := client.InspectContainer(container.ID)
	app.Status.Running = containerInstance.State.Running

}

// Stop stops an application
func (app *App) Stop() {
	log.Info("Stopping application [", app.Name, "]")
	client := Gconfig.DockerClient

	containerName := "protos." + app.Name
	err := Gconfig.DockerClient.StopContainer(containerName, 3)
	if err != nil {
		log.Error("Could not stop application. ", err)
	}

	removeOptions := docker.RemoveContainerOptions{ID: "protos." + app.Name}
	err = client.RemoveContainer(removeOptions)
	if err != nil {
		log.Error("Could not delete container. ", err)
	}
	app.Status.Running = false
}

func tagtoname(tag string, filter string) (string, string, error) {
	log.Debug("Working on [", tag, "]")
	fullName := strings.Split(tag, ":")
	name := fullName[0]
	var version string
	if len(fullName) > 1 {
		version = fullName[1]
	} else {
		version = ""
	}
	if len(filter) > 0 {
		if strings.Contains(name, filter) {
			repo := strings.Split(name, filter)
			return repo[1], version, nil
		}
		return "", "", errors.New("Tag is not related to protos")
	}
	return name, version, nil
}

// LoadApps connects to the Docker daemon and refreshes the internal application list
func LoadApps() {
	client := Gconfig.DockerClient
	apps := make(map[string]*App)
	log.Info("Retrieving applications")

	filters := make(map[string][]string)
	filters["dangling"] = []string{"false"}
	images, err := client.ListImages(docker.ListImagesOptions{All: false, Filters: filters})
	if err != nil {
		log.Fatal(err)
	}

	for _, image := range images {
		for _, tag := range image.RepoTags {
			appname, _, err := tagtoname(tag, "")
			if err != nil {
				continue
			}
			log.Debug("Found image [", appname, "]")
			app := App{Name: appname, ImageID: image.ID}
			apps[appname] = &app
		}
	}
	listcontaineroptions := docker.ListContainersOptions{All: true}
	containers, err := client.ListContainers(listcontaineroptions)
	if err != nil {
		log.Fatal(err)
	}
	for _, container := range containers {
		appname, _, err := tagtoname(container.Image, "")
		if err != nil {
			log.Error(err)
			continue
		}
		apps[appname].Containers = append(apps[appname].Containers, container.ID)
		log.Debug("Found container ", container.ID, " for ", appname)
		container, _ := client.InspectContainer(container.ID)
		apps[appname].Status.Running = container.State.Running
	}
	Apps = apps
}

// GetApps refreshes the application list and returns it
func GetApps() map[string]*App {
	LoadApps()
	return Apps
}

// GetApp refreshes the application list and retrieves a specific application
func GetApp(name string) *App {
	LoadApps()
	log.Info("Retrieving data for [", name, "]")
	return Apps[name]
}
