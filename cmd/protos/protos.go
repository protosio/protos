package main

import (
	"os"
	"protos/api"
	"protos/auth"
	"protos/config"
	"protos/daemon"
	"protos/database"
	"protos/util"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/urfave/cli.v1"
)

func main() {

	app := cli.NewApp()
	app.Name = "protos"
	app.Author = "Alex Giurgiu"
	app.Email = "alex@giurgiu.io"
	app.Version = "0.0.1"

	var configFile string
	var loglevel string

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "config",
			Value:       "protos.yaml",
			Usage:       "Specify a config file",
			Destination: &configFile,
		},
		cli.StringFlag{
			Name:        "loglevel",
			Value:       "info",
			Usage:       "Specify log level: debug, info, warn, error",
			Destination: &loglevel,
		},
	}

	app.Before = func(c *cli.Context) error {
		if loglevel == "debug" {
			util.SetLogLevel(log.DebugLevel)
		} else if configFile == "info" {
			util.SetLogLevel(log.InfoLevel)
		} else if configFile == "warn" {
			util.SetLogLevel(log.WarnLevel)
		} else if configFile == "error" {
			util.SetLogLevel(log.ErrorLevel)
		}
		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:  "daemon",
			Usage: "start the server",
			Action: func(c *cli.Context) error {
				config.Load(configFile)
				database.Open()
				defer database.Close()
				daemon.StartUp()
				daemon.LoadApps()
				api.Websrv()
				return nil
			},
		},
		{
			Name:  "init",
			Usage: "create initial configuration and user",
			Action: func(c *cli.Context) error {
				config.Load(configFile)
				daemon.Initialize()
				database.Open()
				//				defer database.Close()
				database.Initialize()
				auth.InitAdmin()
				return nil
			},
		},
	}

	app.Run(os.Args)
}
