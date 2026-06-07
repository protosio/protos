package main

import (
	"flag"
	"testing"

	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/urfave/cli/v2"
)

func TestJoinOrganisationRequestFromFlagsDefaultsNameToUsername(t *testing.T) {
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	set.String("username", "", "")
	set.String("name", "", "")
	set.String("verification-code", "", "")
	set.String("organization-id", "", "")
	set.String("peer-id", "", "")
	set.String("invite-id", "", "")
	set.String("channel", "", "")
	set.String("mode", "", "")
	if err := set.Parse([]string{
		"--username", "alex",
		"--verification-code", "12345678",
		"--organization-id", "org-1",
		"--peer-id", "peer-1",
		"--mode", "new-device",
	}); err != nil {
		t.Fatalf("parse args: %v", err)
	}

	req := joinOrganisationRequestFromFlags(cli.NewContext(cli.NewApp(), set, nil))

	if req.Username != "alex" {
		t.Fatalf("Username = %q, want alex", req.Username)
	}
	if req.Name != "alex" {
		t.Fatalf("Name = %q, want alex", req.Name)
	}
	if req.OrganisationId != "org-1" {
		t.Fatalf("OrganisationId = %q, want org-1", req.OrganisationId)
	}
	if req.PeerId != "peer-1" {
		t.Fatalf("PeerId = %q, want peer-1", req.PeerId)
	}
	if req.Channel != defaultInviteChannel {
		t.Fatalf("Channel = %q, want %q", req.Channel, defaultInviteChannel)
	}
	if req.VerificationCode != "12345678" {
		t.Fatalf("VerificationCode = %q, want 12345678", req.VerificationCode)
	}
	if req.JoinMode != inviteJoinModeNewDevice {
		t.Fatalf("JoinMode = %q, want %q", req.JoinMode, inviteJoinModeNewDevice)
	}
}

func TestMatchingNearbyOrganisationsAppliesJoinFilters(t *testing.T) {
	items := []*pbApic.NearbyOrganisation{
		{
			OrganisationId: "org-1",
			PeerId:         "peer-1",
			InviteId:       "invite-1",
			Channel:        "mdns",
			JoinMode:       inviteJoinModeNewDevice,
		},
		{
			OrganisationId: "org-2",
			PeerId:         "peer-2",
			InviteId:       "invite-2",
			Channel:        "mdns",
			JoinMode:       inviteJoinModeNewUser,
		},
		{
			OrganisationId: "org-1",
			PeerId:         "peer-3",
			InviteId:       "invite-3",
			Channel:        "url",
			JoinMode:       inviteJoinModeNewDevice,
		},
		nil,
	}
	req := &pbApic.JoinOrganisationRequest{
		OrganisationId: "org-1",
		Channel:        "mdns",
		JoinMode:       inviteJoinModeNewDevice,
	}

	matches := matchingNearbyOrganisations(items, req)

	if len(matches) != 1 {
		t.Fatalf("matches = %d, want 1: %#v", len(matches), matches)
	}
	if matches[0].PeerId != "peer-1" {
		t.Fatalf("matched peer = %q, want peer-1", matches[0].PeerId)
	}
}

func TestInviteJoinModeRejectsAnyInvite(t *testing.T) {
	if _, err := inviteJoinMode("any"); err == nil {
		t.Fatal("inviteJoinMode accepted any")
	}
}
