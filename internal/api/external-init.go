package api

// import (
// 	"net/http"

// 	"github.com/protosio/protos/internal/app"
// 	"github.com/protosio/protos/internal/capability"
// 	"github.com/protosio/protos/internal/resource"

// 	"github.com/pkg/errors"
// )

// var createExternalInitRoutes = func(cm *capability.Manager) routes {
// 	return routes{
// 		route{
// 			"getProtosResources",
// 			"GET",
// 			"/init/resources",
// 			getProtosResources,
// 			cm.GetOrPanic("UserAdmin"),
// 		},
// 		route{
// 			"removeInitProvider",
// 			"DELETE",
// 			"/init/provider",
// 			removeInitProvider,
// 			cm.GetOrPanic("UserAdmin"),
// 		},
// 		route{
// 			"finishInit",
// 			"GET",
// 			"/init/finish",
// 			finishInit,
// 			cm.GetOrPanic("UserAdmin"),
// 		},
// 	}
// }

// //
// // Protos initialisation process
// //

// func removeInitProvider(ha handlerAccess) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		queryParams := r.URL.Query()
// 		provides := queryParams.Get("provides")
// 		if (provides != string(resource.DNS)) && (provides != string(resource.Certificate)) {
// 			log.Errorf("removeInitProvider called with invalid resource type: '%s'", provides)
// 			rend.JSON(w, http.StatusInternalServerError, httperr{Error: "Invalid resource provider type. The only allowed values are 'dns' and  'certificate'"})
// 			return
// 		}

// 		providerFilter := func(app *app.App) bool {
// 			return app.Provides(provides)
// 		}

// 		providerApps := ha.am.Select(providerFilter)
// 		for id, app := range providerApps {
// 			err := app.Stop()
// 			if err != nil {
// 				err = errors.Wrapf(err, "Could not remove init provider for %s", provides)
// 				log.Error(err.Error())
// 			}
// 			err = ha.am.Remove(id)
// 			if err != nil {
// 				err = errors.Wrapf(err, "Could not remove init provider for %s", provides)
// 				log.Error(err.Error())
// 				rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
// 				return
// 			}
// 		}
// 		rend.JSON(w, http.StatusOK, nil)
// 	})
// }

// func getProtosResources(ha handlerAccess) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		resources := ha.m.GetProtosResources()
// 		rend.JSON(w, http.StatusOK, resources)
// 	})
// }

// func finishInit(ha handlerAccess) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		_ = ha.m.GetProtosResources()
// 		// if len(rscs) > 0 {
// 		// 	for id, rsc := range rscs {
// 		// 		if rsc.GetStatus() != core.Created {
// 		// 			err := fmt.Errorf("Can't finish init process because resource '%s'(%s) is not ready", id, string(rsc.GetType()))
// 		// 			log.Error(err)
// 		// 			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
// 		// 			return
// 		// 		}
// 		// 	}
// 		// 	err := ha.m.CleanProtosResources()
// 		// 	if err != nil {
// 		// 		log.Error(err)
// 		// 		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
// 		// 		return
// 		// 	}

// 		// 	err = ha.api.StartExternalWebServer()
// 		// 	if err != nil {
// 		// 		log.Error(err)
// 		// 		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
// 		// 		return
// 		// 	}
// 		// }

// 		rend.JSON(w, http.StatusOK, nil)

// 		err := ha.api.DisableInitRoutes()
// 		if err != nil {
// 			log.Error(err)
// 			return
// 		}
// 	})
// }
