package main

import (
	"testing"
	"time"

	apic "github.com/protosio/protos/apic/proto"
)

func TestAuditProvisionerImageRowsMarksCanonicalCleanupCandidates(t *testing.T) {
	cutoff := time.Unix(2000, 0)
	rows := auditProvisionerImageRows(map[string]*apic.ProvisionerImage{
		"old": {
			Id:            "old",
			Name:          "protos-image-release-19700101000001",
			LogicalName:   "release",
			Location:      "fsn1",
			UpdatedAtUnix: 1,
			Canonical:     true,
		},
		"noncanonical": {
			Id:       "noncanonical",
			Name:     "protos-mixed-e2e-noncanonical",
			Location: "fsn1",
		},
		"other-location": {
			Id:            "other-location",
			Name:          "protos-image-release-19700101000002",
			LogicalName:   "release",
			Location:      "nbg1",
			UpdatedAtUnix: 2,
			Canonical:     true,
		},
	}, "fsn1", cutoff)

	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	states := map[string]string{}
	for _, row := range rows {
		states[row.Image.Id] = row.State
	}
	if states["old"] != "cleanup-candidate" {
		t.Fatalf("old state = %q, want cleanup-candidate", states["old"])
	}
	if states["noncanonical"] != "noncanonical-protos" {
		t.Fatalf("noncanonical state = %q, want noncanonical-protos", states["noncanonical"])
	}
}

func TestCleanupProvisionerImageCandidatesKeepsCleanupCanonicalOnly(t *testing.T) {
	candidates := cleanupProvisionerImageCandidates(map[string]*apic.ProvisionerImage{
		"canonical": {
			Id:            "canonical",
			Name:          "protos-image-release-19700101000001",
			Location:      "fsn1",
			UpdatedAtUnix: 1,
			Canonical:     true,
		},
		"noncanonical": {
			Id:            "noncanonical",
			Name:          "protos-mixed-e2e-noncanonical",
			Location:      "fsn1",
			UpdatedAtUnix: 1,
			Canonical:     false,
		},
	}, "fsn1", time.Unix(2000, 0))

	if len(candidates) != 1 || candidates[0].Id != "canonical" {
		t.Fatalf("candidates = %#v, want only canonical image", candidates)
	}
}
