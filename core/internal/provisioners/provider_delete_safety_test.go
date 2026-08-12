package provisioners

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestProviderLookupErrorTextCannotAuthorizeDurableDelete(t *testing.T) {
	fixture := newPeerDrainAuthorizationCrashFixture(t)
	fixture.provider.getErr = errors.New("credential endpoint returned resource not found")

	err := fixture.runDelete()
	if err == nil || !strings.Contains(err.Error(), "credential endpoint") {
		t.Fatalf("delete error = %v, want provider lookup failure", err)
	}
	if fixture.publishDCalls != 0 {
		t.Fatalf("durable delete publication calls = %d, want 0 while provider instance may be live", fixture.publishDCalls)
	}
	if fixture.provider.deleteCalls != 0 {
		t.Fatalf("provider delete calls = %d, want 0 after inconclusive lookup", fixture.provider.deleteCalls)
	}
}

func TestTypedProviderAbsenceCanAuthorizeDurableDelete(t *testing.T) {
	fixture := newPeerDrainAuthorizationCrashFixture(t)
	fixture.provider.getErr = fmt.Errorf("%w: %s", ErrInstanceNotFound, fixture.instance.ProviderResourceID)

	err := fixture.runDelete()
	if err == nil || !strings.Contains(err.Error(), "stop after provider boundary") {
		t.Fatalf("delete error = %v, want injected durable publication boundary", err)
	}
	if fixture.publishDCalls != 1 {
		t.Fatalf("durable delete publication calls = %d, want 1 after authoritative provider absence", fixture.publishDCalls)
	}
	if fixture.provider.deleteCalls != 0 {
		t.Fatalf("provider delete calls = %d, want 0 for an already-absent instance", fixture.provider.deleteCalls)
	}
}
