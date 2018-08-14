package daemon

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/nustiueudinastea/protos/capability"
	"github.com/nustiueudinastea/protos/platform"
	"github.com/nustiueudinastea/protos/util"
)

// InstallerMetadata holds metadata for the installer
type InstallerMetadata struct {
	Params          []string                 `json:"params"`
	Provides        []string                 `json:"provides"`
	Requires        []string                 `json:"requires"`
	PublicPorts     []util.Port              `json:"publicports"`
	Description     string                   `json:"description"`
	PlatformID      string                   `json:"platformid"`
	PersistancePath string                   `json:"persistancepath"`
	Capabilities    []*capability.Capability `json:"-"`
}

// Installer represents an application installer
type Installer struct {
	Name     string                        `json:"name"`
	ID       string                        `json:"id"`
	Versions map[string]*InstallerMetadata `json:"versions"`
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

func parsePublicPorts(publicports string) []util.Port {
	ports := []util.Port{}
	for _, portstr := range strings.Split(publicports, ",") {
		portParts := strings.Split(portstr, "/")
		if len(portParts) != 2 {
			log.Errorf("Error parsing installer port string %s", portstr)
			continue
		}
		portNr, err := strconv.Atoi(portParts[0])
		if err != nil {
			log.Errorf("Error parsing installer port string %s", portstr)
			continue
		}
		if portNr < 1 || portNr > 0xffff {
			log.Errorf("Installer port is out of range %s", portstr)
			continue
		}
		port := util.Port{Nr: portNr}
		if strings.ToUpper(portParts[1]) == string(util.TCP) {
			port.Type = util.TCP
		} else if strings.ToUpper(portParts[1]) == string(util.UDP) {
			port.Type = util.UDP
		} else {
			log.Errorf("Installer port protocol is invalid %s", portstr)
			continue
		}
		ports = append(ports, port)
	}
	return ports
}

// GetMetadata parses the image metadata from the image labels
func GetMetadata(labels map[string]string) (InstallerMetadata, error) {
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
			case "publicports":
				metadata.PublicPorts = parsePublicPorts(value)
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
		if img.RepoTags[0] == "n/a" {
			continue
		}
		installerStr := strings.Split(img.RepoTags[0], ":")
		installerName := installerStr[0]
		installerID := util.String2SHA1(installerName)
		installers[installerID] = Installer{ID: installerID, Name: installerName, Versions: map[string]*InstallerMetadata{}}
	}

	return installers, nil
}

// ReadInstaller reads a fresh copy of the installer
func ReadInstaller(installerID string) (Installer, error) {
	log.Info("Reading installer ", installerID)

	imgs, err := platform.GetAllDockerImages()
	if err != nil {
		return Installer{}, errors.New("Error retrieving installer " + installerID + ": " + err.Error())
	}

	installer := Installer{ID: installerID, Versions: map[string]*InstallerMetadata{}}

	for _, img := range imgs {
		if img.RepoTags[0] == "n/a" {
			continue
		}
		installerStr := strings.Split(img.RepoTags[0], ":")
		installerName := installerStr[0]
		installerVersion := installerStr[1]
		instID := util.String2SHA1(installerName)
		if installerID != instID {
			continue
		}
		installer.Name = installerName

		img, err := platform.GetDockerImage(img.ID)
		if err != nil {
			return Installer{}, errors.New("Error retrieving docker image: " + err.Error())
		}

		persistancePath, err := platform.GetDockerImageDataPath(img)
		if err != nil {
			return Installer{}, errors.New("Installer " + installerID + " is invalid: " + err.Error())
		}

		metadata, err := GetMetadata(img.Config.Labels)
		if err != nil {
			log.Warnf("Error while parsing metadata for installer %s, version %s: %v", installerID, installerVersion, err)
		}
		metadata.PersistancePath = persistancePath
		metadata.PlatformID = img.ID
		installer.Versions[installerVersion] = &metadata

	}

	if len(installer.Versions) == 0 {
		return Installer{}, errors.New("Could not find installer " + installerID)
	}

	return installer, nil
}

// DownloadInstaller downloads an installer from the application store
func DownloadInstaller(name string, version string) error {
	log.Info("Downloading installer ", name, ", version ", version)
	return platform.PullDockerImage(name, version)
}

// Remove Installer removes an installer image
func (installer *Installer) Remove() error {
	log.Info("Removing installer ", installer.Name, "[", installer.ID, "]")

	for _, metadata := range installer.Versions {
		err := platform.RemoveDockerImage(metadata.PlatformID)
		if err != nil {
			return errors.New("Failed to remove installer: " + err.Error())
		}
	}
	return nil
}
