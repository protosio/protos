package main

import (
	"context"
	"time"

	survey "github.com/AlecAivazis/survey/v2"
	apic "github.com/protosio/protos/apic/proto"
	"github.com/urfave/cli/v2"
)

var username string
var name string
var organization string

var cmdInit *cli.Command = &cli.Command{
	Name:  "init",
	Usage: "Performs Protos user initialization",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "username",
			Required:    true,
			Destination: &username,
		},
		&cli.StringFlag{
			Name:        "name",
			Required:    false,
			Destination: &name,
			DefaultText: username,
		},
		&cli.StringFlag{
			Name:        "organization",
			Required:    false,
			Destination: &organization,
			DefaultText: "home",
		},
	},
	Action: func(c *cli.Context) error {
		return protosUserinit()
	},
}

func protosUserinit() error {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := client.Init(ctx, &apic.InitRequest{Username: username, Name: name, Organisation: organization})
	if err != nil {
		return err
	}
	log.Info("Initialization complete")

	return nil
}

func surveySelect(options []string, message string) *survey.Select {
	return &survey.Select{
		Message: message,
		Options: options,
	}
}
