package apic

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/protosio/protos/internal/p2p"
	"github.com/protosio/protos/internal/tasks"
)

const (
	InstanceImageArchiveUploadTaskStream = "instances.image_archive.upload"

	taskSubjectInstanceImageArchive = "instance_image_archive"
)

type uploadInstanceImageArchiveTaskPayload struct {
	Instance    string `json:"instance"`
	ArchivePath string `json:"archive_path"`
	ImageRef    string `json:"image_ref"`
}

type uploadInstanceImageArchiveTaskResult struct {
	Instance         string `json:"instance"`
	ImageRef         string `json:"image_ref"`
	TargetDigest     string `json:"target_digest"`
	Platform         string `json:"platform"`
	BytesUploaded    uint64 `json:"bytes_uploaded"`
	ArchiveSizeBytes uint64 `json:"archive_size_bytes"`
}

func (b *Backend) UploadInstanceImageArchive(ctx context.Context, in *pbApic.UploadInstanceImageArchiveRequest) (*pbApic.UploadInstanceImageArchiveResponse, error) {
	if err := b.requireProvisionerCapability("upload instance image archive"); err != nil {
		return nil, err
	}
	manager, err := b.taskManager()
	if err != nil {
		return nil, err
	}
	if err := b.registerAPICImageArchiveTaskStream(manager); err != nil {
		return nil, err
	}

	instance := strings.TrimSpace(in.GetInstance())
	archivePath := strings.TrimSpace(in.GetArchivePath())
	imageRef := strings.TrimSpace(in.GetImageRef())
	if instance == "" || instance == "local" {
		return nil, fmt.Errorf("instance is required")
	}
	if archivePath == "" {
		return nil, fmt.Errorf("archive path is empty")
	}
	if imageRef == "" {
		return nil, fmt.Errorf("image ref is empty")
	}
	absPath, err := filepath.Abs(archivePath)
	if err != nil {
		return nil, fmt.Errorf("resolve archive path: %w", err)
	}

	task, err := tasks.EnqueueContext(ctx, manager, tasks.EnqueueOptions[uploadInstanceImageArchiveTaskPayload]{
		Stream:      InstanceImageArchiveUploadTaskStream,
		SubjectType: taskSubjectInstanceImageArchive,
		SubjectID:   fmt.Sprintf("%s/%s", instance, imageRef),
		Title:       fmt.Sprintf("Upload image archive %s", imageRef),
		Message:     "queued",
		Payload: uploadInstanceImageArchiveTaskPayload{
			Instance:    instance,
			ArchivePath: absPath,
			ImageRef:    imageRef,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("queue instance image archive upload: %w", err)
	}
	return &pbApic.UploadInstanceImageArchiveResponse{TaskId: task.ID}, nil
}

func (b *Backend) registerAPICImageArchiveTaskStream(manager *tasks.Manager) error {
	return tasks.RegisterIfAbsent(manager, tasks.Stream[uploadInstanceImageArchiveTaskPayload, uploadInstanceImageArchiveTaskResult]{
		Name: InstanceImageArchiveUploadTaskStream,
		Run:  b.runUploadInstanceImageArchiveTask,
	})
}

func (b *Backend) registerTaskStreamsIfConfigured() {
	manager, err := b.taskManager()
	if err != nil {
		return
	}
	if err := b.registerAPICImageArchiveTaskStream(manager); err != nil {
		log.Errorf("register APIC task streams: %v", err)
	}
}

func (b *Backend) runUploadInstanceImageArchiveTask(ctx context.Context, task *tasks.RunContext[uploadInstanceImageArchiveTaskPayload]) (uploadInstanceImageArchiveTaskResult, error) {
	payload := task.Payload()
	instance := strings.TrimSpace(payload.Instance)
	archivePath := strings.TrimSpace(payload.ArchivePath)
	imageRef := strings.TrimSpace(payload.ImageRef)
	if instance == "" || instance == "local" {
		return uploadInstanceImageArchiveTaskResult{}, fmt.Errorf("instance is required")
	}
	if archivePath == "" {
		return uploadInstanceImageArchiveTaskResult{}, fmt.Errorf("archive path is empty")
	}
	if imageRef == "" {
		return uploadInstanceImageArchiveTaskResult{}, fmt.Errorf("image ref is empty")
	}
	absPath, err := filepath.Abs(archivePath)
	if err != nil {
		return uploadInstanceImageArchiveTaskResult{}, fmt.Errorf("resolve archive path: %w", err)
	}
	if err := task.Update(5, "validating image archive", map[string]any{
		"instance":     instance,
		"image_ref":    imageRef,
		"archive_path": absPath,
	}); err != nil {
		return uploadInstanceImageArchiveTaskResult{}, err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return uploadInstanceImageArchiveTaskResult{}, fmt.Errorf("stat image archive %s: %w", absPath, err)
	}
	if !info.Mode().IsRegular() {
		return uploadInstanceImageArchiveTaskResult{}, fmt.Errorf("image archive %s is not a regular file", absPath)
	}
	total := uint64(info.Size())

	if err := task.Update(10, "connecting to instance", map[string]any{
		"instance":  instance,
		"image_ref": imageRef,
	}); err != nil {
		return uploadInstanceImageArchiveTaskResult{}, err
	}
	if b.protosClient == nil || b.protosClient.P2PManager == nil {
		return uploadInstanceImageArchiveTaskResult{}, fmt.Errorf("cannot upload instance image archive: p2p manager is not configured")
	}

	uploadStartedAt := time.Now()
	if err := task.Update(15, "uploading image archive", imageArchiveUploadDetails(instance, imageRef, absPath, 0, total)); err != nil {
		return uploadInstanceImageArchiveTaskResult{}, err
	}
	uploadID := "task-" + task.Task().ID
	lastLiveProgress := -1
	lastLiveAt := time.Time{}
	result, err := b.protosClient.P2PManager.UploadImageArchiveToInstance(ctx, p2p.ImageArchiveUploadRequest{
		Instance:    instance,
		ArchivePath: absPath,
		ImageRef:    imageRef,
		UploadID:    uploadID,
		Progress: func(progress p2p.ImageArchiveUploadProgress) error {
			if progress.Importing {
				details := addImageArchiveUploadTiming(imageArchiveUploadDetails(instance, imageRef, absPath, progress.BytesUploaded, total), uploadStartedAt, 0, 0)
				return task.Update(90, "importing image archive", details)
			}
			progressValue := imageArchiveUploadProgress(progress.BytesUploaded, total)
			if progressValue == lastLiveProgress && time.Since(lastLiveAt) < 5*time.Second {
				return nil
			}
			lastLiveProgress = progressValue
			lastLiveAt = time.Now()
			details := addImageArchiveUploadTiming(imageArchiveUploadDetails(instance, imageRef, absPath, progress.BytesUploaded, total), uploadStartedAt, progress.ChunkBytes, progress.ChunkDuration)
			return task.Progress(progressValue, "upload in progress", details)
		},
	})
	if err != nil {
		return uploadInstanceImageArchiveTaskResult{}, fmt.Errorf("upload image archive to instance %q: %w", instance, err)
	}
	if err := task.Update(95, "image archive loaded", map[string]any{
		"instance":           instance,
		"image_ref":          result.ImageRef,
		"target_digest":      result.TargetDigest,
		"platform":           result.Platform,
		"bytes_uploaded":     result.BytesUploaded,
		"archive_size_bytes": result.ArchiveSizeBytes,
	}); err != nil {
		return uploadInstanceImageArchiveTaskResult{}, err
	}
	return uploadInstanceImageArchiveTaskResult{
		Instance:         instance,
		ImageRef:         result.ImageRef,
		TargetDigest:     result.TargetDigest,
		Platform:         result.Platform,
		BytesUploaded:    result.BytesUploaded,
		ArchiveSizeBytes: result.ArchiveSizeBytes,
	}, nil
}

func addImageArchiveUploadTiming(details map[string]any, startedAt time.Time, chunkBytes int, chunkDuration time.Duration) map[string]any {
	if details == nil {
		details = map[string]any{}
	}
	if !startedAt.IsZero() {
		elapsed := time.Since(startedAt)
		details["elapsed_ms"] = elapsed.Milliseconds()
		if elapsed > 0 {
			if uploaded, ok := details["bytes_uploaded"].(uint64); ok {
				details["bytes_per_second"] = uint64(float64(uploaded) / elapsed.Seconds())
			}
		}
	}
	if chunkBytes > 0 {
		details["chunk_bytes"] = chunkBytes
		details["chunk_duration_ms"] = chunkDuration.Milliseconds()
	}
	return details
}

func imageArchiveUploadProgress(received uint64, total uint64) int {
	if total == 0 {
		return 85
	}
	if received > total {
		received = total
	}
	return 15 + int(received*70/total)
}

func imageArchiveUploadDetails(instance string, imageRef string, archivePath string, received uint64, total uint64) map[string]any {
	percent := uint64(100)
	if total > 0 && received < total {
		percent = received * 100 / total
	}
	return map[string]any{
		"instance":           instance,
		"image_ref":          imageRef,
		"archive_path":       archivePath,
		"bytes_uploaded":     received,
		"archive_size_bytes": total,
		"percent":            percent,
	}
}
