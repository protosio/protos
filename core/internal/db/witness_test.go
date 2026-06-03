package db

import (
	"reflect"
	"testing"

	"swarmion.dev/protocol"
	swarmionapp "swarmion.dev/runtime/app"
)

func TestWitnessRankForDeviceType(t *testing.T) {
	tests := []struct {
		deviceType string
		want       int
		found      bool
	}{
		{deviceType: WitnessDeviceTypePhone, want: 10, found: true},
		{deviceType: "laptop", want: 50, found: true},
		{deviceType: WitnessDeviceTypeLocalVM, want: 30, found: true},
		{deviceType: "vps", want: 100, found: true},
		{deviceType: "unknown", found: false},
	}
	for _, tt := range tests {
		got, found := WitnessRankForDeviceType(tt.deviceType)
		if found != tt.found {
			t.Fatalf("WitnessRankForDeviceType(%q) found=%v, want %v", tt.deviceType, found, tt.found)
		}
		if got != tt.want {
			t.Fatalf("WitnessRankForDeviceType(%q) rank=%d, want %d", tt.deviceType, got, tt.want)
		}
	}
}

func TestRankedWitnessCandidatesSortsByRankAndKeepsHighestPeerRank(t *testing.T) {
	got := rankedWitnessCandidates([]WitnessCandidate{
		{PeerID: "device", DeviceType: WitnessDeviceTypePhone},
		{PeerID: "cloud", DeviceType: WitnessDeviceTypeCloudVM},
		{PeerID: "device", DeviceType: WitnessDeviceTypeLocalVM},
		{PeerID: "laptop", DeviceType: WitnessDeviceTypeLocalUserClient},
		{PeerID: "ignored", DeviceType: "unknown"},
	})
	want := []rankedWitnessCandidate{
		{PeerID: "cloud", DeviceType: WitnessDeviceTypeCloudVM, Rank: 100},
		{PeerID: "laptop", DeviceType: WitnessDeviceTypeLocalUserClient, Rank: 50},
		{PeerID: "device", DeviceType: WitnessDeviceTypeLocalVM, Rank: 30},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rankedWitnessCandidates() = %#v, want %#v", got, want)
	}
}

func TestEligibleWitnessFormationOnlyIncludesEligibleCandidates(t *testing.T) {
	candidates := []rankedWitnessCandidate{
		{PeerID: "cloud", DeviceType: WitnessDeviceTypeCloudVM, Rank: 100},
		{PeerID: "vm", DeviceType: WitnessDeviceTypeLocalVM, Rank: 50},
		{PeerID: "device", DeviceType: WitnessDeviceTypePhone, Rank: 10},
	}
	got := eligibleWitnessFormation(candidates, map[string]struct{}{
		"cloud":  {},
		"device": {},
	})
	want := []string{"cloud", "device"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("eligibleWitnessFormation() = %#v, want %#v", got, want)
	}
}

func TestEligibleWitnessFormationKeepsLowerRankVMsAsFallbacksBehindLaptop(t *testing.T) {
	candidates := []rankedWitnessCandidate{
		{PeerID: "cloud", DeviceType: WitnessDeviceTypeCloudVM, Rank: 100},
		{PeerID: "laptop", DeviceType: WitnessDeviceTypeLocalUserClient, Rank: 50},
		{PeerID: "vm", DeviceType: WitnessDeviceTypeLocalVM, Rank: 30},
		{PeerID: "phone", DeviceType: WitnessDeviceTypePhone, Rank: 10},
	}
	got := eligibleWitnessFormation(candidates, map[string]struct{}{
		"cloud":  {},
		"laptop": {},
		"vm":     {},
		"phone":  {},
	})
	want := []string{"laptop"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("eligibleWitnessFormation() = %#v, want %#v", got, want)
	}
}

func TestEligibleWitnessFormationDoesNotPromoteSingleCloudWhenLaptopRankIsNotEligibleYet(t *testing.T) {
	candidates := []rankedWitnessCandidate{
		{PeerID: "cloud", DeviceType: WitnessDeviceTypeCloudVM, Rank: 100},
		{PeerID: "laptop", DeviceType: WitnessDeviceTypeLocalUserClient, Rank: 50},
		{PeerID: "vm", DeviceType: WitnessDeviceTypeLocalVM, Rank: 30},
	}
	got := eligibleWitnessFormation(candidates, map[string]struct{}{
		"cloud": {},
		"vm":    {},
	})
	if len(got) != 0 {
		t.Fatalf("eligibleWitnessFormation() = %#v, want no partial cloud promotion", got)
	}
}

func TestEligibleWitnessFormationPromotesTwoCloudVMsTogether(t *testing.T) {
	candidates := []rankedWitnessCandidate{
		{PeerID: "hetzner", DeviceType: WitnessDeviceTypeCloudVM, Rank: 100},
		{PeerID: "scaleway", DeviceType: WitnessDeviceTypeCloudVM, Rank: 100},
		{PeerID: "laptop", DeviceType: WitnessDeviceTypeLocalUserClient, Rank: 50},
		{PeerID: "vm", DeviceType: WitnessDeviceTypeLocalVM, Rank: 30},
	}
	got := eligibleWitnessFormation(candidates, map[string]struct{}{
		"hetzner":  {},
		"scaleway": {},
		"laptop":   {},
		"vm":       {},
	})
	want := []string{"hetzner", "scaleway"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("eligibleWitnessFormation() = %#v, want %#v", got, want)
	}
}

func TestEligibleWitnessCandidateSetKeepsWritablePeersInCloudHandoffFormation(t *testing.T) {
	candidates := []rankedWitnessCandidate{
		{PeerID: "hetzner", DeviceType: WitnessDeviceTypeCloudVM, Rank: 100},
		{PeerID: "scaleway", DeviceType: WitnessDeviceTypeCloudVM, Rank: 100},
		{PeerID: "laptop", DeviceType: WitnessDeviceTypeLocalUserClient, Rank: 50},
		{PeerID: "vm", DeviceType: WitnessDeviceTypeLocalVM, Rank: 30},
		{PeerID: "phone", DeviceType: WitnessDeviceTypePhone, Rank: 10},
	}
	got := rankedCandidatePeerIDs(eligibleWitnessCandidateSet(candidates, map[string]struct{}{
		"hetzner":  {},
		"scaleway": {},
		"laptop":   {},
		"vm":       {},
		"phone":    {},
	}))
	want := []string{"hetzner", "scaleway", "laptop", "vm", "phone"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("eligibleWitnessCandidateSet() = %#v, want %#v", got, want)
	}
}

func TestWitnessCandidateSetForApplyIncludesFullCloudHandoffFormationBeforeRanksFinalize(t *testing.T) {
	candidates := []rankedWitnessCandidate{
		{PeerID: "hetzner", DeviceType: WitnessDeviceTypeCloudVM, Rank: 100},
		{PeerID: "scaleway", DeviceType: WitnessDeviceTypeCloudVM, Rank: 100},
		{PeerID: "laptop", DeviceType: WitnessDeviceTypeLocalUserClient, Rank: 50},
		{PeerID: "vm", DeviceType: WitnessDeviceTypeLocalVM, Rank: 30},
	}
	got := witnessCandidateSetForApply(candidates)
	want := []string{"hetzner", "scaleway", "laptop", "vm"}
	if gotIDs := rankedCandidatePeerIDs(got); !reflect.DeepEqual(gotIDs, want) {
		t.Fatalf("witnessCandidateSetForApply() formation=%#v, want %#v", gotIDs, want)
	}
}

func TestWitnessCandidateSetForApplyKeepsSingleCloudStandbyBehindLaptop(t *testing.T) {
	candidates := []rankedWitnessCandidate{
		{PeerID: "hetzner", DeviceType: WitnessDeviceTypeCloudVM, Rank: 100},
		{PeerID: "laptop", DeviceType: WitnessDeviceTypeLocalUserClient, Rank: 50},
		{PeerID: "vm", DeviceType: WitnessDeviceTypeLocalVM, Rank: 30},
	}
	got := witnessCandidateSetForApply(candidates)
	want := []string{"laptop"}
	if gotIDs := rankedCandidatePeerIDs(got); !reflect.DeepEqual(gotIDs, want) {
		t.Fatalf("witnessCandidateSetForApply() formation=%#v, want %#v", gotIDs, want)
	}
}

func TestActiveWitnessCandidateFormationSatisfiedUsesActiveFormationSet(t *testing.T) {
	status := swarmionapp.Status{
		ActiveEpochID:    "child-1",
		ActiveWitnessIDs: []string{"hetzner", "scaleway"},
		EpochSnapshots: map[string]protocol.EpochSnapshot{
			"child-1": {
				FormationSet:     []protocol.PeerID{"hetzner", "scaleway", "laptop", "vm"},
				ActiveWitnessIDs: []protocol.PeerID{"hetzner", "scaleway"},
			},
		},
	}
	if !activeWitnessCandidateFormationSatisfied(status, []rankedWitnessCandidate{
		{PeerID: "hetzner"},
		{PeerID: "scaleway"},
		{PeerID: "laptop"},
		{PeerID: "vm"},
	}) {
		t.Fatal("activeWitnessCandidateFormationSatisfied() = false, want true for matching active formation")
	}
	if activeWitnessCandidateFormationSatisfied(status, []rankedWitnessCandidate{
		{PeerID: "hetzner"},
		{PeerID: "scaleway"},
		{PeerID: "laptop"},
	}) {
		t.Fatal("activeWitnessCandidateFormationSatisfied() = true, want false for different active formation")
	}
}

func TestEligibleWitnessCandidateSetDoesNotPromoteLocalVMBeforeCloudPair(t *testing.T) {
	candidates := []rankedWitnessCandidate{
		{PeerID: "laptop", DeviceType: WitnessDeviceTypeLocalUserClient, Rank: 50},
		{PeerID: "vm", DeviceType: WitnessDeviceTypeLocalVM, Rank: 30},
	}
	got := rankedCandidatePeerIDs(eligibleWitnessCandidateSet(candidates, map[string]struct{}{
		"laptop": {},
		"vm":     {},
	}))
	want := []string{"laptop"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("eligibleWitnessCandidateSet() = %#v, want %#v", got, want)
	}
}

func TestWitnessFormationInStatusFindsKnownChildEpochFormation(t *testing.T) {
	status := swarmionapp.Status{
		ActiveEpochID:    "main",
		ActiveWitnessIDs: []string{"laptop"},
		EpochSnapshots: map[string]protocol.EpochSnapshot{
			"child-1": {
				ActiveWitnessIDs: []protocol.PeerID{"hetzner", "scaleway"},
			},
		},
	}
	got, epochID, ok := WitnessFormationInStatus(status, []string{"scaleway", "hetzner"})
	if !ok {
		t.Fatal("WitnessFormationInStatus() did not find child formation")
	}
	if epochID != "child-1" {
		t.Fatalf("epochID=%q, want child-1", epochID)
	}
	want := []string{"hetzner", "scaleway"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("witnesses=%v, want %v", got, want)
	}
}

func TestSwarmionWitnessCandidates(t *testing.T) {
	got := swarmionWitnessCandidates([]rankedWitnessCandidate{
		{PeerID: "cloud", Rank: 100},
		{PeerID: "laptop", Rank: 50},
	})
	want := []swarmionapp.WitnessCandidate{
		{PeerID: "cloud", Rank: 100},
		{PeerID: "laptop", Rank: 50},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("swarmionWitnessCandidates() = %#v, want %#v", got, want)
	}
}

func TestSwarmionWitnessCandidateSource(t *testing.T) {
	status := swarmionapp.Status{
		ActiveEpochID:     "child-1",
		FinalizedCommitID: protocol.NewFinalizedCommitID("commit-1"),
		FinalizedRootHash: protocol.NewRootHash("root-1"),
	}
	got := swarmionWitnessCandidateSource(status)
	if got.ActiveEpochID != "child-1" {
		t.Fatalf("ActiveEpochID=%q, want child-1", got.ActiveEpochID)
	}
	if got.FinalizedCommitID != "commit-1" {
		t.Fatalf("FinalizedCommitID=%q, want commit-1", got.FinalizedCommitID)
	}
	if got.FinalizedRootHash == "" {
		t.Fatal("FinalizedRootHash is empty")
	}
}

func TestSwarmionWitnessCandidateSourceKeepsRootAlias(t *testing.T) {
	status := swarmionapp.Status{
		ActiveEpochID:     "main",
		FinalizedCommitID: protocol.NewFinalizedCommitID("commit-1"),
		FinalizedRootHash: protocol.NewRootHash("root-1"),
	}
	got := swarmionWitnessCandidateSource(status)
	if got.ActiveEpochID != "main" {
		t.Fatalf("ActiveEpochID=%q, want main", got.ActiveEpochID)
	}
	if got.FinalizedCommitID != "commit-1" {
		t.Fatalf("FinalizedCommitID=%q, want commit-1", got.FinalizedCommitID)
	}
	if got.FinalizedRootHash == "" {
		t.Fatal("FinalizedRootHash is empty")
	}
}
