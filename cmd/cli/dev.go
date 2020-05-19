package main

import (
	"fmt"
	"net"
	"os"

	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/cloud"
	"github.com/protosio/protos/internal/ssh"
	"github.com/protosio/protos/internal/user"
	pclient "github.com/protosio/protos/pkg/client"
	"github.com/protosio/protos/pkg/types"
	"github.com/urfave/cli/v2"
)

var cmdDev *cli.Command = &cli.Command{
	Name:  "dev",
	Usage: "Manage development instances",
	Subcommands: []*cli.Command{
		{
			Name:      "init",
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
	},
}

func devInit(instanceName string, keyFile string, ipString string) error {
	usr, err := user.Get(envi.DB)
	if err != nil {
		return err
	}

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	instanceInfo := cloud.InstanceInfo{
		VMID:          instanceName,
		PublicIP:      ipString,
		Name:          instanceName,
		CloudType:     cloud.Hyperkit,
		CloudName:     hostname,
		Location:      hostname,
		ProtosVersion: "dev",
	}

	ip := net.ParseIP(ipString)
	if ip == nil {
		return fmt.Errorf("String '%s' is not a valid IP address", ipString)
	}

	auth, err := ssh.NewAuthFromKeyFile(keyFile)
	if err != nil {
		return err
	}

	// allocate network for dev instance
	var instances []cloud.InstanceInfo
	err = envi.DB.GetSet(instanceDS, &instances)
	if err != nil {
		return fmt.Errorf("Failed to allocate network for instance '%s': %w", "dev", err)
	}
	developmentNetwork, err := cloud.AllocateNetwork(instances)
	if err != nil {
		return fmt.Errorf("Failed to allocate network for instance '%s': %w", "dev", err)
	}

	log.Infof("Creating SSH tunnel to dev instance IP '%s'", ipString)
	tunnel := ssh.NewTunnel(ip.String()+":22", "root", auth, "localhost:8080")
	localPort, err := tunnel.Start()
	if err != nil {
		return errors.Wrap(err, "Error while creating the SSH tunnel")
	}

	// wait for the API to be up
	err = cloud.WaitForHTTP(fmt.Sprintf("http://127.0.0.1:%d/ui/", localPort), 20)
	if err != nil {
		return errors.Wrap(err, "Failed to deploy instance")
	}
	log.Infof("Tunnel to '%s' ready", ipString)

	user, err := user.Get(envi.DB)
	if err != nil {
		return err
	}

	// do the initialization
	log.Infof("Initializing instance at '%s'", ipString)
	protos := pclient.NewInitClient(fmt.Sprintf("127.0.0.1:%d", localPort), user.Username, user.Password)
	key, err := ssh.NewKeyFromSeed(usr.Device.KeySeed)
	if err != nil {
		panic(err)
	}

	usrDev := types.UserDevice{
		Name:      usr.Device.Name,
		PublicKey: key.PublicWG().String(),
		Network:   usr.Device.Network,
	}

	// Doing the instance initialization which returns the internal wireguard IP and the public key created using the wireguard library.
	instanceIP, instancePublicKey, err := protos.InitInstance(user.Name, developmentNetwork.String(), user.Domain, []types.UserDevice{usrDev})
	if err != nil {
		return errors.Wrap(err, "Error while doing the instance initialization")
	}
	instanceInfo.InternalIP = instanceIP.String()
	instanceInfo.PublicKey = instancePublicKey
	instanceInfo.Network = developmentNetwork.String()

	err = envi.DB.InsertInSet(instanceDS, instanceInfo)
	if err != nil {
		return errors.Wrapf(err, "Failed to save dev instance '%s'", instanceName)
	}

	// close the SSH tunnel
	err = tunnel.Close()
	if err != nil {
		return errors.Wrap(err, "Error while terminating the SSH tunnel")
	}
	log.Infof("Instance at '%s' is ready", ipString)

	return nil
}
