package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/app"
	"github.com/protosio/protos/internal/auth"
	"github.com/protosio/protos/internal/capability"
	"github.com/protosio/protos/internal/cloud"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/installer"
	"github.com/protosio/protos/internal/meta"
	"github.com/protosio/protos/internal/p2p"
	"github.com/protosio/protos/internal/platform"
	"github.com/protosio/protos/internal/resource"
	"github.com/protosio/protos/internal/ssh"
	"github.com/protosio/protos/internal/task"
	"github.com/protosio/protos/internal/util"
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
	DB  db.DB
	CM  *capability.Manager
	CLM cloud.CloudManager
	UM  *auth.UserManager
	SM  *ssh.Manager
	VPN *vpn.VPN
	AS  *installer.AppStore
	AM  *app.Manager
	Log *logrus.Entry
}

// NewEnv creates and returns an instance of Env
func NewEnv(
	db db.DB,
	capm *capability.Manager,
	clm cloud.CloudManager,
	um *auth.UserManager,
	sm *ssh.Manager,
	vpn *vpn.VPN,
	as *installer.AppStore,
	am *app.Manager,
	log *logrus.Entry) *Env {

	if db == nil || capm == nil || clm == nil || um == nil || sm == nil || vpn == nil || as == nil || am == nil || log == nil {
		panic("env: none of the env inputs should be nil")
	}
	return &Env{DB: db, CM: capm, CLM: clm, UM: um, SM: sm, VPN: vpn, AS: as, AM: am, Log: log}
}

func main() {
	var loglevel string
	var dataPath string
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
			&cli.StringFlag{
				Name:        "path, p",
				Value:       "~/.protos",
				Usage:       "Path where protos data is stored",
				Destination: &dataPath,
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
		configure(c.Args().First(), loglevel, dataPath)
		return nil
	}

	// app.After = func(c *cli.Context) error {
	// 	fmt.Println("1 -----")
	// 	if envi != nil && envi.DB != nil {
	// 		ips := []string{}
	// 		instances, err := envi.CLM.GetInstances()
	// 		if err != nil {
	// 			return err
	// 		}
	// 		for _, instance := range instances {
	// 			ips = append(ips, instance.PublicIP)
	// 		}
	// 		fmt.Println("2 -----")
	// 		err = envi.DB.SyncAll(ips)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		return envi.DB.Close()
	// 	}
	// 	fmt.Println("3 -----")
	// 	return nil
	// }

	err = app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
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

func configure(currentCmd string, logLevel string, dataPath string) {
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		fmt.Println(fmt.Errorf("Log level '%s' is invalid", logLevel))
		os.Exit(1)
	}
	util.SetLogLevel(level)
	log := util.GetLogger("cli")

	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	if dataPath == "~" {
		dataPath = homedir
	} else if strings.HasPrefix(dataPath, "~/") {
		dataPath = filepath.Join(homedir, dataPath[2:])
	}

	// create protos dir
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		err := os.Mkdir(dataPath, 0755)
		if err != nil {
			log.Fatalf("Failed to create protos dir '%s': %s", dataPath, err.Error())
		}
	}

	// open db
	protosDB := "protos.db"
	dbi, err := db.Open(dataPath, protosDB)
	if err != nil {
		log.Fatalf("Failed to open db during configuration: %v", err)
	}

	// get default cfg
	cfg := config.Get()

	// create publisher
	pub := &publisher{pubchan: make(chan interface{}, 100)}

	// create various managers
	rm := resource.CreateManager(dbi)
	sm := ssh.CreateManager(dbi)
	m := meta.SetupForClient(rm, dbi, sm, version.String())
	capm := capability.CreateManager()
	rp := platform.Create(cfg.Runtime, cfg.RuntimeEndpoint, cfg.AppStoreHost, cfg.InContainer, wgtypes.Key{})
	tm := task.CreateManager(dbi, pub)
	as := installer.CreateAppStore(rp, tm, capm)
	um := auth.CreateUserManager(dbi, sm, capm)

	key, err := m.GetKey()
	if err != nil {
		log.Fatalf("Failed to retrieve key during configuration: %v", err)
	}
	p2pManager, err := p2p.NewManager(10500, key, m, um, dbi.GetChunkStore())
	if err != nil {
		log.Fatal(err)
	}
	clm := cloud.CreateManager(dbi, um, sm, p2pManager)
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
