package api

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/protosio/protos/app"
	"github.com/protosio/protos/auth"
	"github.com/protosio/protos/capability"
	"github.com/protosio/protos/meta"
	"github.com/protosio/protos/provider"
	"github.com/protosio/protos/resource"

	"github.com/gorilla/mux"
	"github.com/tidwall/gjson"
)

var internalRoutes = routes{
	route{
		"getOwnResources",
		"GET",
		"/resource",
		getOwnResources,
		capability.ResourceConsumer,
	},
	route{
		"createResource",
		"POST",
		"/resource",
		createResource,
		capability.ResourceConsumer,
	},
	route{
		"deleteResource",
		"DELETE",
		"/resource/{resourceID}",
		deleteResource,
		capability.ResourceConsumer,
	},
	route{
		"registerResourceProvider",
		"POST",
		"/provider/{resourceType}",
		registerResourceProvider,
		capability.RegisterResourceProvider,
	},
	route{
		"deregisterResourceProvider",
		"DELETE",
		"/provider/{resourceType}",
		deregisterResourceProvider,
		capability.DeregisterResourceProvider,
	},
	route{
		"getProviderResources",
		"GET",
		"/resource/provider",
		getProviderResources,
		capability.GetProviderResources,
	},
	route{
		"updateResourceValue",
		"UPDATE",
		"/resource/{resourceID}",
		updateResourceValue,
		capability.ResourceProvider,
	},
	route{
		"setResourceStatus",
		"POST",
		"/resource/{resourceID}",
		setResourceStatus,
		capability.SetResourceStatus,
	},
	route{
		"getResource",
		"GET",
		"/resource/{resourceID}",
		getAppResource,
		capability.ResourceConsumer,
	},
	route{
		"getDomainInfo",
		"GET",
		"/info/domain",
		getDomainInfo,
		capability.GetInformation,
	},
	route{
		"getAdminUser",
		"GET",
		"/info/adminuser",
		getAdminUser,
		capability.GetInformation,
	},
	route{
		"getAppInfo",
		"GET",
		"/info/app",
		getAppInfo,
		capability.GetInformation,
	},
	route{
		"authUser",
		"POST",
		"/user/auth",
		authUser,
		capability.AuthUser,
	},
}

//
// Methods used by resource providers
//

func registerResourceProvider(w http.ResponseWriter, r *http.Request) {
	app := r.Context().Value(appKey).(app.App)

	rtype, _, err := resource.GetType(mux.Vars(r)["resourceType"])
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	err = provider.Register(app, rtype)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
}

func deregisterResourceProvider(w http.ResponseWriter, r *http.Request) {

	app := r.Context().Value(appKey).(app.App)

	rtype, _, err := resource.GetType(mux.Vars(r)["resourceType"])
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	err = provider.Deregister(app, rtype)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getProviderResources(w http.ResponseWriter, r *http.Request) {

	app := r.Context().Value(appKey).(app.App)

	provider, err := provider.Get(app)
	if err != nil {
		err := errors.New("Application " + app.ID + " is not a resource provider")
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	log.Debugf("Retrieving resources for provider %s(%s)", provider.AppID, provider.Type)
	resources := provider.GetResources()
	json.NewEncoder(w).Encode(resources)
}

func updateResourceValue(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resourceID := vars["resourceID"]

	app := r.Context().Value(appKey).(app.App)

	prvd, err := provider.Get(app)
	if err != nil {
		err := errors.New("Application " + app.ID + " is not a resource provider")
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	rsc := prvd.GetResource(resourceID)
	if rsc == nil {
		err := errors.New("Could not find resource " + resourceID)
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	bodyJSON, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	_, newValue, err := resource.GetType(string(rsc.Type))
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
	}
	err = json.Unmarshal(bodyJSON, newValue)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
	}

	rsc.UpdateValue(newValue)
	w.WriteHeader(http.StatusOK)

}

func setResourceStatus(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	resourceID := vars["resourceID"]

	app := r.Context().Value(appKey).(app.App)

	bodyJSON, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	provider, err := provider.Get(app)
	if err != nil {
		err := errors.New("Application " + app.ID + " is not a resource provider")
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	statusName := gjson.GetBytes(bodyJSON, "status").Str
	status, err := resource.GetStatus(statusName)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	rsc := provider.GetResource(resourceID)
	if rsc == nil {
		err := errors.New("Could not find resource " + resourceID)
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	rsc.SetStatus(status)
	w.WriteHeader(http.StatusOK)

}

func getDomainInfo(w http.ResponseWriter, r *http.Request) {
	domain := struct {
		Domain string `json:"domain"`
	}{
		Domain: meta.GetDomain(),
	}

	json.NewEncoder(w).Encode(domain)
}

func getAdminUser(w http.ResponseWriter, r *http.Request) {
	response := struct {
		Username string `json:"username"`
	}{}

	username := meta.GetAdminUser()
	user, err := auth.GetUser(username)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	if user.IsAdmin() != true {
		log.Errorf("User %s is not admin, as recorded in meta", user.Username)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: "Could not find the admin user"})
		return
	}
	response.Username = user.Username

	rend.JSON(w, http.StatusOK, response)
}

//
// Methods used by normal applications to interact with Protos
//

func getOwnResources(w http.ResponseWriter, r *http.Request) {

	app := r.Context().Value(appKey).(app.App)
	resources := app.GetResources()

	json.NewEncoder(w).Encode(resources)

}

func createResource(w http.ResponseWriter, r *http.Request) {

	app := r.Context().Value(appKey).(app.App)

	bodyJSON, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}
	defer r.Body.Close()

	resource, err := app.CreateResource(bodyJSON)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	json.NewEncoder(w).Encode(resource)

}

func getAppResource(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	resourceID := vars["resourceID"]
	app := r.Context().Value(appKey).(app.App)
	rsc := app.GetResource(resourceID)
	if rsc == nil {
		err := errors.New("Could not find resource " + resourceID)
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	json.NewEncoder(w).Encode(rsc)
}

func deleteResource(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	resourceID := vars["resourceID"]

	app := r.Context().Value(appKey).(app.App)
	err := app.DeleteResource(resourceID)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)

}

func getAppInfo(w http.ResponseWriter, r *http.Request) {
	app := r.Context().Value(appKey).(app.App)
	appInfo := struct {
		Name string `json:"name"`
	}{
		Name: app.Name,
	}

	json.NewEncoder(w).Encode(appInfo)
}

//
// User interaction
//

func authUser(w http.ResponseWriter, r *http.Request) {
	var userform struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&userform)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	user, err := auth.ValidateAndGetUser(userform.Username, userform.Password)
	if err != nil {
		log.Debug(err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	json.NewEncoder(w).Encode(user.GetInfo())
}
