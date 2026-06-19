package provisioners

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/release"
	"github.com/protosio/protos/internal/tasks"
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

func TestActiveInstancesExcludesStoppedAndDeleting(t *testing.T) {
	instances := []InstanceInfo{
		{Name: "running", DesiredStatus: ServerStateRunning},
		{Name: "stopped", DesiredStatus: ServerStateStopped},
		{Name: "deleting", DesiredStatus: ServerStateDeleting},
		{Name: "legacy"},
	}

	active := ActiveInstances(instances)
	if len(active) != 2 {
		t.Fatalf("active instances = %#v, want running and legacy only", active)
	}
	if active[0].Name != "running" || active[1].Name != "legacy" {
		t.Fatalf("active instances = %#v, want running and legacy only", active)
	}
}

func TestGetLocalInstanceReportsObservedStatus(t *testing.T) {
	store := openProvisionerTestDB(t)
	cm := &Manager{
		db:           store,
		provisioners: newProvisionerRegistry(fakeDeploymentFactory{}),
	}
	if err := cm.AddProvisioner("local-test", "fake", nil); err != nil {
		t.Fatal(err)
	}
	instance := InstanceInfo{
		ID:                 db.MustNewUUIDv7(),
		Name:               "local-vm",
		Kind:               KindLocalVM,
		KindID:             "local-test",
		ProviderResourceID: "vm-local",
		DesiredStatus:      ServerStateStopped,
		PublicKey:          "public-key",
		Location:           "local",
	}
	im, cmm := createInstanceInsertMapper(instance)
	if err := db.Insert(store, im, cmm); err != nil {
		t.Fatal(err)
	}

	got, err := cm.GetInstance("local-vm")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != ServerStateRunning {
		t.Fatalf("status = %q, want observed running status", got.Status)
	}

	instances, err := cm.GetInstancesWithUpdatedStatus()
	if err != nil {
		t.Fatal(err)
	}
	if len(instances) != 1 || instances[0].Status != ServerStateRunning {
		t.Fatalf("instances = %#v, want observed running local status", instances)
	}
}

func TestLogsRemoteInstanceUsesProvisionerLogs(t *testing.T) {
	store := openProvisionerTestDB(t)
	cm := &Manager{
		db:           store,
		provisioners: newProvisionerRegistry(fakeDeploymentFactory{}),
	}
	if err := cm.AddProvisioner("local-test", "fake", nil); err != nil {
		t.Fatal(err)
	}
	instance := InstanceInfo{
		ID:                 db.MustNewUUIDv7(),
		Name:               "local-vm",
		Kind:               KindLocalVM,
		KindID:             "local-test",
		ProviderResourceID: "provider-vm-id",
		Location:           "test-location",
	}
	im, cmm := createInstanceInsertMapper(instance)
	if err := db.Insert(store, im, cmm); err != nil {
		t.Fatal(err)
	}

	logs, err := cm.LogsRemoteInstance("local-vm")
	if err != nil {
		t.Fatal(err)
	}
	if logs != "logs for provider-vm-id in test-location" {
		t.Fatalf("unexpected logs: %q", logs)
	}
}

func TestDeployInstanceCreatesPendingRecordAndTask(t *testing.T) {
	store := openProvisionerTestDB(t)
	cm := &Manager{
		db:           store,
		provisioners: newProvisionerRegistry(fakeDeploymentFactory{}),
		tasks:        tasks.NewManager(store),
	}
	if err := cm.registerTaskStreams(); err != nil {
		t.Fatal(err)
	}

	instance, err := cm.DeployInstance("vm", "fake", "test-location", release.Release{Version: "dev"}, "small")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.UUIDBytes(instance.ID); err != nil {
		t.Fatalf("instance id = %q, want UUID: %v", instance.ID, err)
	}
	if instance.PublicKey != "" {
		t.Fatalf("public key = %q, want empty until task discovers peer", instance.PublicKey)
	}
	if !strings.Contains(instance.Status, "pending") {
		t.Fatalf("status = %q, want queued/pending status", instance.Status)
	}

	stored, err := db.SelectOne(store, createInstanceQueryMapper(instance.ID))
	if err != nil {
		t.Fatal(err)
	}
	if stored.Name != "vm" || stored.KindID != "fake" || stored.DesiredStatus != ServerStateRunning {
		t.Fatalf("stored instance = %#v", stored)
	}

	task, found, err := cm.tasks.LatestForSubject(InstanceDeploymentTaskStream, taskSubjectInstance, instance.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected deployment task")
	}
	if task.Status != tasks.StatusPending {
		t.Fatalf("task status = %q, want pending", task.Status)
	}
}

func TestDeleteLocalInstanceContinuesWhenProviderManifestMissing(t *testing.T) {
	store := openProvisionerTestDB(t)
	provider := &fakeMissingLocalMetadataProvider{}
	cm := newLifecycleTestManager(t, store, newProvisionerRegistry(fakeMissingLocalMetadataFactory{provider: provider}))
	if err := cm.AddProvisioner("local-test", fakeMissingLocalMetadataType.String(), nil); err != nil {
		t.Fatal(err)
	}

	instance := InstanceInfo{
		ID:                 db.MustNewUUIDv7(),
		Name:               "test1",
		Kind:               KindLocalVM,
		KindID:             "local-test",
		ProviderResourceID: "vm-missing",
		DesiredStatus:      ServerStateRunning,
		Location:           "local",
	}
	im, cmm := createInstanceInsertMapper(instance)
	if err := db.Insert(store, im, cmm); err != nil {
		t.Fatal(err)
	}

	if _, err := cm.DeleteInstance(context.Background(), "test1"); err != nil {
		t.Fatal(err)
	}
	if err := runLifecycleTasks(t, cm); err != nil {
		t.Fatal(err)
	}
	if len(provider.deleted) != 1 || provider.deleted[0] != "vm-missing" {
		t.Fatalf("deleted refs = %#v, want vm-missing", provider.deleted)
	}
	if _, err := db.SelectOne(store, createInstanceQueryMapper(instance.ID)); err == nil {
		t.Fatal("expected instance row to be removed")
	}
}

func TestDeleteInstanceContinuesToProviderDeleteWhenStopFails(t *testing.T) {
	store := openProvisionerTestDB(t)
	provider := &fakeStopFailDeleteProvider{
		instances: map[string]InstanceInfo{
			"provider-vm-id": {
				ID:                 "provider-vm-id",
				Name:               "vm",
				ProviderResourceID: "provider-vm-id",
				Status:             ServerStateRunning,
			},
		},
		stopErr: fmt.Errorf("provider state transition failed"),
	}
	cm := newLifecycleTestManager(t, store, newProvisionerRegistry(fakeStopFailDeleteFactory{provider: provider}))
	if err := cm.AddProvisioner("cloud-test", fakeStopFailDeleteType.String(), nil); err != nil {
		t.Fatal(err)
	}

	instance := InstanceInfo{
		ID:                 db.MustNewUUIDv7(),
		Name:               "vm",
		Kind:               KindCloudVM,
		KindID:             "cloud-test",
		ProviderResourceID: "provider-vm-id",
		DesiredStatus:      ServerStateRunning,
		Location:           "test-location",
	}
	im, cmm := createInstanceInsertMapper(instance)
	if err := db.Insert(store, im, cmm); err != nil {
		t.Fatal(err)
	}

	if _, err := cm.DeleteInstance(context.Background(), "vm"); err != nil {
		t.Fatal(err)
	}
	if err := runLifecycleTasks(t, cm); err != nil {
		t.Fatal(err)
	}
	if provider.stopCalls != 1 {
		t.Fatalf("stop calls = %d, want 1", provider.stopCalls)
	}
	if provider.deleteCalls != 1 {
		t.Fatalf("delete calls = %d, want 1", provider.deleteCalls)
	}
	if _, err := db.SelectOne(store, createInstanceQueryMapper(instance.ID)); err == nil {
		t.Fatal("expected instance row to be removed")
	}
}

func TestDeleteInstanceReturnsVolumeDeleteErrorAndKeepsRecord(t *testing.T) {
	store := openProvisionerTestDB(t)
	provider := &fakeStopFailDeleteProvider{
		instances: map[string]InstanceInfo{
			"provider-vm-id": {
				ID:                 "provider-vm-id",
				Name:               "vm",
				ProviderResourceID: "provider-vm-id",
				Status:             ServerStateStopped,
				Volumes: []VolumeInfo{
					{VolumeID: "volume-id", Name: "vm"},
				},
			},
		},
		volumeErr: fmt.Errorf("provider volume is locked"),
	}
	cm := newLifecycleTestManager(t, store, newProvisionerRegistry(fakeStopFailDeleteFactory{provider: provider}))
	if err := cm.AddProvisioner("cloud-test", fakeStopFailDeleteType.String(), nil); err != nil {
		t.Fatal(err)
	}

	instance := InstanceInfo{
		ID:                 db.MustNewUUIDv7(),
		Name:               "vm",
		Kind:               KindCloudVM,
		KindID:             "cloud-test",
		ProviderResourceID: "provider-vm-id",
		DesiredStatus:      ServerStateRunning,
		Location:           "test-location",
	}
	im, cmm := createInstanceInsertMapper(instance)
	if err := db.Insert(store, im, cmm); err != nil {
		t.Fatal(err)
	}

	if _, err := cm.DeleteInstance(context.Background(), "vm"); err != nil {
		t.Fatal(err)
	}
	err := runLifecycleTasks(t, cm)
	if err == nil {
		t.Fatal("expected volume delete error")
	}
	if !strings.Contains(err.Error(), "could not delete volume 'volume-id'") {
		t.Fatalf("error = %v, want volume delete context", err)
	}
	if provider.deleteCalls != 0 {
		t.Fatalf("delete calls = %d, want 0", provider.deleteCalls)
	}
	if provider.volumeDeleteCalls != 1 {
		t.Fatalf("volume delete calls = %d, want 1", provider.volumeDeleteCalls)
	}
	if _, err := db.SelectOne(store, createInstanceQueryMapper(instance.ID)); err != nil {
		t.Fatalf("expected instance row to remain for retry: %v", err)
	}
}

func TestDeleteInstanceTreatsAttachedVolumeCleanupAsRetryable(t *testing.T) {
	store := openProvisionerTestDB(t)
	provider := &fakeStopFailDeleteProvider{
		instances: map[string]InstanceInfo{
			"provider-vm-id": {
				ID:                 "provider-vm-id",
				Name:               "vm",
				ProviderResourceID: "provider-vm-id",
				Status:             ServerStateStopped,
				Volumes: []VolumeInfo{
					{VolumeID: "attached-volume-id", Name: "vm"},
				},
			},
		},
		volumeErr: fmt.Errorf("volume is attached to a server"),
	}
	cm := newLifecycleTestManager(t, store, newProvisionerRegistry(fakeStopFailDeleteFactory{provider: provider}))
	if err := cm.AddProvisioner("cloud-test", fakeStopFailDeleteType.String(), nil); err != nil {
		t.Fatal(err)
	}

	instance := InstanceInfo{
		ID:                 db.MustNewUUIDv7(),
		Name:               "vm",
		Kind:               KindCloudVM,
		KindID:             "cloud-test",
		ProviderResourceID: "provider-vm-id",
		DesiredStatus:      ServerStateRunning,
		Location:           "test-location",
	}
	im, cmm := createInstanceInsertMapper(instance)
	if err := db.Insert(store, im, cmm); err != nil {
		t.Fatal(err)
	}

	if _, err := cm.DeleteInstance(context.Background(), "vm"); err != nil {
		t.Fatal(err)
	}
	err := runLifecycleTasks(t, cm)
	if err == nil {
		t.Fatal("expected attached volume cleanup error")
	}
	if provider.deleteCalls != 0 {
		t.Fatalf("delete calls = %d, want 0", provider.deleteCalls)
	}
	if provider.volumeDeleteCalls != 1 {
		t.Fatalf("volume delete calls = %d, want 1", provider.volumeDeleteCalls)
	}
	stored, err := db.SelectOne(store, createInstanceQueryMapper(instance.ID))
	if err != nil {
		t.Fatalf("expected instance row to remain for retry: %v", err)
	}
	if stored.DesiredStatus != ServerStateDeleting {
		t.Fatalf("desired status = %q, want %q", stored.DesiredStatus, ServerStateDeleting)
	}
}

func TestDeleteInstanceHonorsCanceledContextBeforeMutatingState(t *testing.T) {
	store := openProvisionerTestDB(t)
	cm := &Manager{db: store}

	instance := InstanceInfo{
		ID:            db.MustNewUUIDv7(),
		Name:          "vm",
		Kind:          KindCloudVM,
		KindID:        "cloud-test",
		DesiredStatus: ServerStateRunning,
		Location:      "test-location",
	}
	im, cmm := createInstanceInsertMapper(instance)
	if err := db.Insert(store, im, cmm); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := cm.DeleteInstance(ctx, "vm")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("DeleteInstance error = %v, want context.Canceled", err)
	}

	stored, err := db.SelectOne(store, createInstanceQueryMapper(instance.ID))
	if err != nil {
		t.Fatal(err)
	}
	if stored.DesiredStatus != ServerStateRunning {
		t.Fatalf("desired status = %q, want %q", stored.DesiredStatus, ServerStateRunning)
	}
}

func TestGetDeclaredInstanceDoesNotRequireLiveProviderStatus(t *testing.T) {
	store := openProvisionerTestDB(t)
	cm := &Manager{db: store}
	instance := InstanceInfo{
		ID:                 db.MustNewUUIDv7(),
		Name:               "vm",
		Kind:               KindCloudVM,
		KindID:             "cloud-test",
		ProviderResourceID: "provider-vm-id",
		DesiredStatus:      ServerStateRunning,
		Location:           "test-location",
		Architecture:       "arm64",
	}
	im, cmm := createInstanceInsertMapper(instance)
	if err := db.Insert(store, im, cmm); err != nil {
		t.Fatal(err)
	}

	stored, err := cm.GetDeclaredInstance(instance.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.ID != instance.ID || stored.Name != instance.Name || stored.Status != "" {
		t.Fatalf("declared instance = %#v, want local record without live status", stored)
	}

	stored, err = cm.GetDeclaredInstance(instance.Name)
	if err != nil {
		t.Fatal(err)
	}
	if stored.ID != instance.ID || stored.Name != instance.Name || stored.Status != "" {
		t.Fatalf("declared instance by name = %#v, want local record without live status", stored)
	}
}

func TestReplicationCandidatesExcludingSkipsDeletingInstances(t *testing.T) {
	store := openProvisionerTestDB(t)
	keyManager := pcrypto.CreateManager(store)
	activeKey, err := keyManager.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	deletingKey, err := keyManager.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	active := InstanceInfo{
		ID:                  db.MustNewUUIDv7(),
		Name:                "active",
		Kind:                KindCloudVM,
		KindID:              "cloud-test",
		PublicIP:            "192.0.2.10",
		PublicKey:           activeKey.PublicString(),
		DesiredStatus:       ServerStateRunning,
		ReplicationPriority: 10,
		Location:            "test-location",
		Architecture:        "amd64",
	}
	deleting := InstanceInfo{
		ID:                  db.MustNewUUIDv7(),
		Name:                "deleting",
		Kind:                KindCloudVM,
		KindID:              "cloud-test",
		PublicIP:            "192.0.2.11",
		PublicKey:           deletingKey.PublicString(),
		DesiredStatus:       ServerStateDeleting,
		ReplicationPriority: 20,
		Location:            "test-location",
		Architecture:        "amd64",
	}
	activeMachine, activeMetadata := createInstanceInsertMapper(active)
	deletingMachine, deletingMetadata := createInstanceInsertMapper(deleting)
	if err := db.Insert(store, activeMachine, activeMetadata, deletingMachine, deletingMetadata); err != nil {
		t.Fatal(err)
	}

	cm := &Manager{db: store}
	candidates, err := cm.replicationCandidatesExcluding("removed-peer")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].PeerID != activeKey.GetID() {
		t.Fatalf("candidates = %#v, want only %s", candidates, activeKey.GetID())
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

type fakeReconcileComputeProvisioner struct {
	ProvisionerMetadata
	instances  map[string]InstanceInfo
	startCalls int
	stopCalls  int
}

func (f *fakeReconcileComputeProvisioner) Init() error {
	return nil
}

func (f *fakeReconcileComputeProvisioner) SupportedLocations() []string {
	return nil
}

func (f *fakeReconcileComputeProvisioner) SupportedMachines(string) (map[string]MachineSpec, error) {
	return nil, nil
}

func (f *fakeReconcileComputeProvisioner) NewInstance(string, string, string, string, string) (string, error) {
	return "", nil
}

func (f *fakeReconcileComputeProvisioner) DeleteInstance(string, string) error {
	return nil
}

func (f *fakeReconcileComputeProvisioner) StartInstance(id string, location string) error {
	f.startCalls++
	info := f.instances[id]
	info.Status = ServerStateRunning
	info.PublicIP = "192.0.2.11"
	f.instances[id] = info
	return nil
}

func (f *fakeReconcileComputeProvisioner) StopInstance(id string, location string) error {
	f.stopCalls++
	info := f.instances[id]
	info.Status = ServerStateStopped
	f.instances[id] = info
	return nil
}

func (f *fakeReconcileComputeProvisioner) GetInstanceInfo(id string, location string) (InstanceInfo, error) {
	info, found := f.instances[id]
	if !found {
		return InstanceInfo{}, fmt.Errorf("not found: %s", id)
	}
	info.Location = location
	return info, nil
}

const fakeReconcileType = Type("fake-reconcile")

type fakeReconcileFactory struct {
	provider *fakeReconcileComputeProvisioner
}

func (fakeReconcileFactory) Type() Type {
	return fakeReconcileType
}

func (fakeReconcileFactory) AuthFields() []string {
	return nil
}

func (f fakeReconcileFactory) NewClient(record ProvisionerRecord, deps ProvisionerDeps) (Provisioner, error) {
	provider := f.provider
	if provider == nil {
		provider = &fakeReconcileComputeProvisioner{instances: map[string]InstanceInfo{}}
	}
	provider.ProvisionerMetadata = newProvisionerMetadata(record, nil)
	return provider, nil
}

func TestReconcileComputeInstanceStartsStoppedInstance(t *testing.T) {
	provider := &fakeReconcileComputeProvisioner{instances: map[string]InstanceInfo{
		"provider-vm-id": {
			ID:                 "provider-vm-id",
			ProviderResourceID: "provider-vm-id",
			Status:             ServerStateStopped,
			PublicIP:           "192.0.2.10",
		},
	}}

	updated, err := reconcileComputeInstance(provider, InstanceInfo{
		ID:                 "peer-id",
		ProviderResourceID: "provider-vm-id",
		DesiredStatus:      ServerStateRunning,
		Location:           "test-location",
	})
	if err != nil {
		t.Fatal(err)
	}
	if provider.startCalls != 1 {
		t.Fatalf("start calls = %d, want 1", provider.startCalls)
	}
	if provider.stopCalls != 0 {
		t.Fatalf("stop calls = %d, want 0", provider.stopCalls)
	}
	if updated.Status != ServerStateRunning {
		t.Fatalf("status = %q, want running", updated.Status)
	}
	if updated.DesiredStatus != ServerStateRunning {
		t.Fatalf("desired status = %q, want running", updated.DesiredStatus)
	}
	if updated.PublicIP != "192.0.2.11" {
		t.Fatalf("public IP = %q, want refreshed value", updated.PublicIP)
	}
}

func TestQueueDesiredInstanceReconcileAppliesLocalVMDesiredStatus(t *testing.T) {
	store := openProvisionerTestDB(t)
	provider := &fakeReconcileComputeProvisioner{instances: map[string]InstanceInfo{
		"provider-vm-id": {
			ID:                 "provider-vm-id",
			ProviderResourceID: "provider-vm-id",
			Status:             ServerStateRunning,
			PublicIP:           "192.0.2.10",
		},
	}}
	cm := newLifecycleTestManager(t, store, newProvisionerRegistry(fakeReconcileFactory{provider: provider}))
	if err := cm.AddProvisioner("local-test", fakeReconcileType.String(), nil); err != nil {
		t.Fatal(err)
	}
	instance := InstanceInfo{
		ID:                 db.MustNewUUIDv7(),
		Name:               "local-vm",
		Kind:               KindLocalVM,
		KindID:             "local-test",
		ProviderResourceID: "provider-vm-id",
		DesiredStatus:      ServerStateStopped,
		Status:             ServerStateRunning,
		PublicKey:          "public-key",
		PublicIP:           "192.0.2.10",
		Location:           "local",
	}
	im, cmm := createInstanceInsertMapper(instance)
	if err := db.Insert(store, im, cmm); err != nil {
		t.Fatal(err)
	}

	if err := cm.QueueDesiredInstanceReconciles(); err != nil {
		t.Fatal(err)
	}
	if err := runLifecycleTasks(t, cm); err != nil {
		t.Fatal(err)
	}
	if provider.stopCalls != 1 {
		t.Fatalf("stop calls = %d, want 1", provider.stopCalls)
	}

	stored, err := db.SelectOne(store, createInstanceQueryMapper(instance.ID))
	if err != nil {
		t.Fatal(err)
	}
	if stored.DesiredStatus != ServerStateStopped {
		t.Fatalf("stored desired status = %q, want stopped", stored.DesiredStatus)
	}
	observed, err := cm.GetInstance(instance.ID)
	if err != nil {
		t.Fatal(err)
	}
	if observed.Status != ServerStateStopped {
		t.Fatalf("observed status = %q, want stopped", observed.Status)
	}
}

func TestReconcileComputeInstanceNoopsWhileChanging(t *testing.T) {
	provider := &fakeReconcileComputeProvisioner{instances: map[string]InstanceInfo{
		"provider-vm-id": {
			ID:                 "provider-vm-id",
			ProviderResourceID: "provider-vm-id",
			Status:             ServerStateChanging,
		},
	}}

	updated, err := reconcileComputeInstance(provider, InstanceInfo{
		ID:                 "peer-id",
		ProviderResourceID: "provider-vm-id",
		DesiredStatus:      ServerStateRunning,
		Location:           "test-location",
	})
	if err != nil {
		t.Fatal(err)
	}
	if provider.startCalls != 0 || provider.stopCalls != 0 {
		t.Fatalf("unexpected provider calls: start=%d stop=%d", provider.startCalls, provider.stopCalls)
	}
	if updated.Status != ServerStateChanging {
		t.Fatalf("status = %q, want changing", updated.Status)
	}
	if updated.DesiredStatus != ServerStateRunning {
		t.Fatalf("desired status = %q, want running", updated.DesiredStatus)
	}
}

func TestReconcileComputeInstanceNoopsWhenAlreadyDesired(t *testing.T) {
	provider := &fakeReconcileComputeProvisioner{instances: map[string]InstanceInfo{
		"provider-vm-id": {
			ID:                 "provider-vm-id",
			ProviderResourceID: "provider-vm-id",
			Status:             ServerStateRunning,
		},
	}}

	updated, err := reconcileComputeInstance(provider, InstanceInfo{
		ID:                 "peer-id",
		ProviderResourceID: "provider-vm-id",
		DesiredStatus:      ServerStateRunning,
		Location:           "test-location",
	})
	if err != nil {
		t.Fatal(err)
	}
	if provider.startCalls != 0 || provider.stopCalls != 0 {
		t.Fatalf("unexpected provider calls: start=%d stop=%d", provider.startCalls, provider.stopCalls)
	}
	if updated.Status != ServerStateRunning {
		t.Fatalf("status = %q, want running", updated.Status)
	}
}

type fakeDeploymentFactory struct{}

func (fakeDeploymentFactory) Type() Type {
	return Type("fake")
}

func (fakeDeploymentFactory) AuthFields() []string {
	return nil
}

func (fakeDeploymentFactory) NewClient(record ProvisionerRecord, deps ProvisionerDeps) (Provisioner, error) {
	return &fakeDeploymentProvider{
		ProvisionerMetadata: newProvisionerMetadata(record, nil),
	}, nil
}

type fakeDeploymentProvider struct {
	ProvisionerMetadata
}

func (p *fakeDeploymentProvider) Init() error {
	return nil
}

func (p *fakeDeploymentProvider) SupportedLocations() []string {
	return []string{"test-location"}
}

func (p *fakeDeploymentProvider) SupportedMachines(location string) (map[string]MachineSpec, error) {
	return map[string]MachineSpec{"small": {Cores: 1, Memory: 1024}}, nil
}

func (p *fakeDeploymentProvider) NewInstance(name string, image string, originPublicKey string, machineType string, location string) (string, error) {
	return "provider-vm-id", nil
}

func (p *fakeDeploymentProvider) DeleteInstance(id string, location string) error {
	return nil
}

func (p *fakeDeploymentProvider) StartInstance(id string, location string) error {
	return nil
}

func (p *fakeDeploymentProvider) StopInstance(id string, location string) error {
	return nil
}

func (p *fakeDeploymentProvider) GetInstanceInfo(id string, location string) (InstanceInfo, error) {
	return InstanceInfo{ID: id, ProviderResourceID: id, Name: "vm", PublicIP: "192.0.2.10", Status: ServerStateRunning, Location: location}, nil
}

func (p *fakeDeploymentProvider) InstanceLogs(id string, location string) (string, error) {
	return fmt.Sprintf("logs for %s in %s", id, location), nil
}

func (p *fakeDeploymentProvider) GetImages() (map[string]ImageInfo, error) {
	return map[string]ImageInfo{"image-id": {ID: "image-id", Name: "dev", Location: "test-location"}}, nil
}

func (p *fakeDeploymentProvider) GetProtosImages() (map[string]ImageInfo, error) {
	return p.GetImages()
}

func (p *fakeDeploymentProvider) AddImage(url string, hash string, version string, location string) (string, error) {
	return "image-id", nil
}

func (p *fakeDeploymentProvider) UploadLocalImage(ctx context.Context, imagePath string, imageName string, location string, timeout time.Duration, progress UploadProgressFunc) (string, error) {
	if progress != nil {
		if err := progress(UploadProgress{Phase: "test", Message: "upload in progress", BytesTransferred: 1, TotalBytes: 1}); err != nil {
			return "", err
		}
	}
	return "image-id", nil
}

func (p *fakeDeploymentProvider) RemoveImage(name string, location string) error {
	return nil
}

func (p *fakeDeploymentProvider) NewVolume(name string, size int, location string) (string, error) {
	return "volume-id", nil
}

func (p *fakeDeploymentProvider) DeleteVolume(id string, location string) error {
	return nil
}

func (p *fakeDeploymentProvider) AttachVolume(volumeID string, instanceID string, location string) error {
	return nil
}

func (p *fakeDeploymentProvider) DettachVolume(volumeID string, instanceID string, location string) error {
	return nil
}

const fakeMissingLocalMetadataType = Type("fake-missing-local-metadata")

type fakeMissingLocalMetadataFactory struct {
	provider *fakeMissingLocalMetadataProvider
}

func (fakeMissingLocalMetadataFactory) Type() Type {
	return fakeMissingLocalMetadataType
}

func (fakeMissingLocalMetadataFactory) AuthFields() []string {
	return nil
}

func (f fakeMissingLocalMetadataFactory) NewClient(record ProvisionerRecord, deps ProvisionerDeps) (Provisioner, error) {
	f.provider.ProvisionerMetadata = newProvisionerMetadata(record, nil)
	return f.provider, nil
}

type fakeMissingLocalMetadataProvider struct {
	ProvisionerMetadata
	deleted []string
}

func (p *fakeMissingLocalMetadataProvider) Init() error {
	return nil
}

func (p *fakeMissingLocalMetadataProvider) SupportedLocations() []string {
	return []string{"local"}
}

func (p *fakeMissingLocalMetadataProvider) SupportedMachines(string) (map[string]MachineSpec, error) {
	return nil, nil
}

func (p *fakeMissingLocalMetadataProvider) NewInstance(string, string, string, string, string) (string, error) {
	return "", nil
}

func (p *fakeMissingLocalMetadataProvider) DeleteInstance(id string, location string) error {
	p.deleted = append(p.deleted, id)
	return nil
}

func (p *fakeMissingLocalMetadataProvider) StartInstance(string, string) error {
	return nil
}

func (p *fakeMissingLocalMetadataProvider) StopInstance(string, string) error {
	return nil
}

func (p *fakeMissingLocalMetadataProvider) GetInstanceInfo(string, string) (InstanceInfo, error) {
	return InstanceInfo{}, fmt.Errorf("open manifest: %w", os.ErrNotExist)
}

func (p *fakeMissingLocalMetadataProvider) NewVolume(string, int, string) (string, error) {
	return "", nil
}

func (p *fakeMissingLocalMetadataProvider) DeleteVolume(string, string) error {
	return nil
}

func (p *fakeMissingLocalMetadataProvider) AttachVolume(string, string, string) error {
	return nil
}

func (p *fakeMissingLocalMetadataProvider) DettachVolume(string, string, string) error {
	return nil
}

const fakeStopFailDeleteType = Type("fake-stop-fail-delete")

type fakeStopFailDeleteFactory struct {
	provider *fakeStopFailDeleteProvider
}

func (fakeStopFailDeleteFactory) Type() Type {
	return fakeStopFailDeleteType
}

func (fakeStopFailDeleteFactory) AuthFields() []string {
	return nil
}

func (f fakeStopFailDeleteFactory) NewClient(record ProvisionerRecord, deps ProvisionerDeps) (Provisioner, error) {
	f.provider.ProvisionerMetadata = newProvisionerMetadata(record, nil)
	return f.provider, nil
}

type fakeStopFailDeleteProvider struct {
	ProvisionerMetadata
	instances         map[string]InstanceInfo
	stopErr           error
	volumeErr         error
	stopCalls         int
	deleteCalls       int
	volumeDeleteCalls int
}

func (p *fakeStopFailDeleteProvider) Init() error {
	return nil
}

func (p *fakeStopFailDeleteProvider) SupportedLocations() []string {
	return []string{"test-location"}
}

func (p *fakeStopFailDeleteProvider) SupportedMachines(string) (map[string]MachineSpec, error) {
	return nil, nil
}

func (p *fakeStopFailDeleteProvider) NewInstance(string, string, string, string, string) (string, error) {
	return "", nil
}

func (p *fakeStopFailDeleteProvider) DeleteInstance(string, string) error {
	p.deleteCalls++
	return nil
}

func (p *fakeStopFailDeleteProvider) StartInstance(string, string) error {
	return nil
}

func (p *fakeStopFailDeleteProvider) StopInstance(string, string) error {
	p.stopCalls++
	return p.stopErr
}

func (p *fakeStopFailDeleteProvider) GetInstanceInfo(id string, location string) (InstanceInfo, error) {
	info, found := p.instances[id]
	if !found {
		return InstanceInfo{}, fmt.Errorf("not found: %s", id)
	}
	info.Location = location
	return info, nil
}

func (p *fakeStopFailDeleteProvider) NewVolume(string, int, string) (string, error) {
	return "", nil
}

func (p *fakeStopFailDeleteProvider) DeleteVolume(string, string) error {
	p.volumeDeleteCalls++
	return p.volumeErr
}

func (p *fakeStopFailDeleteProvider) AttachVolume(string, string, string) error {
	return nil
}

func (p *fakeStopFailDeleteProvider) DettachVolume(string, string, string) error {
	return nil
}

func openProvisionerTestDB(t *testing.T) *db.DB {
	t.Helper()
	cfg := config.Get()
	previousP2PPort := cfg.P2PPort
	cfg.P2PPort = 0
	t.Cleanup(func() {
		cfg.P2PPort = previousP2PPort
	})

	workDir := t.TempDir()
	key, err := pcrypto.GetLocalKey(workDir)
	if err != nil {
		t.Fatal(err)
	}
	store, err := db.Open(workDir, "protos_provisioners_test", key)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	if err := store.Init(); err != nil {
		t.Fatal(err)
	}
	return store
}

func newLifecycleTestManager(t *testing.T, store *db.DB, registry *provisionerRegistry) *Manager {
	t.Helper()
	manager := &Manager{
		db:           store,
		provisioners: registry,
		tasks:        tasks.NewManager(store),
		lifecycleSig: map[string]string{},
	}
	if err := manager.registerTaskStreams(); err != nil {
		t.Fatal(err)
	}
	return manager
}

func runLifecycleTasks(t *testing.T, manager *Manager) error {
	t.Helper()
	return manager.tasks.RunPending(context.Background())
}
