package p2p

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	p2pgrpc "github.com/birros/go-libp2p-grpc"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	connmgr "github.com/libp2p/go-libp2p/p2p/net/connmgr"
	noise "github.com/libp2p/go-libp2p/p2p/security/noise"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/multiformats/go-multiaddr"
	"github.com/protosio/protos/internal/config"
	p2pproto "github.com/protosio/protos/internal/p2p/proto"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/util"
	"google.golang.org/grpc"
)

var log = util.GetLogger("p2p")

const (
	protosRPCProtocol         = "/protos/rpc/0.0.1"
	protosUpdatesTopic        = "/protos/updates/0.0.1"
	destinationStringTemplate = "/ip4/%s/udp/%d/quic-v1/p2p/%s"
)

type AppManager interface {
	GetLogs(name string) ([]byte, error)
	GetStatus(name string) (string, error)
}

type Machine interface {
	GetID() string
	GetPublicKey() string
	GetPublicIP() string
	GetName() string
}

// Client is a remote p2p client
type Client struct {
	p2pproto.PingerClient
	p2pproto.PeerDBClient
	p2pproto.AppsClient
	p2pproto.InstanceClient

	grpcConnection *grpc.ClientConn
}

type P2P struct {
	host       host.Host
	appManager AppManager
	grpcServer *grpc.Server
	externalDB ExternalDB

	restartServerSignal chan initMachine
	newPeerChan         chan peer.AddrInfo

	// the index is the peer ID
	machines *util.Map[string, Machine]
	// the index is the machine ID
	clients *util.Map[string, *Client]
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

func (p2p *P2P) GetClient(id string) (*Client, error) {
	client, found := p2p.clients.Get(id)
	if !found {
		return nil, fmt.Errorf("could not find RPC client for instance '%s'", id)
	}

	return client, nil
}

//
// Methods for handling new peers
//

// ConfigurePeers configures all the peers passed as arguemnt
func (p2p *P2P) ConfigurePeers(machines []Machine) error {
	currentMachines := map[string]Machine{}
	currentPeerIDs := map[string]peer.ID{}

	log.Debugf("configuring p2p peers: %v", machines)

	// create a map of the current machines and their IDs
	for _, machine := range machines {
		peerID, err := p2p.pubKeyToPeerID(machine.GetPublicKey())
		if err != nil {
			return fmt.Errorf("failed to get peer ID from public key: %w", err)
		}

		currentMachines[peerID.String()] = machine
		currentPeerIDs[machine.GetID()] = peerID
	}

	//
	// delete all machines that are not in the current list
	//
	for peerID := range p2p.machines.Snapshot() {
		if _, found := currentMachines[peerID]; !found {
			p2p.machines.Delete(peerID)
		}
	}

	//
	// disconnect and delete all clients that are not in the current list
	//
	for machineID := range p2p.clients.Snapshot() {
		peerID, found := currentPeerIDs[machineID]
		if !found {
			err := p2p.host.Network().ClosePeer(peerID)
			if err != nil {
				log.Debugf("failed to remove peer %s(%s)", machineID, peerID.String())
			}
			p2p.clients.Delete(machineID)
		}
	}

	//
	// add peers (this adds machines, clients and peer mappings)
	//
	for id, machine := range currentMachines {
		_, err := p2p.AddPeer(machine)
		if err != nil {
			log.Errorf("failed to add peer '%s': %s", id, err.Error())
		}
	}

	return nil
}

// AddPeer adds a peer to the p2p manager
func (p2p *P2P) AddPeer(machine Machine) (*Client, error) {

	peerID, err := p2p.pubKeyToPeerID(machine.GetPublicKey())
	if err != nil {
		return nil, fmt.Errorf("failed to convert public key to peer ID: %w", err)
	}

	p2p.machines.Set(peerID.String(), machine)

	client, found := p2p.clients.Get(machine.GetID())
	if found {
		return client, nil
	}

	if machine.GetPublicIP() == "" {
		return nil, nil
	}

	destinationString := fmt.Sprintf(destinationStringTemplate, machine.GetPublicIP(), config.Get().P2PPort, peerID.String())
	maddr, err := multiaddr.NewMultiaddr(destinationString)
	if err != nil {
		return nil, fmt.Errorf("failed to create multi address: %w", err)
	}

	peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return nil, fmt.Errorf("failed to extract info from address: %w", err)
	}

	log.Debugf("adding peer id %s (%s) at ip %s", machine.GetID(), peerID.String(), machine.GetPublicIP())

	err = p2p.host.Connect(context.Background(), *peerInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to peer %s (%s): %s", machine.GetID(), machine.GetID(), err.Error())
	}

	// wait for the client to be created by the connection handler
	tries := 0
	for tries < 5 {
		time.Sleep(1 * time.Second)
		client, found := p2p.clients.Get(machine.GetID())
		if found {
			return client, nil
		}
		tries++
	}

	return nil, fmt.Errorf("failed to retrieve client for peer %s (%s): client not found", machine.GetID(), peerID.String())
}

func (p2p *P2P) RemovePeer(machine Machine) error {
	peerID, err := p2p.pubKeyToPeerID(machine.GetPublicKey())
	if err != nil {
		return fmt.Errorf("failed to convert public key to peer ID: %w", err)
	}

	err = p2p.host.Network().ClosePeer(peerID)
	if err != nil {
		return fmt.Errorf("failed to remove peer %s(%s)", machine.GetID(), peerID.String())
	}

	p2p.machines.Delete(peerID.String())
	p2p.clients.Delete(machine.GetID())

	return nil
}

//
// Methods for creating and starting the p2p server
//

func (p2p *P2P) startGRPCServer() error {

	// register internal grpc servers
	srv := &Server{DB: p2p.externalDB, p2p: p2p}
	p2pproto.RegisterPingerServer(p2p.grpcServer, srv)
	p2pproto.RegisterPeerDBServer(p2p.grpcServer, srv)
	p2pproto.RegisterAppsServer(p2p.grpcServer, srv)
	p2pproto.RegisterInstanceServer(p2p.grpcServer, srv)

	if p2p.externalDB.Initialized() {
		err := p2p.externalDB.EnableGRPCServers(p2p.grpcServer)
		if err != nil {
			return fmt.Errorf("failed to enable grpc servers: %w", err)
		}
	}

	// serve grpc server over libp2p host
	grpcListener := p2pgrpc.NewListener(context.Background(), p2p.host, protosRPCProtocol)
	go func() {
		err := p2p.grpcServer.Serve(grpcListener)
		if err != nil {
			log.Error("grpc serve error: ", err)
			panic(err)
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

	err = p2p.host.Network().Listen()
	if err != nil {
		return func() error { return nil }, fmt.Errorf("failed to listen: %w", err)
	}

	stopper := func() error {
		log.Debug("Stopping p2p server")
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

		log.Info("removing init peer")
		err := p2p.RemovePeer(&im)
		if err != nil {
			log.Error("failed to remove init peer: ", err)
		}

		log.Info("restarting grpc server")

		p2p.grpcServer.GracefulStop()
		p2p.grpcServer = grpc.NewServer(p2pgrpc.WithP2PCredentials())

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
		grpcServer: grpc.NewServer(p2pgrpc.WithP2PCredentials()),
		externalDB: externalDB,

		restartServerSignal: make(chan initMachine),
		newPeerChan:         make(chan peer.AddrInfo),

		clients:  util.NewMap[string, *Client](),
		machines: util.NewMap[string, Machine](),
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
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", p2pPort),
		),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Transport(quic.NewTransport),
		libp2p.ConnectionManager(con),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to setup p2p host: %w", err)
	}

	p2p.host = host
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
