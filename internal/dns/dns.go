package dns

import (
	"net"
	"strconv"
	"strings"

	"github.com/miekg/dns"
	"github.com/pkg/errors"

	"github.com/protosio/protos/internal/app"
	"github.com/protosio/protos/internal/util"
)

var log = util.GetLogger("dns")

var domainsMap map[string]string = map[string]string{}

func (h *handler) localResolve(w dns.ResponseWriter, r *dns.Msg) {
	log.Debugf("Performing local DNS resolve for '%s'", r.Question[0].Name)
	msg := &dns.Msg{}
	msg.SetReply(r)

	switch r.Question[0].Qtype {
	case dns.TypeA:
		msg.Authoritative = true
		domain := msg.Question[0].Name
		address, ok := domainsMap[domain]
		domainParts := strings.Split(domain, ".")

		if ok {
			msg.Answer = append(msg.Answer, &dns.A{
				Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
				A:   net.ParseIP(address),
			})
		} else if app, err := h.appManager.Get(domainParts[0]); err == nil {
			msg.Answer = append(msg.Answer, &dns.A{
				Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
				A:   net.ParseIP(app.IP),
			})
		}
	}

	w.WriteMsg(msg)
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
	w.WriteMsg(resp)

}

type handler struct {
	listenAddr string
	dnsServer  string
	appManager *app.Manager
}

func (h *handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	domainl := strings.TrimSuffix(r.Question[0].Name, ".")
	domainlParts := strings.Split(domainl, ".")

	domainq := strings.TrimSuffix(r.Question[0].Name, ".")
	domainqParts := strings.Split(domainq, ".")
	if len(domainqParts) == 3 && domainqParts[2] == domainlParts[2] && domainqParts[1] == domainlParts[1] {
		h.localResolve(w, r)
	} else if h.dnsServer != "" {
		h.remoteResolve(w, r)
	} else {
		msg := &dns.Msg{}
		msg.SetReply(r)
		w.WriteMsg(msg)
	}
}

var srv *dns.Server

// StartServer starts a DNS server used for resolving internal Protos addresses
func StartServer(internalIP string, port int, dnsServer string, domain string, appManager *app.Manager) func() error {
	log.Infof("Starting DNS server. Listening internally on '%s:%d' for domain '%s'", internalIP, port, domain)
	if dnsServer != "" {
		log.Debugf("Forwarding external DNS queries to '%s'", dnsServer)
	}

	// adding the IP address used for the internal protos domain
	// ToDo: improve this
	domainsMap["protos."+domain+"."] = internalIP

	srv = &dns.Server{Addr: internalIP + ":" + strconv.Itoa(port), Net: "udp"}
	srv.Handler = &handler{listenAddr: internalIP, dnsServer: dnsServer, appManager: appManager}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("Failed to set udp listener %s\n", err.Error())
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
