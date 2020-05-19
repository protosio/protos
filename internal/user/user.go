package user

import (
	"fmt"
	"os"

	"cuelang.org/go/cue"
	"cuelang.org/go/encoding/gocode/gocodec"
	"github.com/protosio/protos/internal/core"
	"github.com/protosio/protos/internal/ssh"
)

const adminDS = "admin"
const config = `
import "strings"

dev :: {
	Name: string & strings.MinRunes(1) & strings.MaxRunes(32)
	KeySeed: bytes
	Network: string
}

UserInfo :: {
    Username: string & strings.MinRunes(1) & strings.MaxRunes(32)
    Name: string & strings.MinRunes(1) & strings.MaxRunes(128)
	Domain: string & strings.MinRunes(3) & strings.MaxRunes(128)
	Password: string & strings.MinRunes(10) & strings.MaxRunes(128)
	Device: dev
}
UserInfo
`

const (
	UserNetwork = "10.100.0.1/24"
)

var r cue.Runtime
var codec = gocodec.New(&r, nil)

// Device represents a user device (laptop, phone, etc)
type Device struct {
	Name    string
	KeySeed []byte
	Network string
}

// Info represents the local Protos user
type Info struct {
	db       core.DB `noms:"-"`
	Username string
	Name     string
	Domain   string
	Password string
	Device   Device
}

// Save saves the user to db
func (ui Info) save() {
	err := ui.db.SaveStruct(adminDS, ui)
	if err != nil {
		panic(err)
	}
}

// SetName enables the changing of the name of the user
func (ui Info) SetName(name string) error {
	ui.Name = name
	ui.save()
	return nil
}

// SetDomain enables the changing of the domain of the user
func (ui Info) SetDomain(domain string) error {
	ui.Domain = domain
	ui.save()
	return nil
}

// Validate checks if the user info conforms to the user CUE schema
func (ui Info) Validate() error {
	uiCueInstance, _ := r.Compile("", config)
	return codec.Validate(uiCueInstance.Value(), &ui)
}

//
// package methods
//

// New creates and returns a new user. Also validates the data
func New(db core.DB, username string, name string, domain string, password string) (Info, error) {
	usrInfo, err := Get(db)
	if err == nil {
		return usrInfo, fmt.Errorf("User '%s' already initialized. Modify it using the 'user set' command", usrInfo.Username)
	}
	host, err := os.Hostname()
	if err != nil {
		return usrInfo, fmt.Errorf("Failed to add user. Could not retrieve hostname: %w", err)
	}
	key, err := ssh.GenerateKey()
	if err != nil {
		return usrInfo, fmt.Errorf("Failed to add user. Could not generate key: %w", err)
	}
	userDevice := Device{Name: host, KeySeed: key.Seed(), Network: UserNetwork}

	user := Info{db: db, Username: username, Name: name, Domain: domain, Password: password, Device: userDevice}
	err = user.Validate()
	if err != nil {
		return user, fmt.Errorf("Failed to add user. Validation error: %v", err)
	}

	user.save()
	return user, nil
}

// Get returns information about the local user
func Get(db core.DB) (Info, error) {
	usr := Info{}
	err := db.GetStruct(adminDS, &usr)
	if err != nil {
		return usr, err
	}

	usr.db = db
	return usr, nil
}
