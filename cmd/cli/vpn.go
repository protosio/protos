package main

import (
	"github.com/urfave/cli/v2"
)

var cmdVPN *cli.Command = &cli.Command{
	Name:  "vpn",
	Usage: "Manage VPN",
	Subcommands: []*cli.Command{
		{
			Name:  "start",
			Usage: "Start the VPN",
			Action: func(c *cli.Context) error {
				return startVPN()
			},
		},
		{
			Name:  "stop",
			Usage: "Stop the VPN",
			Action: func(c *cli.Context) error {
				return stopVPN()
			},
		},
	},
}

func startVPN() error {
	return nil
}

func stopVPN() error {
	return nil
}
