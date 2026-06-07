package provisioners

import (
	"context"
	"fmt"
	"time"

	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/release"
	"github.com/protosio/protos/internal/tasks"
)

const (
	InstanceDeploymentTaskStream = "provisioners.instance.deploy"

	taskSubjectInstance = "instance"
)

type deployInstanceTaskPayload struct {
	PendingInstanceID string          `json:"pending_instance_id"`
	InstanceName      string          `json:"instance_name"`
	CloudName         string          `json:"cloud_name"`
	CloudLocation     string          `json:"cloud_location"`
	Release           release.Release `json:"release"`
	MachineType       string          `json:"machine_type"`
}

type deployInstanceTaskResult struct {
	PendingInstanceID  string `json:"pending_instance_id"`
	InstanceID         string `json:"instance_id"`
	ProviderResourceID string `json:"provider_resource_id"`
	PublicIP           string `json:"public_ip"`
	PublicKey          string `json:"public_key"`
}

func (cm *Manager) registerTaskStreams() error {
	return tasks.Register(cm.tasks, tasks.Stream[deployInstanceTaskPayload, deployInstanceTaskResult]{
		Name: InstanceDeploymentTaskStream,
		Run:  cm.runDeployInstanceTask,
	})
}

func (cm *Manager) StartTaskRunner(ctx context.Context, interval time.Duration) func() error {
	if cm == nil || cm.tasks == nil {
		return func() error { return nil }
	}
	return cm.tasks.Start(ctx, interval)
}

func (cm *Manager) TaskManager() *tasks.Manager {
	if cm == nil {
		return nil
	}
	return cm.tasks
}

func newPendingInstanceID() string {
	return db.MustNewUUIDv7()
}

func (cm *Manager) runDeployInstanceTask(ctx context.Context, task *tasks.RunContext[deployInstanceTaskPayload]) (deployInstanceTaskResult, error) {
	payload := task.Payload()
	if payload.PendingInstanceID == "" {
		return deployInstanceTaskResult{}, fmt.Errorf("deployment task missing pending instance id")
	}
	instance, err := cm.deployInstanceImperative(ctx, task.Update, payload.PendingInstanceID, payload.InstanceName, payload.CloudName, payload.CloudLocation, payload.Release, payload.MachineType)
	if err != nil {
		return deployInstanceTaskResult{}, err
	}
	return deployInstanceTaskResult{
		PendingInstanceID:  payload.PendingInstanceID,
		InstanceID:         instance.ID,
		ProviderResourceID: instance.ProviderResourceID,
		PublicIP:           instance.PublicIP,
		PublicKey:          instance.PublicKey,
	}, nil
}

func (cm *Manager) updateDeploymentPlaceholder(instance InstanceInfo) error {
	im, cmm := createInstanceUpdateMapper(instance)
	if err := db.Update(cm.db, im, cmm); err != nil {
		return fmt.Errorf("failed to update pending instance '%s': %w", instance.Name, err)
	}
	return nil
}

func (cm *Manager) completeDeploymentInstance(pendingID string, instance InstanceInfo) error {
	if _, err := db.SelectOne(cm.db, createInstanceQueryMapper(pendingID)); err != nil {
		return fmt.Errorf("pending instance '%s' no longer exists: %w", pendingID, err)
	}
	instance.ID = pendingID
	im, cmm := createInstanceFinalizeMapper(pendingID, instance)
	if err := db.Update(cm.db, im, cmm); err != nil {
		return err
	}
	if err := db.Insert(cm.db, db.CreatePeerInsertMapper(instance.PublicKey)); err != nil {
		return err
	}
	return nil
}
