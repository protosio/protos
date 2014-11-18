package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/fsouza/go-dockerclient"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type AppConfig struct {
	Description string
	Image       string
	Ports       map[int]int
	Data        string
}

func webadmin(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello admin"))
}

func websrv() {
	rtr := mux.NewRouter()

	//rtr.HandleFunc("/admin", webadmin).Methods("GET")
	//http.Handle("/", rtr)

	rtr.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))
	http.Handle("/", rtr)

	log.Println("Listening...")
	http.ListenAndServe(":9000", nil)

}

func load_cfg(app string) AppConfig {
	log.Println("Reading config for", app)
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

func start_app(app string) {
	log.Println("Starting application [", app, "]")
	endpoint := "unix:///var/run/docker.sock"
	client, _ := docker.NewClient(endpoint)

	app_config := load_cfg(app)

	// Create container
	config := docker.Config{Image: app_config.Image}
	create_options := docker.CreateContainerOptions{Name: "egor_" + app, Config: &config}
	container, err := client.CreateContainer(create_options)
	if err != nil {
		log.Println("Could not create container")
		log.Fatal(err)
	}

	// Start container
	host_config := docker.HostConfig{}
	err2 := client.StartContainer(container.ID, &host_config)
	if err2 != nil {
		log.Println("Could not start container")
		log.Fatal(err2)
	}

}

func main() {

	app := cli.NewApp()
	app.Name = "egor"
	app.Usage = "iz good for your privacy"
	app.Action = func(c *cli.Context) {
		println("I work!")
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
			Name:  "daemon",
			Usage: "starts http server",
			Action: func(c *cli.Context) {
				println("Starting webserver")
				websrv()
			},
		},
		{
			Name:  "validate",
			Usage: "validates application config",
			Action: func(c *cli.Context) {
				load_cfg(c.Args().First())
			},
		},
	}

	app.Run(os.Args)
}
