package dns

import (
	"testing"

	mdns "github.com/miekg/dns"
)

func TestAppendAddressAnswer(t *testing.T) {
	tests := []struct {
		name        string
		qtype       uint16
		address     string
		wantAnswers int
		wantType    uint16
	}{
		{
			name:        "ipv4 A",
			qtype:       mdns.TypeA,
			address:     "192.0.2.10",
			wantAnswers: 1,
			wantType:    mdns.TypeA,
		},
		{
			name:        "ipv6 AAAA",
			qtype:       mdns.TypeAAAA,
			address:     "200:1855:ef37:8cab:9529:68fc:e21e:487c",
			wantAnswers: 1,
			wantType:    mdns.TypeAAAA,
		},
		{
			name:        "ipv6 A ignored",
			qtype:       mdns.TypeA,
			address:     "200:1855:ef37:8cab:9529:68fc:e21e:487c",
			wantAnswers: 0,
		},
		{
			name:        "ipv4 AAAA ignored",
			qtype:       mdns.TypeAAAA,
			address:     "192.0.2.10",
			wantAnswers: 0,
		},
		{
			name:        "invalid ignored",
			qtype:       mdns.TypeA,
			address:     "not-an-ip",
			wantAnswers: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &mdns.Msg{}
			msg.SetQuestion("protos.protos.internal.", tt.qtype)

			appendAddressAnswer(msg, "protos.protos.internal.", tt.address)

			if got := len(msg.Answer); got != tt.wantAnswers {
				t.Fatalf("answer count = %d, want %d", got, tt.wantAnswers)
			}
			if tt.wantAnswers == 0 {
				return
			}
			if got := msg.Answer[0].Header().Rrtype; got != tt.wantType {
				t.Fatalf("answer type = %d, want %d", got, tt.wantType)
			}
		})
	}
}
