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
	P2PPort         int
	Runtime         string
	RuntimeEndpoint string
	StaticAssets    string
	InternalDomain  string
	ProcsQuit       sync.Map
	ExternalDNS     string // format: <ip>:<port>
	Version         *semver.Version
}

var config = Config{
	WorkDir:         "/var/lib/protos",
	P2PPort:         10500,
	Runtime:         "containerd",
	RuntimeEndpoint: "/run/containerd/containerd.sock",
	InternalDomain:  "protos.internal",
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
