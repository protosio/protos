package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/nustiueudinastea/protos/capability"
	"github.com/nustiueudinastea/protos/config"
	"github.com/nustiueudinastea/protos/util"

	"github.com/gorilla/mux"
	"github.com/unrolled/render"
	"github.com/urfave/negroni"
)

var log = util.Log
var gconfig = config.Get()
var rend = render.New(render.Options{IndentJSON: true})

type httperr struct {
	Error string `json:"error"`
}

type route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
	Capability  *capability.Capability
}

type routes []route

func applyAPIroutes(r *mux.Router) {

	// Internal routes
	internalRouter := mux.NewRouter().PathPrefix("/api/v1/i").Subrouter().StrictSlash(true)
	for _, route := range internalRoutes {
		internalRouter.Methods(route.Method).Path(route.Pattern).Name(route.Name).Handler(route.HandlerFunc)
		if route.Capability != nil {
			capability.SetMethodCap(route.Name, route.Capability)
		}
	}

	r.PathPrefix("/api/v1/i").Handler(negroni.New(
		negroni.HandlerFunc(InternalRequestValidator),
		negroni.Wrap(internalRouter),
	))

	// External routes (require auth)
	externalRouter := mux.NewRouter().PathPrefix("/api/v1/e").Subrouter().StrictSlash(true)
	for _, route := range clientRoutes {
		externalRouter.Methods(route.Method).Path(route.Pattern).Name(route.Name).Handler(route.HandlerFunc)
		if route.Capability != nil {
			capability.SetMethodCap(route.Name, route.Capability)
		}
	}

	r.PathPrefix("/api/v1/e").Handler(negroni.New(
		negroni.HandlerFunc(ExternalRequestValidator),
		negroni.Wrap(externalRouter),
	))

	// Authentication routes
	authRouter := mux.NewRouter().PathPrefix("/api/v1/auth").Subrouter().StrictSlash(true)
	if gconfig.InitMode == true {
		authRouter.Methods("POST").Path("/register").Name("register").Handler(http.HandlerFunc(RegisterHandler))
	}
	authRouter.Methods("POST").Path("/login").Name("login").Handler(http.HandlerFunc(LoginHandler))

	r.PathPrefix("/api/v1/auth").Handler(authRouter)

}

// Websrv starts an HTTP server that exposes all the application functionality
func Websrv() {

	mainRtr := mux.NewRouter().StrictSlash(true)
	applyAPIroutes(mainRtr)

	fileHandler := http.FileServer(http.Dir(gconfig.StaticAssets))
	mainRtr.PathPrefix("/static").Handler(fileHandler)
	mainRtr.PathPrefix("/").Handler(fileHandler)

	// Negroni middleware
	n := negroni.New()
	n.Use(negroni.HandlerFunc(HTTPLogger))
	n.UseHandler(mainRtr)

	port := strconv.Itoa(gconfig.Port)
	log.Info("Listening on port " + port)
	server := &http.Server{
		Addr:           ":" + port,
		Handler:        n,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	server.ListenAndServe()

}
