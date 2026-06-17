package p2p

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/protosio/protos/internal/imageregistry"
	p2pproto "github.com/protosio/protos/internal/p2p/proto"
)

const (
	imageBlobChunkBytes    uint64 = 1 << 20
	imageContentTimeout           = 30 * time.Second
	imageBlobChunkTimeout         = 30 * time.Second
	imageBlobImportTimeout        = 5 * time.Minute
	imageCreateTimeout            = 5 * time.Minute
	imagePeerRefreshBudget        = 45 * time.Second
)

type imageContentCandidate struct {
	peerID  string
	client  *Client
	content imageregistry.ImageContent
}

func (p2p *P2P) ResolveImage(ctx context.Context, imageRef string) error {
	imageRef = strings.TrimSpace(imageRef)
	if imageRef == "" {
		return fmt.Errorf("image ref is empty")
	}
	if p2p == nil || p2p.imageManager == nil {
		return fmt.Errorf("image manager is not configured")
	}

	selection := p2p.currentImageResolutionClients()
	candidates := p2p.getImageContentCandidates(ctx, imageRef, selection.clients)
	if len(candidates) == 0 && selection.disconnectedKnownPeerCount > 0 {
		log.Infof("image %s is not available from %d connected image-capable peer(s); refreshing %d disconnected known peer(s) before registry fallback", imageRef, len(selection.clients), selection.disconnectedKnownPeerCount)
		selection = p2p.imageResolutionClients(ctx, imagePeerRefreshBudget)
		candidates = p2p.getImageContentCandidates(ctx, imageRef, selection.clients)
	}
	if len(candidates) == 0 {
		if len(selection.clients) == 0 {
			return fmt.Errorf(
				"no connected image-capable peers can provide image %s (known=%d disconnected=%d)",
				imageRef,
				selection.knownPeerCount,
				selection.disconnectedKnownPeerCount,
			)
		}
		return fmt.Errorf(
			"no connected peers have image %s (queried=%d known=%d disconnected=%d)",
			imageRef,
			len(selection.clients),
			selection.knownPeerCount,
			selection.disconnectedKnownPeerCount,
		)
	}
	return p2p.resolveImageFromCandidates(ctx, imageRef, candidates)
}

func (p2p *P2P) resolveImageFromCandidates(ctx context.Context, imageRef string, candidates []imageContentCandidate) error {
	var lastErr error
	for _, group := range groupedImageContentCandidates(candidates) {
		content := group[0].content
		missing, err := p2p.imageManager.MissingImageContent(ctx, content.Descriptors)
		if err != nil {
			lastErr = err
			log.Warnf("failed to inspect local image content for %s: %v", imageRef, err)
			continue
		}
		if len(missing) > 0 {
			err = p2p.downloadMissingImageBlobs(ctx, group, missing)
			if err != nil {
				lastErr = err
				log.Warnf("failed to download %d missing image blob(s) for %s from %d peer(s): %v", len(missing), imageRef, len(group), err)
				continue
			}
		}

		createCtx, cancel := context.WithTimeout(ctx, imageCreateTimeout)
		err = p2p.imageManager.CreateImageFromContent(createCtx, imageRef, content.Target, imageSourceLabels(content.Labels))
		cancel()
		if err != nil {
			lastErr = err
			log.Warnf("failed to create image %s from peer content: %v", imageRef, err)
			continue
		}

		log.Infof("resolved image %s from %d Protos peer(s), downloaded %d missing content blob(s)", imageRef, len(group), len(missing))
		return nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no usable image content candidates")
	}
	return fmt.Errorf("failed to resolve image %s from peers: %w", imageRef, lastErr)
}

func (p2p *P2P) getImageContentCandidates(ctx context.Context, imageRef string, clients map[string]*Client) []imageContentCandidate {
	var wg sync.WaitGroup
	out := make(chan imageContentCandidate, len(clients))

	for peerID, client := range clients {
		if client == nil || client.ImagesClient == nil {
			continue
		}
		wg.Add(1)
		go func(peerID string, client *Client) {
			defer wg.Done()

			contentCtx, cancel := context.WithTimeout(ctx, imageContentTimeout)
			defer cancel()
			resp, err := client.GetImageContent(contentCtx, &p2pproto.GetImageContentRequest{ImageRef: imageRef})
			if err != nil {
				log.Debugf("peer %s cannot describe image content %s: %v", peerID, imageRef, err)
				return
			}
			if !resp.GetFound() {
				return
			}
			content := imageContentFromProto(resp)
			if err := validateImageContent(content); err != nil {
				log.Debugf("peer %s returned invalid content metadata for image %s: %v", peerID, imageRef, err)
				return
			}
			out <- imageContentCandidate{peerID: peerID, client: client, content: content}
		}(peerID, client)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	var candidates []imageContentCandidate
	for candidate := range out {
		candidates = append(candidates, candidate)
	}
	return candidates
}

func groupedImageContentCandidates(candidates []imageContentCandidate) [][]imageContentCandidate {
	byTarget := map[string][]imageContentCandidate{}
	for _, candidate := range candidates {
		key := fmt.Sprintf("%s/%d/%s", candidate.content.Target.Digest, candidate.content.Target.SizeBytes, candidate.content.Target.MediaType)
		byTarget[key] = append(byTarget[key], candidate)
	}

	groups := make([][]imageContentCandidate, 0, len(byTarget))
	for _, group := range byTarget {
		groups = append(groups, group)
	}
	sort.SliceStable(groups, func(i, j int) bool {
		if len(groups[i]) != len(groups[j]) {
			return len(groups[i]) > len(groups[j])
		}
		return groups[i][0].peerID < groups[j][0].peerID
	})
	return groups
}

func (p2p *P2P) downloadMissingImageBlobs(ctx context.Context, candidates []imageContentCandidate, missing []imageregistry.Descriptor) error {
	if len(candidates) == 0 {
		return fmt.Errorf("no image content candidates")
	}

	jobs := make(chan imageregistry.Descriptor)
	errCh := make(chan error, 1)
	downloadCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	workerCount := len(candidates)
	if workerCount > len(missing) {
		workerCount = len(missing)
	}
	var wg sync.WaitGroup
	for workerID := 0; workerID < workerCount; workerID++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for desc := range jobs {
				if err := p2p.downloadImageBlob(downloadCtx, candidates, workerID, desc); err != nil {
					select {
					case errCh <- err:
						cancel()
					default:
					}
					return
				}
			}
		}(workerID)
	}

	go func() {
		defer close(jobs)
		for _, desc := range missing {
			select {
			case <-downloadCtx.Done():
				return
			case jobs <- desc:
			}
		}
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p2p *P2P) downloadImageBlob(ctx context.Context, candidates []imageContentCandidate, startIndex int, desc imageregistry.Descriptor) error {
	if desc.Digest == "" {
		return fmt.Errorf("image content descriptor has empty digest")
	}
	if desc.SizeBytes > uint64(1<<63-1) {
		return fmt.Errorf("image content blob is too large: digest=%s size=%d", desc.Digest, desc.SizeBytes)
	}

	tmp, err := os.CreateTemp("", "protos-image-blob-*.tmp")
	if err != nil {
		return err
	}
	blobPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(blobPath)
		}
	}()

	for offset := uint64(0); offset < desc.SizeBytes; offset += imageBlobChunkBytes {
		length := minUint64(imageBlobChunkBytes, desc.SizeBytes-offset)
		data, err := fetchImageBlobChunk(ctx, candidates, startIndex, desc.Digest, offset, length)
		if err != nil {
			_ = tmp.Close()
			return err
		}
		if uint64(len(data)) != length {
			_ = tmp.Close()
			return fmt.Errorf("blob %s chunk at offset %d returned %d bytes, want %d", desc.Digest, offset, len(data), length)
		}
		if _, err := tmp.Write(data); err != nil {
			_ = tmp.Close()
			return err
		}
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	importCtx, cancel := context.WithTimeout(ctx, imageBlobImportTimeout)
	err = p2p.imageManager.ImportImageBlob(importCtx, desc, blobPath)
	cancel()
	if err != nil {
		return err
	}
	return nil
}

func fetchImageBlobChunk(ctx context.Context, candidates []imageContentCandidate, startIndex int, digest string, offset uint64, length uint64) ([]byte, error) {
	var errs []error
	for attempt := range candidates {
		candidate := candidates[(startIndex+attempt)%len(candidates)]
		callCtx, cancel := context.WithTimeout(ctx, imageBlobChunkTimeout)
		resp, err := candidate.client.GetImageBlobChunk(callCtx, &p2pproto.GetImageBlobChunkRequest{
			Digest: digest,
			Offset: offset,
			Length: length,
		})
		cancel()
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", candidate.peerID, err))
			continue
		}
		if resp.GetDigest() != digest {
			errs = append(errs, fmt.Errorf("%s: blob digest changed", candidate.peerID))
			continue
		}
		if resp.GetOffset() != offset {
			errs = append(errs, fmt.Errorf("%s: chunk offset changed from %d to %d", candidate.peerID, offset, resp.GetOffset()))
			continue
		}
		return resp.GetData(), nil
	}
	return nil, errors.Join(errs...)
}

func imageContentFromProto(resp *p2pproto.GetImageContentResponse) imageregistry.ImageContent {
	if resp == nil {
		return imageregistry.ImageContent{}
	}
	return imageregistry.ImageContent{
		ImageRef:    strings.TrimSpace(resp.GetImageRef()),
		Target:      imageDescriptorFromProto(resp.GetTarget()),
		Platform:    strings.TrimSpace(resp.GetPlatform()),
		Labels:      copyStringMap(resp.GetLabels()),
		Descriptors: imageDescriptorsFromProto(resp.GetDescriptors()),
	}
}

func imageDescriptorFromProto(desc *p2pproto.ImageContentDescriptor) imageregistry.Descriptor {
	if desc == nil {
		return imageregistry.Descriptor{}
	}
	return imageregistry.Descriptor{
		MediaType:   strings.TrimSpace(desc.GetMediaType()),
		Digest:      strings.TrimSpace(desc.GetDigest()),
		SizeBytes:   desc.GetSizeBytes(),
		Platform:    strings.TrimSpace(desc.GetPlatform()),
		Annotations: copyStringMap(desc.GetAnnotations()),
	}
}

func imageDescriptorsFromProto(descs []*p2pproto.ImageContentDescriptor) []imageregistry.Descriptor {
	out := make([]imageregistry.Descriptor, 0, len(descs))
	for _, desc := range descs {
		out = append(out, imageDescriptorFromProto(desc))
	}
	return out
}

func validateImageContent(content imageregistry.ImageContent) error {
	if content.ImageRef == "" {
		return fmt.Errorf("image ref is empty")
	}
	if err := validateImageDescriptor(content.Target); err != nil {
		return fmt.Errorf("invalid target: %w", err)
	}
	if len(content.Descriptors) == 0 {
		return fmt.Errorf("descriptor list is empty")
	}
	seenTarget := false
	for _, desc := range content.Descriptors {
		if err := validateImageDescriptor(desc); err != nil {
			return err
		}
		if desc.Digest == content.Target.Digest {
			seenTarget = true
		}
	}
	if !seenTarget {
		return fmt.Errorf("descriptor list does not include target %s", content.Target.Digest)
	}
	return nil
}

func validateImageDescriptor(desc imageregistry.Descriptor) error {
	if strings.TrimSpace(desc.Digest) == "" {
		return fmt.Errorf("descriptor digest is empty")
	}
	if desc.SizeBytes == 0 {
		return fmt.Errorf("descriptor %s has zero size", desc.Digest)
	}
	return nil
}

func imageSourceLabels(labels map[string]string) map[string]string {
	out := copyStringMap(labels)
	if out == nil {
		out = map[string]string{}
	}
	out[imageregistry.SourceLabel] = imageregistry.SourceP2P
	return out
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func minUint64(a uint64, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}
