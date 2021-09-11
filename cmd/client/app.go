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
			Name:      "run",
			ArgsUsage: "<name> <installer-id> <instance-id>",
			Usage:     "Run an application",
			Action: func(c *cli.Context) error {
				name := c.Args().Get(0)
				if name == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}

				installerID := c.Args().Get(1)
				if installerID == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}

				instanceID := c.Args().Get(2)
				if instanceID == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}

				return runApp(name, installerID, instanceID)
			},
		},
		{
			Name:      "start",
			ArgsUsage: "<name>",
			Usage:     "Start an application",
			Action: func(c *cli.Context) error {
				name := c.Args().Get(0)
				if name == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}

				return startApp(name)
			},
		},
		{
			Name:      "stop",
			ArgsUsage: "<name>",
			Usage:     "Stop an application",
			Action: func(c *cli.Context) error {
				name := c.Args().Get(0)
				if name == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}

				return stopApp(name)
			},
		},
		{
			Name:      "rm",
			ArgsUsage: "<name>",
			Usage:     "Remove an application",
			Action: func(c *cli.Context) error {
				name := c.Args().Get(0)
				if name == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}

				return removeApp(name)
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
	apps, err := envi.AM.GetAll()
	if err != nil {
		return err
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)

	defer w.Flush()

	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t", "Name", "ID", "Version", "Status", "Instance", "IP")
	fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t%s\t%s\t", "----", "--", "-------", "------", "--------", "--")
	for id, appi := range apps {
		fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t%s\t%s\t", appi.GetName(), id, appi.GetVersion(), appi.DesiredStatus, appi.InstanceName, appi.IP)
	}
	fmt.Fprint(w, "\n")

	return nil
}

func runApp(name string, installerID string, instanceID string) error {
	installer, err := envi.AS.GetInstaller(installerID)
	if err != nil {
		return err
	}

	instMetadata, err := installer.GetMetadata(installer.GetLastVersion())
	if err != nil {
		return err
	}

	// FIXME: read the installer params from the command line
	_, err = envi.AM.Create(installerID, installer.GetLastVersion(), name, instanceID, map[string]string{}, instMetadata)
	if err != nil {
		return err
	}

	return nil
}

func startApp(name string) error {

	err := envi.AM.Start(name)
	if err != nil {
		return err
	}

	return nil
}

func stopApp(name string) error {

	err := envi.AM.Stop(name)
	if err != nil {
		return err
	}

	return nil
}

func removeApp(name string) error {

	err := envi.AM.Remove(name)
	if err != nil {
		return err
	}

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
