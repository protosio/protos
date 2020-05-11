package user

import (
	"fmt"
	"net"
	"os"

	"cuelang.org/go/cue"
	"cuelang.org/go/encoding/gocode/gocodec"
	"github.com/protosio/protos/internal/cloud"
	"github.com/protosio/protos/internal/env"
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
	// Address space for allocating networks
	netSpace    = "10.100.0.0/16"
	userNetwork = "10.100.0.1/24"
)

var r cue.Runtime
var codec = gocodec.New(&r, nil)

// var envi *env.Env

// Device represents a user device (laptop, phone, etc)
type Device struct {
	Name    string
	KeySeed []byte
	Network string
}

// Info represents the local Protos user
type Info struct {
	env      *env.Env `noms:"-"`
	Username string
	Name     string
	Domain   string
	Password string
	Device   Device
}

// Save saves the user to db
func (ui Info) save() {
	err := ui.env.DB.SaveStruct(adminDS, ui)
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
func New(envp *env.Env, username string, name string, domain string, password string) (Info, error) {
	usrInfo, err := Get(envp)
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
	userDevice := Device{Name: host, KeySeed: key.Seed(), Network: userNetwork}

	user := Info{env: envp, Username: username, Name: name, Domain: domain, Password: password, Device: userDevice}
	err = user.Validate()
	if err != nil {
		return user, fmt.Errorf("Failed to add user. Validation error: %v", err)
	}

	user.save()
	return user, nil
}

// Get returns information about the local user
func Get(env *env.Env) (Info, error) {
	usr := Info{}
	err := env.DB.GetStruct(adminDS, &usr)
	if err != nil {
		return usr, err
	}

	usr.env = env
	return usr, nil
}

// AllocateNetwork allocates an unused network for an instance
func AllocateNetwork(instances []cloud.InstanceInfo) (net.IPNet, error) {
	_, userNet, err := net.ParseCIDR(userNetwork)
	if err != nil {
		panic(err)
	}
	// create list of existing networks
	usedNetworks := []net.IPNet{*userNet}
	for _, inst := range instances {
		_, inet, err := net.ParseCIDR(inst.Network)
		if err != nil {
			panic(err)
		}
		usedNetworks = append(usedNetworks, *inet)
	}

	// figure out which is the first network that is not currently used
	_, netspace, _ := net.ParseCIDR(netSpace)
	for i := 0; i <= 255; i++ {
		newNet := *netspace
		newNet.IP[2] = byte(i)
		newNet.Mask[2] = byte(255)
		for _, usedNet := range usedNetworks {
			if !newNet.Contains(usedNet.IP) {
				return newNet, nil
			}
		}
	}

	return net.IPNet{}, fmt.Errorf("Failed to allocate network")
}
