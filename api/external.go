package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/protosio/protos/app"
	"github.com/protosio/protos/installer"
	"github.com/protosio/protos/meta"

	"github.com/protosio/protos/capability"
	"github.com/protosio/protos/resource"

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
	route{
		"getResource",
		"GET",
		"/resources/{resourceID}",
		getResource,
		nil,
	},
	route{
		"removeResource",
		"DELETE",
		"/resources/{resourceID}",
		removeResource,
		capability.UserAdmin,
	},
	route{
		"searchAppStore",
		"GET",
		"/store/search",
		searchAppStore,
		nil,
	},
	route{
		"downloadInstaller",
		"POST",
		"/store/download",
		downloadInstaller,
		nil,
	},
	route{
		"createProtosResources",
		"POST",
		"/protos/resources",
		createProtosResources,
		nil,
	},
}

//
// Apps
//

func getApps(w http.ResponseWriter, r *http.Request) {

	apps := app.GetApps()
	log.Debug("Sending response: ", apps)
	json.NewEncoder(w).Encode(apps)
}

func createApp(w http.ResponseWriter, r *http.Request) {

	var appParams app.App
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&appParams)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	app, err := app.CreateApp(appParams.InstallerID, appParams.InstallerVersion, appParams.Name, appParams.InstallerParams)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	log.Debug("Sending response: ", app)
	json.NewEncoder(w).Encode(app)

}

func getApp(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	appID := vars["appID"]

	app, err := app.Read(appID)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	log.Debug("Sending response: ", app)
	json.NewEncoder(w).Encode(app)

}

func actionApp(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	appID := vars["appID"]

	appInstance, err := app.Read(appID)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	var action app.Action
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	err = decoder.Decode(&action)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	err = appInstance.AddAction(action)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	rend.JSON(w, http.StatusOK, nil)
}

func removeApp(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	appID := vars["appID"]

	app, err := app.Read(appID)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	err = app.Remove()
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	rend.JSON(w, http.StatusOK, nil)
}

//
// Installers
//

func getInstallers(w http.ResponseWriter, r *http.Request) {

	installers, err := installer.GetAll()
	if err != nil {
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	log.Debug("Sending response: ", installers)
	json.NewEncoder(w).Encode(installers)

}

func getInstaller(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	installerID := vars["installerID"]

	installer, err := installer.Read(installerID)
	if err != nil {
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	log.Debug("Sending response: ", installer)
	json.NewEncoder(w).Encode(installer)

}

func removeInstaller(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	installerID := vars["installerID"]

	installer, err := installer.Read(installerID)
	if err != nil {
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}
	err = installer.Remove()
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	rend.JSON(w, http.StatusOK, nil)

}

//
// Resources
//

func getResources(w http.ResponseWriter, r *http.Request) {

	resources := resource.GetAll(true)

	log.Debug("Sending response: ", resources)
	json.NewEncoder(w).Encode(resources)

}

func getResource(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	resourceID := vars["resourceID"]

	rsc, err := resource.Get(resourceID)
	if err != nil {
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}
	rend.JSON(w, http.StatusOK, rsc)

}

func removeResource(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	resourceID := vars["resourceID"]

	rsc, err := resource.Get(resourceID)
	if err != nil {
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}
	err = rsc.Delete()
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	rend.JSON(w, http.StatusOK, nil)

}

//
// App store
//

func searchAppStore(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	installers := map[string]struct {
		Name     string                         `json:"name"`
		Versions map[string]*installer.Metadata `json:"versions"`
	}{}
	var resp *http.Response
	var err error

	if len(queryParams) == 0 {
		resp, err = http.Get(gconfig.AppStoreURL + "/api/v1/installers/all")
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: "Could not query the application store"})
			return
		}
	} else if len(queryParams) == 1 {
		provides, providesFound := queryParams["provides"]
		general, generalFound := queryParams["general"]
		if providesFound == true && len(provides) > 0 {
			resp, err = http.Get(gconfig.AppStoreURL + "/api/v1/search?provides=" + provides[0])
			if err != nil {
				log.Error(err)
				rend.JSON(w, http.StatusInternalServerError, httperr{Error: "Could not query the application store"})
				return
			}
		} else if generalFound && len(general) > 0 {
			resp, err = http.Get(gconfig.AppStoreURL + "/api/v1/search?general=" + general[0])
			if err != nil {
				log.Error(err)
				rend.JSON(w, http.StatusInternalServerError, httperr{Error: "Could not query the application store"})
				return
			}
		} else {
			rend.JSON(w, http.StatusBadRequest, httperr{Error: "'provides' and 'general' are the only allowed search parameters"})
			return
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		errText := fmt.Sprintf("Error (HTTP %d) while querying the application store: \"%s\"", resp.StatusCode, string(bodyBytes))
		log.Error(errText)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: errText})
		return
	}

	err = json.NewDecoder(resp.Body).Decode(&installers)
	defer resp.Body.Close()
	if err != nil {
		log.Errorf("Something went wrong decoding the response from the application store: %s", err.Error())
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: "Something went wrong decoding the response from the application store"})
		return
	}

	rend.JSON(w, http.StatusOK, installers)
}

func downloadInstaller(w http.ResponseWriter, r *http.Request) {
	var installerParams = struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Version string `json:"version"`
	}{}

	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&installerParams)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusBadRequest, httperr{Error: "Could not decode JSON request"})
		return
	}

	err = installer.Download(installerParams.Name, installerParams.Version)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusBadRequest, httperr{Error: err.Error()})
		return
	}

	rend.JSON(w, http.StatusOK, nil)
}

//
// Protos resources (DNS and TLS)
//

func createProtosResources(w http.ResponseWriter, r *http.Request) {
	resources, err := meta.CreateProtosResources()
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusBadRequest, httperr{Error: err.Error()})
		return
	}

	rend.JSON(w, http.StatusOK, resources)
}
