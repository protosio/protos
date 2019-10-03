package api

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"protos/internal/capability"
	"protos/internal/core"

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

func registerResourceProvider(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app := r.Context().Value(appKey).(core.App)

		rtype, _, err := ha.rm.GetType(mux.Vars(r)["resourceType"])
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		err = ha.pm.Register(app, rtype)
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}

func deregisterResourceProvider(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		app := r.Context().Value(appKey).(core.App)

		rtype, _, err := ha.rm.GetType(mux.Vars(r)["resourceType"])
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		err = ha.pm.Deregister(app, rtype)
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}

func getProviderResources(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		app := r.Context().Value(appKey).(core.App)

		provider, err := ha.pm.Get(app)
		if err != nil {
			err := errors.New("Application " + app.GetID() + " is not a resource provider")
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		log.Debugf("Retrieving resources for provider %s(%s)", app.GetID(), provider.TypeName())
		resources := provider.GetResources()
		json.NewEncoder(w).Encode(resources)
	})
}

func updateResourceValue(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		resourceID := vars["resourceID"]

		app := r.Context().Value(appKey).(core.App)

		prvd, err := ha.pm.Get(app)
		if err != nil {
			err := errors.New("Application " + app.GetID() + " is not a resource provider")
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

		_, newValue, err := ha.rm.GetType(string(rsc.GetType()))
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

	})
}

func setResourceStatus(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		resourceID := vars["resourceID"]

		app := r.Context().Value(appKey).(core.App)

		bodyJSON, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		provider, err := ha.pm.Get(app)
		if err != nil {
			err := errors.New("Application " + app.GetID() + " is not a resource provider")
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		statusName := gjson.GetBytes(bodyJSON, "status").Str
		status, err := ha.rm.GetStatus(statusName)
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

	})
}

func getDomainInfo(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		domain := struct {
			Domain string `json:"domain"`
		}{
			Domain: ha.m.GetDomain(),
		}

		json.NewEncoder(w).Encode(domain)
	})
}

func getAdminUser(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := struct {
			Username string `json:"username"`
		}{}

		username := ha.m.GetAdminUser()
		user, err := ha.um.GetUser(username)
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		if user.IsAdmin() != true {
			log.Errorf("User %s is not admin, as recorded in meta", user.GetUsername())
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: "Could not find the admin user"})
			return
		}
		response.Username = user.GetUsername()

		rend.JSON(w, http.StatusOK, response)
	})
}

//
// Methods used by normal applications to interact with Protos
//

func getOwnResources(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		app := r.Context().Value(appKey).(core.App)
		resources := app.GetResources()

		json.NewEncoder(w).Encode(resources)

	})
}

func createResource(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		app := r.Context().Value(appKey).(core.App)

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

	})
}

func getAppResource(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		resourceID := vars["resourceID"]
		app := r.Context().Value(appKey).(core.App)
		rsc, err := app.GetResource(resourceID)
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		json.NewEncoder(w).Encode(rsc)
	})
}

func deleteResource(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		resourceID := vars["resourceID"]

		app := r.Context().Value(appKey).(core.App)
		err := app.DeleteResource(resourceID)
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		w.WriteHeader(http.StatusOK)

	})
}

func getAppInfo(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app := r.Context().Value(appKey).(core.App)
		appInfo := struct {
			Name string `json:"name"`
		}{
			Name: app.GetName(),
		}

		json.NewEncoder(w).Encode(appInfo)
	})
}

//
// User interaction
//

func authUser(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		user, err := ha.um.ValidateAndGetUser(userform.Username, userform.Password)
		if err != nil {
			log.Debug(err)
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		json.NewEncoder(w).Encode(user.GetInfo())
	})
}
