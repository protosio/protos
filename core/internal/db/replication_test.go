package db

import (
	"reflect"
	"testing"
)

func TestReplicationPriorityForDeviceClass(t *testing.T) {
	tests := []struct {
		deviceClass string
		want        int
		found       bool
	}{
		{deviceClass: ReplicationDeviceClassPhone, want: 10, found: true},
		{deviceClass: "laptop", want: 50, found: true},
		{deviceClass: ReplicationDeviceClassLocalVM, want: 30, found: true},
		{deviceClass: "vps", want: 100, found: true},
		{deviceClass: "unknown", found: false},
	}
	for _, tt := range tests {
		got, found := ReplicationPriorityForDeviceClass(tt.deviceClass)
		if found != tt.found {
			t.Fatalf("ReplicationPriorityForDeviceClass(%q) found=%v, want %v", tt.deviceClass, found, tt.found)
		}
		if got != tt.want {
			t.Fatalf("ReplicationPriorityForDeviceClass(%q) priority=%d, want %d", tt.deviceClass, got, tt.want)
		}
	}
}

func TestPrioritizedReplicationCandidatesSortsByPriorityAndKeepsHighestPeerPriority(t *testing.T) {
	got := prioritizedReplicationCandidates([]ReplicationCandidate{
		{PeerID: "device", DeviceClass: ReplicationDeviceClassPhone},
		{PeerID: "cloud", DeviceClass: ReplicationDeviceClassCloudVM},
		{PeerID: "device", DeviceClass: ReplicationDeviceClassLocalVM},
		{PeerID: "laptop", DeviceClass: ReplicationDeviceClassLocalUserClient},
		{PeerID: "ignored", DeviceClass: "unknown"},
	})
	want := []prioritizedReplicationCandidate{
		{PeerID: "cloud", DeviceClass: ReplicationDeviceClassCloudVM, Priority: 100},
		{PeerID: "laptop", DeviceClass: ReplicationDeviceClassLocalUserClient, Priority: 50},
		{PeerID: "device", DeviceClass: ReplicationDeviceClassLocalVM, Priority: 30},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("prioritizedReplicationCandidates() = %#v, want %#v", got, want)
	}
}

func TestPrioritizedReplicationCandidatesUsesExplicitPriority(t *testing.T) {
	got := prioritizedReplicationCandidates([]ReplicationCandidate{
		{PeerID: "low-default", DeviceClass: ReplicationDeviceClassPhone, Priority: 90},
		{PeerID: "cloud", DeviceClass: ReplicationDeviceClassCloudVM},
		{PeerID: "unknown", DeviceClass: "unknown", Priority: 60},
	})
	want := []prioritizedReplicationCandidate{
		{PeerID: "cloud", DeviceClass: ReplicationDeviceClassCloudVM, Priority: 100},
		{PeerID: "low-default", DeviceClass: ReplicationDeviceClassPhone, Priority: 90},
		{PeerID: "unknown", DeviceClass: "unknown", Priority: 60},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("prioritizedReplicationCandidates() = %#v, want %#v", got, want)
	}
}
