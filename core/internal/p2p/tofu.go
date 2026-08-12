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
	sec "github.com/libp2p/go-libp2p/core/sec"
	libp2pswarm "github.com/libp2p/go-libp2p/p2p/net/swarm"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/protosio/protos/internal/swarmionlink"
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
	learnedPeers := map[string]peer.ID{}
	delay := 2 * time.Second
	for {
		for _, addr := range addrs {
			attemptCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			var discovered *DiscoveredPeer
			var err error
			if learnedPeer, ok := learnedPeers[addr]; ok {
				discovered, err = p2p.connectKnownPeerAt(attemptCtx, addr, learnedPeer)
			} else {
				discovered, err = p2p.DiscoverPeerAt(attemptCtx, addr)
			}
			cancel()
			if err == nil {
				return discovered, nil
			}
			var learnedErr *discoveredPeerDialError
			if errors.As(err, &learnedErr) {
				learnedPeers[addr] = learnedErr.peerID
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

	fakeID, err := p2p.identityProbePeerID()
	if err != nil {
		return nil, err
	}
	var identityProbe *swarmionlink.IdentityProbe
	if p2p.routeFence != nil {
		identityProbe, err = p2p.routeFence.BeginIdentityProbe(fakeID)
		if err != nil {
			return nil, fmt.Errorf("begin peer identity probe: %w", err)
		}
		defer identityProbe.Close()
	}

	p2p.addPeerAddr(fakeID, addr)
	if err := p2p.host.Connect(ctx, peer.AddrInfo{ID: fakeID, Addrs: []ma.Multiaddr{addr}}); err == nil {
		defer p2p.forgetEphemeralDiscoveryPeer(fakeID)
		return p2p.discoveredPeer(fakeID, addr.String())
	} else {
		p2p.forgetEphemeralDiscoveryPeer(fakeID)
		var mismatch sec.ErrPeerIDMismatch
		if !errors.As(err, &mismatch) || mismatch.Actual == "" {
			return nil, err
		}
		actualID := mismatch.Actual
		if p2p.afterIdentityLearnedForTest != nil {
			p2p.afterIdentityLearnedForTest(actualID)
		}
		discovered, err := p2p.connectKnownPeerAt(ctx, addr.String(), actualID)
		if err != nil {
			return nil, &discoveredPeerDialError{peerID: actualID, addr: addr.String(), err: err}
		}
		return discovered, nil
	}
}

func (p2p *P2P) identityProbePeerID() (peer.ID, error) {
	if p2p != nil && p2p.newIdentityProbePeerID != nil {
		return p2p.newIdentityProbePeerID()
	}
	return randomPeerID()
}

type discoveredPeerDialError struct {
	peerID peer.ID
	addr   string
	err    error
}

func (e *discoveredPeerDialError) Error() string {
	return fmt.Sprintf("discovered peer %s at %s but follow-up dial failed: %v", e.peerID.String(), e.addr, e.err)
}

func (e *discoveredPeerDialError) Unwrap() error {
	return e.err
}

func (p2p *P2P) connectKnownPeerAt(ctx context.Context, transportAddr string, peerID peer.ID) (*DiscoveredPeer, error) {
	addr, err := ma.NewMultiaddr(strings.TrimSpace(transportAddr))
	if err != nil {
		return nil, fmt.Errorf("parse bootstrap address: %w", err)
	}
	addr, _ = peer.SplitAddr(addr)
	if addr == nil {
		return nil, fmt.Errorf("bootstrap address %q has no transport component", transportAddr)
	}
	var admission *swarmionlink.TemporaryPeerAdmission
	if p2p.routeFence != nil {
		admission, err = p2p.routeFence.BeginTemporaryPeerAdmission(peerID)
		if err != nil {
			return nil, fmt.Errorf("temporarily admit discovered peer %s: %w", peerID, err)
		}
	}
	cleanup := func(cause error) error {
		if admission != nil {
			admission.Close()
		}
		if p2p.host == nil || p2p.host.Network() == nil {
			return cause
		}
		if p2p.routeFence != nil && p2p.routeFence.IsPeerConnectionAllowed(peerID) {
			return cause
		}
		if closeErr := p2p.host.Network().ClosePeer(peerID); closeErr != nil {
			return fmt.Errorf("%w (close rejected discovered-peer route: %w)", cause, closeErr)
		}
		if count := len(p2p.host.Network().ConnsToPeer(peerID)); count != 0 {
			return fmt.Errorf("%w (discovered peer retained %d physical connection(s) after rejection)", cause, count)
		}
		return cause
	}
	p2p.addPeerAddr(peerID, addr)
	if err := p2p.host.Connect(ctx, peer.AddrInfo{ID: peerID, Addrs: []ma.Multiaddr{addr}}); err != nil {
		return nil, cleanup(err)
	}
	discovered, err := p2p.discoveredPeer(peerID, addr.String())
	if err != nil {
		return nil, cleanup(err)
	}
	if admission != nil {
		if err := admission.Promote(); err != nil {
			return nil, cleanup(fmt.Errorf("promote discovered peer %s: %w", peerID, err))
		}
	}
	p2p.pendingPeers.Set(peerID.String(), true)
	return discovered, nil
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
	if pubKey, err := peerID.ExtractPublicKey(); err == nil {
		_ = p2p.host.Peerstore().AddPubKey(peerID, pubKey)
		return pubKey
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
	p2p.addPeerEndpointHints(peer.AddrInfo{ID: peerID, Addrs: []ma.Multiaddr{addr}})
}

// forgetEphemeralDiscoveryPeer removes only the random fake identity created
// by DiscoverPeerAt in this call. Known/authenticated peers share their
// peerstore and dial backoff with Swarmion and are never cleared here.
func (p2p *P2P) forgetEphemeralDiscoveryPeer(peerID peer.ID) {
	if p2p == nil || p2p.host == nil || peerID == "" {
		return
	}
	// The placeholder is random and call-owned, unlike a learned/authenticated
	// peer. Clear all of its address and dial-backoff state so repeated discovery
	// cannot accumulate unreachable fake identities on the shared host.
	p2p.host.Peerstore().ClearAddrs(peerID)
	p2p.host.Peerstore().RemovePeer(peerID)
	if swarm, ok := p2p.host.Network().(*libp2pswarm.Swarm); ok {
		swarm.Backoff().Clear(peerID)
	}
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

	addTCPAddrs := func(prefix string) []string {
		return []string{fmt.Sprintf("%s/tcp/%d", prefix, port)}
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.To4() == nil {
			addrs := addTCPAddrs(fmt.Sprintf("/ip6/%s", ip.String()))
			addrs = append(addrs, fmt.Sprintf("/ip6/%s/udp/%d/quic-v1", ip.String(), port))
			return addrs, nil
		}
		addrs := addTCPAddrs(fmt.Sprintf("/ip4/%s", ip.String()))
		addrs = append(addrs, fmt.Sprintf("/ip4/%s/udp/%d/quic-v1", ip.String(), port))
		return addrs, nil
	}
	addrs := addTCPAddrs(fmt.Sprintf("/dns4/%s", host))
	addrs = append(addrs, fmt.Sprintf("/dns4/%s/udp/%d/quic-v1", host, port))
	return addrs, nil
}
