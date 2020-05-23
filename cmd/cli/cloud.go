package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/core"
	"github.com/urfave/cli/v2"
)

const cloudDS = "cloud"

var cmdCloud *cli.Command = &cli.Command{
	Name:  "cloud",
	Usage: "Manage cloud providers",
	Subcommands: []*cli.Command{
		{
			Name:  "ls",
			Usage: "List existing cloud provider accounts",
			Action: func(c *cli.Context) error {
				return listCloudProviders()
			},
		},
		{
			Name:      "add",
			ArgsUsage: "<name>",
			Usage:     "Add a new cloud provider account",
			Action: func(c *cli.Context) error {
				name := c.Args().Get(0)
				if name == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}
				_, err := addCloudProvider(name)
				return err
			},
		},
		{
			Name:      "delete",
			ArgsUsage: "<name>",
			Usage:     "Delete an existing cloud provider account",
			Action: func(c *cli.Context) error {
				name := c.Args().Get(0)
				if name == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}
				return deleteCloudProvider(name)
			},
		},
		{
			Name:      "info",
			ArgsUsage: "<name>",
			Usage:     "Prints info about cloud provider account and checks if the API is reachable",
			Action: func(c *cli.Context) error {
				name := c.Args().Get(0)
				if name == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}
				return infoCloudProvider(name)
			},
		},
	},
}

//
//  Cloud provider methods
//

func listCloudProviders() error {

	clouds, err := envi.CLM.GetProviders()
	if err != nil {
		return err
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 16, 16, 0, '\t', 0)

	defer w.Flush()

	fmt.Fprintf(w, " %s\t%s\t", "Name", "Type")
	fmt.Fprintf(w, "\n %s\t%s\t", "----", "----")
	for _, cl := range clouds {
		fmt.Fprintf(w, "\n %s\t%s\t", cl.NameStr(), cl.TypeStr())
	}
	fmt.Fprint(w, "\n")
	return nil
}

func addCloudProvider(cloudName string) (core.CloudProvider, error) {

	// select cloud provider
	var cloudType string
	cloudProviderSelect := surveySelect(envi.CLM.SupportedProviders(), "Choose one of the following supported cloud providers:")
	err := survey.AskOne(cloudProviderSelect, &cloudType)
	if err != nil {
		return nil, err
	}

	// create new cloud provider
	provider, err := envi.CLM.NewProvider(cloudName, cloudType)
	if err != nil {
		return nil, err
	}

	// get cloud provider credentials
	cloudCredentials := map[string]interface{}{}
	credFields := provider.AuthFields()
	credentialsQuestions := getCloudCredentialsQuestions(cloudType, credFields)

	err = survey.Ask(credentialsQuestions, &cloudCredentials)
	if err != nil {
		return nil, err
	}

	// set authentication
	err = provider.SetAuth(transformCredentials(cloudCredentials))
	if err != nil {
		return nil, err
	}

	// init cloud client
	err = provider.Init()
	if err != nil {
		return nil, err
	}

	// save the cloud provider in the db
	err = provider.Save()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to save cloud provider info")
	}

	return provider, nil
}

func deleteCloudProvider(name string) error {
	return envi.CLM.DeleteProvider(name)
}

func infoCloudProvider(name string) error {
	provider, err := envi.CLM.GetProvider(name)
	if err != nil {
		return errors.Wrapf(err, "Could not retrieve cloud '%s'", name)
	}

	locations := provider.SupportedLocations()
	err = provider.Init()
	if err != nil {
		log.Error(errors.Wrapf(err, "Error reaching cloud provider '%s'(%s) API", name, provider.TypeStr()))
	}
	machineTypes, err := provider.SupportedMachines(locations[0])
	if err != nil {
		log.Error(errors.Wrapf(err, "Error reaching cloud provider '%s'(%s) API", name, provider.TypeStr()))
	}

	fmt.Printf("Name: %s\n", name)
	fmt.Printf("Type: %s\n", provider.TypeStr())
	fmt.Printf("Supported locations: %s\n", strings.Join(locations, " | "))
	fmt.Printf("Supported machine types: \n")
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 8, 8, 0, ' ', 0)
	for instanceID, instanceSpec := range machineTypes {
		fmt.Fprintf(w, "    %s\t -  Nr of CPUs: %d,\t Memory: %d MiB,\t Storage: %d GB\t\n", instanceID, instanceSpec.Cores, instanceSpec.Memory, instanceSpec.DefaultStorage)
	}
	w.Flush()
	if err != nil {
		fmt.Printf("Status: NOT OK (%s)\n", err.Error())
	} else {
		fmt.Printf("Status: OK - API reachable\n")
	}
	return nil
}
