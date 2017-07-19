package daemon

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/crypto/bcrypt"
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

	var err error
	err = openDatabase()
	if err != nil {
		log.Fatal(err)
	}
	defer shutDown()

	username, password := readCredentials()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	log.Infof("Setting up database")
	err = Gconfig.Db.Update(func(tx *bolt.Tx) error {

		buckets := [3]string{"installer", "app", "user"}

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

	log.Infof("Writing username %s to database", username)
	err = Gconfig.Db.Update(func(tx *bolt.Tx) error {
		userBucket := tx.Bucket([]byte("user"))

		err = userBucket.Put([]byte("username"), []byte(username))
		if err != nil {
			return err
		}

		err = userBucket.Put([]byte("password"), hashedPassword)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

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

	connectDocker()

}

func shutDown() {
	log.Info("Shuting down...")
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
