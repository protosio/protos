package api

import (
	"encoding/json"
	"net/http"

	"github.com/nustiueudinastea/protos/meta"

	"github.com/nustiueudinastea/protos/capability"
	"github.com/nustiueudinastea/protos/daemon"
	"github.com/nustiueudinastea/protos/resource"

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
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	app, err := daemon.CreateApp(appParams.InstallerID, appParams.InstallerVersion, appParams.Name, appParams.InstallerParams)
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

	app, err := daemon.ReadApp(appID)
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

	app, err := daemon.ReadApp(appID)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	var action daemon.AppAction
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	err = decoder.Decode(&action)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	err = app.AddAction(action)
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

	app, err := daemon.ReadApp(appID)
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

func getInstallers(w http.ResponseWriter, r *http.Request) {

	installers, err := daemon.GetInstallers()
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

	installer, err := daemon.ReadInstaller(installerID)
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

	installer, err := daemon.ReadInstaller(installerID)
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

func getResources(w http.ResponseWriter, r *http.Request) {

	resources := resource.GetAll(true)

	log.Debug("Sending response: ", resources)
	json.NewEncoder(w).Encode(resources)

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
	provides, ok := queryParams["provides"]
	if ok != true || len(provides) == 0 {
		rend.JSON(w, http.StatusBadRequest, httperr{Error: "'provides' is the only allowed search parameter"})
		return
	}

	resp, err := http.Get(gconfig.AppStoreURL + "/api/v1/search?provides=" + provides[0])
	if err != nil {
		log.Fatalln(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: "Could not query the application store"})
		return
	}

	installers := map[string]struct {
		Name        string   `json:"name"`
		Provides    []string `json:"provides"`
		Description string   `json:"description"`
		Versions    []string `json:"versions"`
	}{}

	err = json.NewDecoder(resp.Body).Decode(&installers)
	defer resp.Body.Close()
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: "Something went wrong decoding the response from the application store"})
		return
	}

	rend.JSON(w, http.StatusOK, installers)
}

func downloadInstaller(w http.ResponseWriter, r *http.Request) {
	var installer = struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Version string `json:"version"`
	}{}

	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&installer)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusBadRequest, httperr{Error: "Could not decode JSON request"})
		return
	}

	err = daemon.DownloadInstaller(installer.Name, installer.Version)
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

	// err = meta.WaitForProtosResources()
	// if err != nil {
	// 	log.Error(err)
	// 	rend.JSON(w, http.StatusBadRequest, httperr{Error: err.Error()})
	// 	return
	// }

	rend.JSON(w, http.StatusOK, resources)
}
