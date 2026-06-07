package invitations

import (
	"context"
	"testing"
	"time"
)

func TestManagerStartInviteDefaultsToMDNSChannel(t *testing.T) {
	channel := &fakeChannel{name: ChannelMDNS}
	manager, err := NewManager(channel)
	if err != nil {
		t.Fatal(err)
	}

	invite, err := manager.StartInvite(context.Background(), "", Invite{
		OrganisationID:   "org-1",
		OrganisationName: "home",
		DeviceName:       "phone",
		PeerID:           "peer-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if invite.Channel != ChannelMDNS {
		t.Fatalf("Channel = %q, want %q", invite.Channel, ChannelMDNS)
	}
	if len(channel.started) != 1 {
		t.Fatalf("started invites = %d, want 1", len(channel.started))
	}
	if channel.started[0].JoinMode != InviteJoinModeAny {
		t.Fatalf("started invite JoinMode = %q, want %q", channel.started[0].JoinMode, InviteJoinModeAny)
	}
}

func TestManagerBrowseAndFindUsesChannelCache(t *testing.T) {
	expiresAt := time.Now().Add(time.Hour)
	manager, err := NewManager(
		&fakeChannel{
			name: "url",
			browse: []NearbyInvite{{
				InviteID:         "invite-url",
				Channel:          "url",
				OrganisationID:   "org-1",
				OrganisationName: "home",
				DeviceName:       "macbook",
				PeerID:           "peer-url",
				SwarmionAddrs:    []string{"/ip4/192.168.1.11/tcp/10502/p2p/peer-url"},
				ExpiresAt:        expiresAt,
			}},
		},
		&fakeChannel{
			name: ChannelMDNS,
			browse: []NearbyInvite{{
				InviteID:         "invite-mdns",
				Channel:          ChannelMDNS,
				OrganisationID:   "org-1",
				OrganisationName: "home",
				DeviceName:       "phone",
				PeerID:           "peer-mdns",
				SwarmionAddrs:    []string{"/ip4/192.168.1.10/tcp/10502/p2p/peer-mdns"},
				ExpiresAt:        expiresAt,
			}},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	nearby, err := manager.Browse(context.Background(), "", time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if len(nearby) != 2 {
		t.Fatalf("nearby invites = %d, want 2", len(nearby))
	}
	item, err := manager.Find(context.Background(), ChannelMDNS, "org-1", "peer-mdns", "invite-mdns")
	if err != nil {
		t.Fatal(err)
	}
	if item.Channel != ChannelMDNS {
		t.Fatalf("Channel = %q, want %q", item.Channel, ChannelMDNS)
	}
}

func TestManagerStopStopsAllChannels(t *testing.T) {
	first := &fakeChannel{name: ChannelMDNS}
	second := &fakeChannel{name: "url"}
	manager, err := NewManager(first, second)
	if err != nil {
		t.Fatal(err)
	}

	manager.Stop()

	if !first.stopped {
		t.Fatal("first channel was not stopped")
	}
	if !second.stopped {
		t.Fatal("second channel was not stopped")
	}
}

type fakeChannel struct {
	name    string
	started []Invite
	browse  []NearbyInvite
	stopped bool
}

func (c *fakeChannel) Name() string {
	return c.name
}

func (c *fakeChannel) StartInvite(_ context.Context, invite Invite) (Invite, error) {
	c.started = append(c.started, invite)
	invite.InviteID = "invite-1"
	invite.Channel = c.name
	return invite, nil
}

func (c *fakeChannel) Browse(context.Context, time.Duration) ([]NearbyInvite, error) {
	return c.browse, nil
}

func (c *fakeChannel) Stop() {
	c.stopped = true
}
