package api

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/protosio/protos/internal/app"
	"github.com/protosio/protos/internal/auth"
	"github.com/protosio/protos/internal/capability"
	"github.com/protosio/protos/internal/installer"
	"github.com/protosio/protos/internal/meta"
	"github.com/protosio/protos/internal/platform"
	"github.com/protosio/protos/internal/provider"
	"github.com/protosio/protos/internal/resource"
	"github.com/protosio/protos/internal/task"
	"github.com/protosio/protos/pkg/types"

	"github.com/pkg/errors"

	// statik package is use to embed static web assets in the protos binary
	_ "github.com/protosio/protos/internal/statik"

	"github.com/protosio/protos/internal/util"

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/rakyll/statik/fs"
	"github.com/unrolled/render"
	"github.com/urfave/negroni"
)

var log = util.GetLogger("api")
var rend = render.New(render.Options{IndentJSON: true})

type httperr struct {
	Error string `json:"error"`
}

type route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc func(handlerAccess) http.Handler
	Capability  *capability.Capability
}

type apiController interface {
	StartExternalWebServer() (func() error, error)
	DisableInitRoutes() error
}

type handlerAccess struct {
	pm  *provider.Manager
	rm  *resource.Manager
	am  *app.Manager
	tm  *task.Manager
	m   *meta.Meta
	as  *installer.AppStore
	um  *auth.UserManager
	rp  platform.RuntimePlatform
	cm  *capability.Manager
	cs  *sessions.CookieStore
	api apiController
}

type certificate interface {
	GetCertificate() []byte
	GetPrivateKey() []byte
}

type routerSwapper struct {
	mu   sync.Mutex
	root *mux.Router
}

func (rs *routerSwapper) Swap(newRouter *mux.Router) {
	rs.mu.Lock()
	rs.root = newRouter
	rs.mu.Unlock()
}

func (rs *routerSwapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rs.mu.Lock()
	root := rs.root
	rs.mu.Unlock()
	root.ServeHTTP(w, r)
}

type routes []route

func uiHandler(staticAssetsPath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, string(http.Dir(staticAssetsPath))+"/index.html")
	})
}

func uiRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/ui/", 303)
}

func addRoutesToRouter(ha handlerAccess, r *mux.Router, routes []route) *mux.Router {
	for _, route := range routes {
		if route.Method != "" {
			// if route method is set (GET, POST etc), the route is only valid for that method
			r.Methods(route.Method).Path(route.Pattern).Name(route.Name).Handler(route.HandlerFunc(ha))
		} else {
			// if route method is not set, it will work for all methods. Useful for WS
			r.Path(route.Pattern).Name(route.Name).Handler(route.HandlerFunc(ha))
		}
		if route.Capability != nil {
			ha.cm.SetMethodCap(route.Name, route.Capability)
		}
	}
	return r
}

func applyStaticRoutes(r *mux.Router, staticAssetsPath string) {
	// UI routes
	var fileHandler http.Handler
	if staticAssetsPath != "" {
		log.Debugf("Running webserver with static assets from %s", staticAssetsPath)
		fileHandler = http.FileServer(http.Dir(staticAssetsPath))
		r.PathPrefix("/ui/").Name("ui").Handler(uiHandler(staticAssetsPath))
	} else {
		statikFS, err := fs.New()
		if err != nil {
			log.Fatal(err)
		}
		log.Debug("Running webserver with embedded static assets")
		fileHandler = http.FileServer(statikFS)
		file, err := statikFS.Open("/index.html")
		if err != nil {
			log.Fatal(errors.Wrap(err, "Failed to open the embedded index.html file"))
		}
		r.PathPrefix("/ui/").Name("ui").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeContent(w, r, "index.html", time.Now(), file)
		}))
	}
	r.PathPrefix("/static/").Name("static").Handler(http.StripPrefix("/static/", fileHandler))
	r.PathPrefix("/").Name("root").Handler(http.HandlerFunc(uiRedirect))
}

func secureListen(handler http.Handler, certrsc resource.ResourceValue, quit chan bool, httpPort int, httpsPort int) {
	cert, ok := certrsc.(certificate)
	if ok == false {
		log.Fatal("Failed to read TLS certificate")
	}
	tlscert, err := tls.X509KeyPair(cert.GetCertificate(), cert.GetPrivateKey())
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

	httpsport := strconv.Itoa(httpsPort)
	srv := &http.Server{
		Addr:         ":" + httpsport,
		Handler:      handler,
		TLSConfig:    cfg,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	// holds all the internal web servers
	internalSrvs := []*http.Server{}

	log.Infof("Starting HTTPS webserver on '%s'", srv.Addr)
	go func() {
		if err := srv.ListenAndServeTLS("", ""); err != nil {
			if strings.Contains(err.Error(), "Server closed") {
				log.Info("HTTPS API webserver terminated successfully")
			} else {
				log.Errorf("HTTPS API webserver died with error: %s", err.Error())
			}
		}
	}()

	<-quit
	log.Info("Shutting down HTTPS webserver")
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Error(errors.Wrap(err, "Something went wrong while shutting down the HTTPS webserver"))
	}

	for _, isrv := range internalSrvs {
		if err := isrv.Shutdown(context.Background()); err != nil {
			log.Error(errors.Wrap(err, "Something went wrong while shutting down the internal API webserver"))
		}
	}
}

func insecureListen(handler http.Handler, quit chan bool, httpPort int, listenAddress string) {
	httpport := strconv.Itoa(httpPort)
	srv := &http.Server{
		Addr:           listenAddress + ":" + httpport,
		Handler:        handler,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Infof("Starting HTTP webserver on '%s'", srv.Addr)
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			if strings.Contains(err.Error(), "Server closed") {
				log.Info("HTTP webserver terminated successfully")
			} else {
				log.Errorf("HTTP webserver died with error: %s", err.Error())
			}
		}
	}()
	log.Infof("HTTP webserver started")

	<-quit
	log.Infof("Shutting down HTTP '%s' webserver", srv.Addr)
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Error(errors.Wrap(err, "Something went wrong while shutting down the HTTP webserver"))
	}
}

// HTTP is the http API
type HTTP struct {
	staticAssetsPath      string
	localWebServerQuit    chan bool
	internalWebServerQuit chan bool
	externalWebServerQuit chan bool
	wsManagerQuit         chan bool
	router                *routerSwapper
	root                  *negroni.Negroni
	ha                    handlerAccess
	devmode               bool
	httpPort              int
	httpsPort             int
	wsfrontend            chan interface{}
}

func createRouter(httpAPI *HTTP, devmode bool, initmode bool, staticAssetsPath string) *mux.Router {
	//
	// Setting up routes
	//

	rtr := mux.NewRouter().StrictSlash(true)

	// auth routes
	authRouter := rtr.PathPrefix(types.APIAuthPath).Subrouter().StrictSlash(true)
	addRoutesToRouter(httpAPI.ha, authRouter, createAuthRoutes(httpAPI.ha.cm))
	// init routes
	if initmode {
		addRoutesToRouter(httpAPI.ha, authRouter, routes{initRoute})
	}

	// internal routes
	internalRouter := mux.NewRouter().PathPrefix(types.APIInternalPath).Subrouter().StrictSlash(true)
	addRoutesToRouter(httpAPI.ha, internalRouter, createInternalRoutes(httpAPI.ha.cm))
	addRoutesToRouter(httpAPI.ha, internalRouter, internalWSRoutes)
	rtr.PathPrefix(types.APIInternalPath).Handler(negroni.New(
		InternalRequestValidator(httpAPI.ha, internalRouter),
		negroni.Wrap(internalRouter),
	))

	// // external routes
	// externalRouter := mux.NewRouter().PathPrefix(types.APIExternalPath).Subrouter().StrictSlash(true)
	// addRoutesToRouter(httpAPI.ha, externalRouter, createExternalRoutes(httpAPI.ha.cm))
	// addRoutesToRouter(httpAPI.ha, externalRouter, externalWSRoutes)
	// rtr.PathPrefix(types.APIExternalPath).Handler(negroni.New(
	// 	ExternalRequestValidator(httpAPI.ha, externalRouter, initmode),
	// 	negroni.Wrap(externalRouter),
	// ))

	// // init routes
	// if initmode {
	// 	addRoutesToRouter(httpAPI.ha, externalRouter, createExternalInitRoutes(httpAPI.ha.cm))
	// }

	// // if dev mode is enabled we add the dev routes
	// if devmode {
	// 	devRouter := rtr.PathPrefix("/api/v1/dev").Subrouter().StrictSlash(true)
	// 	addRoutesToRouter(httpAPI.ha, devRouter, externalDevRoutes)
	// }

	// static file routes
	applyStaticRoutes(rtr, staticAssetsPath)

	return rtr
}

// New returns a new http API
func New(devmode bool, staticAssetsPath string, wsfrontend chan interface{}, httpPort int, httpsPort int, m *meta.Meta, am *app.Manager, rm *resource.Manager, tm *task.Manager, pm *provider.Manager, as *installer.AppStore, um *auth.UserManager, rp platform.RuntimePlatform, cm *capability.Manager) *HTTP {
	httpAPI := &HTTP{
		devmode:          devmode,
		staticAssetsPath: staticAssetsPath,
		wsfrontend:       wsfrontend,
		httpPort:         httpPort,
		httpsPort:        httpsPort,
		wsManagerQuit:    make(chan bool, 1),
	}
	httpAPI.ha = handlerAccess{
		pm:  pm,
		rm:  rm,
		am:  am,
		tm:  tm,
		m:   m,
		as:  as,
		um:  um,
		rp:  rp,
		cm:  cm,
		api: httpAPI,
	}

	if httpAPI.ha.pm == nil ||
		httpAPI.ha.rm == nil ||
		httpAPI.ha.am == nil ||
		httpAPI.ha.tm == nil ||
		httpAPI.ha.m == nil ||
		httpAPI.ha.as == nil ||
		httpAPI.ha.um == nil ||
		httpAPI.ha.rp == nil ||
		httpAPI.ha.cm == nil {
		log.Panic("Failed to create web server: none of the inputs can be nil")
	}

	//
	// Setting up session cookies
	//

	authKeyOne := securecookie.GenerateRandomKey(64)
	encryptionKeyOne := securecookie.GenerateRandomKey(32)

	httpAPI.ha.cs = sessions.NewCookieStore(
		authKeyOne,
		encryptionKeyOne,
	)

	httpAPI.ha.cs.Options = &sessions.Options{
		Path:   "/",
		MaxAge: 60 * 15,
	}

	return httpAPI
}

// StartLoopbackWebServer starts the HTTP API, used via a SSH connection to the loopback interface
func (api *HTTP) StartLoopbackWebServer(initMode bool) (func() error, error) {
	rtr := createRouter(api, api.devmode, initMode, api.staticAssetsPath)

	api.localWebServerQuit = make(chan bool, 1)
	// Negroni middleware
	api.root = negroni.New()
	rtrSwapper := &routerSwapper{root: rtr}
	api.router = rtrSwapper
	api.root.Use(negroni.HandlerFunc(HTTPLogger))
	api.root.UseHandler(rtrSwapper)

	go insecureListen(api.root, api.localWebServerQuit, api.httpPort, "127.0.0.1")

	stopper := func() error {
		api.localWebServerQuit <- true
		return nil
	}

	return stopper, nil
}

// StopLoopbackWebServer stops the HTTP API
func (api *HTTP) StopLoopbackWebServer() error {
	api.localWebServerQuit <- true
	return nil
}

// StartInternalWebServer starts the HTTP API, used for initilisation
func (api *HTTP) StartInternalWebServer(initMode bool, internalIP string) (func() error, error) {
	rtr := createRouter(api, api.devmode, initMode, api.staticAssetsPath)

	api.internalWebServerQuit = make(chan bool, 1)
	// Negroni middleware
	api.root = negroni.New()
	rtrSwapper := &routerSwapper{root: rtr}
	api.router = rtrSwapper
	api.root.Use(negroni.HandlerFunc(HTTPLogger))
	api.root.UseHandler(rtrSwapper)

	go insecureListen(api.root, api.internalWebServerQuit, api.httpPort, internalIP)

	stopper := func() error {
		return api.StopInternalWebServer()
	}

	return stopper, nil
}

// StopInternalWebServer stops the HTTP API
func (api *HTTP) StopInternalWebServer() error {
	log.Debug("Shutting down internal web server")
	api.internalWebServerQuit <- true
	return nil
}

// DisableInitRoutes disables the routes used during the init process
func (api *HTTP) DisableInitRoutes() error {
	log.Info("Disabling the init routes")
	newRtr := createRouter(api, api.devmode, false, api.staticAssetsPath)
	api.router.Swap(newRtr)
	return nil
}

// StartExternalWebServer starts the HTTPS API using the provided certificate
func (api *HTTP) StartExternalWebServer() (func() error, error) {
	rtr := createRouter(api, api.devmode, false, api.staticAssetsPath)

	api.externalWebServerQuit = make(chan bool, 1)
	// Negroni middleware
	api.root = negroni.New()
	api.root.Use(negroni.HandlerFunc(HTTPLogger))
	api.root.UseHandler(rtr)
	cert := api.ha.m.GetTLSCertificate()
	if cert == nil || cert.GetStatus() != resource.Created {
		return nil, fmt.Errorf("Failed to start secure web server. TLS certificate not available")
	}

	go secureListen(api.root, cert.GetValue(), api.externalWebServerQuit, api.httpPort, api.httpsPort)

	stopper := func() error {
		return api.StopExternalWebServer()
	}

	return stopper, nil
}

// StopExternalWebServer stops the HTTPS API
func (api *HTTP) StopExternalWebServer() error {
	log.Debug("Shutting down external web server")
	api.externalWebServerQuit <- true
	return nil
}

// StartWSManager starts the websocket server
func (api *HTTP) StartWSManager() (func() error, error) {
	go WSManager(api.ha.am, api.wsManagerQuit, api.wsfrontend)

	stopper := func() error {
		return api.StopWSManager()
	}
	return stopper, nil
}

// StopWSManager stops the websocket server
func (api *HTTP) StopWSManager() error {
	api.wsManagerQuit <- true
	return nil
}
