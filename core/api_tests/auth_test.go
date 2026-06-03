package main

// import (
// 	"testing"

// 	"github.com/getlantern/deepcopy"

// 	baloo "gopkg.in/h2non/baloo.v3"
// )

// var test = baloo.New("http://localhost:8080")
// var registerURL = "/api/v1/auth/register"
// var loginURL = "/api/v1/auth/login"

// var registerUserForm = map[string]string{
// 	"username":        "testuser",
// 	"name":            "First Last",
// 	"password":        "testpass12",
// 	"confirmpassword": "testpass12",
// 	"domain":          "test.com",
// }

// const userResponseSchema = `{
// 	"title": "Username Register Response",
// 	"type": "object",
// 	"properties": {
// 	  "username": {
// 		"type": "string",
// 		"enum": ["testuser"]
// 	  },
// 	  "token": {
// 		"type": "string"
// 	  }
// 	},
// 	"required": ["username", "token"]
// }`

// func TestCreateUserPasswordMismatch(t *testing.T) {
// 	invalidUserForm := map[string]string{}
// 	deepcopy.Copy(invalidUserForm, registerUserForm)
// 	invalidUserForm["password"] = "123"
// 	test.Post(registerURL).
// 		JSON(invalidUserForm).
// 		Expect(t).
// 		Status(400).
// 		BodyMatchString("passwords don't match").
// 		Done()
// }

// func TestCreateUserPasswordTooShort(t *testing.T) {
// 	invalidUserForm := map[string]string{}
// 	deepcopy.Copy(&invalidUserForm, registerUserForm)
// 	invalidUserForm["password"] = "123"
// 	invalidUserForm["confirmpassword"] = "123"
// 	test.Post(registerURL).
// 		JSON(invalidUserForm).
// 		Expect(t).
// 		Status(400).
// 		BodyMatchString("Minimum password").
// 		Done()
// }

// func TestCreateUserPasswordInvalidDomain(t *testing.T) {
// 	invalidUserForm := map[string]string{}
// 	deepcopy.Copy(&invalidUserForm, registerUserForm)
// 	invalidUserForm["domain"] = ""
// 	test.Post(registerURL).
// 		JSON(invalidUserForm).
// 		Expect(t).
// 		Status(400).
// 		BodyMatchString("domain cannot be empty").
// 		Done()
// }

// func TestCreateUserSuccess(t *testing.T) {
// 	test.Post(registerURL).
// 		JSON(registerUserForm).
// 		Expect(t).
// 		Status(200).
// 		Type("json").
// 		JSONSchema(userResponseSchema).
// 		Done()
// }

// const loginResponseSchema = `{
// 	"title": "Username Register Response",
// 	"type": "object",
// 	"properties": {
// 	  "username": {
// 			"type": "string",
// 			"enum": ["testuser"]
// 	  },
// 	  "token": {
// 			"type": "string"
// 		},
// 		"role": {
// 			"type": "string",
// 			"enum": ["admin"]
// 		}
// 	},
// 	"required": ["username", "token", "role"]
// }`

// func TestLoginInvalidPassword(t *testing.T) {
// 	test.Post(loginURL).
// 		JSON(map[string]string{"username": "testuser", "password": "wrongpass"}).
// 		Expect(t).
// 		Status(403).
// 		BodyMatchString("Invalid credentials").
// 		Done()
// }

// func TestLoginSuccess(t *testing.T) {
// 	test.Post(loginURL).
// 		JSON(map[string]string{"username": "testuser", "password": "testpass12"}).
// 		Expect(t).
// 		Status(200).
// 		Type("json").
// 		JSONSchema(loginResponseSchema).
// 		Done()
// }
