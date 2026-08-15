package user

import (
	"context"
	"encoding/gob"
	"fmt"

	"github.com/bokwoon95/sq"
	"github.com/pkg/errors"

	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/util"
)

var log = util.GetLogger("auth")

type UserInfo struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	IsAdmin  bool   `json:"isadmin"`
}

// UserDevice - represents a device that a user uses to connect to the instances. A user can have multiple devices (laptop, mobile phone etc)
type UserDevice struct {
	ID                  string `json:"id" validate:"required"`
	PublicKey           string `json:"publickey" validate:"base64"` // ed25519 public key
	Name                string `json:"name" validate:"required"`    // ID that uniquely identifies a machine
	UserID              string `json:"userid" validate:"required"`
	ReplicationPriority int    `json:"replication_priority"`
}

// User represents a Protos user
type User struct {
	// Public members
	ID         string `json:"id"`
	Username   string `json:"username"`
	Name       string `json:"name"`
	IsDisabled bool   `json:"isdisabled"`
}

type DeviceIdentity struct {
	DeviceID     string
	DeviceName   string
	PublicKey    string
	UserID       string
	Username     string
	UserName     string
	UserDisabled bool
}

func getUser(username string, dbi *db.DB) (User, error) {
	return getUserContext(context.Background(), username, dbi)
}

func getUserContext(ctx context.Context, username string, dbi *db.DB) (User, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	userModel := sq.New[db.USER]("")
	user, err := db.SelectOneContext(ctx, dbi, createUserQueryMapper([]sq.Predicate{userModel.USERNAME.EqString(username)}))
	if err != nil {
		return user, fmt.Errorf("failed to retrieve user: %w", err)
	}

	return user, nil
}

//
// UserDevice methods
//

func (ud *UserDevice) GetID() string {
	return ud.ID
}

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

//
// Public package methods
//

// Manager implements the core.Manager interface, which manages users
type Manager struct {
	db *db.DB
	sm *pcrypto.Manager
}

// CreateManager return a Manager instance, which implements the core.Manager interface
func CreateManager(db *db.DB, sm *pcrypto.Manager) *Manager {
	if db == nil || sm == nil {
		log.Panic("Failed to create user manager: none of the inputs can be nil")
	}
	gob.Register(&User{})

	return &Manager{db: db, sm: sm}
}

// CreateUser creates and returns a user
func (um *Manager) CreateUser(username string, name string, isadmin bool) (User, error) {
	user, _, err := um.CreateUserWithConfirmationContext(context.Background(), username, name, isadmin)
	return user, err
}

// CreateUserWithConfirmationContext exposes the exact accepted receipt and
// current availability stage. Pending availability does not make an accepted
// user insert replayable.
func (um *Manager) CreateUserWithConfirmationContext(ctx context.Context, username string, name string, isadmin bool) (User, db.PublishedWriteConfirmation, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	user := User{
		ID:         db.MustNewUUIDv7(),
		Username:   username,
		Name:       name,
		IsDisabled: false,
	}

	confirmation, err := db.InsertWithAvailabilityContext(ctx, um.db, createUserInsertMapper(user))
	if err != nil {
		return user, confirmation, errors.Wrapf(err, "Could not insert user '%s'", user.Username)
	}

	return user, confirmation, nil
}

// GetUser returns a user based on the username
func (um *Manager) GetUser(username string) (*User, error) {
	errInvalid := errors.New("Invalid username")
	user, err := getUser(username, um.db)
	if err != nil {
		log.Debugf("Can't find user '%s' (%s)", username, err)
		return nil, errInvalid
	}
	return &user, nil
}

func (um *Manager) GetUserByID(id string) (*User, error) {
	if _, err := db.UUIDBytes(id); err != nil {
		return nil, errors.Wrap(err, "invalid user id")
	}
	userModel := sq.New[db.USER]("")
	user, err := db.SelectOne(um.db, createUserQueryMapper([]sq.Predicate{db.UUIDEq(userModel.ID, id)}))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve user '%s': %w", id, err)
	}
	return &user, nil
}

func (um *Manager) GetCurrentUser() (*User, error) {
	device, found, err := um.GetCurrentDeviceIfExists()
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("current peer is not a user device")
	}
	return um.GetUserByID(device.UserID)
}

// GetAdmin returns the admin username. Only one admin is allowed at the moment
func (um *Manager) GetAdmin() (User, error) {
	users, err := db.SelectMultiple(um.db, createUserQueryMapper([]sq.Predicate{}))
	if err != nil {
		return User{}, fmt.Errorf("could not retrieve users: %w", err)
	}
	if len(users) == 0 {
		return User{}, fmt.Errorf("could not find admin user")
	}

	return users[0], nil
}

// GetAllDevices returns all devices (without local device)
func (um *Manager) GetAllDevices(excludeLocalDevice bool) ([]UserDevice, error) {

	publicKey := ""
	if excludeLocalDevice {
		key, err := um.sm.GetLocalKey()
		if err != nil {
			return nil, fmt.Errorf("could not retrieve local key: %w", err)
		}
		publicKey = key.PublicString()
	}

	userDevices, err := db.SelectMultiple(um.db, createUserDeviceQueryAllMapper(publicKey))
	if err != nil {
		return userDevices, fmt.Errorf("could not retrieve user devices: %w", err)
	}
	return userDevices, nil
}

func (um *Manager) GetDeviceIdentities() ([]DeviceIdentity, error) {
	if um == nil || um.db == nil {
		return nil, fmt.Errorf("user manager is not configured")
	}
	identities, err := db.SelectMultiple(um.db, createDeviceIdentityQueryMapper())
	if err != nil {
		return nil, fmt.Errorf("could not retrieve device identities: %w", err)
	}
	return identities, nil
}

// AddDevice adds a device to the user
func (um *Manager) AddDevice(userID string, name string, key *pcrypto.Key) error {
	_, err := um.AddDeviceWithConfirmationContext(context.Background(), userID, name, key)
	return err
}

// AddDeviceWithConfirmationContext returns the exact accepted-write boundary
// so callers can continue passive observation instead of replaying on a
// conservative availability miss.
func (um *Manager) AddDeviceWithConfirmationContext(ctx context.Context, userID string, name string, key *pcrypto.Key) (db.PublishedWriteConfirmation, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	user, err := um.resolveUserRefContext(ctx, userID)
	if err != nil {
		return db.PublishedWriteConfirmation{}, err
	}
	ud := UserDevice{
		ID:                  db.MustNewUUIDv7(),
		Name:                name,
		PublicKey:           key.PublicString(),
		UserID:              user.ID,
		ReplicationPriority: db.DefaultReplicationPriorityForUserDeviceName(name),
	}

	confirmation, err := db.InsertWithAvailabilityContext(ctx, um.db, createUserDeviceInsertMapper(ud), db.CreatePeerInsertMapper(ud.PublicKey))
	if err != nil {
		return confirmation, errors.Wrapf(err, "Could not insert user device '%s'", name)
	}

	return confirmation, nil
}

func (um *Manager) resolveUserRefContext(ctx context.Context, ref string) (User, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, err := db.UUIDBytes(ref); err == nil {
		userModel := sq.New[db.USER]("")
		user, selectErr := db.SelectOneContext(ctx, um.db, createUserQueryMapper([]sq.Predicate{db.UUIDEq(userModel.ID, ref)}))
		if selectErr == nil {
			return user, nil
		}
	}
	user, err := getUserContext(ctx, ref, um.db)
	if err != nil {
		return User{}, fmt.Errorf("could not retrieve user '%s': %w", ref, err)
	}
	return user, nil
}

// GetCurrentDevice returns the device that Protos is running on currently
func (um *Manager) GetCurrentDevice() (UserDevice, error) {
	ud, ok, err := um.GetCurrentDeviceIfExists()
	if err != nil {
		return UserDevice{}, err
	}
	if !ok {
		return UserDevice{}, fmt.Errorf("failed to retrieve device: current peer is not a user device")
	}
	return ud, nil
}

// GetCurrentDeviceIfExists returns the local user device when this peer is a user device.
func (um *Manager) GetCurrentDeviceIfExists() (UserDevice, bool, error) {
	key, err := um.sm.GetLocalKey()
	if err != nil {
		return UserDevice{}, false, fmt.Errorf("could not retrieve local key: %w", err)
	}

	devices, err := db.SelectMultiple(um.db, createUserDeviceQueryMapper(key.PublicString()))
	if err != nil {
		return UserDevice{}, false, fmt.Errorf("failed to retrieve device: %w", err)
	}
	if len(devices) == 0 {
		return UserDevice{}, false, nil
	}
	return devices[0], true, nil
}
