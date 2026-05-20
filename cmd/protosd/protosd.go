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
	version, err := semver.NewVersion("0.1.0-dev.23")
	if err != nil {
		panic(err)
	}
	app.Version = version.String()

	var configFile string
	var dataDir string
	var capabilities string
	var loglevel string

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "config",
			Value:       "protos.yaml",
			Usage:       "Specify a config file",
			Destination: &configFile,
		},
		&cli.StringFlag{
			Name:        "data-dir",
			Usage:       "Path where protos data is stored",
			Destination: &dataDir,
		},
		&cli.StringFlag{
			Name:        "capabilities",
			Usage:       "Comma-separated capabilities: api,provisioner,network,app-runtime",
			Destination: &capabilities,
		},
		&cli.StringFlag{
			Name:        "loglevel",
			Value:       "info",
			Usage:       "Specify log level: debug, info, warn, error",
			Destination: &loglevel,
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
		protosd.StartUp(configFile, version, protosd.Options{
			DataDir:      dataDir,
			Capabilities: capabilities,
		})
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
