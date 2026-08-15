package invitations

import "testing"

func TestVerifyNearbyInviteCodeAllowsFormattedDigits(t *testing.T) {
	invite := NearbyInvite{
		InviteID:       "invite-1",
		OrganisationID: "org-1",
		PeerID:         "peer-1",
		PublicKey:      "public-key",
		SwarmionAddrs:  []string{"/ip4/192.168.1.10/tcp/10502/p2p/peer-1"},
	}
	invite.VerificationHash = NearbyInviteVerificationHash(invite, "12345678")

	if err := VerifyNearbyInviteCode(invite, "1234 5678"); err != nil {
		t.Fatalf("VerifyNearbyInviteCode returned error: %v", err)
	}
}

func TestVerifyNearbyInviteCodeBindsBootstrapAddresses(t *testing.T) {
	invite := NearbyInvite{
		InviteID:       "invite-1",
		OrganisationID: "org-1",
		PeerID:         "peer-1",
		PublicKey:      "public-key",
		SwarmionAddrs:  []string{"/ip4/192.168.1.10/tcp/10502/p2p/peer-1"},
	}
	invite.VerificationHash = NearbyInviteVerificationHash(invite, "12345678")

	tampered := invite
	tampered.SwarmionAddrs = []string{"/ip4/192.168.1.11/tcp/10502/p2p/peer-1"}
	if err := VerifyNearbyInviteCode(tampered, "12345678"); err == nil {
		t.Fatal("VerifyNearbyInviteCode accepted tampered bootstrap addresses")
	}
}

func TestVerifyNearbyInviteCodeBindsJoinMode(t *testing.T) {
	invite := NearbyInvite{
		InviteID:       "invite-1",
		OrganisationID: "org-1",
		PeerID:         "peer-1",
		PublicKey:      "public-key",
		JoinMode:       InviteJoinModeNewUser,
		SwarmionAddrs:  []string{"/ip4/192.168.1.10/tcp/10502/p2p/peer-1"},
	}
	invite.VerificationHash = NearbyInviteVerificationHash(invite, "12345678")

	tampered := invite
	tampered.JoinMode = InviteJoinModeNewDevice
	if err := VerifyNearbyInviteCode(tampered, "12345678"); err == nil {
		t.Fatal("VerifyNearbyInviteCode accepted tampered join mode")
	}
}

func TestVerifyNearbyInviteCodeBindsTargetUserID(t *testing.T) {
	invite := NearbyInvite{
		InviteID:       "invite-1",
		OrganisationID: "org-1",
		PeerID:         "peer-1",
		PublicKey:      "public-key",
		JoinMode:       InviteJoinModeNewDevice,
		TargetUserID:   "user-1",
		SwarmionAddrs:  []string{"/ip4/192.168.1.10/tcp/10502/p2p/peer-1"},
	}
	invite.VerificationHash = NearbyInviteVerificationHash(invite, "12345678")

	tampered := invite
	tampered.TargetUserID = "user-2"
	if err := VerifyNearbyInviteCode(tampered, "12345678"); err == nil {
		t.Fatal("VerifyNearbyInviteCode accepted tampered target user")
	}
}

func TestVerifyNearbyInviteCodePreservesIssuedAnyModeHash(t *testing.T) {
	invite := NearbyInvite{
		InviteID:       "invite-1",
		OrganisationID: "org-1",
		PeerID:         "peer-1",
		PublicKey:      "public-key",
		SwarmionAddrs:  []string{"/ip4/192.168.1.10/tcp/10502/p2p/peer-1"},
	}
	hashWithoutMode := NearbyInviteVerificationHash(invite, "12345678")
	invite.JoinMode = InviteJoinModeAny
	invite.VerificationHash = hashWithoutMode

	if err := VerifyNearbyInviteCode(invite, "12345678"); err != nil {
		t.Fatalf("VerifyNearbyInviteCode returned error: %v", err)
	}
}
