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
	DevMode         bool
	AppStoreURL     string
	AppStoreHost    string
	ProcsQuit       sync.Map
	InternalIP      string
	Version         *semver.Version
	WSPublish       chan interface{}
}

var config = Config{
	WorkDir:         "/opt/protos/",
	HTTPport:        8080,
	HTTPSport:       8443,
	Runtime:         "docker",
	RuntimeEndpoint: "unix:///var/run/docker.sock",
	InContainer:     true,
	InitMode:        false,
	DevMode:         false,
	AppStoreURL:     "https://apps.protos.io",
	AppStoreHost:    "apps.protos.io",
	ProcsQuit:       sync.Map{},
	WSPublish:       make(chan interface{}, 100),
}

// GetWSPublishChannel returns the channel that can be used to publish messages to the available websockets
func (cfg *Config) GetWSPublishChannel() chan interface{} {
	return cfg.WSPublish
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

//
// Config methods
//

// SetInternalIP sets the internal ip that all apps use to talk to the internal API
func (cfg *Config) SetInternalIP(ip string) {
	cfg.InternalIP = ip
}
