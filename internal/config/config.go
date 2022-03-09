package config

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/util"

	"gopkg.in/yaml.v2"
)

// Config is the main configuration struct
type Config struct {
	WorkDir         string
	AppsPath        string
	HTTPport        int
	HTTPSport       int
	Runtime         string
	RuntimeEndpoint string
	InContainer     bool
	StaticAssets    string
	InitMode        bool
	InternalDomain  string
	DevMode         bool
	AppStoreURL     string
	AppStoreName    string
	AppStoreHost    string
	ProcsQuit       sync.Map
	ExternalDNS     string // format: <ip>:<port>
	Version         *semver.Version
}

var config = Config{
	WorkDir:         "/var/lib/protos",
	HTTPport:        8080,
	HTTPSport:       8443,
	Runtime:         "containerd",
	RuntimeEndpoint: "/run/containerd/containerd.sock",
	InContainer:     false,
	InitMode:        false,
	DevMode:         false,
	InternalDomain:  "protos.internal",
	AppStoreURL:     "https://apps.protos.io",
	AppStoreName:    "protos.io",
	AppStoreHost:    "apps.protos.io",
	ExternalDNS:     "8.8.8.8:53",
	ProcsQuit:       sync.Map{},
}

// Gconfig maintains a global view of the application configuration parameters.
// var gconfig = &config
var log = util.GetLogger("config")

// Load reads the configuration from a file and maps it to the config struct
func Load(configFile string, version *semver.Version) *Config {
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
	return &config
}

// Get returns a pointer to the global config structure
func Get() *Config {
	return &config
}
