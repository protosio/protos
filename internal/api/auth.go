package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	validator "github.com/go-playground/validator/v10"
	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/core"
	"github.com/protosio/protos/pkg/types"
)

var initRoute = route{
	"init",
	"POST",
	"/init",
	initHandler,
	nil,
}

var createAuthRoutes = func(cm core.CapabilityManager) routes {
	return routes{
		route{
			"login",
			"POST",
			"/login",
			loginHandler,
			nil,
		},
		route{
			"logout",
			"POST",
			"/logout",
			logoutHandler,
			nil,
		},
	}
}

// initHandler is used in the initial user and domain registration. Should be disabled after the initial setup
func initHandler(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var initform types.ReqInit
		err := json.NewDecoder(r.Body).Decode(&initform)
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		validate := validator.New()
		err = validate.Struct(initform)
		if err != nil {
			err = fmt.Errorf("Failed to validate init request: %w", err)
			log.Error(err)
			rend.JSON(w, http.StatusBadRequest, httperr{Error: err.Error()})
			return
		}

		_, network, err := net.ParseCIDR(initform.Network)
		if err != nil {
			err = fmt.Errorf("Cannot perform initialization, network '%s' is invalid: %w", initform.Network, err)
			log.Error(err)
			rend.JSON(w, http.StatusBadRequest, httperr{Error: err.Error()})
			return
		}

		ha.m.SetDomain(initform.Domain)
		ip := ha.m.SetNetwork(*network)

		user, err := ha.um.CreateUser(initform.Username, initform.Password, initform.Name, true, initform.Devices)
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusBadRequest, httperr{Error: err.Error()})
			return
		}
		ha.m.SetAdminUser(user.GetUsername())

		// perform init
		err = ha.rp.Init(*network, initform.Devices)
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusBadRequest, httperr{Error: err.Error()})
			return
		}

		// create session and add user to it

		session, err := ha.cs.Get(r, "session-auth")
		if err != nil && strings.Contains(err.Error(), "the value is not valid") != true {
			err := errors.Wrap(err, "Failed to retrieve session")
			log.Error(err.Error())
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		session.Values["user"] = user

		err = ha.cs.Save(r, w, session)
		if err != nil {
			erru := errors.Wrap(err, "Failed to save session")
			log.Error(erru.Error())
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: erru.Error()})
			return
		}

		initResponse := types.RespInit{
			InstanceIP:    ip.String(),
			InstacePubKey: ha.m.GetKey().PublicKey().String(),
		}

		log.Trace("Sending response: ", initResponse)
		rend.JSON(w, http.StatusOK, initResponse)
	})
}

// loginHandler takes a JSON payload containing a username and password, and creates a session if the user/pass combination is valid
func loginHandler(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := ha.cs.Get(r, "session-auth")
		if err != nil && strings.Contains(err.Error(), "the value is not valid") != true {
			err := errors.Wrap(err, "Failed to retrieve session")
			log.Error(err.Error())
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		var userform struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		err = json.NewDecoder(r.Body).Decode(&userform)
		if err != nil {
			err := errors.Wrap(err, "Failed to decode login form")
			log.Error(err.Error())
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		user, err := ha.um.ValidateAndGetUser(userform.Username, userform.Password)
		if err != nil {
			log.Debug(err)
			rend.JSON(w, http.StatusForbidden, httperr{Error: err.Error()})
			return
		}

		session.Values["user"] = user

		err = ha.cs.Save(r, w, session)
		if err != nil {
			erru := errors.Wrap(err, "Failed to save session")
			log.Error(erru.Error())
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: erru.Error()})
			return
		}

		var role string
		if user.IsAdmin() {
			role = "admin"
		} else {
			role = "user"
		}
		loginResponse := struct {
			Username string `json:"username"`
			Role     string `json:"role"`
		}{
			Username: user.GetUsername(),
			Role:     role,
		}

		log.Trace("Sending response: ", loginResponse)
		rend.JSON(w, http.StatusOK, loginResponse)

	})
}

// logoutandler logs out a user from a session
func logoutHandler(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := ha.cs.Get(r, "session-auth")
		if err != nil && strings.Contains(err.Error(), "the value is not valid") != true {
			err := errors.Wrap(err, "Failed to retrieve session")
			log.Error(err.Error())
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		session.Values["user"] = nil
		session.Options.MaxAge = -1

		err = ha.cs.Save(r, w, session)
		if err != nil {
			erru := errors.Wrap(err, "Failed to save session")
			log.Error(erru.Error())
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: erru.Error()})
			return
		}

		rend.JSON(w, http.StatusOK, struct{}{})
	})
}

func getUser(s *sessions.Session, um core.UserManager) (core.User, error) {
	val := s.Values["user"]
	user, ok := val.(core.User)
	if !ok {
		return nil, errors.New("Failed to get user for session")
	}
	usr, err := um.SetParent(user)
	if err != nil {
		return nil, err
	}
	return usr, nil
}
