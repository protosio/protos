package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/urfave/cli/v2"
)

var cmdResource *cli.Command = &cli.Command{
	Name:  "resource",
	Usage: "Manage resources",
	Subcommands: []*cli.Command{
		{
			Name:  "ls",
			Usage: "List resources",
			Action: func(c *cli.Context) error {
				return listResources()
			},
		},
		{
			Name:      "delete",
			ArgsUsage: "<id>",
			Usage:     "Delete an existing resource",
			Action: func(c *cli.Context) error {
				id := c.Args().Get(0)
				if id == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}

				return deleteResource(id)
			},
		},
	},
}

func listResources() error {
	resources := envi.RM.GetAll(false)

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)

	defer w.Flush()

	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t", "ID", "App", "Type", "Status")
	fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t", "----", "----", "----", "----")
	for id, rsc := range resources {
		fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t", id, rsc.GetAppID(), rsc.GetType(), rsc.GetStatus())
	}
	fmt.Fprint(w, "\n")

	return nil
}

func deleteResource(id string) error {

	err := envi.RM.Delete(id)
	if err != nil {
		return err
	}

	return nil
}
