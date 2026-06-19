package p2p

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	p2pgrpc "github.com/birros/go-libp2p-grpc"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	connmgr "github.com/libp2p/go-libp2p/p2p/net/connmgr"
	libp2pswarm "github.com/libp2p/go-libp2p/p2p/net/swarm"
	relayclient "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	noise "github.com/libp2p/go-libp2p/p2p/security/noise"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/multiformats/go-multiaddr"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/imageregistry"
	networkmodule "github.com/protosio/protos/internal/network/module"
	p2pproto "github.com/protosio/protos/internal/p2p/proto"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/tasks"
	"github.com/protosio/protos/internal/util"
	"google.golang.org/grpc"
)

var log = util.GetLogger("p2p")

const (
	protosRPCProtocol               = "/protos/rpc/0.0.1"
	swarmionBootstrapProtocol       = "/protos/swarmion-bootstrap/0.0.1"
	imageBlobStreamProtocol         = "/protos/image-blob/0.0.1"
	imageArchiveUploadProtocol      = "/protos/image-archive-upload/0.0.1"
	peerCapabilityImageContent      = "image.content"
	peerCapabilitySwarmionTransport = "swarmion.transport"
	destinationQUICIPv4Template     = "/ip4/%s/udp/%d/quic-v1/p2p/%s"
	destinationQUICIPv6Template     = "/ip6/%s/udp/%d/quic-v1/p2p/%s"
	destinationTCPIPv4Template      = "/ip4/%s/tcp/%d/p2p/%s"
	destinationTCPIPv6Template      = "/ip6/%s/tcp/%d/p2p/%s"
	peerRetryInterval               = 10 * time.Second
	peerRetryMaxBackoff             = time.Minute
	peerConnectDialTimeout          = 10 * time.Second
)

type AppManager interface {
	GetLogs(name string) ([]byte, error)
	GetStatus(name string) (string, error)
}

type ImageManager interface {
	DescribeImage(ctx context.Context, imageRef string) (imageRefOut string, targetDigest string, platform string, labels map[string]string, found bool, err error)
	GetImageContent(ctx context.Context, imageRef string) (imageregistry.ImageContent, bool, error)
	OpenImageBlob(ctx context.Context, digest string) (imageregistry.ImageBlobReader, error)
	MissingImageContent(ctx context.Context, descriptors []imageregistry.Descriptor) ([]imageregistry.Descriptor, error)
	EnsureImageContent(ctx context.Context, request imageregistry.ImageContentImport, progress func(int, string, any) error) error
	LoadImageArchive(ctx context.Context, archivePath string, imageRef string) (imageregistry.LoadedImage, error)
}

type Machine interface {
	GetID() string
	GetPublicKey() string
	GetPublicIP() string
	GetName() string
}

type internalIPMachine interface {
	GetInternalIP() string
}

type peerDBConnector interface {
	ConnectPeer(peerID string, publicIP string) error
}

type peerDBIPConnector interface {
	ConnectPeerIPs(peerID string, ips []string) error
}

type NetworkInspector interface {
	State() (networkmodule.State, error)
}

// Client is a remote p2p client
type Client struct {
	p2pproto.PingerClient
	p2pproto.PeerDBClient
	p2pproto.AppsClient
	p2pproto.ImagesClient
	p2pproto.InstanceClient

	capabilitiesMu sync.RWMutex
	capabilities   map[string]struct{}

	grpcConnection *grpc.ClientConn
}

func (client *Client) setCapabilities(capabilities []string) {
	if client == nil {
		return
	}
	next := map[string]struct{}{}
	for _, capability := range capabilities {
		capability = strings.TrimSpace(capability)
		if capability != "" {
			next[capability] = struct{}{}
		}
	}
	client.capabilitiesMu.Lock()
	client.capabilities = next
	client.capabilitiesMu.Unlock()
}

func (client *Client) supportsCapability(capability string) bool {
	if client == nil {
		return false
	}
	client.capabilitiesMu.RLock()
	defer client.capabilitiesMu.RUnlock()
	_, found := client.capabilities[capability]
	return found
}

type PeerStatus string

const (
	PeerStatusDesired      PeerStatus = "desired"
	PeerStatusConnecting   PeerStatus = "connecting"
	PeerStatusConnected    PeerStatus = "connected"
	PeerStatusUnreachable  PeerStatus = "unreachable"
	PeerStatusDisconnected PeerStatus = "disconnected"
)

type PeerState struct {
	PeerID      string
	MachineID   string
	Name        string
	PublicIP    string
	Status      PeerStatus
	LastAttempt time.Time
	LastSeen    time.Time
	LastError   string
	Attempts    int
}

func (state PeerState) Reachability() string {
	switch state.Status {
	case PeerStatusConnected:
		return "reachable"
	case PeerStatusDesired:
		return string(PeerStatusConnecting)
	case "":
		return string(PeerStatusConnecting)
	default:
		return string(state.Status)
	}
}

type P2P struct {
	host         host.Host
	appManager   AppManager
	imageManager ImageManager
	taskManager  *tasks.Manager
	grpcServer   *grpc.Server
	externalDB   ExternalDB
	network      NetworkInspector
	p2pPort      int

	restartServerSignal chan initMachine

	initMu     sync.RWMutex
	initPeerID peer.ID
	grpcMu     sync.Mutex
	grpcGen    uint64

	// the index is the peer ID
	machines *util.Map[string, Machine]
	// the index is the peer ID
	clients *util.Map[string, *Client]
	// the index is the peer ID
	peerStates *util.Map[string, PeerState]
	// the index is the peer ID. Pending peers are temporary deploy/init
	// connections that are not in the declarative DB yet.
	pendingPeers *util.Map[string, bool]
	// the index is the relay peer ID.
	relayReservations *util.Map[string, time.Time]

	reconcileCh chan struct{}
	stopCh      chan struct{}
	stopOnce    sync.Once
}

func (p2p *P2P) SetNetworkInspector(network NetworkInspector) {
	if p2p == nil {
		return
	}
	p2p.network = network
}

func (p2p *P2P) SetImageManager(imageManager ImageManager) {
	if p2p == nil {
		return
	}
	p2p.imageManager = imageManager
}

func (p2p *P2P) SetTaskManager(taskManager *tasks.Manager) {
	if p2p == nil {
		return
	}
	p2p.taskManager = taskManager
}

func (p2p *P2P) PeerID() string {
	if p2p == nil || p2p.host == nil {
		return ""
	}
	return p2p.host.ID().String()
}

func (p2p *P2P) ListenAddresses() []string {
	if p2p == nil || p2p.host == nil {
		return nil
	}
	addrs := p2p.host.Addrs()
	out := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		out = append(out, addr.String())
	}
	return out
}

func (p2p *P2P) listenPort() int {
	if p2p != nil && p2p.p2pPort != 0 {
		return p2p.p2pPort
	}
	return config.Get().P2PPort
}

func listenAddrsForPort(port int) []string {
	return []string{
		fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port),
		fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", port),
		fmt.Sprintf("/ip6/::/tcp/%d", port),
		fmt.Sprintf("/ip6/::/udp/%d/quic-v1", port),
	}
}

func (p2p *P2P) swarmionPort() int {
	port := p2p.listenPort()
	if port <= 0 {
		return 0
	}
	return port + 1
}

// GetPeerID adds a peer to the p2p manager
func (p2p *P2P) pubKeyToPeerID(pubKey string) (peer.ID, error) {
	pubKeyBytes, err := base64.StdEncoding.DecodeString(pubKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode public key: %w", err)
	}

	pk, err := crypto.UnmarshalEd25519PublicKey(pubKeyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshall public key: %w", err)
	}

	peerID, err := peer.IDFromPublicKey(pk)
	if err != nil {
		return "", fmt.Errorf("failed to create peer ID from public key: %w", err)
	}
	return peerID, nil
}

func (p2p *P2P) GetClient(ctx context.Context, id string) (*Client, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if p2p == nil {
		return nil, fmt.Errorf("p2p manager is not configured")
	}
	client, found := p2p.connectedClient(id)
	if found {
		return client, nil
	}

	machinePeerID, machine, found := p2p.resolveMachineByIdentity(id)
	if !found {
		return nil, fmt.Errorf("could not find RPC client for instance '%s'", id)
	}
	if client, found = p2p.connectedClient(machinePeerID); found {
		return client, nil
	}
	client, err := p2p.connectPeer(ctx, machinePeerID, machine)
	if err != nil {
		p2p.requestReconcile()
		return nil, fmt.Errorf("could not connect RPC client for instance '%s': %w", id, err)
	}
	if client == nil {
		p2p.requestReconcile()
		return nil, fmt.Errorf("could not find RPC client for instance '%s'", id)
	}
	return client, nil
}

func (p2p *P2P) resolveMachineByIdentity(id string) (string, Machine, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", nil, false
	}
	for peerID, machine := range p2p.machines.Snapshot() {
		if peerID == id || machine.GetID() == id || machine.GetName() == id {
			return peerID, machine, true
		}
	}
	return "", nil, false
}

func (p2p *P2P) connectedClient(peerIDString string) (*Client, bool) {
	if p2p == nil || p2p.host == nil {
		return nil, false
	}
	client, found := p2p.clients.Get(peerIDString)
	if !found {
		return nil, false
	}
	peerID, err := peer.Decode(peerIDString)
	if err != nil {
		p2p.clients.Delete(peerIDString)
		return nil, false
	}
	if !usablePeerConnectedness(p2p.host.Network().Connectedness(peerID)) {
		p2p.clients.Delete(peerIDString)
		p2p.markPeerDisconnected(peerIDString)
		p2p.requestReconcile()
		return nil, false
	}
	return client, true
}

func usablePeerConnectedness(connectedness network.Connectedness) bool {
	switch connectedness {
	case network.Connected, network.Limited:
		return true
	default:
		return false
	}
}

func (p2p *P2P) GetPeerState(id string) (PeerState, bool) {
	if state, found := p2p.peerStates.Get(id); found {
		return state, true
	}
	for peerID, machine := range p2p.machines.Snapshot() {
		if machine.GetID() != id && machine.GetName() != id {
			continue
		}
		state, found := p2p.peerStates.Get(peerID)
		if found {
			return state, true
		}
		return PeerState{
			PeerID:    peerID,
			MachineID: machine.GetID(),
			Name:      machine.GetName(),
			PublicIP:  machine.GetPublicIP(),
			Status:    PeerStatusDesired,
		}, true
	}
	return PeerState{}, false
}

func (p2p *P2P) RequestReconnect(machine Machine) error {
	if machine == nil {
		return nil
	}
	peerID, err := p2p.trackPeer(machine)
	if err != nil {
		return err
	}
	p2p.updatePeerState(peerID, func(state *PeerState) {
		state.Status = PeerStatusDesired
		state.LastAttempt = time.Time{}
		state.LastError = ""
	})
	p2p.requestReconcile()
	return nil
}

//
// Methods for handling new peers
//

// ConfigurePeers configures all the peers passed as arguemnt
func (p2p *P2P) ConfigurePeers(machines []Machine) error {
	currentMachines := map[string]Machine{}

	log.Debugf("configuring p2p peers: %v", machines)

	for _, machine := range machines {
		localPeerID, err := p2p.pubKeyToPeerID(machine.GetPublicKey())
		if err != nil {
			return fmt.Errorf("failed to get peer ID from public key: %w", err)
		}
		localPeerIDString := localPeerID.String()
		if p2p.isLocalPeerID(localPeerIDString) {
			p2p.machines.Delete(localPeerIDString)
			p2p.peerStates.Delete(localPeerIDString)
			p2p.clients.Delete(localPeerIDString)
			p2p.pendingPeers.Delete(localPeerIDString)
			continue
		}
		peerID, err := p2p.trackPeer(machine)
		if err != nil {
			return err
		}
		currentMachines[peerID] = machine
		p2p.pendingPeers.Delete(peerID)
	}

	for peerID := range p2p.machines.Snapshot() {
		if _, found := currentMachines[peerID]; !found {
			if pending, _ := p2p.pendingPeers.Get(peerID); pending {
				continue
			}
			p2p.machines.Delete(peerID)
			p2p.peerStates.Delete(peerID)
		}
	}

	for peerIDString := range p2p.clients.Snapshot() {
		if _, found := currentMachines[peerIDString]; !found {
			if pending, _ := p2p.pendingPeers.Get(peerIDString); pending {
				continue
			}
			peerID, err := peer.Decode(peerIDString)
			if err != nil {
				log.Debugf("failed to decode stale peer %s: %v", peerIDString, err)
				p2p.clients.Delete(peerIDString)
				continue
			}
			err = p2p.host.Network().ClosePeer(peerID)
			if err != nil {
				log.Debugf("failed to remove peer %s: %v", peerIDString, err)
			}
			p2p.clients.Delete(peerIDString)
			p2p.removeExternalDBPeer(peerIDString)
		}
	}

	p2p.requestReconcile()
	return nil
}

func (p2p *P2P) isLocalPeerID(peerID string) bool {
	peerID = strings.TrimSpace(peerID)
	return peerID != "" && peerID == strings.TrimSpace(p2p.PeerID())
}

// AddPeer adds a peer to the p2p manager
func (p2p *P2P) AddPeer(machine Machine) (*Client, error) {
	peerIDString, err := p2p.trackPeer(machine)
	if err != nil {
		return nil, err
	}
	p2p.pendingPeers.Set(peerIDString, true)

	client, err := p2p.connectPeer(context.Background(), peerIDString, machine)
	if err != nil {
		p2p.requestReconcile()
	}
	return client, err
}

func (p2p *P2P) connectExternalDBPeer(peerIDString string, machine Machine, client *Client) {
	if p2p == nil || p2p.externalDB == nil || machine == nil {
		return
	}
	if client == nil || !client.supportsCapability(peerCapabilitySwarmionTransport) {
		return
	}
	ips := p2p.knownPeerIPStrings(machine)
	if len(ips) == 0 {
		return
	}
	peerID, err := p2p.pubKeyToPeerID(machine.GetPublicKey())
	if err != nil {
		log.Debugf("failed to get DB peer ID for %s: %v", machine.GetID(), err)
		return
	}
	if peerIDString = strings.TrimSpace(peerIDString); peerIDString != "" && peerID.String() != peerIDString {
		log.Debugf("skipping DB peer connect for %s: machine public key resolves to peer %s", peerIDString, peerID.String())
		return
	}

	if connector, ok := p2p.externalDB.(peerDBIPConnector); ok {
		if err := connector.ConnectPeerIPs(peerID.String(), ips); err != nil {
			log.Debugf("failed to connect DB peer %s: %v", peerID.String(), err)
		}
		return
	}

	publicIP := strings.TrimSpace(machine.GetPublicIP())
	if publicIP == "" {
		return
	}
	connector, ok := p2p.externalDB.(peerDBConnector)
	if !ok {
		return
	}
	if err := connector.ConnectPeer(peerID.String(), publicIP); err != nil {
		log.Debugf("failed to connect DB peer %s: %v", peerID.String(), err)
	}
}

func (p2p *P2P) trackPeer(machine Machine) (string, error) {
	if machine == nil {
		return "", fmt.Errorf("machine is nil")
	}
	peerID, err := p2p.pubKeyToPeerID(machine.GetPublicKey())
	if err != nil {
		return "", fmt.Errorf("failed to get peer ID from public key: %w", err)
	}
	peerIDString := peerID.String()
	p2p.machines.Set(peerIDString, machine)
	p2p.updatePeerState(peerIDString, func(state *PeerState) {
		state.PeerID = peerIDString
		state.MachineID = machine.GetID()
		state.Name = machine.GetName()
		state.PublicIP = machine.GetPublicIP()
		if state.Status == "" {
			state.Status = PeerStatusDesired
		}
	})
	return peerIDString, nil
}

func (p2p *P2P) connectPeer(ctx context.Context, peerIDString string, machine Machine) (*Client, error) {
	if p2p == nil || p2p.host == nil {
		return nil, fmt.Errorf("p2p host is not configured")
	}
	client, found := p2p.connectedClient(peerIDString)
	if found {
		p2p.markPeerConnected(peerIDString, machine)
		p2p.connectExternalDBPeer(peerIDString, machine, client)
		return client, nil
	}

	destinations := p2p.destinationStrings(peerIDString, machine)
	if len(destinations) == 0 {
		p2p.markPeerFailed(peerIDString, machine, fmt.Errorf("no usable peer addresses"))
		return nil, nil
	}

	p2p.updatePeerState(peerIDString, func(state *PeerState) {
		state.Status = PeerStatusConnecting
		state.LastAttempt = time.Now()
		state.LastError = ""
		state.Attempts++
	})

	peerInfo, err := peerAddrInfoFromDestinations(peerIDString, destinations)
	if err != nil {
		p2p.markPeerFailed(peerIDString, machine, err)
		return nil, fmt.Errorf("failed to connect to peer %s: %w", peerIDString, err)
	}
	client, err = p2p.connectPeerWithAddrInfo(ctx, peerIDString, machine, peerInfo, destinations)
	if err != nil {
		p2p.markPeerFailed(peerIDString, machine, err)
		return nil, fmt.Errorf("failed to connect to peer %s: %w", peerIDString, err)
	}
	return client, nil
}

func (p2p *P2P) connectPeerWithAddrInfo(ctx context.Context, peerIDString string, machine Machine, peerInfo peer.AddrInfo, destinations []string) (*Client, error) {
	p2p.host.Peerstore().ClearAddrs(peerInfo.ID)
	p2p.host.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, peerstore.TempAddrTTL)
	if swarm, ok := p2p.host.Network().(*libp2pswarm.Swarm); ok {
		swarm.Backoff().Clear(peerInfo.ID)
	}

	log.Debugf("dialing peer id %s using %d destination(s): %s", peerIDString, len(destinations), strings.Join(destinations, ", "))

	connectCtx, cancel := context.WithTimeout(ctx, peerConnectDialTimeout)
	defer cancel()

	if err := p2p.host.Connect(connectCtx, peerInfo); err != nil {
		return nil, err
	}

	if client, found := p2p.connectedClient(peerIDString); found {
		p2p.markPeerConnected(peerIDString, machine)
		p2p.connectExternalDBPeer(peerIDString, machine, client)
		return client, nil
	}

	client, err := p2p.createClientForPeer(ctx, peerInfo.ID)
	if err != nil {
		return nil, err
	}
	p2p.clients.Set(peerIDString, client)
	p2p.markPeerConnected(peerIDString, machine)
	p2p.connectExternalDBPeer(peerIDString, machine, client)
	if p2p.externalDB != nil {
		if err := p2p.externalDB.AddPeer(peerIDString, client.grpcConnection); err != nil {
			return nil, err
		}
	}
	return client, nil
}

func peerAddrInfoFromDestinations(peerIDString string, destinations []string) (peer.AddrInfo, error) {
	peerID, err := peer.Decode(peerIDString)
	if err != nil {
		return peer.AddrInfo{}, fmt.Errorf("failed to decode peer ID %s: %w", peerIDString, err)
	}

	seen := map[string]struct{}{}
	addrs := make([]multiaddr.Multiaddr, 0, len(destinations))
	var parseErrors []string
	for _, destinationString := range destinations {
		maddr, err := multiaddr.NewMultiaddr(destinationString)
		if err != nil {
			parseErrors = append(parseErrors, fmt.Sprintf("%s: %v", destinationString, err))
			continue
		}
		peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			parseErrors = append(parseErrors, fmt.Sprintf("%s: %v", destinationString, err))
			continue
		}
		if peerInfo.ID != peerID {
			parseErrors = append(parseErrors, fmt.Sprintf("%s: destination peer %s does not match %s", destinationString, peerInfo.ID, peerIDString))
			continue
		}
		for _, addr := range peerInfo.Addrs {
			key := addr.String()
			if _, found := seen[key]; found {
				continue
			}
			seen[key] = struct{}{}
			addrs = append(addrs, addr)
		}
	}
	if len(addrs) == 0 {
		if len(parseErrors) == 0 {
			return peer.AddrInfo{}, fmt.Errorf("no usable peer addresses")
		}
		return peer.AddrInfo{}, fmt.Errorf("no usable peer addresses: %s", strings.Join(parseErrors, "; "))
	}
	return peer.AddrInfo{ID: peerID, Addrs: addrs}, nil
}

func (p2p *P2P) destinationStrings(peerIDString string, machine Machine) []string {
	var destinations []string
	seen := map[string]struct{}{}
	add := func(destination string) {
		if destination == "" {
			return
		}
		if _, found := seen[destination]; found {
			return
		}
		seen[destination] = struct{}{}
		destinations = append(destinations, destination)
	}

	for _, ip := range p2p.knownPeerIPs(machine) {
		if ip.To4() == nil {
			add(fmt.Sprintf(destinationTCPIPv6Template, ip.String(), p2p.listenPort(), peerIDString))
			add(fmt.Sprintf(destinationQUICIPv6Template, ip.String(), p2p.listenPort(), peerIDString))
		} else {
			add(fmt.Sprintf(destinationTCPIPv4Template, ip.String(), p2p.listenPort(), peerIDString))
			add(fmt.Sprintf(destinationQUICIPv4Template, ip.String(), p2p.listenPort(), peerIDString))
		}
	}

	for relayID, relayMachine := range p2p.machines.Snapshot() {
		if relayID == peerIDString || strings.TrimSpace(relayMachine.GetPublicIP()) != "" {
			continue
		}
		if _, connected := p2p.clients.Get(relayID); !connected {
			continue
		}
		relayPeerID, err := peer.Decode(relayID)
		if err != nil {
			log.Debugf("failed to decode relay peer %s: %v", relayID, err)
			continue
		}
		relaySuffix, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s/p2p-circuit/p2p/%s", relayID, peerIDString))
		if err != nil {
			log.Debugf("failed to build relay address through peer %s: %v", relayID, err)
			continue
		}
		for _, conn := range p2p.host.Network().ConnsToPeer(relayPeerID) {
			add(conn.RemoteMultiaddr().Encapsulate(relaySuffix).String())
		}
		for _, relayAddr := range p2p.host.Peerstore().Addrs(relayPeerID) {
			add(relayAddr.Encapsulate(relaySuffix).String())
		}
		add(relaySuffix.String())
	}

	return destinations
}

func (p2p *P2P) knownPeerIPStrings(machine Machine) []string {
	ips := p2p.knownPeerIPs(machine)
	out := make([]string, 0, len(ips))
	for _, ip := range ips {
		out = append(out, ip.String())
	}
	return out
}

// knownPeerIPs is the single owner of peer transport-address ordering. Peer
// reachability is a lower-layer concern: callers (image resolver, DB transport,
// admin/APIC clients) must consume this ordered list rather than infer
// reachability from raw machine fields themselves. Ordering is overlay-first:
// (1) the internal overlay IP, (2) the derived WireGuard IPv6, then (3) the
// provider-reported public IP last. The provider public IP can actually be a
// host-only/private address (e.g. local macOS 192.168.64.x) that is reachable
// from the host but not guest-to-guest; ordering it last keeps the overlay path
// authoritative while still letting libp2p race it where it is reachable.
func (p2p *P2P) knownPeerIPs(machine Machine) []net.IP {
	if machine == nil {
		return nil
	}
	seen := map[string]struct{}{}
	var ips []net.IP
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		ip := net.ParseIP(value)
		if ip == nil {
			log.Debugf("ignoring invalid peer IP for %s: %s", machine.GetID(), value)
			return
		}
		normalized := ip.String()
		if _, found := seen[normalized]; found {
			return
		}
		seen[normalized] = struct{}{}
		ips = append(ips, ip)
	}

	if internalMachine, ok := machine.(internalIPMachine); ok {
		add(internalMachine.GetInternalIP())
	}
	if key, err := pcrypto.CreatePublicKeyFromBase64(machine.GetPublicKey()); err == nil {
		internalIP := key.IPv6Address()
		if internalIP.IsValid() && internalIP.Is6() {
			add(internalIP.String())
		}
	} else {
		log.Debugf("failed to derive internal peer IP for %s: %v", machine.GetID(), err)
	}
	add(machine.GetPublicIP())

	return ips
}

func (p2p *P2P) markPeerConnected(peerIDString string, machine Machine) {
	p2p.updatePeerState(peerIDString, func(state *PeerState) {
		state.Status = PeerStatusConnected
		state.LastSeen = time.Now()
		state.LastError = ""
		state.Attempts = 0
		if machine != nil {
			state.MachineID = machine.GetID()
			state.Name = machine.GetName()
			state.PublicIP = machine.GetPublicIP()
		}
	})
}

func (p2p *P2P) markPeerFailed(peerIDString string, machine Machine, err error) {
	p2p.updatePeerState(peerIDString, func(state *PeerState) {
		state.Status = PeerStatusUnreachable
		if machine != nil {
			state.MachineID = machine.GetID()
			state.Name = machine.GetName()
			state.PublicIP = machine.GetPublicIP()
		}
		if err != nil {
			state.LastError = err.Error()
		}
	})
}

func (p2p *P2P) markPeerDisconnected(peerIDString string) {
	p2p.updatePeerState(peerIDString, func(state *PeerState) {
		state.Status = PeerStatusDisconnected
	})
}

func (p2p *P2P) updatePeerState(peerIDString string, update func(*PeerState)) {
	state, _ := p2p.peerStates.Get(peerIDString)
	state.PeerID = peerIDString
	update(&state)
	p2p.peerStates.Set(peerIDString, state)
}

func (p2p *P2P) requestReconcile() {
	select {
	case p2p.reconcileCh <- struct{}{}:
	default:
	}
}

func (p2p *P2P) startPeerReconciler() func() error {
	go p2p.reconcileLoop()
	p2p.requestReconcile()
	return func() error {
		p2p.stopOnce.Do(func() {
			close(p2p.stopCh)
		})
		return nil
	}
}

func (p2p *P2P) reconcileLoop() {
	ticker := time.NewTicker(peerRetryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p2p.reconcileCh:
			p2p.reconcilePeers()
		case <-ticker.C:
			p2p.reconcilePeers()
		case <-p2p.stopCh:
			return
		}
	}
}

func (p2p *P2P) reconcilePeers() {
	p2p.pruneStaleClients()
	for peerIDString, machine := range p2p.machines.Snapshot() {
		if client, found := p2p.connectedClient(peerIDString); found {
			p2p.markPeerConnected(peerIDString, machine)
			p2p.connectExternalDBPeer(peerIDString, machine, client)
			p2p.ensureRelayReservation(peerIDString, machine)
			continue
		}

		state, _ := p2p.peerStates.Get(peerIDString)
		if !shouldRetryPeer(state) {
			continue
		}

		if _, err := p2p.connectPeer(context.Background(), peerIDString, machine); err != nil {
			log.Debugf("failed to reconcile peer %s: %v", peerIDString, err)
		}
	}
}

func (p2p *P2P) ensureRelayReservation(peerIDString string, machine Machine) {
	if machine == nil || strings.TrimSpace(machine.GetPublicIP()) != "" {
		return
	}
	if expiresAt, found := p2p.relayReservations.Get(peerIDString); found && time.Until(expiresAt) > time.Minute {
		return
	}

	relayID, err := peer.Decode(peerIDString)
	if err != nil {
		log.Debugf("failed to decode relay peer %s: %v", peerIDString, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reservation, err := relayclient.Reserve(ctx, p2p.host, peer.AddrInfo{ID: relayID})
	if err != nil {
		log.Debugf("failed to reserve relay slot on peer %s: %v", peerIDString, err)
		return
	}
	p2p.relayReservations.Set(peerIDString, reservation.Expiration)
	log.Debugf("reserved relay slot on peer %s until %s", peerIDString, reservation.Expiration.Format(time.RFC3339))
}

func shouldRetryPeer(state PeerState) bool {
	if state.Status == PeerStatusConnecting {
		return false
	}
	if state.LastAttempt.IsZero() {
		return true
	}
	return time.Since(state.LastAttempt) >= peerBackoff(state.Attempts)
}

func peerBackoff(attempts int) time.Duration {
	if attempts <= 0 {
		return 0
	}
	delay := 2 * time.Second
	for i := 1; i < attempts; i++ {
		delay *= 2
		if delay >= peerRetryMaxBackoff {
			return peerRetryMaxBackoff
		}
	}
	return delay
}

func (p2p *P2P) RemovePeer(machine Machine) error {
	log.Debugf("removing peer '%s'", machine.GetID())
	peerID, err := p2p.pubKeyToPeerID(machine.GetPublicKey())
	if err != nil {
		return fmt.Errorf("failed to convert public key to peer ID: %w", err)
	}
	peerIDString := peerID.String()

	if p2p.host != nil {
		if err := p2p.host.Network().ClosePeer(peerID); err != nil {
			log.Debugf("failed to close peer %s for removed machine %s: %v", peerIDString, machine.GetID(), err)
		}
	}

	p2p.machines.Delete(peerIDString)
	p2p.clients.Delete(peerIDString)
	p2p.peerStates.Delete(peerIDString)
	p2p.pendingPeers.Delete(peerIDString)
	p2p.removeExternalDBPeer(peerIDString)
	p2p.requestReconcile()

	return nil
}

func (p2p *P2P) removeExternalDBPeer(peerIDString string) {
	if p2p == nil || p2p.externalDB == nil || peerIDString == "" {
		return
	}
	if err := p2p.externalDB.RemovePeer(peerIDString); err != nil {
		log.Debugf("failed to remove DB peer %s: %v", peerIDString, err)
	}
}

//
// Methods for creating and starting the p2p server
//

func (p2p *P2P) startGRPCServer() error {
	p2p.grpcMu.Lock()
	defer p2p.grpcMu.Unlock()

	return p2p.startGRPCServerLocked()
}

func (p2p *P2P) startGRPCServerLocked() error {
	if p2p.grpcServer == nil {
		p2p.grpcServer = newP2PGRPCServer()
	}

	server := p2p.grpcServer

	// register internal grpc servers
	srv := &Server{DB: p2p.externalDB, p2p: p2p}
	p2pproto.RegisterPingerServer(server, srv)
	p2pproto.RegisterPeerDBServer(server, srv)
	p2pproto.RegisterAppsServer(server, srv)
	p2pproto.RegisterImagesServer(server, srv)
	p2pproto.RegisterInstanceServer(server, srv)

	if p2p.externalDB.Initialized() {
		err := p2p.externalDB.EnableGRPCServers(server)
		if err != nil {
			return fmt.Errorf("failed to enable grpc servers: %w", err)
		}
	}

	// serve grpc server over libp2p host
	grpcListener := p2pgrpc.NewListener(context.Background(), p2p.host, protosRPCProtocol)
	p2p.grpcGen++
	generation := p2p.grpcGen
	go func() {
		if err := server.Serve(grpcListener); err != nil {
			p2p.handleGRPCServeError(server, generation, err)
		}
	}()

	return nil
}

func (p2p *P2P) handleGRPCServeError(server *grpc.Server, generation uint64, err error) {
	select {
	case <-p2p.stopCh:
		log.Debugf("p2p grpc server stopped: %v", err)
		return
	default:
	}

	p2p.grpcMu.Lock()
	if p2p.grpcServer != server || p2p.grpcGen != generation {
		p2p.grpcMu.Unlock()
		log.Debugf("stale p2p grpc server generation %d stopped: %v", generation, err)
		return
	}
	log.Errorf("p2p grpc server stopped unexpectedly; restarting: %v", err)
	p2p.grpcServer = newP2PGRPCServer()
	p2p.grpcMu.Unlock()
	server.Stop()

	time.Sleep(time.Second)
	if restartErr := p2p.startGRPCServer(); restartErr != nil {
		log.Errorf("failed to restart p2p grpc server after serve error: %v", restartErr)
	}
}

// StartServer starts listening for p2p connections
func (p2p *P2P) StartServer() (func() error, error) {
	log.Info("starting p2p server")

	err := p2p.startGRPCServer()
	if err != nil {
		return func() error { return nil }, fmt.Errorf("failed to prepare grpc server: %w", err)
	}

	err = p2p.host.Network().Listen()
	if err != nil {
		return func() error { return nil }, fmt.Errorf("failed to listen: %w", err)
	}

	stopReconciler := p2p.startPeerReconciler()
	stopper := func() error {
		log.Debug("Stopping p2p server")
		_ = stopReconciler()
		p2p.grpcMu.Lock()
		server := p2p.grpcServer
		p2p.grpcGen++
		p2p.grpcMu.Unlock()
		if server != nil {
			server.GracefulStop()
		}
		return p2p.host.Close()
	}
	return stopper, nil

}

func (p2p *P2P) restartServerAfterInit() {
	go func() {
		im, ok := <-p2p.restartServerSignal
		if !ok {
			return
		}

		log.Info("restarting grpc server")

		p2p.grpcMu.Lock()
		oldServer := p2p.grpcServer
		p2p.grpcServer = newP2PGRPCServer()
		p2p.grpcGen++
		p2p.grpcMu.Unlock()
		if oldServer != nil {
			oldServer.GracefulStop()
		}

		peerID, err := p2p.pubKeyToPeerID(im.GetPublicKey())
		if err != nil {
			log.Error("failed to decode init peer: ", err)
		} else {
			if err := p2p.host.Network().ClosePeer(peerID); err != nil {
				log.Debugf("failed to close init peer %s: %v", peerID.String(), err)
			}
			p2p.clients.Delete(peerID.String())
			p2p.machines.Set(peerID.String(), &im)
			p2p.markPeerDisconnected(peerID.String())
		}

		err = p2p.startGRPCServer()
		if err != nil {
			log.Error("failed to prepare grpc server: ", err)
			return
		}
	}()
}

// NewManager creates and returns a new p2p manager
func NewManager(key *pcrypto.Key, appManager AppManager, externalDB ExternalDB, p2pPort int) (*P2P, error) {
	p2p := &P2P{
		appManager: appManager,
		grpcServer: newP2PGRPCServer(),
		externalDB: externalDB,
		p2pPort:    p2pPort,

		restartServerSignal: make(chan initMachine),

		clients:      util.NewMap[string, *Client](),
		machines:     util.NewMap[string, Machine](),
		peerStates:   util.NewMap[string, PeerState](),
		pendingPeers: util.NewMap[string, bool](),

		relayReservations: util.NewMap[string, time.Time](),

		reconcileCh: make(chan struct{}, 1),
		stopCh:      make(chan struct{}),
	}

	prvKey, err := crypto.UnmarshalEd25519PrivateKey(key.Private())
	if err != nil {
		return nil, err
	}

	con, err := connmgr.NewConnManager(100, 400)
	if err != nil {
		return nil, err
	}

	host, err := libp2p.New(
		libp2p.Identity(prvKey),
		libp2p.ListenAddrStrings(listenAddrsForPort(p2pPort)...),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(quic.NewTransport),
		libp2p.ConnectionManager(con),
		libp2p.EnableRelay(),
		libp2p.EnableRelayService(),
		libp2p.ForceReachabilityPublic(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to setup p2p host: %w", err)
	}

	p2p.host = host
	p2p.host.SetStreamHandler(swarmionBootstrapProtocol, p2p.handleSwarmionBootstrapStream)
	p2p.host.SetStreamHandler(imageBlobStreamProtocol, p2p.handleImageBlobStream)
	p2p.host.SetStreamHandler(imageArchiveUploadProtocol, p2p.handleImageArchiveUploadStream)
	nb := network.NotifyBundle{
		ConnectedF:    p2p.newConnectionHandler,
		DisconnectedF: p2p.closeConnectionHandler,
	}
	p2p.host.Network().Notify(&nb)

	log.Infof("using host with ID '%s'", host.ID().String())
	if !p2p.externalDB.Initialized() {
		p2p.restartServerAfterInit()
	}

	return p2p, nil
}
