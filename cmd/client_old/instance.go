package main

// import (
// 	"encoding/base64"
// 	"fmt"
// 	"os"
// 	"text/tabwriter"

// 	"github.com/pkg/errors"
// 	"github.com/protosio/protos/internal/cloud"
// 	"github.com/protosio/protos/internal/release"
// 	"github.com/urfave/cli/v2"
// )

// const (
// 	instanceDS = "instance"
// )

// var machineType string
// var devImg string

// var cmdInstance *cli.Command = &cli.Command{
// 	Name:  "instance",
// 	Usage: "Manage Protos instances",
// 	Subcommands: []*cli.Command{
// 		{
// 			Name:  "ls",
// 			Usage: "List instances",
// 			Action: func(c *cli.Context) error {
// 				return listInstances()
// 			},
// 		},
// 		{
// 			Name:      "info",
// 			ArgsUsage: "<name>",
// 			Usage:     "Display information about an instance",
// 			Action: func(c *cli.Context) error {
// 				name := c.Args().Get(0)
// 				if name == "" {
// 					cli.ShowSubcommandHelp(c)
// 					os.Exit(1)
// 				}
// 				return infoInstance(name)
// 			},
// 		},
// 		{
// 			Name:      "deploy",
// 			ArgsUsage: "<name>",
// 			Usage:     "Deploy a new Protos instance",
// 			Flags: []cli.Flag{
// 				&cli.StringFlag{
// 					Name:        "cloud",
// 					Usage:       "Specify which `CLOUD` to deploy the instance on",
// 					Required:    true,
// 					Destination: &cloudName,
// 				},
// 				&cli.StringFlag{
// 					Name:        "location",
// 					Usage:       "Specify one of the supported `LOCATION`s to deploy the instance in (cloud specific)",
// 					Required:    true,
// 					Destination: &cloudLocation,
// 				},
// 				&cli.StringFlag{
// 					Name:        "version",
// 					Usage:       "Specify Protos `VERSION` to deploy",
// 					Required:    false,
// 					Destination: &protosVersion,
// 				},
// 				&cli.StringFlag{
// 					Name:        "devimg",
// 					Usage:       "Use a dev image uploaded to your cloud accoun",
// 					Required:    false,
// 					Destination: &devImg,
// 				},
// 				&cli.StringFlag{
// 					Name:        "type",
// 					Usage:       "Specify cloud machine type `TYPE` to deploy. Get it from 'cloud info' subcommand",
// 					Required:    true,
// 					Destination: &machineType,
// 				},
// 			},
// 			Action: func(c *cli.Context) error {
// 				name := c.Args().Get(0)
// 				if name == "" {
// 					cli.ShowSubcommandHelp(c)
// 					os.Exit(1)
// 				}
// 				releases, err := getProtosAvailableReleases()
// 				if err != nil {
// 					return err
// 				}
// 				rls := release.Release{}
// 				if devImg != "" {
// 					rls.Version = devImg
// 				} else if protosVersion != "" {
// 					rls, err = releases.GetVersion(protosVersion)
// 					if err != nil {
// 						return err
// 					}
// 				} else {
// 					rls, err = releases.GetLatest()
// 					if err != nil {
// 						return err
// 					}
// 				}

// 				_, err = deployInstance(name, cloudName, cloudLocation, rls, machineType)
// 				return err
// 			},
// 		},
// 		{
// 			Name:      "delete",
// 			ArgsUsage: "<name>",
// 			Usage:     "Delete instance",
// 			Flags: []cli.Flag{
// 				&cli.BoolFlag{
// 					Name:  "local",
// 					Usage: "Deletes the instance from the db and ignores any cloud resources",
// 				},
// 			},
// 			Action: func(c *cli.Context) error {
// 				name := c.Args().Get(0)
// 				if name == "" {
// 					cli.ShowSubcommandHelp(c)
// 					os.Exit(1)
// 				}
// 				return deleteInstance(name, c.Bool("local"))
// 			},
// 		},
// 		{
// 			Name:      "start",
// 			ArgsUsage: "<name>",
// 			Usage:     "Power on instance",
// 			Action: func(c *cli.Context) error {
// 				name := c.Args().Get(0)
// 				if name == "" {
// 					cli.ShowSubcommandHelp(c)
// 					os.Exit(1)
// 				}
// 				return startInstance(name)
// 			},
// 		},
// 		{
// 			Name:      "stop",
// 			ArgsUsage: "<name>",
// 			Usage:     "Power off instance",
// 			Action: func(c *cli.Context) error {
// 				name := c.Args().Get(0)
// 				if name == "" {
// 					cli.ShowSubcommandHelp(c)
// 					os.Exit(1)
// 				}
// 				return stopInstance(name)
// 			},
// 		},
// 		{
// 			Name:      "tunnel",
// 			ArgsUsage: "<name>",
// 			Usage:     "Creates SSH encrypted tunnel to instance dashboard",
// 			Action: func(c *cli.Context) error {
// 				name := c.Args().Get(0)
// 				if name == "" {
// 					cli.ShowSubcommandHelp(c)
// 					os.Exit(1)
// 				}
// 				return tunnelInstance(name)
// 			},
// 		},
// 		{
// 			Name:      "key",
// 			ArgsUsage: "<name>",
// 			Usage:     "Prints to stdout the SSH key associated with the instance",
// 			Action: func(c *cli.Context) error {
// 				name := c.Args().Get(0)
// 				if name == "" {
// 					cli.ShowSubcommandHelp(c)
// 					os.Exit(1)
// 				}
// 				return keyInstance(name)
// 			},
// 		},
// 		{
// 			Name:      "devinit",
// 			ArgsUsage: "<instance name> <key> <ip>",
// 			Usage:     "Initiate a development instance",
// 			Action: func(c *cli.Context) error {
// 				name := c.Args().Get(0)
// 				if name == "" {
// 					cli.ShowSubcommandHelp(c)
// 					os.Exit(1)
// 				}

// 				key := c.Args().Get(1)
// 				if key == "" {
// 					cli.ShowSubcommandHelp(c)
// 					os.Exit(1)
// 				}

// 				ip := c.Args().Get(2)
// 				if ip == "" {
// 					cli.ShowSubcommandHelp(c)
// 					os.Exit(1)
// 				}
// 				return devInit(name, key, ip)
// 			},
// 		},
// 		{
// 			Name:      "logs",
// 			ArgsUsage: "<instance name>",
// 			Usage:     "Pulls and displays Protos logs for instance",
// 			Flags: []cli.Flag{
// 				&cli.BoolFlag{
// 					Name:     "f",
// 					Usage:    "Follow logs",
// 					Required: false,
// 					Value:    false,
// 				},
// 			},
// 			Action: func(c *cli.Context) error {
// 				name := c.Args().Get(0)
// 				if name == "" {
// 					cli.ShowSubcommandHelp(c)
// 					os.Exit(1)
// 				}

// 				follow := c.Bool("f")

// 				return getLogs(name, follow)
// 			},
// 		},
// 	},
// }

// //
// // Instance methods
// //

// func listInstances() error {
// 	instances, err := envi.CLM.GetInstances()
// 	if err != nil {
// 		return err
// 	}

// 	w := new(tabwriter.Writer)
// 	w.Init(os.Stdout, 0, 0, 2, ' ', 0)

// 	defer w.Flush()

// 	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t%s\t", "Name", "Public IP", "Net", "Cloud", "VM ID", "Location", "Status")
// 	fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t%s\t%s\t%s\t", "----", "---------", "---", "-----", "-----", "--------", "------")
// 	for _, instance := range instances {
// 		fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t%s\t%s\t%s\t", instance.Name, instance.PublicIP, instance.Network, instance.CloudName, instance.VMID, instance.Location, "n/a")
// 	}
// 	fmt.Fprint(w, "\n")
// 	return nil
// }

// func infoInstance(instanceName string) error {
// 	instance, err := envi.CLM.GetInstance(instanceName)
// 	if err != nil {
// 		return fmt.Errorf("Could not retrieve instance '%s': %w", instanceName, err)
// 	}

// 	encodedPublicKey := base64.StdEncoding.EncodeToString(instance.PublicKey)
// 	fmt.Printf("Name: %s\n", instance.Name)
// 	fmt.Printf("VM ID: %s\n", instance.VMID)
// 	fmt.Printf("Public Key (wireguard): %s\n", encodedPublicKey)
// 	fmt.Printf("Public IP: %s\n", instance.PublicIP)
// 	fmt.Printf("Internal IP: %s\n", instance.InternalIP)
// 	fmt.Printf("Network: %s\n", instance.Network)
// 	fmt.Printf("Cloud type: %s\n", instance.CloudType)
// 	fmt.Printf("Cloud name: %s\n", instance.CloudName)
// 	fmt.Printf("Location: %s\n", instance.Location)
// 	fmt.Printf("Protosd version: %s\n", instance.ProtosVersion)
// 	return nil
// }

// func deployInstance(instanceName string, cloudName string, cloudLocation string, release release.Release, machineType string) (cloud.InstanceInfo, error) {
// 	return envi.CLM.DeployInstance(instanceName, cloudName, cloudLocation, release, machineType)
// }

// func deleteInstance(name string, localOnly bool) error {
// 	return envi.CLM.DeleteInstance(name)
// }

// func startInstance(name string) error {
// 	return envi.CLM.StartInstance(name)
// }

// func stopInstance(name string) error {
// 	return envi.CLM.StopInstance(name)
// }

// func tunnelInstance(name string) error {
// 	return envi.CLM.TunnelInstance(name)
// }

// func keyInstance(name string) error {
// 	instanceInfo, err := envi.CLM.GetInstance(name)
// 	if err != nil {
// 		return errors.Wrapf(err, "Could not retrieve instance '%s'", name)
// 	}
// 	if len(instanceInfo.SSHKeySeed) == 0 {
// 		return errors.Errorf("Instance '%s' is missing its SSH key", name)
// 	}
// 	key, err := envi.SM.NewKeyFromSeed(instanceInfo.SSHKeySeed)
// 	if err != nil {
// 		return errors.Wrapf(err, "Instance '%s' has an invalid SSH key", name)
// 	}
// 	fmt.Print(key.EncodePrivateKeytoPEM())
// 	return nil
// }

// func devInit(instanceName string, keyFile string, ipString string) error {

// 	hostname, err := os.Hostname()
// 	if err != nil {
// 		return err
// 	}

// 	err = envi.CLM.InitDevInstance(instanceName, hostname, hostname, keyFile, ipString)
// 	if err != nil {
// 		return fmt.Errorf("Could not init dev instance '%s': %w", instanceName, err)
// 	}

// 	return nil
// }

// func getLogs(instanceName string, follow bool) error {
// 	logs, err := envi.CLM.LogsRemoteInstance(instanceName)
// 	if err != nil {
// 		return fmt.Errorf("Failed to retrieve logs for instance '%s': %w", instanceName, err)
// 	}
// 	fmt.Println(logs)
// 	return nil
// }
