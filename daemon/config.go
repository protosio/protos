package daemon

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/boltdb/bolt"
	"github.com/fsouza/go-dockerclient"
	"gopkg.in/yaml.v2"
)

// Config is the main configuration struct
type Config struct {
	WorkDir        string
	AppsPath       string
	Port           int
	DockerEndpoint string
	DockerClient   *docker.Client
	StaticAssets   string
	Db             *bolt.DB
}

// Gconfig maintains a global view of the application configuration parameters.
var Gconfig Config

func readCredentials() (string, string) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Username: ")
	username, _ := reader.ReadString('\n')

	fmt.Print("Enter Password: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatal(err)
	}
	password := string(bytePassword)

	return strings.TrimSpace(username), strings.TrimSpace(password)
}

// Initialize creates an initial detabase and populates the credentials.
func Initialize() {

	log.Info("Initializing...")
	var userBucket = []byte("user")

	// create the workdir if it does not exist
	if _, err := os.Stat(Gconfig.WorkDir); err != nil {
		if os.IsNotExist(err) {
			log.Info("Creating working directory [", Gconfig.WorkDir, "]")
			err = os.Mkdir(Gconfig.WorkDir, 0755)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal(err)
		}
	}

	// open the database
	var err error
	Gconfig.Db, err = bolt.Open(path.Join(Gconfig.WorkDir, "protos.db"), 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer shutDown()

	username, password := readCredentials()

	log.Infof("Writing username %s to database", username)
	err = Gconfig.Db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(userBucket)
		if err != nil {
			return err
		}

		err = bucket.Put([]byte("username"), []byte(username))
		if err != nil {
			return err
		}

		err = bucket.Put([]byte("password"), []byte(password))
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

}

func startup() {
	log.Info("Starting up...")
	var err error
	Gconfig.Db, err = bolt.Open(path.Join(Gconfig.WorkDir, "protos.db"), 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func shutDown() {
	log.Info("Shuting down...")
	Gconfig.Db.Close()
}

// LoadCfg reads the configuration from a file and maps it to the config struct
func LoadCfg(configFile string) Config {
	log.Info("Reading main config [", configFile, "]")
	filename, _ := filepath.Abs(configFile)
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		log.Fatal(err)
	}

	var config Config

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatal(err)
	}

	Gconfig = config

	return Gconfig
}

func connectDocker() error {
	log.Info("Connecting to the docker daemon")
	client, err := docker.NewClient(Gconfig.DockerEndpoint)
	if err != nil {
		log.Fatal(err)
	}
	Gconfig.DockerClient = client
	return nil
}
