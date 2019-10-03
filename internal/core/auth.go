package core

import "protos/internal/capability"

// UserInfo holds information about a user that is meant to be returned to external applications or the web interface
type UserInfo struct {
	Username string `json:"username"`
	Name     string `json:"name"`
	IsAdmin  bool   `json:"isadmin"`
}

// UserManager manages users
type UserManager interface {
	CreateUser(username string, password string, name string, isadmin bool) (User, error)
	ValidateAndGetUser(username string, password string) (User, error)
	GetUser(username string) (User, error)
	GetUserForToken(token string) (User, error)
}

// User represents a Protos user
type User interface {
	Save() error
	AddToken(token string)
	ValidateCapability(cap *capability.Capability) error
	IsAdmin() bool
	GetInfo() UserInfo
	GetUsername() string
}
