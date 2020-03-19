package main

import (
	"os"

	"github.com/protosio/protos/internal/daemon"
	"github.com/protosio/protos/internal/util"

	"github.com/Masterminds/semver"
	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli"
)

func main() {

	app := cli.NewApp()
	app.Name = "protosd"
	app.Author = "Alex Giurgiu"
	app.Email = "alex@giurgiu.io"
	version, err := semver.NewVersion("0.0.0-dev.3")
	if err != nil {
		panic(err)
	}
	app.Version = version.String()

	var configFile string
	var loglevel string
	var devmode bool

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
		cli.BoolFlag{
			Name:        "dev",
			Usage:       "Allows unauthenticated dev operations via the API",
			Destination: &devmode,
		},
	}

	app.Before = func(c *cli.Context) error {
		level, err := logrus.ParseLevel(loglevel)
		if err != nil {
			return err
		}
		util.SetLogLevel(level)
		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:  "daemon",
			Usage: "start the server",
			Action: func(c *cli.Context) error {
				daemon.StartUp(configFile, false, version, devmode)
				return nil
			},
		},
		{
			Name:  "init",
			Usage: "create initial configuration and user",
			Action: func(c *cli.Context) error {
				daemon.StartUp(configFile, true, version, devmode)
				return nil
			},
		},
	}

	app.Run(os.Args)
}
