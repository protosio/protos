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

func TestAuthorizedDeleteVolumeFailurePreservesInstanceForRetry(t *testing.T) {
	fixture := newPeerDrainAuthorizationCrashFixture(t)
	providerInstance := fixture.provider.instances[fixture.instance.ProviderResourceID]
	providerInstance.Volumes = []VolumeInfo{{
		VolumeID: "attached-volume-id",
		Name:     fixture.instance.Name,
	}}
	fixture.provider.instances[fixture.instance.ProviderResourceID] = providerInstance
	volumeErr := errors.New("provider volume remains attached")
	fixture.provider.volumeErr = volumeErr

	err := fixture.runDelete()
	if !errors.Is(err, volumeErr) {
		t.Fatalf("delete error = %v, want provider volume failure", err)
	}
	if fixture.publishPCalls != 1 || fixture.runtime.finalizeCalls != 1 {
		t.Fatalf(
			"authorization publications=%d peer-drain finalizations=%d, want 1/1",
			fixture.publishPCalls,
			fixture.runtime.finalizeCalls,
		)
	}
	if fixture.provider.volumeDeleteCalls != 1 {
		t.Fatalf("provider volume delete calls = %d, want 1", fixture.provider.volumeDeleteCalls)
	}
	if fixture.provider.deleteCalls != 0 {
		t.Fatalf("provider instance delete calls = %d, want 0 after volume failure", fixture.provider.deleteCalls)
	}
	if fixture.publishDCalls != 0 {
		t.Fatalf("durable delete publication calls = %d, want 0 after volume failure", fixture.publishDCalls)
	}
	stored, err := fixture.manager.getInstanceRecord(fixture.instance.ID)
	if err != nil {
		t.Fatalf("read retained instance after volume failure: %v", err)
	}
	if stored.ID != fixture.instance.ID || stored.ProviderResourceID != fixture.instance.ProviderResourceID {
		t.Fatalf("retained instance = %+v, want identity %s/%s", stored, fixture.instance.ID, fixture.instance.ProviderResourceID)
	}
}
