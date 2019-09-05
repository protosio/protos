package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"protos/internal/app"
)

var externalDevRoutes = routes{
	route{
		"createDevApp",
		"POST",
		"/apps",
		createDevApp,
		nil,
	},
	route{
		"replaceAppContainer",
		"POST",
		"/apps/{appID}/container",
		replaceAppContainer,
		nil,
	},
}

//
// Apps
//

func createDevApp(ha handlerAccess) http.Handler {
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

		app, err := ha.am.CreateDevApp(appParams.InstallerID, appParams.InstallerVersion, appParams.Name, appParams.InstallerMetadata, appParams.InstallerParams)
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}
		rend.JSON(w, http.StatusOK, app)
	})
}

func replaceAppContainer(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		appID := vars["appID"]

		appInstance, err := ha.am.Read(appID)
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		var cntID struct {
			ID string `json:"id"`
		}
		decoder := json.NewDecoder(r.Body)
		defer r.Body.Close()
		err = decoder.Decode(&cntID)
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		err = appInstance.ReplaceContainer(cntID.ID)
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		rend.JSON(w, http.StatusOK, nil)
	})
}
