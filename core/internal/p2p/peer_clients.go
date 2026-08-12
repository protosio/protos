package p2p

import (
	"context"
	"time"
)

const (
	peerClientRefreshRetryDelay       = 2 * time.Second
	peerClientReconnectAttemptTimeout = 30 * time.Second
)

func (p2p *P2P) controlClientReadinessTimeout() time.Duration {
	if p2p != nil && p2p.controlClientTimeout > 0 {
		return p2p.controlClientTimeout
	}
	return peerClientReconnectAttemptTimeout
}

type peerClientSelection struct {
	clients                    map[string]*Client
	knownPeerCount             int
	disconnectedKnownPeerCount int
}

func (p2p *P2P) currentImageResolutionClients() peerClientSelection {
	return p2p.currentPeerClientSelection(true)
}

func (p2p *P2P) imageResolutionClients(ctx context.Context, refreshBudget time.Duration) peerClientSelection {
	return p2p.peerClients(ctx, peerClientRefreshOptions{
		refreshBudget: refreshBudget,
		requireImages: true,
	})
}

type peerClientRefreshOptions struct {
	refreshBudget time.Duration
	requireImages bool
}

// peerClients is the lower-level peer client selector. It keeps RPC client
// freshness and reconnect attempts out of feature code such as image resolution.
func (p2p *P2P) peerClients(ctx context.Context, options peerClientRefreshOptions) peerClientSelection {
	selection := p2p.currentPeerClientSelection(options.requireImages)
	if p2p == nil || p2p.host == nil || options.refreshBudget <= 0 || selection.disconnectedKnownPeerCount == 0 {
		return selection
	}

	refreshCtx, cancel := context.WithTimeout(ctx, options.refreshBudget)
	defer cancel()

	p2p.requestReconcile()
	for selection.disconnectedKnownPeerCount > 0 && refreshCtx.Err() == nil {
		log.Debugf("refreshing %d disconnected known peer(s) before selecting RPC clients", selection.disconnectedKnownPeerCount)
		p2p.refreshDisconnectedPeerClients(refreshCtx)
		selection = p2p.currentPeerClientSelection(options.requireImages)
		if selection.disconnectedKnownPeerCount == 0 {
			break
		}
		select {
		case <-refreshCtx.Done():
			return selection
		case <-time.After(peerClientRefreshRetryDelay):
		}
	}
	return selection
}

func (p2p *P2P) currentPeerClientSelection(requireImages bool) peerClientSelection {
	if p2p == nil {
		return peerClientSelection{}
	}
	connectedClients := p2p.connectedClientsSnapshot()
	knownPeers := p2p.machines.Snapshot()
	clients := connectedClients
	if requireImages {
		clients = imageCapableClients(connectedClients)
	}
	return peerClientSelection{
		clients:                    clients,
		knownPeerCount:             len(knownPeers),
		disconnectedKnownPeerCount: disconnectedKnownPeerCount(knownPeers, connectedClients),
	}
}

func (p2p *P2P) connectedClientsSnapshot() map[string]*Client {
	if p2p == nil {
		return nil
	}
	clients := map[string]*Client{}
	for peerID := range p2p.clients.Snapshot() {
		client, found := p2p.connectedClient(peerID)
		if found {
			clients[peerID] = client
		}
	}
	return clients
}

func imageCapableClients(clients map[string]*Client) map[string]*Client {
	out := map[string]*Client{}
	for peerID, client := range clients {
		if client == nil || client.ImagesClient == nil || !client.supportsCapability(peerCapabilityImageContent) {
			continue
		}
		out[peerID] = client
	}
	return out
}

func disconnectedKnownPeerCount(knownPeers map[string]Machine, connectedClients map[string]*Client) int {
	if len(knownPeers) == 0 {
		return 0
	}
	var count int
	for peerID := range knownPeers {
		if _, found := connectedClients[peerID]; !found {
			count++
		}
	}
	return count
}

func (p2p *P2P) refreshDisconnectedPeerClients(ctx context.Context) {
	if p2p == nil || p2p.host == nil {
		return
	}
	connectedClients := p2p.connectedClientsSnapshot()
	for peerID, machine := range p2p.machines.Snapshot() {
		if _, found := connectedClients[peerID]; found {
			continue
		}
		attemptCtx, cancel := context.WithTimeout(ctx, peerClientReconnectAttemptTimeout)
		_, err := p2p.connectPeer(attemptCtx, peerID, machine)
		cancel()
		if err != nil {
			log.Debugf("peer client refresh failed to reconnect peer %s: %v", peerID, err)
		}
		if ctx.Err() != nil {
			return
		}
	}
}

func (p2p *P2P) pruneStaleClients() {
	if p2p == nil {
		return
	}
	_ = p2p.connectedClientsSnapshot()
}
