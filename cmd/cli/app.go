package main

import (
	// "github.com/protosio/cli/internal/app"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/urfave/cli/v2"
)

var cmdApp *cli.Command = &cli.Command{
	Name:  "app",
	Usage: "Manage applications",
	Subcommands: []*cli.Command{
		{
			Name:  "ls",
			Usage: "List installed applications",
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
			Name:  "store",
			Usage: "Subcommands to interact with the app store",
			Subcommands: []*cli.Command{
				{
					Name:  "ls",
					Usage: "List all applications in the store",
					Action: func(c *cli.Context) error {
						return listAppStoreApps()
					},
				},
				{
					Name:      "info",
					ArgsUsage: "<id>",
					Usage:     "Display extended information about an app from the store",
					Action: func(c *cli.Context) error {
						id := c.Args().Get(0)
						if id == "" {
							cli.ShowSubcommandHelp(c)
							os.Exit(1)
						}
						return infoAppStoreApp(id)
					},
				},
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

func listAppStoreApps() error {
	installers, err := envi.AS.GetInstallers()
	if err != nil {
		return err
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)

	defer w.Flush()

	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t", "Name", "ID", "Version", "Description")
	fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t", "----", "--", "-------", "-----------")
	for id, installer := range installers {
		instMetadata, err := installer.GetMetadata(installer.GetLastVersion())
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t", installer.GetName(), id, installer.GetLastVersion(), instMetadata.Description)
	}
	fmt.Fprint(w, "\n")
	return nil
}

func infoAppStoreApp(id string) error {
	installer, err := envi.AS.GetInstaller(id)
	if err != nil {
		return err
	}

	instMetadata, err := installer.GetMetadata(installer.GetLastVersion())
	if err != nil {
		return err
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)

	defer w.Flush()

	fmt.Fprintf(w, "%s\t%s\t", "Name:", installer.GetName())
	fmt.Fprintf(w, "\n%s\t%s\t", "ID:", id)
	fmt.Fprintf(w, "\n%s\t%s\t", "Description:", instMetadata.Description)
	fmt.Fprintf(w, "\n%s\t%s\t", "Version: ", installer.GetLastVersion())
	fmt.Fprintf(w, "\n%s\t%s\t", "Requires resources: ", strings.Join(instMetadata.Requires, ","))
	fmt.Fprintf(w, "\n%s\t%s\t", "Provides resources: ", strings.Join(instMetadata.Provides, ","))
	fmt.Fprintf(w, "\n%s\t%s\t", "Capabilities: ", strings.Join(instMetadata.Capabilities, ","))
	fmt.Fprint(w, "\n")

	return nil
}
