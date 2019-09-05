package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"protos/internal/auth"
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

		user, err := auth.CreateUser(registerform.Username, registerform.Password, registerform.Name, true)
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusBadRequest, httperr{Error: err.Error()})
			return
		}
		ha.m.SetAdminUser(user.Username)

		tokenString, sstring, err := createToken()
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		user.AddToken(sstring)

		tokenResponse := struct {
			Token    string `json:"token"`
			Username string `json:"username"`
		}{
			Token:    tokenString,
			Username: user.Username,
		}

		log.Debug("Sending response: ", tokenResponse)
		w.Header().Add("Content-Type", "application/json")
		rend.JSON(w, http.StatusOK, tokenResponse)
	})
}

// loginHandler takes a JSON payload containing a username and password, and returns a JWT if they are valid
func loginHandler(ha handlerAccess) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var userform struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		err := json.NewDecoder(r.Body).Decode(&userform)
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		user, err := auth.ValidateAndGetUser(userform.Username, userform.Password)
		if err != nil {
			log.Debug(err)
			rend.JSON(w, http.StatusForbidden, httperr{Error: err.Error()})
			return
		}

		tokenString, sstring, err := createToken()
		if err != nil {
			log.Error(err)
			rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
			return
		}

		user.AddToken(sstring)

		var role string
		if user.IsAdmin() {
			role = "admin"
		} else {
			role = "user"
		}
		tokenResponse := struct {
			Token    string `json:"token"`
			Username string `json:"username"`
			Role     string `json:"role"`
		}{
			Token:    tokenString,
			Username: user.Username,
			Role:     role,
		}

		log.Debug("Sending response: ", tokenResponse)
		rend.JSON(w, http.StatusOK, tokenResponse)

	})
}
