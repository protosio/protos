package main

import (
	"context"
	"time"

	apic "github.com/protosio/protos/apic/proto"
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := client.StartVPN(ctx, &apic.StartVPNRequest{})
	if err != nil {
		return err
	}
	return nil
}

func stopVPN() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := client.StopVPN(ctx, &apic.StopVPNRequest{})
	if err != nil {
		return err
	}
	return nil
}
