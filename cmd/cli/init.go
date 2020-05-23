package main

import (
	"bytes"
	"fmt"
	"os"
	"text/tabwriter"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/denisbrodbeck/machineid"
	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/core"
	ssh "github.com/protosio/protos/internal/ssh"
	"github.com/protosio/protos/pkg/types"
	"github.com/urfave/cli/v2"
)

var cmdInit *cli.Command = &cli.Command{
	Name:  "init",
	Usage: "Initializes Protos locally and deploys an instance in one of the supported clouds",
	Subcommands: []*cli.Command{
		{
			Name:  "minimal",
			Usage: "Initialize local database and user details",
			Action: func(c *cli.Context) error {
				return protosMinimalInit()
			},
		},
		{
			Name:  "full",
			Usage: "Initialize a protos instance. Created local db, user, adds a cloud provider and a Protos instance.",
			Action: func(c *cli.Context) error {
				return protosFullInit()
			},
		},
	},
}

func protosUserinit() error {

	usrInfo, err := envi.UM.GetAdmin()
	if err == nil {
		return fmt.Errorf("User '%s' already initialized", usrInfo.GetUsername())
	}

	userNameQuestion := []*survey.Question{{
		Name:     "username",
		Prompt:   &survey.Input{Message: "A username to uniquely identify you.\nUSERNAME: "},
		Validate: survey.Required,
	}}
	var username string
	err = survey.Ask(userNameQuestion, &username)
	if err != nil {
		return err
	}

	nameQuestion := []*survey.Question{{
		Name:     "name",
		Prompt:   &survey.Input{Message: "Your name. This field is not mandatory and if left blank, your username will be used instead.\nNAME: "},
		Validate: survey.Required,
	}}
	var name string
	err = survey.Ask(nameQuestion, &name)
	if err != nil {
		return err
	}

	password := ""
	passwordconfirm := " "

	for password != passwordconfirm {
		passwordQuestion := []*survey.Question{{
			Name:     "password",
			Prompt:   &survey.Password{Message: "Password used to authenticate on the Protos instance and apps that you deploy on it.\nPASSWORD: "},
			Validate: survey.Required,
		}}
		err = survey.Ask(passwordQuestion, &password)
		if err != nil {
			return err
		}
		passwordConfirmQuestion := []*survey.Question{{
			Name:     "passwordconfirm",
			Prompt:   &survey.Password{Message: "CONFIRM PASSWORD: "},
			Validate: survey.Required,
		}}
		err = survey.Ask(passwordConfirmQuestion, &passwordconfirm)
		if err != nil {
			return err
		}
		if password != passwordconfirm {
			envi.Log.Error("Passwords don't match")
		}
	}

	domainQuestion := []*survey.Question{{
		Name:     "name",
		Prompt:   &survey.Input{Message: "Fill in a domain name that you would like to use.\nIMPORTANT NOTE: ideally you own the domain or it is available for registration. If not, the domain will only be able to be used internally.\nDOMAIN: "},
		Validate: survey.Required,
	}}
	var domain string
	err = survey.Ask(domainQuestion, &domain)
	if err != nil {
		return err
	}

	host, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("Failed to add user. Could not retrieve hostname: %w", err)
	}
	key, err := envi.SM.GenerateKey()
	if err != nil {
		return fmt.Errorf("Failed to add user. Could not generate key: %w", err)
	}

	machineID, err := machineid.ProtectedID("protos")
	if err != nil {
		return fmt.Errorf("Failed to add user. Error while generating machine id: %w", err)
	}

	devices := []types.UserDevice{{Name: host, PublicKey: key.PublicWG().String(), MachineID: machineID, Network: "10.100.0.1/24"}}

	_, err = envi.UM.CreateUser(username, password, name, domain, true, devices)
	if err != nil {
		return err
	}

	// saving the key to disk
	key.Save()

	return nil
}

func protosMinimalInit() error {
	err := protosUserinit()
	if err != nil {
		return err
	}

	return nil
}

func protosFullInit() error {

	//
	// add user
	//

	err := protosUserinit()
	if err != nil {
		return err
	}

	//
	// add cloud provider
	//

	// get a name to use internally for this specific cloud provider + credentials. This allows for adding multiple accounts of the same cloud
	cloudNameQuestion := []*survey.Question{{
		Name:     "name",
		Prompt:   &survey.Input{Message: "In the following step you will add a cloud provider. Write a name used to identify this cloud provider account internally:"},
		Validate: survey.Required,
	}}
	var cloudName string
	err = survey.Ask(cloudNameQuestion, &cloudName)
	if err != nil {
		return err
	}

	cloudProvider, err := addCloudProvider(cloudName)
	if err != nil {
		return err
	}

	//
	// Protos instance creation steps
	//

	// select one of the supported locations by this particular cloud
	var cloudLocation string
	supportedLocations := cloudProvider.SupportedLocations()
	cloudLocationQuestions := surveySelect(supportedLocations, fmt.Sprintf("Choose one of the following supported locations for '%s':", cloudProvider.TypeStr()))
	err = survey.AskOne(cloudLocationQuestions, &cloudLocation)
	if err != nil {
		return errors.Wrap(err, "Failed to initialize Protos")
	}

	// get a name to use internally for this instance. This name should be reflected accordingly in the cloud provider account
	vmNameQuestion := []*survey.Question{{
		Name:     "name",
		Prompt:   &survey.Input{Message: "Write a name used to identify Protos instance that will be deployed next:"},
		Validate: survey.Required,
	}}
	var vmName string
	err = survey.Ask(vmNameQuestion, &vmName)

	// get latest Protos release
	releases, err := getProtosAvailableReleases()
	if err != nil {
		return errors.Wrap(err, "Failed to initialize Protos")
	}
	latestRelease, err := releases.GetLatest()
	if err != nil {
		return errors.Wrap(err, "Failed to initialize Protos")
	}

	// select one of the supported locations by this particular cloud
	var machineType string
	supportedMachineTypes, err := cloudProvider.SupportedMachines(cloudLocation)
	if err != nil {
		return errors.Wrap(err, "Failed to initialize Protos")
	}
	supportedMachineTypeIDs := []string{}
	for id := range supportedMachineTypes {
		supportedMachineTypeIDs = append(supportedMachineTypeIDs, id)
	}
	machineTypesStr := createMachineTypesString(supportedMachineTypes)
	machineTypeQuestion := surveySelect(supportedMachineTypeIDs, fmt.Sprintf("Choose one of the following supported machine types for '%s'.\n%s", cloudProvider.TypeStr(), machineTypesStr))
	err = survey.AskOne(machineTypeQuestion, &machineType)
	if err != nil {
		return errors.Wrap(err, "Failed to initialize Protos")
	}

	// deploy the vm
	instanceInfo, err := deployInstance(vmName, cloudName, cloudLocation, latestRelease, machineType)
	if err != nil {
		return errors.Wrap(err, "Failed to initialize Protos")
	}

	//
	// Perform setup via SSH tunnel
	//

	key, err := envi.SM.NewKeyFromSeed(instanceInfo.KeySeed)
	if err != nil {
		return errors.Wrap(err, "Failed to initialize Protos")
	}

	// test SSH and create SSH tunnel used for initialisation
	tempClient, err := ssh.NewConnection(instanceInfo.PublicIP, "root", key.SSHAuth(), 10)
	if err != nil {
		return errors.Wrap(err, "Failed to connect to Protos instance via SSH")
	}
	tempClient.Close()
	log.Info("Instance is ready and accepting SSH connections. Perform instance setup using the web based dashboard")

	// create tunnel to reach the instance dashboard
	tunnelInstance(instanceInfo.Name)
	log.Infof("Protos instance '%s' - '%s' deployed successfully", vmName, instanceInfo.PublicIP)

	return nil
}

func getUserDetailsQuestions(ud *userDetails) []*survey.Question {
	return []*survey.Question{
		{
			Name:      "username",
			Prompt:    &survey.Input{Message: "Username:"},
			Validate:  survey.Required,
			Transform: survey.ToLower,
		},
		{
			Name:      "name",
			Prompt:    &survey.Input{Message: "Name:"},
			Validate:  survey.Required,
			Transform: survey.Title,
		},
		{
			Name:     "password",
			Prompt:   &survey.Password{Message: "Password:"},
			Validate: survey.Required,
		},
		{
			Name:   "passwordconfirm",
			Prompt: &survey.Password{Message: "Confirm password:"},
			Validate: func(val interface{}) error {
				if str, ok := val.(string); ok && str != ud.Password {
					return fmt.Errorf("passwords don't match")
				}
				return nil
			},
		},
		{
			Name:     "domain",
			Prompt:   &survey.Input{Message: "Domain name (registered with one of the supported domain providers)"},
			Validate: survey.Required,
		},
	}
}

func surveySelect(options []string, message string) *survey.Select {
	return &survey.Select{
		Message: message,
		Options: options,
	}
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

func createMachineTypesString(machineTypes map[string]core.MachineSpec) string {
	var machineTypesStr bytes.Buffer
	w := new(tabwriter.Writer)
	w.Init(&machineTypesStr, 8, 8, 0, ' ', 0)
	for instanceID, instanceSpec := range machineTypes {
		fmt.Fprintf(w, "    %s\t -  Nr of CPUs: %d,\t Memory: %d MiB,\t Storage: %d GB\t\n", instanceID, instanceSpec.Cores, instanceSpec.Memory, instanceSpec.DefaultStorage)
	}
	w.Flush()
	return machineTypesStr.String()
}
