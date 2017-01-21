package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// Websrv starts an HTTP server that exposes all the application functionality
func Websrv() {

	rtr := mux.NewRouter()

	//fileHandler := http.FileServer(http.Dir(Gconfig.StaticAssets))

	rtr.HandleFunc("/apps", appsHandler)
	rtr.HandleFunc("/apps/{app}", appHandler)
	rtr.HandleFunc("/", indexHandler)
	//rtr.PathPrefix("/static").Handler(fileHandler)
	//rtr.PathPrefix("/").Handler(fileHandler)
	http.Handle("/", rtr)

	port := strconv.Itoa(Gconfig.Port)
	log.Info("Listening on port " + port)
	http.ListenAndServe(":"+port, nil)

}

func indexHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "Not implemented.")

}

func appsHandler(w http.ResponseWriter, r *http.Request) {

	apps := GetApps()

	data := struct {
		Apps map[string]*App
	}{
		apps,
	}

	log.Debug("Sending response: ", apps)
	json.NewEncoder(w).Encode(data)

}

func appHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)

	decoder := json.NewDecoder(r.Body)
	var appParams App
	err := decoder.Decode(&appParams)
	if err != nil {
		log.Error("Invalid request: ", r.Body)
	}
	log.Debug("Received app state change request: ", appParams)

	appname := vars["app"]

	app := GetApp(appname)

	if r.Method == "POST" {
		if appParams.Status.Running == true {
			app.Start()
		} else if appParams.Status.Running == false {
			app.Stop()
		}
	}

	log.Debug("Sending response: ", app)
	json.NewEncoder(w).Encode(app)

}
