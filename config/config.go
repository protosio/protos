package config

import (
	"io/ioutil"
	"path/filepath"

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
}

var config = Config{InitMode: true, AppStoreURL: "https://apps.protos.io/", AppStoreHost: "apps.protos.io"}

// Gconfig maintains a global view of the application configuration parameters.
// var gconfig = &config
var log = util.Log

// Load reads the configuration from a file and maps it to the config struct
func Load(configFile string) {
	log.Info("Reading main config [", configFile, "]")
	filename, _ := filepath.Abs(configFile)
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatal(err)
	}
}

// Get returns a pointer to the global config structure
func Get() *Config {
	return &config
}
