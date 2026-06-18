package provisioners

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/release"
	"github.com/protosio/protos/internal/tasks"
)

const (
	InstanceDeploymentTaskStream     = "provisioners.instance.deploy"
	InstanceLifecycleTaskStream      = "provisioners.instance.lifecycle"
	ProvisionerImageUploadTaskStream = "provisioners.image.upload"

	taskSubjectInstance         = "instance"
	taskSubjectProvisionerImage = "provisioner_image"

	instanceLifecycleOperationReconcile = "reconcile"
	instanceLifecycleOperationDelete    = "delete"
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

type instanceLifecycleTaskPayload struct {
	InstanceID     string `json:"instance_id"`
	InstanceName   string `json:"instance_name"`
	Operation      string `json:"operation"`
	DesiredStatus  string `json:"desired_status"`
	LocalOnly      bool   `json:"local_only"`
	DesiredSig     string `json:"desired_sig,omitempty"`
	RequestedByAPI bool   `json:"requested_by_api,omitempty"`
}

type instanceLifecycleTaskResult struct {
	InstanceID    string `json:"instance_id"`
	InstanceName  string `json:"instance_name"`
	Operation     string `json:"operation"`
	DesiredStatus string `json:"desired_status,omitempty"`
	Deleted       bool   `json:"deleted,omitempty"`
	Changed       bool   `json:"changed,omitempty"`
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
	if err := tasks.Register(cm.tasks, tasks.Stream[instanceLifecycleTaskPayload, instanceLifecycleTaskResult]{
		Name: InstanceLifecycleTaskStream,
		Run:  cm.runInstanceLifecycleTask,
	}); err != nil {
		return err
	}
	return tasks.Register(cm.tasks, tasks.Stream[uploadLocalImageTaskPayload, uploadLocalImageTaskResult]{
		Name: ProvisionerImageUploadTaskStream,
		Run:  cm.runUploadLocalImageTask,
	})
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

func (cm *Manager) QueueDesiredInstanceReconciles() error {
	if cm == nil || cm.tasks == nil {
		return fmt.Errorf("task manager is not configured")
	}
	instances, err := cm.GetInstances(false)
	if err != nil {
		return err
	}
	var failures []string
	for _, instance := range instances {
		if IsDeletingInstance(instance) || strings.TrimSpace(instance.PublicKey) == "" {
			continue
		}
		desiredStatus := normalizeDesiredInstanceStatus(instance.DesiredStatus)
		if desiredStatus == "" {
			continue
		}
		sig := lifecycleDesiredSignature(instance, desiredStatus)
		if cm.lifecycleSignatureCurrent(instance.ID, sig) {
			continue
		}
		if _, err := cm.queueInstanceLifecycle(instance, instanceLifecycleOperationReconcile, desiredStatus, false, false, sig); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", instance.Name, err))
			continue
		}
		cm.setLifecycleSignature(instance.ID, sig)
	}
	if len(failures) > 0 {
		return fmt.Errorf("%s", strings.Join(failures, "; "))
	}
	return nil
}

func (cm *Manager) QueueStartInstance(id string) (tasks.Record, error) {
	return cm.queueSetInstanceDesiredStatus(id, ServerStateRunning)
}

func (cm *Manager) QueueStopInstance(id string) (tasks.Record, error) {
	return cm.queueSetInstanceDesiredStatus(id, ServerStateStopped)
}

func (cm *Manager) QueueDeleteInstance(ctx context.Context, id string) (tasks.Record, error) {
	return cm.queueDeleteInstance(ctx, id, false)
}

func (cm *Manager) QueueDeleteInstanceLocal(ctx context.Context, id string) (tasks.Record, error) {
	return cm.queueDeleteInstance(ctx, id, true)
}

func (cm *Manager) queueSetInstanceDesiredStatus(id string, desiredStatus string) (tasks.Record, error) {
	instance, err := cm.getInstanceRecord(id)
	if err != nil {
		return tasks.Record{}, fmt.Errorf("could not retrieve instance '%s': %w", id, err)
	}
	desiredStatus = normalizeDesiredInstanceStatus(desiredStatus)
	if desiredStatus == "" {
		return tasks.Record{}, fmt.Errorf("invalid desired instance status")
	}
	instance.DesiredStatus = desiredStatus
	im, _ := createInstanceUpdateMapper(instance)
	if err := db.Update(cm.db, im); err != nil {
		return tasks.Record{}, fmt.Errorf("failed to save instance '%s': %w", id, err)
	}
	sig := lifecycleDesiredSignature(instance, desiredStatus)
	cm.clearLifecycleSignature(instance.ID)
	record, err := cm.queueInstanceLifecycle(instance, instanceLifecycleOperationReconcile, desiredStatus, false, true, sig)
	if err != nil {
		return tasks.Record{}, err
	}
	cm.setLifecycleSignature(instance.ID, sig)
	log.Infof("Queued desired status '%s' for instance '%s' as task '%s'", desiredStatus, instance.Name, record.ID)
	return record, nil
}

func (cm *Manager) queueDeleteInstance(ctx context.Context, id string, localOnly bool) (tasks.Record, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return tasks.Record{}, err
	}
	instance, err := cm.getInstanceRecord(id)
	if err != nil {
		return tasks.Record{}, fmt.Errorf("could not retrieve instance '%s': %w", id, err)
	}
	if existing, found, err := cm.tasks.LatestForSubject(InstanceLifecycleTaskStream, taskSubjectInstance, instanceLifecycleSubjectID(instance.ID, instanceLifecycleOperationReconcile)); err == nil && found && isActiveLifecycleTask(existing) {
		if cancelErr := cm.tasks.Cancel(existing.ID, "instance delete requested"); cancelErr != nil {
			log.Warnf("failed to cancel reconcile task for instance '%s': %s", instance.Name, cancelErr.Error())
		}
	}
	record, err := cm.queueInstanceLifecycle(instance, instanceLifecycleOperationDelete, ServerStateDeleting, localOnly, true, "")
	if err != nil {
		return tasks.Record{}, err
	}
	cm.clearLifecycleSignature(instance.ID)
	log.Infof("Queued delete for instance '%s' as task '%s'", instance.Name, record.ID)
	return record, nil
}

func (cm *Manager) queueInstanceLifecycle(instance InstanceInfo, operation string, desiredStatus string, localOnly bool, requestedByAPI bool, desiredSig string) (tasks.Record, error) {
	if cm == nil || cm.tasks == nil {
		return tasks.Record{}, fmt.Errorf("task manager is not configured")
	}
	operation = strings.TrimSpace(operation)
	if operation == "" {
		return tasks.Record{}, fmt.Errorf("instance lifecycle operation is empty")
	}
	title := fmt.Sprintf("Reconcile instance %s", instance.Name)
	message := "queued"
	if operation == instanceLifecycleOperationDelete {
		title = fmt.Sprintf("Delete instance %s", instance.Name)
	}
	record, _, err := tasks.EnqueueUnique(cm.tasks, tasks.EnqueueUniqueOptions[instanceLifecycleTaskPayload]{
		EnqueueOptions: tasks.EnqueueOptions[instanceLifecycleTaskPayload]{
			Stream:      InstanceLifecycleTaskStream,
			SubjectType: taskSubjectInstance,
			SubjectID:   instanceLifecycleSubjectID(instance.ID, operation),
			Title:       title,
			Message:     message,
			Payload: instanceLifecycleTaskPayload{
				InstanceID:     instance.ID,
				InstanceName:   instance.Name,
				Operation:      operation,
				DesiredStatus:  desiredStatus,
				LocalOnly:      localOnly,
				DesiredSig:     desiredSig,
				RequestedByAPI: requestedByAPI,
			},
			MaxAttempts: 1,
		},
	})
	return record, err
}

func (cm *Manager) runInstanceLifecycleTask(ctx context.Context, task *tasks.RunContext[instanceLifecycleTaskPayload]) (instanceLifecycleTaskResult, error) {
	payload := task.Payload()
	instanceID := strings.TrimSpace(payload.InstanceID)
	if instanceID == "" {
		return instanceLifecycleTaskResult{}, fmt.Errorf("instance lifecycle task missing instance id")
	}
	operation := strings.TrimSpace(payload.Operation)
	switch operation {
	case instanceLifecycleOperationReconcile:
		if err := task.Update(10, "reconciling desired instance state", lifecycleTaskDetails(payload)); err != nil {
			return instanceLifecycleTaskResult{}, err
		}
		changed, instance, err := cm.reconcileDesiredInstance(ctx, task.Progress, instanceID)
		if err != nil {
			cm.clearLifecycleSignature(instanceID)
			return instanceLifecycleTaskResult{}, err
		}
		return instanceLifecycleTaskResult{
			InstanceID:    instance.ID,
			InstanceName:  instance.Name,
			Operation:     operation,
			DesiredStatus: instance.DesiredStatus,
			Changed:       changed,
		}, nil
	case instanceLifecycleOperationDelete:
		if err := task.Update(5, "deleting instance", lifecycleTaskDetails(payload)); err != nil {
			return instanceLifecycleTaskResult{}, err
		}
		if err := cm.deleteInstanceImperative(ctx, task.Update, instanceID, payload.LocalOnly); err != nil {
			return instanceLifecycleTaskResult{}, err
		}
		return instanceLifecycleTaskResult{
			InstanceID:   instanceID,
			InstanceName: payload.InstanceName,
			Operation:    operation,
			Deleted:      true,
		}, nil
	default:
		return instanceLifecycleTaskResult{}, fmt.Errorf("unsupported instance lifecycle operation %q", operation)
	}
}

func instanceLifecycleSubjectID(instanceID string, operation string) string {
	return strings.TrimSpace(instanceID) + "/" + strings.TrimSpace(operation)
}

func isActiveLifecycleTask(record tasks.Record) bool {
	return record.Status == tasks.StatusPending || record.Status == tasks.StatusRunning
}

func lifecycleTaskDetails(payload instanceLifecycleTaskPayload) map[string]any {
	return map[string]any{
		"instance_id":      payload.InstanceID,
		"instance_name":    payload.InstanceName,
		"operation":        payload.Operation,
		"desired_status":   payload.DesiredStatus,
		"local_only":       payload.LocalOnly,
		"requested_by_api": payload.RequestedByAPI,
	}
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
