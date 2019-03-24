package api

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/protosio/protos/app"
	"github.com/protosio/protos/meta"
	"github.com/protosio/protos/resource"
	"github.com/protosio/protos/util"
)

var externalInitRoutes = routes{
	route{
		"createProtosResources",
		"POST",
		"/init/resources",
		createProtosResources,
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
		"getProtosResources",
		"GET",
		"/init/resources",
		getProtosResources,
		nil,
	},
	route{
		"removeInitProvider",
		"DELETE",
		"/init/provider",
		removeInitProvider,
		nil,
	},
	route{
		"finishInit",
		"GET",
		"/init/finish",
		finishInit,
		nil,
	},
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

	providerFilter := func(app *app.App) bool {
		if prov, _ := util.StringInSlice(provides, app.InstallerMetadata.Provides); prov {
			return true
		}
		return false
	}

	providerApps := app.Select(providerFilter)
	for _, a := range providerApps {
		err := a.Stop()
		if err != nil {
			err = errors.Wrapf(err, "Could not remove init provider for %s", provides)
			log.Error(err.Error())
		}
		err = a.Remove()
		if err != nil {
			err = errors.Wrapf(err, "Could not remove init provider for %s", provides)
			log.Error(err.Error())
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
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

func getProtosResources(w http.ResponseWriter, r *http.Request) {
	resources := meta.GetProtosResources()
	rend.JSON(w, http.StatusOK, resources)
}

func finishInit(w http.ResponseWriter, r *http.Request) {
	err := meta.CleanProtosResources()
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusBadRequest, httperr{Error: err.Error()})
		return
	}
	dashboard := struct {
		Domain string `json:"domain"`
	}{
		Domain: meta.GetDashboardDomain(),
	}
	quitChan, ok := gconfig.ProcsQuit.Load("initwebserver")
	if ok == false {
		log.Error("Failed to find quit channel for initwebserver")
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: "Failed to stop init web server"})
		return
	}
	quitChan.(chan bool) <- false
	rend.JSON(w, http.StatusOK, dashboard)
}
