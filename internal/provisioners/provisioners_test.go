package provisioners

import (
	"fmt"
	"testing"
)

type fakeComputeProvisioner struct {
	instances map[string]InstanceInfo
}

func (f fakeComputeProvisioner) SupportedLocations() []string {
	return nil
}

func (f fakeComputeProvisioner) SupportedMachines(string) (map[string]MachineSpec, error) {
	return nil, nil
}

func (f fakeComputeProvisioner) NewInstance(string, string, string, string, string) (string, error) {
	return "", nil
}

func (f fakeComputeProvisioner) DeleteInstance(string, string) error {
	return nil
}

func (f fakeComputeProvisioner) StartInstance(string, string) error {
	return nil
}

func (f fakeComputeProvisioner) StopInstance(string, string) error {
	return nil
}

func (f fakeComputeProvisioner) GetInstanceInfo(id string, location string) (InstanceInfo, error) {
	info, found := f.instances[id]
	if !found {
		return InstanceInfo{}, fmt.Errorf("not found: %s", id)
	}
	info.Location = location
	return info, nil
}

func TestGetProviderInstanceInfoPrefersProviderResourceID(t *testing.T) {
	provider := fakeComputeProvisioner{instances: map[string]InstanceInfo{
		"provider-vm-id": {
			ID:                 "provider-vm-id",
			Name:               "vm",
			ProviderResourceID: "provider-vm-id",
			PublicIP:           "192.0.2.10",
		},
	}}

	info, providerID, err := getProviderInstanceInfo(provider, InstanceInfo{
		ID:                 "peer-id",
		Name:               "vm",
		ProviderResourceID: "provider-vm-id",
		Location:           "local",
	})
	if err != nil {
		t.Fatal(err)
	}
	if providerID != "provider-vm-id" {
		t.Fatalf("expected provider id, got %q", providerID)
	}
	if info.ID != "provider-vm-id" {
		t.Fatalf("expected provider info, got %#v", info)
	}
}

func TestMergedReconciledInstancePreservesPeerIdentity(t *testing.T) {
	current := InstanceInfo{
		ID:                 "peer-id",
		Name:               "vm",
		PublicKey:          "public-key",
		Kind:               KindCloudVM,
		KindID:             "local",
		ProviderResourceID: "provider-vm-id",
		DesiredStatus:      ServerStateRunning,
		Location:           "local",
		Architecture:       "arm64",
	}
	observed := InstanceInfo{
		ID:       "provider-vm-id",
		Name:     "vm",
		PublicIP: "192.0.2.10",
		Status:   ServerStateRunning,
	}

	merged := mergedReconciledInstance(current, observed)
	if merged.ID != current.ID {
		t.Fatalf("expected peer id %q, got %q", current.ID, merged.ID)
	}
	if merged.ProviderResourceID != current.ProviderResourceID {
		t.Fatalf("expected provider resource id %q, got %q", current.ProviderResourceID, merged.ProviderResourceID)
	}
	if merged.PublicKey != current.PublicKey {
		t.Fatalf("expected public key to be preserved")
	}
}
