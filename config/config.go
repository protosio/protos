package config

import (
	"io/ioutil"
	"path/filepath"

	"github.com/nustiueudinastea/protos/util"

	"gopkg.in/yaml.v2"
)

// Config is the main configuration struct
type Config struct {
	WorkDir        string
	AppsPath       string
	Port           int
	DockerEndpoint string
	StaticAssets   string
	Secret         []byte
}

var config = Config{}

// Gconfig maintains a global view of the application configuration parameters.
var Gconfig = &config
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
