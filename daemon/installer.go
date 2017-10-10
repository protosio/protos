package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/boltdb/bolt"
	"github.com/docker/docker/api/types"
)

// InstallerMetadata holds metadata for the installer
type InstallerMetadata struct {
	Params      []string `json:"params"`
	Provides    []string `json:"provides"`
	Requires    []string `json:"requires"`
	Description string   `json:"description"`
}

// Installer represents an application installer
type Installer struct {
	Name     string             `json:"name"`
	ID       string             `json:"id"`
	Metadata *InstallerMetadata `json:"metadata"`
}

func getMetadata(labels map[string]string) (InstallerMetadata, error) {
	r := regexp.MustCompile("(^protos.installer.metadata.)(\\w+)")
	metadata := InstallerMetadata{}
	for label, value := range labels {
		labelParts := r.FindStringSubmatch(label)
		if len(labelParts) == 3 {
			switch labelParts[2] {
			case "params":
				metadata.Params = strings.Split(value, ",")
			case "provides":
				metadata.Provides = strings.Split(value, ",")
			case "requires":
				metadata.Requires = strings.Split(value, ",")
			case "description":
				metadata.Description = value
			}
		}

	}
	if metadata.Description == "" {
		return metadata, errors.New("Installer metadata field 'description' is mandatory.")
	}
	return metadata, nil
}

// GetInstallers gets all the local images and returns them
func GetInstallers() map[string]Installer {
	installers := make(map[string]Installer)
	log.Info("Retrieving installers")
	images, err := dockerClient.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		log.Warn(err)
		return nil
	}

	for _, image := range images {
		var name string
		if _, valid := image.Labels["protos"]; valid == false {
			continue
		}

		if len(image.RepoTags) > 0 && image.RepoTags[0] != "<none>:<none>" {
			name = image.RepoTags[0]
		} else {
			name = "n/a"
		}
		installers[image.ID] = Installer{Name: name, ID: image.ID}
	}

	return installers
}

// ReadInstaller reads a fresh copy of the installer
func ReadInstaller(installerID string) (Installer, error) {
	log.Info("Reading installer ", installerID)

	image, _, err := dockerClient.ImageInspectWithRaw(context.Background(), installerID)
	if err != nil {
		return Installer{}, err
	}

	var name string
	if len(image.RepoTags) > 0 {
		name = image.RepoTags[0]
	} else {
		name = "n/a"
	}

	installer := Installer{Name: name, ID: image.ID}

	metadata, err := getMetadata(image.Config.Labels)
	if err != nil {
		log.Warnf("Protos labeled image %s does not have any metadata", installerID)
		installer.Metadata = nil
	} else {
		installer.Metadata = &metadata
	}

	return installer, nil
}

// WriteMetadata adds metadata for an installer
func (installer *Installer) WriteMetadata(metadata InstallerMetadata) error {

	log.Infof("Writing metadata for installler %s", installer.ID)
	err := db.Update(func(tx *bolt.Tx) error {
		userBucket := tx.Bucket([]byte("installer"))

		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			return err
		}

		err = userBucket.Put([]byte(installer.ID), metadataJSON)
		if err != nil {
			return err
		}
		return nil
	})

	return err
}

// Remove Installer removes an installer image
func (installer *Installer) Remove() error {
	log.Info("Removing installer ", installer.Name, "[", installer.ID, "]")

	_, err := dockerClient.ImageRemove(context.Background(), installer.ID, types.ImageRemoveOptions{PruneChildren: true})
	if err != nil {
		return err
	}
	return nil
}
