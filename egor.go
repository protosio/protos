package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/fsouza/go-dockerclient"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"egor/daemon"
)

// Defines structure for config parameters
// specific to each application
type AppConfig struct {
	Description string
	Image       string
	Ports       map[string]string
	Data        string
}

type Config struct {
	DataPath string
	AppsPath string
}

var Gconfig Config

func webadmin(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello admin"))
}


func load_app_cfg(app string) AppConfig {
	log.Println("Reading config for [", app, "]")
	filename, _ := filepath.Abs(fmt.Sprintf("./images/%s/app.yaml", app))
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

func load_cfg(config_file string) Config {
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

	return config
}

func start_app(app string) {
	log.Println("Starting application [", app, "]")
	endpoint := "unix:///var/run/docker.sock"
	client, _ := docker.NewClient(endpoint)

	app_config := load_app_cfg(app)

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

func stop_app(app string) {
	log.Println("Stopping application [", app, "]")
	endpoint := "unix:///var/run/docker.sock"
	client, _ := docker.NewClient(endpoint)

	err := client.StopContainer("egor_"+app, 3)
	if err != nil {
		log.Println("Could not stop application")
		log.Fatal(err)
	}

	remove_options := docker.RemoveContainerOptions{ID: "egor_" + app}
	err2 := client.RemoveContainer(remove_options)
	if err2 != nil {
		log.Fatal(err)
	}
}

//func get_apps() []string {
//	endpoint := "unix:///var/run/docker.sock"
//	client, _ := docker.NewClient(endpoint)
//		
//}

func main() {

	app := cli.NewApp()
	app.Name = "egor"
	app.Usage = "iz good for your privacy"
	app.Author = "Alex Giurgiu"
	app.Email = "alex@giurgiu.io"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "c, config",
			Value: "egor.yaml",
			Usage: "Specify a config file (default: egor.yaml)",
		},
	}

	app.Before = func(c *cli.Context) error {
		Gconfig = load_cfg(c.String("config"))
		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:  "start",
			Usage: "starts an application",
			Action: func(c *cli.Context) {
				start_app(c.Args().First())
			},
		},
		{
			Name:  "stop",
			Usage: "stops an application",
			Action: func(c *cli.Context) {
				stop_app(c.Args().First())
			},
		},
		{
			Name:  "daemon",
			Usage: "starts http server",
			Action: func(c *cli.Context) {
				daemon.Websrv()
			},
		},
		{
			Name:  "validate",
			Usage: "validates application config",
			Action: func(c *cli.Context) {
				load_app_cfg(c.Args().First())
			},
		},
	}

	app.Run(os.Args)
}
