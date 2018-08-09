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
	Params       []string                 `json:"params"`
	Provides     []string                 `json:"provides"`
	Requires     []string                 `json:"requires"`
	PublicPorts  []util.Port              `json:"publicports"`
	Description  string                   `json:"description"`
	Capabilities []*capability.Capability `json:"-"`
}

// Installer represents an application installer
type Installer struct {
	Name            string             `json:"name"`
	ID              string             `json:"id"`
	PlatformID      string             `json:"platformid"`
	PersistancePath string             `json:"persistancepath"`
	Metadata        *InstallerMetadata `json:"metadata"`
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
		installerID := util.String2SHA1(strings.Split(img.RepoTags[0], ":")[0])
		installers[installerID] = Installer{ID: installerID, Name: img.RepoTags[0], PlatformID: img.ID}
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

	imageID := ""
	installerName := ""
	for _, img := range imgs {
		installerStr := strings.Split(img.RepoTags[0], ":")
		installerName = installerStr[0]
		instID := util.String2SHA1(installerName)
		if installerID == instID {
			imageID = img.ID
		}
	}

	if imageID == "" {
		return Installer{}, errors.New("Could not find installer " + installerID)
	}

	img, err := platform.GetDockerImage(imageID)
	if err != nil {
		return Installer{}, errors.New("Error retrieving docker image: " + err.Error())
	}

	persistancePath, err := platform.GetDockerImageDataPath(img)
	if err != nil {
		return Installer{}, errors.New("Installer " + installerID + " is invalid: " + err.Error())
	}

	installer := Installer{Name: installerName, ID: installerID, PlatformID: img.ID, PersistancePath: persistancePath}
	metadata, err := GetMetadata(img.Config.Labels)
	if err != nil {
		log.Warnf("Protos labeled image %s does not have any metadata", installerID)
		installer.Metadata = nil
	} else {
		installer.Metadata = &metadata
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

	err := platform.RemoveDockerImage(installer.PlatformID)
	if err != nil {
		return errors.New("Failed to remove installer: " + err.Error())
	}
	return nil
}
