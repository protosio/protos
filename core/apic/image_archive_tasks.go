package apic

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	pbApic "github.com/protosio/protos/apic/proto"
	p2pproto "github.com/protosio/protos/internal/p2p/proto"
	"github.com/protosio/protos/internal/tasks"
)

const (
	InstanceImageArchiveUploadTaskStream = "instances.image_archive.upload"

	taskSubjectInstanceImageArchive = "instance_image_archive"

	defaultInstanceImageArchiveChunkSize = 1024 * 1024
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

	task, err := tasks.Enqueue(manager, tasks.EnqueueOptions[uploadInstanceImageArchiveTaskPayload]{
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

	file, err := os.Open(absPath)
	if err != nil {
		return uploadInstanceImageArchiveTaskResult{}, fmt.Errorf("open image archive %s: %w", absPath, err)
	}
	defer file.Close()

	if err := task.Update(10, "connecting to instance", map[string]any{
		"instance":  instance,
		"image_ref": imageRef,
	}); err != nil {
		return uploadInstanceImageArchiveTaskResult{}, err
	}
	client, err := b.instanceAdminClient(ctx, instance, "upload instance image archive")
	if err != nil {
		return uploadInstanceImageArchiveTaskResult{}, err
	}

	uploadStartedAt := time.Now()
	if err := task.Update(15, "uploading image archive", imageArchiveUploadDetails(instance, imageRef, absPath, 0, total)); err != nil {
		return uploadInstanceImageArchiveTaskResult{}, err
	}
	uploadID := "task-" + task.Task().ID
	buf := make([]byte, defaultInstanceImageArchiveChunkSize)
	var offset uint64
	lastLiveProgress := -1
	lastLiveAt := time.Time{}
	for {
		if err := ctx.Err(); err != nil {
			return uploadInstanceImageArchiveTaskResult{}, err
		}
		n, readErr := file.Read(buf)
		eof := readErr == io.EOF
		if readErr != nil && !eof {
			return uploadInstanceImageArchiveTaskResult{}, fmt.Errorf("read image archive %s: %w", absPath, readErr)
		}
		if n == 0 && !eof {
			continue
		}
		expectedReceived := offset + uint64(n)
		if eof {
			if err := task.Update(90, "importing image archive", addImageArchiveUploadTiming(imageArchiveUploadDetails(instance, imageRef, absPath, expectedReceived, total), uploadStartedAt, 0, 0)); err != nil {
				return uploadInstanceImageArchiveTaskResult{}, err
			}
		}
		callTimeout := 2 * time.Minute
		if eof {
			callTimeout = 5 * time.Minute
		}
		callCtx, cancel := context.WithTimeout(ctx, callTimeout)
		chunkStartedAt := time.Now()
		resp, err := client.UploadImageArchiveChunk(callCtx, &p2pproto.UploadImageArchiveChunkRequest{
			UploadId: uploadID,
			ImageRef: imageRef,
			Offset:   offset,
			Data:     append([]byte(nil), buf[:n]...),
			Eof:      eof,
		})
		chunkDuration := time.Since(chunkStartedAt)
		cancel()
		if err != nil {
			return uploadInstanceImageArchiveTaskResult{}, fmt.Errorf("upload image archive chunk to instance %q: %w", instance, err)
		}
		offset = resp.GetReceivedBytes()
		if !eof {
			progress := imageArchiveUploadProgress(offset, total)
			if progress != lastLiveProgress || time.Since(lastLiveAt) >= 5*time.Second {
				lastLiveProgress = progress
				lastLiveAt = time.Now()
				details := addImageArchiveUploadTiming(imageArchiveUploadDetails(instance, imageRef, absPath, offset, total), uploadStartedAt, n, chunkDuration)
				if err := task.Progress(progress, "upload in progress", details); err != nil {
					return uploadInstanceImageArchiveTaskResult{}, err
				}
			}
			continue
		}
		if !resp.GetLoaded() {
			return uploadInstanceImageArchiveTaskResult{}, fmt.Errorf("image archive upload to %s reached EOF without loading image", instance)
		}
		if err := task.Update(95, "image archive loaded", map[string]any{
			"instance":           instance,
			"image_ref":          resp.GetImageRef(),
			"target_digest":      resp.GetTargetDigest(),
			"platform":           resp.GetPlatform(),
			"bytes_uploaded":     offset,
			"archive_size_bytes": total,
		}); err != nil {
			return uploadInstanceImageArchiveTaskResult{}, err
		}
		return uploadInstanceImageArchiveTaskResult{
			Instance:         instance,
			ImageRef:         resp.GetImageRef(),
			TargetDigest:     resp.GetTargetDigest(),
			Platform:         resp.GetPlatform(),
			BytesUploaded:    offset,
			ArchiveSizeBytes: total,
		}, nil
	}
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
