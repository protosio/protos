package provisioners

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/bokwoon95/sq"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/p2p"
	"github.com/protosio/protos/internal/p2p/proto"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/release"
	"github.com/protosio/protos/internal/tasks"
	"github.com/protosio/protos/internal/user"
)

const admissionSafetyProvisionerType = Type("admission-safety")

type admissionSafetyFactory struct {
	provider *admissionSafetyProvider
}

func (admissionSafetyFactory) Type() Type           { return admissionSafetyProvisionerType }
func (admissionSafetyFactory) AuthFields() []string { return nil }
func (factory admissionSafetyFactory) NewClient(record ProvisionerRecord, _ ProvisionerDeps) (Provisioner, error) {
	factory.provider.ProvisionerMetadata = newProvisionerMetadata(record, nil)
	return factory.provider, nil
}

type admissionSafetyProvider struct {
	fakeDeploymentProvider
	newCalls          int
	stopCalls         int
	deleteCalls       int
	volumeDeleteCalls int
}

func (provider *admissionSafetyProvider) NewInstance(name, image, originPublicKey, machineType, location string) (string, error) {
	provider.newCalls++
	return provider.fakeDeploymentProvider.NewInstance(name, image, originPublicKey, machineType, location)
}

func (provider *admissionSafetyProvider) StopInstance(id, location string) error {
	provider.stopCalls++
	return nil
}

func (provider *admissionSafetyProvider) DeleteInstance(id, location string) error {
	provider.deleteCalls++
	return nil
}

func (provider *admissionSafetyProvider) DeleteVolume(id, location string) error {
	provider.volumeDeleteCalls++
	return nil
}

func newAdmissionSafetyManager(t *testing.T, provider *admissionSafetyProvider) (*Manager, string, string) {
	t.Helper()
	store := openProvisionerTestDB(t)
	manager := newLifecycleTestManager(t, store, newProvisionerRegistry(admissionSafetyFactory{provider: provider}))
	originKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	peerKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	manager.currentDeviceForInstance = func() (user.UserDevice, error) {
		return user.UserDevice{Name: "origin", PublicKey: originKey.PublicString()}, nil
	}
	manager.originBootstrapAddrsForInstance = func(string, string) []string { return nil }
	return manager, originKey.PublicString(), peerKey.PublicString()
}

func TestDeployFailureAfterDiscoveryRetainsProviderAndReplicatedIdentity(t *testing.T) {
	provider := &admissionSafetyProvider{}
	manager, _, peerPublicKey := newAdmissionSafetyManager(t, provider)
	manager.discoverPeerForInstance = func(context.Context, string) (*p2p.DiscoveredPeer, error) {
		peerID, err := db.PeerIDFromPublicKeyString(peerPublicKey)
		if err != nil {
			t.Fatal(err)
		}
		return &p2p.DiscoveredPeer{ID: peerID, PublicKey: peerPublicKey, Address: "192.0.2.10"}, nil
	}
	addErr := errors.New("injected add peer failure")
	manager.addPeerForInstance = func(InstanceInfo) (*p2p.Client, error) { return nil, addErr }

	pending, err := manager.DeployInstance("vm", admissionSafetyProvisionerType.String(), "test-location", release.Release{Version: "dev"}, "small")
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.tasks.RunPending(context.Background()); err == nil || !strings.Contains(err.Error(), addErr.Error()) {
		t.Fatalf("RunPending error = %v, want injected AddPeer failure", err)
	}
	if provider.newCalls != 1 || provider.stopCalls != 0 || provider.deleteCalls != 0 || provider.volumeDeleteCalls != 0 {
		t.Fatalf("post-discovery compensation calls new=%d stop=%d delete=%d volume_delete=%d", provider.newCalls, provider.stopCalls, provider.deleteCalls, provider.volumeDeleteCalls)
	}

	stored, err := manager.getInstanceRecord(pending.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.PublicKey != peerPublicKey || stored.ProviderResourceID != "provider-vm-id" {
		t.Fatalf("retained deployment = %#v, want discovered key and original provider resource", stored)
	}
	peerID, err := db.PeerIDFromPublicKeyString(peerPublicKey)
	if err != nil {
		t.Fatal(err)
	}
	peerIDs, err := db.GetPeerIDs(manager.db)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := peerIDs[peerID]; !found {
		t.Fatalf("replicated peer %s missing after admission failure: %#v", peerID, peerIDs)
	}
	task, found, err := manager.tasks.LatestForSubject(InstanceDeploymentTaskStream, taskSubjectInstance, pending.ID)
	if err != nil || !found {
		t.Fatalf("deployment task lookup found=%v err=%v", found, err)
	}
	if task.Status != tasks.StatusFailed || !strings.Contains(task.ErrorMessage, ErrInstanceInitializationRecoveryRequired.Error()) {
		t.Fatalf("deployment task = %#v, want terminal recovery-required failure", task)
	}
	if err := manager.tasks.RunPending(context.Background()); err != nil {
		t.Fatalf("second RunPending should have no replayable deployment: %v", err)
	}
	if provider.newCalls != 1 {
		t.Fatalf("deployment replayed provider creation %d times", provider.newCalls)
	}
}

func TestDeployFailureBeforeDiscoveryPreservesCompensation(t *testing.T) {
	provider := &admissionSafetyProvider{}
	manager, _, _ := newAdmissionSafetyManager(t, provider)
	discoverErr := errors.New("injected discovery failure")
	manager.discoverPeerForInstance = func(context.Context, string) (*p2p.DiscoveredPeer, error) {
		return nil, discoverErr
	}

	if _, err := manager.DeployInstance("vm", admissionSafetyProvisionerType.String(), "test-location", release.Release{Version: "dev"}, "small"); err != nil {
		t.Fatal(err)
	}
	if err := manager.tasks.RunPending(context.Background()); err == nil || !strings.Contains(err.Error(), discoverErr.Error()) {
		t.Fatalf("RunPending error = %v, want discovery failure", err)
	}
	if provider.newCalls != 1 || provider.stopCalls != 1 || provider.deleteCalls != 1 || provider.volumeDeleteCalls != 1 {
		t.Fatalf("pre-discovery compensation calls new=%d stop=%d delete=%d volume_delete=%d, want 1 each", provider.newCalls, provider.stopCalls, provider.deleteCalls, provider.volumeDeleteCalls)
	}
}

func TestDeployCompletionReusesAtomicallyPersistedPeer(t *testing.T) {
	provider := &admissionSafetyProvider{}
	manager, _, peerPublicKey := newAdmissionSafetyManager(t, provider)
	manager.discoverPeerForInstance = func(context.Context, string) (*p2p.DiscoveredPeer, error) {
		peerID, err := db.PeerIDFromPublicKeyString(peerPublicKey)
		if err != nil {
			t.Fatal(err)
		}
		return &p2p.DiscoveredPeer{ID: peerID, PublicKey: peerPublicKey}, nil
	}
	manager.addPeerForInstance = func(InstanceInfo) (*p2p.Client, error) { return &p2p.Client{}, nil }
	manager.initializePeerForInstance = func(context.Context, *p2p.Client, *proto.InitRequest) (*proto.InitResponse, error) {
		return &proto.InitResponse{Architecture: "arm64"}, nil
	}

	pending, err := manager.DeployInstance("vm", admissionSafetyProvisionerType.String(), "test-location", release.Release{Version: "dev"}, "small")
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.tasks.RunPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	stored, err := manager.getInstanceRecord(pending.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.PublicKey != peerPublicKey || stored.Architecture != "arm64" {
		t.Fatalf("completed deployment = %#v", stored)
	}
	if provider.newCalls != 1 || provider.stopCalls != 0 || provider.deleteCalls != 0 || provider.volumeDeleteCalls != 0 {
		t.Fatalf("successful deployment provider calls new=%d stop=%d delete=%d volume_delete=%d", provider.newCalls, provider.stopCalls, provider.deleteCalls, provider.volumeDeleteCalls)
	}
}

func TestLegacyInitFailureRetainsReplicatedMachineAndPeer(t *testing.T) {
	store := openProvisionerTestDB(t)
	originKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	peerKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	manager := &Manager{db: store}
	manager.currentDeviceForInstance = func() (user.UserDevice, error) {
		return user.UserDevice{Name: "origin", PublicKey: originKey.PublicString()}, nil
	}
	manager.discoverPeerForInstance = func(context.Context, string) (*p2p.DiscoveredPeer, error) {
		peerID, peerErr := db.PeerIDFromPublicKeyString(peerKey.PublicString())
		return &p2p.DiscoveredPeer{ID: peerID, PublicKey: peerKey.PublicString()}, peerErr
	}
	manager.addPeerForInstance = func(InstanceInfo) (*p2p.Client, error) {
		return nil, errors.New("injected legacy AddPeer failure")
	}
	manager.originBootstrapAddrsForInstance = func(string, string) []string { return nil }

	err = manager.InitInstance("legacy-vm", KindCloudVM, "legacy-provider", "test", "192.0.2.20")
	if !errors.Is(err, ErrInstanceInitializationRecoveryRequired) {
		t.Fatalf("InitInstance error = %v, want recovery required", err)
	}
	stored, err := manager.getInstanceRecord("legacy-vm")
	if err != nil {
		t.Fatalf("replicated machine was removed after admission failure: %v", err)
	}
	if stored.PublicKey != peerKey.PublicString() {
		t.Fatalf("stored public key = %q, want discovered identity", stored.PublicKey)
	}
	peerID, err := db.PeerIDFromPublicKeyString(peerKey.PublicString())
	if err != nil {
		t.Fatal(err)
	}
	peerIDs, err := db.GetPeerIDs(store)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := peerIDs[peerID]; !found {
		t.Fatalf("replicated peer %s was removed after admission failure", peerID)
	}
}

func TestInterruptedDeploymentWithProviderIdentityIsNotRecoveredForReplay(t *testing.T) {
	provider := &admissionSafetyProvider{}
	manager, _, _ := newAdmissionSafetyManager(t, provider)
	pending, err := manager.DeployInstance("vm", admissionSafetyProvisionerType.String(), "test-location", release.Release{Version: "dev"}, "small")
	if err != nil {
		t.Fatal(err)
	}
	stored, err := manager.getInstanceRecord(pending.ID)
	if err != nil {
		t.Fatal(err)
	}
	stored.ProviderResourceID = "provider-vm-id"
	if err := manager.updateDeploymentPlaceholder(context.Background(), stored); err != nil {
		t.Fatal(err)
	}
	task, found, err := manager.tasks.LatestForSubject(InstanceDeploymentTaskStream, taskSubjectInstance, pending.ID)
	if err != nil || !found {
		t.Fatalf("deployment task lookup found=%v err=%v", found, err)
	}
	if _, err := db.UpdateWithReceiptContext(context.Background(), manager.db, func() sq.UpdateQuery {
		model := sq.New[db.TASK]("")
		return sq.Update(model).SetFunc(func(column *sq.Column) {
			column.SetString(model.STATUS, string(tasks.StatusRunning))
			column.SetInt(model.ATTEMPTS, 1)
		}).Where(db.UUIDEq(model.ID, task.ID))
	}); err != nil {
		t.Fatal(err)
	}

	recovered, err := manager.tasks.RecoverOwnedRunning()
	if err != nil {
		t.Fatal(err)
	}
	if recovered != 0 {
		t.Fatalf("recovered %d provider-backed deployment tasks for replay, want 0", recovered)
	}
	task, err = manager.tasks.Get(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != tasks.StatusRunning || task.Attempts != 1 {
		t.Fatalf("deferred deployment task changed during recovery: %#v", task)
	}
	classified := classifyDeploymentTaskError(fmt.Errorf("%w: injected", ErrInstanceInitializationRecoveryRequired))
	if !tasks.IsPermanent(classified) {
		t.Fatalf("recovery-required deployment error was not marked permanent: %T %v", classified, classified)
	}
}

func TestAdmissionPathsContainNoRawIdentityCompensation(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "provisioners.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	wanted := map[string]bool{"InitInstance": true, "deployInstanceImperative": true}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil || !wanted[function.Name.Name] {
			continue
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch called := call.Fun.(type) {
			case *ast.Ident:
				if function.Name.Name == "InitInstance" && called.Name == "createInstanceDeleteMapper" {
					t.Errorf("%s contains raw machine-row compensation %s", function.Name.Name, called.Name)
				}
			case *ast.SelectorExpr:
				owner, _ := called.X.(*ast.Ident)
				if function.Name.Name == "InitInstance" && owner != nil && owner.Name == "db" && (called.Sel.Name == "Delete" || called.Sel.Name == "CreatePeerDeleteMapper") {
					t.Errorf("%s contains raw replicated-state compensation db.%s", function.Name.Name, called.Sel.Name)
				}
				if function.Name.Name == "deployInstanceImperative" && called.Sel.Name == "RemovePeer" {
					t.Errorf("%s contains P2P route compensation %s", function.Name.Name, called.Sel.Name)
				}
			}
			return true
		})
	}
}
