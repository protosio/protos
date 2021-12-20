package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	apic "github.com/protosio/protos/apic/proto"
	"github.com/urfave/cli/v2"
)

var cloudName string
var cloudLocation string
var protosVersion string
var devImg string
var machineType string

var cmdInstance *cli.Command = &cli.Command{
	Name:  "instance",
	Usage: "Manage Protos instances",
	Subcommands: []*cli.Command{
		{
			Name:  "ls",
			Usage: "List instances",
			Action: func(c *cli.Context) error {
				return listInstances()
			},
		},
		{
			Name:      "info",
			ArgsUsage: "<name>",
			Usage:     "Display information about an instance",
			Action: func(c *cli.Context) error {
				name := c.Args().Get(0)
				if name == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}
				return infoInstance(name)
			},
		},
		{
			Name:      "deploy",
			ArgsUsage: "<name>",
			Usage:     "Deploy a new Protos instance",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:        "cloud",
					Usage:       "Specify which `CLOUD` to deploy the instance on",
					Required:    true,
					Destination: &cloudName,
				},
				&cli.StringFlag{
					Name:        "location",
					Usage:       "Specify one of the supported `LOCATION`s to deploy the instance in (cloud specific)",
					Required:    true,
					Destination: &cloudLocation,
				},
				&cli.StringFlag{
					Name:        "version",
					Usage:       "Specify Protosd `VERSION` to deploy",
					Required:    false,
					Destination: &protosVersion,
				},
				&cli.StringFlag{
					Name:        "devimg",
					Usage:       "Use a dev image uploaded to your cloud accoun",
					Required:    false,
					Destination: &devImg,
				},
				&cli.StringFlag{
					Name:        "type",
					Usage:       "Specify cloud machine type `TYPE` to deploy. Get it from 'cloud info' subcommand",
					Required:    true,
					Destination: &machineType,
				},
			},
			Action: func(c *cli.Context) error {
				name := c.Args().Get(0)
				if name == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}

				return deployInstance(name, cloudName, cloudLocation, protosVersion, machineType, devImg)
			},
		},
		{
			Name:      "delete",
			ArgsUsage: "<name>",
			Usage:     "Delete instance",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "local",
					Usage: "Deletes the instance from the db and ignores any cloud resources",
				},
			},
			Action: func(c *cli.Context) error {
				name := c.Args().Get(0)
				if name == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}
				local := c.Bool("local")
				return deleteInstance(name, local)
			},
		},
		{
			Name:      "start",
			ArgsUsage: "<name>",
			Usage:     "Power on instance",
			Action: func(c *cli.Context) error {
				name := c.Args().Get(0)
				if name == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}
				return startInstance(name)
			},
		},
		{
			Name:      "stop",
			ArgsUsage: "<name>",
			Usage:     "Power off instance",
			Action: func(c *cli.Context) error {
				name := c.Args().Get(0)
				if name == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}
				return stopInstance(name)
			},
		},
		{
			Name:      "key",
			ArgsUsage: "<name>",
			Usage:     "Prints to stdout the SSH key associated with the instance",
			Action: func(c *cli.Context) error {
				name := c.Args().Get(0)
				if name == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}
				return getInstanceKey(name)
			},
		},
		{
			Name:      "devinit",
			ArgsUsage: "<instance name> <key> <ip>",
			Usage:     "Initiate a development instance",
			Action: func(c *cli.Context) error {
				name := c.Args().Get(0)
				if name == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}

				key := c.Args().Get(1)
				if key == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}

				ip := c.Args().Get(2)
				if ip == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}
				return devInit(name, key, ip)
			},
		},
		{
			Name:      "logs",
			ArgsUsage: "<instance name>",
			Usage:     "Pulls and displays Protos logs for instance",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:     "f",
					Usage:    "Follow logs",
					Required: false,
					Value:    false,
				},
			},
			Action: func(c *cli.Context) error {
				name := c.Args().Get(0)
				if name == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}

				follow := c.Bool("f")

				return getInstanceLogs(name, follow)
			},
		},
	},
}

//
// Instance methods
//

func listInstances() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetInstances(ctx, &apic.GetInstancesRequest{})
	if err != nil {
		return err
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)

	defer w.Flush()

	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t%s\t", "Name", "Public IP", "Net", "Cloud", "VM ID", "Location", "Status")
	fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t%s\t%s\t%s\t", "----", "---------", "---", "-----", "-----", "--------", "------")
	for _, instance := range resp.Instances {
		fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t%s\t%s\t%s\t", instance.Name, instance.PublicIp, instance.Network, instance.CloudName, instance.VmId, instance.Location, "n/a")
	}
	fmt.Fprint(w, "\n")
	return nil
}

func infoInstance(instanceName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetInstance(ctx, &apic.GetInstanceRequest{Name: instanceName})
	if err != nil {
		return fmt.Errorf("could not retrieve instance '%s': %w", instanceName, err)
	}
	instance := resp.Instance

	fmt.Printf("Name: %s\n", instance.Name)
	fmt.Printf("VM ID: %s\n", instance.VmId)
	fmt.Printf("Public Key (wireguard): %s\n", instance.PublicKey)
	fmt.Printf("Public IP: %s\n", instance.PublicIp)
	fmt.Printf("Internal IP: %s\n", instance.InternalIp)
	fmt.Printf("Network: %s\n", instance.Network)
	fmt.Printf("Cloud type: %s\n", instance.CloudType)
	fmt.Printf("Cloud name: %s\n", instance.CloudName)
	fmt.Printf("Location: %s\n", instance.Location)
	fmt.Printf("Protosd version: %s\n", instance.ProtosVersion)
	fmt.Printf("Status: %s\n", instance.Status)
	return nil
}

func deployInstance(instanceName string, cloudName string, cloudLocation string, protosVersion string, machineType string, devImage string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Second)
	defer cancel()
	resp, err := client.DeployInstance(ctx, &apic.DeployInstanceRequest{Name: instanceName, CloudName: cloudName, CloudLocation: cloudLocation, ProtosVersion: protosVersion, MachineType: machineType, DevImg: devImage})
	if err != nil {
		return fmt.Errorf("could not deploy instance '%s': %w", instanceName, err)
	}
	instance := resp.Instance
	fmt.Printf("Name: %s\n", instance.Name)
	fmt.Printf("VM ID: %s\n", instance.VmId)
	fmt.Printf("Public Key (wireguard): %s\n", instance.PublicKey)
	fmt.Printf("Public IP: %s\n", instance.PublicIp)
	fmt.Printf("Internal IP: %s\n", instance.InternalIp)
	fmt.Printf("Network: %s\n", instance.Network)
	fmt.Printf("Cloud type: %s\n", instance.CloudType)
	fmt.Printf("Cloud name: %s\n", instance.CloudName)
	fmt.Printf("Location: %s\n", instance.Location)
	fmt.Printf("Protosd version: %s\n", instance.ProtosVersion)
	fmt.Printf("Status: %s\n", instance.Status)
	return nil
}

func deleteInstance(instanceName string, localOnly bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Second)
	defer cancel()
	_, err := client.RemoveInstance(ctx, &apic.RemoveInstanceRequest{Name: instanceName, LocalOnly: localOnly})
	if err != nil {
		return fmt.Errorf("could not remove instance '%s': %w", instanceName, err)
	}
	return nil
}

func startInstance(instanceName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()
	_, err := client.StartInstance(ctx, &apic.StartInstanceRequest{Name: instanceName})
	if err != nil {
		return fmt.Errorf("could not start instance '%s': %w", instanceName, err)
	}
	return nil
}

func stopInstance(instanceName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()
	_, err := client.StopInstance(ctx, &apic.StopInstanceRequest{Name: instanceName})
	if err != nil {
		return fmt.Errorf("could not stop instance '%s': %w", instanceName, err)
	}
	return nil
}

func getInstanceKey(instanceName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetInstanceKey(ctx, &apic.GetInstanceKeyRequest{Name: instanceName})
	if err != nil {
		return fmt.Errorf("could not get instance '%s' key: %w", instanceName, err)
	}
	fmt.Print(resp.Key)
	return nil
}

func getInstanceLogs(instanceName string, follow bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetInstanceLogs(ctx, &apic.GetInstanceLogsRequest{Name: instanceName})
	if err != nil {
		return fmt.Errorf("could not get instance '%s' logs: %w", instanceName, err)
	}
	fmt.Print(resp.Logs)
	return nil
}

func devInit(instanceName string, keyFile string, ipString string) error {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := client.InitDevInstance(ctx, &apic.InitDevInstanceRequest{Name: instanceName, KeyFile: keyFile, Ip: ipString})
	if err != nil {
		return fmt.Errorf("could not get instance '%s' key: %w", instanceName, err)
	}

	return nil
}
