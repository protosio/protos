package swarmionlink

import (
	"time"

	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"

	"github.com/nustiueudinastea/swarmion/transports"
)

type stream struct {
	raw libp2pnetwork.Stream
}

func wrapStream(raw libp2pnetwork.Stream) *stream {
	return &stream{raw: raw}
}

func (s *stream) Read(payload []byte) (int, error) {
	return s.raw.Read(payload)
}

func (s *stream) Write(payload []byte) (int, error) {
	return s.raw.Write(payload)
}

func (s *stream) LocalPeer() transports.PeerID {
	return transports.PeerID(s.raw.Conn().LocalPeer().String())
}

func (s *stream) RemotePeer() transports.PeerID {
	return transports.PeerID(s.raw.Conn().RemotePeer().String())
}

func (s *stream) Protocol() transports.ProtocolID {
	return transports.ProtocolID(s.raw.Protocol())
}

func (s *stream) SetDeadline(deadline time.Time) error {
	return s.raw.SetDeadline(deadline)
}

func (s *stream) SetReadDeadline(deadline time.Time) error {
	return s.raw.SetReadDeadline(deadline)
}

func (s *stream) SetWriteDeadline(deadline time.Time) error {
	return s.raw.SetWriteDeadline(deadline)
}

func (s *stream) CloseRead() error {
	return s.raw.CloseRead()
}

func (s *stream) CloseWrite() error {
	return s.raw.CloseWrite()
}

func (s *stream) Reset() error {
	return s.raw.Reset()
}

func (s *stream) Close() error {
	return s.raw.Close()
}

var _ transports.Stream = (*stream)(nil)
