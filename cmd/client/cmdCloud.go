package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	survey "github.com/AlecAivazis/survey/v2"
	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/urfave/cli/v2"
)

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
				err := addCloudProvider(name)
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

func getCloudCredentialsQuestions(providerName string, fields []string) []*survey.Question {
	qs := []*survey.Question{}
	for _, field := range fields {
		qs = append(qs, &survey.Question{
			Name:     field,
			Prompt:   &survey.Input{Message: providerName + " " + field + ":"},
			Validate: survey.Required})
	}
	return qs
}

func transformCredentials(creds map[string]interface{}) map[string]string {
	transformed := map[string]string{}
	for name, val := range creds {
		transformed[name] = val.(string)
	}
	return transformed
}

//
//  Cloud provider methods
//

func listCloudProviders() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetCloudProviders(ctx, &pbApic.GetCloudProvidersRequest{})
	if err != nil {
		return fmt.Errorf("failed to retrieve cloud providers: %w", err)
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 16, 16, 0, '\t', 0)

	defer w.Flush()

	fmt.Fprintf(w, " %s\t%s\t", "Name", "Type")
	fmt.Fprintf(w, "\n %s\t%s\t", "----", "----")
	for _, cl := range resp.CloudProviders {
		fmt.Fprintf(w, "\n %s\t%s\t", cl.Name, cl.Type.Name)
	}
	fmt.Fprint(w, "\n")
	return nil
}

func addCloudProvider(cloudName string) error {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetSupportedCloudProviders(ctx, &pbApic.GetSupportedCloudProvidersRequest{})
	if err != nil {
		return fmt.Errorf("failed to retrieve supported cloud providers: %w", err)
	}

	supportedCloudTypes := make([]string, len(resp.CloudTypes))
	supportedCloudTypesCredentials := make(map[string][]string, len(resp.CloudTypes))
	i := 0
	for _, cloudType := range resp.CloudTypes {
		supportedCloudTypes[i] = cloudType.Name
		supportedCloudTypesCredentials[cloudType.Name] = cloudType.AuthenticationFields
		i++
	}

	// select cloud provider
	var cloudType string
	cloudProviderSelect := surveySelect(supportedCloudTypes, "Choose one of the following supported cloud providers:")
	err = survey.AskOne(cloudProviderSelect, &cloudType)
	if err != nil {
		return err
	}

	// get cloud provider credentials
	cloudCredentials := map[string]interface{}{}
	credentialsQuestions := getCloudCredentialsQuestions(cloudType, supportedCloudTypesCredentials[cloudType])
	err = survey.Ask(credentialsQuestions, &cloudCredentials)
	if err != nil {
		return err
	}

	// add cloud provider
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = client.AddCloudProvider(ctx, &pbApic.AddCloudProviderRequest{
		Name:        cloudName,
		Type:        cloudType,
		Credentials: transformCredentials(cloudCredentials),
	})
	if err != nil {
		return fmt.Errorf("failed to add cloud provider: %w", err)
	}

	return nil
}

func deleteCloudProvider(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := client.RemoveCloudProvider(ctx, &pbApic.RemoveCloudProviderRequest{Name: name})
	if err != nil {
		return fmt.Errorf("failed to remove cloud provider '%s': %w", name, err)
	}
	return nil
}

func infoCloudProvider(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetCloudProvider(ctx, &pbApic.GetCloudProviderRequest{Name: name})
	if err != nil {
		return fmt.Errorf("failed to retrieve cloud provider '%s': %w", name, err)
	}

	fmt.Printf("Name: %s\n", name)
	fmt.Printf("Type: %s\n", resp.CloudProvider.Type.Name)
	fmt.Printf("Supported locations: %s\n", strings.Join(resp.CloudProvider.SupportedLocations, " | "))
	fmt.Printf("Supported machine types: \n")
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 8, 8, 0, ' ', 0)
	for instanceID, instanceSpec := range resp.CloudProvider.SupportedMachines {
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
