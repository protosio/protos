package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/util"

	"gopkg.in/yaml.v3"
)

const (
	ReleasesURL  = "https://releases.protos.io/releases.json"
	LocalDNSPort = 10053
	DBName       = "protos"
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
	ExternalDNS     string // format: <ip>:<port>
	Capabilities    []string
	Version         *semver.Version
}

// Gconfig maintains a global view of the application configuration parameters.
// var gconfig = &config
var log = util.GetLogger("config")

// Load reads the configuration from a file and maps it to the config struct
func Load(configFile string, version *semver.Version) Config {
	log.Info("Reading main config [", configFile, "]")
	filename, _ := filepath.Abs(configFile)
	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			log.Info("No config file found, using default config values")
		} else {
			log.Fatal(errors.Wrap(err, "Failed to load protos config file"))
		}
	}

	err = yaml.Unmarshal(yamlFile, config)
	if err != nil {
		log.Fatal(err)
	}
	config.Version = version
	return *config
}

// Get returns a pointer to the global config structure
func New(workdir string, version *semver.Version) Config {
	if workdir != "" {
		config.WorkDir = workdir
	}

	config.Version = version

	return *config
}

func Get() *Config {
	return config
}
