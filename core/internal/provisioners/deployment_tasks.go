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
	InstanceDeploymentTaskStream     = "provisioners.instance.deploy"
	ProvisionerImageUploadTaskStream = "provisioners.image.upload"

	taskSubjectInstance         = "instance"
	taskSubjectProvisionerImage = "provisioner_image"
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

type uploadLocalImageTaskPayload struct {
	ImagePath       string `json:"image_path"`
	ImageName       string `json:"image_name"`
	ProvisionerName string `json:"provisioner_name"`
	Location        string `json:"location"`
	TimeoutSeconds  int64  `json:"timeout_seconds"`
}

type uploadLocalImageTaskResult struct {
	ImageID         string `json:"image_id"`
	ImageName       string `json:"image_name"`
	ProvisionerName string `json:"provisioner_name"`
	Location        string `json:"location"`
}

func (cm *Manager) registerTaskStreams() error {
	if err := tasks.Register(cm.tasks, tasks.Stream[deployInstanceTaskPayload, deployInstanceTaskResult]{
		Name: InstanceDeploymentTaskStream,
		Run:  cm.runDeployInstanceTask,
	}); err != nil {
		return err
	}
	return tasks.Register(cm.tasks, tasks.Stream[uploadLocalImageTaskPayload, uploadLocalImageTaskResult]{
		Name: ProvisionerImageUploadTaskStream,
		Run:  cm.runUploadLocalImageTask,
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

func (cm *Manager) QueueUploadLocalImage(imagePath string, imageName string, provisionerName string, location string, timeout time.Duration) (tasks.Record, error) {
	if cm == nil || cm.tasks == nil {
		return tasks.Record{}, fmt.Errorf("task manager is not configured")
	}
	if imagePath == "" {
		return tasks.Record{}, fmt.Errorf("image path is empty")
	}
	if imageName == "" {
		return tasks.Record{}, fmt.Errorf("image name is empty")
	}
	if provisionerName == "" {
		return tasks.Record{}, fmt.Errorf("provisioner name is empty")
	}
	return tasks.Enqueue(cm.tasks, tasks.EnqueueOptions[uploadLocalImageTaskPayload]{
		Stream:      ProvisionerImageUploadTaskStream,
		SubjectType: taskSubjectProvisionerImage,
		SubjectID:   uploadLocalImageSubjectID(provisionerName, location, imageName),
		Title:       fmt.Sprintf("Upload image %s", imageName),
		Message:     "queued",
		Payload: uploadLocalImageTaskPayload{
			ImagePath:       imagePath,
			ImageName:       imageName,
			ProvisionerName: provisionerName,
			Location:        location,
			TimeoutSeconds:  int64(timeout / time.Second),
		},
	})
}

func (cm *Manager) runUploadLocalImageTask(ctx context.Context, task *tasks.RunContext[uploadLocalImageTaskPayload]) (uploadLocalImageTaskResult, error) {
	payload := task.Payload()
	progress := func(progress int, message string, details any, durable bool) error {
		if durable {
			return task.Update(progress, message, details)
		}
		return task.Progress(progress, message, details)
	}
	imageID, err := cm.uploadLocalImageImperative(
		ctx,
		progress,
		payload.ImagePath,
		payload.ImageName,
		payload.ProvisionerName,
		payload.Location,
		time.Duration(payload.TimeoutSeconds)*time.Second,
	)
	if err != nil {
		return uploadLocalImageTaskResult{}, err
	}
	return uploadLocalImageTaskResult{
		ImageID:         imageID,
		ImageName:       payload.ImageName,
		ProvisionerName: payload.ProvisionerName,
		Location:        payload.Location,
	}, nil
}

func uploadLocalImageSubjectID(provisionerName string, location string, imageName string) string {
	if location == "" {
		return fmt.Sprintf("%s/%s", provisionerName, imageName)
	}
	return fmt.Sprintf("%s/%s/%s", provisionerName, location, imageName)
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
