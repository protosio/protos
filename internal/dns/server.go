package dns

import (
	"net"
	"strconv"

	"github.com/miekg/dns"
	"github.com/pkg/errors"

	"github.com/protosio/protos/internal/util"
)

var log = util.GetLogger("dns")

var domainsToAddresses map[string]string = map[string]string{
	"protos.": "192.168.40.210",
}

type handler struct{}

func (h *handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := dns.Msg{}
	msg.SetReply(r)
	switch r.Question[0].Qtype {
	case dns.TypeA:
		msg.Authoritative = true
		domain := msg.Question[0].Name
		address, ok := domainsToAddresses[domain]

		if ok {
			msg.Answer = append(msg.Answer, &dns.A{
				Hdr: dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
				A:   net.ParseIP(address),
			})
		}
	}
	w.WriteMsg(&msg)
}

// Server starts a DNS server used for resolving internal Protos addresses
func Server(quit chan bool) {
	log.Infof("Listening internally on ':%s' (DNS)", "53")
	srv := &dns.Server{Addr: ":" + strconv.Itoa(53), Net: "udp"}
	srv.Handler = &handler{}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("Failed to set udp listener %s\n", err.Error())
		}
	}()
	<-quit
	log.Info("Shutting down DNS webserver")
	if err := srv.Shutdown(); err != nil {
		log.Error(errors.Wrap(err, "Something went wrong while shutting down the DNS webserver"))
	}
	srv.Shutdown()
}
