package main

import (
	"context"
	"time"

	survey "github.com/AlecAivazis/survey/v2"
	apic "github.com/protosio/protos/apic/proto"
	"github.com/urfave/cli/v2"
)

var cmdInit *cli.Command = &cli.Command{
	Name:  "init",
	Usage: "Performs Protos user initialization",
	Action: func(c *cli.Context) error {
		return protosUserinit()
	},
}

func protosUserinit() error {

	userNameQuestion := []*survey.Question{{
		Name:     "username",
		Prompt:   &survey.Input{Message: "A username to uniquely identify you.\nUSERNAME: "},
		Validate: survey.Required,
	}}
	var username string
	err := survey.Ask(userNameQuestion, &username)
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
			log.Error("Passwords don't match")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = client.Init(ctx, &apic.InitRequest{Username: username, Password: password, Name: name})
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
