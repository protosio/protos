package p2p

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	swarmionapp "github.com/nustiueudinastea/swarmion/runtime/app"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	p2pproto "github.com/protosio/protos/internal/p2p/proto"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/util"
	"google.golang.org/grpc"
)

type testMachine struct {
	id         string
	name       string
	publicKey  string
	publicIP   string
	internalIP string
}

func (m testMachine) GetID() string        { return m.id }
func (m testMachine) GetPublicKey() string { return m.publicKey }
func (m testMachine) GetPublicIP() string  { return m.publicIP }
func (m testMachine) GetName() string {
	if m.name != "" {
		return m.name
	}
	return m.id
}
func (m testMachine) GetInternalIP() string {
	return m.internalIP
}

func newTestMachine(t *testing.T, publicIP string) (testMachine, string, string) {
	t.Helper()

	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	libp2pPub, err := crypto.UnmarshalEd25519PublicKey(pub)
	if err != nil {
		t.Fatalf("unmarshal libp2p key: %v", err)
	}
	peerID, err := peer.IDFromPublicKey(libp2pPub)
	if err != nil {
		t.Fatalf("peer id: %v", err)
	}

	key := pcrypto.Key{Pub: pub}
	internalIP := key.IPv6Address().String()
	machine := testMachine{
		id:        "test-peer",
		publicKey: base64.StdEncoding.EncodeToString(pub),
		publicIP:  publicIP,
	}
	return machine, peerID.String(), internalIP
}

func newTestP2P() *P2P {
	return &P2P{
		machines:   util.NewMap[string, Machine](),
		clients:    util.NewMap[string, *Client](),
		peerStates: util.NewMap[string, PeerState](),
	}
}

func TestGetPeersKeysByPeerID(t *testing.T) {
	p2p := newTestP2P()
	p2p.machines.Set("peer-id-1", testMachine{id: "machine-id-1", name: "display-name-1"})
	p2p.peerStates.Set("peer-id-1", PeerState{Status: PeerStatusConnected})

	server := &Server{p2p: p2p}
	resp, err := server.GetPeers(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetPeers() error = %v", err)
	}

	if got := resp.GetPeers()["peer-id-1"]; got != string(PeerStatusConnected) {
		t.Fatalf("GetPeers()[peer-id-1] = %q, want %q", got, PeerStatusConnected)
	}
	if _, found := resp.GetPeers()["display-name-1"]; found {
		t.Fatalf("GetPeers() used machine name as key: %v", resp.GetPeers())
	}
}

func TestConfigurePeersIgnoresLocalPeer(t *testing.T) {
	localKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("local key: %v", err)
	}
	p2p, err := NewManager(localKey, nil, testExternalDB{initialized: true}, 0)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	defer p2p.host.Close()

	localMachine := testMachine{
		id:        "local",
		name:      "local-device",
		publicKey: base64.StdEncoding.EncodeToString(localKey.Public()),
		publicIP:  "127.0.0.1",
	}
	remoteMachine, remotePeerID, _ := newTestMachine(t, "203.0.113.10")

	if err := p2p.ConfigurePeers([]Machine{localMachine, remoteMachine}); err != nil {
		t.Fatalf("configure peers: %v", err)
	}

	if _, found := p2p.machines.Get(p2p.PeerID()); found {
		t.Fatalf("local peer %s was tracked as a remote machine", p2p.PeerID())
	}
	if _, found := p2p.peerStates.Get(p2p.PeerID()); found {
		t.Fatalf("local peer %s has remote peer state", p2p.PeerID())
	}
	if _, found := p2p.machines.Get(remotePeerID); !found {
		t.Fatalf("remote peer %s was not tracked", remotePeerID)
	}
}

func TestDisconnectedKnownPeerCountIgnoresExtraPendingClients(t *testing.T) {
	known := map[string]Machine{
		"peer-a": testMachine{id: "peer-a"},
		"peer-b": testMachine{id: "peer-b"},
	}
	connected := map[string]*Client{
		"peer-a":       {},
		"pending-peer": {},
	}
	if got := disconnectedKnownPeerCount(known, connected); got != 1 {
		t.Fatalf("disconnectedKnownPeerCount() = %d, want 1", got)
	}
}

func TestUsablePeerConnectednessAcceptsLimitedConnections(t *testing.T) {
	for _, connectedness := range []network.Connectedness{network.Connected, network.Limited} {
		if !usablePeerConnectedness(connectedness) {
			t.Fatalf("usablePeerConnectedness(%s) = false, want true", connectedness)
		}
	}
	if usablePeerConnectedness(network.NotConnected) {
		t.Fatalf("usablePeerConnectedness(%s) = true, want false", network.NotConnected)
	}
}

func TestDestinationStringsIncludeWireGuardIPv6WithoutPublicIP(t *testing.T) {
	machine, peerID, internalIP := newTestMachine(t, "")
	p2p := newTestP2P()

	got := p2p.destinationStrings(peerID, machine)
	want := fmt.Sprintf(destinationTCPIPv6Template, internalIP, config.Get().P2PPort, peerID)
	if !slices.Contains(got, want) {
		t.Fatalf("destinationStrings() = %v, want %s", got, want)
	}
}

func TestDestinationStringsIncludeMultipleKnownIPs(t *testing.T) {
	machine, peerID, derivedInternalIP := newTestMachine(t, "203.0.113.10")
	machine.internalIP = "fd00::10"
	p2p := newTestP2P()

	got := p2p.destinationStrings(peerID, machine)
	wants := []string{
		fmt.Sprintf(destinationTCPIPv6Template, machine.internalIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationQUICIPv6Template, machine.internalIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationTCPIPv6Template, derivedInternalIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationQUICIPv6Template, derivedInternalIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationTCPIPv4Template, machine.publicIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationQUICIPv4Template, machine.publicIP, config.Get().P2PPort, peerID),
	}
	for _, want := range wants {
		if !slices.Contains(got, want) {
			t.Fatalf("destinationStrings() = %v, want %s", got, want)
		}
	}
}

func TestDestinationStringsPreferWireGuardIPv6BeforePublicIP(t *testing.T) {
	machine, peerID, derivedInternalIP := newTestMachine(t, "203.0.113.10")
	machine.internalIP = "fd00::10"
	p2p := newTestP2P()

	got := p2p.destinationStrings(peerID, machine)
	want := []string{
		fmt.Sprintf(destinationTCPIPv6Template, machine.internalIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationQUICIPv6Template, machine.internalIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationTCPIPv6Template, derivedInternalIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationQUICIPv6Template, derivedInternalIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationTCPIPv4Template, machine.publicIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationQUICIPv4Template, machine.publicIP, config.Get().P2PPort, peerID),
	}
	if len(got) < len(want) {
		t.Fatalf("destinationStrings() = %v, want at least %d destinations", got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("destinationStrings()[%d] = %s, want %s; all destinations: %v", i, got[i], want[i], got)
		}
	}
}

func TestDestinationStringsDoNotPromotePrivatePublicIPBeforeWireGuardIPv6(t *testing.T) {
	machine, peerID, derivedInternalIP := newTestMachine(t, "192.168.64.10")
	p2p := newTestP2P()

	got := p2p.destinationStrings(peerID, machine)
	want := []string{
		fmt.Sprintf(destinationTCPIPv6Template, derivedInternalIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationQUICIPv6Template, derivedInternalIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationTCPIPv4Template, machine.publicIP, config.Get().P2PPort, peerID),
		fmt.Sprintf(destinationQUICIPv4Template, machine.publicIP, config.Get().P2PPort, peerID),
	}
	if len(got) < len(want) {
		t.Fatalf("destinationStrings() = %v, want at least %d destinations", got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("destinationStrings()[%d] = %s, want %s; all destinations: %v", i, got[i], want[i], got)
		}
	}
}

func TestPeerAddrInfoFromDestinationsCombinesPeerAddresses(t *testing.T) {
	_, peerID, _ := newTestMachine(t, "")
	destinations := []string{
		fmt.Sprintf(destinationTCPIPv4Template, "192.0.2.10", 10500, peerID),
		fmt.Sprintf(destinationTCPIPv4Template, "192.0.2.10", 10500, peerID),
		fmt.Sprintf(destinationQUICIPv4Template, "192.0.2.11", 10500, peerID),
	}

	info, err := peerAddrInfoFromDestinations(peerID, destinations)
	if err != nil {
		t.Fatalf("peerAddrInfoFromDestinations() error = %v", err)
	}

	if info.ID.String() != peerID {
		t.Fatalf("AddrInfo ID = %s, want %s", info.ID, peerID)
	}
	var addrs []string
	for _, addr := range info.Addrs {
		addrs = append(addrs, addr.String())
	}
	want := []string{
		"/ip4/192.0.2.10/tcp/10500",
		"/ip4/192.0.2.11/udp/10500/quic-v1",
	}
	if !slices.Equal(addrs, want) {
		t.Fatalf("AddrInfo addrs = %v, want %v", addrs, want)
	}
}

func TestPeerAddrInfoFromDestinationsRejectsWrongPeerID(t *testing.T) {
	_, peerID, _ := newTestMachine(t, "")
	_, otherPeerID, _ := newTestMachine(t, "")
	destinations := []string{
		fmt.Sprintf(destinationTCPIPv4Template, "192.0.2.10", 10500, otherPeerID),
	}

	if _, err := peerAddrInfoFromDestinations(peerID, destinations); err == nil {
		t.Fatal("peerAddrInfoFromDestinations() returned nil error for wrong destination peer")
	}
}

func TestDiscoverPeerAtLearnsRemoteIdentityWithFakePeerID(t *testing.T) {
	localKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("local key: %v", err)
	}
	remoteKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("remote key: %v", err)
	}
	local, err := NewManager(localKey, nil, testExternalDB{initialized: true}, 0)
	if err != nil {
		t.Fatalf("local manager: %v", err)
	}
	defer local.host.Close()
	remote, err := NewManager(remoteKey, nil, testExternalDB{}, 0)
	if err != nil {
		t.Fatalf("remote manager: %v", err)
	}
	defer remote.host.Close()

	localStop, err := local.StartServer()
	if err != nil {
		t.Fatalf("start local server: %v", err)
	}
	defer func() {
		if err := localStop(); err != nil {
			t.Errorf("stop local server: %v", err)
		}
	}()
	remoteStop, err := remote.StartServer()
	if err != nil {
		t.Fatalf("start remote server: %v", err)
	}
	defer func() {
		if err := remoteStop(); err != nil {
			t.Errorf("stop remote server: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	discovered, err := local.DiscoverPeerAt(ctx, loopbackQUICAddr(t, remote))
	if err != nil {
		t.Fatalf("discover peer: %v", err)
	}
	if discovered.ID != remoteKey.GetID() {
		t.Fatalf("discovered peer ID = %s, want %s", discovered.ID, remoteKey.GetID())
	}
	if discovered.PublicKey != remoteKey.PublicKey() {
		t.Fatalf("discovered public key = %s, want %s", discovered.PublicKey, remoteKey.PublicKey())
	}
	if discovered.Fingerprint == "" {
		t.Fatal("expected non-empty fingerprint")
	}
}

func TestRemovePeerUnregistersExternalDBPeer(t *testing.T) {
	localKey, err := pcrypto.GetLocalKey(t.TempDir())
	if err != nil {
		t.Fatalf("local key: %v", err)
	}
	externalDB := &recordingExternalDB{testExternalDB: testExternalDB{initialized: true}}
	p2p, err := NewManager(localKey, nil, externalDB, 0)
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	defer p2p.host.Close()

	machine, peerID, _ := newTestMachine(t, "203.0.113.10")
	if trackedPeerID, err := p2p.trackPeer(machine); err != nil {
		t.Fatalf("track peer: %v", err)
	} else if trackedPeerID != peerID {
		t.Fatalf("trackPeer() = %s, want %s", trackedPeerID, peerID)
	}
	p2p.pendingPeers.Set(peerID, true)

	if err := p2p.RemovePeer(machine); err != nil {
		t.Fatalf("RemovePeer() error = %v", err)
	}

	if _, found := p2p.machines.Get(peerID); found {
		t.Fatalf("machine map still has removed peer %s", peerID)
	}
	if _, found := p2p.peerStates.Get(peerID); found {
		t.Fatalf("peer state map still has removed peer %s", peerID)
	}
	if pending, found := p2p.pendingPeers.Get(peerID); found || pending {
		t.Fatalf("pending peer map still has removed peer %s", peerID)
	}
	if !slices.Equal(externalDB.removed, []string{peerID}) {
		t.Fatalf("external DB removals = %#v, want [%s]", externalDB.removed, peerID)
	}
}

func TestTofuTransportAddrsPreferTCPBeforeQUIC(t *testing.T) {
	p2p := &P2P{p2pPort: 1234}

	got, err := p2p.tofuTransportAddrs("203.0.113.10")
	if err != nil {
		t.Fatalf("tofuTransportAddrs() error = %v", err)
	}
	want := []string{
		"/ip4/203.0.113.10/tcp/1234",
		"/ip4/203.0.113.10/udp/1234/quic-v1",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("tofuTransportAddrs() = %v, want %v", got, want)
	}
}

func TestGetRuntimeStateRequiresStrictCatchUp(t *testing.T) {
	wantErr := fmt.Errorf("strict catch-up failed")
	server := &Server{DB: runtimeStateExternalDB{strictErr: wantErr}}

	_, err := server.GetRuntimeState(context.Background(), &p2pproto.GetRuntimeStateRequest{})
	if err == nil {
		t.Fatal("GetRuntimeState returned nil error, want strict catch-up error")
	}
	if err.Error() != wantErr.Error() {
		t.Fatalf("GetRuntimeState error = %v, want %v", err, wantErr)
	}
}

func TestImageCapableClientsRequireAdvertisedCapability(t *testing.T) {
	withCapability := &Client{ImagesClient: fakeImagesClient{}}
	withCapability.setCapabilities([]string{peerCapabilityImageContent})
	withoutCapability := &Client{ImagesClient: fakeImagesClient{}}

	got := imageCapableClients(map[string]*Client{
		"with":    withCapability,
		"without": withoutCapability,
		"nil":     nil,
	})

	if _, found := got["with"]; !found {
		t.Fatal("client advertising image content capability was not selected")
	}
	if _, found := got["without"]; found {
		t.Fatal("client without image content capability was selected")
	}
	if _, found := got["nil"]; found {
		t.Fatal("nil client was selected")
	}
}

func TestPeerCapabilitiesAdvertiseSwarmionTransportWhenDBInitialized(t *testing.T) {
	uninitialized := (&Server{
		DB:  testExternalDB{initialized: false},
		p2p: &P2P{},
	}).peerCapabilities()
	if slices.Contains(uninitialized, peerCapabilitySwarmionTransport) {
		t.Fatalf("uninitialized DB capabilities = %v, should not include %s", uninitialized, peerCapabilitySwarmionTransport)
	}

	initialized := (&Server{
		DB:  testExternalDB{initialized: true},
		p2p: &P2P{},
	}).peerCapabilities()
	if !slices.Contains(initialized, peerCapabilitySwarmionTransport) {
		t.Fatalf("initialized DB capabilities = %v, want %s", initialized, peerCapabilitySwarmionTransport)
	}
}

func TestConnectExternalDBPeerRequiresSwarmionTransportCapability(t *testing.T) {
	machine, peerID, _ := newTestMachine(t, "203.0.113.10")
	externalDB := &recordingExternalDB{}
	manager := &P2P{externalDB: externalDB}

	manager.connectExternalDBPeer(peerID, machine, &Client{})
	if len(externalDB.connectedIPs) != 0 {
		t.Fatalf("DB peer connects without capability = %#v, want none", externalDB.connectedIPs)
	}

	client := &Client{}
	client.setCapabilities([]string{peerCapabilitySwarmionTransport})
	manager.connectExternalDBPeer(peerID, machine, client)
	if len(externalDB.connectedIPs) != 1 {
		t.Fatalf("DB peer connects with capability = %#v, want one call", externalDB.connectedIPs)
	}
	if externalDB.connectedIPs[0].peerID != peerID {
		t.Fatalf("connected peer ID = %s, want %s", externalDB.connectedIPs[0].peerID, peerID)
	}
}

func loopbackQUICAddr(t *testing.T, p2p *P2P) string {
	t.Helper()
	for _, addr := range p2p.host.Addrs() {
		port, err := addr.ValueForProtocol(ma.P_UDP)
		if err != nil {
			continue
		}
		return fmt.Sprintf("/ip4/127.0.0.1/udp/%s/quic-v1", port)
	}
	t.Fatalf("no QUIC listen address found: %v", p2p.host.Addrs())
	return ""
}

type testExternalDB struct {
	initialized bool
}

type recordingExternalDB struct {
	testExternalDB
	removed      []string
	connectedIPs []recordedDBPeerConnect
}

type runtimeStateExternalDB struct {
	testExternalDB
	strictErr error
}

type recordedDBPeerConnect struct {
	peerID string
	ips    []string
}

type fakeImagesClient struct{}

func (fakeImagesClient) DescribeImage(context.Context, *p2pproto.DescribeImageRequest, ...grpc.CallOption) (*p2pproto.DescribeImageResponse, error) {
	return nil, nil
}

func (fakeImagesClient) GetImageContent(context.Context, *p2pproto.GetImageContentRequest, ...grpc.CallOption) (*p2pproto.GetImageContentResponse, error) {
	return nil, nil
}

func (fakeImagesClient) GetImageBlob(context.Context, *p2pproto.GetImageBlobRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[p2pproto.GetImageBlobResponse], error) {
	return nil, nil
}

func (f testExternalDB) AddPeer(string, *grpc.ClientConn) error { return nil }
func (f testExternalDB) RemovePeer(string) error                { return nil }
func (f testExternalDB) GetAllCommits() ([]db.Commit, error)    { return nil, nil }
func (f testExternalDB) ExecuteSQL(context.Context, string, int) (db.SQLResult, error) {
	return db.SQLResult{}, nil
}
func (f testExternalDB) GetLastCommit(string) (db.Commit, error) { return db.Commit{}, nil }
func (f testExternalDB) CatchUpCheckpoint(context.Context, string) error {
	return nil
}
func (f testExternalDB) CatchUpCheckpointStrict(context.Context, string) error {
	return nil
}
func (f testExternalDB) InitFromPeerContext(context.Context, string, []string) error { return nil }
func (f testExternalDB) EnableGRPCServers(*grpc.Server) error {
	return nil
}
func (f testExternalDB) Initialized() bool { return f.initialized }

func (f *recordingExternalDB) RemovePeer(peerID string) error {
	f.removed = append(f.removed, peerID)
	return nil
}

func (f *recordingExternalDB) ConnectPeerIPs(peerID string, ips []string) error {
	f.connectedIPs = append(f.connectedIPs, recordedDBPeerConnect{
		peerID: peerID,
		ips:    append([]string(nil), ips...),
	})
	return nil
}

func (f runtimeStateExternalDB) CatchUpCheckpointStrict(context.Context, string) error {
	return f.strictErr
}

func (f runtimeStateExternalDB) SwarmionStatus() (swarmionapp.Status, bool) {
	return swarmionapp.Status{}, true
}

func (f runtimeStateExternalDB) SwarmionCompatibility(context.Context) ([]swarmionapp.ManifestCompatibility, error) {
	return nil, nil
}

func (f runtimeStateExternalDB) SwarmionPeerStatus(context.Context) ([]swarmionapp.PeerStatus, error) {
	return nil, nil
}

func (f runtimeStateExternalDB) SwarmionContentSyncTrace() ([]string, bool) {
	return nil, false
}
