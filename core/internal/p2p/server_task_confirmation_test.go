package p2p

import (
	"slices"
	"testing"

	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/tasks"
)

func TestTaskWriteConfirmationToP2PProtoPreservesCurrentMachineReadableObservation(t *testing.T) {
	t.Parallel()

	confirmation := tasks.WriteConfirmation{
		Stage:                  db.PublishedWriteConfirmationOtherPeerAvailable,
		EventID:                "event-1",
		PublishedRootHash:      "root-1",
		RequiredOtherPeers:     1,
		ConfirmedOtherPeers:    1,
		AvailabilityPending:    false,
		CandidateScope:         "current_logical_peers",
		EligiblePeerIDs:        []string{"peer-b"},
		NoCurrentEligiblePeers: false,
		ReasonCode:             "",
		AvailabilityError:      "internal prose must not cross P2P",
	}
	got := taskWriteConfirmationToP2PProto(confirmation)
	if got == nil {
		t.Fatal("confirmation is nil")
	}
	if got.GetStage() != "other_peer_available" || got.GetEventId() != "event-1" || got.GetPublishedRootHash() != "root-1" {
		t.Fatalf("unexpected receipt boundary: %+v", got)
	}
	if got.GetRequiredOtherPeers() != 1 || got.GetConfirmedOtherPeers() != 1 || got.GetAvailabilityPending() {
		t.Fatalf("unexpected availability evidence: %+v", got)
	}
	if got.GetCandidateScope() != "current_logical_peers" || !slices.Equal(got.GetEligiblePeerIds(), []string{"peer-b"}) ||
		got.GetNoCurrentEligiblePeers() || got.GetReasonCode() != "" {
		t.Fatalf("unexpected topology summary: %+v", got)
	}
	confirmation.EligiblePeerIDs[0] = "mutated-after-mapping"
	if !slices.Equal(got.GetEligiblePeerIds(), []string{"peer-b"}) {
		t.Fatalf("eligible peers were aliased across P2P mapping: %+v", got)
	}
	if taskWriteConfirmationToP2PProto(tasks.WriteConfirmation{}) != nil {
		t.Fatal("empty current observation should remain absent")
	}
}
