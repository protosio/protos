package runtime

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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

const maxServedImageBlobChunkBytes uint64 = 4 << 20

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
	children := images.ChildrenHandler(cs)
	children = images.FilterPlatforms(children, platforms.DefaultStrict())
	children = images.LimitManifests(children, platforms.DefaultStrict(), 1)

	if err := images.Dispatch(ctx, images.Handlers(collector, children), nil, target); err != nil {
		return nil, err
	}
	return descriptors, nil
}

func (cdp *containerdPlatform) MissingImageContent(ctx context.Context, descriptors []imageregistry.Descriptor) ([]imageregistry.Descriptor, error) {
	ctx = namespacesContext(ctx)
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
		exists, err := content.Exists(ctx, cs, desc)
		if err != nil {
			return nil, fmt.Errorf("failed to inspect content blob %s: %w", desc.Digest, err)
		}
		if !exists {
			missing = append(missing, descriptor)
		}
	}
	return missing, nil
}

func (cdp *containerdPlatform) ReadImageBlobChunk(ctx context.Context, digest string, offset uint64, length uint64) ([]byte, bool, error) {
	digest = strings.TrimSpace(digest)
	if digest == "" {
		return nil, false, fmt.Errorf("digest is empty")
	}
	if length == 0 || length > maxServedImageBlobChunkBytes {
		length = maxServedImageBlobChunkBytes
	}
	desc, err := descriptorToOCI(imageregistry.Descriptor{Digest: digest})
	if err != nil {
		return nil, false, err
	}
	ctx = namespacesContext(ctx)

	reader, err := cdp.client.ContentStore().ReaderAt(ctx, desc)
	if err != nil {
		return nil, false, err
	}
	defer reader.Close()

	size := reader.Size()
	if size < 0 {
		return nil, false, fmt.Errorf("content blob %s has unknown size", digest)
	}
	if offset >= uint64(size) {
		return nil, true, nil
	}
	remaining := uint64(size) - offset
	if length > remaining {
		length = remaining
	}

	data := make([]byte, int(length))
	n, err := reader.ReadAt(data, int64(offset))
	if err != nil && err != io.EOF {
		return nil, false, err
	}
	data = data[:n]
	eof := offset+uint64(n) >= uint64(size)
	return data, eof, nil
}

func (cdp *containerdPlatform) ImportImageBlob(ctx context.Context, descriptor imageregistry.Descriptor, blobPath string) error {
	desc, err := descriptorToOCI(descriptor)
	if err != nil {
		return err
	}
	if desc.Size <= 0 {
		return fmt.Errorf("content blob %s has invalid size %d", desc.Digest, desc.Size)
	}
	ctx = namespacesContext(ctx)
	cs := cdp.client.ContentStore()
	exists, err := content.Exists(ctx, cs, desc)
	if err != nil {
		return fmt.Errorf("failed to inspect content blob %s: %w", desc.Digest, err)
	}
	if exists {
		return nil
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
	if labels == nil {
		labels = map[string]string{}
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
	if err := image.Unpack(ctx, cdp.snapshotter); err != nil && !errdefs.IsAlreadyExists(err) {
		return fmt.Errorf("failed to unpack image ref %s after content import: %w", imageRef, err)
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
