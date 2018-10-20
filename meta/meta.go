package meta

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/protosio/protos/resource"

	"github.com/protosio/protos/database"
	"github.com/protosio/protos/util"
	"github.com/tidwall/gjson"
)

var log = util.GetLogger("meta")

type meta struct {
	ID        string
	Domain    string
	PublicIP  string
	AdminUser string
	Resources []string
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

func findPublicIP() (string, error) {
	log.Info("Finding the public IP of this Protos instance")
	resp, err := http.Get("https://api.ipify.org?format=json")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	bodyJSON, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return gjson.GetBytes(bodyJSON, "ip").Str, nil
}

// SetDomain sets the instance domain name
func SetDomain(domainName string) {
	log.Debugf("Setting instance domain name to %s", domainName)
	metaRoot.Domain = domainName
	err := database.Save(&metaRoot)
	if err != nil {
		log.Fatalf("Failed to write the metaroot domain to database: %s", err.Error())
	}
}

// setPublicIP sets the public ip of the instance
func setPublicIP() {
	ip, err := findPublicIP()
	if err != nil {
		log.Fatalf("Could not find instance public ip: %s", err.Error())
	}
	log.Debugf("Setting instance IP address to %s", ip)
	metaRoot.PublicIP = ip
	err = database.Save(&metaRoot)
	if err != nil {
		log.Fatalf("Failed to write the metaroot IP to database: %s", err.Error())
	}
}

// Setup reads the domain and other information on first run and save this information to the database
func Setup() {
	log.Debug("Creating metaroot database entry")
	metaRoot = meta{ID: "metaroot"}
	err := database.Save(&metaRoot)
	if err != nil {
		log.Fatalf("Failed to write the metaroot to database: %s", err.Error())
	}
	setPublicIP()
}

// SetAdminUser takes a username that gets saved as the instance admin user
func SetAdminUser(username string) error {
	log.Debugf("Setting admin user to [%s]", username)
	metaRoot.AdminUser = username
	err := database.Save(&metaRoot)
	if err != nil {
		return err
	}
	return nil
}

// GetAdminUser returns the username of the admin user
func GetAdminUser() string {
	return metaRoot.AdminUser
}

// InitCheck checks the instance information at program startup
func InitCheck() {
	log.Debug("Reading instance information from database")
	err := database.One("ID", "metaroot", &metaRoot)
	if err != nil {
		log.Fatalf("Can't load instance information from database: %s", err.Error())
	}

	if metaRoot.PublicIP == "" {
		log.Fatalf("Instance public ip is empty. Please run init")
	}

	if metaRoot.Domain == "" {
		log.Fatal("Instance domain is empty. Please run init")
	}

	if metaRoot.AdminUser == "" {
		log.Fatal("Instance admin user is empty. Please run init")
	}

	log.Infof("Running under domain %s using public IP %s", metaRoot.Domain, metaRoot.PublicIP)
	if len(metaRoot.Resources) < 2 {
		log.Fatal("DNS and TLS certificate resources have not been created. Please run init")
	}
}

// GetDomain returns the domain name used in this Protos instance
func GetDomain() string {
	return metaRoot.Domain
}

// GetPublicIP returns the public IP of the Protos instance
func GetPublicIP() string {
	return metaRoot.PublicIP
}

// GetTLSCertificate returns the TLS certificate resource owned by the instance
func GetTLSCertificate() resource.Resource {
	for _, rscid := range metaRoot.Resources {
		rsc, err := resource.Get(rscid)
		if err != nil {
			log.Errorf("Could not find protos resource: %s", err.Error())
			continue
		}
		if rsc.Type == resource.RType("certificate") {
			return *rsc
		}
	}
	return resource.Resource{}
}

// CleanProtosResources removes the MX record resource owned by the instance, created during the init process
func CleanProtosResources() error {
	for _, rscid := range metaRoot.Resources {
		rsc, err := resource.Get(rscid)
		if err != nil {
			log.Errorf("Could not find protos resource: %s", err.Error())
			continue
		}
		if rsc.Type == resource.RType("dns") {
			val := rsc.Value.(*resource.DNSResource)
			if val.Type == "MX" {
				err = rsc.Delete()
				if err != nil {
					return errors.Wrap(err, "Could not clean Protos resources")
				}
				return nil
			}
		}
	}
	return errors.New("Could not clean Protos resources: MX DNS record not found")
}

// CreateProtosResources creates the DNS and TLS certificate for the Protos dashboard
func CreateProtosResources() (map[string]*resource.Resource, error) {
	resources := map[string]*resource.Resource{}
	protosDNS := resource.DNSResource{
		Host:  "protos",
		Value: GetPublicIP(),
		Type:  "A",
		TTL:   300,
	}
	dnsrsc, err := resource.Create(resource.DNS, &protosDNS)
	if err != nil {
		if strings.Contains(err.Error(), "already registered") == false {
			return resources, errors.Wrap(err, "Failed to create Protos resources")
		}
	}
	metaRoot.Resources = append(metaRoot.Resources, dnsrsc.ID)
	protosMX := resource.DNSResource{
		Host:  "@",
		Value: "protos." + GetDomain(),
		Type:  "MX",
		TTL:   300,
	}
	mxrsc, err := resource.Create(resource.DNS, &protosMX)
	if err != nil {
		if strings.Contains(err.Error(), "already registered") == false {
			return resources, errors.Wrap(err, "Failed to create Protos resources")
		}
	}
	metaRoot.Resources = append(metaRoot.Resources, mxrsc.ID)

	protosCert := resource.CertificateResource{
		Domains: []string{"protos"},
	}
	certrsc, err := resource.Create(resource.Certificate, &protosCert)
	if err != nil {
		if strings.Contains(err.Error(), "already registered") == false {
			return resources, errors.Wrap(err, "Failed to create Protos resources")
		}
	}
	metaRoot.Resources = append(metaRoot.Resources, certrsc.ID)

	err = database.Save(&metaRoot)
	if err != nil {
		return resources, errors.Wrap(err, "Failed to create Protos resources")
	}
	resources[dnsrsc.ID] = dnsrsc
	resources[certrsc.ID] = certrsc
	resources[mxrsc.ID] = mxrsc

	return resources, nil
}
