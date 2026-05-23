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

func TestPruneWitnessFormation(t *testing.T) {
	got := pruneWitnessFormation(
		[]string{"cloud", "vm", "device"},
		[]string{"vm"},
		[]string{"missing", "device"},
	)
	want := []string{"cloud"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pruneWitnessFormation() = %#v, want %#v", got, want)
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

func TestWitnessChangeEpochID(t *testing.T) {
	if got := witnessChangeEpochID("main"); got != "" {
		t.Fatalf("witnessChangeEpochID(main)=%q, want empty current epoch", got)
	}
	if got := witnessChangeEpochID("split-1"); got != "split-1" {
		t.Fatalf("witnessChangeEpochID(split-1)=%q", got)
	}
}
