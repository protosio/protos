package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	"github.com/gorilla/mux"
	"github.com/nustiueudinastea/protos/auth"
	"github.com/nustiueudinastea/protos/capability"
	"github.com/nustiueudinastea/protos/daemon"
)

func checkCapability(capChecker capability.Checker, routeName string) error {
	methodcap, err := capability.GetMethodCap(routeName)
	if err != nil {
		log.Warn(err.Error())
		return nil
	}
	log.Debugf("Required capability for route %s is %s", routeName, methodcap.Name)
	err = capChecker.ValidateCapability(methodcap)
	if err != nil {
		return err
	}
	return nil
}

func createToken() (string, string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := make(jwt.MapClaims)
	claims["exp"] = time.Now().Add(time.Hour * time.Duration(3)).Unix()
	claims["iat"] = time.Now().Unix()
	token.Claims = claims

	tokenString, err := token.SignedString(gconfig.Secret)
	if err != nil {
		return "", "", err
	}
	sstring, err := token.SigningString()
	if err != nil {
		return "", "", err
	}
	return tokenString, sstring, nil
}

// HTTPLogger is a http middleware that logs requests
func HTTPLogger(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	start := time.Now()

	next(w, r)

	log.Debugf(
		"HTTP|%s|%s -\t%s",
		r.Method,
		time.Since(start),
		r.RequestURI,
	)
}

// InternalRequestValidator validates requests coming from the containers (correct IP and AppID)
func InternalRequestValidator(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {

	appID := r.Header.Get("Appid")
	if appID == "" {
		log.Debugf("Can't identify request to resource %s. App ID is missing.", r.URL)
		rend.JSON(rw, http.StatusUnauthorized, httperr{Error: "Can't identify request. App ID is missing."})
		return
	}
	app, err := daemon.ReadApp(appID)
	if err != nil {
		log.Errorf("Request for resource %s from non-existent app %s: %s", r.URL, appID, err.Error())
		rend.JSON(rw, http.StatusUnauthorized, httperr{Error: "Request for resource from non-existent app"})
		return
	}
	ip := strings.Split(r.RemoteAddr, ":")[0]
	if app.IP != ip {
		log.Errorf("App IP mismatch for request for resource %s: ip %s incorrect for %s", r.URL, ip, appID)
		rend.JSON(rw, http.StatusUnauthorized, httperr{Error: "App IP mismatch"})
		return
	}
	log.Debugf("Validated %s request to %s as coming from app %s(%s)", r.Method, r.URL.Path, appID, app.Name)

	routeName := mux.CurrentRoute(r).GetName()
	err = checkCapability(app, routeName)
	if err != nil {
		log.Error(err.Error())
		http.Error(rw, "Application not authorized to access that resource", http.StatusUnauthorized)
		return
	}

	ctx := context.WithValue(r.Context(), "app", app)
	next.ServeHTTP(rw, r.WithContext(ctx))
}

// ExternalRequestValidator validates client request contains a valid JWT token
func ExternalRequestValidator(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {

	token, err := request.ParseFromRequest(r, request.AuthorizationHeaderExtractor,
		func(token *jwt.Token) (interface{}, error) {
			return gconfig.Secret, nil
		})
	if err != nil {
		log.Errorf("Unauthorized access to resource %s with Error: %s", r.URL, err.Error())
		rend.JSON(rw, http.StatusUnauthorized, httperr{Error: "Unauthorized access to this resource"})
		return
	}

	if token.Valid == false {
		log.Error("Token is not valid")
		rend.JSON(rw, http.StatusUnauthorized, httperr{Error: "Token is not valid"})
		return
	}

	sstring, err := token.SigningString()
	if err != nil {
		log.Error(err)
		rend.JSON(rw, http.StatusUnauthorized, httperr{Error: err.Error()})
		return
	}

	user, err := auth.GetUserForToken(sstring)
	if err != nil {
		log.Error(err)
		rend.JSON(rw, http.StatusUnauthorized, httperr{Error: err.Error()})
		return
	}

	routeName := mux.CurrentRoute(r).GetName()
	err = checkCapability(user, routeName)
	if err != nil {
		log.Error(err.Error())
		rend.JSON(rw, http.StatusUnauthorized, httperr{Error: "User not authorized to access that resource"})
		return
	}

	next.ServeHTTP(rw, r)
}
