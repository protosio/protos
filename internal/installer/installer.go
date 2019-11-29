package installer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/protosio/protos/internal/core"

	"github.com/protosio/protos/internal/config"

	"github.com/pkg/errors"

	"github.com/protosio/protos/internal/util"
)

var gconfig = config.Get()
var log = util.GetLogger("installer")

type installerParent interface {
	getPlatform() core.RuntimePlatform
	getTaskManager() core.TaskManager
}

type localInstaller struct {
	Installer
	Versions map[string]struct {
		core.InstallerMetadata
		Capabilities []map[string]string `json:"capabilities"`
	} `json:"versions"`
}

func (li localInstaller) convert(as *AppStore) core.Installer {
	li.Installer.Versions = map[string]core.InstallerMetadata{}
	for version, metadata := range li.Versions {
		for _, cap := range metadata.Capabilities {
			if capName, ok := cap["Name"]; ok {
				if _, err := as.cm.GetByName(capName); err == nil {
					metadata.InstallerMetadata.Capabilities = append(metadata.InstallerMetadata.Capabilities, capName)
				}
			}
		}
		li.Installer.Versions[version] = metadata.InstallerMetadata
	}
	li.Installer.parent = as
	return li.Installer
}

type localInstallers map[string]localInstaller

func (li localInstallers) convert(as *AppStore) map[string]core.Installer {
	installers := map[string]core.Installer{}
	for id, inst := range li {
		installers[id] = inst.convert(as)
	}
	return installers
}

// Installer represents an application installer
type Installer struct {
	Name      string                            `json:"name"`
	ID        string                            `json:"id"`
	Thumbnail string                            `json:"thumbnail,omitempty"`
	Versions  map[string]core.InstallerMetadata `json:"versions"`
	parent    installerParent
}

// AppStore manages and downloads application installers
type AppStore struct {
	rp core.RuntimePlatform
	tm core.TaskManager
	cm core.CapabilityManager
}

// CreateAppStore creates and returns an app store instance
func CreateAppStore(rp core.RuntimePlatform, tm core.TaskManager, cm core.CapabilityManager) *AppStore {
	if rp == nil || tm == nil || cm == nil {
		log.Panic("Failed to create AppStore: none of the inputs can be nil")
	}

	return &AppStore{rp: rp, tm: tm, cm: cm}
}

func validateInstallerCapabilities(cm core.CapabilityManager, capstring string) []string {
	caps := []string{}
	for _, capname := range strings.Split(capstring, ",") {
		_, err := cm.GetByName(capname)
		if err != nil {
			log.Error(err)
		} else {
			caps = append(caps, capname)
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
			log.Errorf("Installer port is out of range %s (valid range is 1-65535)", portstr)
			continue
		}
		port := util.Port{Nr: portNr}
		if strings.ToUpper(portParts[1]) == string(util.TCP) {
			port.Type = util.TCP
		} else if strings.ToUpper(portParts[1]) == string(util.UDP) {
			port.Type = util.UDP
		} else {
			log.Errorf("Invalid protocol(%s) for port(%s)", portParts[1], portParts[0])
			continue
		}
		ports = append(ports, port)
	}
	return ports
}

// parseMetadata parses the image metadata from the image labels
func parseMetadata(cm core.CapabilityManager, labels map[string]string) (core.InstallerMetadata, error) {
	r := regexp.MustCompile("(^protos.installer.metadata.)(\\w+)")
	metadata := core.InstallerMetadata{}
	for label, value := range labels {
		labelParts := r.FindStringSubmatch(label)
		if len(labelParts) == 3 {
			switch labelParts[2] {
			case "capabilities":
				metadata.Capabilities = validateInstallerCapabilities(cm, value)
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

//
// Installer methods
//

// GetMetadata returns the metadata for a specific installer version
func (inst Installer) GetMetadata(version string) (core.InstallerMetadata, error) {
	var metadata core.InstallerMetadata
	var found bool

	if metadata, found = inst.Versions[version]; found == false {
		return metadata, fmt.Errorf("Could not find version '%s' for installer with id '%s'", version, inst.ID)
	}
	return metadata, nil
}

// Download downloads an installer from the application store
func (inst Installer) Download(dt DownloadTask) error {
	metadata, err := inst.GetMetadata(dt.Version)
	if err != nil {
		return errors.Wrapf(err, "Failed to download installer '%s' version '%s'", inst.ID, dt.Version)
	}

	log.Infof("Downloading image '%s' for installer '%s'(%s) version '%s'", metadata.PlatformID, inst.Name, inst.ID, dt.Version)
	err = inst.parent.getPlatform().PullImage(dt.b, metadata.PlatformID, inst.Name, dt.Version)
	if err != nil {
		return errors.Wrapf(err, "Failed to download installer '%s' version '%s'", inst.ID, dt.Version)
	}
	return nil
}

// DownloadAsync triggers an async installer download, returns a generic task
func (inst Installer) DownloadAsync(version string, appID string) core.Task {
	return inst.parent.getTaskManager().New("Download application installer", &DownloadTask{Inst: inst, Version: version, AppID: appID})
}

// IsPlatformImageAvailable checks if the associated docker image for an installer is available locally
func (inst Installer) IsPlatformImageAvailable(version string) (bool, error) {
	metadata, err := inst.GetMetadata(version)
	if err != nil {
		return false, errors.Wrapf(err, "Failed to check local image for installer %s(%s)", inst.Name, inst.ID)
	}

	img, err := inst.parent.getPlatform().GetImage(metadata.PlatformID)
	if err != nil {
		return false, errors.Wrapf(err, "Failed to check local image for installer %s(%s)", inst.Name, inst.ID)
	}
	if img == nil {
		return false, nil
	}
	return true, nil
}

// Remove Installer removes an installer image
func (inst *Installer) Remove() error {
	log.Info("Removing installer ", inst.Name, "[", inst.ID, "]")

	for _, metadata := range inst.Versions {
		err := inst.parent.getPlatform().RemoveImage(metadata.PlatformID)
		if err != nil {
			return errors.Wrapf(err, "Failed to remove install %s(%s)", inst.Name, inst.ID)
		}
	}
	return nil
}

// GetLastVersion returns the last version available for the installer
func (inst Installer) GetLastVersion() string {
	vs := []*semver.Version{}
	for k := range inst.Versions {
		v, err := semver.NewVersion(k)
		if err != nil {
			log.Errorf("Error parsing version '%s' for installer '%s' : %s", k, inst.ID, err)
			continue
		}
		vs = append(vs, v)
	}
	sort.Sort(semver.Collection(vs))
	if len(vs) == 0 {
		log.Panicf("Installer '%s' should have at least 1 version. None found.", inst.ID)
	}
	return vs[len(vs)-1].String()
}

//
// InstallerCache operations (AppStore implements the core.InstallerCache interface for now)
//

// // GetLocalInstallers retrieves all locally available installers
// func (as *AppStore) GetLocalInstallers() (map[string]core.Installer, error) {
// 	installers := map[string]core.Installer{}
// 	log.Info("Retrieving local installers")

// 	imgs, err := as.rp.GetAllImages()
// 	if err != nil {
// 		return installers, errors.Wrap(err, "Error retrieving local installers")
// 	}

// 	for _, img := range imgs {
// 		if img.GetRepoTags()[0] == "n/a" {
// 			continue
// 		}
// 		installerStr := strings.Split(img.GetRepoTags()[0], ":")
// 		installerName := installerStr[0]
// 		installerID := util.String2SHA1(installerName)
// 		installers[installerID] = Installer{ID: installerID, Name: installerName, Versions: map[string]core.InstallerMetadata{}, parent: as}
// 	}

// 	return installers, nil

// }

// // GetLocalInstaller retrieves an installer if its available locally
// func (as *AppStore) GetLocalInstaller(id string) (core.Installer, error) {
// 	log.Infof("Retrieving local installer with id '%s'", id)

// 	imgs, err := as.rp.GetAllImages()
// 	if err != nil {
// 		return Installer{}, errors.Wrapf(err, "Error retrieving local installer with id '%s'", id)
// 	}

// 	installer := Installer{ID: id, Versions: map[string]core.InstallerMetadata{}, parent: as}

// 	for _, img := range imgs {
// 		if img.GetRepoTags()[0] == "n/a" {
// 			continue
// 		}
// 		installerStr := strings.Split(img.GetRepoTags()[0], ":")
// 		installerName := installerStr[0]
// 		installerVersion := installerStr[1]
// 		instID := util.String2SHA1(installerName)
// 		if id != instID {
// 			continue
// 		}
// 		installer.Name = installerName

// 		metadata, err := parseMetadata(as.cm, img.GetLabels())
// 		if err != nil {
// 			log.Warnf("Error while parsing metadata for installer %s, version %s: %v", id, installerVersion, err)
// 		}
// 		metadata.PersistancePath = img.GetDataPath()
// 		metadata.PlatformID = img.GetID()
// 		installer.Versions[installerVersion] = metadata

// 	}

// 	if len(installer.Versions) == 0 {
// 		return nil, errors.New("Could not find installer '" + id + "'")
// 	}

// 	return installer, nil
// }

// // RemoveLocalInstaller removes an installer image that has been downloaded locally
// func (as *AppStore) RemoveLocalInstaller(id string) error {
// 	inst, err := as.GetLocalInstaller(id)
// 	if err != nil {
// 		return errors.Wrapf(err, "Failed to remove local installer with id '%s'", id)
// 	}

// 	log.Info("Removing installer ", inst.(Installer).Name, "[", inst.(Installer).ID, "]")

// 	for _, metadata := range inst.(Installer).Versions {
// 		err := as.rp.RemoveImage(metadata.PlatformID)
// 		if err != nil {
// 			return errors.Wrapf(err, "Failed to remove local installer %s[%s]", inst.(Installer).Name, id)
// 		}
// 	}
// 	return nil
// }

//
// AppStore operations
//

type httpClient interface {
	Get(url string) (resp *http.Response, err error)
}

var getHTTPClient = func() httpClient {
	return http.DefaultClient
}

// GetInstallers returns all installers from the application store
func (as *AppStore) GetInstallers() (map[string]core.Installer, error) {
	installers := map[string]core.Installer{}
	localInstallers := localInstallers{}

	client := getHTTPClient()

	url := gconfig.AppStoreURL + "/api/v1/installers/all"
	log.Debugf("Querying app store at '%s'", url)
	resp, err := client.Get(url)
	if err != nil {
		return installers, errors.Wrap(err, "Could not retrieve installers from app store")
	}

	if err := util.HTTPBadResponse(resp); err != nil {
		return installers, errors.Wrap(err, "Could not retrieve installers from app store")
	}

	err = json.NewDecoder(resp.Body).Decode(&localInstallers)
	defer resp.Body.Close()
	if err != nil {
		return installers, errors.Wrap(err, "Could not retrieve installers from app store. Decoding error")
	}

	return localInstallers.convert(as), nil
}

// GetInstaller returns a single installer based on its id
func (as *AppStore) GetInstaller(id string) (core.Installer, error) {
	localInstaller := localInstaller{}
	client := getHTTPClient()
	url := fmt.Sprintf("%s/api/v1/installers/%s", gconfig.AppStoreURL, id)
	log.Debugf("Querying app store at '%s'", url)
	resp, err := client.Get(url)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not retrieve installer '%s' from app store", id)
	}

	if err := util.HTTPBadResponse(resp); err != nil {
		return nil, errors.Wrapf(err, "Could not retrieve installer '%s' from app store", id)
	}

	err = json.NewDecoder(resp.Body).Decode(&localInstaller)
	defer resp.Body.Close()
	if err != nil {
		return nil, errors.Wrapf(err, "Could not retrieve installer '%s' from app store. Decoding error", id)
	}

	return localInstaller.convert(as), nil
}

// Search takes a map of search terms and performs a search on the app store
func (as *AppStore) Search(key string, value string) (map[string]core.Installer, error) {
	installers := map[string]core.Installer{}
	localInstallers := localInstallers{}

	client := getHTTPClient()
	url := fmt.Sprintf("%s/api/v1/search?%s=%s", gconfig.AppStoreURL, key, value)
	log.Debugf("Querying app store at '%s'", url)
	resp, err := client.Get(url)
	if err != nil {
		return installers, errors.Wrap(err, "Could not retrieve search results from the app store")
	}

	if err := util.HTTPBadResponse(resp); err != nil {
		return installers, errors.Wrap(err, "Could not retrieve search results from the app store")
	}

	err = json.NewDecoder(resp.Body).Decode(&localInstallers)
	defer resp.Body.Close()
	if err != nil {
		return installers, errors.Wrap(err, "Could not retrieve search results from the app store. Decoding error")
	}

	return localInstallers.convert(as), nil
}

//
// AppStore methods that satisfy the installerParent interface
//

func (as *AppStore) getPlatform() core.RuntimePlatform {
	return as.rp
}

func (as *AppStore) getTaskManager() core.TaskManager {
	return as.tm
}
