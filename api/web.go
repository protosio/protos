package api

import (
	"fmt"
	"net/http"
	"protos/daemon"
	"protos/util"
	"strconv"

	"github.com/gorilla/mux"
)

var log = util.Log

type route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

type routes []route

func newRouter() *mux.Router {

	appRoutes := []routes{
		clientRoutes,
		internalRoutes,
	}

	var allRoutes routes
	for _, r := range appRoutes {
		allRoutes = append(allRoutes, r...)
	}

	router := mux.NewRouter().StrictSlash(true)
	for _, route := range allRoutes {
		var handler http.Handler

		handler = route.HandlerFunc
		handler = util.HTTPLogger(handler, route.Name)

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)

	}

	return router
}

// Websrv starts an HTTP server that exposes all the application functionality
func Websrv() {

	rtr := newRouter()

	fileHandler := http.FileServer(http.Dir(daemon.Gconfig.StaticAssets))
	rtr.PathPrefix("/static").Handler(fileHandler)
	rtr.PathPrefix("/").Handler(fileHandler)
	http.Handle("/", rtr)

	port := strconv.Itoa(daemon.Gconfig.Port)
	log.Info("Listening on port " + port)
	http.ListenAndServe(":"+port, nil)

}

func indexHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "Not implemented.")

}
