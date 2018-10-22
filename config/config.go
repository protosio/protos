package config

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	"github.com/protosio/protos/util"

	"gopkg.in/yaml.v2"
)

// Config is the main configuration struct
type Config struct {
	WorkDir        string
	AppsPath       string
	HTTPport       int
	HTTPSport      int
	DockerEndpoint string
	StaticAssets   string
	Secret         []byte
	InitMode       bool
	AppStoreURL    string
	AppStoreHost   string
	ProcsQuit      map[string]chan bool
	InternalIP     string
	Version        *semver.Version
}

var config = Config{
	WorkDir:        "/opt/protos/",
	HTTPport:       8080,
	HTTPSport:      8443,
	DockerEndpoint: "unix:///var/run/docker.sock",
	InitMode:       false,
	AppStoreURL:    "https://apps.protos.io",
	AppStoreHost:   "apps.protos.io",
	ProcsQuit:      make(map[string]chan bool)}

// Gconfig maintains a global view of the application configuration parameters.
// var gconfig = &config
var log = util.GetLogger("config")

// Load reads the configuration from a file and maps it to the config struct
func Load(configFile string, version *semver.Version) {
	log.Info("Reading main config [", configFile, "]")
	filename, _ := filepath.Abs(configFile)
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			log.Info("No config file found, using default config values")
		} else {
			log.Fatal(errors.Wrap(err, "Failed to load protos config file"))
		}
	}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatal(err)
	}
	config.Version = version
}

// Get returns a pointer to the global config structure
func Get() *Config {
	return &config
}
