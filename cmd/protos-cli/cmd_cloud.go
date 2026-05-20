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
	Name:    "provisioner",
	Aliases: []string{"cloud", "prov"},
	Usage:   "Manage provisioners",
	Subcommands: []*cli.Command{
		{
			Name:  "ls",
			Usage: "List existing provisioners",
			Action: func(c *cli.Context) error {
				return listProvisioners()
			},
		},
		{
			Name:      "add",
			ArgsUsage: "<name>",
			Usage:     "Add a new provisioner",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "type",
					Usage: "Provisioner `TYPE` to add without prompting",
				},
				&cli.StringSliceFlag{
					Name:  "credential",
					Usage: "Provisioner credential as `KEY=VALUE`; may be repeated",
				},
			},
			Action: func(c *cli.Context) error {
				name := c.Args().Get(0)
				if name == "" {
					return showSubcommandHelp(c)
				}
				err := addProvisioner(name, c.String("type"), c.StringSlice("credential"))
				return err
			},
		},
		{
			Name:      "delete",
			ArgsUsage: "<name>",
			Usage:     "Delete an existing provisioner",
			Action: func(c *cli.Context) error {
				name := c.Args().Get(0)
				if name == "" {
					return showSubcommandHelp(c)
				}
				return deleteProvisioner(name)
			},
		},
		{
			Name:      "info",
			ArgsUsage: "<name>",
			Usage:     "Prints info about provisioner account and checks if the API is reachable",
			Action: func(c *cli.Context) error {
				name := c.Args().Get(0)
				if name == "" {
					return showSubcommandHelp(c)
				}
				return infoProvisioner(name)
			},
		},
	},
}

func getProvisionerCredentialsQuestions(provisionerName string, fields []string) []*survey.Question {
	qs := []*survey.Question{}
	for _, field := range fields {
		qs = append(qs, &survey.Question{
			Name:     field,
			Prompt:   &survey.Input{Message: provisionerName + " " + field + ":"},
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
//  Provisioner methods
//

func listProvisioners() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetProvisioners(ctx, &pbApic.GetProvisionersRequest{})
	if err != nil {
		return fmt.Errorf("failed to retrieve provisioners: %w", err)
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 16, 16, 0, '\t', 0)

	defer w.Flush()

	fmt.Fprintf(w, " %s\t%s\t", "Name", "Type")
	fmt.Fprintf(w, "\n %s\t%s\t", "----", "----")
	for _, provisioner := range resp.Provisioners {
		fmt.Fprintf(w, "\n %s\t%s\t", provisioner.Name, provisioner.Type.Name)
	}
	fmt.Fprint(w, "\n")
	return nil
}

func addProvisioner(provisionerName string, requestedProvisionerType string, credentialPairs []string) error {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetSupportedProvisioners(ctx, &pbApic.GetSupportedProvisionersRequest{})
	if err != nil {
		return fmt.Errorf("failed to retrieve supported provisioners: %w", err)
	}

	supportedProvisionerTypes := make([]string, len(resp.ProvisionerTypes))
	supportedProvisionerTypeCredentials := make(map[string][]string, len(resp.ProvisionerTypes))
	i := 0
	for _, provisionerType := range resp.ProvisionerTypes {
		supportedProvisionerTypes[i] = provisionerType.Name
		supportedProvisionerTypeCredentials[provisionerType.Name] = provisionerType.AuthenticationFields
		i++
	}

	var provisionerType string
	if requestedProvisionerType != "" {
		if _, found := supportedProvisionerTypeCredentials[requestedProvisionerType]; !found {
			return fmt.Errorf("provisioner type '%s' is not supported", requestedProvisionerType)
		}
		provisionerType = requestedProvisionerType
	} else {
		provisionerSelect := surveySelect(supportedProvisionerTypes, "Choose one of the following supported provisioners:")
		err = survey.AskOne(provisionerSelect, &provisionerType)
		if err != nil {
			return err
		}
	}

	provisionerCredentials := map[string]interface{}{}
	for _, pair := range credentialPairs {
		key, value, found := strings.Cut(pair, "=")
		if !found || strings.TrimSpace(key) == "" {
			return fmt.Errorf("credential must be in KEY=VALUE form")
		}
		provisionerCredentials[key] = value
	}
	if requestedProvisionerType == "" || len(supportedProvisionerTypeCredentials[provisionerType]) > len(provisionerCredentials) {
		credentialsQuestions := getProvisionerCredentialsQuestions(provisionerType, supportedProvisionerTypeCredentials[provisionerType])
		err = survey.Ask(credentialsQuestions, &provisionerCredentials)
		if err != nil {
			return err
		}
	}

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = client.AddProvisioner(ctx, &pbApic.AddProvisionerRequest{
		Name:        provisionerName,
		Type:        provisionerType,
		Credentials: transformCredentials(provisionerCredentials),
	})
	if err != nil {
		return fmt.Errorf("failed to add provisioner: %w", err)
	}

	return nil
}

func deleteProvisioner(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := client.RemoveProvisioner(ctx, &pbApic.RemoveProvisionerRequest{Name: name})
	if err != nil {
		return fmt.Errorf("failed to remove provisioner '%s': %w", name, err)
	}
	return nil
}

func infoProvisioner(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetProvisioner(ctx, &pbApic.GetProvisionerRequest{Name: name})
	if err != nil {
		return fmt.Errorf("failed to retrieve provisioner '%s': %w", name, err)
	}

	fmt.Printf("Name: %s\n", name)
	fmt.Printf("Type: %s\n", resp.Provisioner.Type.Name)
	fmt.Printf("Supported locations: %s\n", strings.Join(resp.Provisioner.SupportedLocations, " | "))
	fmt.Printf("Supported machine types: \n")
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 8, 8, 0, ' ', 0)
	for instanceID, instanceSpec := range resp.Provisioner.SupportedMachines {
		fmt.Fprintf(w, "    %s\t -  Nr of CPUs: %d,\t Memory: %d MiB,\t Storage: %d GB\t\n", instanceID, instanceSpec.Cores, instanceSpec.Memory, instanceSpec.DefaultStorage)
	}
	w.Flush()
	return nil
}
