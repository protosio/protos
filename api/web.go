package api

import (
	"encoding/json"
	"net/http"
	"protos/daemon"
	"protos/util"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
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

	router := mux.NewRouter().StrictSlash(true)

	// Cllient routes (requires auth)
	for _, route := range clientRoutes {
		protectedRoute := ValidateTokenMiddleware(route.HandlerFunc)
		handler := util.HTTPLogger(protectedRoute, route.Name)
		router.Methods(route.Method).Path(route.Pattern).Name(route.Name).Handler(handler)
	}

	// Internal routes
	for _, route := range clientRoutes {
		handler := util.HTTPLogger(route.HandlerFunc, route.Name)
		router.Methods(route.Method).Path(route.Pattern).Name(route.Name).Handler(handler)
	}

	// Authentication route
	handler := util.HTTPLogger(http.HandlerFunc(LoginHandler), "login")
	router.Methods("POST").Path("/login").Name("login").Handler(handler)

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

// ValidateTokenMiddleware checks that the request contains a valid JWT token
func ValidateTokenMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		token, err := request.ParseFromRequest(r, request.AuthorizationHeaderExtractor,
			func(token *jwt.Token) (interface{}, error) {
				return daemon.Gconfig.Secret, nil
			})
		if err != nil {
			log.Debugf("Unauthorized access to resource %s with error: %s", r.URL, err.Error())
			http.Error(w, "Unauthorized access to this resource", http.StatusUnauthorized)
			return
		}

		if token.Valid == false {
			log.Debug("Token is not valid")
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

	user, err := daemon.ValidateAndGetUser(userform.Username, userform.Password)
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

	tokenString, err := token.SignedString(daemon.Gconfig.Secret)
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
