package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/cloud"
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
	var clouds []cloud.ProviderInfo
	err := envi.DB.GetSet(cloudDS, &clouds)
	if err != nil {
		return err
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 16, 16, 0, '\t', 0)

	defer w.Flush()

	fmt.Fprintf(w, " %s\t%s\t", "Name", "Type")
	fmt.Fprintf(w, "\n %s\t%s\t", "----", "----")
	for _, cl := range clouds {
		fmt.Fprintf(w, "\n %s\t%s\t", cl.Name, cl.Type)
	}
	fmt.Fprint(w, "\n")
	return nil
}

func addCloudProvider(cloudName string) (cloud.Provider, error) {
	// select cloud provider
	var cloudType string
	cloudProviderSelect := surveySelect(cloud.SupportedProviders(), "Choose one of the following supported cloud providers:")
	err := survey.AskOne(cloudProviderSelect, &cloudType)
	if err != nil {
		return nil, err
	}

	// create new cloud provider
	client, err := cloud.NewProvider(cloudName, cloudType)
	if err != nil {
		return nil, err
	}

	// get cloud provider credentials
	cloudCredentials := map[string]interface{}{}
	credFields := client.AuthFields()
	credentialsQuestions := getCloudCredentialsQuestions(cloudType, credFields)

	err = survey.Ask(credentialsQuestions, &cloudCredentials)
	if err != nil {
		return nil, err
	}

	// init cloud client
	err = client.Init(transformCredentials(cloudCredentials))
	if err != nil {
		return nil, err
	}

	// save the cloud provider in the db
	cloudProviderInfo := client.GetInfo()
	err = envi.DB.InsertInSet(cloudDS, cloudProviderInfo)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to save cloud provider info")
	}

	return client, nil
}

func getCloudProvider(name string) (cloud.ProviderInfo, error) {
	clouds := []cloud.ProviderInfo{}
	err := envi.DB.GetSet(cloudDS, &clouds)
	if err != nil {
		return cloud.ProviderInfo{}, err
	}
	for _, cld := range clouds {
		if cld.Name == name {
			return cld, nil
		}
	}
	return cloud.ProviderInfo{}, fmt.Errorf("Could not find cloud provider '%s'", name)
}

func deleteCloudProvider(name string) error {
	cld, err := getCloudProvider(name)
	if err != nil {
		return err
	}
	err = envi.DB.RemoveFromSet(cloudDS, cld)
	if err != nil {
		return err
	}

	return nil
}

func infoCloudProvider(name string) error {
	cloud, err := getCloudProvider(name)
	if err != nil {
		return errors.Wrapf(err, "Could not retrieve cloud '%s'", name)
	}
	client := cloud.Client()
	locations := client.SupportedLocations()
	err = client.Init(cloud.Auth)
	if err != nil {
		log.Error(errors.Wrapf(err, "Error reaching cloud provider '%s'(%s) API", name, cloud.Type.String()))
	}
	machineTypes, err := client.SupportedMachines(locations[0])
	if err != nil {
		log.Error(errors.Wrapf(err, "Error reaching cloud provider '%s'(%s) API", name, cloud.Type.String()))
	}
	fmt.Printf("Name: %s\n", cloud.Name)
	fmt.Printf("Type: %s\n", cloud.Type.String())
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
