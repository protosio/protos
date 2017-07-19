package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

type Routes []Route

var routes = Routes{
	Route{
		"getApps",
		"GET",
		"/apps",
		getApps,
	},
	Route{
		"createApp",
		"POST",
		"/apps",
		createApp,
	},
	Route{
		"getApp",
		"GET",
		"/apps/{appID}",
		getApp,
	},
	Route{
		"removeApp",
		"DELETE",
		"/apps/{appID}",
		removeApp,
	},
	Route{
		"actionApp",
		"POST",
		"/apps/{appID}/action",
		actionApp,
	},
	Route{
		"getInstallers",
		"GET",
		"/installers",
		getInstallers,
	},
	Route{
		"getInstaller",
		"GET",
		"/installers/{installerID}",
		getInstaller,
	},
	Route{
		"removeInstaller",
		"DELETE",
		"/installers/{installerID}",
		removeInstaller,
	},
	Route{
		"writeInstallerMetadata",
		"POST",
		"/installers/{installerID}/metadata",
		writeInstallerMetadata,
	},
	Route{
		"registerResourceProvider",
		"POST",
		"/resources/providers",
		registerResourceProvider,
	},
}

func newRouter() *mux.Router {

	router := mux.NewRouter().StrictSlash(true)
	for _, route := range routes {
		var handler http.Handler

		handler = route.HandlerFunc
		handler = httpLogger(handler, route.Name)

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)

	}

	return router
}

// Websrv starts an HTTP server that exposes all the application functionality
func Websrv() {

	rtr := newRouter()

	fileHandler := http.FileServer(http.Dir(Gconfig.StaticAssets))
	rtr.PathPrefix("/static").Handler(fileHandler)
	rtr.PathPrefix("/").Handler(fileHandler)
	http.Handle("/", rtr)

	port := strconv.Itoa(Gconfig.Port)
	log.Info("Listening on port " + port)
	http.ListenAndServe(":"+port, nil)

}

func indexHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "Not implemented.")

}

func getApps(w http.ResponseWriter, r *http.Request) {

	apps := GetApps()
	log.Debug("Sending response: ", apps)
	json.NewEncoder(w).Encode(apps)

}

func createApp(w http.ResponseWriter, r *http.Request) {

	var appParams App
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	err := decoder.Decode(&appParams)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}

	app, err := CreateApp(appParams.ImageID, appParams.Name, appParams.Command)
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

	app, err := ReadApp(appID)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}

	log.Debug("Sending response: ", app)
	json.NewEncoder(w).Encode(app)

}

func actionApp(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	appID := vars["appID"]

	app, err := ReadApp(appID)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}

	var action AppAction
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

	app, err := ReadApp(appID)
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

	installers := GetInstallers()

	log.Debug("Sending response: ", installers)
	json.NewEncoder(w).Encode(installers)

}

func getInstaller(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	installerID := vars["installerID"]

	installer, err := ReadInstaller(installerID)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}

	log.Debug("Sending response: ", installer)
	json.NewEncoder(w).Encode(installer)

}

func removeInstaller(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	installerID := vars["installerID"]

	installer, err := ReadInstaller(installerID)
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

	vars := mux.Vars(r)
	installerID := vars["installerID"]

	installer, err := ReadInstaller(installerID)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}

	var metadata InstallerMetadata
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	err = decoder.Decode(&metadata)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}

	if err = installer.writeMetadata(metadata); err != nil {
		http.Error(w, err.Error(), 500)
	}

	w.WriteHeader(http.StatusOK)

}

func registerResourceProvider(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
