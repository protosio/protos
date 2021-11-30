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
			ArgsUsage: "<cloud name>",
			Usage:     "Retrieve and list the Protosd images available in a specific cloud provider",
			Action: func(c *cli.Context) error {
				cloudName := c.Args().Get(0)
				if cloudName == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}

				err := listProtosCloudImages(cloudName)
				if err != nil {
					return err
				}
				return nil
			},
		},
		{
			Name:      "upload",
			ArgsUsage: "<image path> <image name>",
			Usage:     "Upload a locally built image to a cloud provider. Used for development",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:        "cloud",
					Usage:       "Specify which `CLOUD` provider to upload the image to",
					Required:    true,
					Destination: &cloudName,
				},
				&cli.StringFlag{
					Name:        "location",
					Usage:       "Specify one of the supported `LOCATION`s to upload the image (cloud specific). Not required for all cloud providers",
					Required:    false,
					Destination: &cloudLocation,
				},
				&cli.DurationFlag{
					Name:     "timeout",
					Usage:    "Upload timeout in minutes",
					Required: false,
					Value:    time.Minute * 25,
				},
			},
			Action: func(c *cli.Context) error {
				imagePath := c.Args().Get(0)
				if imagePath == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}

				imageName := c.Args().Get(1)
				if imageName == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}

				timeout := c.Int("timeout")

				return uploadLocalImageToCloud(imagePath, imageName, cloudName, cloudLocation, int32(timeout))
			},
		},
		{
			Name:      "delete",
			ArgsUsage: "<image name>",
			Usage:     "Delete an image from a cloud provider",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:        "cloud",
					Usage:       "Specify which `CLOUD` provider to upload the image to",
					Required:    true,
					Destination: &cloudName,
				},
				&cli.StringFlag{
					Name:        "location",
					Usage:       "Specify one of the supported `LOCATION`s to upload the image (cloud specific). Not required for all cloud providers",
					Required:    false,
					Destination: &cloudLocation,
				},
			},
			Action: func(c *cli.Context) error {
				imageName := c.Args().Get(0)
				if imageName == "" {
					cli.ShowSubcommandHelp(c)
					os.Exit(1)
				}

				return deleteImageFromCloud(imageName, cloudName, cloudLocation)
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

func listProtosCloudImages(cloudName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetCloudImages(ctx, &apic.GetCloudImagesRequest{Name: cloudName})
	if err != nil {
		return err
	}
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)

	defer w.Flush()

	fmt.Fprintf(w, " %s\t%s\t%s\t", "Version", "ID", "Location")
	fmt.Fprintf(w, "\n %s\t%s\t%s\t", "-------", "--", "--------")
	for _, img := range resp.CloudImages {
		fmt.Fprintf(w, "\n %s\t%s\t%s\t", img.Name, img.Id, img.Location)
	}
	fmt.Fprint(w, "\n")

	return nil
}

func uploadLocalImageToCloud(imagePath string, imageName string, cloudName string, cloudLocation string, timeout int32) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Second)
	defer cancel()
	_, err := client.UploadCloudImage(ctx, &apic.UploadCloudImageRequest{ImagePath: imagePath, ImageName: imageName, CloudName: cloudName, CloudLocation: cloudLocation, Timeout: timeout})
	if err != nil {
		return err
	}

	return nil
}

func deleteImageFromCloud(imageName string, cloudName string, cloudLocation string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := client.RemoveCloudImage(ctx, &apic.RemoveCloudImageRequest{ImageName: imageName, CloudName: cloudName, CloudLocation: cloudLocation})
	if err != nil {
		return err
	}

	return nil
}
