package daemon

import (
	"errors"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
)

// Defines structure for config parameters
// specific to each application
const (
	Running = "Running"
	Stopped = "Stopped"
)

type AppConfig struct {
	Description string
	Image       string
	Ports       map[string]string
	Data        string
}

type AppStatus struct {
	Running bool
}

type App struct {
	Name       string
	ImageID    string
	Containers []string
	Status     AppStatus
	Config     AppConfig
}

type Config struct {
	DataPath       string
	AppsPath       string
	Port           int
	DockerEndpoint string
	DockerClient   *docker.Client
}

var Gconfig Config
var Apps map[string]*App

func (app *App) LoadCfg() {
	log.Println("Reading config for [", app.Name, "]")
	filename, _ := filepath.Abs(fmt.Sprintf("%s/%s/app.yaml", Gconfig.AppsPath, app.Name))
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(yamlFile, &app.Config)
	if err != nil {
		log.Fatal(err)
	}
}

func LoadCfg(config_file string) Config {
	log.Println("Reading main config [", config_file, "]")
	filename, _ := filepath.Abs(config_file)
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		log.Fatal(err)
	}

	var config Config

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatal(err)
	}

	Gconfig = config

	log.Println("Connecting to the docker daemon")
	client, err := docker.NewClient(Gconfig.DockerEndpoint)
	if err != nil {
		log.Fatal(err)
	}
	Gconfig.DockerClient = client

	LoadApps()

	return config
}

func (app *App) Start() {
	log.Println("Starting application [", app.Name, "]")
	client := Gconfig.DockerClient

	app.LoadCfg()

	//Configure volumes
	volumes := make(map[string]struct{})
	var tmp struct{}
	volumes[app.Config.Data] = tmp

	// Create container
	config := docker.Config{Image: app.Config.Image, Volumes: volumes}
	create_options := docker.CreateContainerOptions{Name: "egor." + app.Name, Config: &config}
	container, err := client.CreateContainer(create_options)
	if err != nil {
		log.Println("Could not create container")
		log.Fatal(err)
	}

	// Configure ports
	portsWrapper := make(map[docker.Port][]docker.PortBinding)
	for key, value := range app.Config.Ports {
		ports := []docker.PortBinding{docker.PortBinding{HostIP: "0.0.0.0", HostPort: value}}
		port_host := docker.Port(key)
		portsWrapper[port_host] = ports
	}

	// Bind volumes
	binds := []string{fmt.Sprintf("%s/%s:%s:rw", Gconfig.DataPath, app.Name, app.Config.Data)}

	// Start container
	host_config := docker.HostConfig{PortBindings: portsWrapper, Binds: binds}
	err2 := client.StartContainer(container.ID, &host_config)
	if err2 != nil {
		log.Println("Could not start container")
		log.Fatal(err2)
	}
	container_instance, _ := client.InspectContainer(container.ID)
	app.Status.Running = container_instance.State.Running

}

func (app *App) Stop() {
	log.Println("Stopping application [", app.Name, "]")
	client := Gconfig.DockerClient

	container_name := "egor." + app.Name
	err := Gconfig.DockerClient.StopContainer(container_name, 3)
	if err != nil {
		log.Println("Could not stop application")
		log.Fatal(err)
	}

	remove_options := docker.RemoveContainerOptions{ID: "egor." + app.Name}
	err = client.RemoveContainer(remove_options)
	if err != nil {
		log.Fatal(err)
	}
	app.Status.Running = false
}

func tagtoname(tag string) (string, error) {
	name := strings.Split(tag, ":")
	if strings.Contains(name[0], "/") {
		repo := strings.Split(name[0], "/")
		if strings.Contains(repo[0], "egor") {
			return repo[1], nil
		}
	}
	return "", errors.New("Tag is not related to egor")
}

func LoadApps() {
	client := Gconfig.DockerClient
	apps := make(map[string]*App)
	log.Println("Retrieving applications")

	images, err := client.ListImages(true)
	if err != nil {
		log.Fatal(err)
	}

	for _, image := range images {
		for _, tag := range image.RepoTags {
			appname, err := tagtoname(tag)
			if err != nil {
				continue
			}
			log.Println("Found image [", appname, "]")
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
		appname, err := tagtoname(container.Image)
		if err != nil {
			continue
		}
		apps[appname].Containers = append(apps[appname].Containers, container.ID)
		log.Println("Found container", container.ID, "for", appname)
		container, _ := client.InspectContainer(container.ID)
		apps[appname].Status.Running = container.State.Running
	}
	Apps = apps
}

func GetApps() map[string]*App {
	LoadApps()
	return Apps
}

func GetApp(name string) *App {
	LoadApps()
	log.Println("Retrieving data for [", name, "]")
	return Apps[name]
}
