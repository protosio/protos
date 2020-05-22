package main

import (
	"encoding/base64"
	"fmt"
	"os"

	ssh "github.com/protosio/protos/internal/ssh"
	"github.com/urfave/cli/v2"
)

var cmdUser *cli.Command = &cli.Command{
	Name:  "user",
	Usage: "Manage local user details",
	Subcommands: []*cli.Command{
		{
			Name:  "info",
			Usage: "Prints info about the local user configured during init",
			Action: func(c *cli.Context) error {
				return infoUser()
			},
		},
		{
			Name:  "set",
			Usage: "Allows you to modify details about the user (name and domain supported)",
			Subcommands: []*cli.Command{
				{
					Name:  "name",
					Usage: "Set new `NAME` for the user",
					Action: func(c *cli.Context) error {
						name := c.Args().Get(0)
						if name == "" {
							cli.ShowSubcommandHelp(c)
							os.Exit(1)
						}
						usr, err := envi.UM.GetAdmin()
						if err != nil {
							return err
						}
						return usr.SetName(name)
					},
				},
				{
					Name:  "domain",
					Usage: "Set new `DOMAIN` for the user",
					Action: func(c *cli.Context) error {
						domain := c.Args().Get(0)
						if domain == "" {
							cli.ShowSubcommandHelp(c)
							os.Exit(1)
						}
						usr, err := envi.UM.GetAdmin()
						if err != nil {
							return err
						}
						return usr.SetDomain(domain)
					},
				},
			},
		},
	},
}

func infoUser() error {
	user, err := envi.UM.GetAdmin()
	if err != nil {
		return err
	}

	userInfo := user.GetInfo()

	// FIXME: current device and current key should be togheter
	dev := user.GetCurrentDevice()
	keySeed, err := user.GetKeyCurrentDevice()
	if err != nil {
		return err
	}

	key, err := ssh.NewKeyFromSeed(keySeed)
	if err != nil {
		return err
	}

	encodedPrivateKey := base64.StdEncoding.EncodeToString(key.Seed())
	fmt.Printf("Username: %s\n", userInfo.Username)
	fmt.Printf("Name: %s\n", userInfo.Name)
	fmt.Printf("Domain: %s\n", userInfo.Domain)
	fmt.Printf("Device name: %s\n", dev.Name)
	fmt.Printf("Device private key: %s\n", encodedPrivateKey)
	fmt.Printf("Device public key (wireguard): %s\n", key.PublicWG().String())
	fmt.Printf("Device network: %s\n", dev.Network)
	return nil
}
