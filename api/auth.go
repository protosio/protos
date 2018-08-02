package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/nustiueudinastea/protos/auth"
	"github.com/nustiueudinastea/protos/meta"
)

// RegisterHandler is used in the initial user and domain registration. Should be disabled after the initial setup
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
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
	log.Debug(registerform)

	if registerform.Password != registerform.ConfirmPassword {
		err = errors.New("Passwords don't match")
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	if registerform.Domain == "" {
		err = errors.New("Domain cannot be empty")
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	err = meta.Setup(registerform.Domain)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusInternalServerError, httperr{Error: err.Error()})
		return
	}

	user, err := auth.CreateUser(registerform.Username, registerform.Password, registerform.Name, true)
	if err != nil {
		log.Error(err)
		rend.JSON(w, http.StatusBadRequest, httperr{Error: err.Error()})
		return
	}

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
	rend.JSON(w, http.StatusOK, tokenResponse)
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

	tokenResponse := struct {
		Token    string `json:"token"`
		Username string `json:"username"`
	}{
		Token:    tokenString,
		Username: user.Username,
	}

	log.Debug("Sending response: ", tokenResponse)
	rend.JSON(w, http.StatusOK, tokenResponse)

}
