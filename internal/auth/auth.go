package auth

import (
	"encoding/gob"
	"fmt"

	"github.com/bokwoon95/sq"
	"github.com/pkg/errors"

	"github.com/protosio/protos/internal/capability"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/util"

	"github.com/denisbrodbeck/machineid"
)

var log = util.GetLogger("auth")

type PeerConfigurator interface {
	Refresh() error
}

type UserInfo struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	IsAdmin  bool   `json:"isadmin"`
}

// UserDevice - represents a device that a user uses to connect to the instances. A user can have multiple devices (laptop, mobile phone etc)
type UserDevice struct {
	Name      string `json:"name" validate:"required"`
	PublicKey string `json:"publickey" validate:"base64"`   // ed25519 public key
	Network   string `json:"network" validate:"cidrv4"`     // CIDR notation
	MachineID string `json:"machineid" validate:"required"` // ID that uniquely identifies a machine
}

// User represents a Protos user
type User struct {
	// Public members
	Username   string       `json:"username"`
	Name       string       `json:"name"`
	IsDisabled bool         `json:"isdisabled"`
	Devices    []UserDevice `json:"devices"`
}

func getUser(username string, dbi *db.DB) (User, error) {
	userModel := sq.New[db.USER]("")
	user, err := db.SelectOne(dbi, createUserQueryMapper(userModel, []sq.Predicate{userModel.USERNAME.EqString(username)}))
	if err != nil {
		return user, fmt.Errorf("failed to retrieve user: %w", err)
	}

	return user, nil
}

//
// UserDevice methods
//

func (ud *UserDevice) GetPublicKey() string {
	return ud.PublicKey
}

func (ud *UserDevice) GetPublicIP() string {
	return ""
}

func (ud *UserDevice) GetName() string {
	return ud.Name
}

//
// User instance methods
//

// GetUsername returns the username of the user in string format
func (user *User) GetUsername() string {
	return user.Username
}

// IsAdmin checks if a user is an admin or not
func (user *User) IsAdmin() bool {
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

//
// Public package methods
//

// AuthManager implements the core.AuthManager interface, which manages users
type AuthManager struct {
	db *db.DB
	cm *capability.Manager
	sm *pcrypto.Manager
}

// CreateAuthManager return a AuthManager instance, which implements the core.AuthManager interface
func CreateAuthManager(db *db.DB, sm *pcrypto.Manager, cm *capability.Manager, configurator PeerConfigurator) *AuthManager {
	if db == nil || sm == nil || cm == nil || configurator == nil {
		log.Panic("Failed to create user manager: none of the inputs can be nil")
	}
	gob.Register(&User{})

	return &AuthManager{db: db, sm: sm, cm: cm}
}

// CreateUser creates and returns a user
func (um *AuthManager) CreateUser(username string, name string, isadmin bool) (User, error) {

	user := User{
		Username:   username,
		Name:       name,
		IsDisabled: false,
		Devices:    []UserDevice{},
	}

	err := db.Insert(um.db, createUserInsertMapper(user))
	if err != nil {
		return user, errors.Wrapf(err, "Could not insert user '%s'", user.Username)
	}

	return user, nil
}

// GetUser returns a user based on the username
func (um *AuthManager) GetUser(username string) (*User, error) {
	errInvalid := errors.New("Invalid username")
	user, err := getUser(username, um.db)
	if err != nil {
		log.Debugf("Can't find user '%s' (%s)", username, err)
		return nil, errInvalid
	}
	return &user, nil
}

// GetAdmin returns the admin username. Only one admin is allowed at the moment
func (um *AuthManager) GetAdmin() (User, error) {
	users, err := db.SelectMultiple(um.db, createUserQueryMapper(sq.New[db.USER](""), []sq.Predicate{}))
	if err != nil {
		return User{}, fmt.Errorf("could not retrieve users: %w", err)
	}
	if len(users) == 0 {
		return User{}, fmt.Errorf("could not find admin user")
	}

	return users[0], nil
}

// AddDevice adds a device to the user
func (um *AuthManager) AddDevice(userID string, id string, name string, publicKey string, network string) error {
	ud := UserDevice{
		Name:      name,
		PublicKey: publicKey,
		Network:   network,
		MachineID: id,
	}

	err := db.Insert(um.db, createUserDeviceInsertMapper(ud, userID))
	if err != nil {
		return errors.Wrapf(err, "Could not insert user device '%s'", name)
	}

	return nil
}

// GetCurrentDevice returns the device that Protos is running on currently
func (um *AuthManager) GetCurrentDevice() (UserDevice, error) {
	id, err := machineid.ProtectedID("protos")
	if err != nil {
		return UserDevice{}, fmt.Errorf("failed to generate machine id: %w", err)
	}

	udModel := sq.New[db.USER_DEVICE]("")
	ud, err := db.SelectOne(um.db, createUserDeviceQueryMapper(udModel, []sq.Predicate{udModel.ID.EqString(id)}))
	if err != nil {
		return UserDevice{}, fmt.Errorf("failed to retrieve device: %w", err)
	}

	return ud, nil
}
