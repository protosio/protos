package auth

import (
	"encoding/gob"
	"fmt"

	"github.com/pkg/errors"

	"github.com/protosio/protos/internal/core"
	"github.com/protosio/protos/pkg/types"

	"github.com/protosio/protos/internal/util"

	"golang.org/x/crypto/bcrypt"
)

const (
	userBucket = "user"
	authDS     = "auth"
)

var log = util.GetLogger("auth")

// User represents a Protos user
type User struct {
	parent *UserManager `noms:"-"`

	// Public members
	Username     string             `json:"username" storm:"id"`
	Password     string             `json:"password"`
	Name         string             `json:"name"`
	IsDisabled   bool               `json:"isdisabled"`
	Capabilities []string           `json:"capabilities"`
	Devices      []types.UserDevice `json:"devices"`
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

func getUser(username string, db core.DBCLI) (User, error) {
	var users []User
	err := db.GetSet(authDS, &users)
	if err != nil {
		return User{}, err
	}
	for _, user := range users {
		if user.Username == username {
			return user, nil
		}
	}
	return User{}, fmt.Errorf("Could not find user '%s'", username)
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
	return user.parent.db.InsertInSet(authDS, *user)
}

// ValidateCapability implements the capability checker interface
func (user *User) ValidateCapability(cap core.Capability) error {
	for _, usercap := range user.Capabilities {
		if user.parent.cm.Validate(cap, usercap) {
			return nil
		}
	}
	return errors.Errorf("Method capability '%s' not satisfied by user '%s'", cap.GetName(), user.Username)
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

// GetDevices returns the devices that belong to a user
func (user *User) GetDevices() []types.UserDevice {
	return user.Devices
}

//
// Public package methods
//

// UserManager implements the core.UserManager interface, which manages users
type UserManager struct {
	db core.DBCLI
	cm core.CapabilityManager
}

// CreateUserManager return a UserManager instance, which implements the core.UserManager interface
func CreateUserManager(db core.DBCLI, cm core.CapabilityManager) *UserManager {
	if db == nil || cm == nil {
		log.Panic("Failed to create user manager: none of the inputs can be nil")
	}
	gob.Register(&User{})

	return &UserManager{db: db, cm: cm}
}

// CreateUser creates and returns a user
func (um *UserManager) CreateUser(username string, password string, name string, isadmin bool, devices []types.UserDevice) (core.User, error) {

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

	user, err := getUser(username, um.db)
	if err != nil {
		log.Debugf("Can't find user '%s' (%s)", username, err)
		return nil, errInvalid
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		log.Debugf("Invalid password for user '%s'", username)
		return nil, errInvalid
	}

	if user.IsDisabled {
		log.Debugf("User '%s' is disabled", username)
		return nil, errInvalid
	}

	log.Debugf("User '%s' logged in successfuly", username)
	user.parent = um
	return &user, nil
}

// GetUser returns a user based on the username
func (um *UserManager) GetUser(username string) (core.User, error) {
	errInvalid := errors.New("Invalid username")
	user, err := getUser(username, um.db)
	if err != nil {
		log.Debugf("Can't find user '%s' (%s)", username, err)
		return nil, errInvalid
	}
	user.parent = um
	return &user, nil
}

// SetParent returns sets the parent (user manager) for a given user
func (um *UserManager) SetParent(user core.User) (core.User, error) {
	usr, ok := user.(*User)
	if !ok {
		return nil, errors.New("Failed to cast core.User to local type")
	}
	usr.parent = um
	return usr, nil
}
