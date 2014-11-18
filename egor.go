package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

type AppConfig struct {
	Description string
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

func validate_cfg(app string) {
	log.Println("Validating config for", app)
	filename, _ := filepath.Abs(fmt.Sprintf("./images/%s/app.yaml", app))
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		panic(err)
	}

	var config AppConfig

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Value: %#v\n", config)
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
				println("Running application: ", c.Args().First())
				out, err := exec.Command("sh", "-c", fmt.Sprintf("./images/%s/start.sh", c.Args().First())).Output()
				if err != nil {
					fmt.Printf("%s", err)
				}
				fmt.Printf("%s", out)
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
				validate_cfg(c.Args().First())
			},
		},
	}

	app.Run(os.Args)
}
