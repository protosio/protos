package main

import (
	"egor/daemon"
	"github.com/codegangsta/cli"
	"log"
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
	}

	app.Before = func(c *cli.Context) error {
		daemon.LoadCfg(c.String("config"))
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
