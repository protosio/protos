package swarmionlink

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"

	"github.com/nustiueudinastea/swarmion/transports"
)

// ApplicationStreamAdmission runs before an inbound stream is offered by an
// application listener. Returning an error rejects and resets the stream. The
// callback may inspect the authenticated remote peer through stream.Conn().
type ApplicationStreamAdmission func(context.Context, libp2pnetwork.Stream) error

// NewApplicationListener builds a net.Listener-backed protocol handler that
// can be included in an ApplicationProtocolBundle. The protocol is not
// installed until the returned handler is passed to
// RegisterApplicationProtocols, which lets the listener and ordinary
// application handlers be installed as one atomic bundle.
//
// Each connection returned by Accept also implements network.Stream, making
// it suitable for transports such as go-libp2p-grpc that authenticate the
// underlying libp2p stream. The registry callback remains live until the
// accepted connection is closed. Closing the bundle registration cancels and
// drains those callbacks; closing only the listener stops future Accept calls
// but deliberately does not unregister a protocol behind the registration's
// back.
func (r *Registry) NewApplicationListener(
	id protocol.ID,
	admit ApplicationStreamAdmission,
) (net.Listener, ApplicationProtocolHandler, error) {
	if r == nil || r.link == nil || r.link.host == nil {
		return nil, ApplicationProtocolHandler{}, transports.NewError(
			transports.ErrorCodeClosedLink,
			"create application listener",
			transports.ErrClosedLink,
		)
	}
	if id == "" {
		return nil, ApplicationProtocolHandler{}, invalidArgumentError(
			"create application listener",
			"",
			"",
			fmt.Errorf("protocol is empty"),
		)
	}

	listener := newApplicationListener(r.link.host.Network().ListenAddresses, r.link.host.ID().String())
	handler := ApplicationProtocolHandler{
		ID: id,
		Handler: func(ctx context.Context, stream libp2pnetwork.Stream) {
			listener.handle(ctx, stream, admit)
		},
	}
	return listener, handler, nil
}

type applicationListener struct {
	listenAddresses func() []ma.Multiaddr
	localPeer       string
	connections     chan *applicationStreamConn
	done            chan struct{}
	closeOnce       sync.Once
	closed          atomic.Bool
}

func newApplicationListener(
	listenAddresses func() []ma.Multiaddr,
	localPeer string,
) *applicationListener {
	return &applicationListener{
		listenAddresses: listenAddresses,
		localPeer:       localPeer,
		connections:     make(chan *applicationStreamConn),
		done:            make(chan struct{}),
	}
}

func (l *applicationListener) Accept() (net.Conn, error) {
	if l == nil || l.closed.Load() {
		return nil, net.ErrClosed
	}
	select {
	case <-l.done:
		return nil, net.ErrClosed
	case conn := <-l.connections:
		// Resolve a simultaneous Close in favor of listener shutdown. The
		// handler is unblocked by reset rather than leaking an offered stream.
		if l.closed.Load() {
			_ = conn.Reset()
			return nil, net.ErrClosed
		}
		return conn, nil
	}
}

func (l *applicationListener) Close() error {
	if l == nil {
		return nil
	}
	l.closeOnce.Do(func() {
		l.closed.Store(true)
		close(l.done)
	})
	return nil
}

func (l *applicationListener) Addr() net.Addr {
	if l == nil {
		return applicationListenerAddr("swarmionlink")
	}
	if l != nil && l.listenAddresses != nil {
		for _, address := range l.listenAddresses() {
			if netAddress, err := manet.ToNetAddr(address); err == nil {
				return netAddress
			}
		}
	}
	return applicationListenerAddr(l.localPeer)
}

func (l *applicationListener) handle(
	ctx context.Context,
	raw libp2pnetwork.Stream,
	admit ApplicationStreamAdmission,
) {
	if admit != nil {
		if err := admit(ctx, raw); err != nil {
			_ = raw.Reset()
			return
		}
	}

	conn := newApplicationStreamConn(raw)
	select {
	case <-ctx.Done():
		_ = conn.Reset()
		return
	case <-l.done:
		_ = conn.Reset()
		return
	case l.connections <- conn:
	}

	// Listener.Close only stops admission. An already accepted net.Conn has
	// the normal independent lifetime. Registration.Close cancels ctx and
	// forcibly drains the stream through the shared protocol scope.
	select {
	case <-ctx.Done():
		_ = conn.Reset()
	case <-conn.done:
	}
}

type applicationStreamConn struct {
	libp2pnetwork.Stream
	done chan struct{}
	once sync.Once
}

func newApplicationStreamConn(stream libp2pnetwork.Stream) *applicationStreamConn {
	return &applicationStreamConn{Stream: stream, done: make(chan struct{})}
}

func (c *applicationStreamConn) Close() error {
	err := c.Stream.Close()
	c.finish()
	return err
}

func (c *applicationStreamConn) Reset() error {
	err := c.Stream.Reset()
	c.finish()
	return err
}

func (c *applicationStreamConn) ResetWithError(code libp2pnetwork.StreamErrorCode) error {
	err := c.Stream.ResetWithError(code)
	c.finish()
	return err
}

func (c *applicationStreamConn) LocalAddr() net.Addr {
	if addr, err := manet.ToNetAddr(c.Stream.Conn().LocalMultiaddr()); err == nil {
		return addr
	}
	return applicationListenerAddr(c.Stream.Conn().LocalPeer().String())
}

func (c *applicationStreamConn) RemoteAddr() net.Addr {
	if addr, err := manet.ToNetAddr(c.Stream.Conn().RemoteMultiaddr()); err == nil {
		return addr
	}
	return applicationListenerAddr(c.Stream.Conn().RemotePeer().String())
}

func (c *applicationStreamConn) finish() {
	c.once.Do(func() { close(c.done) })
}

type applicationListenerAddr string

func (a applicationListenerAddr) Network() string { return "libp2p" }
func (a applicationListenerAddr) String() string  { return string(a) }

var (
	_ net.Listener         = (*applicationListener)(nil)
	_ net.Conn             = (*applicationStreamConn)(nil)
	_ libp2pnetwork.Stream = (*applicationStreamConn)(nil)
)
