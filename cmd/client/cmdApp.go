package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	pbApic "github.com/protosio/protos/apic/proto"
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
			Name:      "create",
			ArgsUsage: "<name> <installer> <instance-id>",
			Usage:     "Create an application",
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

				return createApp(name, installerID, instanceID)
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetApps(ctx, &pbApic.GetAppsRequest{})
	if err != nil {
		return fmt.Errorf("failed to list apps: %w", err)
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)

	defer w.Flush()

	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t", "Name", "ID", "Installer", "Desired state", "Instance", "IP")
	fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t%s\t%s\t", "----", "--", "---------", "-------------", "--------", "--")
	for _, appi := range resp.Apps {
		fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t%s\t%s\t", appi.Name, appi.Id, appi.Installer, appi.DesiredStatus, appi.InstanceName, appi.Ip)
	}
	fmt.Fprint(w, "\n")

	return nil
}

func createApp(name string, installerID string, instanceID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := client.CreateApp(ctx, &pbApic.CreateAppRequest{Name: name, InstallerId: installerID, InstanceId: instanceID})
	if err != nil {
		return fmt.Errorf("failed to run app '%s': %w", name, err)
	}
	return nil
}

func startApp(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := client.StartApp(ctx, &pbApic.StartAppRequest{Name: name})
	if err != nil {
		return fmt.Errorf("failed to start app '%s': %w", name, err)
	}
	return nil
}

func stopApp(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := client.StopApp(ctx, &pbApic.StopAppRequest{Name: name})
	if err != nil {
		return fmt.Errorf("failed to stop app '%s': %w", name, err)
	}
	return nil
}

func removeApp(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := client.RemoveApp(ctx, &pbApic.RemoveAppRequest{Name: name})
	if err != nil {
		return fmt.Errorf("failed to remove app '%s': %w", name, err)
	}
	return nil
}

func listAppStoreApps() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetInstallers(ctx, &pbApic.GetInstallersRequest{})
	if err != nil {
		return fmt.Errorf("failed to retrieve installers: %w", err)
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)

	defer w.Flush()

	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t", "Name", "ID", "Version", "Description")
	fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t", "----", "--", "-------", "-----------")
	for _, installer := range resp.Installers {
		fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t", installer.Name, installer.Id, installer.Version, installer.Description)
	}
	fmt.Fprint(w, "\n")
	return nil
}

func infoAppStoreApp(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetInstaller(ctx, &pbApic.GetInstallerRequest{Id: id})
	if err != nil {
		return fmt.Errorf("failed to retrieve installers: %w", err)
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)

	defer w.Flush()

	fmt.Fprintf(w, "%s\t%s\t", "Name:", resp.Installer.Name)
	fmt.Fprintf(w, "\n%s\t%s\t", "ID:", id)
	fmt.Fprintf(w, "\n%s\t%s\t", "Description:", resp.Installer.Description)
	fmt.Fprintf(w, "\n%s\t%s\t", "Version: ", resp.Installer.Version)
	fmt.Fprintf(w, "\n%s\t%s\t", "Requires resources: ", strings.Join(resp.Installer.RequiresResources, ","))
	fmt.Fprintf(w, "\n%s\t%s\t", "Provides resources: ", strings.Join(resp.Installer.ProvidesResources, ","))
	fmt.Fprintf(w, "\n%s\t%s\t", "Capabilities: ", strings.Join(resp.Installer.Capabilities, ","))
	fmt.Fprint(w, "\n")

	return nil
}
