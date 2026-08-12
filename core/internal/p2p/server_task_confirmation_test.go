package p2p

import (
	"testing"

	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/tasks"
)

func TestTaskWriteConfirmationToP2PProtoPreservesCurrentMachineReadableObservation(t *testing.T) {
	t.Parallel()

	confirmation := tasks.WriteConfirmation{
		Stage:               db.PublishedWriteConfirmationOtherPeerAvailable,
		EventID:             "event-1",
		PublishedRootHash:   "root-1",
		RequiredOtherPeers:  1,
		ConfirmedOtherPeers: 1,
		AvailabilityPending: false,
		AvailabilityError:   "internal prose must not cross P2P",
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
	if taskWriteConfirmationToP2PProto(tasks.WriteConfirmation{}) != nil {
		t.Fatal("empty current observation should remain absent")
	}
}
