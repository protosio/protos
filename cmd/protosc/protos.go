package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/Masterminds/semver"
	"github.com/getlantern/systray"
	"github.com/protosio/protos/apic"
	"github.com/protosio/protos/cmd/protosc/icon"
	"github.com/protosio/protos/internal/util"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var log = util.GetLogger("protosc")
var stoppers = map[string]func() error{}

func stopServers() {
	for _, stopper := range stoppers {
		err := stopper()
		if err != nil {
			log.Error(err)
		}
	}
}

func handleQuitSignals(osSigs chan os.Signal, traySig chan struct{}) {
	select {
	case osSig := <-osSigs:
		log.Infof("Received OS signal %s. Terminating", osSig.String())
	case <-traySig:
		log.Info("Received tray quit signal. Terminating")
	}

	log.Info("Shutting down Protos client")
	stopServers()
	systray.Quit()
}

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
			log.Info("Starting Protos client")
			systray.Run(onReady, onExit)
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

func onReady() {
	systray.SetTemplateIcon(icon.Data, icon.Data)
	systray.SetTooltip("Protos")
	mQuitOrig := systray.AddMenuItem("Quit", "Quit")

	grpcStopper, err := apic.StartGRPCServer("/var/run/protos")
	if err != nil {
		log.Fatalf("Failed to start gRPC server: %s", err.Error())
	}
	stoppers["grpc"] = grpcStopper

	// Handle OS signals and tray icon quit signal
	osSigs := make(chan os.Signal, 1)
	signal.Notify(osSigs, syscall.SIGINT, syscall.SIGTERM)
	go handleQuitSignals(osSigs, mQuitOrig.ClickedCh)
}

func onExit() {
	log.Info("Shutdown complete")
}
