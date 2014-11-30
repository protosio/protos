package main

import (
	"github.com/codegangsta/cli"
	"os"
	"egor/daemon"
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
				daemon.StartApp(c.Args().First())
			},
		},
		{
			Name:  "stop",
			Usage: "stops an application",
			Action: func(c *cli.Context) {
				daemon.StopApp(c.Args().First())
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
				daemon.LoadAppCfg(c.Args().First())
			},
		},
	}

	app.Run(os.Args)
}
