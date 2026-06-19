package runtime

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/content"
	"github.com/containerd/containerd/v2/core/images"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/errdefs"
	"github.com/containerd/platforms"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/imageregistry"
)

func (cdp *containerdPlatform) DescribeImage(ctx context.Context, imageRef string) (string, string, string, map[string]string, bool, error) {
	imageRef = strings.TrimSpace(imageRef)
	if imageRef == "" {
		return "", "", "", nil, false, fmt.Errorf("image ref is empty")
	}
	ctx = namespacesContext(ctx)

	image, err := cdp.client.GetImage(ctx, imageRef)
	if err != nil {
		if errdefs.IsNotFound(err) || strings.Contains(strings.ToLower(err.Error()), "not found") {
			return "", "", "", nil, false, nil
		}
		return "", "", "", nil, false, fmt.Errorf("failed to describe image %s: %w", imageRef, err)
	}

	return image.Name(), image.Target().Digest.String(), platforms.DefaultString(), copyLabels(image.Labels()), true, nil
}

func (cdp *containerdPlatform) GetImageContent(ctx context.Context, imageRef string) (imageregistry.ImageContent, bool, error) {
	imageRef = strings.TrimSpace(imageRef)
	if imageRef == "" {
		return imageregistry.ImageContent{}, false, fmt.Errorf("image ref is empty")
	}
	ctx = namespacesContext(ctx)

	image, err := cdp.client.GetImage(ctx, imageRef)
	if err != nil {
		if errdefs.IsNotFound(err) || strings.Contains(strings.ToLower(err.Error()), "not found") {
			return imageregistry.ImageContent{}, false, nil
		}
		return imageregistry.ImageContent{}, false, fmt.Errorf("failed to inspect image %s: %w", imageRef, err)
	}

	descriptors, err := cdp.imageContentDescriptors(ctx, image.Target())
	if err != nil {
		return imageregistry.ImageContent{}, false, fmt.Errorf("failed to inspect image content %s: %w", imageRef, err)
	}
	return imageregistry.ImageContent{
		ImageRef:    image.Name(),
		Target:      descriptorFromOCI(image.Target()),
		Platform:    platforms.DefaultString(),
		Labels:      copyLabels(image.Labels()),
		Descriptors: descriptors,
	}, true, nil
}

func (cdp *containerdPlatform) imageContentDescriptors(ctx context.Context, target ocispec.Descriptor) ([]imageregistry.Descriptor, error) {
	cs := cdp.client.ContentStore()
	seen := map[string]struct{}{}
	var descriptors []imageregistry.Descriptor

	collector := images.HandlerFunc(func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		key := desc.Digest.String()
		if _, found := seen[key]; found {
			return nil, images.ErrStopHandler
		}
		seen[key] = struct{}{}
		if desc.Size <= 0 {
			info, err := cs.Info(ctx, desc.Digest)
			if err == nil {
				desc.Size = info.Size
			}
		}
		descriptors = append(descriptors, descriptorFromOCI(desc))
		return nil, nil
	})
	children := platformImageChildrenHandler(cs)

	if err := images.Walk(ctx, images.Handlers(collector, children), target); err != nil {
		return nil, err
	}
	return descriptors, nil
}

func platformImageChildrenHandler(cs content.Store) images.HandlerFunc {
	children := images.ChildrenHandler(cs)
	children = images.FilterPlatforms(children, platforms.DefaultStrict())
	children = images.LimitManifests(children, platforms.DefaultStrict(), 1)
	return children
}

func (cdp *containerdPlatform) MissingImageContent(ctx context.Context, descriptors []imageregistry.Descriptor) ([]imageregistry.Descriptor, error) {
	ctx = namespacesContext(ctx)
	return cdp.missingImageContent(ctx, descriptors)
}

func (cdp *containerdPlatform) missingImageContent(ctx context.Context, descriptors []imageregistry.Descriptor) ([]imageregistry.Descriptor, error) {
	cs := cdp.client.ContentStore()
	missing := make([]imageregistry.Descriptor, 0)
	seen := map[string]struct{}{}
	for _, descriptor := range descriptors {
		desc, err := descriptorToOCI(descriptor)
		if err != nil {
			return nil, err
		}
		if _, found := seen[desc.Digest.String()]; found {
			continue
		}
		seen[desc.Digest.String()] = struct{}{}
		ready, err := contentBlobReady(ctx, cs, desc)
		if err != nil {
			return nil, fmt.Errorf("failed to inspect content blob %s: %w", desc.Digest, err)
		}
		if !ready {
			missing = append(missing, descriptor)
		}
	}
	return missing, nil
}

func (cdp *containerdPlatform) OpenImageBlob(ctx context.Context, digest string) (imageregistry.ImageBlobReader, error) {
	digest = strings.TrimSpace(digest)
	if digest == "" {
		return nil, fmt.Errorf("digest is empty")
	}
	desc, err := descriptorToOCI(imageregistry.Descriptor{Digest: digest})
	if err != nil {
		return nil, err
	}
	ctx = namespacesContext(ctx)

	reader, err := cdp.client.ContentStore().ReaderAt(ctx, desc)
	if err != nil {
		return nil, err
	}

	if reader.Size() < 0 {
		_ = reader.Close()
		return nil, fmt.Errorf("content blob %s has unknown size", digest)
	}
	return reader, nil
}

func (cdp *containerdPlatform) ImportImageBlob(ctx context.Context, descriptor imageregistry.Descriptor, blobPath string) error {
	ctx = namespacesContext(ctx)
	return cdp.importImageBlob(ctx, descriptor, blobPath)
}

func (cdp *containerdPlatform) importImageBlob(ctx context.Context, descriptor imageregistry.Descriptor, blobPath string) error {
	desc, err := descriptorToOCI(descriptor)
	if err != nil {
		return err
	}
	if desc.Size <= 0 {
		return fmt.Errorf("content blob %s has invalid size %d", desc.Digest, desc.Size)
	}
	cs := cdp.client.ContentStore()
	ready, err := contentBlobReady(ctx, cs, desc)
	if err != nil {
		return fmt.Errorf("failed to inspect content blob %s: %w", desc.Digest, err)
	}
	if ready {
		return nil
	}
	if err := cs.Delete(ctx, desc.Digest); err != nil && !errdefs.IsNotFound(err) {
		return fmt.Errorf("failed to remove unreadable content blob %s before import: %w", desc.Digest, err)
	}

	file, err := os.Open(blobPath)
	if err != nil {
		return err
	}
	defer file.Close()

	ref := "protos-p2p-" + strings.ReplaceAll(desc.Digest.String(), ":", "-")
	if err := content.WriteBlob(ctx, cs, ref, file, desc); err != nil {
		if errdefs.IsAlreadyExists(err) {
			return nil
		}
		return fmt.Errorf("failed to import content blob %s: %w", desc.Digest, err)
	}
	return nil
}

func contentBlobReady(ctx context.Context, cs content.Store, desc ocispec.Descriptor) (bool, error) {
	reader, err := cs.ReaderAt(ctx, desc)
	if err != nil {
		if errdefs.IsNotFound(err) || strings.Contains(strings.ToLower(err.Error()), "not found") {
			return false, nil
		}
		return false, err
	}
	defer reader.Close()

	if desc.Size > 0 && reader.Size() != desc.Size {
		return false, nil
	}
	return true, nil
}

func (cdp *containerdPlatform) EnsureImageContent(ctx context.Context, request imageregistry.ImageContentImport, progress func(int, string, any) error) error {
	imageRef := strings.TrimSpace(request.ImageRef)
	if imageRef == "" {
		return fmt.Errorf("image ref is empty")
	}
	if len(request.Descriptors) == 0 {
		return fmt.Errorf("image content descriptor list is empty")
	}
	targetDesc, err := descriptorToOCI(request.Target)
	if err != nil {
		return err
	}

	ctx = namespacesContext(ctx)
	leaseCtx, releaseLease, err := cdp.client.WithLease(ctx)
	if err != nil {
		return fmt.Errorf("failed to create containerd lease for image content import %s: %w", imageRef, err)
	}
	ctx = leaseCtx
	defer func() {
		if err := releaseLease(context.WithoutCancel(ctx)); err != nil {
			log.Warnf("failed to release containerd image content import lease for %s: %v", imageRef, err)
		}
	}()
	labels := copyLabels(request.Labels)
	if labels == nil {
		labels = map[string]string{}
	}

	for i, blob := range request.Blobs {
		if strings.TrimSpace(blob.Path) == "" {
			return fmt.Errorf("downloaded image blob path is empty for %s", blob.Descriptor.Digest)
		}
		reportImageContentProgress(progress, 64+(i*4)/max(1, len(request.Blobs)), "importing image blob", map[string]any{
			"image":           imageRef,
			"completed_blobs": i,
			"total_blobs":     len(request.Blobs),
			"digest":          blob.Descriptor.Digest,
			"size_bytes":      blob.Descriptor.SizeBytes,
		})
		startedAt := time.Now()
		if err := cdp.importImageBlob(ctx, blob.Descriptor, blob.Path); err != nil {
			return fmt.Errorf("failed to import content blob %s for image %s: %w", blob.Descriptor.Digest, imageRef, err)
		}
		duration := time.Since(startedAt)
		log.Infof("imported image content blob %s for image %s: size=%d duration=%s throughput=%dB/s", blob.Descriptor.Digest, imageRef, blob.Descriptor.SizeBytes, duration.Round(time.Millisecond), imageContentBytesPerSecond(blob.Descriptor.SizeBytes, duration))
		reportImageContentProgress(progress, 64+((i+1)*4)/max(1, len(request.Blobs)), "imported image blob", map[string]any{
			"image":            imageRef,
			"completed_blobs":  i + 1,
			"total_blobs":      len(request.Blobs),
			"digest":           blob.Descriptor.Digest,
			"size_bytes":       blob.Descriptor.SizeBytes,
			"duration_ms":      duration.Milliseconds(),
			"bytes_per_second": imageContentBytesPerSecond(blob.Descriptor.SizeBytes, duration),
		})
	}
	reportImageContentProgress(progress, 69, "verifying image content", map[string]any{"image": imageRef, "descriptors": len(request.Descriptors)})
	verifyStartedAt := time.Now()
	if missing, err := cdp.MissingImageContent(ctx, request.Descriptors); err != nil {
		return err
	} else if len(missing) > 0 {
		return fmt.Errorf("image content import for %s is missing %d blob(s), first missing digest %s", imageRef, len(missing), missing[0].Digest)
	}
	log.Infof("verified image content for %s: descriptors=%d duration=%s", imageRef, len(request.Descriptors), time.Since(verifyStartedAt).Round(time.Millisecond))

	reportImageContentProgress(progress, 70, "linking image content", map[string]string{"image": imageRef})
	linkStartedAt := time.Now()
	if err := cdp.labelImageContentChildren(ctx, targetDesc); err != nil {
		return fmt.Errorf("failed to link image content %s for containerd GC: %w", imageRef, err)
	}
	log.Infof("linked image content graph for %s: duration=%s", imageRef, time.Since(linkStartedAt).Round(time.Millisecond))

	reportImageContentProgress(progress, 70, "creating image record", map[string]string{"image": imageRef})
	recordStartedAt := time.Now()
	record := images.Image{Name: imageRef, Target: targetDesc, Labels: labels}
	previous, hadPrevious, err := cdp.createOrUpdateImageRecord(ctx, record)
	if err != nil {
		return err
	}
	log.Infof("created image record for %s: duration=%s", imageRef, time.Since(recordStartedAt).Round(time.Millisecond))

	reportImageContentProgress(progress, 72, "unpacking image", map[string]string{"image": imageRef})
	unpackStartedAt := time.Now()
	createdImage := client.NewImageWithPlatform(cdp.client, record, platforms.DefaultStrict())
	if err := cdp.ensureImageReadyForSandbox(ctx, createdImage); err != nil {
		cdp.restoreImageRecordAfterFailedImport(ctx, imageRef, targetDesc, previous, hadPrevious)
		return fmt.Errorf("failed to prepare image %s for sandbox after content import: %w", imageRef, err)
	}
	unpackDuration := time.Since(unpackStartedAt)
	log.Infof("unpacked and verified image %s with snapshotter %s: duration=%s", imageRef, cdp.snapshotter, unpackDuration.Round(time.Millisecond))
	reportImageContentProgress(progress, 73, "unpacked image", map[string]any{"image": imageRef, "snapshotter": cdp.snapshotter, "duration_ms": unpackDuration.Milliseconds()})
	reportImageContentProgress(progress, 74, "verified image snapshot", map[string]string{"image": imageRef})
	reportImageContentProgress(progress, 74, "verifying image record", map[string]string{"image": imageRef})
	recordVerifyStartedAt := time.Now()
	if _, err := cdp.imageContentDescriptors(ctx, targetDesc); err != nil {
		cdp.restoreImageRecordAfterFailedImport(ctx, imageRef, targetDesc, previous, hadPrevious)
		return fmt.Errorf("failed to verify image ref %s after content import: %w", imageRef, err)
	}
	log.Infof("verified image record for %s: duration=%s", imageRef, time.Since(recordVerifyStartedAt).Round(time.Millisecond))
	return nil
}

func (cdp *containerdPlatform) labelImageContentChildren(ctx context.Context, target ocispec.Descriptor) error {
	cs := cdp.client.ContentStore()
	return images.Walk(ctx, images.SetChildrenLabels(cs, platformImageChildrenHandler(cs)), target)
}

func reportImageContentProgress(progress func(int, string, any) error, value int, message string, details any) {
	if progress == nil {
		return
	}
	if err := progress(value, message, details); err != nil {
		log.Debugf("failed to report image content progress: %s", err.Error())
	}
}

func imageContentBytesPerSecond(bytes uint64, duration time.Duration) uint64 {
	if bytes == 0 || duration <= 0 {
		return 0
	}
	return uint64(float64(bytes) / duration.Seconds())
}

func (cdp *containerdPlatform) createOrUpdateImageRecord(ctx context.Context, record images.Image) (images.Image, bool, error) {
	_, err := cdp.client.ImageService().Create(ctx, record)
	if err == nil {
		return images.Image{}, false, nil
	}
	if !errdefs.IsAlreadyExists(err) {
		return images.Image{}, false, fmt.Errorf("failed to create image ref %s: %w", record.Name, err)
	}

	previous, getErr := cdp.client.ImageService().Get(ctx, record.Name)
	if getErr != nil {
		return images.Image{}, false, fmt.Errorf("failed to retrieve existing image ref %s: %w", record.Name, getErr)
	}
	_, updateErr := cdp.client.ImageService().Update(ctx, record, "target", "labels")
	if updateErr != nil {
		return images.Image{}, false, fmt.Errorf("failed to update image ref %s: %w", record.Name, updateErr)
	}
	return previous, true, nil
}

func (cdp *containerdPlatform) restoreImageRecordAfterFailedImport(ctx context.Context, imageRef string, target ocispec.Descriptor, previous images.Image, hadPrevious bool) {
	if hadPrevious {
		if _, err := cdp.client.ImageService().Update(ctx, previous, "target", "labels"); err != nil {
			log.Warnf("failed to restore image ref %s after failed content import: %v", imageRef, err)
		}
		return
	}
	if err := cdp.client.ImageService().Delete(ctx, imageRef, images.DeleteTarget(&target), images.SynchronousDelete()); err != nil && !errdefs.IsNotFound(err) {
		log.Warnf("failed to remove partial image ref %s after failed content import: %v", imageRef, err)
	}
}

func (cdp *containerdPlatform) CreateImageFromContent(ctx context.Context, imageRef string, target imageregistry.Descriptor, labels map[string]string) error {
	imageRef = strings.TrimSpace(imageRef)
	if imageRef == "" {
		return fmt.Errorf("image ref is empty")
	}
	targetDesc, err := descriptorToOCI(target)
	if err != nil {
		return err
	}
	ctx = namespacesContext(ctx)
	leaseCtx, releaseLease, err := cdp.client.WithLease(ctx)
	if err != nil {
		return fmt.Errorf("failed to create containerd lease for image ref %s: %w", imageRef, err)
	}
	ctx = leaseCtx
	defer func() {
		if err := releaseLease(context.WithoutCancel(ctx)); err != nil {
			log.Warnf("failed to release containerd image ref lease for %s: %v", imageRef, err)
		}
	}()
	if labels == nil {
		labels = map[string]string{}
	}
	if err := cdp.labelImageContentChildren(ctx, targetDesc); err != nil {
		return fmt.Errorf("failed to link image content %s for containerd GC: %w", imageRef, err)
	}

	record := images.Image{Name: imageRef, Target: targetDesc, Labels: labels}
	created, err := cdp.client.ImageService().Create(ctx, record)
	if err != nil {
		if !errdefs.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create image ref %s: %w", imageRef, err)
		}
		created, err = cdp.client.ImageService().Update(ctx, record, "target", "labels")
		if err != nil {
			return fmt.Errorf("failed to update image ref %s: %w", imageRef, err)
		}
	}

	image := client.NewImageWithPlatform(cdp.client, created, platforms.DefaultStrict())
	if err := cdp.ensureImageReadyForSandbox(ctx, image); err != nil {
		return fmt.Errorf("failed to prepare image %s for sandbox after content import: %w", imageRef, err)
	}
	return nil
}

func (cdp *containerdPlatform) LoadImageArchive(ctx context.Context, archivePath string, imageRef string) (imageregistry.LoadedImage, error) {
	archivePath = strings.TrimSpace(archivePath)
	if archivePath == "" {
		return imageregistry.LoadedImage{}, fmt.Errorf("image archive path is empty")
	}
	archivePath, err := filepath.Abs(archivePath)
	if err != nil {
		return imageregistry.LoadedImage{}, err
	}
	file, err := os.Open(archivePath)
	if err != nil {
		return imageregistry.LoadedImage{}, err
	}
	defer file.Close()

	reader, closeReader, err := maybeGunzip(file)
	if err != nil {
		return imageregistry.LoadedImage{}, err
	}
	defer closeReader()

	ctx = namespacesContext(ctx)
	importOpts := []client.ImportOpt{
		client.WithImportPlatform(platforms.DefaultStrict()),
		client.WithImageLabels(map[string]string{imageregistry.SourceLabel: imageregistry.SourceLocalTar}),
	}
	imageRef = strings.TrimSpace(imageRef)
	if imageRef != "" {
		importOpts = append(importOpts, client.WithIndexName(imageRef))
	}

	log.Infof("importing image archive into containerd: path=%s image=%s", archivePath, imageRef)
	imported, err := cdp.client.Import(ctx, reader, importOpts...)
	if err != nil {
		return imageregistry.LoadedImage{}, fmt.Errorf("failed to import image archive %s: %w", archivePath, err)
	}
	if len(imported) == 0 {
		return imageregistry.LoadedImage{}, fmt.Errorf("image archive %s imported no images", archivePath)
	}
	target := imported[0].Target
	if imageRef == "" {
		imageRef = imported[0].Name
	} else if image, err := cdp.client.GetImage(ctx, imageRef); err == nil {
		target = image.Target()
	}
	if strings.TrimSpace(imageRef) == "" {
		return imageregistry.LoadedImage{}, fmt.Errorf("image archive %s did not provide an image ref; pass --ref", archivePath)
	}

	if err := cdp.CreateImageFromContent(ctx, imageRef, descriptorFromOCI(target), map[string]string{imageregistry.SourceLabel: imageregistry.SourceLocalTar}); err != nil {
		return imageregistry.LoadedImage{}, err
	}
	log.Infof("imported image archive into containerd: path=%s image=%s digest=%s platform=%s", archivePath, imageRef, target.Digest.String(), platforms.DefaultString())
	return imageregistry.LoadedImage{
		ImageRef:     imageRef,
		TargetDigest: target.Digest.String(),
		Platform:     platforms.DefaultString(),
	}, nil
}

func cleanupLegacyImageArchives() {
	archiveDir := filepath.Join(config.Get().WorkDir, "image-archives")
	if strings.TrimSpace(config.Get().WorkDir) == "" || archiveDir == "/" {
		return
	}
	if err := os.RemoveAll(archiveDir); err != nil {
		log.Warnf("failed to clean legacy image archive cache %s: %v", archiveDir, err)
	}
}

func namespacesContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return namespaces.WithNamespace(ctx, protosNamespace)
}

func descriptorFromOCI(desc ocispec.Descriptor) imageregistry.Descriptor {
	size := uint64(0)
	if desc.Size > 0 {
		size = uint64(desc.Size)
	}
	platform := ""
	if desc.Platform != nil {
		platform = platforms.Format(*desc.Platform)
	}
	return imageregistry.Descriptor{
		MediaType:   desc.MediaType,
		Digest:      desc.Digest.String(),
		SizeBytes:   size,
		Platform:    platform,
		Annotations: copyLabels(desc.Annotations),
	}
}

func descriptorToOCI(desc imageregistry.Descriptor) (ocispec.Descriptor, error) {
	dgst, err := digestFromString(desc.Digest)
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	size := int64(desc.SizeBytes)
	if desc.SizeBytes > uint64(1<<63-1) {
		return ocispec.Descriptor{}, fmt.Errorf("descriptor %s is too large: %d bytes", desc.Digest, desc.SizeBytes)
	}
	out := ocispec.Descriptor{
		MediaType:   strings.TrimSpace(desc.MediaType),
		Digest:      dgst,
		Size:        size,
		Annotations: copyLabels(desc.Annotations),
	}
	if strings.TrimSpace(desc.Platform) != "" {
		platform, err := platforms.Parse(desc.Platform)
		if err != nil {
			return ocispec.Descriptor{}, fmt.Errorf("invalid descriptor platform %q: %w", desc.Platform, err)
		}
		out.Platform = &platform
	}
	return out, nil
}

func digestFromString(value string) (digest.Digest, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("descriptor digest is empty")
	}
	dgst, err := digest.Parse(value)
	if err != nil {
		return "", fmt.Errorf("invalid descriptor digest %q: %w", value, err)
	}
	return dgst, nil
}

func copyLabels(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func maybeGunzip(file *os.File) (io.Reader, func(), error) {
	var header [2]byte
	n, err := file.Read(header[:])
	if err != nil && err != io.EOF {
		return nil, func() {}, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, func() {}, err
	}
	if n == 2 && header[0] == 0x1f && header[1] == 0x8b {
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, func() {}, fmt.Errorf("failed to read gzip image archive: %w", err)
		}
		return gzipReader, func() { _ = gzipReader.Close() }, nil
	}
	return file, func() {}, nil
}
