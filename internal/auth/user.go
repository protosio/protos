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
	parent *UserManager

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

// Save saves the User struct to the database. The username is used as an unique key
func (user *User) Save() error {
	err := db.Insert(user.parent.db, createUserInsertMapper(*user))
	if err != nil {
		return errors.Wrapf(err, "Could not insert user '%s'", user.Username)
	}

	return nil
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

// AddDevice adds a device to the user
func (user *User) AddDevice(id string, name string, publicKey string, network string) error {
	device := UserDevice{
		MachineID: id,
		Name:      name,
		PublicKey: publicKey,
		Network:   network,
	}
	user.Devices = append(user.Devices, device)
	return user.Save()
}

// GetDevices returns the devices that belong to a user
func (user *User) GetDevices() []UserDevice {
	return user.Devices
}

// GetCurrentDevice returns the device that Protos is running on currently
func (user *User) GetCurrentDevice() (UserDevice, error) {
	id, err := machineid.ProtectedID("protos")
	if err != nil {
		return UserDevice{}, fmt.Errorf("failed to generate machine id: %w", err)
	}

	for _, dev := range user.Devices {
		if dev.MachineID == id {
			return dev, nil
		}
	}
	return UserDevice{}, fmt.Errorf("failed to find machine with id '%s'", id)
}

// GetKeyCurrentDevice returns the private key for the current device
func (user *User) GetKeyCurrentDevice() (pcrypto.Key, error) {
	device, err := user.GetCurrentDevice()
	if err != nil {
		return pcrypto.Key{}, err
	}

	key, err := user.parent.sm.GetKeyByPub(device.PublicKey)
	if err != nil {
		return pcrypto.Key{}, err
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
	db *db.DB
	cm *capability.Manager
	sm *pcrypto.Manager
}

// CreateUserManager return a UserManager instance, which implements the core.UserManager interface
func CreateUserManager(db *db.DB, sm *pcrypto.Manager, cm *capability.Manager, configurator PeerConfigurator) *UserManager {
	if db == nil || sm == nil || cm == nil || configurator == nil {
		log.Panic("Failed to create user manager: none of the inputs can be nil")
	}
	gob.Register(&User{})

	return &UserManager{db: db, sm: sm, cm: cm}
}

// CreateUser creates and returns a user
func (um *UserManager) CreateUser(username string, name string, isadmin bool) (*User, error) {

	user := User{
		parent:     um,
		Username:   username,
		Name:       name,
		IsDisabled: false,
		Devices:    []UserDevice{},
	}

	return &user, user.Save()
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
func (um *UserManager) GetAdmin() (User, error) {
	users, err := db.SelectMultiple(um.db, createUserQueryMapper(sq.New[db.USER](""), nil))
	if err != nil {
		return User{}, fmt.Errorf("could not retrieve users: %w", err)
	}
	if len(users) == 0 {
		return User{}, fmt.Errorf("could not find admin user")
	}

	return users[0], nil
}

// SetParent returns sets the parent (user manager) for a given user
func (um *UserManager) SetParent(user *User) (*User, error) {
	user.parent = um
	return user, nil
}
