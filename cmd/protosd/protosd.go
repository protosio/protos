package main

import (
	"os"

	"github.com/protosio/protos/internal/protosd"
	"github.com/protosio/protos/internal/util"

	"github.com/Masterminds/semver"
	"github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
)

var log = util.GetLogger("protosd")

func main() {

	app := cli.NewApp()
	app.Name = "protosd"
	app.Authors = []*cli.Author{{Name: "Alex Giurgiu", Email: "alex@giurgiu.io"}}
	version, err := semver.NewVersion("0.1.0-dev.4")
	if err != nil {
		panic(err)
	}
	app.Version = version.String()

	var configFile string
	var loglevel string
	var devmode bool

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "config",
			Value:       "protos.yaml",
			Usage:       "Specify a config file",
			Destination: &configFile,
		},
		&cli.StringFlag{
			Name:        "loglevel",
			Value:       "info",
			Usage:       "Specify log level: debug, info, warn, error",
			Destination: &loglevel,
		},
		&cli.BoolFlag{
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

	app.Action = func(c *cli.Context) error {
		log.Info("Starting Protos daemon")
		protosd.StartUp(configFile, false, version, devmode)
		return nil
	}

	app.Run(os.Args)
}
