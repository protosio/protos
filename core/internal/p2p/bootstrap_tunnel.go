package p2p

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

func (p2p *P2P) StartSwarmionBootstrapTunnel(ctx context.Context, peerIDString string) (string, func(), error) {
	if p2p == nil || p2p.host == nil {
		return "", nil, fmt.Errorf("p2p host is nil")
	}
	if p2p.swarmionPort() <= 0 {
		return "", nil, fmt.Errorf("swarmion port is not configured")
	}
	peerID, err := peer.Decode(peerIDString)
	if err != nil {
		return "", nil, err
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}

	done := make(chan struct{})
	var closeOnce sync.Once
	closeFn := func() {
		closeOnce.Do(func() {
			close(done)
			_ = listener.Close()
		})
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-done:
					return
				default:
					log.Debugf("swarmion bootstrap tunnel accept failed: %v", err)
					return
				}
			}
			go p2p.forwardSwarmionBootstrapConn(ctx, peerID, conn)
		}
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	addr := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", port, peerIDString)
	log.Debugf("created libp2p bootstrap tunnel for Swarmion at '%s'", addr)
	return addr, closeFn, nil
}

func (p2p *P2P) handleSwarmionBootstrapStream(stream network.Stream) {
	remotePeer := stream.Conn().RemotePeer()
	if !p2p.peerKnownOrPending(remotePeer) {
		log.Warnf("rejecting swarmion bootstrap tunnel from unknown peer '%s'", remotePeer.String())
		_ = stream.Reset()
		return
	}

	target := fmt.Sprintf("127.0.0.1:%d", p2p.swarmionPort())
	conn, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		log.Errorf("failed to connect swarmion bootstrap tunnel to %s: %v", target, err)
		_ = stream.Reset()
		return
	}
	proxyConns(conn, stream)
}

func (p2p *P2P) forwardSwarmionBootstrapConn(ctx context.Context, peerID peer.ID, conn net.Conn) {
	defer conn.Close()

	streamCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	stream, err := p2p.host.NewStream(streamCtx, peerID, swarmionBootstrapProtocol)
	if err != nil {
		log.Errorf("failed to open swarmion bootstrap tunnel stream to %s: %v", peerID.String(), err)
		return
	}
	proxyConns(conn, stream)
}

func proxyConns(a io.ReadWriteCloser, b io.ReadWriteCloser) {
	errCh := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(a, b)
		errCh <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(b, a)
		errCh <- struct{}{}
	}()
	<-errCh
	_ = a.Close()
	_ = b.Close()
}

func (p2p *P2P) peerKnownOrPending(peerID peer.ID) bool {
	if p2p == nil || peerID == "" {
		return false
	}
	if _, found := p2p.machines.Get(peerID.String()); found {
		return true
	}
	if _, found := p2p.pendingPeers.Get(peerID.String()); found {
		return true
	}
	return false
}
