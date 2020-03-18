package meta

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/core"
	"github.com/tidwall/gjson"

	"github.com/pkg/errors"

	"github.com/protosio/protos/internal/util"
)

var log = util.GetLogger("meta")
var gconfig = config.Get()

// Meta contains information about the Protos instance
type Meta struct {
	ID                 string
	Domain             string
	DashboardSubdomain string
	PublicIP           string
	AdminUser          string
	Resources          []string
	version            string
	rm                 core.ResourceManager
	db                 core.DB
}

type dnsResource interface {
	IsType(string) bool
	UpdateValueAndTTL(value string, ttl int)
}

// Setup reads the domain and other information on first run and save this information to the database
func Setup(rm core.ResourceManager, db core.DB, version string) *Meta {
	if rm == nil || db == nil {
		log.Panic("Failed to setup meta package: none of the inputs can be nil")
	}

	metaRoot := Meta{}
	log.Debug("Reading instance information from database")
	err := db.One("ID", "metaroot", &metaRoot)
	if err != nil {
		log.Debug("Creating metaroot database entry")
		metaRoot = Meta{
			ID:                 "metaroot",
			DashboardSubdomain: "protos",
		}
	} else {
		metaRoot.ID = "metaroot"
		metaRoot.DashboardSubdomain = "protos"
	}

	metaRoot.db = db
	metaRoot.rm = rm
	metaRoot.version = version
	err = db.Save(&metaRoot)
	if err != nil {
		log.Fatalf("Failed to write the metaroot to database: %s", err.Error())
	}
	metaRoot.setPublicIP()
	return &metaRoot
}

func (m *Meta) save() {
	err := m.db.Save(m)
	if err != nil {
		log.Fatalf("Failed to write the metaroot domain to database: %s", err.Error())
	}
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
func (m *Meta) SetDomain(domainName string) {
	log.Debugf("Setting instance domain name to '%s'", domainName)
	m.Domain = domainName
	m.save()
}

// setPublicIP sets the public ip of the instance
func (m *Meta) setPublicIP() {
	ip, err := findPublicIP()
	if err != nil {
		log.Fatalf("Could not find instance public ip: %s", err.Error())
	}
	log.Debugf("Setting external instance IP address to '%s'", ip)
	m.PublicIP = ip
	m.save()

}

// SetAdminUser takes a username that gets saved as the instance admin user
func (m *Meta) SetAdminUser(username string) {
	log.Debugf("Setting admin user to '%s'", username)
	m.AdminUser = username
	m.save()
}

// GetAdminUser returns the username of the admin user
func (m *Meta) GetAdminUser() string {
	return m.AdminUser
}

// GetDomain returns the domain name used in this Protos instance
func (m *Meta) GetDomain() string {
	return m.Domain
}

// GetPublicIP returns the public IP of the Protos instance
func (m *Meta) GetPublicIP() string {
	return m.PublicIP
}

// GetTLSCertificate returns the TLS certificate resource owned by the instance
func (m *Meta) GetTLSCertificate() core.Resource {

	for _, rscid := range m.Resources {
		rsc, err := m.rm.Get(rscid)
		if err != nil {
			log.Errorf("Could not find protos resource: %s", err.Error())
			continue
		}
		if rsc.GetType() == core.ResourceType("certificate") {
			return rsc
		}
	}
	return nil
}

// CleanProtosResources removes the MX record resource owned by the instance, created during the init process
func (m *Meta) CleanProtosResources() error {
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
func (m *Meta) GetDashboardDomain() string {
	dashboardDomain := m.DashboardSubdomain + "." + m.GetDomain()
	if gconfig.HTTPSport != 443 {
		dashboardDomain = fmt.Sprintf("%s:%d", dashboardDomain, gconfig.HTTPSport)
	}
	return dashboardDomain
}

// GetVersion returns current version
func (m *Meta) GetVersion() string {
	return m.version
}

// CreateProtosResources creates the DNS and TLS certificate for the Protos dashboard
func (m *Meta) CreateProtosResources() (map[string]core.Resource, error) {
	resources := map[string]core.Resource{}

	// creating the protos subdomain for the dashboard
	dnsrsc, err := m.rm.CreateDNS("protos", "protos", "A", m.PublicIP, 300)
	if err != nil {
		switch err := errors.Cause(err).(type) {
		case core.ErrResourceExists:
			dnsrscValue, ok := dnsrsc.GetValue().(dnsResource)
			if ok == false {
				log.Fatal("dnsrscValue does not implement interface dnsResource")
			}
			dnsrscValue.UpdateValueAndTTL(m.PublicIP, 300)
			dnsrsc.UpdateValue(dnsrscValue.(core.ResourceValue))
		default:
			return resources, errors.Wrap(err, "Could not create or update Protos DNS resource")
		}
	}
	// creating the bogus MX record, which is checked by LetsEncrypt before creating a certificate
	mxrsc, err := m.rm.CreateDNS("protos", "@", "MX", "protos."+m.Domain, 300)
	if err != nil {
		switch err := errors.Cause(err).(type) {
		case core.ErrResourceExists:
			mxrscValue, ok := mxrsc.GetValue().(dnsResource)
			if ok == false {
				log.Fatal("mxrscValue does not implement interface dnsResource")
			}
			mxrscValue.UpdateValueAndTTL("protos."+m.Domain, 300)
			dnsrsc.UpdateValue(mxrscValue.(core.ResourceValue))
		default:
			return resources, errors.Wrap(err, "Could not create or update Protos DNS resource")
		}
	}
	// creating a TLS certificate for the protos subdomain
	certrsc, err := m.rm.CreateCert("protos", []string{"protos"})
	if err != nil {
		switch err := errors.Cause(err).(type) {
		case core.ErrResourceExists:
		default:
			return resources, errors.Wrap(err, "Could not create Protos certificate resource")
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
func (m *Meta) GetProtosResources() map[string]core.Resource {
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
func (m *Meta) GetService() util.Service {
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

// InitMode returns the status of the init process
func (m *Meta) InitMode() bool {
	type certificate interface {
		GetCertificate() []byte
		GetPrivateKey() []byte
	}

	if m.PublicIP == "" || m.Domain == "" || m.AdminUser == "" {
		log.Warnf("Instance info (public IP: '%s', domain: '%s', admin user: '%s') is not set. Running in init mode", m.PublicIP, m.Domain, m.AdminUser)
		return true
	}

	resources := m.GetProtosResources()
	if len(resources) == 0 {
		log.Warn("Protos resources not found. Running in init mode")
		return true
	}

	tlsResource := m.GetTLSCertificate()
	if tlsResource == nil {
		log.Warn("Protos TLS certificate resource not found. Running in init mode")
		return true
	}

	tlsCertRsc := tlsResource.GetValue()
	cert, ok := tlsCertRsc.(certificate)
	if ok == false || len(cert.GetCertificate()) == 0 {
		log.Warn("Protos TLS certificate resource not found. Running in init mode")
		return true
	}

	return false
}
