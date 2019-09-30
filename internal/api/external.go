package api

import (
	"encoding/json"
	"net/http"

	"protos/internal/app"
	"protos/internal/core"
	"protos/internal/installer"
	"protos/internal/platform"

	"protos/internal/capability"

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
		"cancelTask",
		"PUT",
		"/tasks/{taskID}/cancel",
		cancelTask,
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
	route{
		"getServices",
		"GET",
		"/services",
		getServices,
		nil,
	},
	route{
		"getHWStats",
		"GET",
		"/hwstats",
		getHWStats,
		nil,
	},
}

//
// Apps
//

func getApps(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		apps := ha.am.GetAllPublic()
		log.Debug("Sending response: ", apps)
		json.NewEncoder(w).Encode(apps)
	})
}

func createApp(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var appParams app.App
		defer r.Body.Close()

		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&appParams)
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		tsk := ha.am.CreateAsync(
			appParams.InstallerID,
			appParams.InstallerVersion,
			appParams.Name,
			appParams.InstallerMetadata,
			appParams.InstallerParams,
			true,
		)

		rend.JSON(w, http.StatusAccepted, tsk.Copy())
	})
}

func getApp(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		appID := vars["appID"]

		app, err := ha.am.GetCopy(appID)
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		log.Debug("Sending response: ", app)
		json.NewEncoder(w).Encode(app.Public())

	})
}

func actionApp(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		appID := vars["appID"]

		appInstance, err := ha.am.Read(appID)
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		var action struct {
			Name string
		}
		decoder := json.NewDecoder(r.Body)
		defer r.Body.Close()
		err = decoder.Decode(&action)
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		tsk, err := appInstance.AddAction(action.Name)
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		rend.JSON(w, http.StatusOK, tsk.Copy())
	})
}

func removeApp(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		appID := vars["appID"]

		_, err := ha.am.Read(appID)
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		tsk := ha.am.RemoveAsync(appID)

		rend.JSON(w, http.StatusOK, tsk.Copy())
	})
}

//
// Installers
//

func getInstallers(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		installers, err := installer.GetAll()
		if err != nil {
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		log.Debug("Sending response: ", installers)
		json.NewEncoder(w).Encode(installers)

	})
}

func getInstaller(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		installerID := vars["installerID"]

		installer, err := installer.Read(installerID)
		if err != nil {
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		log.Debug("Sending response: ", installer)
		json.NewEncoder(w).Encode(installer)

	})
}

func removeInstaller(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

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

	})
}

//
// Resources
//

func getResources(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		resources := ha.rm.GetAll(true)

		log.Debug("Sending response: ", resources)
		json.NewEncoder(w).Encode(resources)

	})
}

func getResource(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		resourceID := vars["resourceID"]

		rsc, err := ha.rm.Get(resourceID)
		if err != nil {
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}
		rend.JSON(w, http.StatusOK, rsc.Sanitize())

	})
}

func removeResource(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		resourceID := vars["resourceID"]

		err := ha.rm.Delete(resourceID)
		if err != nil {
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		rend.JSON(w, http.StatusOK, nil)

	})
}

//
// Tasks
//

func getTasks(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tasks := ha.tm.GetLast()
		json, err := tasks.ToJSON()
		if err != nil {
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		}
		log.Debug("Retrieved and sending all tasks: ", tasks)
		rend.Data(w, http.StatusOK, json)
	})
}

func getTask(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		taskID := vars["taskID"]

		tsk, err := ha.tm.Get(taskID)
		if err != nil {
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}
		rend.JSON(w, http.StatusOK, tsk.Copy())

	})
}

func cancelTask(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		taskID := vars["taskID"]

		tsk, err := ha.tm.Get(taskID)
		if err != nil {
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		err = tsk.Kill()
		if err != nil {
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}
		rend.JSON(w, http.StatusOK, nil)

	})
}

//
// App store
//

func searchAppStore(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queryParams := r.URL.Query()
		var err error
		var installers map[string]core.Installer

		if len(queryParams) == 0 {
			installers, err = ha.as.GetInstallers()
		} else if len(queryParams) == 1 {
			if val := queryParams.Get("provides"); val != "" {
				installers, err = ha.as.Search("provides", val)
			} else if val := queryParams.Get("general"); val != "" {
				installers, err = ha.as.Search("general", val)
			} else {
				installers, err = ha.as.GetInstallers()
			}
		}

		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: "Could not query the application store"})
			return
		}

		rend.JSON(w, http.StatusOK, installers)
	})
}

//
// Info endpoints are used to retrieve some general information about the instance and hardware
//

func getInfo(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := struct {
			Version string `json:"version"`
		}{
			Version: gconfig.Version.String(),
		}
		rend.JSON(w, http.StatusOK, info)
	})
}

func getServices(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		services := ha.am.GetServices()
		protosService := ha.m.GetService()
		services = append(services, protosService)
		rend.JSON(w, http.StatusOK, services)
	})
}

func getHWStats(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hwstats, err := platform.GetHWStats()
		if err != nil {
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: "Failed to retrieve hardware stats: " + err.Error()})
		}
		rend.JSON(w, http.StatusOK, hwstats)
	})
}
