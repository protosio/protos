package daemon

import (
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
type AppConfig struct {
	Description string
	Image       string
	Ports       map[string]string
	Data        string
}

type App struct {
	Name   string
	Status string
}

type Config struct {
	DataPath       string
	AppsPath       string
	Port           int
	DockerEndpoint string
	DockerClient   *docker.Client
}

var Gconfig Config

func LoadAppCfg(app string) AppConfig {
	log.Println("Reading config for [", app, "]")
	filename, _ := filepath.Abs(fmt.Sprintf("%s/%s/app.yaml", Gconfig.AppsPath, app))
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		log.Fatal(err)
	}

	var config AppConfig

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatal(err)
	}

	return config
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

	return config
}

func StartApp(app string) {
	log.Println("Starting application [", app, "]")
	client := Gconfig.DockerClient

	app_config := LoadAppCfg(app)

	//Configure volumes
	volumes := make(map[string]struct{})
	var tmp struct{}
	volumes[app_config.Data] = tmp

	// Create container
	config := docker.Config{Image: app_config.Image, Volumes: volumes}
	create_options := docker.CreateContainerOptions{Name: "egor_" + app, Config: &config}
	container, err := client.CreateContainer(create_options)
	if err != nil {
		log.Println("Could not create container")
		log.Fatal(err)
	}

	// Configure ports
	portsWrapper := make(map[docker.Port][]docker.PortBinding)
	for key, value := range app_config.Ports {
		ports := []docker.PortBinding{docker.PortBinding{HostIP: "0.0.0.0", HostPort: value}}
		port_host := docker.Port(key)
		portsWrapper[port_host] = ports
	}

	// Bind volumes
	binds := []string{fmt.Sprintf("%s/%s:%s:rw", Gconfig.DataPath, app, app_config.Data)}

	// Start container
	host_config := docker.HostConfig{PortBindings: portsWrapper, Binds: binds}
	err2 := client.StartContainer(container.ID, &host_config)
	if err2 != nil {
		log.Println("Could not start container")
		log.Fatal(err2)
	}

}

func StopApp(app string) {
	log.Println("Stopping application [", app, "]")
	client := Gconfig.DockerClient

	err := client.StopContainer("egor_"+app, 3)
	if err != nil {
		log.Println("Could not stop application")
		log.Fatal(err)
	}

	remove_options := docker.RemoveContainerOptions{ID: "egor_" + app}
	err = client.RemoveContainer(remove_options)
	if err != nil {
		log.Fatal(err)
	}
}

func GetApps() []App {
	client := Gconfig.DockerClient
	apps := []App{}
	log.Println("Retrieving applications")

	listcontaineroptions := docker.ListContainersOptions{All: true}
	containers, err := client.ListContainers(listcontaineroptions)
	if err != nil {
		log.Fatal(err)
	}

	images, err := client.ListImages(true)
	if err != nil {
		log.Fatal(err)
	}

	app_images := []docker.APIImages{}
	for _, image := range images {
		for _, tag := range image.RepoTags {
			name := strings.Split(tag, ":")
			if strings.Contains(name[0], "/") {
				repo := strings.Split(name[0], "/")
				if strings.Contains(repo[0], "egor") {
					log.Println("Found image [", repo[1], "]")
					app_images = append(app_images, image)
				}
			}
		}
	}

	for _, container := range containers {
		app := App{Name: container.Names[0], Status: container.Status}
		apps = append(apps, app)
		//log.Println(app)
	}
	return apps
}
