package api

import (
	"crypto/tls"
	"net/http"
	"strconv"
	"time"

	"github.com/protosio/protos/resource"

	"github.com/protosio/protos/capability"
	"github.com/protosio/protos/config"
	"github.com/protosio/protos/util"

	"github.com/gorilla/mux"
	"github.com/unrolled/render"
	"github.com/urfave/negroni"
)

var log = util.GetLogger()
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

// initListen is only used to start a http web server to complete the initialisation phase
func initListen(handler http.Handler) {
	httpport := strconv.Itoa(gconfig.HTTPport)
	log.Info("Listening on port " + httpport)
	srv := &http.Server{
		Addr:           ":" + httpport,
		Handler:        handler,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(srv.ListenAndServe())
}

func secureListen(handler http.Handler, certrsc resource.Type) {
	cert := certrsc.(*resource.CertificateResource)
	tlscert, err := tls.X509KeyPair(cert.Certificate, cert.PrivateKey)
	if err != nil {
		log.Fatalf("Failed to parse the TLS certificate: %s", err.Error())
	}
	cfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
		Certificates: []tls.Certificate{tlscert},
	}

	httpsport := strconv.Itoa(gconfig.HTTPSport)
	httpport := strconv.Itoa(gconfig.HTTPport)
	srv := &http.Server{
		Addr:         ":" + httpsport,
		Handler:      handler,
		TLSConfig:    cfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}
	ips, err := util.GetLocalIPs()
	if err != nil {
		log.Fatal(err)
	}
	for _, ip := range ips {
		log.Infof("Listening internally on %s:%s (HTTP)", ip, httpport)
		go func() {
			log.Fatal(http.ListenAndServe(ip+":"+httpport, handler))
		}()
	}

	log.Infof("Listening on port %s (HTTPS)", httpsport)
	log.Fatal(srv.ListenAndServeTLS("", ""))
}

// Websrv starts an HTTP server that exposes all the application functionality
func Websrv(certrsc *resource.Resource) {

	mainRtr := mux.NewRouter().StrictSlash(true)
	applyAPIroutes(mainRtr)

	fileHandler := http.FileServer(http.Dir(gconfig.StaticAssets))
	mainRtr.PathPrefix("/static").Handler(fileHandler)
	mainRtr.PathPrefix("/").Handler(fileHandler)

	// Negroni middleware
	n := negroni.New()
	n.Use(negroni.HandlerFunc(HTTPLogger))
	n.UseHandler(mainRtr)

	if certrsc != nil {
		secureListen(n, certrsc.Value)
	} else {
		initListen(n)
	}

}
