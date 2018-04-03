package auth

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/nustiueudinastea/protos/capability"

	"github.com/nustiueudinastea/protos/database"
	"github.com/nustiueudinastea/protos/util"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	userBucket = "user"
)

var log = util.Log

// User represents a Protos user
type User struct {
	Username     string   `json:"username" storm:"id"`
	Password     string   `json:"password"`
	Name         string   `json:"name"`
	IsDisabled   bool     `json:"isdisabled"`
	Capabilities []string `json:"capabilities"`
}

// UserInfo holds information about a user that is meant to be returned to external applications or the web interface
type UserInfo struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	IsAdmin  bool   `json:"isadmin"`
}

var usersTokens = map[string]*User{}

// readCredentials reads a username and password interactively
func readCredentials() (string, string, string) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print("Enter Password: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatal(err)
	}
	password := string(bytePassword)

	fmt.Print("Enter Name: ")
	name, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	return strings.TrimSpace(username), strings.TrimSpace(password), strings.TrimSpace(name)
}

// generatePasswordHash takes a string representing the raw password, and generates a hash
func generatePasswordHash(password string) (string, error) {

	if len([]rune(password)) < 10 {
		return "", errors.New("Minimum password length is 10 characters")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashedPassword), nil
}

//
// User instance methods
//

// Save saves the User struct to the database. The username is used as an unique key
func (user *User) Save() error {
	log.Debugf("Writing username %s to database", user.Username)
	return database.Save(user)
}

// AddToken associates a JWT token with a username
func (user *User) AddToken(token string) {
	for tkn, usr := range usersTokens {
		if usr.Username == user.Username {
			log.Debugf("Removing old token %s for user %s", tkn, usr.Username)
			delete(usersTokens, tkn)
		}
	}
	log.Debugf("Adding token %s for username %s", token, user.Username)
	usersTokens[token] = user
}

// ValidateCapability implements the capability checker interface
func (user *User) ValidateCapability(cap *capability.Capability) error {
	for _, usercap := range user.Capabilities {
		if capability.Validate(cap, usercap) {
			return nil
		}
	}
	return errors.New("Method capability " + cap.Name + " not satisfied by user " + user.Username)
}

// IsAdmin checks if a user is an admin or not
func (user *User) IsAdmin() bool {
	if user.ValidateCapability(capability.UserAdmin) != nil {
		return false
	}
	return true
}

// GetInfo returns public information about a user
func (user *User) GetInfo() UserInfo {
	return UserInfo{
		Username: user.Username,
		Name:     user.Name,
		IsAdmin:  user.IsAdmin(),
	}
}

//
// Public package methods
//

// CreateUser creates and returns a user
func CreateUser(username string, password string, name string, isadmin bool) (*User, error) {

	passwordHash, err := generatePasswordHash(password)
	if err != nil {
		return nil, err
	}

	user := User{
		Username:     username,
		Password:     passwordHash,
		Name:         name,
		IsDisabled:   false,
		Capabilities: []string{},
	}
	if isadmin {
		user.Capabilities = append(user.Capabilities, capability.UserAdmin.Name)
	}

	return &user, user.Save()
}

// ValidateAndGetUser takes a username and password and returns the full User struct if credentials are valid
func ValidateAndGetUser(username string, password string) (*User, error) {
	log.Debugf("Searching for username %s", username)

	errInvalid := errors.New("Invalid credentials")
	var user User
	err := database.One("Username", username, &user)
	if err != nil {
		log.Debugf("Can't find user %s (%s)", username, err)
		return nil, errInvalid
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		log.Debugf("Invalid password for user %s", username)
		return nil, errInvalid
	}

	if user.IsDisabled {
		log.Debugf("User %s is disabled", username)
		return nil, errInvalid
	}

	log.Debugf("User %s logged in successfuly", username)
	return &user, nil
}

// GetUser returns a username for a specific token
func GetUser(token string) (*User, error) {
	if usr, ok := usersTokens[token]; ok {
		return usr, nil
	}
	return nil, errors.New("No user found for token " + token)
}

// SetupAdmin creates and initial admin user
func SetupAdmin() {
	username, clearpassword, name := readCredentials()
	user, err := CreateUser(username, clearpassword, name, true)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("User %s has been created.", user.Username)
}
