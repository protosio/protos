package main

import (
	"fmt"
	"os"
	osuser "os/user"

	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/env"
	"github.com/protosio/protos/internal/user"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log *logrus.Logger
var envi *env.Env
var cloudName string
var cloudLocation string
var protosVersion string

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

	homedir := os.Getenv("HOME")
	if homedir == "" {
		usr, _ := osuser.Current()
		homedir = usr.HomeDir
	}
	protosDir := homedir + "/.protos"
	protosDB := "protos.db"

	dbi, err := db.Open(protosDir, protosDB)
	if err != nil {
		log.Fatal(err)
	}

	envi = env.New(dbi, log)

	if currentCmd != "init" {
		_, err = user.Get(envi.DB)
		if err != nil {
			log.Fatal(errors.Wrap(err, "Please run init command to setup Protos"))
		}
	}
}
