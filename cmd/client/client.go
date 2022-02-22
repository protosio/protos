package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver"
	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/protosio/protos/internal/util"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
)

var log = util.GetLogger("protos")
var client pbApic.ProtosClientApiClient
var loglevel string
var unixSocket string

func main() {

	version, err := semver.NewVersion("0.1.0-dev.23")
	if err != nil {
		panic(err)
	}

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
			&cli.StringFlag{
				Name:        "unix-socket",
				Value:       "~/.protos/protos.socket",
				Usage:       "Path to unix socket API",
				Destination: &unixSocket,
			},
		},
		Commands: []*cli.Command{
			cmdInfo,
			cmdInit,
			cmdApp,
			cmdCloud,
			cmdInstance,
			cmdRelease,
		},
	}

	var conn *grpc.ClientConn

	app.Before = func(c *cli.Context) error {
		level, err := logrus.ParseLevel(loglevel)
		if err != nil {
			return err
		}
		util.SetLogLevel(level)

		homedir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal("Failed to retrieve home directory: %w", err)
		}

		if strings.HasPrefix(unixSocket, "~/") {
			unixSocket = filepath.Join(homedir, unixSocket[2:])
		}

		// connecting to the GRPC API first
		var opts []grpc.DialOption
		opts = append(opts, grpc.WithInsecure())
		conn, err = grpc.Dial("unix://"+unixSocket, opts...)
		if err != nil {
			log.Fatalf("Failed to connect to GRPC API: %s", err.Error())
		}
		client = pbApic.NewProtosClientApiClient(conn)

		return nil
	}

	app.After = func(c *cli.Context) error {
		return conn.Close()
	}

	err = app.Run(os.Args)
	if err != nil {
		log.Errorf("Error while running client: %s", err.Error())
	}
}
