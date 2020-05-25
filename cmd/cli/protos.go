package main

import (
	"fmt"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/auth"
	"github.com/protosio/protos/internal/capability"
	"github.com/protosio/protos/internal/cloud"
	"github.com/protosio/protos/internal/core"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/ssh"
	"github.com/protosio/protos/internal/vpn"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log *logrus.Logger
var envi *Env
var cloudName string
var cloudLocation string
var protosVersion string

// Env is a struct that containts program dependencies that get injected in other modules
type Env struct {
	DB  core.DB
	CM  core.CapabilityManager
	CLM core.CloudManager
	UM  core.UserManager
	SM  core.SSHManager
	VPN core.VPN
	Log *logrus.Logger
}

// NewEnv creates and returns an instance of Env
func NewEnv(
	db core.DB,
	capm core.CapabilityManager,
	clm core.CloudManager,
	um core.UserManager,
	sm core.SSHManager,
	vpn core.VPN,
	log *logrus.Logger) *Env {

	if db == nil || capm == nil || clm == nil || um == nil || sm == nil || vpn == nil || log == nil {
		panic("env: non of the env inputs should be nil")
	}
	return &Env{DB: db, CM: capm, CLM: clm, UM: um, SM: sm, VPN: vpn, Log: log}
}

func main() {
	var loglevel string
	app := &cli.App{
		Name:    "protos-cli",
		Usage:   "Command-line client for Protos",
		Version: "0.0.0-dev.2",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "log, l",
				Value:       "info",
				Usage:       "Log level: warn, info, debug",
				Destination: &loglevel,
			},
		},
		Commands: []*cli.Command{
			cmdInit,
			cmdRelease,
			cmdCloud,
			cmdInstance,
			cmdUser,
			cmdVPN,
		},
	}

	app.Before = func(c *cli.Context) error {
		config(c.Args().First(), loglevel)
		return nil
	}

	app.After = func(c *cli.Context) error {
		if envi != nil && envi.DB != nil {
			return envi.DB.Close()
		}
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}

type userDetails struct {
	Username        string
	Name            string
	Password        string
	PasswordConfirm string
	Domain          string
}

func transformCredentials(creds map[string]interface{}) map[string]string {
	transformed := map[string]string{}
	for name, val := range creds {
		transformed[name] = val.(string)
	}
	return transformed
}

func config(currentCmd string, logLevel string) {
	log = logrus.New()
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		fmt.Println(fmt.Errorf("Log level '%s' is invalid", logLevel))
		os.Exit(1)
	}
	log.SetLevel(level)

	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	// create protos dir
	protosDir := path.Join(homedir, "/.protos")
	if _, err := os.Stat(protosDir); os.IsNotExist(err) {
		err := os.Mkdir(protosDir, 0755)
		if err != nil {
			log.Fatalf("Failed to create protos dir '%s': %w", protosDir, err)
		}
	}

	// open db
	protosDB := "protos.db"
	dbi, err := db.Open(protosDir, protosDB)
	if err != nil {
		log.Fatal(err)
	}

	// create various managers
	sm := ssh.CreateManager(dbi)
	capm := capability.CreateManager()
	um := auth.CreateUserManager(dbi, sm, capm)
	clm := cloud.CreateManager(dbi, um, sm)
	vpn, err := vpn.New(dbi, um)
	if err != nil {
		log.Fatal(err)
	}

	envi = NewEnv(dbi, capm, clm, um, sm, vpn, log)

	if currentCmd != "init" {
		_, err = envi.UM.GetAdmin()
		if err != nil {
			log.Fatal(errors.Wrap(err, "Please run init command to setup Protos"))
		}
	}
}
