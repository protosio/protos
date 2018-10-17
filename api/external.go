package api

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
	"github.com/protosio/protos/app"
	"github.com/protosio/protos/installer"
	"github.com/protosio/protos/meta"
	"github.com/protosio/protos/task"
	"github.com/protosio/protos/util"

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
		"createProtosResources",
		"POST",
		"/init/resources",
		createProtosResources,
		nil,
	},
	route{
		"removeInitProvider",
		"DELETE",
		"/init/provider",
		removeInitProvider,
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

	task, err := task.New(app.CreateAppTask{
		InstallerID:      appParams.InstallerID,
		InstallerVersion: appParams.InstallerVersion,
		AppName:          appParams.Name,
		InstallerParams:  appParams.InstallerParams,
		StartOnCreation:  true,
	})

	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	rend.JSON(w, http.StatusAccepted, task)
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
// Tasks
//

func getTasks(w http.ResponseWriter, r *http.Request) {
	tasks := task.GetAll()
	log.Debug("Retrieved and sending all tasks: ", tasks)
	rend.JSON(w, http.StatusOK, tasks)
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
// Protos initialisation process
//

func removeInitProvider(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	provides := queryParams.Get("provides")
	if (provides != string(resource.DNS)) && (provides != string(resource.Certificate)) {
		log.Errorf("removeInitProvider called with invalid resource type: '%s'", provides)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: "Invalid resource provider type. The only allowed values are 'dns' and  'certificate'"})
		return
	}

	apps := app.GetApps()
	for _, a := range apps {
		if prov, _ := util.StringInSlice(provides, a.InstallerMetadata.Provides); prov {
			err := a.Stop()
			if err != nil {
				err = errors.Wrapf(err, "Could not remove init provider for %s", provides)
				log.Error(err.Error())
				rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
				return
			}
			err = a.Remove()
			if err != nil {
				err = errors.Wrapf(err, "Could not remove init provider for %s", provides)
				log.Error(err.Error())
				rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
				return
			}
		}
	}
	rend.JSON(w, http.StatusOK, nil)
}

func createProtosResources(w http.ResponseWriter, r *http.Request) {
	resources, err := meta.CreateProtosResources()
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusBadRequest, httperr{Error: err.Error()})
		return
	}

	rend.JSON(w, http.StatusOK, resources)
}
