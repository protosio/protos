package main

import (
	"os"
	"protos/daemon"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/urfave/cli.v1"
)

func main() {

	app := cli.NewApp()
	app.Name = "protos"
	app.Usage = "self hosting platform"
	app.Author = "Alex Giurgiu"
	app.Email = "alex@giurgiu.io"

	var config string
	var loglevel string

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "config",
			Value:       "protos.yaml",
			Usage:       "Specify a config file",
			Destination: &config,
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
			daemon.SetLogLevel(log.DebugLevel)
		} else if config == "info" {
			daemon.SetLogLevel(log.InfoLevel)
		} else if config == "warn" {
			daemon.SetLogLevel(log.WarnLevel)
		} else if config == "error" {
			daemon.SetLogLevel(log.ErrorLevel)
		}
		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:  "daemon",
			Usage: "start the server",
			Action: func(c *cli.Context) error {
				daemon.StartUp(config)
				daemon.LoadApps()
				daemon.Websrv()
				return nil
			},
		},
		{
			Name:  "init",
			Usage: "create initial configuration and user",
			Action: func(c *cli.Context) error {
				daemon.LoadCfg(config)
				daemon.Initialize()
				return nil
			},
		},
	}

	app.Run(os.Args)
}
