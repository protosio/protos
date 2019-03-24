package main

import (
	"os"

	"github.com/protosio/protos/daemon"
	"github.com/protosio/protos/util"

	"github.com/Masterminds/semver"
	"github.com/sirupsen/logrus"
	cli "gopkg.in/urfave/cli.v1"
)

func main() {

	app := cli.NewApp()
	app.Name = "protos"
	app.Author = "Alex Giurgiu"
	app.Email = "alex@giurgiu.io"
	version, err := semver.NewVersion("0.0.1-alpha.1")
	if err != nil {
		panic(err)
	}
	app.Version = version.String()

	var configFile string
	var loglevel string
	var incontainer bool
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
			Name:        "incontainer",
			Usage:       "When running in a container, tells Protos to not do any network initialization",
			Destination: &incontainer,
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
				daemon.StartUp(configFile, false, version, incontainer, devmode)
				return nil
			},
		},
		{
			Name:  "init",
			Usage: "create initial configuration and user",
			Action: func(c *cli.Context) error {
				daemon.StartUp(configFile, true, version, incontainer, devmode)
				return nil
			},
		},
	}

	app.Run(os.Args)
}
