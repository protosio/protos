package api

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"protos/capability"
	"protos/daemon"
	"protos/provider"
	"protos/resource"

	"github.com/gorilla/mux"
	"github.com/tidwall/gjson"
)

var internalRoutes = routes{
	route{
		"getOwnResources",
		"GET",
		"/internal/resource",
		getOwnResources,
		capability.ResourceProvider,
	},
	route{
		"createResource",
		"POST",
		"/internal/resource",
		createResource,
		capability.ResourceProvider,
	},
	route{
		"deleteResource",
		"DELETE",
		"/internal/resource/{resourceID}",
		deleteResource,
		capability.ResourceProvider,
	},
	route{
		"registerResourceProvider",
		"POST",
		"/internal/provider/{resourceType}",
		registerResourceProvider,
		capability.RegisterResourceProvider,
	},
	route{
		"deregisterResourceProvider",
		"DELETE",
		"/internal/provider/{resourceType}",
		deregisterResourceProvider,
		capability.DeregisterResourceProvider,
	},
	route{
		"getProviderResources",
		"GET",
		"/internal/resource/provider",
		getProviderResources,
		capability.GetProviderResources,
	},
	route{
		"setResourceStatus",
		"POST",
		"/internal/resource/{resourceID}",
		setResourceStatus,
		capability.SetResourceStatus,
	},
	route{
		"getResource",
		"GET",
		"/internal/resource/{resourceID}",
		getResource,
		nil,
	},
}

//
// Methods used by resource providers
//

func registerResourceProvider(w http.ResponseWriter, r *http.Request) {
	app := r.Context().Value("app").(*daemon.App)

	rtype, err := resource.GetType(mux.Vars(r)["resourceType"])
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = provider.Register(app, rtype)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func deregisterResourceProvider(w http.ResponseWriter, r *http.Request) {

	app := r.Context().Value("app").(*daemon.App)

	rtype, err := resource.GetType(mux.Vars(r)["resourceType"])
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = provider.Deregister(app, rtype)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getProviderResources(w http.ResponseWriter, r *http.Request) {

	app := r.Context().Value("app").(*daemon.App)

	resources, err := provider.GetResources(app)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resources)
}

func setResourceStatus(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	resourceID := vars["resourceID"]

	app := r.Context().Value("app").(*daemon.App)

	bodyJSON, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	statusName := gjson.GetBytes(bodyJSON, "status").Str
	status, err := resource.GetStatus(statusName)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rsc := app.Provider.GetResource(resourceID)
	if rsc == nil {
		err := errors.New("Could not find resource " + resourceID)
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rsc.SetStatus(status)
	w.WriteHeader(http.StatusOK)

}

//
// Methods used by normal applications to manipulate their own resources
//

func getOwnResources(w http.ResponseWriter, r *http.Request) {

	app := r.Context().Value("app").(*daemon.App)
	resources := app.GetResources()

	json.NewEncoder(w).Encode(resources)

}

func createResource(w http.ResponseWriter, r *http.Request) {

	app := r.Context().Value("app").(*daemon.App)

	bodyJSON, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	resource, err := app.CreateResource(bodyJSON)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(resource)

}

//ToDo: implement functionality
func getResource(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	resourceID := vars["resourceID"]
	app := r.Context().Value("app").(*daemon.App)
	rsc := app.GetResource(resourceID)
	if rsc == nil {
		err := errors.New("Could not find resource " + resourceID)
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(rsc)
	w.WriteHeader(http.StatusOK)

}

func deleteResource(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	resourceID := vars["resourceID"]

	app := r.Context().Value("app").(*daemon.App)
	err := app.DeleteResource(resourceID)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

}
