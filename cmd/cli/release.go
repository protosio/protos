package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"text/tabwriter"

	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/cloud"
	"github.com/protosio/protos/internal/release"
	"github.com/urfave/cli/v2"
)

const (
	releasesURL = "https://releases.protos.io/releases.json"
)

var cmdRelease *cli.Command = &cli.Command{
	Name:  "release",
	Usage: "Manage Protos releases",
	Subcommands: []*cli.Command{
		{
			Name:  "available",
			Usage: "Show the available Protosd releases",
			Action: func(c *cli.Context) error {
				releases, err := getProtosAvailableReleases()
				if err != nil {
					return err
				}
				printProtosAvailableReleases(releases)
				return nil
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

				err := printProtosCloudImages(cloudName)
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

				return uploadLocalImageToCloud(imagePath, imageName, cloudName, cloudLocation)
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

func printProtosAvailableReleases(releases release.Releases) {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)

	defer w.Flush()

	fmt.Fprintf(w, " %s\t%s\t%s\t", "Version", "Date", "Description")
	fmt.Fprintf(w, "\n %s\t%s\t%s\t", "-------", "----", "-----------")
	for _, release := range releases.Releases {
		fmt.Fprintf(w, "\n %s\t%s\t%s\t", release.Version, release.ReleaseDate.Format("Jan 2, 2006"), release.Description)
	}
	fmt.Fprint(w, "\n")
}

func getProtosAvailableReleases() (release.Releases, error) {
	var releases release.Releases
	resp, err := http.Get(releasesURL)
	if err != nil {
		return releases, errors.Wrapf(err, "Failed to retrieve releases from '%s'", releasesURL)
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&releases)
	if err != nil {
		return releases, errors.Wrap(err, "Failed to JSON decode the releases response")
	}

	if len(releases.Releases) == 0 {
		return releases, errors.Errorf("Something went wrong. Parsed 0 releases from '%s'", releasesURL)
	}

	return releases, nil
}

func printProtosCloudImages(cloudName string) error {
	cm, err := cloud.NewCloudManager(envi.DB)
	if err != nil {
		return err
	}

	// init cloud
	provider, err := cm.GetCloudProvider(cloudName)
	if err != nil {
		return errors.Wrapf(err, "Could not retrieve cloud '%s'", cloudName)
	}

	err = provider.Init()
	if err != nil {
		return errors.Wrapf(err, "Failed to connect to cloud provider '%s'(%s) API", cloudName, provider.TypeStr())
	}

	images, err := provider.GetProtosImages()
	if err != nil {
		return errors.Wrapf(err, "Failed to retrieve cloud images")
	}

	//
	// do the printing
	//
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)

	defer w.Flush()

	fmt.Fprintf(w, " %s\t%s\t%s\t", "Version", "ID", "Location")
	fmt.Fprintf(w, "\n %s\t%s\t%s\t", "-------", "--", "--------")
	for _, img := range images {
		fmt.Fprintf(w, "\n %s\t%s\t%s\t", img.Name, img.ID, img.Location)
	}
	fmt.Fprint(w, "\n")

	return nil
}

func uploadLocalImageToCloud(imagePath string, imageName string, cloudName string, cloudLocation string) error {
	errMsg := fmt.Sprintf("Failed to upload local image '%s' to cloud '%s'", imagePath, cloudName)
	// check local image file
	finfo, err := os.Stat(imagePath)
	if err != nil {
		return errors.Wrapf(err, "Could not stat file '%s'", imagePath)
	}
	if finfo.IsDir() {
		return fmt.Errorf("Path '%s' is a directory", imagePath)
	}
	if finfo.Size() == 0 {
		return fmt.Errorf("Image '%s' has 0 bytes", imagePath)
	}

	cm, err := cloud.NewCloudManager(envi.DB)
	if err != nil {
		return err
	}

	// init cloud
	provider, err := cm.GetCloudProvider(cloudName)
	if err != nil {
		return errors.Wrapf(err, "Could not retrieve cloud '%s'", cloudName)
	}

	err = provider.Init()
	if err != nil {
		return errors.Wrapf(err, "Failed to connect to cloud provider '%s'(%s) API", cloudName, provider.TypeStr())
	}

	// find image
	images, err := provider.GetImages()
	if err != nil {
		return errors.Wrap(err, errMsg)
	}
	for _, img := range images {
		if img.Location == cloudLocation && img.Name == imageName {
			return errors.Wrap(fmt.Errorf("Found an image with the same name"), errMsg)
		}
	}

	// upload image
	_, err = provider.UploadLocalImage(imagePath, imageName, cloudLocation)
	if err != nil {
		return errors.Wrap(err, errMsg)
	}

	return nil
}

func deleteImageFromCloud(imageName string, cloudName string, cloudLocation string) error {
	errMsg := fmt.Sprintf("Failed to delete image '%s' from cloud '%s'", imageName, cloudName)
	// init cloud
	cm, err := cloud.NewCloudManager(envi.DB)
	if err != nil {
		return err
	}
	provider, err := cm.GetCloudProvider(cloudName)
	if err != nil {
		return errors.Wrapf(err, "Could not retrieve cloud '%s'", cloudName)
	}

	err = provider.Init()
	if err != nil {
		return errors.Wrapf(err, "Failed to connect to cloud provider '%s'(%s) API", cloudName, provider.TypeStr())
	}

	// delete image
	err = provider.RemoveImage(imageName, cloudLocation)
	if err != nil {
		return errors.Wrap(err, errMsg)
	}

	return nil
}
