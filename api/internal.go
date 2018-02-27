package api

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/nustiueudinastea/protos/capability"
	"github.com/nustiueudinastea/protos/daemon"
	"github.com/nustiueudinastea/protos/meta"
	"github.com/nustiueudinastea/protos/provider"
	"github.com/nustiueudinastea/protos/resource"

	"github.com/gorilla/mux"
	"github.com/tidwall/gjson"
)

var internalRoutes = routes{
	route{
		"getOwnResources",
		"GET",
		"/internal/resource",
		getOwnResources,
		capability.ResourceConsumer,
	},
	route{
		"createResource",
		"POST",
		"/internal/resource",
		createResource,
		capability.ResourceConsumer,
	},
	route{
		"deleteResource",
		"DELETE",
		"/internal/resource/{resourceID}",
		deleteResource,
		capability.ResourceConsumer,
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
		"updateResourceValue",
		"UPDATE",
		"/internal/resource/{resourceID}",
		updateResourceValue,
		capability.ResourceProvider,
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
	route{
		"getDomainInfo",
		"GET",
		"/internal/info/domain",
		getDomainInfo,
		capability.GetInformation,
	},
}

//
// Methods used by resource providers
//

func registerResourceProvider(w http.ResponseWriter, r *http.Request) {
	app := r.Context().Value("app").(*daemon.App)

	rtype, _, err := resource.GetType(mux.Vars(r)["resourceType"])
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

	rtype, _, err := resource.GetType(mux.Vars(r)["resourceType"])
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

	provider, err := provider.Get(app)
	if err != nil {
		err := errors.New("Application " + app.ID + " is not a resource provider")
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Debugf("Retrieving resources for provider %s(%s)", provider.App.ID, provider.Type)
	resources := provider.GetResources()
	json.NewEncoder(w).Encode(resources)
}

func updateResourceValue(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resourceID := vars["resourceID"]

	app := r.Context().Value("app").(*daemon.App)

	prvd, err := provider.Get(app)
	if err != nil {
		err := errors.New("Application " + app.ID + " is not a resource provider")
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rsc := prvd.GetResource(resourceID)
	if rsc == nil {
		err := errors.New("Could not find resource " + resourceID)
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	bodyJSON, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, newValue, err := resource.GetType(string(rsc.Type))
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	err = json.Unmarshal(bodyJSON, newValue)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	rsc.Value.Update(newValue)
	w.WriteHeader(http.StatusOK)

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

	provider, err := provider.Get(app)
	if err != nil {
		err := errors.New("Application " + app.ID + " is not a resource provider")
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

	rsc := provider.GetResource(resourceID)
	if rsc == nil {
		err := errors.New("Could not find resource " + resourceID)
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rsc.SetStatus(status)
	w.WriteHeader(http.StatusOK)

}

func getDomainInfo(w http.ResponseWriter, r *http.Request) {
	domain := struct {
		Domain string
	}{
		Domain: meta.GetDomain(),
	}

	json.NewEncoder(w).Encode(domain)
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
