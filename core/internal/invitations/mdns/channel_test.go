package mdns

import (
	"net"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/grandcat/zeroconf"

	"github.com/protosio/protos/internal/invitations"
)

func TestInviteTXTRoundTrip(t *testing.T) {
	expiresAt := time.Now().Add(time.Hour).Truncate(time.Second)
	invite := invitations.Invite{
		InviteID:         "invite-1",
		OrganisationID:   "org-1",
		OrganisationName: "home",
		DeviceName:       "macbook",
		PeerID:           "peer-1",
		PublicKey:        "public-key",
		Port:             10501,
		P2PAddrs: []string{
			"/ip4/192.168.1.10/tcp/10501/p2p/peer-1",
			"/ip4/192.168.1.10/tcp/10501/p2p/peer-1",
		},
		SwarmionAddrs: []string{
			"/ip4/192.168.1.10/tcp/10502/p2p/peer-1",
		},
		ExpiresAt: expiresAt,
	}

	got, ok := parseEntry(&zeroconf.ServiceEntry{
		HostName: "macbook.local.",
		Port:     invite.Port,
		Text:     inviteTXT(invite),
		AddrIPv4: []net.IP{net.ParseIP("192.168.1.10")},
	})
	if !ok {
		t.Fatal("parseEntry returned ok=false")
	}
	if got.InviteID != invite.InviteID {
		t.Fatalf("InviteID = %q, want %q", got.InviteID, invite.InviteID)
	}
	if got.Channel != invitations.ChannelMDNS {
		t.Fatalf("Channel = %q, want %q", got.Channel, invitations.ChannelMDNS)
	}
	if got.OrganisationID != invite.OrganisationID {
		t.Fatalf("OrganisationID = %q, want %q", got.OrganisationID, invite.OrganisationID)
	}
	if got.OrganisationName != invite.OrganisationName {
		t.Fatalf("OrganisationName = %q, want %q", got.OrganisationName, invite.OrganisationName)
	}
	if got.DeviceName != invite.DeviceName {
		t.Fatalf("DeviceName = %q, want %q", got.DeviceName, invite.DeviceName)
	}
	if got.PeerID != invite.PeerID {
		t.Fatalf("PeerID = %q, want %q", got.PeerID, invite.PeerID)
	}
	if got.PublicKey != invite.PublicKey {
		t.Fatalf("PublicKey = %q, want %q", got.PublicKey, invite.PublicKey)
	}
	if len(got.P2PAddrs) != 0 {
		t.Fatalf("P2PAddrs = %#v, want none", got.P2PAddrs)
	}
	wantSwarmionAddrs := []string{"/ip4/192.168.1.10/tcp/10502/p2p/peer-1"}
	if !reflect.DeepEqual(got.SwarmionAddrs, wantSwarmionAddrs) {
		t.Fatalf("SwarmionAddrs = %#v, want %#v", got.SwarmionAddrs, wantSwarmionAddrs)
	}
	if !got.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("ExpiresAt = %s, want %s", got.ExpiresAt, expiresAt)
	}
}

func TestInviteTXTOmitsBulkAddresses(t *testing.T) {
	txt := inviteTXT(invitations.Invite{
		InviteID:         "invite-1",
		Channel:          invitations.ChannelMDNS,
		OrganisationID:   "org-1",
		OrganisationName: "home",
		DeviceName:       "macbook",
		PeerID:           "peer-1",
		PublicKey:        "public-key",
		Port:             10500,
		P2PAddrs:         []string{"/ip4/192.168.1.10/tcp/10500/p2p/peer-1"},
		SwarmionAddrs:    []string{"/ip4/192.168.1.10/tcp/10501/p2p/peer-1"},
		ExpiresAt:        time.Now().Add(time.Hour),
	})
	for _, value := range txt {
		if strings.HasPrefix(value, "p2p_addr=") {
			t.Fatalf("TXT contains p2p_addr entry: %q", value)
		}
		if strings.HasPrefix(value, "swarmion_addr=") {
			t.Fatalf("TXT contains swarmion_addr entry: %q", value)
		}
	}
}

func TestParseEntryBuildsFallbackSwarmionAddress(t *testing.T) {
	got, ok := parseEntry(&zeroconf.ServiceEntry{
		HostName: "macbook.local.",
		Port:     10501,
		Text: []string{
			"v=1",
			"invite_id=invite-1",
			"org_id=org-1",
			"org_name=home",
			"device_name=macbook",
			"peer_id=peer-1",
			"expires_at=" + strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10),
		},
		AddrIPv4: []net.IP{net.ParseIP("192.168.1.10")},
	})
	if !ok {
		t.Fatal("parseEntry returned ok=false")
	}
	want := []string{"/ip4/192.168.1.10/tcp/10502/p2p/peer-1"}
	if !reflect.DeepEqual(got.SwarmionAddrs, want) {
		t.Fatalf("SwarmionAddrs = %#v, want %#v", got.SwarmionAddrs, want)
	}
}
