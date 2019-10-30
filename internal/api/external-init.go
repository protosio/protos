package api

import (
	"net/http"

	"github.com/protosio/protos/internal/core"

	"github.com/pkg/errors"
)

var createExternalInitRoutes = func(cm core.CapabilityManager) routes {
	return routes{
		route{
			"createProtosResources",
			"POST",
			"/init/resources",
			createProtosResources,
			cm.GetOrPanic("UserAdmin"),
		},
		route{
			"getProtosResources",
			"GET",
			"/init/resources",
			getProtosResources,
			cm.GetOrPanic("UserAdmin"),
		},
		route{
			"removeInitProvider",
			"DELETE",
			"/init/provider",
			removeInitProvider,
			cm.GetOrPanic("UserAdmin"),
		},
		route{
			"finishInit",
			"GET",
			"/init/finish",
			finishInit,
			cm.GetOrPanic("UserAdmin"),
		},
	}
}

//
// Protos initialisation process
//

func removeInitProvider(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queryParams := r.URL.Query()
		provides := queryParams.Get("provides")
		if (provides != string(core.DNS)) && (provides != string(core.Certificate)) {
			log.Errorf("removeInitProvider called with invalid resource type: '%s'", provides)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: "Invalid resource provider type. The only allowed values are 'dns' and  'certificate'"})
			return
		}

		providerFilter := func(app core.App) bool {
			return app.Provides(provides)
		}

		providerApps := ha.am.Select(providerFilter)
		for id, a := range providerApps {
			err := a.Stop()
			if err != nil {
				err = errors.Wrapf(err, "Could not remove init provider for %s", provides)
				log.Error(err.Error())
			}
			err = ha.am.Remove(id)
			if err != nil {
				err = errors.Wrapf(err, "Could not remove init provider for %s", provides)
				log.Error(err.Error())
				rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
				return
			}
		}
		rend.JSON(w, http.StatusOK, nil)
	})
}

func createProtosResources(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resources, err := ha.m.CreateProtosResources()
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusBadRequest, httperr{Error: err.Error()})
			return
		}

		rend.JSON(w, http.StatusOK, resources)
	})
}

func getProtosResources(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resources := ha.m.GetProtosResources()
		rend.JSON(w, http.StatusOK, resources)
	})
}

func finishInit(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := ha.m.CleanProtosResources()
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusBadRequest, httperr{Error: err.Error()})
			return
		}
		dashboard := struct {
			Domain string `json:"domain"`
		}{
			Domain: ha.m.GetDashboardDomain(),
		}
		quitChan, ok := gconfig.ProcsQuit.Load("initwebserver")
		if ok == false {
			log.Error("Failed to find quit channel for initwebserver")
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: "Failed to stop init web server"})
			return
		}
		quitChan.(chan bool) <- false
		rend.JSON(w, http.StatusOK, dashboard)
	})
}
