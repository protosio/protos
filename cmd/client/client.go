package main

import (
	"os"

	"github.com/Masterminds/semver"
	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/protosio/protos/internal/util"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
)

var log = util.GetLogger("protos")
var client pbApic.ProtosClientApiClient

func main() {

	version, err := semver.NewVersion("0.1.0-dev.4")
	if err != nil {
		panic(err)
	}
	var loglevel string

	app := &cli.App{
		Name:    "protos",
		Usage:   "Protos cmd line client",
		Authors: []*cli.Author{{Name: "Alex Giurgiu", Email: "alex@giurgiu.io"}},
		Version: version.String(),
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
			cmdApp,
			cmdCloud,
			cmdInstance,
			cmdRelease,
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

	// connecting to the GRPC API first
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial("unix:///var/run/protos/protos.socket", opts...)
	if err != nil {
		log.Fatalf("Failed to connect to GRPC API: %s", err.Error())
	}
	defer conn.Close()
	client = pbApic.NewProtosClientApiClient(conn)

	err = app.Run(os.Args)
	if err != nil {
		log.Errorf("Error while running client: %s", err.Error())
	}
}
