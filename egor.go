package main

import (
	"egor/daemon"
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"os"
)

func main() {

	app := cli.NewApp()
	app.Name = "egor"
	app.Usage = "iz good for your privacy"
	app.Author = "Alex Giurgiu"
	app.Email = "alex@giurgiu.io"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "c, config",
			Value: "egor.yaml",
			Usage: "Specify a config file (default: egor.yaml)",
		},
		cli.StringFlag{
			Name:  "l, loglevel",
			Value: "info",
			Usage: "Specify log level: debug, info, warn, error (default: info)",
		},
	}

	app.Before = func(c *cli.Context) error {
		daemon.LoadCfg(c.String("config"))
		if c.String("loglevel") == "debug" {
			daemon.SetLogLevel(log.DebugLevel)
		} else if c.String("loglevel") == "info" {
			daemon.SetLogLevel(log.InfoLevel)
		} else if c.String("loglevel") == "warn" {
			daemon.SetLogLevel(log.WarnLevel)
		} else if c.String("loglevel") == "error" {
			daemon.SetLogLevel(log.ErrorLevel)
		}
		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:  "start",
			Usage: "starts an application",
			Action: func(c *cli.Context) {
				app_name := c.Args().First()
				app := daemon.GetApp(app_name)
				app.Start()
			},
		},
		{
			Name:  "stop",
			Usage: "stops an application",
			Action: func(c *cli.Context) {
				app_name := c.Args().First()
				app := daemon.GetApp(app_name)
				app.Stop()
			},
		},
		{
			Name:  "daemon",
			Usage: "starts http server",
			Action: func(c *cli.Context) {
				daemon.Websrv()
			},
		},
		{
			Name:  "validate",
			Usage: "validates application config",
			Action: func(c *cli.Context) {
				app_name := c.Args().First()
				app := daemon.GetApp(app_name)
				app.LoadCfg()

			},
		},
		{
			Name:  "list",
			Usage: "list applications",
			Action: func(c *cli.Context) {
				apps := daemon.GetApps()
				for _, app := range apps {
					log.Println(app.Name)
				}
			},
		},
	}

	app.Run(os.Args)
}
