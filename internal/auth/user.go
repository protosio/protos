package auth

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/protosio/protos/internal/core"

	"github.com/protosio/protos/internal/util"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	userBucket = "user"
)

var log = util.GetLogger("auth")

// User represents a Protos user
type User struct {
	Username     string   `json:"username" storm:"id"`
	Password     string   `json:"password"`
	Name         string   `json:"name"`
	IsDisabled   bool     `json:"isdisabled"`
	Capabilities []string `json:"capabilities"`
	parent       *UserManager
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

// GetUsername returns the username of the user in string format
func (user *User) GetUsername() string {
	return user.Username
}

// Save saves the User struct to the database. The username is used as an unique key
func (user *User) Save() error {
	log.Debugf("Writing username %s to database", user.Username)
	return user.parent.db.Save(user)
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
func (user *User) ValidateCapability(cap core.Capability) error {
	for _, usercap := range user.Capabilities {
		if user.parent.cm.Validate(cap, usercap) {
			return nil
		}
	}
	return errors.New("Method capability " + cap.GetName() + " not satisfied by user " + user.Username)
}

// IsAdmin checks if a user is an admin or not
func (user *User) IsAdmin() bool {
	userAdminCap, err := user.parent.cm.GetByName("UserAdmin")
	if err != nil {
		return false
	}
	if user.ValidateCapability(userAdminCap) != nil {
		return false
	}
	return true
}

// GetInfo returns public information about a user
func (user *User) GetInfo() core.UserInfo {
	return core.UserInfo{
		Username: user.Username,
		Name:     user.Name,
		IsAdmin:  user.IsAdmin(),
	}
}

//
// Public package methods
//

// UserManager implements the core.UserManager interface, which manages users
type UserManager struct {
	db core.DB
	cm core.CapabilityManager
}

// CreateUserManager return a UserManager instance, which implements the core.UserManager interface
func CreateUserManager(db core.DB, cm core.CapabilityManager) *UserManager {
	if db == nil || cm == nil {
		log.Panic("Failed to create user manager: none of the inputs can be nil")
	}

	return &UserManager{db: db, cm: cm}
}

// CreateUser creates and returns a user
func (um *UserManager) CreateUser(username string, password string, name string, isadmin bool) (core.User, error) {

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
		parent:       um,
	}
	if isadmin {
		user.Capabilities = append(user.Capabilities, "UserAdmin")
	}

	return &user, user.Save()
}

// ValidateAndGetUser takes a username and password and returns the full User struct if credentials are valid
func (um *UserManager) ValidateAndGetUser(username string, password string) (core.User, error) {
	log.Debugf("Searching for username %s", username)

	errInvalid := errors.New("Invalid credentials")
	var user User
	err := um.db.One("Username", username, &user)
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
	user.parent = um
	return &user, nil
}

// GetUser returns a user based on the username
func (um *UserManager) GetUser(username string) (core.User, error) {
	errInvalid := errors.New("Invalid username")
	var user User
	err := um.db.One("Username", username, &user)
	if err != nil {
		log.Debugf("Can't find user %s (%s)", username, err)
		return nil, errInvalid
	}
	user.parent = um
	return &user, nil
}

// GetUserForToken returns a user for a specific token
func (um *UserManager) GetUserForToken(token string) (core.User, error) {
	if usr, ok := usersTokens[token]; ok {
		return usr, nil
	}
	return nil, errors.New("No user found for token " + token)
}

// SetupAdmin creates and initial admin user
func (um *UserManager) SetupAdmin() {
	username, clearpassword, name := readCredentials()
	user, err := um.CreateUser(username, clearpassword, name, true)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("User %s has been created.", user.GetUsername())
}
