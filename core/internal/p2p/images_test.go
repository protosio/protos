package p2p

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/libp2p/go-libp2p/core/network"
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
	blob   []byte
	opens  int
	closes int
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

func (f *fakeImageManager) LoadImageArchive(context.Context, string, string) (imageregistry.LoadedImage, error) {
	return imageregistry.LoadedImage{}, nil
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
