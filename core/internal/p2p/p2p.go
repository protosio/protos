package p2p

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	connmgr "github.com/libp2p/go-libp2p/p2p/net/connmgr"
	relayclient "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	noise "github.com/libp2p/go-libp2p/p2p/security/noise"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	tcp "github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/multiformats/go-multiaddr"
	swarmiontransport "github.com/nustiueudinastea/swarmion/transports"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/imageregistry"
	networkmodule "github.com/protosio/protos/internal/network/module"
	p2pproto "github.com/protosio/protos/internal/p2p/proto"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/swarmionlink"
	"github.com/protosio/protos/internal/tasks"
	"github.com/protosio/protos/internal/util"
	"google.golang.org/grpc"
)

var log = util.GetLogger("p2p")

const (
	protosRPCProtocol               = "/protos/rpc/0.0.1"
	imageBlobStreamProtocol         = "/protos/image-blob/0.0.1"
	imageArchiveUploadProtocol      = "/protos/image-archive-upload/0.0.1"
	peerCapabilityImageContent      = "image.content"
	peerCapabilitySwarmionTransport = "swarmion.transport"
	peerCapabilityRelayService      = "libp2p.relay-service"
	destinationQUICIPv4Template     = "/ip4/%s/udp/%d/quic-v1/p2p/%s"
	destinationQUICIPv6Template     = "/ip6/%s/udp/%d/quic-v1/p2p/%s"
	destinationTCPIPv4Template      = "/ip4/%s/tcp/%d/p2p/%s"
	destinationTCPIPv6Template      = "/ip6/%s/tcp/%d/p2p/%s"
	peerRetryInterval               = 10 * time.Second
	peerRetryMaxBackoff             = time.Minute
	peerConnectDialTimeout          = 10 * time.Second
	grpcGracefulStopTimeout         = 5 * time.Second
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
	host                 host.Host
	registry             *swarmionlink.Registry
	routeFence           *swarmionlink.RouteFence
	ownsHost             bool
	appManager           AppManager
	imageManager         ImageManager
	taskManager          *tasks.Manager
	grpcServer           *grpc.Server
	grpcListener         net.Listener
	dbMu                 sync.RWMutex
	externalDB           ExternalDB
	network              NetworkInspector
	p2pPort              int
	controlClientTimeout time.Duration
	notify               network.Notifiee
	protocolRegistration swarmiontransport.Registration
	// Focused fence tests replace these physical-network observations to prove
	// that a close failure or a surviving sibling connection fails closed.
	closePeerForDrain       func(peer.ID) error
	peerConnectionsForDrain func(peer.ID) int
	// Focused TOFU tests pin the random placeholder so they can prove its
	// dial-only identity-probe scope is revoked on every return path.
	newIdentityProbePeerID      func() (peer.ID, error)
	afterIdentityLearnedForTest func(peer.ID)

	initMu     sync.RWMutex
	initPeerID peer.ID
	grpcMu     sync.Mutex
	clientMu   sync.Mutex
	routeMu    sync.Mutex
	peerRoutes map[string]map[string]struct{}

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

	reconcileCh    chan struct{}
	stopCh         chan struct{}
	stopOnce       sync.Once
	serverStopOnce sync.Once
	scopeStopOnce  sync.Once
	scopeStopErr   error
}

func (p2p *P2P) SetAppManager(appManager AppManager) {
	if p2p == nil {
		return
	}
	p2p.appManager = appManager
}

// SetExternalDB completes construction after the application-owned transport
// has been registered. Node startup deliberately registers the Protos protocol
// scope before opening Swarmion so both consumers share one physical host and
// Swarmion can detect protocol collisions instead of being overwritten later.
func (p2p *P2P) SetExternalDB(externalDB ExternalDB) error {
	if p2p == nil {
		return fmt.Errorf("p2p manager is nil")
	}
	if externalDB == nil {
		return fmt.Errorf("external database is nil")
	}
	p2p.dbMu.Lock()
	if p2p.externalDB != nil {
		p2p.dbMu.Unlock()
		return fmt.Errorf("external database is already configured")
	}
	p2p.externalDB = externalDB
	p2p.dbMu.Unlock()
	return nil
}

func (p2p *P2P) externalDatabase() ExternalDB {
	if p2p == nil {
		return nil
	}
	p2p.dbMu.RLock()
	defer p2p.dbMu.RUnlock()
	return p2p.externalDB
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

// PhysicalConnectedPeerIDs reports only peers connected to the
// application-owned libp2p host. It deliberately does not infer connectivity
// from Swarmion's routed, participating, or logical database views.
func (p2p *P2P) PhysicalConnectedPeerIDs() []string {
	if p2p == nil || p2p.host == nil || p2p.host.Network() == nil {
		return nil
	}
	peers := p2p.host.Network().Peers()
	out := make([]string, 0, len(peers))
	for _, peerID := range peers {
		if peerID == "" || peerID == p2p.host.ID() {
			continue
		}
		out = append(out, peerID.String())
	}
	sort.Strings(out)
	return out
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

// ListenMultiaddrs returns the application host's current endpoints with its
// authenticated peer identity. These addresses are also Swarmion bootstrap
// addresses because both protocols borrow the same physical host.
func (p2p *P2P) ListenMultiaddrs() []string {
	if p2p == nil || p2p.host == nil {
		return nil
	}
	peerSuffix, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", p2p.host.ID()))
	if err != nil {
		return nil
	}
	addrs := make([]string, 0, len(p2p.host.Addrs()))
	for _, addr := range p2p.host.Addrs() {
		addrs = append(addrs, addr.Encapsulate(peerSuffix).String())
	}
	return dedupeStrings(addrs)
}

// DialableListenMultiaddrs supplements the host's advertised endpoints with
// explicit addresses known by the product (for example a VM's provider IP or
// the macOS VM gateway). It always uses the one shared Protos/Swarmion port.
func (p2p *P2P) DialableListenMultiaddrs(ips []string) []string {
	if p2p == nil || p2p.host == nil {
		return nil
	}
	port := p2p.listenPort()
	peerID := p2p.host.ID().String()
	addrs := make([]string, 0, len(ips)*2+len(p2p.host.Addrs()))
	for _, rawIP := range ips {
		ip := net.ParseIP(strings.TrimSpace(rawIP))
		if ip == nil || port <= 0 {
			continue
		}
		if ip.To4() == nil {
			addrs = append(addrs,
				fmt.Sprintf(destinationTCPIPv6Template, ip.String(), port, peerID),
				fmt.Sprintf(destinationQUICIPv6Template, ip.String(), port, peerID),
			)
		} else {
			addrs = append(addrs,
				fmt.Sprintf(destinationTCPIPv4Template, ip.String(), port, peerID),
				fmt.Sprintf(destinationQUICIPv4Template, ip.String(), port, peerID),
			)
		}
	}
	addrs = append(addrs, p2p.ListenMultiaddrs()...)
	return dedupeStrings(addrs)
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
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
	activePeers := make([]peer.ID, 0, len(machines))

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
			if errors.Is(err, swarmionlink.ErrPeerFenced) {
				// ConfigurePeers may be applying a replicated snapshot captured
				// before a delete acquired its route generation. Ordinary
				// admission is monotonic and cannot reopen that generation.
				log.Debugf("ignoring stale configured peer while its deletion fence is active: %v", err)
				continue
			}
			return err
		}
		currentMachines[peerID] = machine
		decodedPeerID, decodeErr := peer.Decode(peerID)
		if decodeErr != nil {
			return fmt.Errorf("decode configured peer ID %s: %w", peerID, decodeErr)
		}
		activePeers = append(activePeers, decodedPeerID)
		p2p.pendingPeers.Delete(peerID)
	}

	if p2p.routeFence != nil {
		pendingPeers := p2p.pendingPeers.Snapshot()
		temporaryPeers := make([]peer.ID, 0, len(pendingPeers))
		for peerIDString := range pendingPeers {
			peerID, err := peer.Decode(peerIDString)
			if err == nil && peerID != "" {
				temporaryPeers = append(temporaryPeers, peerID)
			}
		}
		p2p.routeFence.ReconcileAdmittedPeers(activePeers, temporaryPeers)
		for _, peerID := range p2p.host.Network().Peers() {
			if !p2p.routeFence.IsPeerConnectionAllowed(peerID) {
				if err := p2p.host.Network().ClosePeer(peerID); err != nil {
					log.Debugf("failed to close fenced peer %s: %v", peerID, err)
				}
			}
		}
	}

	for peerID := range p2p.machines.Snapshot() {
		if _, found := currentMachines[peerID]; !found {
			if pending, _ := p2p.pendingPeers.Get(peerID); pending {
				continue
			}
			p2p.machines.Delete(peerID)
			p2p.peerStates.Delete(peerID)
			p2p.relayReservations.Delete(peerID)
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
			p2p.clientMu.Lock()
			if client, found := p2p.clients.Get(peerIDString); found && client != nil && client.grpcConnection != nil {
				_ = client.grpcConnection.Close()
			}
			p2p.clients.Delete(peerIDString)
			p2p.clientMu.Unlock()
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

func (p2p *P2P) trackPeer(machine Machine) (string, error) {
	if machine == nil {
		return "", fmt.Errorf("machine is nil")
	}
	peerID, err := p2p.pubKeyToPeerID(machine.GetPublicKey())
	if err != nil {
		return "", fmt.Errorf("failed to get peer ID from public key: %w", err)
	}
	peerIDString := peerID.String()
	if p2p.routeFence != nil {
		if err := p2p.routeFence.AdmitPeer(peerID); err != nil {
			return "", fmt.Errorf("admit peer %s: %w", peerIDString, err)
		}
	}
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
	// Endpoint hints are additive on the application-owned host. Clearing the
	// shared peerstore or dial backoff here would erase Identify, relay, and
	// Swarmion observations belonging to other protocol scopes.
	p2p.addPeerEndpointHints(peerInfo)

	log.Debugf("dialing peer id %s using %d destination(s): %s", peerIDString, len(destinations), strings.Join(destinations, ", "))

	connectCtx, cancel := context.WithTimeout(ctx, peerConnectDialTimeout)
	defer cancel()

	if err := p2p.host.Connect(connectCtx, peerInfo); err != nil {
		return nil, err
	}

	if client, found := p2p.connectedClient(peerIDString); found {
		p2p.markPeerConnected(peerIDString, machine)
		return client, nil
	}

	clientCtx, cancelClient := context.WithTimeout(ctx, p2p.controlClientReadinessTimeout())
	defer cancelClient()
	client, err := p2p.ensureControlClient(clientCtx, peerInfo.ID)
	if err != nil {
		return nil, err
	}
	p2p.markPeerConnected(peerIDString, machine)
	return client, nil
}

func (p2p *P2P) addPeerEndpointHints(peerInfo peer.AddrInfo) {
	if p2p == nil || p2p.host == nil || peerInfo.ID == "" || len(peerInfo.Addrs) == 0 {
		return
	}
	p2p.host.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, peerstore.TempAddrTTL)
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
		if relayID == peerIDString || !isPublicRelayCandidate(relayMachine) {
			continue
		}
		relayClient, connected := p2p.clients.Get(relayID)
		if !connected || !relayClient.supportsCapability(peerCapabilityRelayService) {
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
		if _, found := p2p.connectedClient(peerIDString); found {
			p2p.markPeerConnected(peerIDString, machine)
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
	if !isPublicRelayCandidate(machine) || p2p.hasPublicHostAddress() {
		return
	}
	client, connected := p2p.clients.Get(peerIDString)
	if !connected || !client.supportsCapability(peerCapabilityRelayService) {
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

func (p2p *P2P) hasPublicHostAddress() bool {
	if p2p == nil || p2p.host == nil {
		return false
	}
	for _, addr := range p2p.host.Addrs() {
		for _, protocolCode := range []int{multiaddr.P_IP4, multiaddr.P_IP6} {
			value, err := addr.ValueForProtocol(protocolCode)
			if err != nil {
				continue
			}
			ip := net.ParseIP(value)
			if ip != nil && ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() {
				return true
			}
		}
	}
	return false
}

func isPublicRelayCandidate(machine Machine) bool {
	if machine == nil {
		return false
	}
	ip := net.ParseIP(strings.TrimSpace(machine.GetPublicIP()))
	return ip != nil && ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback()
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

// RefreshPeerControlAfterRemoteRestart drops only the short-lived Protos gRPC
// client after Init. The shared physical connection stays alive for Swarmion
// and any other application protocol while control reconnects on demand.
func (p2p *P2P) RefreshPeerControlAfterRemoteRestart(machine Machine) error {
	if p2p == nil || machine == nil {
		return nil
	}
	peerID, err := p2p.pubKeyToPeerID(machine.GetPublicKey())
	if err != nil {
		return fmt.Errorf("failed to convert public key to peer ID: %w", err)
	}
	peerIDString := peerID.String()
	p2p.clientMu.Lock()
	if client, found := p2p.clients.Get(peerIDString); found && client != nil && client.grpcConnection != nil {
		if err := client.grpcConnection.Close(); err != nil {
			log.Debugf("failed to close control client for restarting peer %s: %v", peerIDString, err)
		}
	}
	p2p.clients.Delete(peerIDString)
	p2p.clientMu.Unlock()
	p2p.updatePeerState(peerIDString, func(state *PeerState) {
		state.Status = PeerStatusDesired
		state.LastAttempt = time.Time{}
		state.LastError = ""
	})
	p2p.requestReconcile()
	return nil
}

func (p2p *P2P) RemovePeer(machine Machine) error {
	_, _, err := p2p.FencePeer(machine)
	return err
}

// FencePeer withdraws every Protos and Swarmion route before closing physical
// connections. The returned opaque generation is valid only for this process
// and this exact fence attempt and must accompany the Swarmion drain handshake.
func (p2p *P2P) FencePeer(machine Machine) (string, string, error) {
	if p2p == nil || p2p.routeFence == nil {
		return "", "", fmt.Errorf("p2p route fence is not configured")
	}
	if machine == nil {
		return "", "", fmt.Errorf("machine is nil")
	}
	log.Debugf("removing peer '%s'", machine.GetID())
	peerID, err := p2p.pubKeyToPeerID(machine.GetPublicKey())
	if err != nil {
		return "", "", fmt.Errorf("failed to convert public key to peer ID: %w", err)
	}
	peerIDString := peerID.String()
	generation, err := p2p.routeFence.FencePeer(peerID)
	if err != nil {
		return "", "", fmt.Errorf("fence peer %s: %w", peerIDString, err)
	}

	if p2p.host == nil || p2p.registry == nil {
		return "", "", fmt.Errorf("peer %s is fenced at generation %s but its shared host/registry is unavailable", peerIDString, generation)
	}
	if !p2p.routeFence.LinkRouteWithdrawn(peerID) {
		return "", "", fmt.Errorf("peer %s is fenced at generation %s but remains exposed by the Swarmion Link", peerIDString, generation)
	}

	p2p.clientMu.Lock()
	if client, found := p2p.clients.Get(peerIDString); found && client != nil && client.grpcConnection != nil {
		_ = client.grpcConnection.Close()
	}
	p2p.clientMu.Unlock()
	closePeer := p2p.closePeerForDrain
	if closePeer == nil {
		closePeer = p2p.host.Network().ClosePeer
	}
	if err := closePeer(peerID); err != nil {
		return "", "", fmt.Errorf("peer %s remains fenced at generation %s after physical close failed: %w", peerIDString, generation, err)
	}
	connectionCount := p2p.peerConnectionsForDrain
	if connectionCount == nil {
		connectionCount = func(peerID peer.ID) int { return len(p2p.host.Network().ConnsToPeer(peerID)) }
	}
	if count := connectionCount(peerID); count != 0 {
		return "", "", fmt.Errorf("peer %s remains fenced at generation %s with %d physical connection(s)", peerIDString, generation, count)
	}
	streamDrainCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := p2p.registry.DrainPeerStreams(streamDrainCtx, peerID); err != nil {
		return "", "", fmt.Errorf("peer %s remains fenced at generation %s with active protocol streams: %w", peerIDString, generation, err)
	}

	p2p.machines.Delete(peerIDString)
	p2p.clientMu.Lock()
	p2p.clients.Delete(peerIDString)
	p2p.clientMu.Unlock()
	p2p.peerStates.Delete(peerIDString)
	p2p.pendingPeers.Delete(peerIDString)
	p2p.relayReservations.Delete(peerIDString)
	p2p.requestReconcile()

	return peerIDString, generation, nil
}

// WithPeerFenceGeneration prevents route admission or replacement from racing
// the final generation-matched Swarmion drain call.
func (p2p *P2P) WithPeerFenceGeneration(
	ctx context.Context,
	peerIDString string,
	generation string,
	fn func() error,
) error {
	if p2p == nil || p2p.routeFence == nil {
		return fmt.Errorf("p2p route fence is not configured")
	}
	peerID, err := peer.Decode(strings.TrimSpace(peerIDString))
	if err != nil {
		return fmt.Errorf("decode peer ID %q: %w", peerIDString, err)
	}
	return p2p.routeFence.WithGeneration(ctx, peerID, generation, fn)
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
	externalDB := p2p.externalDatabase()
	if externalDB == nil {
		return fmt.Errorf("external database is not configured")
	}
	if p2p.grpcServer == nil {
		p2p.grpcServer = newP2PGRPCServer()
	}

	server := p2p.grpcServer

	// register internal grpc servers
	srv := &Server{DB: externalDB, p2p: p2p}
	p2pproto.RegisterPingerServer(server, srv)
	p2pproto.RegisterPeerDBServer(server, srv)
	p2pproto.RegisterAppsServer(server, srv)
	p2pproto.RegisterImagesServer(server, srv)
	p2pproto.RegisterInstanceServer(server, srv)

	// Serve gRPC through the permanent, registry-owned application listener
	// installed before Swarmion opens on the shared host.
	grpcListener := p2p.grpcListener
	if grpcListener == nil {
		return fmt.Errorf("Protos control listener is not configured")
	}
	go func() {
		if err := server.Serve(grpcListener); err != nil {
			select {
			case <-p2p.stopCh:
				log.Debugf("p2p grpc server stopped: %v", err)
			default:
				log.Errorf("p2p grpc server stopped unexpectedly: %v", err)
			}
		}
	}()

	return nil
}

// StartServer starts listening for p2p connections
func (p2p *P2P) StartServer() (func() error, error) {
	log.Info("starting p2p server")

	err := p2p.startGRPCServer()
	if err != nil {
		return func() error { return nil }, fmt.Errorf("failed to prepare grpc server: %w", err)
	}

	stopReconciler := p2p.startPeerReconciler()
	stopper := func() error {
		_ = stopReconciler()
		return p2p.StopServer()
	}
	return stopper, nil

}

// StopServer closes Protos control admission and its reconciler while keeping
// the application protocol registration and shared physical host alive. Node
// shutdown invokes this before closing the DB so no API/control writer can race
// Swarmion's drain boundary.
func (p2p *P2P) StopServer() error {
	if p2p == nil {
		return nil
	}
	p2p.serverStopOnce.Do(func() {
		p2p.stopOnce.Do(func() { close(p2p.stopCh) })
		p2p.grpcMu.Lock()
		server := p2p.grpcServer
		listener := p2p.grpcListener
		p2p.grpcListener = nil
		p2p.grpcMu.Unlock()
		if listener != nil {
			_ = listener.Close()
		}
		if server != nil {
			if forced := boundedGRPCStop(server, grpcGracefulStopTimeout); forced {
				log.Warn("forced p2p gRPC shutdown after graceful-stop deadline")
			}
		}
	})
	return nil
}

func boundedGRPCStop(server *grpc.Server, timeout time.Duration) bool {
	if server == nil {
		return false
	}
	if timeout <= 0 {
		server.Stop()
		return true
	}
	done := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(done)
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
		return false
	case <-timer.C:
		server.Stop()
		<-done
		return true
	}
}

// Close removes only this manager's protocol registrations, observers, and
// workers when the host is borrowed. A manager returned by NewManager owns its
// reference host and closes it after the scoped shutdown.
func (p2p *P2P) Close() error {
	if p2p == nil {
		return nil
	}
	p2p.scopeStopOnce.Do(func() {
		log.Debug("stopping p2p server")
		_ = p2p.StopServer()

		p2p.clientMu.Lock()
		for _, client := range p2p.clients.Snapshot() {
			if client != nil && client.grpcConnection != nil {
				_ = client.grpcConnection.Close()
			}
		}
		p2p.clientMu.Unlock()
		if p2p.host != nil {
			if p2p.notify != nil {
				p2p.host.Network().StopNotify(p2p.notify)
			}
			if p2p.protocolRegistration != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := p2p.protocolRegistration.Close(ctx); err != nil {
					p2p.scopeStopErr = fmt.Errorf("close Protos protocol scope: %w", err)
				}
				cancel()
			}
			if p2p.ownsHost {
				if err := p2p.host.Close(); err != nil && p2p.scopeStopErr == nil {
					p2p.scopeStopErr = err
				}
			}
		}
	})
	return p2p.scopeStopErr
}

// NewHost creates the single physical libp2p host owned by Protos. Swarmion
// receives a borrowed adapter around this host; its lifecycle never creates or
// closes another physical transport.
func NewHost(key *pcrypto.Key, p2pPort int) (host.Host, error) {
	if key == nil {
		return nil, fmt.Errorf("p2p key is nil")
	}
	prvKey, err := crypto.UnmarshalEd25519PrivateKey(key.Private())
	if err != nil {
		return nil, err
	}

	con, err := connmgr.NewConnManager(100, 400)
	if err != nil {
		return nil, err
	}
	routeFence, err := swarmionlink.NewRouteFence()
	if err != nil {
		return nil, err
	}

	peerHost, err := libp2p.New(
		libp2p.Identity(prvKey),
		libp2p.ListenAddrStrings(listenAddrsForPort(p2pPort)...),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(quic.NewTransport),
		libp2p.ConnectionManager(con),
		libp2p.ConnectionGater(routeFence),
		libp2p.EnableRelay(),
		libp2p.EnableRelayService(),
		libp2p.NATPortMap(),
		libp2p.EnableHolePunching(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to setup p2p host: %w", err)
	}
	return &routeFencedHost{Host: peerHost, routeFence: routeFence}, nil
}

type routeFencedHost struct {
	host.Host
	routeFence *swarmionlink.RouteFence
}

func (h *routeFencedHost) SwarmionRouteFence() *swarmionlink.RouteFence {
	if h == nil {
		return nil
	}
	return h.routeFence
}

// NewManagerWithHost registers the Protos protocol scope on a borrowed,
// application-owned host. Call this before opening Swarmion so protocol
// collisions are detected rather than silently overwritten.
func NewManagerWithHost(peerHost host.Host, appManager AppManager, externalDB ExternalDB, p2pPort int) (*P2P, error) {
	registry, err := swarmionlink.NewRegistry(peerHost)
	if err != nil {
		return nil, err
	}
	return NewManagerWithRegistry(peerHost, registry, appManager, externalDB, p2pPort)
}

// NewManagerWithRegistry installs Protos through the same collision-aware
// registry whose Link is passed to Swarmion.
func NewManagerWithRegistry(peerHost host.Host, registry *swarmionlink.Registry, appManager AppManager, externalDB ExternalDB, p2pPort int) (*P2P, error) {
	if peerHost == nil {
		return nil, fmt.Errorf("p2p host is nil")
	}
	if registry == nil {
		return nil, fmt.Errorf("shared protocol registry is nil")
	}
	p2p := &P2P{
		host:       peerHost,
		registry:   registry,
		routeFence: registry.RouteFence(),
		appManager: appManager,
		grpcServer: newP2PGRPCServer(),
		externalDB: externalDB,
		p2pPort:    p2pPort,

		clients:      util.NewMap[string, *Client](),
		machines:     util.NewMap[string, Machine](),
		peerStates:   util.NewMap[string, PeerState](),
		pendingPeers: util.NewMap[string, bool](),

		relayReservations: util.NewMap[string, time.Time](),
		peerRoutes:        make(map[string]map[string]struct{}),

		reconcileCh: make(chan struct{}, 1),
		stopCh:      make(chan struct{}),
	}

	registration, err := p2p.registerProtocolScope(registry)
	if err != nil {
		return nil, err
	}
	p2p.protocolRegistration = registration
	nb := &network.NotifyBundle{
		ConnectedF:    p2p.newConnectionHandler,
		DisconnectedF: p2p.closeConnectionHandler,
	}
	p2p.notify = nb
	p2p.host.Network().Notify(nb)
	p2p.initializePhysicalRoutes()

	log.Infof("using shared application host with ID '%s'", peerHost.ID().String())
	return p2p, nil
}

// NewManager retains the standalone owned-host constructor for tests and
// applications that deliberately delegate the entire physical host lifecycle.
func NewManager(key *pcrypto.Key, appManager AppManager, externalDB ExternalDB, p2pPort int) (*P2P, error) {
	peerHost, err := NewHost(key, p2pPort)
	if err != nil {
		return nil, err
	}
	manager, err := NewManagerWithHost(peerHost, appManager, externalDB, p2pPort)
	if err != nil {
		_ = peerHost.Close()
		return nil, err
	}
	manager.ownsHost = true
	return manager, nil
}
