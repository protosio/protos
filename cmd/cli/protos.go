package main

import (
	"fmt"
	"os"
	"path"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/app"
	"github.com/protosio/protos/internal/auth"
	"github.com/protosio/protos/internal/capability"
	"github.com/protosio/protos/internal/cloud"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/core"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/installer"
	"github.com/protosio/protos/internal/meta"
	"github.com/protosio/protos/internal/platform"
	"github.com/protosio/protos/internal/resource"
	"github.com/protosio/protos/internal/ssh"
	"github.com/protosio/protos/internal/task"
	"github.com/protosio/protos/internal/vpn"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var log *logrus.Logger
var envi *Env
var cloudName string
var cloudLocation string
var protosVersion string
var version *semver.Version

// Env is a struct that containts program dependencies that get injected in other modules
type Env struct {
	DB  core.DB
	CM  core.CapabilityManager
	CLM core.CloudManager
	UM  core.UserManager
	SM  core.SSHManager
	VPN core.VPN
	AS  core.AppStore
	AM  core.AppManager
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
	as core.AppStore,
	am core.AppManager,
	log *logrus.Logger) *Env {

	if db == nil || capm == nil || clm == nil || um == nil || sm == nil || vpn == nil || as == nil || am == nil || log == nil {
		panic("env: non of the env inputs should be nil")
	}
	return &Env{DB: db, CM: capm, CLM: clm, UM: um, SM: sm, VPN: vpn, AS: as, AM: am, Log: log}
}

func main() {
	var loglevel string
	var err error

	version, err = semver.NewVersion("0.0.0-dev.4")
	if err != nil {
		panic(err)
	}

	app := &cli.App{
		Name:    "protos",
		Usage:   "Command-line client for Protos",
		Authors: []*cli.Author{{Name: "Alex Giurgiu", Email: "alex@giurgiu.io"}},
		Version: version.String(),
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
			cmdApp,
			cmdUser,
			cmdVPN,
		},
	}

	app.Before = func(c *cli.Context) error {
		configure(c.Args().First(), loglevel)
		return nil
	}

	app.After = func(c *cli.Context) error {
		if envi != nil && envi.DB != nil {
			return envi.DB.Close()
		}
		return nil
	}

	err = app.Run(os.Args)
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

type publisher struct {
	pubchan chan interface{}
}

// GetWSPublishChannel returns the channel that can be used to publish messages to the available websockets
func (pub *publisher) GetWSPublishChannel() chan interface{} {
	return pub.pubchan
}

func transformCredentials(creds map[string]interface{}) map[string]string {
	transformed := map[string]string{}
	for name, val := range creds {
		transformed[name] = val.(string)
	}
	return transformed
}

func configure(currentCmd string, logLevel string) {
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

	// get default cfg
	cfg := config.Get()

	// create publisher
	pub := &publisher{pubchan: make(chan interface{}, 100)}

	// create various managers

	sm := ssh.CreateManager(dbi)
	capm := capability.CreateManager()
	rp := platform.Create(cfg.Runtime, cfg.RuntimeEndpoint, cfg.AppStoreHost, cfg.InContainer, wgtypes.Key{})
	tm := task.CreateManager(dbi, pub)
	as := installer.CreateAppStore(rp, tm, capm)
	um := auth.CreateUserManager(dbi, sm, capm)
	clm := cloud.CreateManager(dbi, um, sm)
	rm := resource.CreateManager(dbi)
	m := meta.SetupForClient(rm, dbi, version.String())
	am := app.CreateManager(rm, tm, rp, dbi, m, pub, as, capm)
	vpn, err := vpn.New(dbi, um, clm)
	if err != nil {
		log.Fatal(err)
	}

	envi = NewEnv(dbi, capm, clm, um, sm, vpn, as, am, log)

	if currentCmd != "init" {
		_, err = envi.UM.GetAdmin()
		if err != nil {
			log.Fatal(errors.Wrap(err, "Please run init command to setup Protos"))
		}
	}
}
