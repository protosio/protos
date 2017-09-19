package api

import (
	"encoding/json"
	"net/http"
	"protos/daemon"

	"github.com/gorilla/mux"
)

var clientRoutes = routes{
	route{
		"getApps",
		"GET",
		"/apps",
		getApps,
	},
	route{
		"createApp",
		"POST",
		"/apps",
		createApp,
	},
	route{
		"getApp",
		"GET",
		"/apps/{appID}",
		getApp,
	},
	route{
		"removeApp",
		"DELETE",
		"/apps/{appID}",
		removeApp,
	},
	route{
		"actionApp",
		"POST",
		"/apps/{appID}/action",
		actionApp,
	},
	route{
		"getInstallers",
		"GET",
		"/installers",
		getInstallers,
	},
	route{
		"getInstaller",
		"GET",
		"/installers/{installerID}",
		getInstaller,
	},
	route{
		"removeInstaller",
		"DELETE",
		"/installers/{installerID}",
		removeInstaller,
	},
	route{
		"writeInstallerMetadata",
		"POST",
		"/installers/{installerID}/metadata",
		writeInstallerMetadata,
	},
	route{
		"getResources",
		"GET",
		"/resources",
		getResources,
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
		http.Error(w, err.Error(), 500)
		return
	}

	app, err := daemon.CreateApp(appParams.ImageID, appParams.Name, appParams.Command, appParams.PublicPorts, appParams.InstallerParams)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
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
		http.Error(w, err.Error(), 500)
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
		http.Error(w, err.Error(), 500)
		return
	}

	var action daemon.AppAction
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	err = decoder.Decode(&action)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}

	err = app.AddAction(action)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
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
		http.Error(w, err.Error(), 500)
		return
	}

	err = app.Remove()
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(http.StatusOK)

}

func getInstallers(w http.ResponseWriter, r *http.Request) {

	installers := daemon.GetInstallers()

	log.Debug("Sending response: ", installers)
	json.NewEncoder(w).Encode(installers)

}

func getInstaller(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	installerID := vars["installerID"]

	installer, err := daemon.ReadInstaller(installerID)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}

	log.Debug("Sending response: ", installer)
	json.NewEncoder(w).Encode(installer)

}

func removeInstaller(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	installerID := vars["installerID"]

	installer, err := daemon.ReadInstaller(installerID)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
	err = installer.Remove()
	if err != nil {
		http.Error(w, err.Error(), 500)
	}

	w.WriteHeader(http.StatusOK)

}

func writeInstallerMetadata(w http.ResponseWriter, r *http.Request) {

	type Payload struct {
		Metadata string `json:"metadata"`
	}

	vars := mux.Vars(r)
	installerID := vars["installerID"]

	installer, err := daemon.ReadInstaller(installerID)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}

	var payload Payload
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	err = decoder.Decode(&payload)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}

	var metadata daemon.InstallerMetadata
	err = json.Unmarshal([]byte(payload.Metadata), &metadata)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}

	if err = installer.WriteMetadata(metadata); err != nil {
		http.Error(w, err.Error(), 500)
	}

	w.WriteHeader(http.StatusOK)

}

func getResources(w http.ResponseWriter, r *http.Request) {

	resources := daemon.GetResources()

	log.Debug("Sending response: ", resources)
	json.NewEncoder(w).Encode(resources)

}
