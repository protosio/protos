package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/capability"

	"github.com/gorilla/mux"
	"github.com/urfave/negroni"
)

type key int

const (
	appKey key = iota
)

func checkCapability(cm *capability.Manager, capChecker capability.Checker, routeName string) error {
	methodcap, err := cm.GetMethodCap(routeName)
	if err != nil {
		log.Warn(err.Error())
		return nil
	}
	log.Tracef("Required capability for route '%s' is '%s'", routeName, methodcap.GetName())
	err = capChecker.ValidateCapability(methodcap)
	if err != nil {
		return err
	}
	return nil
}

// HTTPLogger is a http middleware that logs requests
func HTTPLogger(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	start := time.Now()

	next(w, r)

	log.Tracef(
		"HTTP|%s|%s -\t%s",
		r.Method,
		time.Since(start),
		r.RequestURI,
	)
}

// InternalRequestValidator validates requests coming from the containers (correct IP and AppID)
func InternalRequestValidator(ha handlerAccess, router *mux.Router) negroni.HandlerFunc {
	return negroni.HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {

		appID := r.Header.Get("Appid")
		if appID == "" {
			log.Debugf("Can't identify request to '%s'. App ID is missing.", r.URL)
			rend.JSON(rw, http.StatusUnauthorized, httperr{Error: "Can't identify request. App ID is missing."})
			return
		}
		appInstance, err := ha.am.GetByID(appID)
		if err != nil {
			log.Errorf("Internal request to '%s' from non-existent app '%s': %s", r.URL, appID, err.Error())
			rend.JSON(rw, http.StatusUnauthorized, httperr{Error: "Request for resource from non-existent app"})
			return
		}
		ip := strings.Split(r.RemoteAddr, ":")[0]
		if appInstance.GetIP() != ip {
			log.Errorf("App IP mismatch for request to '%s': ip '%s' incorrect for app '%s'", r.URL, ip, appID)
			rend.JSON(rw, http.StatusUnauthorized, httperr{Error: "App IP mismatch"})
			return
		}
		log.Debugf("Validated '%s' request to '%s' as coming from app '%s'(%s)", r.Method, r.URL.Path, appID, appInstance.GetName())

		rmatch := &mux.RouteMatch{}
		router.Match(r, rmatch)
		err = checkCapability(ha.cm, appInstance, rmatch.Route.GetName())
		if err != nil {
			log.Error(err.Error())
			rend.JSON(rw, http.StatusUnauthorized, httperr{Error: "Application not authorized to access that resource"})
			return
		}

		ctx := context.WithValue(r.Context(), appKey, appInstance)
		next.ServeHTTP(rw, r.WithContext(ctx))
	})
}

// ExternalRequestValidator validates client request. It checks if request contains a valid session and if the user is
// authorized to access the resource
func ExternalRequestValidator(ha handlerAccess, router *mux.Router, initMode bool) negroni.HandlerFunc {
	return negroni.HandlerFunc(func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		var httpErrStatus int
		if initMode {
			httpErrStatus = http.StatusFailedDependency
		} else {
			httpErrStatus = http.StatusUnauthorized
		}

		session, err := ha.cs.Get(r, "session-auth")
		if err != nil {
			err := errors.Wrap(err, "Failed to validate session")
			log.Error(err.Error())
			if strings.Contains(err.Error(), "the value is not valid") {
				rend.JSON(rw, httpErrStatus, httperr{Error: "User not authorized to access that resource"})
				return
			}
			rend.JSON(rw, http.StatusInternalServerError, httperr{Error: err.Error()})
			return

		}

		user, err := getUser(session, ha.um)
		if err != nil {
			log.Error(err.Error())
			err = session.Save(r, rw)
			if err != nil {
				log.Error(err.Error())
				rend.JSON(rw, http.StatusInternalServerError, httperr{Error: err.Error()})
				return
			}
			rend.JSON(rw, httpErrStatus, httperr{Error: "User not authorized to access that resource"})
			return
		}

		//
		// Checks if a user is authorized to access a specific route
		//

		rmatch := &mux.RouteMatch{}
		router.Match(r, rmatch)
		err = checkCapability(ha.cm, user, rmatch.Route.GetName())
		if err != nil {
			log.Error(err.Error())
			rend.JSON(rw, httpErrStatus, httperr{Error: "User not authorized to access that resource"})
			return
		}

		next.ServeHTTP(rw, r)
	})
}
