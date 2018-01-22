package main

import (
	"os"
	"protos/api"
	"protos/auth"
	"protos/capability"
	"protos/config"
	"protos/daemon"
	"protos/database"
	"protos/util"
	"sync"

	"github.com/Sirupsen/logrus"
	"gopkg.in/urfave/cli.v1"
)

func run(configFile string) {
	var wg sync.WaitGroup
	config.Load(configFile)
	database.Open()
	capability.Initialize()
	defer database.Close()
	daemon.StartUp()
	daemon.LoadApps()
	wg.Add(2)
	go func() {
		auth.LDAPsrv()
		wg.Done()
	}()
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
				daemon.Initialize()
				database.Open()
				defer database.Close()
				auth.InitAdmin()
				return nil
			},
		},
	}

	app.Run(os.Args)
}
