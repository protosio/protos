package p2p

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/protosio/protos/internal/imageregistry"
	p2pproto "github.com/protosio/protos/internal/p2p/proto"
)

func TestReceiveImageBlobStreamsBlob(t *testing.T) {
	receiver := &scriptedImageBlobReceiver{
		responses: []*p2pproto.GetImageBlobResponse{
			{Digest: "sha256:test", Offset: 0, Data: []byte("abc")},
			{Digest: "sha256:test", Offset: 3, Data: []byte("def"), Eof: true},
		},
	}
	var out bytes.Buffer
	err := receiveImageBlob(&out, imageregistry.Descriptor{Digest: "sha256:test", SizeBytes: 6}, receiver.Recv)
	if err != nil {
		t.Fatalf("receiveImageBlob error = %v", err)
	}
	if got := out.String(); got != "abcdef" {
		t.Fatalf("downloaded content = %q, want %q", got, "abcdef")
	}
}

func TestReceiveImageBlobRejectsOffsetMismatch(t *testing.T) {
	receiver := &scriptedImageBlobReceiver{
		responses: []*p2pproto.GetImageBlobResponse{
			{Digest: "sha256:test", Offset: 1, Data: []byte("abc"), Eof: true},
		},
	}
	var out bytes.Buffer
	err := receiveImageBlob(&out, imageregistry.Descriptor{Digest: "sha256:test", SizeBytes: 3}, receiver.Recv)
	if err == nil {
		t.Fatal("receiveImageBlob succeeded; want offset mismatch error")
	}
	if !strings.Contains(err.Error(), "offset changed") {
		t.Fatalf("error = %v, want offset mismatch", err)
	}
}

func TestSendImageBlobRangeOpensReaderOnce(t *testing.T) {
	manager := &fakeImageManager{blob: []byte("abcdefghijkl")}
	p := &P2P{imageManager: manager}

	var responses []*p2pproto.GetImageBlobResponse
	sent, err := p.sendImageBlobRange(context.Background(), "sha256:test", 3, 6, 2, func(resp *p2pproto.GetImageBlobResponse) error {
		responses = append(responses, &p2pproto.GetImageBlobResponse{
			Digest: resp.GetDigest(),
			Offset: resp.GetOffset(),
			Data:   append([]byte(nil), resp.GetData()...),
			Eof:    resp.GetEof(),
		})
		return nil
	})
	if err != nil {
		t.Fatalf("sendImageBlobRange error = %v", err)
	}
	if sent != 6 {
		t.Fatalf("sent bytes = %d, want 6", sent)
	}
	if manager.opens != 1 {
		t.Fatalf("reader opens = %d, want 1", manager.opens)
	}
	if manager.closes != 1 {
		t.Fatalf("reader closes = %d, want 1", manager.closes)
	}
	if len(responses) != 3 {
		t.Fatalf("responses = %d, want 3", len(responses))
	}
	if got := string(responses[0].GetData()) + string(responses[1].GetData()) + string(responses[2].GetData()); got != "defghi" {
		t.Fatalf("sent data = %q, want defghi", got)
	}
	if !responses[2].GetEof() {
		t.Fatal("last response did not set eof")
	}
}

// TestServeImageBlobStreamServesSequentialRangesOverReusedStream verifies the
// data-plane stream-reuse path: a single connection carries multiple sequential
// range requests (as the limited-peer transfer now does), the server serves
// each eof-terminated range in a loop, and the client reassembles the full blob.
func TestServeImageBlobStreamServesSequentialRangesOverReusedStream(t *testing.T) {
	blob := []byte("abcdefghijklmnopqrstuvwxyz0123456789")
	manager := &fakeImageManager{blob: blob}
	p := &P2P{imageManager: manager}
	digest := "sha256:test"

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		reader := bufio.NewReaderSize(server, 64*1024)
		writer := bufio.NewWriterSize(server, 64*1024)
		for {
			var req p2pproto.GetImageBlobRequest
			if err := readImageBlobFrame(reader, imageBlobRequestFrameMaxBytes, &req); err != nil {
				return
			}
			if err := p.serveImageBlobRangeRequest(nil, writer, &req); err != nil {
				return
			}
		}
	}()

	cw := bufio.NewWriterSize(client, 64*1024)
	cr := bufio.NewReaderSize(client, 64*1024)
	var got []byte
	for _, rng := range [][2]uint64{{0, 10}, {10, 10}, {20, uint64(len(blob)) - 20}} {
		req := &p2pproto.GetImageBlobRequest{Digest: digest, ChunkSize: imageBlobStreamChunkBytes, Offset: rng[0], Length: rng[1]}
		if err := writeImageBlobFrame(cw, req); err != nil {
			t.Fatalf("write request offset=%d: %v", rng[0], err)
		}
		if err := cw.Flush(); err != nil {
			t.Fatalf("flush request offset=%d: %v", rng[0], err)
		}
		var buf bytes.Buffer
		if err := receiveImageBlobDataRange(&buf, digest, rng[0], rng[1], func() (uint64, []byte, bool, error) {
			return readImageBlobDataFrame(cr)
		}); err != nil {
			t.Fatalf("receive range offset=%d length=%d: %v", rng[0], rng[1], err)
		}
		got = append(got, buf.Bytes()...)
	}
	client.Close()
	<-serverDone

	if !bytes.Equal(got, blob) {
		t.Fatalf("reassembled %q, want %q", got, blob)
	}
	if manager.opens != 3 {
		t.Fatalf("reader opens = %d, want 3 (one per range request)", manager.opens)
	}
}

func TestImageBlobDataFrameRoundTrip(t *testing.T) {
	var raw bytes.Buffer
	writer := bufio.NewWriter(&raw)
	if err := writeImageBlobDataFrame(writer, 7, []byte("payload"), true); err != nil {
		t.Fatalf("writeImageBlobDataFrame error = %v", err)
	}
	if err := writer.Flush(); err != nil {
		t.Fatalf("flush error = %v", err)
	}

	offset, data, eof, err := readImageBlobDataFrame(bufio.NewReader(&raw))
	if err != nil {
		t.Fatalf("readImageBlobDataFrame error = %v", err)
	}
	if offset != 7 {
		t.Fatalf("offset = %d, want 7", offset)
	}
	if string(data) != "payload" {
		t.Fatalf("data = %q, want payload", data)
	}
	if !eof {
		t.Fatal("eof = false, want true")
	}
}

func TestImageBlobDataFrameRejectsEmptyNonEOF(t *testing.T) {
	var raw bytes.Buffer
	writer := bufio.NewWriter(&raw)
	if err := writeImageBlobDataFrame(writer, 0, nil, false); err != nil {
		t.Fatalf("writeImageBlobDataFrame error = %v", err)
	}
	if err := writer.Flush(); err != nil {
		t.Fatalf("flush error = %v", err)
	}

	_, _, _, err := readImageBlobDataFrame(bufio.NewReader(&raw))
	if err == nil {
		t.Fatal("readImageBlobDataFrame succeeded; want empty frame error")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("error = %v, want empty frame", err)
	}
}

func TestImageBlobDownloadConcurrencyForPolicy(t *testing.T) {
	if got := imageBlobDownloadConcurrencyForPolicy(0, true); got != 0 {
		t.Fatalf("zero candidate concurrency = %d, want 0", got)
	}
	if got := imageBlobDownloadConcurrencyForPolicy(1, true); got != 1 {
		t.Fatalf("limited candidate concurrency = %d, want 1", got)
	}
	if got := imageBlobDownloadConcurrencyForPolicy(1, false); got != imageBlobDownloadConcurrency {
		t.Fatalf("direct candidate concurrency = %d, want %d", got, imageBlobDownloadConcurrency)
	}
}

func TestImageBlobUseRangeStreamsSkipsLimitedConnections(t *testing.T) {
	if imageBlobUseRangeStreams(network.Limited, imageBlobParallelThresholdBytes*2) {
		t.Fatal("limited connections should not use parallel range streams")
	}
	if !imageBlobUseRangeStreams(network.Connected, imageBlobParallelThresholdBytes) {
		t.Fatal("direct large blobs should use parallel range streams")
	}
	if imageBlobUseRangeStreams(network.Connected, imageBlobParallelThresholdBytes-1) {
		t.Fatal("direct small blobs should use a single stream")
	}
}

func TestImageBlobLimitedRangeLength(t *testing.T) {
	if got := imageBlobLimitedRangeLength(0); got != 0 {
		t.Fatalf("empty remaining range = %d, want 0", got)
	}
	if got := imageBlobLimitedRangeLength(imageBlobLimitedStreamChunkBytes - 1); got != imageBlobLimitedStreamChunkBytes-1 {
		t.Fatalf("short remaining range = %d, want %d", got, imageBlobLimitedStreamChunkBytes-1)
	}
	if got := imageBlobLimitedRangeLength(imageBlobLimitedStreamChunkBytes); got != imageBlobLimitedStreamChunkBytes {
		t.Fatalf("exact remaining range = %d, want %d", got, imageBlobLimitedStreamChunkBytes)
	}
	if got := imageBlobLimitedRangeLength(imageBlobLimitedStreamChunkBytes + 1); got != imageBlobLimitedStreamChunkBytes {
		t.Fatalf("long remaining range = %d, want %d", got, imageBlobLimitedStreamChunkBytes)
	}
}

func TestServeImageArchiveUploadStreamLoadsArchive(t *testing.T) {
	uploadID := "test-archive-upload"
	imageRef := "example/app:latest"
	content := []byte("archive-content-over-stream")
	if path, err := imageArchiveUploadPath(uploadID); err == nil {
		_ = os.Remove(path)
		defer os.Remove(path)
	}

	manager := &fakeImageManager{
		loadedImage: imageregistry.LoadedImage{
			ImageRef:     imageRef,
			TargetDigest: "sha256:archive",
			Platform:     "linux/amd64",
		},
	}
	p := &P2P{imageManager: manager}

	client, server := net.Pipe()
	defer client.Close()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- p.serveImageArchiveUploadStream(&testNetworkStream{conn: server})
	}()

	writer := bufio.NewWriterSize(client, 64*1024)
	reader := bufio.NewReaderSize(client, 64*1024)
	resp := writeImageArchiveUploadFrame(t, writer, reader, &p2pproto.UploadImageArchiveChunkRequest{
		UploadId: uploadID,
		ImageRef: imageRef,
		Offset:   0,
		Data:     content[:8],
	})
	if resp.GetReceivedBytes() != 8 || resp.GetLoaded() {
		t.Fatalf("first response = %+v, want received=8 loaded=false", resp)
	}
	resp = writeImageArchiveUploadFrame(t, writer, reader, &p2pproto.UploadImageArchiveChunkRequest{
		UploadId: uploadID,
		ImageRef: imageRef,
		Offset:   8,
		Data:     content[8:],
	})
	if resp.GetReceivedBytes() != uint64(len(content)) || resp.GetLoaded() {
		t.Fatalf("second response = %+v, want received=%d loaded=false", resp, len(content))
	}
	resp = writeImageArchiveUploadFrame(t, writer, reader, &p2pproto.UploadImageArchiveChunkRequest{
		UploadId: uploadID,
		ImageRef: imageRef,
		Offset:   uint64(len(content)),
		Eof:      true,
	})
	if !resp.GetLoaded() {
		t.Fatalf("final response loaded = false, want true")
	}
	if resp.GetImageRef() != imageRef || resp.GetTargetDigest() != "sha256:archive" || resp.GetPlatform() != "linux/amd64" {
		t.Fatalf("final response = %+v, want loaded metadata", resp)
	}
	if err := <-serverErr; err != nil {
		t.Fatalf("serveImageArchiveUploadStream error = %v", err)
	}
	if manager.loadCalls != 1 {
		t.Fatalf("load calls = %d, want 1", manager.loadCalls)
	}
	if manager.loadImageRef != imageRef {
		t.Fatalf("loaded image ref = %q, want %q", manager.loadImageRef, imageRef)
	}
	if !bytes.Equal(manager.loadArchiveContent, content) {
		t.Fatalf("loaded archive content = %q, want %q", manager.loadArchiveContent, content)
	}
	if _, err := os.Stat(manager.loadArchivePath); !os.IsNotExist(err) {
		t.Fatalf("archive temp path still exists after import: err=%v", err)
	}
}

func TestImageArchiveUploadServerStateResumesAcrossStreams(t *testing.T) {
	uploadID := "test-archive-resume"
	imageRef := "example/app:latest"
	content := []byte("resumable archive content")
	if path, err := imageArchiveUploadPath(uploadID); err == nil {
		_ = os.Remove(path)
		defer os.Remove(path)
	}

	manager := &fakeImageManager{}
	p := &P2P{imageManager: manager}

	first := &imageArchiveUploadServerState{p2p: p}
	resp, err := first.accept(context.Background(), &p2pproto.UploadImageArchiveChunkRequest{
		UploadId: uploadID,
		ImageRef: imageRef,
		Offset:   0,
		Data:     content[:9],
	})
	if err != nil {
		t.Fatalf("first chunk error = %v", err)
	}
	if resp.GetReceivedBytes() != 9 {
		t.Fatalf("first received = %d, want 9", resp.GetReceivedBytes())
	}
	if err := first.close(); err != nil {
		t.Fatalf("close first upload state: %v", err)
	}

	second := &imageArchiveUploadServerState{p2p: p}
	resp, err = second.accept(context.Background(), &p2pproto.UploadImageArchiveChunkRequest{
		UploadId: uploadID,
		ImageRef: imageRef,
		Offset:   9,
		Data:     content[9:],
	})
	if err != nil {
		t.Fatalf("resumed chunk error = %v", err)
	}
	if resp.GetReceivedBytes() != uint64(len(content)) {
		t.Fatalf("resumed received = %d, want %d", resp.GetReceivedBytes(), len(content))
	}
	resp, err = second.accept(context.Background(), &p2pproto.UploadImageArchiveChunkRequest{
		UploadId: uploadID,
		ImageRef: imageRef,
		Offset:   uint64(len(content)),
		Eof:      true,
	})
	if err != nil {
		t.Fatalf("eof chunk error = %v", err)
	}
	if !resp.GetLoaded() {
		t.Fatalf("loaded = false, want true")
	}
	if !bytes.Equal(manager.loadArchiveContent, content) {
		t.Fatalf("loaded archive content = %q, want %q", manager.loadArchiveContent, content)
	}
}

func writeImageArchiveUploadFrame(t *testing.T, writer *bufio.Writer, reader *bufio.Reader, req *p2pproto.UploadImageArchiveChunkRequest) *p2pproto.UploadImageArchiveChunkResponse {
	t.Helper()
	if err := writeImageBlobFrame(writer, req); err != nil {
		t.Fatalf("write upload request: %v", err)
	}
	if err := writer.Flush(); err != nil {
		t.Fatalf("flush upload request: %v", err)
	}
	var resp p2pproto.UploadImageArchiveChunkResponse
	if err := readImageBlobFrame(reader, imageArchiveUploadResponseFrameBytes, &resp); err != nil {
		t.Fatalf("read upload response: %v", err)
	}
	return &resp
}

type testNetworkStream struct {
	conn net.Conn
}

func (s *testNetworkStream) Read(data []byte) (int, error)      { return s.conn.Read(data) }
func (s *testNetworkStream) Write(data []byte) (int, error)     { return s.conn.Write(data) }
func (s *testNetworkStream) Close() error                       { return s.conn.Close() }
func (s *testNetworkStream) LocalAddr() net.Addr                { return s.conn.LocalAddr() }
func (s *testNetworkStream) RemoteAddr() net.Addr               { return s.conn.RemoteAddr() }
func (s *testNetworkStream) SetDeadline(t time.Time) error      { return s.conn.SetDeadline(t) }
func (s *testNetworkStream) SetReadDeadline(t time.Time) error  { return s.conn.SetReadDeadline(t) }
func (s *testNetworkStream) SetWriteDeadline(t time.Time) error { return s.conn.SetWriteDeadline(t) }
func (s *testNetworkStream) CloseWrite() error                  { return nil }
func (s *testNetworkStream) CloseRead() error                   { return nil }
func (s *testNetworkStream) Reset() error                       { return s.Close() }
func (s *testNetworkStream) ResetWithError(network.StreamErrorCode) error {
	return s.Close()
}
func (s *testNetworkStream) ID() string                    { return "test-stream" }
func (s *testNetworkStream) Protocol() protocol.ID         { return imageArchiveUploadProtocol }
func (s *testNetworkStream) SetProtocol(protocol.ID) error { return nil }
func (s *testNetworkStream) Stat() network.Stats           { return network.Stats{} }
func (s *testNetworkStream) Conn() network.Conn            { return nil }
func (s *testNetworkStream) Scope() network.StreamScope    { return nil }

type scriptedImageBlobReceiver struct {
	responses []*p2pproto.GetImageBlobResponse
	err       error
	index     int
}

func (s *scriptedImageBlobReceiver) Recv() (*p2pproto.GetImageBlobResponse, error) {
	if s.index < len(s.responses) {
		resp := s.responses[s.index]
		s.index++
		return resp, nil
	}
	if s.err != nil {
		return nil, s.err
	}
	return nil, io.EOF
}

type fakeImageManager struct {
	blob               []byte
	opens              int
	closes             int
	loadedImage        imageregistry.LoadedImage
	loadArchivePath    string
	loadImageRef       string
	loadArchiveContent []byte
	loadCalls          int
	loadErr            error
}

func (f *fakeImageManager) DescribeImage(context.Context, string) (string, string, string, map[string]string, bool, error) {
	return "", "", "", nil, false, nil
}

func (f *fakeImageManager) GetImageContent(context.Context, string) (imageregistry.ImageContent, bool, error) {
	return imageregistry.ImageContent{}, false, nil
}

func (f *fakeImageManager) OpenImageBlob(context.Context, string) (imageregistry.ImageBlobReader, error) {
	f.opens++
	return &fakeImageBlobReader{
		Reader: bytes.NewReader(f.blob),
		close: func() {
			f.closes++
		},
	}, nil
}

func (f *fakeImageManager) MissingImageContent(context.Context, []imageregistry.Descriptor) ([]imageregistry.Descriptor, error) {
	return nil, nil
}

func (f *fakeImageManager) EnsureImageContent(context.Context, imageregistry.ImageContentImport, func(int, string, any) error) error {
	return nil
}

func (f *fakeImageManager) LoadImageArchive(_ context.Context, archivePath string, imageRef string) (imageregistry.LoadedImage, error) {
	f.loadCalls++
	f.loadArchivePath = archivePath
	f.loadImageRef = imageRef
	data, err := os.ReadFile(archivePath)
	if err == nil {
		f.loadArchiveContent = data
	}
	if f.loadErr != nil {
		return imageregistry.LoadedImage{}, f.loadErr
	}
	loaded := f.loadedImage
	if loaded.ImageRef == "" {
		loaded.ImageRef = imageRef
	}
	if loaded.TargetDigest == "" {
		loaded.TargetDigest = "sha256:loaded"
	}
	if loaded.Platform == "" {
		loaded.Platform = "linux/amd64"
	}
	return loaded, nil
}

type fakeImageBlobReader struct {
	*bytes.Reader
	close func()
}

func (r *fakeImageBlobReader) Close() error {
	if r.close != nil {
		r.close()
	}
	return nil
}
