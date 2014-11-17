package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/gorilla/mux"
	//"gopkg.in/yaml.v2"
	"log"
	"net/http"
	"os"
	"os/exec"
)

//type AppConfig struct {
//
//}

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

func main() {

	app := cli.NewApp()
	app.Name = "egor"
	app.Usage = "iz good for your privacy"
	app.Action = func(c *cli.Context) {
		println("I work!")
	}

	app.Commands = []cli.Command{
		{
			Name:      "start",
			ShortName: "s",
			Usage:     "starts an application",
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
			Name:      "daemon",
			ShortName: "d",
			Usage:     "starts http server",
			Action: func(c *cli.Context) {
				println("Starting webserver")
				websrv()
			},
		},
	}

	app.Run(os.Args)
}
