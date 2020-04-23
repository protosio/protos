package core

import "github.com/protosio/protos/pkg/types"

// UserInfo holds information about a user that is meant to be returned to external applications or the web interface
type UserInfo struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	IsAdmin  bool   `json:"isadmin"`
}

// UserManager manages users
type UserManager interface {
	CreateUser(username string, password string, name string, isadmin bool, devices []types.UserDevice) (User, error)
	ValidateAndGetUser(username string, password string) (User, error)
	GetUser(username string) (User, error)
	SetParent(user User) (User, error)
}

// User represents a Protos user
type User interface {
	Save() error
	ValidateCapability(cap Capability) error
	IsAdmin() bool
	GetInfo() UserInfo
	GetUsername() string
	GetDevices() []types.UserDevice
}
