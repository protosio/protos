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
	cm := &Manager{
		db:           store,
		provisioners: newProvisionerRegistry(fakeMissingLocalMetadataFactory{provider: provider}),
	}
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

	if err := cm.DeleteInstance(context.Background(), "test1"); err != nil {
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
	cm := &Manager{
		db:           store,
		provisioners: newProvisionerRegistry(fakeStopFailDeleteFactory{provider: provider}),
	}
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

	if err := cm.DeleteInstance(context.Background(), "vm"); err != nil {
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
	cm := &Manager{
		db:           store,
		provisioners: newProvisionerRegistry(fakeStopFailDeleteFactory{provider: provider}),
	}
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

	err := cm.DeleteInstance(context.Background(), "vm")
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
	err := cm.DeleteInstance(ctx, "vm")
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
	instances  map[string]InstanceInfo
	startCalls int
	stopCalls  int
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

func (p *fakeDeploymentProvider) GetImages() (map[string]ImageInfo, error) {
	return map[string]ImageInfo{"image-id": {ID: "image-id", Name: "dev", Location: "test-location"}}, nil
}

func (p *fakeDeploymentProvider) GetProtosImages() (map[string]ImageInfo, error) {
	return p.GetImages()
}

func (p *fakeDeploymentProvider) AddImage(url string, hash string, version string, location string) (string, error) {
	return "image-id", nil
}

func (p *fakeDeploymentProvider) UploadLocalImage(imagePath string, imageName string, location string, timeout time.Duration) (string, error) {
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
