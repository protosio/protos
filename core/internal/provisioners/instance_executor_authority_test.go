package provisioners

import (
	"context"
	stdsql "database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/tasks"
)

func TestInstanceLifecycleOwnerSerializesForeignReconcileAgainstOwnerDelete(t *testing.T) {
	store := openProvisionerTestDB(t)
	status, ok := store.SwarmionStatus()
	if !ok || strings.TrimSpace(status.PeerID) == "" {
		t.Fatal("owner peer identity is unavailable")
	}
	ownerPeerID := status.PeerID

	instance := instanceWithTestLifecycleOwner(t, store, peerDrainAuthorizationTestInstance(t))
	instance.KindID = "executor-authority-provider"
	insertInstanceForDeleteReceiptTest(t, store, &instance)
	provider := &fakeStopFailDeleteProvider{instances: map[string]InstanceInfo{
		instance.ProviderResourceID: {
			ID:                 instance.ProviderResourceID,
			ProviderResourceID: instance.ProviderResourceID,
			Status:             ServerStateRunning,
		},
	}}
	registry := newProvisionerRegistry(fakeStopFailDeleteFactory{provider: provider})
	owner := newLifecycleTestManager(t, store, registry)
	if err := owner.AddProvisioner(instance.KindID, fakeStopFailDeleteType.String(), nil); err != nil {
		t.Fatal(err)
	}
	nonOwner := newLifecycleTestManager(t, store, registry)
	nonOwner.tasks.SetExecutorPeerID("peer-b")

	if _, err := nonOwner.QueueStopInstance(instance.ID); !errors.Is(err, ErrInstanceLifecycleOwnerConflict) {
		t.Fatalf("foreign QueueStopInstance error=%v, want lifecycle-owner conflict", err)
	}
	deleteTask, err := nonOwner.QueueDeleteInstance(context.Background(), instance.ID)
	if err != nil {
		t.Fatalf("foreign peer queue delete for owner: %v", err)
	}
	if deleteTask.OwnerPeerID != ownerPeerID {
		t.Fatalf("delete task owner=%q, want %q", deleteTask.OwnerPeerID, ownerPeerID)
	}
	var deletePayload instanceLifecycleTaskPayload
	if err := json.Unmarshal(deleteTask.Payload, &deletePayload); err != nil {
		t.Fatal(err)
	}
	if deletePayload.DeleteOperation == nil || deletePayload.DeleteOperation.AuthorPeerID != ownerPeerID ||
		deletePayload.PeerDrainAuthorization == nil || deletePayload.PeerDrainAuthorization.AuthorPeerID != ownerPeerID ||
		deletePayload.PeerDrainAuthorization.Instance.LifecycleOwnerPeerID != ownerPeerID {
		t.Fatalf("delete/P authority escaped persisted owner: %+v", deletePayload)
	}

	// Simulate a stale task created by B before learning the owner assignment.
	// Even though B owns this task row and its runner selects it, the stream-level
	// guard must fail before provider initialization, status lookup, Start or Stop.
	stale, err := tasks.Enqueue(nonOwner.tasks, tasks.EnqueueOptions[instanceLifecycleTaskPayload]{
		Stream:      InstanceLifecycleTaskStream,
		SubjectType: taskSubjectInstance,
		SubjectID:   instance.ID + "/stale-peer-b-reconcile",
		OwnerPeerID: "peer-b",
		Title:       "stale foreign reconcile",
		Payload: instanceLifecycleTaskPayload{
			InstanceID:    instance.ID,
			InstanceName:  instance.Name,
			Operation:     instanceLifecycleOperationReconcile,
			DesiredStatus: ServerStateStopped,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := nonOwner.tasks.RunPending(context.Background()); err == nil || !strings.Contains(err.Error(), ErrInstanceLifecycleOwnerConflict.Error()) {
		t.Fatalf("foreign stale reconcile error=%v, want lifecycle-owner conflict", err)
	}
	stale, err = nonOwner.tasks.Get(stale.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stale.Status != tasks.StatusFailed {
		t.Fatalf("stale foreign task status=%q, want failed", stale.Status)
	}
	if provider.startCalls != 0 || provider.stopCalls != 0 || provider.deleteCalls != 0 {
		t.Fatalf("foreign executor entered provider: start=%d stop=%d delete=%d", provider.startCalls, provider.stopCalls, provider.deleteCalls)
	}

	owner.peerDrainRuntime = &fakeReplicationPeerDrainRuntime{
		available: true,
		begin: swarmionapp.PeerDrainStatus{
			Active:                           true,
			RouteGenerationMatches:           true,
			LocalCheckpointCovered:           true,
			PreFenceHeartbeatIngressObserved: true,
			ReadyToFinalize:                  true,
		},
		finalize: swarmionapp.PeerDrainFinalizeResponse{Finalized: true},
	}
	owner.peerRouteFence = &fakeReplicationPeerRouteFence{prefix: "executor-authority"}
	if err := owner.tasks.RunPending(context.Background()); err != nil {
		t.Fatalf("owner delete: %v", err)
	}
	if provider.startCalls != 0 || provider.stopCalls != 1 || provider.deleteCalls != 1 {
		t.Fatalf("owner provider calls start=%d stop=%d delete=%d, want 0/1/1", provider.startCalls, provider.stopCalls, provider.deleteCalls)
	}
	if _, err := owner.getInstanceRecord(instance.ID); !errors.Is(err, stdsql.ErrNoRows) {
		t.Fatalf("deleted instance lookup error=%v, want absent", err)
	}

	// Restart with the same physical identity retains authority. A different
	// identity cannot infer a takeover merely because the original process died.
	restartedOwner := newLifecycleTestManager(t, store, registry)
	if restartedOwner.tasks.ExecutorPeerID() != ownerPeerID {
		t.Fatalf("restarted executor=%q, want %q", restartedOwner.tasks.ExecutorPeerID(), ownerPeerID)
	}
}

func TestInstanceLifecycleOwnerOrdersBlockedReconcileBeforeRemoteDelete(t *testing.T) {
	store := openProvisionerTestDB(t)
	status, ok := store.SwarmionStatus()
	if !ok || strings.TrimSpace(status.PeerID) == "" {
		t.Fatal("owner peer identity is unavailable")
	}
	ownerPeerID := status.PeerID

	instance := instanceWithTestLifecycleOwner(t, store, peerDrainAuthorizationTestInstance(t))
	instance.KindID = "blocked-owner-reconcile-provider"
	insertInstanceForDeleteReceiptTest(t, store, &instance)
	stopEntered := make(chan struct{}, 1)
	releaseStop := make(chan struct{})
	provider := &fakeStopFailDeleteProvider{
		instances: map[string]InstanceInfo{
			instance.ProviderResourceID: {
				ID:                 instance.ProviderResourceID,
				ProviderResourceID: instance.ProviderResourceID,
				PublicIP:           instance.PublicIP,
				Status:             ServerStateRunning,
			},
		},
		stopEntered: stopEntered,
		releaseStop: releaseStop,
	}
	registry := newProvisionerRegistry(fakeStopFailDeleteFactory{provider: provider})
	owner := newLifecycleTestManager(t, store, registry)
	if err := owner.AddProvisioner(instance.KindID, fakeStopFailDeleteType.String(), nil); err != nil {
		t.Fatal(err)
	}
	nonOwner := newLifecycleTestManager(t, store, registry)
	nonOwner.tasks.SetExecutorPeerID("peer-b")
	owner.peerDrainRuntime = &fakeReplicationPeerDrainRuntime{
		available: true,
		begin: swarmionapp.PeerDrainStatus{
			Active:                           true,
			RouteGenerationMatches:           true,
			LocalCheckpointCovered:           true,
			PreFenceHeartbeatIngressObserved: true,
			ReadyToFinalize:                  true,
		},
		finalize: swarmionapp.PeerDrainFinalizeResponse{Finalized: true},
	}
	owner.peerRouteFence = &fakeReplicationPeerRouteFence{prefix: "blocked-owner-reconcile"}

	reconcileTask, err := owner.QueueStopInstance(instance.ID)
	if err != nil {
		t.Fatalf("queue owner reconcile: %v", err)
	}
	reconcileDone := make(chan error, 1)
	go func() {
		reconcileDone <- owner.tasks.RunPending(context.Background())
	}()
	select {
	case <-stopEntered:
	case <-time.After(10 * time.Second):
		t.Fatal("owner reconcile did not enter provider StopInstance")
	}

	deleteTask, err := nonOwner.QueueDeleteInstance(context.Background(), instance.ID)
	if err != nil {
		t.Fatalf("remote queue delete while owner reconcile is blocked: %v", err)
	}
	if deleteTask.OwnerPeerID != ownerPeerID {
		t.Fatalf("remote delete owner=%q, want %q", deleteTask.OwnerPeerID, ownerPeerID)
	}
	if err := nonOwner.tasks.RunPending(context.Background()); err != nil {
		t.Fatalf("non-owner pending scan: %v", err)
	}
	if provider.stopCalls != 1 || provider.deleteCalls != 0 {
		t.Fatalf("remote executor overlapped provider I/O: stop=%d delete=%d", provider.stopCalls, provider.deleteCalls)
	}
	queuedDelete, err := owner.tasks.Get(deleteTask.ID)
	if err != nil {
		t.Fatal(err)
	}
	if queuedDelete.Status != tasks.StatusPending {
		t.Fatalf("delete status while reconcile blocked=%q, want pending", queuedDelete.Status)
	}

	close(releaseStop)
	select {
	case err := <-reconcileDone:
		if err != nil {
			t.Fatalf("complete owner reconcile: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("owner reconcile did not finish after provider release")
	}
	completedReconcile, err := owner.tasks.Get(reconcileTask.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completedReconcile.Status != tasks.StatusSucceeded {
		t.Fatalf("reconcile status=%q, want succeeded", completedReconcile.Status)
	}
	if err := owner.tasks.RunPending(context.Background()); err != nil {
		t.Fatalf("owner delete after reconcile: %v", err)
	}
	if got := strings.Join(provider.callOrder, ","); got != "stop,stop,delete" {
		t.Fatalf("provider call order=%q, want stop,stop,delete", got)
	}
}

func TestLegacyBlankInstanceLifecycleOwnerFailsClosed(t *testing.T) {
	store := openProvisionerTestDB(t)
	instance := peerDrainAuthorizationTestInstance(t)
	instance.LifecycleOwnerPeerID = ""
	machine, metadata := createInstanceInsertMapper(instance)
	if _, err := db.InsertWithReceiptContext(context.Background(), store, machine, metadata); err != nil {
		t.Fatal(err)
	}
	provider := &fakeStopFailDeleteProvider{instances: map[string]InstanceInfo{}}
	manager := newLifecycleTestManager(t, store, newProvisionerRegistry(fakeStopFailDeleteFactory{provider: provider}))
	if _, err := manager.QueueDeleteInstance(context.Background(), instance.ID); !errors.Is(err, ErrInstanceLifecycleOwnerUnavailable) {
		t.Fatalf("legacy blank delete error=%v, want owner unavailable", err)
	}
	if _, err := manager.QueueStartInstance(instance.ID); !errors.Is(err, ErrInstanceLifecycleOwnerUnavailable) {
		t.Fatalf("legacy blank start error=%v, want owner unavailable", err)
	}
	if provider.startCalls != 0 || provider.stopCalls != 0 || provider.deleteCalls != 0 {
		t.Fatalf("blank authority entered provider: start=%d stop=%d delete=%d", provider.startCalls, provider.stopCalls, provider.deleteCalls)
	}
}
