package main

import (
	"os"
	"sync"

	"github.com/nustiueudinastea/protos/api"
	"github.com/nustiueudinastea/protos/auth"
	"github.com/nustiueudinastea/protos/capability"
	"github.com/nustiueudinastea/protos/config"
	"github.com/nustiueudinastea/protos/daemon"
	"github.com/nustiueudinastea/protos/database"
	"github.com/nustiueudinastea/protos/meta"
	"github.com/nustiueudinastea/protos/util"

	"github.com/Sirupsen/logrus"
	"gopkg.in/urfave/cli.v1"
)

func run(configFile string) {
	var wg sync.WaitGroup
	config.Load(configFile)
	database.Open()
	defer database.Close()
	capability.Initialize()
	meta.Initialize()
	daemon.StartUp()
	daemon.LoadAppsDB()
	wg.Add(2)
	// go func() {
	// 	auth.LDAPsrv()
	// 	wg.Done()
	// }()
	go func() {
		api.Websrv()
		wg.Done()
	}()
	wg.Wait()
}

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
				run(configFile)
				return nil
			},
		},
		{
			Name:  "init",
			Usage: "create initial configuration and user",
			Action: func(c *cli.Context) error {
				config.Load(configFile)
				daemon.Setup()
				database.Open()
				defer database.Close()
				meta.Setup()
				auth.SetupAdmin()
				return nil
			},
		},
	}

	app.Run(os.Args)
}
