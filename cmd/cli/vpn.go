package main

import (
	"github.com/protosio/protos/internal/vpn"
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
	v, err := vpn.New(envi.DB)
	if err != nil {
		return err
	}

	return v.Start()
}

func stopVPN() error {
	v, err := vpn.New(envi.DB)
	if err != nil {
		return err
	}

	return v.Stop()
}
