package main

import (
	"os"

	"github.com/Masterminds/semver"
	"github.com/getlantern/systray"
	"github.com/protosio/protos/apic"
	"github.com/protosio/protos/cmd/protosc/icon"
	"github.com/protosio/protos/internal/util"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = util.GetLogger("protosClient")

func main() {

	version, err := semver.NewVersion("0.1.0-dev.4")
	if err != nil {
		panic(err)
	}

	app := &cli.App{
		Name:    "protosc",
		Usage:   "Protos client",
		Authors: []*cli.Author{{Name: "Alex Giurgiu", Email: "alex@giurgiu.io"}},
		Version: version.String(),
		Action: func(c *cli.Context) error {
			start()
			return nil
		},
	}

	var loglevel string

	app.Flags = []cli.Flag{
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

	app.Run(os.Args)

}

func start() {
	onExit := func() {
		log.Info("Shutdown complete")
	}

	log.Info("Starting Protos client")
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTemplateIcon(icon.Data, icon.Data)
	systray.SetTooltip("Protos")
	mQuitOrig := systray.AddMenuItem("Quit", "Quit")

	grpcStopper, err := apic.StartGRPCServer("/var/run/protos")
	if err != nil {
		log.Fatalf("Failed to start gRPC server: %s", err.Error())
	}

	go func() {
		<-mQuitOrig.ClickedCh
		log.Info("Shutting down Protos client")
		grpcStopper()
		systray.Quit()
	}()

}
