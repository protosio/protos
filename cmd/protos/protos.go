package main

import (
	"os"
	"sync"

	"github.com/protosio/protos/app"

	"github.com/protosio/protos/platform"
	"github.com/protosio/protos/provider"
	"github.com/protosio/protos/resource"

	"github.com/protosio/protos/api"
	"github.com/protosio/protos/auth"
	"github.com/protosio/protos/capability"
	"github.com/protosio/protos/config"
	"github.com/protosio/protos/daemon"
	"github.com/protosio/protos/database"
	"github.com/protosio/protos/meta"
	"github.com/protosio/protos/util"

	"github.com/sirupsen/logrus"
	cli "gopkg.in/urfave/cli.v1"
)

func run(configFile string) {
	var wg sync.WaitGroup
	config.Load(configFile)
	log := util.GetLogger()

	if database.Exists() {
		database.Open()
		defer database.Close()
		capability.Initialize()
		platform.Initialize()      // required to connect to the Docker daemon
		app.LoadAppsDB()           // required to register the application structs with the DB
		resource.LoadResourcesDB() // required to register the resource structs with the DB
		provider.LoadProvidersDB() // required to register the provider structs with the DB
		cert := meta.Initialize()

		daemon.StartUp()
		wg.Add(2)
		go func() {
			api.Websrv(cert)
			wg.Done()
		}()
		wg.Wait()
	} else {
		log.Info("Database file doesn't exists. Running in web init mode")
		database.Open()
		defer database.Close()
		capability.Initialize()

		meta.Setup()
		meta.SetPublicIP()

		platform.Initialize()      // required to connect to the Docker daemon
		app.LoadAppsDB()           // required to register the application structs with the DB
		resource.LoadResourcesDB() // required to register the resource structs with the DB
		provider.LoadProvidersDB() // required to register the provider structs with the DB
		daemon.StartUp()
		wg.Add(2)
		go func() {
			api.Websrv(nil)
			wg.Done()
		}()
		wg.Wait()

	}

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
