package p2p

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	sec "github.com/libp2p/go-libp2p/core/sec"
	libp2pswarm "github.com/libp2p/go-libp2p/p2p/net/swarm"
	ma "github.com/multiformats/go-multiaddr"
)

type DiscoveredPeer struct {
	ID          string
	PublicKey   string
	Fingerprint string
	Address     string
}

func (p2p *P2P) SetInitPeerPublicKey(publicKey string) error {
	peerID, err := p2p.pubKeyToPeerID(publicKey)
	if err != nil {
		return err
	}
	p2p.initMu.Lock()
	p2p.initPeerID = peerID
	p2p.initMu.Unlock()
	log.Infof("restricted initialisation to peer '%s'", peerID.String())
	return nil
}

func (p2p *P2P) expectedInitPeer() (peer.ID, bool) {
	if p2p == nil {
		return "", false
	}
	p2p.initMu.RLock()
	defer p2p.initMu.RUnlock()
	if p2p.initPeerID == "" {
		return "", false
	}
	return p2p.initPeerID, true
}

func (p2p *P2P) initPeerAllowed(peerID peer.ID) bool {
	expected, restricted := p2p.expectedInitPeer()
	return !restricted || expected == peerID
}

func (p2p *P2P) DiscoverPeer(ctx context.Context, target string) (*DiscoveredPeer, error) {
	addrs, err := p2p.tofuTransportAddrs(target)
	if err != nil {
		return nil, err
	}
	var lastErr error
	delay := 2 * time.Second
	for {
		for _, addr := range addrs {
			attemptCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			discovered, err := p2p.DiscoverPeerAt(attemptCtx, addr)
			cancel()
			if err == nil {
				return discovered, nil
			}
			lastErr = err
		}
		wait := delay
		if lastErr != nil && strings.Contains(lastErr.Error(), "rate limit exceeded") {
			wait = 15 * time.Second
		}
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return nil, fmt.Errorf("discover peer at %s: %w", target, lastErr)
			}
			return nil, ctx.Err()
		case <-time.After(wait):
			if delay < 10*time.Second {
				delay *= 2
				if delay > 10*time.Second {
					delay = 10 * time.Second
				}
			}
		}
	}
}

func (p2p *P2P) DiscoverPeerAt(ctx context.Context, transportAddr string) (*DiscoveredPeer, error) {
	if p2p == nil || p2p.host == nil {
		return nil, errors.New("p2p host is nil")
	}
	addr, err := ma.NewMultiaddr(strings.TrimSpace(transportAddr))
	if err != nil {
		return nil, fmt.Errorf("parse bootstrap address: %w", err)
	}
	addr, _ = peer.SplitAddr(addr)
	if addr == nil {
		return nil, fmt.Errorf("bootstrap address %q has no transport component", transportAddr)
	}

	fakeID, err := randomPeerID()
	if err != nil {
		return nil, err
	}
	defer p2p.forgetPeer(fakeID)

	p2p.addPeerAddr(fakeID, addr)
	if err := p2p.host.Connect(ctx, peer.AddrInfo{ID: fakeID, Addrs: []ma.Multiaddr{addr}}); err == nil {
		return p2p.discoveredPeer(fakeID, addr.String())
	} else {
		var mismatch sec.ErrPeerIDMismatch
		if !errors.As(err, &mismatch) || mismatch.Actual == "" {
			return nil, err
		}
		actualID := mismatch.Actual
		p2p.pendingPeers.Set(actualID.String(), true)
		p2p.addPeerAddr(actualID, addr)
		if err := p2p.host.Connect(ctx, peer.AddrInfo{ID: actualID, Addrs: []ma.Multiaddr{addr}}); err != nil {
			p2p.pendingPeers.Delete(actualID.String())
			return nil, err
		}
		return p2p.discoveredPeer(actualID, addr.String())
	}
}

func (p2p *P2P) discoveredPeer(peerID peer.ID, addr string) (*DiscoveredPeer, error) {
	pubKey := p2p.remotePublicKey(peerID)
	if pubKey == nil {
		return nil, fmt.Errorf("public key for discovered peer %s not found", peerID.String())
	}
	publicKey, fingerprint, err := encodePeerPublicKey(pubKey)
	if err != nil {
		return nil, err
	}
	return &DiscoveredPeer{
		ID:          peerID.String(),
		PublicKey:   publicKey,
		Fingerprint: fingerprint,
		Address:     addr,
	}, nil
}

func (p2p *P2P) remotePublicKey(peerID peer.ID) crypto.PubKey {
	if p2p == nil || p2p.host == nil {
		return nil
	}
	if pubKey := p2p.host.Peerstore().PubKey(peerID); pubKey != nil {
		return pubKey
	}
	for _, conn := range p2p.host.Network().ConnsToPeer(peerID) {
		if pubKey := conn.RemotePublicKey(); pubKey != nil {
			_ = p2p.host.Peerstore().AddPubKey(peerID, pubKey)
			return pubKey
		}
	}
	return nil
}

func encodePeerPublicKey(pubKey crypto.PubKey) (string, string, error) {
	if pubKey.Type() != crypto.Ed25519 {
		return "", "", fmt.Errorf("unsupported peer public key type %s", pubKey.Type().String())
	}
	raw, err := pubKey.Raw()
	if err != nil {
		return "", "", fmt.Errorf("extract peer public key: %w", err)
	}
	marshaled, err := crypto.MarshalPublicKey(pubKey)
	if err != nil {
		return "", "", fmt.Errorf("marshal peer public key: %w", err)
	}
	sum := sha256.Sum256(marshaled)
	return base64.StdEncoding.EncodeToString(raw), "SHA256:" + base64.RawStdEncoding.EncodeToString(sum[:]), nil
}

func randomPeerID() (peer.ID, error) {
	_, pubKey, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return "", err
	}
	return peer.IDFromPublicKey(pubKey)
}

func (p2p *P2P) addPeerAddr(peerID peer.ID, addr ma.Multiaddr) {
	p2p.host.Peerstore().ClearAddrs(peerID)
	p2p.host.Peerstore().AddAddrs(peerID, []ma.Multiaddr{addr}, peerstore.TempAddrTTL)
	if swarm, ok := p2p.host.Network().(*libp2pswarm.Swarm); ok {
		swarm.Backoff().Clear(peerID)
	}
}

func (p2p *P2P) forgetPeer(peerID peer.ID) {
	if p2p == nil || p2p.host == nil || peerID == "" {
		return
	}
	if swarm, ok := p2p.host.Network().(*libp2pswarm.Swarm); ok {
		swarm.Backoff().Clear(peerID)
	}
	p2p.host.Peerstore().ClearAddrs(peerID)
	p2p.host.Peerstore().RemovePeer(peerID)
}

func (p2p *P2P) tofuTransportAddrs(target string) ([]string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("bootstrap target is empty")
	}
	if strings.HasPrefix(target, "/") {
		return []string{target}, nil
	}

	host := target
	port := p2p.listenPort()
	if parsedHost, parsedPort, err := net.SplitHostPort(target); err == nil {
		host = parsedHost
		p, err := strconv.Atoi(parsedPort)
		if err != nil {
			return nil, fmt.Errorf("invalid bootstrap port %q: %w", parsedPort, err)
		}
		port = p
	}
	if port <= 0 {
		return nil, fmt.Errorf("p2p port is not configured")
	}

	if ip := net.ParseIP(host); ip != nil {
		if ip.To4() == nil {
			return []string{fmt.Sprintf("/ip6/%s/udp/%d/quic-v1", ip.String(), port)}, nil
		}
		return []string{fmt.Sprintf("/ip4/%s/udp/%d/quic-v1", ip.String(), port)}, nil
	}
	return []string{fmt.Sprintf("/dns4/%s/udp/%d/quic-v1", host, port)}, nil
}
