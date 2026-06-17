package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
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
				&cli.BoolFlag{
					Name:  "follow",
					Usage: "Stream upload task progress until completion",
				},
				&cli.BoolFlag{
					Name:  "jsonl",
					Usage: "Stream upload task progress as JSON lines",
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

				return uploadLocalImageToProvisioner(imagePath, imageName, cloudName, cloudLocation, int32(timeout), c.Bool("follow") || c.Bool("jsonl"), c.Bool("jsonl"))
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
		{
			Name:      "audit",
			ArgsUsage: "<provisioner name>",
			Usage:     "Dry-run audit of Protos images and provider-owned image resources",
			Flags: []cli.Flag{
				&cli.DurationFlag{
					Name:  "older-than",
					Usage: "Mark canonical images older than `DURATION` as cleanup candidates",
				},
				&cli.StringFlag{
					Name:        "location",
					Usage:       "Restrict audit to a specific `LOCATION`",
					Required:    false,
					Destination: &cloudLocation,
				},
			},
			Action: func(c *cli.Context) error {
				provisionerName := c.Args().Get(0)
				if provisionerName == "" {
					return showSubcommandHelp(c)
				}
				olderThan := c.Duration("older-than")
				if olderThan < 0 {
					return fmt.Errorf("--older-than cannot be negative")
				}
				return auditProvisionerImages(provisionerName, cloudLocation, olderThan)
			},
		},
		{
			Name:      "cleanup",
			ArgsUsage: "<provisioner name>",
			Usage:     "List or delete old canonical Protos images from a provisioner",
			Flags: []cli.Flag{
				&cli.DurationFlag{
					Name:     "older-than",
					Usage:    "Only delete canonical images older than `DURATION`",
					Required: true,
				},
				&cli.StringFlag{
					Name:        "location",
					Usage:       "Restrict cleanup to a specific `LOCATION`",
					Required:    false,
					Destination: &cloudLocation,
				},
				&cli.BoolFlag{
					Name:  "confirm",
					Usage: "Actually delete matching images. Without this flag the command only lists candidates",
				},
			},
			Action: func(c *cli.Context) error {
				provisionerName := c.Args().Get(0)
				if provisionerName == "" {
					return showSubcommandHelp(c)
				}
				olderThan := c.Duration("older-than")
				if olderThan <= 0 {
					return fmt.Errorf("--older-than must be greater than zero")
				}
				return cleanupProvisionerImages(provisionerName, cloudLocation, olderThan, c.Bool("confirm"))
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

	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t", "Name", "Logical", "Date", "Age", "Location", "ID")
	fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t%s\t%s\t", "----", "-------", "----", "---", "--------", "--")
	for _, img := range sortedProvisionerImages(resp.Images) {
		fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t%s\t%s\t", img.Name, img.LogicalName, imageDateLabel(img), imageAgeLabel(img), img.Location, img.Id)
	}
	fmt.Fprint(w, "\n")

	return nil
}

func auditProvisionerImages(provisionerName string, location string, olderThan time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	resp, err := client.GetProvisionerImages(ctx, &apic.GetProvisionerImagesRequest{Name: provisionerName})
	cancel()
	if err != nil {
		return err
	}
	cutoff := time.Time{}
	if olderThan > 0 {
		cutoff = time.Now().Add(-olderThan)
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	rows := auditProvisionerImageRows(resp.Images, location, cutoff)
	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t%s\t", "State", "Name", "Logical", "Date", "Age", "Location", "ID")
	fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t%s\t%s\t%s\t", "-----", "----", "-------", "----", "---", "--------", "--")
	for _, row := range rows {
		img := row.Image
		fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t%s\t%s\t%s\t", row.State, img.Name, img.LogicalName, imageDateLabel(img), imageAgeLabel(img), img.Location, img.Id)
	}
	if len(rows) == 0 {
		fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t%s\t%s\t%s\t", "none", "-", "-", "-", "-", locationLabel(location), "-")
	}
	fmt.Fprint(w, "\n")
	return nil
}

func cleanupProvisionerImages(provisionerName string, location string, olderThan time.Duration, confirm bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	resp, err := client.GetProvisionerImages(ctx, &apic.GetProvisionerImagesRequest{Name: provisionerName})
	cancel()
	if err != nil {
		return err
	}
	candidates := cleanupProvisionerImageCandidates(resp.Images, location, time.Now().Add(-olderThan))

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	action := "would-delete"
	if confirm {
		action = "deleted"
	}
	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t", "Action", "Name", "Age", "Location", "ID")
	fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t%s\t", "------", "----", "---", "--------", "--")
	for _, img := range candidates {
		if confirm {
			if err := deleteImageFromProvisioner(imageDeleteRef(img), provisionerName, img.Location); err != nil {
				return err
			}
		}
		fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t%s\t", action, img.Name, imageAgeLabel(img), img.Location, img.Id)
	}
	if len(candidates) == 0 {
		fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t%s\t", "none", "-", "-", locationLabel(location), "-")
	}
	fmt.Fprint(w, "\n")
	return nil
}

type provisionerImageAuditRow struct {
	State string
	Image *apic.CloudSpecificImage
}

func auditProvisionerImageRows(images map[string]*apic.CloudSpecificImage, location string, cutoff time.Time) []provisionerImageAuditRow {
	sorted := sortedProvisionerImages(images)
	rows := make([]provisionerImageAuditRow, 0, len(sorted))
	for _, img := range sorted {
		if img == nil {
			continue
		}
		if location != "" && img.Location != location {
			continue
		}
		rows = append(rows, provisionerImageAuditRow{
			State: auditProvisionerImageState(img, cutoff),
			Image: img,
		})
	}
	return rows
}

func auditProvisionerImageState(img *apic.CloudSpecificImage, cutoff time.Time) string {
	if img == nil {
		return "unknown"
	}
	if img.Canonical {
		if !cutoff.IsZero() && img.UpdatedAtUnix > 0 && time.Unix(img.UpdatedAtUnix, 0).Before(cutoff) {
			return "cleanup-candidate"
		}
		return "canonical"
	}
	return "legacy-protos"
}

func cleanupProvisionerImageCandidates(images map[string]*apic.CloudSpecificImage, location string, cutoff time.Time) []*apic.CloudSpecificImage {
	candidates := make([]*apic.CloudSpecificImage, 0, len(images))
	for _, img := range images {
		if img == nil || !img.Canonical || img.UpdatedAtUnix <= 0 {
			continue
		}
		if location != "" && img.Location != location {
			continue
		}
		if time.Unix(img.UpdatedAtUnix, 0).Before(cutoff) {
			candidates = append(candidates, img)
		}
	}
	sortProvisionerImages(candidates)
	return candidates
}

func sortedProvisionerImages(images map[string]*apic.CloudSpecificImage) []*apic.CloudSpecificImage {
	out := make([]*apic.CloudSpecificImage, 0, len(images))
	for _, img := range images {
		if img != nil {
			out = append(out, img)
		}
	}
	sortProvisionerImages(out)
	return out
}

func sortProvisionerImages(images []*apic.CloudSpecificImage) {
	sort.Slice(images, func(i, j int) bool {
		left, right := images[i], images[j]
		if left.UpdatedAtUnix != right.UpdatedAtUnix {
			return left.UpdatedAtUnix > right.UpdatedAtUnix
		}
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		return left.Id < right.Id
	})
}

func imageDateLabel(img *apic.CloudSpecificImage) string {
	if img == nil {
		return "n/a"
	}
	if img.UpdatedAtUnix > 0 {
		return time.Unix(img.UpdatedAtUnix, 0).UTC().Format("2006-01-02 15:04")
	}
	if img.DateSuffix != "" {
		return img.DateSuffix
	}
	return "n/a"
}

func imageAgeLabel(img *apic.CloudSpecificImage) string {
	if img == nil || img.UpdatedAtUnix <= 0 {
		return "n/a"
	}
	age := time.Since(time.Unix(img.UpdatedAtUnix, 0))
	if age < 0 {
		return "0h"
	}
	if age < 24*time.Hour {
		hours := int(age / time.Hour)
		if hours < 1 {
			return "<1h"
		}
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dd", int(age/(24*time.Hour)))
}

func imageDeleteRef(img *apic.CloudSpecificImage) string {
	if img == nil {
		return ""
	}
	if img.Name != "" {
		return img.Name
	}
	return img.Id
}

func locationLabel(location string) string {
	if location == "" {
		return "all"
	}
	return location
}

func uploadLocalImageToProvisioner(imagePath string, imageName string, provisionerName string, location string, timeout int32, follow bool, jsonl bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := client.UploadProvisionerImage(ctx, &apic.UploadProvisionerImageRequest{ImagePath: imagePath, ImageName: imageName, ProvisionerName: provisionerName, Location: location, Timeout: timeout})
	if err != nil {
		return err
	}
	taskID := resp.GetTaskId()
	if taskID == "" {
		if resp.GetId() != "" && !jsonl {
			fmt.Printf("uploaded image id: %s\n", resp.GetId())
		}
		return nil
	}
	if !jsonl {
		fmt.Printf("queued upload task: %s\n", taskID)
	}
	if !follow {
		return nil
	}
	followCtx, followCancel := context.WithTimeout(context.Background(), time.Duration(timeout+5)*time.Minute)
	defer followCancel()
	task, err := followTaskUntilTerminal(followCtx, taskID, jsonl)
	if err != nil {
		return err
	}
	if !jsonl {
		if imageID := uploadTaskImageID(task); imageID != "" {
			fmt.Printf("uploaded image id: %s\n", imageID)
		}
	}

	return nil
}

func uploadTaskImageID(task *apic.Task) string {
	if task == nil || task.GetResultJson() == "" {
		return ""
	}
	var result struct {
		ImageID string `json:"image_id"`
	}
	if err := json.Unmarshal([]byte(task.GetResultJson()), &result); err != nil {
		return ""
	}
	return result.ImageID
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
