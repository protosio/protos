package p2p

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	p2pproto "github.com/protosio/protos/internal/p2p/proto"
)

const (
	imageArchiveUploadChunkBytes         uint64 = imageBlobStreamChunkBytes
	imageArchiveUploadRequestFrameBytes         = imageBlobResponseFrameMaxBytes
	imageArchiveUploadResponseFrameBytes        = imageBlobRequestFrameMaxBytes
	imageArchiveUploadAttempts                  = 4
	imageArchiveUploadImportTimeout             = 5 * time.Minute
)

type ImageArchiveUploadRequest struct {
	Instance    string
	ArchivePath string
	ImageRef    string
	UploadID    string
	Progress    func(ImageArchiveUploadProgress) error
}

type ImageArchiveUploadProgress struct {
	BytesUploaded    uint64
	ArchiveSizeBytes uint64
	ChunkBytes       int
	ChunkDuration    time.Duration
	Importing        bool
}

type ImageArchiveUploadResult struct {
	ImageRef         string
	TargetDigest     string
	Platform         string
	BytesUploaded    uint64
	ArchiveSizeBytes uint64
}

func (p2p *P2P) UploadImageArchiveToInstance(ctx context.Context, req ImageArchiveUploadRequest) (ImageArchiveUploadResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if p2p == nil || p2p.host == nil {
		return ImageArchiveUploadResult{}, fmt.Errorf("p2p host is not configured")
	}
	if p2p.machines == nil {
		return ImageArchiveUploadResult{}, fmt.Errorf("p2p peers are not configured")
	}

	instance := strings.TrimSpace(req.Instance)
	archivePath := strings.TrimSpace(req.ArchivePath)
	imageRef := strings.TrimSpace(req.ImageRef)
	if instance == "" || instance == "local" {
		return ImageArchiveUploadResult{}, fmt.Errorf("instance is required")
	}
	if archivePath == "" {
		return ImageArchiveUploadResult{}, fmt.Errorf("image archive path is empty")
	}
	if imageRef == "" {
		return ImageArchiveUploadResult{}, fmt.Errorf("image ref is empty")
	}

	peerIDString, _, found := p2p.resolveMachineByIdentity(instance)
	if !found {
		return ImageArchiveUploadResult{}, fmt.Errorf("could not find peer for instance %q", instance)
	}

	info, err := os.Stat(archivePath)
	if err != nil {
		return ImageArchiveUploadResult{}, fmt.Errorf("stat image archive %s: %w", archivePath, err)
	}
	if !info.Mode().IsRegular() {
		return ImageArchiveUploadResult{}, fmt.Errorf("image archive %s is not a regular file", archivePath)
	}
	total := uint64(info.Size())
	if total > uint64(1<<63-1) {
		return ImageArchiveUploadResult{}, fmt.Errorf("image archive %s is too large: %d bytes", archivePath, total)
	}

	uploadID := strings.TrimSpace(req.UploadID)
	if uploadID == "" {
		uploadID = fmt.Sprintf("upload-%d", time.Now().UnixNano())
	}
	if _, err := imageArchiveUploadPath(uploadID); err != nil {
		return ImageArchiveUploadResult{}, err
	}

	file, err := os.Open(archivePath)
	if err != nil {
		return ImageArchiveUploadResult{}, fmt.Errorf("open image archive %s: %w", archivePath, err)
	}
	defer file.Close()

	var stream *imageArchiveUploadStream
	defer func() {
		if stream != nil {
			stream.close()
		}
	}()

	buf := make([]byte, int(imageArchiveUploadChunkBytes))
	var offset uint64
	for {
		if err := ctx.Err(); err != nil {
			return ImageArchiveUploadResult{}, err
		}
		if _, err := file.Seek(int64(offset), io.SeekStart); err != nil {
			return ImageArchiveUploadResult{}, fmt.Errorf("seek image archive %s to %d: %w", archivePath, offset, err)
		}
		n, readErr := file.Read(buf)
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return ImageArchiveUploadResult{}, fmt.Errorf("read image archive %s: %w", archivePath, readErr)
		}
		if n == 0 && !errors.Is(readErr, io.EOF) {
			continue
		}

		eof := n == 0 && errors.Is(readErr, io.EOF)
		if eof && req.Progress != nil {
			if err := req.Progress(ImageArchiveUploadProgress{
				BytesUploaded:    offset,
				ArchiveSizeBytes: total,
				Importing:        true,
			}); err != nil {
				return ImageArchiveUploadResult{}, err
			}
		}

		var resp *p2pproto.UploadImageArchiveChunkResponse
		var chunkDuration time.Duration
		var chunkErr error
		for attempt := 1; attempt <= imageArchiveUploadAttempts; attempt++ {
			if stream == nil {
				opened, err := p2p.openImageArchiveUploadStream(ctx, peerIDString)
				if err != nil {
					chunkErr = err
				} else {
					stream = opened
				}
			}
			if stream != nil {
				startedAt := time.Now()
				resp, chunkErr = stream.uploadChunk(uploadID, imageRef, offset, buf[:n], eof)
				chunkDuration = time.Since(startedAt)
				if chunkErr == nil {
					break
				}
				stream.reset()
				stream = nil
			}

			if attempt < imageArchiveUploadAttempts {
				log.Debugf("retrying image archive upload %s to peer %s: offset=%d bytes=%d eof=%t attempt=%d err=%v", imageRef, peerIDString, offset, n, eof, attempt+1, chunkErr)
				p2p.requestReconcile()
				if err := sleepContext(ctx, time.Duration(attempt)*250*time.Millisecond); err != nil {
					return ImageArchiveUploadResult{}, errors.Join(chunkErr, err)
				}
			}
		}
		if chunkErr != nil {
			return ImageArchiveUploadResult{}, fmt.Errorf("upload image archive %s to peer %s at offset %d: %w", imageRef, peerIDString, offset, chunkErr)
		}

		expectedReceived := offset + uint64(n)
		if resp.GetReceivedBytes() != expectedReceived {
			return ImageArchiveUploadResult{}, fmt.Errorf("image archive upload offset mismatch: peer received %d, expected %d", resp.GetReceivedBytes(), expectedReceived)
		}
		offset = resp.GetReceivedBytes()
		if !eof {
			if req.Progress != nil && n > 0 {
				if err := req.Progress(ImageArchiveUploadProgress{
					BytesUploaded:    offset,
					ArchiveSizeBytes: total,
					ChunkBytes:       n,
					ChunkDuration:    chunkDuration,
				}); err != nil {
					return ImageArchiveUploadResult{}, err
				}
			}
			continue
		}
		if !resp.GetLoaded() {
			return ImageArchiveUploadResult{}, fmt.Errorf("image archive upload reached EOF without loading image")
		}
		return ImageArchiveUploadResult{
			ImageRef:         resp.GetImageRef(),
			TargetDigest:     resp.GetTargetDigest(),
			Platform:         resp.GetPlatform(),
			BytesUploaded:    offset,
			ArchiveSizeBytes: total,
		}, nil
	}
}

type imageArchiveUploadStream struct {
	stream network.Stream
	reader *bufio.Reader
	writer *bufio.Writer
}

func (p2p *P2P) openImageArchiveUploadStream(ctx context.Context, peerIDString string) (*imageArchiveUploadStream, error) {
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
		streamCtx = network.WithAllowLimitedConn(ctx, "protos image archive upload")
	}
	stream, err := p2p.host.NewStream(streamCtx, peerID, imageArchiveUploadProtocol)
	if err != nil {
		return nil, err
	}
	conn := stream.Conn()
	log.Infof("opened image archive upload stream to peer %s: local=%s remote=%s limited=%t", peerIDString, conn.LocalMultiaddr().String(), conn.RemoteMultiaddr().String(), conn.Stat().Limited)
	return &imageArchiveUploadStream{
		stream: stream,
		reader: bufio.NewReaderSize(stream, 64*1024),
		writer: bufio.NewWriterSize(stream, 256*1024),
	}, nil
}

func (s *imageArchiveUploadStream) uploadChunk(uploadID string, imageRef string, offset uint64, data []byte, eof bool) (*p2pproto.UploadImageArchiveChunkResponse, error) {
	if s == nil || s.stream == nil {
		return nil, fmt.Errorf("image archive upload stream is closed")
	}
	timeout := imageBlobStreamTimeout(uint64(len(data)))
	if eof {
		timeout = imageArchiveUploadImportTimeout
	}
	if err := s.stream.SetDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}
	req := &p2pproto.UploadImageArchiveChunkRequest{
		UploadId: uploadID,
		ImageRef: imageRef,
		Offset:   offset,
		Data:     data,
		Eof:      eof,
	}
	if err := writeImageBlobFrame(s.writer, req); err != nil {
		return nil, err
	}
	if err := s.writer.Flush(); err != nil {
		return nil, err
	}
	var resp p2pproto.UploadImageArchiveChunkResponse
	if err := readImageBlobFrame(s.reader, imageArchiveUploadResponseFrameBytes, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *imageArchiveUploadStream) close() {
	if s == nil || s.stream == nil {
		return
	}
	_ = s.stream.Close()
	s.stream = nil
}

func (s *imageArchiveUploadStream) reset() {
	if s == nil || s.stream == nil {
		return
	}
	_ = s.stream.Reset()
	s.stream = nil
}

func (p2p *P2P) handleImageArchiveUploadStream(stream network.Stream) {
	if err := p2p.serveImageArchiveUploadStream(stream); err != nil {
		log.Debugf("image archive upload stream failed from peer %s: %v", stream.Conn().RemotePeer().String(), err)
		_ = stream.Reset()
		return
	}
	_ = stream.Close()
}

func (p2p *P2P) serveImageArchiveUploadStream(stream network.Stream) error {
	if p2p == nil || p2p.imageManager == nil {
		return fmt.Errorf("image manager is not configured")
	}
	reader := bufio.NewReaderSize(stream, 256*1024)
	writer := bufio.NewWriterSize(stream, 64*1024)
	state := &imageArchiveUploadServerState{p2p: p2p}
	defer state.close()

	for {
		var req p2pproto.UploadImageArchiveChunkRequest
		if err := readImageBlobFrame(reader, imageArchiveUploadRequestFrameBytes, &req); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		resp, err := state.accept(context.Background(), &req)
		if err != nil {
			return err
		}
		if err := writeImageBlobFrame(writer, resp); err != nil {
			return err
		}
		if err := writer.Flush(); err != nil {
			return err
		}
		if resp.GetLoaded() {
			_ = os.Remove(state.path)
			return nil
		}
	}
}

type imageArchiveUploadServerState struct {
	p2p      *P2P
	uploadID string
	imageRef string
	path     string
	file     *os.File
}

func (state *imageArchiveUploadServerState) accept(ctx context.Context, req *p2pproto.UploadImageArchiveChunkRequest) (*p2pproto.UploadImageArchiveChunkResponse, error) {
	if state == nil || state.p2p == nil || state.p2p.imageManager == nil {
		return nil, fmt.Errorf("image manager is not configured")
	}
	uploadID := strings.TrimSpace(req.GetUploadId())
	imageRef := strings.TrimSpace(req.GetImageRef())
	if uploadID == "" {
		return nil, fmt.Errorf("image archive upload id is empty")
	}
	if imageRef == "" {
		return nil, fmt.Errorf("image ref is empty")
	}
	if len(req.GetData()) > int(imageArchiveUploadChunkBytes) {
		return nil, fmt.Errorf("image archive upload frame is too large: %d bytes", len(req.GetData()))
	}
	if len(req.GetData()) == 0 && !req.GetEof() {
		return nil, fmt.Errorf("image archive upload frame made no progress at offset %d", req.GetOffset())
	}
	if state.uploadID != "" && state.uploadID != uploadID {
		return nil, fmt.Errorf("image archive upload id changed from %s to %s", state.uploadID, uploadID)
	}
	if state.imageRef != "" && state.imageRef != imageRef {
		return nil, fmt.Errorf("image archive upload ref changed from %s to %s", state.imageRef, imageRef)
	}
	if state.path == "" {
		archivePath, err := imageArchiveUploadPath(uploadID)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Dir(archivePath), 0700); err != nil {
			return nil, err
		}
		state.path = archivePath
		state.uploadID = uploadID
		state.imageRef = imageRef
	}
	if state.file == nil {
		flags := os.O_CREATE | os.O_RDWR
		if req.GetOffset() == 0 {
			flags |= os.O_TRUNC
		}
		file, err := os.OpenFile(state.path, flags, 0600)
		if err != nil {
			return nil, err
		}
		state.file = file
	}

	received, err := state.write(req)
	if err != nil {
		return nil, err
	}
	resp := &p2pproto.UploadImageArchiveChunkResponse{ReceivedBytes: received}
	if !req.GetEof() {
		return resp, nil
	}

	if err := state.close(); err != nil {
		return nil, err
	}
	log.Infof("image archive upload stream complete: upload=%s image=%s bytes=%d; importing into runtime", uploadID, imageRef, received)
	loaded, err := state.p2p.imageManager.LoadImageArchive(ctx, state.path, imageRef)
	if err != nil {
		_ = os.Remove(state.path)
		return nil, err
	}
	resp.Loaded = true
	resp.ImageRef = loaded.ImageRef
	resp.TargetDigest = loaded.TargetDigest
	resp.Platform = loaded.Platform
	log.Infof("image archive import complete: upload=%s image=%s digest=%s platform=%s bytes=%d", uploadID, loaded.ImageRef, loaded.TargetDigest, loaded.Platform, received)
	return resp, nil
}

func (state *imageArchiveUploadServerState) write(req *p2pproto.UploadImageArchiveChunkRequest) (uint64, error) {
	if state.file == nil {
		return 0, fmt.Errorf("image archive upload file is not open")
	}
	offset := req.GetOffset()
	data := req.GetData()
	if offset > uint64(1<<63-1) {
		return 0, fmt.Errorf("image archive upload offset is too large: %d", offset)
	}
	if uint64(len(data)) > uint64(1<<63-1)-offset {
		return 0, fmt.Errorf("image archive upload range is too large: offset=%d bytes=%d", offset, len(data))
	}
	endOffset := offset + uint64(len(data))
	info, err := state.file.Stat()
	if err != nil {
		return 0, err
	}
	currentSize := uint64(info.Size())
	if offset > currentSize {
		return 0, fmt.Errorf("image archive upload offset mismatch: got %d, current size %d", offset, currentSize)
	}
	if endOffset <= currentSize {
		if req.GetEof() && len(data) == 0 && endOffset != currentSize {
			return 0, fmt.Errorf("image archive EOF offset mismatch: got %d, current size %d", endOffset, currentSize)
		}
		return endOffset, nil
	}
	if offset < currentSize {
		return 0, fmt.Errorf("image archive upload partial overlap: got offset=%d end=%d current size=%d", offset, endOffset, currentSize)
	}
	written, err := state.file.WriteAt(data, int64(offset))
	if err != nil {
		return 0, err
	}
	if written != len(data) {
		return 0, io.ErrShortWrite
	}
	return endOffset, nil
}

func (state *imageArchiveUploadServerState) close() error {
	if state == nil || state.file == nil {
		return nil
	}
	err := state.file.Close()
	state.file = nil
	return err
}

func imageArchiveUploadPath(uploadID string) (string, error) {
	uploadID = strings.TrimSpace(uploadID)
	if uploadID == "" {
		return "", fmt.Errorf("image archive upload id is empty")
	}
	if len(uploadID) > 96 {
		return "", fmt.Errorf("image archive upload id is too long")
	}
	for _, r := range uploadID {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			continue
		}
		return "", fmt.Errorf("image archive upload id contains invalid character %q", r)
	}
	return filepath.Join(os.TempDir(), "protos-p2p-image-upload-"+uploadID+".tar.gz"), nil
}
