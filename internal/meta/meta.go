package meta

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/core"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/ssh"
	"github.com/tidwall/gjson"

	// "golang.zx2c4.com/wireguard/wgctrl/wgtypes"

	"github.com/pkg/errors"

	"github.com/protosio/protos/internal/util"
)

const (
	metaDS = "meta"
)

var log = util.GetLogger("meta")
var gconfig = config.Get()

// Meta contains information about the Protos instance
type Meta struct {
	rm                 core.ResourceManager `noms:"-"`
	db                 db.DB                `noms:"-"`
	keymngr            *ssh.Manager         `noms:"-"`
	version            string               `noms:"-"`
	networkSetSignal   chan net.IP          `noms:"-"`
	domainSetSignal    chan string          `noms:"-"`
	adminUserSetSignal chan string          `noms:"-"`

	// Public members
	ID                 string
	Domain             string
	DashboardSubdomain string
	PublicIP           net.IP
	AdminUser          string
	Resources          []string
	Network            net.IPNet
	InternalIP         net.IP
	PrivateKeySeed     []byte
}

type dnsResource interface {
	IsType(string) bool
	UpdateValueAndTTL(value string, ttl int)
}

// Setup reads the domain and other information on first run and save this information to the database
func Setup(rm core.ResourceManager, db db.DB, keymngr *ssh.Manager, version string) *Meta {
	if rm == nil || db == nil || keymngr == nil {
		log.Panic("Failed to setup meta package: none of the inputs can be nil")
	}

	metaRoot := Meta{}
	log.Debug("Reading instance information from database")
	err := db.GetStruct(metaDS, &metaRoot)
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

	if len(metaRoot.PrivateKeySeed) == 0 {
		key, err := keymngr.GenerateKey()
		if err != nil {
			log.Fatalf("Failed to generate instance key: ", err.Error())
		}
		metaRoot.PrivateKeySeed = key.Seed()
		log.Infof("Generated instance key. Wireguard public key: '%s'", key.PublicWG().String())
		err = ioutil.WriteFile("/tmp/protos_key.txt", []byte(key.PublicWG().String()), 0644)
		if err != nil {
			log.Fatalf("Failed to write public key to disk: ", err.Error())
		}
	}

	metaRoot.db = db
	metaRoot.rm = rm
	metaRoot.keymngr = keymngr
	metaRoot.version = version
	metaRoot.networkSetSignal = make(chan net.IP, 1)
	metaRoot.domainSetSignal = make(chan string, 1)
	metaRoot.adminUserSetSignal = make(chan string, 1)
	err = db.SaveStruct(metaDS, metaRoot)
	if err != nil {
		log.Fatalf("Failed to write the metaroot to database: %s", err.Error())
	}
	// metaRoot.setPublicIP()
	return &metaRoot
}

// SetupForClient reads the domain and other information on first run and save this information to the database
func SetupForClient(rm core.ResourceManager, db db.DB, version string) *Meta {
	if rm == nil || db == nil {
		log.Panic("Failed to setup meta package: none of the inputs can be nil")
	}

	metaRoot := Meta{}
	log.Debug("Reading instance information from database")
	err := db.GetStruct(metaDS, &metaRoot)
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
	metaRoot.networkSetSignal = make(chan net.IP, 1)
	metaRoot.domainSetSignal = make(chan string, 1)
	metaRoot.adminUserSetSignal = make(chan string, 1)
	err = db.SaveStruct(metaDS, metaRoot)
	if err != nil {
		log.Fatalf("Failed to write the metaroot to database: %s", err.Error())
	}
	return &metaRoot
}

func (m *Meta) save() {
	err := m.db.SaveStruct(metaDS, *m)
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
	m.domainSetSignal <- m.Domain
}

// GetDomain returns the domain name used in this Protos instance
func (m *Meta) GetDomain() string {
	return m.Domain
}

// SetNetwork sets the instance network
func (m *Meta) SetNetwork(network net.IPNet) net.IP {
	log.Debugf("Setting instance network to '%s'", network.String())
	ip := network.IP.Mask(network.Mask)
	ip[3]++
	m.InternalIP = ip
	m.Network = network
	m.save()
	m.networkSetSignal <- ip
	return ip
}

// GetNetwork gets the instance network
func (m *Meta) GetNetwork() net.IPNet {
	return m.Network
}

// GetInternalIP gets the instance IP
func (m *Meta) GetInternalIP() net.IP {
	return m.InternalIP
}

// setPublicIP sets the public ip of the instance
func (m *Meta) setPublicIP() {
	ipstr, err := findPublicIP()
	if err != nil {
		log.Errorf("Could not find instance public IP: %s", err.Error())
		if m.PublicIP != nil {
			log.Warnf("Using stale public IP '%s'", m.PublicIP.String())
			return
		}
		log.Fatal("No IP found in the database")
	}
	log.Debugf("Setting external instance IP address to '%s'", ipstr)
	ip := net.ParseIP(ipstr)
	if ip == nil {
		log.Fatalf("Could not parse instance public ip: %s", err.Error())
	}
	m.PublicIP = ip
	m.save()

}

// SetAdminUser takes a username that gets saved as the instance admin user
func (m *Meta) SetAdminUser(username string) {
	log.Debugf("Setting admin user to '%s'", username)
	m.AdminUser = username
	m.save()
	m.adminUserSetSignal <- username
}

// GetAdminUser returns the username of the admin user
func (m *Meta) GetAdminUser() string {
	return m.AdminUser
}

// GetPublicIP returns the public IP of the Protos instance
func (m *Meta) GetPublicIP() string {
	return m.PublicIP.String()
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

// GetKey returns the private key of the instance, in wireguard format
func (m *Meta) GetKey() (*ssh.Key, error) {
	key, err := m.keymngr.NewKeyFromSeed(m.PrivateKeySeed)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// CleanProtosResources removes the MX record resource owned by the instance, created during the init process
func (m *Meta) CleanProtosResources() error {
	log.Info("Cleaning fake DNS (MX) Protos resource")
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
	dnsrsc, err := m.rm.CreateDNS("protos", "protos", "A", m.PublicIP.String(), 300)
	if err != nil {
		switch err := errors.Cause(err).(type) {
		case core.ErrResourceExists:
			dnsrscValue, ok := dnsrsc.GetValue().(dnsResource)
			if ok == false {
				log.Fatal("dnsrscValue does not implement interface dnsResource")
			}
			dnsrscValue.UpdateValueAndTTL(m.PublicIP.String(), 300)
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

	if m.PublicIP == nil || m.Domain == "" || m.AdminUser == "" {
		log.Warnf("Instance info (public IP: '%s', domain: '%s', admin user: '%s') is not set. Running in init mode", m.PublicIP, m.Domain, m.AdminUser)
		return true
	}

	return false
}

// WaitForInit returns when both the domain and network has been set
func (m *Meta) WaitForInit(ctx context.Context) (net.IP, net.IPNet, string, string) {
	if m.InternalIP != nil && m.Domain != "" && m.AdminUser != "" {
		return m.InternalIP, m.Network, m.Domain, m.AdminUser
	}

	var domain string
	var internalIP net.IP
	var adminUser string

	initialized := make(chan bool)

	go func() {
		log.Debug("Waiting for initialisation to complete")
		domain = <-m.domainSetSignal
		internalIP = <-m.networkSetSignal
		adminUser = <-m.adminUserSetSignal
		initialized <- true
	}()

	select {
	case <-ctx.Done():
		log.Debug("Init did not finish. Canceled by user")
		return internalIP, m.Network, domain, adminUser
	case <-initialized:
		log.Debug("Init finished")
		return internalIP, m.Network, domain, adminUser
	}

}
