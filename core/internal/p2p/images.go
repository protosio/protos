package p2p

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/protosio/protos/internal/imageregistry"
	p2pproto "github.com/protosio/protos/internal/p2p/proto"
	goproto "google.golang.org/protobuf/proto"
)

const (
	imageBlobStreamChunkBytes        uint64 = 4 << 20
	imageBlobParallelThresholdBytes  uint64 = 8 << 20
	imageBlobStreamMinBytesPerSecond uint64 = 256 << 10
	imageBlobRequestFrameMaxBytes           = 64 << 10
	imageBlobResponseFrameMaxBytes          = 5 << 20
	imageBlobLimitedStreamChunkBytes uint64 = 512 << 10
	imageBlobRangeConcurrency               = 4
	imageBlobDownloadConcurrency            = 3
	imageBlobDataFrameHeaderBytes           = 13
	imageBlobDataFrameFlagEOF               = 1
	imageContentTimeout                     = 30 * time.Second
	imageBlobStreamMinTimeout               = 2 * time.Minute
	imageBlobStreamMaxTimeout               = 15 * time.Minute
	imageCreateTimeout                      = 5 * time.Minute
	imagePeerRefreshBudget                  = 45 * time.Second
	imageBlobProgressInterval               = 2 * time.Second
	imageBlobCandidateAttempts              = 2
	imageBlobLimitedRangeAttempts           = 4
)

type imageContentCandidate struct {
	peerID  string
	client  *Client
	content imageregistry.ImageContent
}

func (p2p *P2P) ResolveImage(ctx context.Context, imageRef string, progress func(int, string, any) error) error {
	imageRef = strings.TrimSpace(imageRef)
	if imageRef == "" {
		return fmt.Errorf("image ref is empty")
	}
	if p2p == nil || p2p.imageManager == nil {
		return fmt.Errorf("image manager is not configured")
	}

	reportImageResolveProgress(progress, 28, "selecting image peers", map[string]string{"image": imageRef})
	selection := p2p.currentImageResolutionClients()
	reportImageResolveProgress(progress, 30, "querying image peers", map[string]any{"image": imageRef, "connected_peers": len(selection.clients)})
	candidates := p2p.getImageContentCandidates(ctx, imageRef, selection.clients)
	if len(candidates) == 0 && selection.disconnectedKnownPeerCount > 0 {
		log.Infof("image %s is not available from %d connected image-capable peer(s); refreshing %d disconnected known peer(s) before registry fallback", imageRef, len(selection.clients), selection.disconnectedKnownPeerCount)
		reportImageResolveProgress(progress, 32, "refreshing image peers", map[string]any{"image": imageRef, "disconnected_peers": selection.disconnectedKnownPeerCount})
		selection = p2p.imageResolutionClients(ctx, imagePeerRefreshBudget)
		reportImageResolveProgress(progress, 35, "querying refreshed image peers", map[string]any{"image": imageRef, "connected_peers": len(selection.clients)})
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
	reportImageResolveProgress(progress, 40, "found image peer content", map[string]any{"image": imageRef, "candidates": len(candidates)})
	return p2p.resolveImageFromCandidates(ctx, imageRef, candidates, progress)
}

func (p2p *P2P) resolveImageFromCandidates(ctx context.Context, imageRef string, candidates []imageContentCandidate, progress func(int, string, any) error) error {
	var lastErr error
	for _, group := range groupedImageContentCandidates(candidates) {
		content := group[0].content
		reportImageResolveProgress(progress, 42, "checking local image content", map[string]any{"image": imageRef, "descriptors": len(content.Descriptors)})
		missing, err := p2p.imageManager.MissingImageContent(ctx, content.Descriptors)
		if err != nil {
			lastErr = err
			log.Warnf("failed to inspect local image content for %s: %v", imageRef, err)
			continue
		}
		var blobs []imageregistry.ImageContentBlob
		cleanupBlobs := func() {}
		if len(missing) > 0 {
			reportImageResolveProgress(progress, 45, "downloading image content", map[string]any{"image": imageRef, "missing_blobs": len(missing), "providers": len(group)})
			blobs, cleanupBlobs, err = p2p.downloadMissingImageBlobs(ctx, group, missing, imageRef, progress)
			if err != nil {
				lastErr = err
				log.Warnf("failed to download %d missing image blob(s) for %s from %d peer(s): %v", len(missing), imageRef, len(group), err)
				continue
			}
		}

		reportImageResolveProgress(progress, 62, "importing image content", map[string]any{"image": imageRef, "downloaded_blobs": len(blobs), "descriptors": len(content.Descriptors)})
		createCtx, cancel := context.WithTimeout(ctx, imageCreateTimeout)
		err = p2p.imageManager.EnsureImageContent(createCtx, imageregistry.ImageContentImport{
			ImageRef:    imageRef,
			Target:      content.Target,
			Labels:      imageSourceLabels(content.Labels),
			Descriptors: content.Descriptors,
			Blobs:       blobs,
		}, progress)
		cancel()
		cleanupBlobs()
		if err != nil {
			lastErr = err
			log.Warnf("failed to create image %s from peer content: %v", imageRef, err)
			continue
		}

		log.Infof("resolved image %s from %d Protos peer(s), downloaded %d missing content blob(s)", imageRef, len(group), len(missing))
		reportImageResolveProgress(progress, 75, "image content ready", map[string]any{"image": imageRef, "downloaded_blobs": len(missing), "providers": len(group)})
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

func (p2p *P2P) downloadMissingImageBlobs(ctx context.Context, candidates []imageContentCandidate, missing []imageregistry.Descriptor, imageRef string, progress func(int, string, any) error) ([]imageregistry.ImageContentBlob, func(), error) {
	if len(candidates) == 0 {
		return nil, func() {}, fmt.Errorf("no image content candidates")
	}

	jobs := make(chan imageregistry.Descriptor)
	results := make(chan imageregistry.ImageContentBlob, len(missing))
	errCh := make(chan error, 1)
	downloadCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	progressTracker := newImageBlobDownloadProgress(imageRef, missing, progress)

	workerCount := p2p.imageBlobDownloadConcurrency(candidates)
	if workerCount > len(missing) {
		workerCount = len(missing)
	}
	var completed atomic.Int32
	var wg sync.WaitGroup
	for workerID := 0; workerID < workerCount; workerID++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for desc := range jobs {
				blob, err := p2p.downloadImageBlob(downloadCtx, candidates, workerID, desc, progressTracker.Add)
				if err != nil {
					select {
					case errCh <- err:
						cancel()
					default:
					}
					return
				}
				done := int(completed.Add(1))
				progressTracker.Complete(desc, done)
				select {
				case results <- blob:
				case <-downloadCtx.Done():
					_ = os.Remove(blob.Path)
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
		close(results)
		close(done)
	}()

	select {
	case <-done:
		blobs := make([]imageregistry.ImageContentBlob, 0, len(missing))
		for blob := range results {
			blobs = append(blobs, blob)
		}
		return blobs, cleanupImageContentBlobs(blobs), nil
	case err := <-errCh:
		cancel()
		<-done
		blobs := drainImageContentBlobResults(results)
		cleanupImageContentBlobs(blobs)()
		return nil, func() {}, err
	case <-ctx.Done():
		cancel()
		<-done
		blobs := drainImageContentBlobResults(results)
		cleanupImageContentBlobs(blobs)()
		return nil, func() {}, ctx.Err()
	}
}

type imageBlobDownloadProgress struct {
	imageRef   string
	totalBlobs int
	totalBytes uint64
	startedAt  time.Time
	progress   func(int, string, any) error

	mu              sync.Mutex
	downloadedBytes uint64
	completedBlobs  int
	lastProgress    int
	lastReport      time.Time
}

func newImageBlobDownloadProgress(imageRef string, descriptors []imageregistry.Descriptor, progress func(int, string, any) error) *imageBlobDownloadProgress {
	tracker := &imageBlobDownloadProgress{
		imageRef:     imageRef,
		totalBlobs:   len(descriptors),
		startedAt:    time.Now(),
		progress:     progress,
		lastProgress: -1,
	}
	for _, desc := range descriptors {
		tracker.totalBytes += desc.SizeBytes
	}
	return tracker
}

func (tracker *imageBlobDownloadProgress) Add(desc imageregistry.Descriptor, bytes uint64) {
	if tracker == nil || bytes == 0 {
		return
	}
	tracker.mu.Lock()
	tracker.downloadedBytes += bytes
	details, progressValue, shouldReport := tracker.progressDetailsLocked(desc, false)
	tracker.mu.Unlock()
	if shouldReport {
		reportImageResolveProgress(tracker.progress, progressValue, "transferring image content", details)
	}
}

func (tracker *imageBlobDownloadProgress) Complete(desc imageregistry.Descriptor, completedBlobs int) {
	if tracker == nil {
		return
	}
	tracker.mu.Lock()
	if completedBlobs > tracker.completedBlobs {
		tracker.completedBlobs = completedBlobs
	}
	details, progressValue, shouldReport := tracker.progressDetailsLocked(desc, true)
	tracker.mu.Unlock()
	if shouldReport {
		reportImageResolveProgress(tracker.progress, progressValue, "downloaded image blob", details)
	}
}

func (tracker *imageBlobDownloadProgress) progressDetailsLocked(desc imageregistry.Descriptor, force bool) (map[string]any, int, bool) {
	now := time.Now()
	visibleBytes := tracker.downloadedBytes
	if tracker.totalBytes > 0 && visibleBytes > tracker.totalBytes {
		visibleBytes = tracker.totalBytes
	}
	progressValue := 45
	if tracker.totalBytes > 0 {
		progressValue += int((visibleBytes * 15) / tracker.totalBytes)
	} else if tracker.totalBlobs > 0 {
		progressValue += (tracker.completedBlobs * 15) / tracker.totalBlobs
	}
	if progressValue > 60 {
		progressValue = 60
	}

	if !force && progressValue == tracker.lastProgress && now.Sub(tracker.lastReport) < imageBlobProgressInterval {
		return nil, progressValue, false
	}
	tracker.lastProgress = progressValue
	tracker.lastReport = now

	elapsed := now.Sub(tracker.startedAt)
	details := map[string]any{
		"image":             tracker.imageRef,
		"completed_blobs":   tracker.completedBlobs,
		"total_blobs":       tracker.totalBlobs,
		"bytes_downloaded":  visibleBytes,
		"bytes_total":       tracker.totalBytes,
		"bytes_transferred": tracker.downloadedBytes,
		"elapsed_ms":        elapsed.Milliseconds(),
	}
	if elapsed > 0 {
		details["bytes_per_second"] = bytesPerSecond(tracker.downloadedBytes, elapsed)
	}
	if strings.TrimSpace(desc.Digest) != "" {
		details["digest"] = desc.Digest
		details["size_bytes"] = desc.SizeBytes
	}
	return details, progressValue, true
}

func reportImageResolveProgress(progress func(int, string, any) error, value int, message string, details any) {
	if progress == nil {
		return
	}
	if err := progress(value, message, details); err != nil {
		log.Debugf("failed to report image resolution progress: %s", err.Error())
	}
}

func (p2p *P2P) downloadImageBlob(ctx context.Context, candidates []imageContentCandidate, startIndex int, desc imageregistry.Descriptor, onBytes func(imageregistry.Descriptor, uint64)) (imageregistry.ImageContentBlob, error) {
	if desc.Digest == "" {
		return imageregistry.ImageContentBlob{}, fmt.Errorf("image content descriptor has empty digest")
	}
	if desc.SizeBytes > uint64(1<<63-1) {
		return imageregistry.ImageContentBlob{}, fmt.Errorf("image content blob is too large: digest=%s size=%d", desc.Digest, desc.SizeBytes)
	}

	tmp, err := os.CreateTemp("", "protos-image-blob-*.tmp")
	if err != nil {
		return imageregistry.ImageContentBlob{}, err
	}
	blobPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(blobPath)
		}
	}()

	var errs []error
	attempts := len(candidates) * imageBlobCandidateAttempts
	if attempts < len(candidates) {
		attempts = len(candidates)
	}
	for attempt := 0; attempt < attempts; attempt++ {
		candidate := candidates[(startIndex+attempt)%len(candidates)]
		startedAt := time.Now()
		log.Infof("downloading image blob %s from peer %s: size=%d", desc.Digest, candidate.peerID, desc.SizeBytes)
		if err := tmp.Truncate(0); err != nil {
			_ = tmp.Close()
			return imageregistry.ImageContentBlob{}, err
		}
		if _, err := tmp.Seek(0, io.SeekStart); err != nil {
			_ = tmp.Close()
			return imageregistry.ImageContentBlob{}, err
		}
		if err := p2p.downloadImageBlobFromCandidate(ctx, tmp, candidate, desc, func(bytes uint64) {
			if onBytes != nil {
				onBytes(desc, bytes)
			}
		}); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", candidate.peerID, err))
			log.Warnf("failed to download image blob %s from peer %s after %s: %v", desc.Digest, candidate.peerID, time.Since(startedAt).Round(time.Millisecond), err)
			continue
		}
		if err := tmp.Close(); err != nil {
			return imageregistry.ImageContentBlob{}, err
		}
		duration := time.Since(startedAt)
		log.Infof("downloaded image blob %s from peer %s: size=%d duration=%s throughput=%dB/s", desc.Digest, candidate.peerID, desc.SizeBytes, duration.Round(time.Millisecond), bytesPerSecond(desc.SizeBytes, duration))
		cleanup = false
		return imageregistry.ImageContentBlob{Descriptor: desc, Path: blobPath}, nil
	}
	_ = tmp.Close()
	return imageregistry.ImageContentBlob{}, errors.Join(errs...)
}

func cleanupImageContentBlobs(blobs []imageregistry.ImageContentBlob) func() {
	return func() {
		for _, blob := range blobs {
			if blob.Path != "" {
				_ = os.Remove(blob.Path)
			}
		}
	}
}

func drainImageContentBlobResults(results <-chan imageregistry.ImageContentBlob) []imageregistry.ImageContentBlob {
	var blobs []imageregistry.ImageContentBlob
	for {
		select {
		case blob, ok := <-results:
			if !ok {
				return blobs
			}
			blobs = append(blobs, blob)
		default:
			return blobs
		}
	}
}

func (p2p *P2P) downloadImageBlobFromCandidate(ctx context.Context, file *os.File, candidate imageContentCandidate, desc imageregistry.Descriptor, onBytes func(uint64)) error {
	connectedness := p2p.peerConnectedness(candidate.peerID)
	if connectedness == network.Limited {
		return p2p.downloadImageBlobLimitedFromCandidate(ctx, file, candidate, desc, onBytes)
	}
	if imageBlobUseRangeStreams(connectedness, desc.SizeBytes) {
		return p2p.downloadImageBlobRangesFromCandidate(ctx, file, candidate, desc, onBytes)
	}
	return p2p.downloadImageBlobRangeFromCandidate(ctx, file, candidate, desc.Digest, 0, desc.SizeBytes, onBytes)
}

func (p2p *P2P) imageBlobDownloadConcurrency(candidates []imageContentCandidate) int {
	if len(candidates) == 0 {
		return 0
	}
	allLimited := true
	for _, candidate := range candidates {
		if !p2p.peerConnectionLimited(candidate.peerID) {
			allLimited = false
			break
		}
	}
	return imageBlobDownloadConcurrencyForPolicy(len(candidates), allLimited)
}

func imageBlobDownloadConcurrencyForPolicy(candidateCount int, allLimited bool) int {
	if candidateCount == 0 {
		return 0
	}
	if allLimited {
		return 1
	}
	return imageBlobDownloadConcurrency
}

func imageBlobUseRangeStreams(connectedness network.Connectedness, sizeBytes uint64) bool {
	if connectedness == network.Limited {
		return false
	}
	return sizeBytes >= imageBlobParallelThresholdBytes
}

func (p2p *P2P) peerConnectedness(peerIDString string) network.Connectedness {
	if p2p == nil || p2p.host == nil {
		return network.NotConnected
	}
	peerID, err := peer.Decode(peerIDString)
	if err != nil {
		return network.NotConnected
	}
	return p2p.host.Network().Connectedness(peerID)
}

func (p2p *P2P) peerConnectionLimited(peerIDString string) bool {
	return p2p.peerConnectedness(peerIDString) == network.Limited
}

func (p2p *P2P) downloadImageBlobRangesFromCandidate(ctx context.Context, file *os.File, candidate imageContentCandidate, desc imageregistry.Descriptor, onBytes func(uint64)) error {
	if desc.SizeBytes > uint64(1<<63-1) {
		return fmt.Errorf("image content blob is too large: digest=%s size=%d", desc.Digest, desc.SizeBytes)
	}
	if err := file.Truncate(int64(desc.SizeBytes)); err != nil {
		return err
	}

	rangeSize := (desc.SizeBytes + uint64(imageBlobRangeConcurrency) - 1) / uint64(imageBlobRangeConcurrency)
	if rangeSize < imageBlobStreamChunkBytes {
		rangeSize = imageBlobStreamChunkBytes
	}
	rangeCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, imageBlobRangeConcurrency)
	var wg sync.WaitGroup
	rangeCount := 0
	startedAt := time.Now()
	for start := uint64(0); start < desc.SizeBytes; start += rangeSize {
		length := rangeSize
		if remaining := desc.SizeBytes - start; length > remaining {
			length = remaining
		}
		rangeCount++
		wg.Add(1)
		go func(start uint64, length uint64) {
			defer wg.Done()
			writer := &imageBlobRangeWriter{file: file, offset: start}
			if err := p2p.downloadImageBlobRangeFromCandidate(rangeCtx, writer, candidate, desc.Digest, start, length, onBytes); err != nil {
				select {
				case errCh <- err:
					cancel()
				default:
				}
			}
		}(start, length)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	var err error
	select {
	case <-done:
		select {
		case err = <-errCh:
		default:
		}
	case err := <-errCh:
		cancel()
		<-done
		return err
	case <-ctx.Done():
		cancel()
		<-done
		err = ctx.Err()
	}
	duration := time.Since(startedAt)
	if err != nil {
		log.Warnf("failed to download image blob %s from peer %s using %d range stream(s) after %s: %v", desc.Digest, candidate.peerID, rangeCount, duration.Round(time.Millisecond), err)
		return err
	}
	log.Infof("downloaded image blob %s from peer %s using %d range stream(s): size=%d range_size=%d duration=%s throughput=%dB/s", desc.Digest, candidate.peerID, rangeCount, desc.SizeBytes, rangeSize, duration.Round(time.Millisecond), bytesPerSecond(desc.SizeBytes, duration))
	return nil
}

func (p2p *P2P) downloadImageBlobLimitedFromCandidate(ctx context.Context, file *os.File, candidate imageContentCandidate, desc imageregistry.Descriptor, onBytes func(uint64)) error {
	if desc.SizeBytes > uint64(1<<63-1) {
		return fmt.Errorf("image content blob is too large: digest=%s size=%d", desc.Digest, desc.SizeBytes)
	}
	if err := file.Truncate(int64(desc.SizeBytes)); err != nil {
		return err
	}

	startedAt := time.Now()
	rangeCount := 0

	// A limited (relay/circuit) peer transfers the blob as small resumable slices
	// over a single reused raw stream. Reusing the stream amortizes the relay
	// connection setup that previously dominated cost (one fresh stream per
	// slice), while keeping slice-granular resumability: a failed slice resets
	// the stream, reconnects through the peer layer, and retries from the same
	// offset without re-fetching completed slices.
	var stream *limitedBlobStream
	defer func() {
		if stream != nil {
			stream.close()
		}
	}()

	for offset := uint64(0); offset < desc.SizeBytes; {
		length := imageBlobLimitedRangeLength(desc.SizeBytes - offset)
		if length == 0 {
			return fmt.Errorf("image blob %s limited transfer made no progress at offset %d", desc.Digest, offset)
		}

		var sliceErr error
		for attempt := 1; attempt <= imageBlobLimitedRangeAttempts; attempt++ {
			if stream == nil {
				opened, err := p2p.openLimitedBlobStream(ctx, candidate.peerID)
				if err != nil {
					sliceErr = err
				} else {
					stream = opened
				}
			}
			if stream != nil {
				writer := &imageBlobRangeWriter{file: file, offset: offset}
				if err := stream.downloadRange(writer, desc.Digest, offset, length, onBytes); err != nil {
					sliceErr = err
					// The reused stream is no longer trustworthy after a failure;
					// drop it so the next attempt reconnects and reopens.
					stream.reset()
					stream = nil
				} else {
					sliceErr = nil
					break
				}
			}

			if attempt < imageBlobLimitedRangeAttempts {
				log.Debugf("retrying image blob %s limited slice offset=%d length=%d from peer %s: attempt=%d err=%v", desc.Digest, offset, length, candidate.peerID, attempt+1, sliceErr)
				p2p.requestReconcile()
				if err := sleepContext(ctx, time.Duration(attempt)*250*time.Millisecond); err != nil {
					return errors.Join(sliceErr, err)
				}
			}
		}
		if sliceErr != nil {
			return fmt.Errorf("download image blob %s slice offset=%d length=%d from limited peer %s: %w", desc.Digest, offset, length, candidate.peerID, sliceErr)
		}
		offset += length
		rangeCount++
	}

	duration := time.Since(startedAt)
	log.Infof("downloaded image blob %s from limited peer %s using %d resumable stream slice(s) over a reused stream: size=%d slice_size=%d duration=%s throughput=%dB/s", desc.Digest, candidate.peerID, rangeCount, desc.SizeBytes, imageBlobLimitedStreamChunkBytes, duration.Round(time.Millisecond), bytesPerSecond(desc.SizeBytes, duration))
	return nil
}

// limitedBlobStream is a reusable raw blob data stream to a single limited peer.
// It carries many sequential range requests so the data plane amortizes the
// relay stream setup across slices. It is part of the data plane only: it runs
// on imageBlobStreamProtocol, never on the gRPC control channel.
type limitedBlobStream struct {
	stream network.Stream
	reader *bufio.Reader
	writer *bufio.Writer
}

func (p2p *P2P) openLimitedBlobStream(ctx context.Context, peerIDString string) (*limitedBlobStream, error) {
	if p2p == nil || p2p.host == nil {
		return nil, fmt.Errorf("p2p host is not configured")
	}
	if err := p2p.ensureImagePeerConnected(ctx, peerIDString); err != nil {
		return nil, err
	}
	peerID, err := peer.Decode(peerIDString)
	if err != nil {
		return nil, err
	}
	connectedness := p2p.host.Network().Connectedness(peerID)
	if !usablePeerConnectedness(connectedness) {
		return nil, fmt.Errorf("not connected to image peer %s: %s", peerIDString, connectedness)
	}
	streamCtx := ctx
	if connectedness == network.Limited {
		streamCtx = network.WithAllowLimitedConn(ctx, "protos image blob stream")
	}
	stream, err := p2p.host.NewStream(streamCtx, peerID, imageBlobStreamProtocol)
	if err != nil {
		return nil, err
	}
	conn := stream.Conn()
	log.Infof("opened reusable image blob stream to limited peer %s: local=%s remote=%s limited=%t", peerIDString, conn.LocalMultiaddr().String(), conn.RemoteMultiaddr().String(), conn.Stat().Limited)
	return &limitedBlobStream{
		stream: stream,
		reader: bufio.NewReaderSize(stream, 256*1024),
		writer: bufio.NewWriterSize(stream, 64*1024),
	}, nil
}

// downloadRange sends one range request over the reused stream and writes its
// eof-terminated data frames at the requested offset. A per-slice deadline
// bounds a stalled relay read so a stuck slice fails fast and is retried.
func (s *limitedBlobStream) downloadRange(writer io.Writer, digest string, offset uint64, length uint64, onBytes func(uint64)) error {
	if s == nil || s.stream == nil {
		return fmt.Errorf("limited blob stream is closed")
	}
	if err := s.stream.SetDeadline(time.Now().Add(imageBlobStreamTimeout(length))); err != nil {
		return err
	}
	request := &p2pproto.GetImageBlobRequest{
		Digest:    digest,
		ChunkSize: imageBlobStreamChunkBytes,
		Offset:    offset,
		Length:    length,
	}
	if err := writeImageBlobFrame(s.writer, request); err != nil {
		return err
	}
	if err := s.writer.Flush(); err != nil {
		return err
	}
	progressWriter := imageBlobProgressWriter{writer: writer, onBytes: onBytes}
	return receiveImageBlobDataRange(progressWriter, digest, offset, length, func() (uint64, []byte, bool, error) {
		return readImageBlobDataFrame(s.reader)
	})
}

// close cleanly half-closes the stream so the server's serve loop observes EOF
// at a frame boundary and shuts down its side.
func (s *limitedBlobStream) close() {
	if s == nil || s.stream == nil {
		return
	}
	_ = s.stream.Close()
	s.stream = nil
}

// reset aborts the stream after a failure so neither side keeps using it.
func (s *limitedBlobStream) reset() {
	if s == nil || s.stream == nil {
		return
	}
	_ = s.stream.Reset()
	s.stream = nil
}

func (p2p *P2P) ensureImagePeerConnected(ctx context.Context, peerIDString string) error {
	connectedness := p2p.peerConnectedness(peerIDString)
	if usablePeerConnectedness(connectedness) {
		return nil
	}
	if p2p == nil || p2p.machines == nil {
		return fmt.Errorf("not connected to image peer %s: %s", peerIDString, connectedness)
	}
	machine, found := p2p.machines.Get(peerIDString)
	if !found {
		p2p.requestReconcile()
		return fmt.Errorf("not connected to image peer %s: %s", peerIDString, connectedness)
	}
	connectCtx, cancel := context.WithTimeout(ctx, peerClientReconnectAttemptTimeout)
	defer cancel()
	if _, err := p2p.connectPeer(connectCtx, peerIDString, machine); err != nil {
		p2p.requestReconcile()
		return err
	}
	connectedness = p2p.peerConnectedness(peerIDString)
	if !usablePeerConnectedness(connectedness) {
		return fmt.Errorf("not connected to image peer %s after reconnect: %s", peerIDString, connectedness)
	}
	return nil
}

func imageBlobLimitedRangeLength(remaining uint64) uint64 {
	if remaining < imageBlobLimitedStreamChunkBytes {
		return remaining
	}
	return imageBlobLimitedStreamChunkBytes
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type imageBlobRangeWriter struct {
	file   *os.File
	offset uint64
}

func (w *imageBlobRangeWriter) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	if w.offset > uint64(1<<63-1) {
		return 0, fmt.Errorf("image blob range offset is too large: %d", w.offset)
	}
	n, err := w.file.WriteAt(data, int64(w.offset))
	w.offset += uint64(n)
	return n, err
}

type imageBlobProgressWriter struct {
	writer  io.Writer
	onBytes func(uint64)
}

func (w imageBlobProgressWriter) Write(data []byte) (int, error) {
	n, err := w.writer.Write(data)
	if n > 0 && w.onBytes != nil {
		w.onBytes(uint64(n))
	}
	return n, err
}

func (p2p *P2P) downloadImageBlobRangeFromCandidate(ctx context.Context, writer io.Writer, candidate imageContentCandidate, digest string, offset uint64, length uint64, onBytes func(uint64)) error {
	if p2p == nil || p2p.host == nil {
		return fmt.Errorf("p2p host is not configured")
	}
	peerID, err := peer.Decode(candidate.peerID)
	if err != nil {
		return err
	}
	callCtx, cancel := context.WithTimeout(ctx, imageBlobStreamTimeout(length))
	defer cancel()
	connectedness := p2p.host.Network().Connectedness(peerID)
	if usablePeerConnectedness(connectedness) {
		if connectedness == network.Limited {
			callCtx = network.WithAllowLimitedConn(callCtx, "protos image blob stream")
		}
	} else {
		return fmt.Errorf("not connected to image peer %s: %s", candidate.peerID, connectedness)
	}
	stream, err := p2p.host.NewStream(callCtx, peerID, imageBlobStreamProtocol)
	if err != nil {
		return err
	}
	defer stream.Close()
	if length >= imageBlobStreamChunkBytes || offset == 0 {
		conn := stream.Conn()
		log.Infof("opened image blob stream to peer %s: digest=%s offset=%d length=%d local=%s remote=%s limited=%t", candidate.peerID, digest, offset, length, conn.LocalMultiaddr().String(), conn.RemoteMultiaddr().String(), conn.Stat().Limited)
	}

	request := &p2pproto.GetImageBlobRequest{
		Digest:    digest,
		ChunkSize: imageBlobStreamChunkBytes,
		Offset:    offset,
		Length:    length,
	}
	bufferedWriter := bufio.NewWriterSize(stream, 64*1024)
	if err := writeImageBlobFrame(bufferedWriter, request); err != nil {
		return err
	}
	if err := bufferedWriter.Flush(); err != nil {
		return err
	}

	bufferedReader := bufio.NewReaderSize(stream, 256*1024)
	progressWriter := imageBlobProgressWriter{writer: writer, onBytes: onBytes}
	return receiveImageBlobDataRange(progressWriter, digest, offset, length, func() (uint64, []byte, bool, error) {
		return readImageBlobDataFrame(bufferedReader)
	})
}

func (p2p *P2P) sendImageBlobRange(ctx context.Context, digest string, offset uint64, length uint64, chunkSize uint64, send func(*p2pproto.GetImageBlobResponse) error) (uint64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if p2p == nil || p2p.imageManager == nil {
		return 0, fmt.Errorf("image manager is not configured")
	}
	if strings.TrimSpace(digest) == "" {
		return 0, fmt.Errorf("image blob digest is empty")
	}
	if send == nil {
		return 0, fmt.Errorf("image blob sender is not configured")
	}
	if chunkSize == 0 || chunkSize > imageBlobStreamChunkBytes {
		chunkSize = imageBlobStreamChunkBytes
	}

	reader, err := p2p.imageManager.OpenImageBlob(ctx, digest)
	if err != nil {
		return 0, err
	}
	defer reader.Close()

	size := reader.Size()
	if size < 0 {
		return 0, fmt.Errorf("content blob %s has unknown size", digest)
	}
	sizeBytes := uint64(size)
	if offset > sizeBytes {
		return 0, fmt.Errorf("image blob %s offset %d exceeds size %d", digest, offset, sizeBytes)
	}

	endOffset := sizeBytes
	if length > 0 {
		remaining := sizeBytes - offset
		if length > remaining {
			return 0, fmt.Errorf("image blob %s range offset=%d length=%d exceeds size %d", digest, offset, length, sizeBytes)
		}
		endOffset = offset + length
	}
	if offset == endOffset {
		err := send(&p2pproto.GetImageBlobResponse{
			Digest: digest,
			Offset: offset,
			Eof:    true,
		})
		return 0, err
	}

	currentOffset := offset
	var sentBytes uint64
	for currentOffset < endOffset {
		if err := ctx.Err(); err != nil {
			return sentBytes, err
		}
		readSize := chunkSize
		if remaining := endOffset - currentOffset; readSize > remaining {
			readSize = remaining
		}
		data := make([]byte, int(readSize))
		n, err := reader.ReadAt(data, int64(currentOffset))
		if err != nil && !errors.Is(err, io.EOF) {
			return sentBytes, err
		}
		if n == 0 {
			return sentBytes, fmt.Errorf("image blob %s reader made no progress at offset %d", digest, currentOffset)
		}
		data = data[:n]
		eof := currentOffset+uint64(n) >= endOffset
		if err := send(&p2pproto.GetImageBlobResponse{
			Digest: digest,
			Offset: currentOffset,
			Data:   data,
			Eof:    eof,
		}); err != nil {
			return sentBytes, err
		}
		sentBytes += uint64(n)
		currentOffset += uint64(n)
		if eof {
			return sentBytes, nil
		}
	}
	return sentBytes, nil
}

func receiveImageBlob(writer io.Writer, desc imageregistry.Descriptor, recv func() (*p2pproto.GetImageBlobResponse, error)) error {
	return receiveImageBlobRange(writer, desc.Digest, 0, desc.SizeBytes, recv)
}

func receiveImageBlobRange(writer io.Writer, digest string, startOffset uint64, sizeBytes uint64, recv func() (*p2pproto.GetImageBlobResponse, error)) error {
	return receiveImageBlobDataRange(writer, digest, startOffset, sizeBytes, func() (uint64, []byte, bool, error) {
		resp, err := recv()
		if err != nil {
			return 0, nil, false, err
		}
		if resp.GetDigest() != digest {
			return resp.GetOffset(), nil, false, fmt.Errorf("blob digest changed from %s to %s", digest, resp.GetDigest())
		}
		return resp.GetOffset(), resp.GetData(), resp.GetEof(), nil
	})
}

func receiveImageBlobDataRange(writer io.Writer, digest string, startOffset uint64, sizeBytes uint64, recv func() (uint64, []byte, bool, error)) error {
	offset := startOffset
	endOffset := startOffset + sizeBytes
	eofSeen := false
	for {
		respOffset, data, eof, err := recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if respOffset != offset {
			return fmt.Errorf("blob %s stream offset changed from %d to %d", digest, offset, respOffset)
		}
		if len(data) == 0 && !eof {
			return fmt.Errorf("blob %s stream made no progress at offset %d", digest, offset)
		}
		if len(data) > 0 {
			if _, err := writer.Write(data); err != nil {
				return err
			}
			offset += uint64(len(data))
			if offset > endOffset {
				return fmt.Errorf("blob %s stream sent through offset %d, want end offset %d", digest, offset, endOffset)
			}
		}
		if eof {
			eofSeen = true
			break
		}
	}
	if offset != endOffset {
		return fmt.Errorf("blob %s stream ended at offset %d, want end offset %d", digest, offset, endOffset)
	}
	if !eofSeen {
		log.Debugf("blob %s stream ended without explicit eof at offset %d", digest, offset)
	}
	return nil
}

func (p2p *P2P) handleImageBlobStream(stream network.Stream) {
	if err := p2p.serveImageBlobStream(stream); err != nil {
		log.Debugf("image blob stream failed from peer %s: %v", stream.Conn().RemotePeer().String(), err)
		_ = stream.Reset()
		return
	}
	_ = stream.Close()
}

func (p2p *P2P) serveImageBlobStream(stream network.Stream) error {
	if p2p == nil || p2p.imageManager == nil {
		return fmt.Errorf("image manager is not configured")
	}
	reader := bufio.NewReaderSize(stream, 64*1024)
	writer := bufio.NewWriterSize(stream, 256*1024)
	// Serve range requests in a loop so a single stream can carry many sequential
	// slices. Reusing the stream amortizes the (relay) connection setup that
	// otherwise dominates limited-peer transfer cost. A clean EOF at a frame
	// boundary means the client finished sending range requests.
	for {
		var req p2pproto.GetImageBlobRequest
		if err := readImageBlobFrame(reader, imageBlobRequestFrameMaxBytes, &req); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if err := p2p.serveImageBlobRangeRequest(stream.Conn(), writer, &req); err != nil {
			return err
		}
	}
}

func (p2p *P2P) serveImageBlobRangeRequest(conn network.Conn, writer *bufio.Writer, req *p2pproto.GetImageBlobRequest) error {
	digest := strings.TrimSpace(req.GetDigest())
	if digest == "" {
		return fmt.Errorf("image blob digest is empty")
	}
	chunkSize := req.GetChunkSize()
	if chunkSize == 0 {
		chunkSize = imageBlobStreamChunkBytes
	}
	if chunkSize > imageBlobStreamChunkBytes {
		chunkSize = imageBlobStreamChunkBytes
	}
	offset := req.GetOffset()
	remaining := req.GetLength()
	requestedLength := remaining
	startOffset := offset
	startedAt := time.Now()

	servedBytes, err := p2p.sendImageBlobRange(context.Background(), digest, offset, remaining, chunkSize, func(resp *p2pproto.GetImageBlobResponse) error {
		return writeImageBlobDataFrame(writer, resp.GetOffset(), resp.GetData(), resp.GetEof())
	})
	if err != nil {
		return err
	}
	// Flush once per range request rather than once per data frame: the client
	// waits for the whole range's eof before issuing the next request, so a
	// single flush at the range boundary is sufficient and avoids a relay packet
	// boundary per frame.
	if err := writer.Flush(); err != nil {
		return err
	}
	duration := time.Since(startedAt)
	if conn != nil && (duration >= 5*time.Second || requestedLength >= imageBlobParallelThresholdBytes) {
		log.Infof("served image blob %s to peer %s: offset=%d length=%d bytes=%d duration=%s throughput=%dB/s local=%s remote=%s limited=%t", digest, conn.RemotePeer().String(), startOffset, requestedLength, servedBytes, duration.Round(time.Millisecond), bytesPerSecond(servedBytes, duration), conn.LocalMultiaddr().String(), conn.RemoteMultiaddr().String(), conn.Stat().Limited)
	}
	return nil
}

func writeImageBlobDataFrame(writer *bufio.Writer, offset uint64, data []byte, eof bool) error {
	if len(data) > imageBlobResponseFrameMaxBytes {
		return fmt.Errorf("image blob data frame is too large: %d bytes", len(data))
	}
	var header [imageBlobDataFrameHeaderBytes]byte
	binary.BigEndian.PutUint64(header[0:8], offset)
	binary.BigEndian.PutUint32(header[8:12], uint32(len(data)))
	if eof {
		header[12] = imageBlobDataFrameFlagEOF
	}
	if _, err := writer.Write(header[:]); err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	_, err := writer.Write(data)
	return err
}

func readImageBlobDataFrame(reader *bufio.Reader) (uint64, []byte, bool, error) {
	var header [imageBlobDataFrameHeaderBytes]byte
	if _, err := io.ReadFull(reader, header[:]); err != nil {
		return 0, nil, false, err
	}
	offset := binary.BigEndian.Uint64(header[0:8])
	length := binary.BigEndian.Uint32(header[8:12])
	flags := header[12]
	if length > imageBlobResponseFrameMaxBytes {
		return 0, nil, false, fmt.Errorf("image blob data frame is too large: %d bytes", length)
	}
	eof := flags&imageBlobDataFrameFlagEOF != 0
	if length == 0 {
		if !eof {
			return 0, nil, false, fmt.Errorf("image blob data frame is empty")
		}
		return offset, nil, eof, nil
	}
	data := make([]byte, int(length))
	if _, err := io.ReadFull(reader, data); err != nil {
		return 0, nil, false, err
	}
	return offset, data, eof, nil
}

func writeImageBlobFrame(writer *bufio.Writer, message goproto.Message) error {
	payload, err := goproto.Marshal(message)
	if err != nil {
		return err
	}
	if len(payload) > imageBlobResponseFrameMaxBytes {
		return fmt.Errorf("image blob frame is too large: %d bytes", len(payload))
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))
	if _, err := writer.Write(header[:]); err != nil {
		return err
	}
	_, err = writer.Write(payload)
	return err
}

func readImageBlobFrame(reader *bufio.Reader, maxBytes int, message goproto.Message) error {
	var header [4]byte
	if _, err := io.ReadFull(reader, header[:]); err != nil {
		return err
	}
	length := binary.BigEndian.Uint32(header[:])
	if length == 0 {
		return fmt.Errorf("image blob frame is empty")
	}
	if length > uint32(maxBytes) {
		return fmt.Errorf("image blob frame is too large: %d bytes", length)
	}
	payload := make([]byte, int(length))
	if _, err := io.ReadFull(reader, payload); err != nil {
		return err
	}
	return goproto.Unmarshal(payload, message)
}

func imageBlobStreamTimeout(sizeBytes uint64) time.Duration {
	if sizeBytes == 0 {
		return imageBlobStreamMinTimeout
	}
	seconds := time.Duration(sizeBytes/imageBlobStreamMinBytesPerSecond) * time.Second
	if sizeBytes%imageBlobStreamMinBytesPerSecond != 0 {
		seconds += time.Second
	}
	timeout := imageBlobStreamMinTimeout + seconds
	if timeout > imageBlobStreamMaxTimeout {
		return imageBlobStreamMaxTimeout
	}
	return timeout
}

func bytesPerSecond(bytes uint64, duration time.Duration) uint64 {
	if bytes == 0 || duration <= 0 {
		return 0
	}
	return uint64(float64(bytes) / duration.Seconds())
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
