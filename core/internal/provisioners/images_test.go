package provisioners

import (
	"testing"
	"time"
)

func TestNewProtosCloudImageNamesUsesCanonicalPattern(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 34, 56, 0, time.FixedZone("test", 3*60*60))
	names, err := NewProtosCloudImageNames("Mixed Cloud/e2e", now)
	if err != nil {
		t.Fatal(err)
	}
	if names.LogicalName != "mixed-cloud-e2e" {
		t.Fatalf("logical name = %q, want mixed-cloud-e2e", names.LogicalName)
	}
	if names.ImageName != "protos-image-mixed-cloud-e2e-20260615093456" {
		t.Fatalf("image name = %q", names.ImageName)
	}
	if names.SnapshotName != "protos-snapshot-mixed-cloud-e2e-20260615093456" {
		t.Fatalf("snapshot name = %q", names.SnapshotName)
	}
}

func TestProtosImageMatchesRefMatchesIDCanonicalAndLogicalNames(t *testing.T) {
	image := ImageInfo{
		ID:          "provider-id",
		Name:        "protos-image-release-20260615123456",
		LogicalName: "release",
	}
	for _, ref := range []string{"provider-id", "protos-image-release-20260615123456", "release"} {
		if !ProtosImageMatchesRef(image, ref) {
			t.Fatalf("expected image to match %q", ref)
		}
	}
	if ProtosImageMatchesRef(image, "other") {
		t.Fatal("unexpected match for other")
	}
}

func TestProtosCloudImageInfoParsesUpdatedAt(t *testing.T) {
	info := ProtosCloudImageInfo("id", "protos-image-release-20260615123456", "fsn1", "")
	if info.LogicalName != "release" {
		t.Fatalf("logical name = %q, want release", info.LogicalName)
	}
	if info.DateSuffix != "20260615123456" {
		t.Fatalf("date suffix = %q, want 20260615123456", info.DateSuffix)
	}
	if !info.Canonical {
		t.Fatal("canonical should be true")
	}
	if info.UpdatedAt.IsZero() {
		t.Fatal("updated time should be parsed from canonical suffix")
	}
}

func TestSelectProtosImageForRefUsesNewestLogicalMatch(t *testing.T) {
	images := map[string]ImageInfo{
		"old": ProtosCloudImageInfo("old", "protos-image-release-20260615120000", "fsn1", ""),
		"new": ProtosCloudImageInfo("new", "protos-image-release-20260615130000", "fsn1", ""),
	}
	id, image, found := SelectProtosImageForRef(images, "fsn1", "release")
	if !found {
		t.Fatal("expected image match")
	}
	if id != "new" || image.Name != "protos-image-release-20260615130000" {
		t.Fatalf("selected %q %#v, want newest image", id, image)
	}
}

func TestSelectProtosImageForRefPrefersExactRef(t *testing.T) {
	images := map[string]ImageInfo{
		"old": ProtosCloudImageInfo("old", "protos-image-release-20260615120000", "fsn1", ""),
		"new": ProtosCloudImageInfo("new", "protos-image-release-20260615130000", "fsn1", ""),
	}
	id, image, found := SelectProtosImageForRef(images, "fsn1", "old")
	if !found {
		t.Fatal("expected exact image match")
	}
	if id != "old" || image.Name != "protos-image-release-20260615120000" {
		t.Fatalf("selected %q %#v, want exact image", id, image)
	}
}
