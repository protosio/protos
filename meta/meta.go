package meta

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/nustiueudinastea/protos/database"
	"github.com/nustiueudinastea/protos/util"
)

var log = util.Log

type meta struct {
	ID     string
	Domain string
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

// Setup reads the domain and other information on first run and save this information to the database
func Setup() {
	domainName := readDomain()
	metaRoot = meta{ID: "metaroot", Domain: domainName}
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

	if metaRoot.Domain == "" {
		log.Fatal("Instance domain is empty. Please run init")
	} else {
		log.Infof("Running under domain %s", metaRoot.Domain)
	}
}

// GetDomain returns the domain name used in this Protos instance
func GetDomain() string {
	return metaRoot.Domain
}
