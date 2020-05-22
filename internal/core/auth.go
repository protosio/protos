package core

import "github.com/protosio/protos/pkg/types"

// UserInfo holds information about a user that is meant to be returned to external applications or the web interface
type UserInfo struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	IsAdmin  bool   `json:"isadmin"`
	Domain   string `json:"domain"`
}

// UserManager manages users
type UserManager interface {
	CreateUser(username string, password string, name string, domain string, isadmin bool, devices []types.UserDevice) (User, error)
	ValidateAndGetUser(username string, password string) (User, error)
	GetUser(username string) (User, error)
	SetParent(user User) (User, error)
	GetAdmin() (User, error)
}

// User represents a Protos user
type User interface {
	Save() error
	ValidateCapability(cap Capability) error
	IsAdmin() bool
	GetInfo() UserInfo
	GetUsername() string
	GetPassword() string
	GetDevices() []types.UserDevice
	GetCurrentDevice() types.UserDevice
	GetKeyCurrentDevice() ([]byte, error)
	SetName(name string) error
	SetDomain(domain string) error
}
