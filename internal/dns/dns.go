package dns

import (
	"net"
	"strconv"
	"strings"

	"github.com/miekg/dns"
	"github.com/pkg/errors"

	"github.com/protosio/protos/internal/app"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/util"
)

var log = util.GetLogger("dns")

var domainsMap map[string]string = map[string]string{}

func (h *handler) localResolve(w dns.ResponseWriter, r *dns.Msg) {
	log.Debugf("Performing local DNS resolve for '%s'", r.Question[0].Name)
	msg := &dns.Msg{}
	msg.SetReply(r)

	switch r.Question[0].Qtype {
	case dns.TypeA, dns.TypeAAAA:
		msg.Authoritative = true
		domain := msg.Question[0].Name
		address, ok := domainsMap[domain]
		domainParts := strings.Split(domain, ".")

		if ok {
			appendAddressAnswer(msg, domain, address)
		} else if app, err := h.appManager.Get(domainParts[0]); err == nil {
			appendAddressAnswer(msg, domain, app.IP.String())
		}
	}

	if err := w.WriteMsg(msg); err != nil {
		log.Errorf("Failed to write DNS response: %s", err.Error())
	}
}

func appendAddressAnswer(msg *dns.Msg, domain string, address string) {
	ip := net.ParseIP(address)
	if ip == nil {
		return
	}
	if ip4 := ip.To4(); ip4 != nil {
		if msg.Question[0].Qtype != dns.TypeA {
			return
		}
		msg.Answer = append(msg.Answer, &dns.A{
			Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
			A:   ip4,
		})
		return
	}
	if msg.Question[0].Qtype != dns.TypeAAAA {
		return
	}
	msg.Answer = append(msg.Answer, &dns.AAAA{
		Hdr:  dns.RR_Header{Name: domain, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 60},
		AAAA: ip,
	})
}

func (h *handler) remoteResolve(w dns.ResponseWriter, r *dns.Msg) {
	log.Debugf("Performing external DNS resolve @ '%s' for '%s'", h.dnsServer, r.Question[0].Name)
	c := &dns.Client{Net: "udp"}
	resp, _, err := c.Exchange(r, h.dnsServer)
	if err != nil {
		log.Errorf("Failed to resolve '%s': %s", r.Question[0].Name, err.Error())
		dns.HandleFailed(w, r)
		return
	}
	if err := w.WriteMsg(resp); err != nil {
		log.Errorf("Failed to write DNS response: %s", err.Error())
	}

}

type handler struct {
	listenAddr string
	dnsServer  string
	domain     string
	appManager *app.Manager
}

func (h *handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	if isLocalDomainQuery(r.Question[0].Name, h.domain) {
		h.localResolve(w, r)
	} else if h.dnsServer != "" {
		h.remoteResolve(w, r)
	} else {
		msg := &dns.Msg{}
		msg.SetReply(r)
		if err := w.WriteMsg(msg); err != nil {
			log.Errorf("Failed to write DNS response: %s", err.Error())
		}
	}
}

func isLocalDomainQuery(name string, domain string) bool {
	name = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
	domain = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
	if name == "" || domain == "" {
		return false
	}
	return name == domain || strings.HasSuffix(name, "."+domain)
}

var srv *dns.Server

// StartServer starts a DNS server used for resolving internal Protos addresses
func StartServer(key *pcrypto.Key, port int, dnsServer string, domain string, appManager *app.Manager) func() error {
	listenAddr := key.IPv6Address().String()
	if port != 53 {
		listenAddr = "127.0.0.1"
	}
	log.Infof("Starting DNS server. Listening internally on '%s:%d' for domain '%s'", listenAddr, port, domain)
	if dnsServer != "" {
		log.Debugf("Forwarding external DNS queries to '%s'", dnsServer)
	}

	// adding the IP address used for the internal protos domain
	// ToDo: improve this
	domainsMap["protos."+domain+"."] = key.IPv6Address().String()

	srv = &dns.Server{Addr: net.JoinHostPort(listenAddr, strconv.Itoa(port)), Net: "udp"}
	srv.Handler = &handler{listenAddr: listenAddr, dnsServer: dnsServer, domain: domain, appManager: appManager}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("Failed to start DNS UDP listener %s\n", err.Error())
		}
	}()

	stopper := func() error {
		return StopServer()
	}
	return stopper
}

// StopServer starts a DNS server used for resolving internal Protos addresses
func StopServer() error {
	log.Debug("Shutting down DNS server")
	if err := srv.Shutdown(); err != nil {
		return errors.Wrap(err, "Something went wrong while shutting down the DNS server")
	}
	return nil
}
