package daemon

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/docker/docker/api/types"
)

// InstallerMetadata holds metadata for the installer
type InstallerMetadata struct {
	Params      []string `json:"params"`
	Provides    []string `json:"provides"`
	Requires    []string `json:"requires"`
	Description []string `json:"description"`
}

// Installer represents an application installer
type Installer struct {
	Name     string            `json:"name"`
	ID       string            `json:"id"`
	Metadata InstallerMetadata `json:"metadata"`
}

// GetInstallers gets all the local images and returns them
func GetInstallers() map[string]Installer {
	client := Gconfig.DockerClient
	installers := make(map[string]Installer)
	log.Info("Retrieving installers")
	images, err := client.ImageList(context.Background(), types.ImageListOptions{})
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
	var installerBucket = []byte("installer")
	client := Gconfig.DockerClient
	db := Gconfig.Db

	image, _, err := client.ImageInspectWithRaw(context.Background(), installerID)
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

	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(installerBucket)
		if bucket == nil {
			return fmt.Errorf("Bucket %q not found!", installerBucket)
		}

		metadata := bucket.Get([]byte(installerID))
		if len(metadata) == 0 {
			log.Warnf("Image %s does not have any metadata stored", installerID)
			return nil
		}

		if err := json.Unmarshal(metadata, &installer.Metadata); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return Installer{}, err
	}

	return installer, nil
}

func (installer *Installer) writeMetadata(metadata InstallerMetadata) error {

	log.Infof("Writing metadata for installler %s", installer.ID)
	err := Gconfig.Db.Update(func(tx *bolt.Tx) error {
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
	client := Gconfig.DockerClient

	_, err := client.ImageRemove(context.Background(), installer.ID, types.ImageRemoveOptions{})
	if err != nil {
		log.Error(err)
		return err
	}
	return nil
}
