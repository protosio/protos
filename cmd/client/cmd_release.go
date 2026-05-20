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

var cmdRelease *cli.Command = &cli.Command{
	Name:  "release",
	Usage: "Manage Protos releases",
	Subcommands: []*cli.Command{
		{
			Name:  "available",
			Usage: "Show the available Protosd releases",
			Action: func(c *cli.Context) error {
				return listProtosAvailableReleases()
			},
		},
		{
			Name:      "ls",
			ArgsUsage: "<provisioner name>",
			Usage:     "Retrieve and list the Protosd images available in a specific provisioner",
			Action: func(c *cli.Context) error {
				provisionerName := c.Args().Get(0)
				if provisionerName == "" {
					return showSubcommandHelp(c)
				}

				err := listProtosProvisionerImages(provisionerName)
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			Name:      "upload",
			ArgsUsage: "<image path> <image name>",
			Usage:     "Upload a locally built image to a provisioner. Used for development",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:        "provisioner",
					Aliases:     []string{"cloud"},
					Usage:       "Specify which `PROVISIONER` to upload the image to",
					Required:    true,
					Destination: &cloudName,
				},
				&cli.StringFlag{
					Name:        "location",
					Usage:       "Specify one of the supported `LOCATION`s to upload the image. Not required for all provisioners",
					Required:    false,
					Destination: &cloudLocation,
				},
				&cli.IntFlag{
					Name:     "timeout",
					Usage:    "Upload timeout in minutes",
					Required: false,
					Value:    30,
				},
			},
			Action: func(c *cli.Context) error {
				imagePath := c.Args().Get(0)
				if imagePath == "" {
					return showSubcommandHelp(c)
				}

				imageName := c.Args().Get(1)
				if imageName == "" {
					return showSubcommandHelp(c)
				}

				timeout := c.Int("timeout")

				return uploadLocalImageToProvisioner(imagePath, imageName, cloudName, cloudLocation, int32(timeout))
			},
		},
		{
			Name:      "delete",
			ArgsUsage: "<image name>",
			Usage:     "Delete an image from a provisioner",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:        "provisioner",
					Aliases:     []string{"cloud"},
					Usage:       "Specify which `PROVISIONER` to delete the image from",
					Required:    true,
					Destination: &cloudName,
				},
				&cli.StringFlag{
					Name:        "location",
					Usage:       "Specify one of the supported `LOCATION`s for the image. Not required for all provisioners",
					Required:    false,
					Destination: &cloudLocation,
				},
			},
			Action: func(c *cli.Context) error {
				imageName := c.Args().Get(0)
				if imageName == "" {
					return showSubcommandHelp(c)
				}

				return deleteImageFromProvisioner(imageName, cloudName, cloudLocation)
			},
		},
	},
}

//
// Releases methods
//

func listProtosAvailableReleases() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetProtosdReleases(ctx, &apic.GetProtosdReleasesRequest{})
	if err != nil {
		return err
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)

	defer w.Flush()

	fmt.Fprintf(w, " %s\t%s\t%s\t", "Version", "Date", "Description")
	fmt.Fprintf(w, "\n %s\t%s\t%s\t", "-------", "----", "-----------")
	for _, release := range resp.Releases {
		fmt.Fprintf(w, "\n %s\t%s\t%s\t", release.Version, time.Unix(release.ReleaseDate, 0).Format("Jan 2, 2006"), release.Description)
	}
	fmt.Fprint(w, "\n")
	return nil
}

func listProtosProvisionerImages(provisionerName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetProvisionerImages(ctx, &apic.GetProvisionerImagesRequest{Name: provisionerName})
	if err != nil {
		return err
	}
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)

	defer w.Flush()

	fmt.Fprintf(w, " %s\t%s\t%s\t", "Version", "ID", "Location")
	fmt.Fprintf(w, "\n %s\t%s\t%s\t", "-------", "--", "--------")
	for _, img := range resp.Images {
		fmt.Fprintf(w, "\n %s\t%s\t%s\t", img.Name, img.Id, img.Location)
	}
	fmt.Fprint(w, "\n")

	return nil
}

func uploadLocalImageToProvisioner(imagePath string, imageName string, provisionerName string, location string, timeout int32) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout+1)*time.Minute)
	defer cancel()
	_, err := client.UploadProvisionerImage(ctx, &apic.UploadProvisionerImageRequest{ImagePath: imagePath, ImageName: imageName, ProvisionerName: provisionerName, Location: location, Timeout: timeout})
	if err != nil {
		return err
	}

	return nil
}

func deleteImageFromProvisioner(imageName string, provisionerName string, location string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := client.RemoveProvisionerImage(ctx, &apic.RemoveProvisionerImageRequest{ImageName: imageName, ProvisionerName: provisionerName, Location: location})
	if err != nil {
		return err
	}

	return nil
}
