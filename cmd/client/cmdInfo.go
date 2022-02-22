package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/urfave/cli/v2"
)

var cmdInfo *cli.Command = &cli.Command{
	Name:  "info",
	Usage: "Various commands for displaying information about the user and local instance",
	Subcommands: []*cli.Command{
		{
			Name:  "devices",
			Usage: "List user devices",
			Action: func(c *cli.Context) error {
				return listDevices()
			},
		},
		{
			Name:  "user",
			Usage: "Display information about the user",
			Action: func(c *cli.Context) error {
				return userInfo()
			},
		},
	},
}

func listDevices() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetUserDevices(ctx, &pbApic.GetUserDevicesRequest{})
	if err != nil {
		return fmt.Errorf("failed to list user devices: %w", err)
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)

	defer w.Flush()

	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t", "Name", "Network", "WG Public Key", "ID")
	fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t", "----", "-------", "-------------", "--")
	for _, device := range resp.Devices {
		fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t", device.Name, device.Network, device.PublicKey, device.PublicKeyWireguard)
	}
	fmt.Fprint(w, "\n")

	return nil
}

func userInfo() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetUserInfo(ctx, &pbApic.GetUserInfoRequest{})
	if err != nil {
		return fmt.Errorf("failed to list user devices: %w", err)
	}

	fmt.Printf("Name: %s\n", resp.Name)
	fmt.Printf("Username: %s\n", resp.Username)
	fmt.Printf("Is admin: %t\n", resp.IsAdmin)

	return nil
}
