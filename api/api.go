package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"protos/auth"
	"protos/capability"
	"protos/config"
	"protos/daemon"
	"protos/util"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"github.com/gorilla/mux"
)

var log = util.Log
var gconfig = config.Gconfig

type route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
	Capability  *capability.Capability
}

type routes []route

func newRouter() *mux.Router {

	router := mux.NewRouter().StrictSlash(true)

	// Internal routes
	for _, route := range internalRoutes {
		protectedRoute := ValidateInternalRequest(route.HandlerFunc, router)
		handler := util.HTTPLogger(protectedRoute, route.Name)
		router.Methods(route.Method).Path(route.Pattern).Name(route.Name).Handler(handler)
		if route.Capability != nil {
			capability.SetMethodCap(route.Name, route.Capability)
		}
	}

	// Web routes (require auth)
	for _, route := range clientRoutes {
		protectedRoute := ValidateExternalRequest(route.HandlerFunc)
		handler := util.HTTPLogger(protectedRoute, route.Name)
		router.Methods(route.Method).Path(route.Pattern).Name(route.Name).Handler(handler)
	}

	// Authentication routes
	loginHdlr := util.HTTPLogger(http.HandlerFunc(LoginHandler), "login")
	router.Methods("POST").Path("/login").Name("login").Handler(loginHdlr)

	return router
}

// Websrv starts an HTTP server that exposes all the application functionality
func Websrv() {

	rtr := newRouter()

	fileHandler := http.FileServer(http.Dir(gconfig.StaticAssets))
	rtr.PathPrefix("/static").Handler(fileHandler)
	rtr.PathPrefix("/").Handler(fileHandler)

	http.Handle("/", rtr)

	port := strconv.Itoa(gconfig.Port)
	log.Info("Listening on port " + port)
	http.ListenAndServe(":"+port, nil)

}

// ValidateInternalRequest validates requests coming from the containers (correct IP and AppID)
func ValidateInternalRequest(next http.Handler, rtr *mux.Router) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		appID := r.Header.Get("Appid")
		if appID == "" {
			log.Debugf("Can't identify request to resource %s. App ID is missing.", r.URL)
			http.Error(w, "Can't identify request. App ID is missing.", http.StatusUnauthorized)
			return
		}
		app, err := daemon.ReadApp(appID)
		if err != nil {
			log.Errorf("Request for resource %s from non-existent app %s: %s", r.URL, appID, err.Error())
			http.Error(w, "Request for resource from non-existent app", http.StatusUnauthorized)
			return
		}
		ip := strings.Split(r.RemoteAddr, ":")[0]
		if app.IP != ip {
			log.Errorf("App IP mismatch for request for resource %s: ip %s incorrect for %s", r.URL, ip, appID)
			http.Error(w, "App IP mismatch", http.StatusUnauthorized)
			return
		}
		log.Debugf("Validated %s request to %s as coming from app %s(%s)", r.Method, r.URL.Path, appID, app.Name)

		err = checkCapability(&app, rtr, r)
		if err != nil {
			log.Error(err.Error())
			http.Error(w, "Application not authorized to access that resource", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "app", &app)
		next.ServeHTTP(w, r.WithContext(ctx))
	})

}

func checkCapability(app *daemon.App, rtr *mux.Router, r *http.Request) error {
	var match mux.RouteMatch
	if rtr.Match(r, &match) {
		methodcap, err := capability.GetMethodCap(match.Route.GetName())
		if err != nil {
			return err
		}
		log.Debugf("Required capability for route %s is %s", match.Route.GetName(), methodcap.Name)
		for _, appcap := range app.Capabilities {
			if capability.ValidateCapability(methodcap, appcap) {
				return nil
			}
		}
		return errors.New("Method capability " + methodcap.Name + " not satisfied by application " + app.ID)
	}

	return errors.New("Route not matched in capability check")
}

// ValidateExternalRequest validates client request contains a valid JWT token
func ValidateExternalRequest(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		token, err := request.ParseFromRequest(r, request.AuthorizationHeaderExtractor,
			func(token *jwt.Token) (interface{}, error) {
				return gconfig.Secret, nil
			})
		if err != nil {
			log.Errorf("Unauthorized access to resource %s with error: %s", r.URL, err.Error())
			http.Error(w, "Unauthorized access to this resource", http.StatusUnauthorized)
			return
		}

		if token.Valid == false {
			log.Error("Token is not valid")
			http.Error(w, "Token is not valid", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})

}

// LoginHandler takes a JSON payload containing a username and password, and returns a JWT if they are valid
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var userform struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&userform)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user, err := auth.ValidateAndGetUser(userform.Username, userform.Password)
	if err != nil {
		log.Debug(err)
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	token := jwt.New(jwt.SigningMethodHS256)
	claims := make(jwt.MapClaims)
	claims["admin"] = user.IsAdmin
	claims["exp"] = time.Now().Add(time.Hour * time.Duration(1)).Unix()
	claims["iat"] = time.Now().Unix()
	token.Claims = claims

	tokenString, err := token.SignedString(gconfig.Secret)
	if err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tokenResponse := struct {
		Token    string `json:"token"`
		Username string `json:"username"`
	}{
		Token:    tokenString,
		Username: user.Username,
	}

	log.Debug("Sending response: ", tokenResponse)
	json.NewEncoder(w).Encode(tokenResponse)

}
