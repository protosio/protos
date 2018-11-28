package api

import (
	"encoding/json"
	"net/http"

	"github.com/protosio/protos/app"
	"github.com/protosio/protos/installer"
	"github.com/protosio/protos/task"

	"github.com/protosio/protos/capability"
	"github.com/protosio/protos/resource"

	"github.com/gorilla/mux"
)

var externalRoutes = routes{
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
		"getTasks",
		"GET",
		"/tasks",
		getTasks,
		nil,
	},
	route{
		"getTask",
		"GET",
		"/tasks/{taskID}",
		getTask,
		nil,
	},
	route{
		"searchAppStore",
		"GET",
		"/store/search",
		searchAppStore,
		nil,
	},
	route{
		"getInfo",
		"GET",
		"/info",
		getInfo,
		nil,
	},
}

//
// Apps
//

func getApps(w http.ResponseWriter, r *http.Request) {

	apps := app.GetAllPublic()
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

	tsk := app.CreateAsync(
		appParams.InstallerID,
		appParams.InstallerVersion,
		appParams.Name,
		appParams.InstallerParams,
		true,
	)

	rend.JSON(w, http.StatusAccepted, tsk)
}

func getApp(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	appID := vars["appID"]

	app, err := app.GetCopy(appID)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	log.Debug("Sending response: ", app)
	json.NewEncoder(w).Encode(app.Public())

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

	tsk, err := appInstance.AddAction(action)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	rend.JSON(w, http.StatusOK, tsk)
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

	tsk := app.RemoveAsync()
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	rend.JSON(w, http.StatusOK, tsk)
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
	rend.JSON(w, http.StatusOK, rsc.Sanitize())

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
// Tasks
//

func getTasks(w http.ResponseWriter, r *http.Request) {
	tasks := task.GetLast()
	json, err := tasks.ToJSON()
	if err != nil {
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
	}
	log.Debug("Retrieved and sending all tasks: ", tasks)
	rend.Data(w, http.StatusOK, json)
}

func getTask(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	taskID := vars["taskID"]

	tsk, err := task.Get(taskID)
	if err != nil {
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}
	rend.JSON(w, http.StatusOK, tsk)

}

//
// App store
//

func searchAppStore(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	var err error
	var installers map[string]installer.Installer

	if len(queryParams) == 0 {
		installers, err = installer.StoreGetAll()
	} else if len(queryParams) == 1 {
		if val := queryParams.Get("provides"); val != "" {
			installers, err = installer.StoreSearch("provides", val)
		} else if val := queryParams.Get("general"); val != "" {
			installers, err = installer.StoreSearch("general", val)
		} else {
			installers, err = installer.StoreGetAll()
		}
	}

	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: "Could not query the application store"})
		return
	}

	rend.JSON(w, http.StatusOK, installers)
}

//
// Info endpoint is used for now only to retrieve some general information
//

func getInfo(w http.ResponseWriter, r *http.Request) {
	info := struct {
		Version string `json:"version"`
	}{
		Version: gconfig.Version.String(),
	}
	rend.JSON(w, http.StatusOK, info)
}
