package api

import (
	"encoding/json"
	"net/http"

	"github.com/protosio/protos/app"
)

var externalDevRoutes = routes{
	route{
		"createDevApp",
		"POST",
		"/apps",
		createDevApp,
		nil,
	},
}

//
// Apps
//

func createDevApp(w http.ResponseWriter, r *http.Request) {
	var appParams app.App
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&appParams)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	err = app.CreateDevApp(appParams.InstallerID, appParams.InstallerVersion, appParams.Name, appParams.InstallerMetadata, appParams.InstallerParams)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}
	rend.JSON(w, http.StatusOK, nil)
}
