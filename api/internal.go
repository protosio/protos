package api

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"protos/daemon"
	"strings"
)

var internalRoutes = routes{
	route{
		"registerResource",
		"POST",
		"/internal/resource",
		registerResource,
	},
	route{
		"registerResourceProvider",
		"POST",
		"/internal/provider",
		registerResourceProvider,
	},
	route{
		"unregisterResourceProvider",
		"DELETE",
		"/internal/provider",
		unregisterResourceProvider,
	},
	route{
		"getProviderResources",
		"GET",
		"/internal/provider/resources",
		getProviderResources,
	},
}

func registerResourceProvider(w http.ResponseWriter, r *http.Request) {

	app, err := daemon.ReadAppByIP(strings.Split(r.RemoteAddr, ":")[0])
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}

	var provider daemon.Provider
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	err = decoder.Decode(&provider)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}
	err = daemon.RegisterProvider(provider, &app)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func unregisterResourceProvider(w http.ResponseWriter, r *http.Request) {

	app, err := daemon.ReadAppByIP(strings.Split(r.RemoteAddr, ":")[0])
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}

	var provider daemon.Provider
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	err = decoder.Decode(&provider)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}
	err = daemon.UnregisterProvider(provider, &app)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getProviderResources(w http.ResponseWriter, r *http.Request) {

	app, err := daemon.ReadAppByIP(strings.Split(r.RemoteAddr, ":")[0])
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}

	resources, err := daemon.GetProviderResources(&app)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	log.Debug(resources)

	json.NewEncoder(w).Encode(resources)
}

func registerResource(w http.ResponseWriter, r *http.Request) {

	app, err := daemon.ReadAppByIP(strings.Split(r.RemoteAddr, ":")[0])
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}

	bodyJSON, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}
	defer r.Body.Close()

	resource, err := daemon.GetResourceFromJSON(bodyJSON)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}

	err = daemon.AddResource(resource, &app)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(http.StatusOK)

}
