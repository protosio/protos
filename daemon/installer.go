package daemon

import (
	"errors"
	"protos/capability"
	"protos/platform"
	"regexp"
	"strings"
)

// InstallerMetadata holds metadata for the installer
type InstallerMetadata struct {
	Params       []string                 `json:"params"`
	Provides     []string                 `json:"provides"`
	Requires     []string                 `json:"requires"`
	Description  string                   `json:"description"`
	Capabilities []*capability.Capability `json:"-"`
}

// Installer represents an application installer
type Installer struct {
	Name     string             `json:"name"`
	ID       string             `json:"id"`
	Metadata *InstallerMetadata `json:"metadata"`
}

func parseInstallerCapabilities(capstring string) []*capability.Capability {
	caps := []*capability.Capability{}
	for _, capname := range strings.Split(capstring, ",") {
		cap, err := capability.GetByName(capname)
		if err != nil {
			log.Error(err)
		} else {
			caps = append(caps, cap)
		}
	}
	return caps
}

func getMetadata(labels map[string]string) (InstallerMetadata, error) {
	r := regexp.MustCompile("(^protos.installer.metadata.)(\\w+)")
	metadata := InstallerMetadata{}
	for label, value := range labels {
		labelParts := r.FindStringSubmatch(label)
		if len(labelParts) == 3 {
			switch labelParts[2] {
			case "capabilities":
				metadata.Capabilities = parseInstallerCapabilities(value)
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
		return metadata, errors.New("installer metadata field 'description' is mandatory")
	}
	return metadata, nil
}

// GetInstallers gets all the local images and returns them
func GetInstallers() (map[string]Installer, error) {
	installers := make(map[string]Installer)
	log.Info("Retrieving installers")

	imgs, err := platform.GetAllDockerImages()
	if err != nil {
		return installers, errors.New("Error retrieving docker images: " + err.Error())
	}

	for _, img := range imgs {
		installers[img.ID] = Installer{ID: img.ID, Name: img.RepoTags[0]}
	}

	return installers, nil
}

// ReadInstaller reads a fresh copy of the installer
func ReadInstaller(installerID string) (Installer, error) {
	log.Info("Reading installer ", installerID)

	img, err := platform.GetDockerImage(installerID)
	if err != nil {
		return Installer{}, errors.New("Error retrieving docker image: " + err.Error())
	}

	installer := Installer{Name: img.RepoTags[0], ID: img.ID}
	metadata, err := getMetadata(img.Config.Labels)
	if err != nil {
		log.Warnf("Protos labeled image %s does not have any metadata", installerID)
		installer.Metadata = nil
	} else {
		installer.Metadata = &metadata
	}

	return installer, nil
}

// Remove Installer removes an installer image
func (installer *Installer) Remove() error {
	log.Info("Removing installer ", installer.Name, "[", installer.ID, "]")

	err := platform.RemoveDockerImage(installer.ID)
	if err != nil {
		return errors.New("Failed to remove installer: " + err.Error())
	}
	return nil
}
