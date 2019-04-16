package meta

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/protosio/protos/config"
	"github.com/protosio/protos/core"

	"github.com/protosio/protos/database"
	"github.com/protosio/protos/util"
	"github.com/tidwall/gjson"
)

var log = util.GetLogger("meta")
var gconfig = config.Get()

type meta struct {
	ID                 string
	Domain             string
	DashboardSubdomain string
	PublicIP           string
	AdminUser          string
	Resources          []string
	rm                 core.ResourceManager
}

type dnsResource interface {
	IsType(string) bool
}

// Setup reads the domain and other information on first run and save this information to the database
func Setup(rm core.ResourceManager) core.Meta {
	metaRoot := meta{}
	log.Debug("Reading instance information from database")
	err := database.One("ID", "metaroot", &metaRoot)
	if err != nil {
		log.Debug("Creating metaroot database entry")
		metaRoot = meta{
			ID:                 "metaroot",
			DashboardSubdomain: "protos",
		}
	} else {
		metaRoot.ID = "metaroot"
		metaRoot.DashboardSubdomain = "protos"
	}

	err = database.Save(&metaRoot)
	if err != nil {
		log.Fatalf("Failed to write the metaroot to database: %s", err.Error())
	}
	metaRoot.setPublicIP()
	return &metaRoot
}

func (m *meta) save() {
	err := database.Save(m)
	if err != nil {
		log.Fatalf("Failed to write the metaroot domain to database: %s", err.Error())
	}
}

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
func (m *meta) SetDomain(domainName string) {
	log.Debugf("Setting instance domain name to %s", domainName)
	m.Domain = domainName
	m.save()
}

// setPublicIP sets the public ip of the instance
func (m *meta) setPublicIP() {
	ip, err := findPublicIP()
	if err != nil {
		log.Fatalf("Could not find instance public ip: %s", err.Error())
	}
	log.Debugf("Setting instance IP address to %s", ip)
	m.PublicIP = ip
	m.save()

}

// SetAdminUser takes a username that gets saved as the instance admin user
func (m *meta) SetAdminUser(username string) {
	log.Debugf("Setting admin user to [%s]", username)
	m.AdminUser = username
	m.save()
}

// GetAdminUser returns the username of the admin user
func (m *meta) GetAdminUser() string {
	return m.AdminUser
}

// InitCheck checks the instance information at program startup
func (m *meta) InitCheck() {

	if m.PublicIP == "" {
		log.Fatalf("Instance public ip is empty. Please run init")
	}

	if m.Domain == "" {
		log.Fatal("Instance domain is empty. Please run init")
	}

	if m.AdminUser == "" {
		log.Fatal("Instance admin user is empty. Please run init")
	}

	log.Infof("Running under domain %s using public IP %s", m.Domain, m.PublicIP)
	if len(m.Resources) < 2 {
		log.Fatal("DNS and TLS certificate resources have not been created. Please run init")
	}
}

// GetDomain returns the domain name used in this Protos instance
func (m *meta) GetDomain() string {
	return m.Domain
}

// GetPublicIP returns the public IP of the Protos instance
func (m *meta) GetPublicIP() string {
	return m.PublicIP
}

// GetTLSCertificate returns the TLS certificate resource owned by the instance
func (m *meta) GetTLSCertificate() core.Resource {

	for _, rscid := range m.Resources {
		rsc, err := m.rm.Get(rscid)
		if err != nil {
			log.Errorf("Could not find protos resource: %s", err.Error())
			continue
		}
		if rsc.GetType() == core.RType("certificate") {
			return rsc
		}
	}
	return nil
}

// CleanProtosResources removes the MX record resource owned by the instance, created during the init process
func (m *meta) CleanProtosResources() error {
	for i, rscid := range m.Resources {
		rsc, err := m.rm.Get(rscid)
		if err != nil {
			log.Errorf("Could not find protos resource: %s", err.Error())
			continue
		}
		if rsc.GetType() == core.DNS {
			val := rsc.GetValue().(dnsResource)
			if val.IsType("MX") {
				err = m.rm.Delete(rscid)
				if err != nil {
					return errors.Wrap(err, "Could not clean Protos resources")
				}
				m.Resources = util.RemoveStringFromSlice(m.Resources, i)
				m.save()
				return nil
			}
		}
	}
	return errors.New("Could not clean Protos resources: MX DNS record not found")
}

// GetDashboardDomain returns the full domain through which the dashboard can be accessed
func (m *meta) GetDashboardDomain() string {
	dashboardDomain := m.DashboardSubdomain + "." + m.GetDomain()
	if gconfig.HTTPSport != 443 {
		dashboardDomain = fmt.Sprintf("%s:%d", dashboardDomain, gconfig.HTTPSport)
	}
	return dashboardDomain
}

// CreateProtosResources creates the DNS and TLS certificate for the Protos dashboard
func (m *meta) CreateProtosResources() (map[string]core.Resource, error) {
	resources := map[string]core.Resource{}
	rc := m.rm.(core.ResourceCreator)

	// creating the protos subdomain for the dashboard
	dnsrsc, err := rc.CreateDNS("protos", "protos", "A", m.PublicIP, 300)
	if err != nil {
		if strings.Contains(err.Error(), "already registered") == false {
			return resources, errors.Wrap(err, "Failed to create Protos resources")
		}
	}
	// creating the bogus MX record, which is checked by LetsEncrypt before creating a certificate
	mxrsc, err := rc.CreateDNS("protos", "@", "MX", "protos."+m.Domain, 300)
	if err != nil {
		if strings.Contains(err.Error(), "already registered") == false {
			return resources, errors.Wrap(err, "Failed to create Protos resources")
		}
	}
	// creating a TLS certificate for the protos subdomain
	certrsc, err := rc.CreateCert("protos", []string{"protos"})
	if err != nil {
		if strings.Contains(err.Error(), "already registered") == false {
			return resources, errors.Wrap(err, "Failed to create Protos resources")
		}
	}
	m.Resources = append(m.Resources, dnsrsc.GetID(), mxrsc.GetID(), certrsc.GetID())
	m.save()

	resources[dnsrsc.GetID()] = dnsrsc
	resources[certrsc.GetID()] = certrsc
	resources[mxrsc.GetID()] = mxrsc

	return resources, nil
}

// GetProtosResources returns the resources owned by Protos
func (m *meta) GetProtosResources() map[string]core.Resource {
	resources := map[string]core.Resource{}
	for _, rscid := range m.Resources {
		rsc, err := m.rm.Get(rscid)
		if err != nil {
			log.Errorf("Could not find protos resource: %s", err.Error())
			continue
		}
		resources[rscid] = rsc

	}
	return resources
}

// GetService returns the protos dashboard service
func (m *meta) GetService() util.Service {
	ports := []util.Port{}
	ports = append(ports, util.Port{Nr: gconfig.HTTPport, Type: util.TCP})
	ports = append(ports, util.Port{Nr: gconfig.HTTPSport, Type: util.TCP})
	protosService := util.Service{
		Name:   "protos dashboard",
		Domain: m.DashboardSubdomain + "." + m.GetDomain(),
		IP:     m.GetPublicIP(),
		Ports:  ports,
		Status: util.StatusActive,
	}
	return protosService
}
