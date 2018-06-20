package meta

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/nustiueudinastea/protos/database"
	"github.com/nustiueudinastea/protos/util"
	"github.com/tidwall/gjson"
)

var log = util.Log

type meta struct {
	ID       string
	Domain   string
	PublicIP string
}

var metaRoot meta

// readDomain reads the Protos instance domain interactively
func readDomain() string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter domain: ")
	domain, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	return strings.TrimSpace(domain)
}

func findPublicIP() string {
	log.Info("Finding the public IP of this Protos instance")
	resp, err := http.Get("https://api.ipify.org?format=json")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	bodyJSON, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return gjson.GetBytes(bodyJSON, "ip").Str
}

// Setup reads the domain and other information on first run and save this information to the database
func Setup() {
	domainName := readDomain()
	ip := findPublicIP()
	log.Debugf("Instance running using domain %s and IP %s", domainName, ip)
	metaRoot = meta{ID: "metaroot", Domain: domainName, PublicIP: ip}
	err := database.Save(&metaRoot)
	if err != nil {
		log.Fatal(err)
	}
}

// Initialize loads the instance information at program startup
func Initialize() {
	log.Debug("Reading instance information from database")
	err := database.One("ID", "metaroot", &metaRoot)
	if err != nil {
		log.Error(err)
		log.Fatal("Can't load instance information from database")
	}

	publicIP := findPublicIP()
	if metaRoot.PublicIP != publicIP {
		metaRoot.PublicIP = findPublicIP()
		err = database.Save(&metaRoot)
		if err != nil {
			log.Fatal(err)
		}
	}

	if metaRoot.Domain == "" {
		log.Fatal("Instance domain is empty. Please run init")
	}
	log.Infof("Running under domain %s using public IP %s", metaRoot.Domain, metaRoot.PublicIP)
}

// GetDomain returns the domain name used in this Protos instance
func GetDomain() string {
	return metaRoot.Domain
}

// GetPublicIP returns the public IP of the Protos instance
func GetPublicIP() string {
	return metaRoot.PublicIP
}
