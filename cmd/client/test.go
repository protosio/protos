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
				_, err = m.Listen()
				if err != nil {
					return err
				}
				<-make(chan struct{})
				return nil
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

				m.AddPeer([]byte{}, c.Args().First())

				// return m.Connect(c.Args().First())
				return nil
			},
		},
	},
}
