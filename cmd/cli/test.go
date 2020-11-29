package main

import (
	"github.com/protosio/protos/internal/p2p"
	"github.com/urfave/cli/v2"
)

var cmdTest *cli.Command = &cli.Command{
	Name:  "test",
	Usage: "test",
	Subcommands: []*cli.Command{
		{
			Name:  "srv",
			Usage: "srv",
			Action: func(c *cli.Context) error {
				user, err := envi.UM.GetAdmin()
				if err != nil {
					return err
				}

				key, err := user.GetKeyCurrentDevice()
				if err != nil {
					return err
				}

				m, err := p2p.NewManager(10500, key)
				if err != nil {
					return err
				}
				return m.Listen()
			},
		},
		{
			Name:  "client",
			Usage: "client",
			Action: func(c *cli.Context) error {
				user, err := envi.UM.GetAdmin()
				if err != nil {
					return err
				}

				key, err := user.GetKeyCurrentDevice()
				if err != nil {
					return err
				}
				m, err := p2p.NewManager(10600, key)
				if err != nil {
					return err
				}

				m.AddPeer("12D3KooWCrB97oqtJctK2Dk8zxFyPrWhKdYG4s5cyeABHw4Q9qCs", c.Args().First())

				return m.Connect(c.Args().First())
			},
		},
	},
}
