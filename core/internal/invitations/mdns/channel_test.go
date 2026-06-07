package mdns

import (
	"net"
	"reflect"
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
		VerificationCode: "12345678",
		VerificationHash: "verification-hash",
		Port:             10501,
		P2PAddrs: []string{
			"/ip4/192.168.1.10/tcp/10501/p2p/peer-1",
			"/ip4/192.168.1.10/tcp/10501/p2p/peer-1",
		},
		SwarmionAddrs: []string{
			"/ip4/192.168.1.10/tcp/10502/p2p/peer-1",
			"/ip4/192.168.1.11/tcp/10502/p2p/peer-1",
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
	if got.VerificationHash == "" {
		t.Fatal("VerificationHash is empty")
	}
	if len(got.P2PAddrs) != 0 {
		t.Fatalf("P2PAddrs = %#v, want none", got.P2PAddrs)
	}
	wantSwarmionAddrs := []string{"/ip4/192.168.1.10/tcp/10502/p2p/peer-1"}
	wantSwarmionAddrs = append(wantSwarmionAddrs, "/ip4/192.168.1.11/tcp/10502/p2p/peer-1")
	if !reflect.DeepEqual(got.SwarmionAddrs, wantSwarmionAddrs) {
		t.Fatalf("SwarmionAddrs = %#v, want %#v", got.SwarmionAddrs, wantSwarmionAddrs)
	}
	if !got.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("ExpiresAt = %s, want %s", got.ExpiresAt, expiresAt)
	}
}

func TestInviteTXTIncludesBootstrapAndOmitsVerificationCode(t *testing.T) {
	txt := inviteTXT(invitations.Invite{
		InviteID:         "invite-1",
		Channel:          invitations.ChannelMDNS,
		OrganisationID:   "org-1",
		OrganisationName: "home",
		DeviceName:       "macbook",
		PeerID:           "peer-1",
		PublicKey:        "public-key",
		VerificationHash: "verification-hash",
		Port:             10500,
		P2PAddrs:         []string{"/ip4/192.168.1.10/tcp/10500/p2p/peer-1"},
		SwarmionAddrs:    []string{"/ip4/192.168.1.10/tcp/10501/p2p/peer-1"},
		ExpiresAt:        time.Now().Add(time.Hour),
	})
	hasSwarmionAddr := false
	for _, value := range txt {
		if strings.HasPrefix(value, "p2p_addr=") {
			t.Fatalf("TXT contains p2p_addr entry: %q", value)
		}
		if strings.HasPrefix(value, "swarmion_addr=") {
			t.Fatalf("TXT contains dictionary-unsafe swarmion_addr entry: %q", value)
		}
		if strings.HasPrefix(value, "swarmion_addr_0=") {
			hasSwarmionAddr = true
		}
		if strings.HasPrefix(value, "verification_code=") {
			t.Fatalf("TXT contains verification_code entry: %q", value)
		}
	}
	if !hasSwarmionAddr {
		t.Fatal("TXT does not contain indexed swarmion_addr")
	}
}

func TestParseEntryRejectsMissingExplicitSwarmionAddress(t *testing.T) {
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
			"public_key=public-key",
			"verification_hash=verification-hash",
			"expires_at=1790000000",
		},
		AddrIPv4: []net.IP{net.ParseIP("192.168.1.10")},
	})
	if ok {
		t.Fatalf("parseEntry returned ok=true with %#v", got)
	}
}
