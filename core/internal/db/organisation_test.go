package db

import (
	"regexp"
	"testing"
)

func TestGenerateOrganisationIDReturnsUUIDV7(t *testing.T) {
	id, err := GenerateOrganisationID()
	if err != nil {
		t.Fatal(err)
	}
	pattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !pattern.MatchString(id) {
		t.Fatalf("GenerateOrganisationID() = %q, want UUID v7", id)
	}
}

func TestNormalizeOrganisationNameFallsBackToHome(t *testing.T) {
	if got := normalizeOrganisationName("  "); got != DefaultOrganisationName {
		t.Fatalf("normalizeOrganisationName() = %q, want %q", got, DefaultOrganisationName)
	}
	if got := normalizeOrganisationName(" personal "); got != "personal" {
		t.Fatalf("normalizeOrganisationName() = %q, want personal", got)
	}
}
