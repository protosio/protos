package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/protosio/protos/internal/capability"
	"github.com/protosio/protos/internal/task"
	"github.com/regclient/regclient"
	regmanifest "github.com/regclient/regclient/types/manifest"
	"github.com/regclient/regclient/types/ref"

	"github.com/protosio/protos/internal/config"

	"github.com/pkg/errors"

	"github.com/protosio/protos/internal/util"
)

var gconfig = config.Get()
var log = util.GetLogger("installer")

type ImageManager interface {
	PullImage(imageRef string) error
	ImageExistsLocally(id string) (bool, error)
	RemoveImage(id string) error
}

// InstallerMetadata holds metadata for the installer
type InstallerMetadata struct {
	Params          []string    `json:"params"`
	Provides        []string    `json:"provides"`
	Requires        []string    `json:"requires"`
	PublicPorts     []util.Port `json:"publicports"`
	Description     string      `json:"description"`
	PlatformID      string      `json:"platformid"`
	PlatformType    string      `json:"platformtype"`
	PersistancePath string      `json:"persistancepath"`
	Capabilities    []string    `json:"capabilities"`
}

type localInstaller struct {
	Installer
	Metadata struct {
		InstallerMetadata
		Capabilities []map[string]string `json:"capabilities"`
	} `json:"versions"`
}

func (li localInstaller) convert(as *AppStore) *Installer {
	li.Installer.Metadata = &InstallerMetadata{}
	for _, cap := range li.Metadata.Capabilities {
		if capName, ok := cap["Name"]; ok {
			if _, err := as.cm.GetByName(capName); err == nil {
				li.Metadata.InstallerMetadata.Capabilities = append(li.Metadata.InstallerMetadata.Capabilities, capName)
			}
		}
	}
	li.Installer.Metadata = &li.Metadata.InstallerMetadata
	li.Installer.parent = as
	return &li.Installer
}

type localInstallers map[string]localInstaller

func (li localInstallers) convert(as *AppStore) map[string]*Installer {
	installers := map[string]*Installer{}
	for id, inst := range li {
		installers[id] = inst.convert(as)
	}
	return installers
}

// Installer represents an application installer
type Installer struct {
	parent   *AppStore
	manifest regmanifest.Manifest
	ref      ref.Ref

	Name          string             `json:"name"`
	ID            string             `json:"id"`
	Version       string             `json:"version"`
	Metadata      *InstallerMetadata `json:"metadata"`
	Architectures []string           `json:"architectures"`
}

// AppStore manages and downloads application installers
type AppStore struct {
	rp ImageManager
	tm *task.Manager
	cm *capability.Manager
}

// CreateAppStore creates and returns an app store instance
func CreateAppStore(rp ImageManager, tm *task.Manager, cm *capability.Manager) *AppStore {
	if tm == nil || cm == nil {
		log.Panic("Failed to create AppStore: none of the inputs can be nil")
	}

	return &AppStore{rp: rp, tm: tm, cm: cm}
}

// func validateInstallerCapabilities(cm *capability.Manager, capstring string) []string {
// 	caps := []string{}
// 	for _, capname := range strings.Split(capstring, ",") {
// 		_, err := cm.GetByName(capname)
// 		if err != nil {
// 			log.Error(err)
// 		} else {
// 			caps = append(caps, capname)
// 		}
// 	}
// 	return caps
// }

// func parsePublicPorts(publicports string) []util.Port {
// 	ports := []util.Port{}
// 	for _, portstr := range strings.Split(publicports, ",") {
// 		portParts := strings.Split(portstr, "/")
// 		if len(portParts) != 2 {
// 			log.Errorf("Error parsing installer port string %s", portstr)
// 			continue
// 		}
// 		portNr, err := strconv.Atoi(portParts[0])
// 		if err != nil {
// 			log.Errorf("Error parsing installer port string %s", portstr)
// 			continue
// 		}
// 		if portNr < 1 || portNr > 0xffff {
// 			log.Errorf("Installer port is out of range %s (valid range is 1-65535)", portstr)
// 			continue
// 		}
// 		port := util.Port{Nr: portNr}
// 		if strings.ToUpper(portParts[1]) == string(util.TCP) {
// 			port.Type = util.TCP
// 		} else if strings.ToUpper(portParts[1]) == string(util.UDP) {
// 			port.Type = util.UDP
// 		} else {
// 			log.Errorf("Invalid protocol(%s) for port(%s)", portParts[1], portParts[0])
// 			continue
// 		}
// 		ports = append(ports, port)
// 	}
// 	return ports
// }

// // parseMetadata parses the image metadata from the image labels
// func parseMetadata(cm *capability.Manager, labels map[string]string) (InstallerMetadata, error) {
// 	r := regexp.MustCompile("(^protos.installer.metadata.)(\\w+)")
// 	metadata := InstallerMetadata{}
// 	for label, value := range labels {
// 		labelParts := r.FindStringSubmatch(label)
// 		if len(labelParts) == 3 {
// 			switch labelParts[2] {
// 			case "capabilities":
// 				metadata.Capabilities = validateInstallerCapabilities(cm, value)
// 			case "params":
// 				metadata.Params = strings.Split(value, ",")
// 			case "provides":
// 				metadata.Provides = strings.Split(value, ",")
// 			case "requires":
// 				metadata.Requires = strings.Split(value, ",")
// 			case "publicports":
// 				metadata.PublicPorts = parsePublicPorts(value)
// 			case "description":
// 				metadata.Description = value
// 			}
// 		}

// 	}
// 	if metadata.Description == "" {
// 		return metadata, errors.New("installer metadata field 'description' is mandatory")
// 	}
// 	return metadata, nil
// }

//
// Installer methods
//

// GetName returns the name of the installer
func (inst *Installer) GetName() string {
	return inst.Name
}

// GetMetadata returns the metadata for a specific installer version
func (inst *Installer) GetMetadata() (InstallerMetadata, error) {
	if inst.Metadata != nil {
		return *inst.Metadata, nil
	}
	return InstallerMetadata{}, fmt.Errorf("installer '%s' has no metadata", inst.ref.CommonName())
}

// Download downloads an installer from the application store
func (inst *Installer) Pull() error {

	available, err := inst.IsPlatformImageAvailable()
	if err != nil {
		return fmt.Errorf("could not pull installer '%s': %w", inst.ref.CommonName(), err)
	}
	if !available {
		log.Debugf("Downloading installer '%s'", inst.ref.CommonName())
		err = inst.parent.getPlatform().PullImage(inst.ref.CommonName())
		if err != nil {
			return fmt.Errorf("failed to download installer '%s': %w", inst.ref.CommonName(), err)
		}
	} else {
		log.Debugf("Installer '%s' found locally", inst.ref.CommonName())
	}

	return nil
}

// Download downloads an installer from the application store
func (inst *Installer) Download(dt DownloadTask) error {
	log.Infof("Downloading installer '%s'", inst.ref.CommonName())
	err := inst.parent.getPlatform().PullImage(inst.ref.CommonName())
	if err != nil {
		return errors.Wrapf(err, "Failed to download installer '%s'", inst.ref.CommonName())
	}
	return nil
}

// DownloadAsync triggers an async installer download, returns a generic task
func (inst *Installer) DownloadAsync(version string, appID string) *task.Base {
	return inst.parent.getTaskManager().New("Download application installer", &DownloadTask{Inst: *inst, Version: version, AppID: appID})
}

// IsPlatformImageAvailable checks if the associated docker image for an installer is available locally
func (inst *Installer) IsPlatformImageAvailable() (bool, error) {
	exists, err := inst.parent.getPlatform().ImageExistsLocally(inst.ref.CommonName())
	if err != nil {
		return false, errors.Wrapf(err, "Failed to check local image for installer %s(%s)", inst.Name, inst.ID)
	}
	return exists, nil
}

// Remove Installer removes an installer image
func (inst *Installer) Remove() error {
	log.Info("Removing installer ", inst.Name, "[", inst.ID, "]")

	err := inst.parent.getPlatform().RemoveImage(inst.ref.CommonName())
	if err != nil {
		return errors.Wrapf(err, "Failed to remove install %s(%s)", inst.Name, inst.ID)
	}

	return nil
}

func (inst *Installer) GetDescription() string {
	if inst.Metadata != nil {
		return inst.Metadata.Description
	}
	return "n/a"
}

func (inst *Installer) GetRequires() []string {
	if inst.Metadata != nil {
		return inst.Metadata.Requires
	}
	return []string{}
}

func (inst *Installer) GetProvides() []string {
	if inst.Metadata != nil {
		return inst.Metadata.Provides
	}
	return []string{}
}

func (inst *Installer) GetParams() []string {
	if inst.Metadata != nil {
		return inst.Metadata.Params
	}
	return []string{}
}

func (inst *Installer) GetCapabilities() []string {
	if inst.Metadata != nil {
		return inst.Metadata.Capabilities
	}
	return []string{}
}

func (inst *Installer) SupportsArchitecture(architecture string) bool {
	for _, arch := range inst.Architectures {
		if arch == architecture {
			return true
		}
	}
	return false
}

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
func (as *AppStore) GetInstallers() (map[string]*Installer, error) {
	installers := map[string]*Installer{}
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
func (as *AppStore) GetInstaller(imageRef string) (*Installer, error) {
	installer := &Installer{parent: as}
	var err error

	rc := regclient.New()
	installer.ref, err = ref.New(imageRef)
	if err != nil {
		return nil, fmt.Errorf("could not parse ref for image '%s': %w", imageRef, err)
	}

	if installer.ref.Tag == "latest" {
		return nil, fmt.Errorf("use of version 'latest' not allowed")
	}

	manifest, err := rc.ManifestGet(context.Background(), installer.ref)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve manifest for image '%s': %w", imageRef, err)
	}

	installer.ID = manifest.GetDigest().String()
	installer.manifest = manifest
	installer.Name = installer.ref.CommonName()
	installer.Version = installer.ref.Tag

	if manifest.IsList() {
		platforms, err := manifest.GetPlatformList()
		if err != nil {
			return nil, fmt.Errorf("could not retrieve platforms for image '%s': %w", imageRef, err)
		}

		for _, platform := range platforms {
			installer.Architectures = append(installer.Architectures, platform.Architecture)
		}
	} else {
		installer.Architectures = append(installer.Architectures, "amd64")
	}

	// client := getHTTPClient()
	// url := fmt.Sprintf("%s/api/v1/installers/%s", gconfig.AppStoreURL, id)
	// log.Debugf("Querying app store at '%s'", url)
	// resp, err := client.Get(url)
	// if err != nil {
	// 	return nil, errors.Wrapf(err, "Could not retrieve installer '%s' from app store", id)
	// }

	// if err := util.HTTPBadResponse(resp); err != nil {
	// 	return nil, errors.Wrapf(err, "Could not retrieve installer '%s' from app store", id)
	// }

	// err = json.NewDecoder(resp.Body).Decode(&localInstaller)
	// defer resp.Body.Close()
	// if err != nil {
	// 	return nil, errors.Wrapf(err, "Could not retrieve installer '%s' from app store. Decoding error", id)
	// }

	return installer, nil
}

// Search takes a map of search terms and performs a search on the app store
func (as *AppStore) Search(key string, value string) (map[string]*Installer, error) {
	installers := map[string]*Installer{}
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

func (as *AppStore) getPlatform() ImageManager {
	return as.rp
}

func (as *AppStore) getTaskManager() *task.Manager {
	return as.tm
}
