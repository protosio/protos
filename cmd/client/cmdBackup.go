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

var cmdBackup *cli.Command = &cli.Command{
	Name:  "backup",
	Usage: "Manage backups",
	Subcommands: []*cli.Command{
		{
			Name:  "backends",
			Usage: "Subcommands to interact with the backup backends",
			Subcommands: []*cli.Command{
				{
					Name:  "ls",
					Usage: "List application backends",
					Action: func(c *cli.Context) error {
						return listBackupProviders()
					},
				},
				{
					Name:      "info",
					ArgsUsage: "<name>",
					Usage:     "Display extended information about a backup backend",
					Action: func(c *cli.Context) error {
						name := c.Args().Get(0)
						if name == "" {
							cli.ShowSubcommandHelp(c)
							os.Exit(1)
						}
						return infoBackupProviders(name)
					},
				},
			},
		},
		{
			Name:  "ls",
			Usage: "List backups",
			Action: func(c *cli.Context) error {
				return listBackups()
			},
		},
		{
			Name:      "info",
			ArgsUsage: "<name>",
			Usage:     "Display information about a backup",
			Action: func(c *cli.Context) error {
				name := c.Args().Get(0)
				if name == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}
				return infoBackup(name)
			},
		},
		{
			Name:      "create",
			ArgsUsage: "<name>",
			Usage:     "Create a backup",
			Action: func(c *cli.Context) error {
				name := c.Args().Get(0)
				if name == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}
				return createBackup(name)
			},
		},
		{
			Name:      "rm",
			ArgsUsage: "<name>",
			Usage:     "Remove backup",
			Action: func(c *cli.Context) error {
				name := c.Args().Get(0)
				if name == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}
				return rmBackup(name)
			},
		},
	},
}

func listBackupProviders() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	response, err := client.GetBackupProviders(ctx, &pbApic.GetBackupProvidersRequest{})
	if err != nil {
		return fmt.Errorf("failed to retrieve backup providers: %w", err)
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)

	defer w.Flush()

	fmt.Fprintf(w, " %s\t%s\t", "Name", "Type")
	fmt.Fprintf(w, "\n %s\t%s", "----", "----")
	for _, backupProvider := range response.BackupProviders {
		fmt.Fprintf(w, "\n %s\t%s\t", backupProvider.Name, backupProvider.Type)
	}
	fmt.Fprint(w, "\n")
	return nil
}

func infoBackupProviders(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	response, err := client.GetBackupProviderInfo(ctx, &pbApic.GetBackupProviderInfoRequest{Name: name})
	if err != nil {
		return fmt.Errorf("failed to retrieve backup providers: %w", err)
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintf(w, "%s\t%s\t", "Name:", response.BackupProvider.Name)
	fmt.Fprintf(w, "\n%s\t%s\t", "Type:", response.BackupProvider.Type)
	fmt.Fprint(w, "\n")

	return nil
}

func listBackups() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	backups, err := client.GetBackups(ctx, &pbApic.GetBackupsRequest{})
	if err != nil {
		return fmt.Errorf("failed to retrieve backups: %w", err)
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintf(w, " %s\t%s\t%s\t", "Name", "App", "Provider")
	fmt.Fprintf(w, "\n %s\t%s\t%s\t", "----", "---", "--------")
	for _, backup := range backups.Backups {
		fmt.Fprintf(w, "\n %s\t%s\t%s\t", backup.Name, backup.App, backup.Provider)
	}
	fmt.Fprint(w, "\n")
	return nil
}

func infoBackup(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	response, err := client.GetBackupInfo(ctx, &pbApic.GetBackupInfoRequest{Name: name})
	if err != nil {
		return fmt.Errorf("failed to retrieve backup info: %w", err)
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintf(w, "%s\t%s\t", "Name:", response.Backup.Name)
	fmt.Fprintf(w, "\n%s\t%s\t", "App:", response.Backup.App)
	fmt.Fprintf(w, "\n%s\t%s\t", "Provider:", response.Backup.Provider)
	fmt.Fprint(w, "\n")

	return nil
}

func createBackup(name string) error {
	return nil
}

func rmBackup(name string) error {
	return nil
}
