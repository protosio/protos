package main

import (
	// "github.com/protosio/cli/internal/app"
	"github.com/urfave/cli/v2"
)

var cmdApp *cli.Command = &cli.Command{
	Name:  "app",
	Usage: "Manage applications",
	Subcommands: []*cli.Command{
		{
			Name:  "ls",
			Usage: "List applications",
			Action: func(c *cli.Context) error {
				return listApps()
			},
		},
		{
			Name:  "run",
			Usage: "Run a new application",
			Action: func(c *cli.Context) error {
				return listApps()
			},
		},
		{
			Name:  "search",
			Usage: "Searches the app store for applications",
			Action: func(c *cli.Context) error {
				return listApps()
			},
		},
	},
}

func listApps() error {
	// prv := app.NewProvider()
	// apps := prv.GetApps()
	// fmt.Println(apps)
	return nil
}
