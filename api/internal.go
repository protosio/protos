package api

import (
	"encoding/json"
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
}

func registerResourceProvider(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func registerResource(w http.ResponseWriter, r *http.Request) {

	app, err := daemon.ReadAppByIP(strings.Split(r.RemoteAddr, ":")[0])
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}
	log.Debug(app)

	var resource daemon.Resource
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	err = decoder.Decode(&resource)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), 500)
		return
	}
	err = daemon.AddResources(resource)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(http.StatusOK)

}
