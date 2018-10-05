package installer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/protosio/protos/capability"
	"github.com/protosio/protos/config"
	"github.com/protosio/protos/platform"
	"github.com/protosio/protos/util"
)

var gconfig = config.Get()
var log = util.GetLogger("installer")

// Metadata holds metadata for the installer
type Metadata struct {
	Params          []string                 `json:"params"`
	Provides        []string                 `json:"provides"`
	Requires        []string                 `json:"requires"`
	PublicPorts     []util.Port              `json:"publicports"`
	Description     string                   `json:"description"`
	PlatformID      string                   `json:"platformid"`
	PlatformType    string                   `json:"platformtype"`
	PersistancePath string                   `json:"persistancepath"`
	Capabilities    []*capability.Capability `json:"capabilities"`
}

// Installer represents an application installer
type Installer struct {
	Name      string               `json:"name"`
	ID        string               `json:"id"`
	Thumbnail string               `json:"thumbnail,omitempty"`
	Versions  map[string]*Metadata `json:"versions"`
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
func GetMetadata(labels map[string]string) (Metadata, error) {
	r := regexp.MustCompile("(^protos.installer.metadata.)(\\w+)")
	metadata := Metadata{}
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

// GetAll gets all the local images and returns them
func GetAll() (map[string]Installer, error) {
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
		installers[installerID] = Installer{ID: installerID, Name: installerName, Versions: map[string]*Metadata{}}
	}

	return installers, nil
}

// Read reads a fresh copy of the installer
func Read(installerID string) (Installer, error) {
	log.Info("Reading installer ", installerID)

	imgs, err := platform.GetAllDockerImages()
	if err != nil {
		return Installer{}, errors.New("Error retrieving installer " + installerID + ": " + err.Error())
	}

	installer := Installer{ID: installerID, Versions: map[string]*Metadata{}}

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

// ReadVersion returns the metadata for a specific installer version
func (inst Installer) ReadVersion(version string) (Metadata, error) {
	var metadata *Metadata
	var found bool

	if metadata, found = inst.Versions[version]; found == false {
		return *metadata, fmt.Errorf("Could not find version %s for installer %s", version, inst.ID)
	}
	return *metadata, nil
}

// Download downloads an installer from the application store
func (inst Installer) Download(version string) error {
	metadata, err := inst.ReadVersion(version)
	if err != nil {
		return errors.Wrapf(err, "Failed to download installer %s version %s", inst.ID, version)
	}

	log.Infof("Downloading platform image for installer %s(%s) version %s", inst.Name, inst.ID, version)
	return platform.PullDockerImage(metadata.PlatformID, inst.Name, version)
}

// IsPlatformImageAvailable checks if the associated docker image for an installer is available locally
func (inst Installer) IsPlatformImageAvailable(version string) bool {
	metadata, err := inst.ReadVersion(version)
	if err != nil {
		log.Error()
		return false
	}

	_, err = platform.GetDockerImage(metadata.PlatformID)
	if err != nil {
		if util.IsErrorType(err, platform.ErrDockerImageNotFound) == false {
			log.Error(err)
		}
		return false
	}
	return true
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

//
// App store operations
//

// StoreGetAll returns all installers from the application store
func StoreGetAll() (map[string]Installer, error) {
	installers := map[string]Installer{}
	resp, err := http.Get(gconfig.AppStoreURL + "/api/v1/installers/all")
	if err != nil {
		return installers, err
	}

	if err := util.HTTPBadResponse(resp); err != nil {
		return installers, err
	}

	err = json.NewDecoder(resp.Body).Decode(&installers)
	defer resp.Body.Close()
	if err != nil {
		return installers, fmt.Errorf("Something went wrong decoding the response from the application store: %s", err.Error())
	}
	return installers, nil
}

// StoreGetID returns a single installer based on its id
func StoreGetID(id string) (Installer, error) {
	installer := Installer{}
	resp, err := http.Get(gconfig.AppStoreURL + "/api/v1/installers/" + id)
	if err != nil {
		return installer, err
	}

	if err := util.HTTPBadResponse(resp); err != nil {
		return installer, err
	}

	err = json.NewDecoder(resp.Body).Decode(&installer)
	defer resp.Body.Close()
	if err != nil {
		return installer, fmt.Errorf("Something went wrong decoding the response from the application store: %s", err.Error())
	}
	return installer, nil
}

// StoreSearch takes a map of search terms and performs a search on the app store
func StoreSearch(key string, value string) (map[string]Installer, error) {
	var installers map[string]Installer

	resp, err := http.Get(fmt.Sprintf("%s/api/v1/search?%s=%s", gconfig.AppStoreURL, key, value))
	if err != nil {
		return installers, err
	}

	if err := util.HTTPBadResponse(resp); err != nil {
		return installers, err
	}

	err = json.NewDecoder(resp.Body).Decode(&installers)
	defer resp.Body.Close()
	if err != nil {
		return installers, fmt.Errorf("Something went wrong decoding the response from the application store: %s", err.Error())
	}
	return installers, nil

}
