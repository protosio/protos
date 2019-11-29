package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/core"
)

// registerHandler is used in the initial user and domain registration. Should be disabled after the initial setup
func registerHandler(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var registerform struct {
			Username        string `json:"username"`
			Name            string `json:"name"`
			Password        string `json:"password"`
			ConfirmPassword string `json:"confirmpassword"`
			Domain          string `json:"domain"`
		}

		err := json.NewDecoder(r.Body).Decode(&registerform)
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		if registerform.Password != registerform.ConfirmPassword {
			err = errors.New("Cannot perform registration: passwords don't match")
			log.Error(err)
			rend.JSON(w, http.StatusBadRequest, httperr{Error: err.Error()})
			return
		}

		if registerform.Domain == "" {
			err = errors.New("Cannot perform registration: domain cannot be empty")
			log.Error(err)
			rend.JSON(w, http.StatusBadRequest, httperr{Error: err.Error()})
			return
		}

		ha.m.SetDomain(registerform.Domain)

		user, err := ha.um.CreateUser(registerform.Username, registerform.Password, registerform.Name, true)
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusBadRequest, httperr{Error: err.Error()})
			return
		}
		ha.m.SetAdminUser(user.GetUsername())

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

		registerResponse := struct {
			Username string `json:"username"`
		}{
			Username: user.GetUsername(),
		}

		log.Trace("Sending response: ", registerResponse)
		rend.JSON(w, http.StatusOK, registerResponse)
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
