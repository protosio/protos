package api

import (
	"encoding/json"
	"net/http"
	"protos/daemon"
	"protos/resource"

	"github.com/gorilla/mux"
)

var clientRoutes = routes{
	route{
		"getApps",
		"GET",
		"/apps",
		getApps,
		nil,
	},
	route{
		"createApp",
		"POST",
		"/apps",
		createApp,
		nil,
	},
	route{
		"getApp",
		"GET",
		"/apps/{appID}",
		getApp,
		nil,
	},
	route{
		"removeApp",
		"DELETE",
		"/apps/{appID}",
		removeApp,
		nil,
	},
	route{
		"actionApp",
		"POST",
		"/apps/{appID}/action",
		actionApp,
		nil,
	},
	route{
		"getInstallers",
		"GET",
		"/installers",
		getInstallers,
		nil,
	},
	route{
		"getInstaller",
		"GET",
		"/installers/{installerID}",
		getInstaller,
		nil,
	},
	route{
		"removeInstaller",
		"DELETE",
		"/installers/{installerID}",
		removeInstaller,
		nil,
	},
	route{
		"getResources",
		"GET",
		"/resources",
		getResources,
		nil,
	},
}

func getApps(w http.ResponseWriter, r *http.Request) {

	apps := daemon.GetApps()
	log.Debug("Sending response: ", apps)
	json.NewEncoder(w).Encode(apps)

}

func createApp(w http.ResponseWriter, r *http.Request) {

	var appParams daemon.App
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&appParams)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	app, err := daemon.CreateApp(appParams.InstallerID, appParams.Name, appParams.PublicPorts, appParams.InstallerParams)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Debug("Sending response: ", app)
	json.NewEncoder(w).Encode(app)

}

func getApp(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	appID := vars["appID"]

	app, err := daemon.ReadApp(appID)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Debug("Sending response: ", app)
	json.NewEncoder(w).Encode(app)

}

func actionApp(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	appID := vars["appID"]

	app, err := daemon.ReadApp(appID)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var action daemon.AppAction
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	err = decoder.Decode(&action)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = app.AddAction(action)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

}

func removeApp(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	appID := vars["appID"]

	app, err := daemon.ReadApp(appID)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = app.Remove()
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

}

func getInstallers(w http.ResponseWriter, r *http.Request) {

	installers, err := daemon.GetInstallers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	log.Debug("Sending response: ", installers)
	json.NewEncoder(w).Encode(installers)

}

func getInstaller(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	installerID := vars["installerID"]

	installer, err := daemon.ReadInstaller(installerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	log.Debug("Sending response: ", installer)
	json.NewEncoder(w).Encode(installer)

}

func removeInstaller(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	installerID := vars["installerID"]

	installer, err := daemon.ReadInstaller(installerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	err = installer.Remove()
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)

}

func getResources(w http.ResponseWriter, r *http.Request) {

	resources := resource.GetAll()

	log.Debug("Sending response: ", resources)
	json.NewEncoder(w).Encode(resources)

}
