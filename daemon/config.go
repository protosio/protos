package daemon

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"protos/util"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/boltdb/bolt"
	docker "github.com/docker/docker/client"

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
	Secret         []byte
	Db             *bolt.DB
}

// Gconfig maintains a global view of the application configuration parameters.
var Gconfig Config
var log = util.Log

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

	err := openDatabase()
	if err != nil {
		log.Fatal(err)
	}
	defer shutDown()

	log.Info("Setting up database")
	err = Gconfig.Db.Update(func(tx *bolt.Tx) error {

		buckets := [4]string{"installer", "app", "user"}

		for _, bname := range buckets {
			_, err := tx.CreateBucketIfNotExists([]byte(bname))
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	// Creating initial user (admin)
	username, clearpassword := readCredentials()
	user, err := CreateUser(username, clearpassword, true)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("User %s has been created.", user.Username)

}

// StartUp triggers a sequence of steps required to start the application
func StartUp(configFile string) {
	log.Info("Starting up...")
	var err error

	err = LoadCfg(configFile)
	if err != nil {
		log.Fatal(err)
	}

	err = openDatabase()
	if err != nil {
		log.Fatal(err)
	}

	// Generate secret key used for JWT
	log.Info("Generating secret for JWT")
	Gconfig.Secret, err = util.GenerateRandomBytes(32)

	connectDocker()

}

func shutDown() {
	log.Info("Closing database and shuting down...")
	Gconfig.Db.Close()
}

// LoadCfg reads the configuration from a file and maps it to the config struct
func LoadCfg(configFile string) error {
	log.Info("Reading main config [", configFile, "]")
	filename, _ := filepath.Abs(configFile)
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	var config Config

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return err
	}

	Gconfig = config

	return nil
}

func connectDocker() error {
	log.Info("Connecting to the docker daemon")

	// Gconfig.DockerEndpoint
	client, err := docker.NewEnvClient()
	if err != nil {
		panic(err)
	}
	Gconfig.DockerClient = client
	return nil
}
