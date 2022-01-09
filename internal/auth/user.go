package auth

import (
	"encoding/gob"
	"fmt"

	"github.com/pkg/errors"

	"github.com/protosio/protos/internal/capability"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/ssh"
	"github.com/protosio/protos/internal/util"

	"github.com/denisbrodbeck/machineid"
	"golang.org/x/crypto/bcrypt"
)

const (
	authDS = "auth"
)

var log = util.GetLogger("auth")

type UserInfo struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	IsAdmin  bool   `json:"isadmin"`
}

// UserDevice - represents a device that a user uses to connect to the instances. A user can have multiple devices (laptop, mobile phone etc)
type UserDevice struct {
	Name      string `json:"name" validate:"required"`
	PublicKey string `json:"publickey" validate:"base64"`   // ed25519 base64 encoded public key
	Network   string `json:"network" validate:"cidrv4"`     // CIDR notation
	MachineID string `json:"machineid" validate:"required"` // ID that uniquely identifies a machine
}

// User represents a Protos user
type User struct {
	parent *UserManager `noms:"-"`

	// Public members
	Username     string       `json:"username"`
	Password     string       `json:"-"`
	PasswordHash string       `json:"-"`
	Name         string       `json:"name"`
	IsDisabled   bool         `json:"isdisabled"`
	Capabilities []string     `json:"capabilities"`
	Devices      []UserDevice `json:"devices"`
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

func getUser(username string, db db.DB) (User, error) {
	var users map[string]User
	err := db.GetMap(authDS, &users)
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

// GetPassword returns the password of the user in string format
func (user *User) GetPassword() string {
	return user.Password
}

// Save saves the User struct to the database. The username is used as an unique key
func (user *User) Save() error {
	log.Debugf("Writing username %s to database", user.Username)
	return user.parent.db.InsertInMap(authDS, user.Username, *user)
}

// ValidateCapability implements the capability checker interface
func (user *User) ValidateCapability(cap *capability.Capability) error {
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
func (user *User) GetInfo() UserInfo {
	return UserInfo{
		Username: user.Username,
		Name:     user.Name,
		IsAdmin:  user.IsAdmin(),
	}
}

// GetDevices returns the devices that belong to a user
func (user *User) GetDevices() []UserDevice {
	return user.Devices
}

// GetCurrentDevice returns the device that Protos is running on currently
func (user *User) GetCurrentDevice() (UserDevice, error) {
	id, err := machineid.ProtectedID("protos")
	if err != nil {
		return UserDevice{}, fmt.Errorf("Failed to generate machine id: %w", err)
	}
	for _, dev := range user.Devices {
		if dev.MachineID == id {
			return dev, nil
		}
	}
	return UserDevice{}, fmt.Errorf("Failed to find machine with id '%s'", id)
}

// GetKeyCurrentDevice returns the private key for the current device
func (user *User) GetKeyCurrentDevice() (*ssh.Key, error) {
	dev, err := user.GetCurrentDevice()
	if err != nil {
		return nil, err
	}
	key, err := user.parent.sm.GetKeyByPub(dev.PublicKey)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// SetName enables the changing of the name of the user
func (user *User) SetName(name string) error {
	user.Name = name
	return user.Save()
}

//
// Public package methods
//

// UserManager implements the core.UserManager interface, which manages users
type UserManager struct {
	db db.DB
	cm *capability.Manager
	sm *ssh.Manager
}

// CreateUserManager return a UserManager instance, which implements the core.UserManager interface
func CreateUserManager(db db.DB, sm *ssh.Manager, cm *capability.Manager) *UserManager {
	if db == nil || sm == nil || cm == nil {
		log.Panic("Failed to create user manager: none of the inputs can be nil")
	}
	gob.Register(&User{})

	err := db.InitDataset(authDS, nil)
	if err != nil {
		log.Fatal("Failed to initialize auth dataset: ", err)
	}

	return &UserManager{db: db, sm: sm, cm: cm}
}

// CreateUser creates and returns a user
func (um *UserManager) CreateUser(username string, password string, name string, isadmin bool, devices []UserDevice) (*User, error) {

	passwordHash, err := generatePasswordHash(password)
	if err != nil {
		return nil, err
	}

	if len(devices) == 0 {
		return nil, fmt.Errorf("Failed to create user '%s': 0 user devices provided", username)
	}

	// FIXME: get rid of the password somehow. Need to authenticate based on priv/pub key
	// the current setup is dangerous because the password will be synced to instances
	user := User{
		parent:       um,
		Username:     username,
		Password:     password,
		PasswordHash: passwordHash,
		Name:         name,
		IsDisabled:   false,
		Capabilities: []string{},
		Devices:      devices,
	}
	if isadmin {
		user.Capabilities = append(user.Capabilities, "UserAdmin")
	}

	return &user, user.Save()
}

// ValidateAndGetUser takes a username and password and returns the full User struct if credentials are valid
func (um *UserManager) ValidateAndGetUser(username string, password string) (*User, error) {
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
func (um *UserManager) GetUser(username string) (*User, error) {
	errInvalid := errors.New("Invalid username")
	user, err := getUser(username, um.db)
	if err != nil {
		log.Debugf("Can't find user '%s' (%s)", username, err)
		return nil, errInvalid
	}
	user.parent = um
	return &user, nil
}

// GetAdmin returns the admin username. Only one admin is allowed at the moment
func (um *UserManager) GetAdmin() (*User, error) {
	var users map[string]User
	err := um.db.GetMap(authDS, &users)
	if err != nil {
		return &User{}, err
	}
	for _, usr := range users {
		usr.parent = um
		if usr.IsAdmin() == true {
			return &usr, nil
		}
	}
	return &User{}, fmt.Errorf("Could not find admin user")
}

// SetParent returns sets the parent (user manager) for a given user
func (um *UserManager) SetParent(user *User) (*User, error) {
	user.parent = um
	return user, nil
}
