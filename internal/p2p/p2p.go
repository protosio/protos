package p2p

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"
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
	p2pproto "github.com/protosio/protos/internal/p2p/proto"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var log = util.GetLogger("p2p")

const (
	protosRPCProtocol              = "/protos/rpc/0.0.1"
	protosUpdatesTopic             = "/protos/updates/0.0.1"
	destinationStringTemplate      = "/ip4/%s/udp/%d/quic-v1/%s"
	p2pPort                   uint = 10500
	initMachineName                = "initMachine"
)

type AppManager interface {
	GetLogs(name string) ([]byte, error)
	GetStatus(name string) (string, error)
}

type Machine interface {
	GetPublicKey() string
	GetPublicIP() string
	GetName() string
}

type rpcPeer struct {
	mu      sync.Mutex
	machine Machine
	client  *Client
}

func (peer *rpcPeer) GetClient() *Client {
	peer.mu.Lock()
	defer peer.mu.Unlock()
	return peer.client
}

func (peer *rpcPeer) SetClient(client *Client) {
	peer.mu.Lock()
	defer peer.mu.Unlock()
	peer.client = client
}

func (peer *rpcPeer) GetMachine() Machine {
	peer.mu.Lock()
	defer peer.mu.Unlock()
	return peer.machine
}

func (peer *rpcPeer) SetMachine(machine Machine) {
	peer.mu.Lock()
	defer peer.mu.Unlock()
	peer.machine = machine
}

// Client is a remote p2p client
type Client struct {
	p2pproto.PingerClient
	p2pproto.TesterClient
	p2pproto.AppsClient
	p2pproto.InstanceClient

	peer peer.ID
}

type P2P struct {
	host        host.Host
	peers       *util.Map[string, *rpcPeer]
	appManager  AppManager
	grpcServer  *grpc.Server
	newPeerChan chan peer.AddrInfo
	initMode    bool

	externalDB ExternalDB
}

func (p2p *P2P) GetGRPCServer() *grpc.Server {
	return p2p.grpcServer
}

// GetPeerID adds a peer to the p2p manager
func (p2p *P2P) PubKeyToPeerID(pubKey []byte) (string, error) {
	pk, err := crypto.UnmarshalEd25519PublicKey(pubKey)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshall public key: %w", err)
	}
	peerID, err := peer.IDFromPublicKey(pk)
	if err != nil {
		return "", fmt.Errorf("failed to create peer ID from public key: %w", err)
	}
	return peerID.String(), nil
}

func (p2p *P2P) GetClient(name string) (*Client, error) {
	for _, rpcpeer := range p2p.peers.Snapshot() {
		client := rpcpeer.GetClient()
		machine := rpcpeer.GetMachine()
		if machine != nil && client != nil && machine.GetName() == name {
			return client, nil
		}
	}

	return nil, fmt.Errorf("could not find RPC client for instance '%s'", name)
}

// getRPCPeer returns the rpc client for a peer
func (p2p *P2P) getRPCPeer(peerID peer.ID) (*rpcPeer, error) {
	rpcpeer, found := p2p.peers.Get(peerID.String())
	if found {
		return rpcpeer, nil
	}
	return nil, fmt.Errorf("could not find RPC peer '%s'", peerID.String())
}

// createClientForPeer returns the remote client that can reach all remote handlers
func (p2p *P2P) createClientForPeer(peerID peer.ID) (client *Client, err error) {

	// grpc conn
	conn, err := grpc.Dial(
		peerID.String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		p2pgrpc.WithP2PDialer(p2p.host, protosRPCProtocol),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to grpc dial peer '%s': %w", peerID.String(), err)
	}

	client = &Client{
		PingerClient:   p2pproto.NewPingerClient(conn),
		TesterClient:   p2pproto.NewTesterClient(conn),
		AppsClient:     p2pproto.NewAppsClient(conn),
		InstanceClient: p2pproto.NewInstanceClient(conn),
		peer:           peerID,
	}

	tries := 0
	for {
		_, err = client.Ping(context.TODO(), &p2pproto.PingRequest{
			Ping: "pong",
		})
		if err != nil {
			if tries < 19 {
				time.Sleep(200 * time.Millisecond)
				tries++
				continue
			} else {
				return nil, err
			}
		} else {
			break
		}
	}

	return client, nil
}

//
// Methods for handling new peers
//

// ConfigurePeers configures all the peers passed as arguemnt
func (p2p *P2P) ConfigurePeers(machines []Machine) error {
	currentPeers := map[string]peer.ID{}
	log.Debugf("Configuring p2p peers")

	// add new peers
	for _, machine := range machines {
		client, err := p2p.AddPeer(machine)
		if err != nil {
			log.Errorf("Failed to add peer '%s': %s", machine.GetName(), err.Error())
			continue
		}
		currentPeers[client.peer.String()] = client.peer
	}

	// delete old peers
	for id, rpcpeer := range p2p.peers.Snapshot() {
		if _, found := currentPeers[id]; !found {
			name := "unknown"
			machine := rpcpeer.GetMachine()
			if machine != nil {
				name = machine.GetName()
			}
			if name == initMachineName {
				continue
			}
			log.Debugf("Removing old peer '%s'(%s)", name, id)
			p2p.peers.Delete(id)
			client := rpcpeer.GetClient()
			if client != nil {
				err := p2p.host.Network().ClosePeer(client.peer)
				if err != nil {
					log.Debugf("Failed to disconnect from old peer '%s'(%s)", id, name)
				}
			}
		}
	}

	return nil
}

// AddPeer adds a peer to the p2p manager
func (p2p *P2P) AddPeer(machine Machine) (*Client, error) {
	pubKeyBytes, err := base64.StdEncoding.DecodeString(machine.GetPublicKey())
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	pk, err := crypto.UnmarshalEd25519PublicKey(pubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall public key: %w", err)
	}
	peerID, err := peer.IDFromPublicKey(pk)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer ID from public key: %w", err)
	}

	rpcpeer, found := p2p.peers.Get(peerID.String())
	if !found {
		destinationString := ""
		if machine.GetPublicIP() != "" {
			destinationString = fmt.Sprintf(destinationStringTemplate, machine.GetPublicIP(), p2pPort, peerID.String())
		} else {
			destinationString = fmt.Sprintf("/p2p/%s", peerID.String())
		}
		maddr, err := multiaddr.NewMultiaddr(destinationString)
		if err != nil {
			return nil, fmt.Errorf("failed to create multi address: %w", err)
		}

		peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			return nil, fmt.Errorf("failed to extract info from address: %w", err)
		}
		rpcpeer := &rpcPeer{machine: machine}
		p2p.peers.Set(peerID.String(), rpcpeer)

		log.Debugf("Adding peer id '%s'(%s) at ip '%s'", machine.GetName(), peerInfo.ID.String(), machine.GetPublicIP())

		err = p2p.host.Connect(context.Background(), *peerInfo)
		if err != nil {
			log.Errorf("Failed to connect to peer '%s'(%s): %s", machine.GetName(), peerID.String(), err.Error())

		}

		client, err := p2p.createClientForPeer(peerID)
		if err != nil {
			return nil, fmt.Errorf("failed to add peer '%s': %w", peerID.String(), err)
		}
		rpcpeer.SetClient(client)
	} else if rpcpeer.GetMachine() != nil {
		if rpcpeer.GetMachine().GetName() == initMachineName {
			log.Infof("Replacing machine info for peer '%s' that triggerd initialisation", machine.GetName())
		} else {
			log.Infof("Replacing machine info for peer '%s'", machine.GetName())
		}
		rpcpeer.SetMachine(machine)
	}

	return rpcpeer.GetClient(), nil
}

func (p2p *P2P) newConnectionHandler(netw network.Network, conn network.Conn) {
	go func() {
		if conn.Stat().Transient {
			return
		}

		var rpcpeer *rpcPeer
		var machine Machine
		if p2p.initMode {
			machine = &initMachine{name: initMachineName}
			rpcpeer = &rpcPeer{machine: machine}
		} else {
			rpcpeer, found := p2p.peers.Get(conn.RemotePeer().String())
			if !found {
				log.Errorf("Peer '%s' not recognized while creating client", conn.RemotePeer().String())
				conn.Close()
				return
			}
			lmachine := rpcpeer.GetMachine()
			if lmachine != nil {
				machine = lmachine
			} else {
				machine = &initMachine{name: "unknown"}
			}
		}
		rpcpeer.SetMachine(machine)

		log.Debugf("New connection with peer '%s'(%s). Creating client", machine.GetName(), conn.RemotePeer().String())
		rpcClient, err := p2p.createClientForPeer(conn.RemotePeer())
		if err != nil {
			log.Errorf("Failed to create client for new peer '%s'(%s): %s", machine.GetName(), conn.RemotePeer().String(), err.Error())
			conn.Close()
			return
		}
		rpcpeer.SetClient(rpcClient)
	}()
}

//
// Methods for handling peer removal
//

func (p2p *P2P) closeConnectionHandler(netw network.Network, conn network.Conn) {
	log.Infof("Disconnected from %s", conn.RemotePeer().String())
	if err := conn.Close(); err != nil {
		log.Errorf("Error while disconnecting from peer '%s': %v", conn.RemotePeer().String(), err)
	}
	p2p.peers.Delete(conn.RemotePeer().String())
	if p2p.externalDB != nil {
		if err := p2p.externalDB.RemovePeer(conn.RemotePeer().String()); err != nil {
			log.Errorf("Failed to remove DB peer for '%s': %v", conn.RemotePeer().String(), err)
		}
	}
}

//
// Methods for creating and starting the p2p server
//

// StartServer starts listening for p2p connections
func (p2p *P2P) StartServer(metaConfigurator MetaConfigurator) (func() error, error) {
	log.Info("Starting p2p server")

	// register internal grpc servers
	srv := &Server{DB: p2p.externalDB, metaConfigurator: metaConfigurator, p2p: p2p}
	p2pproto.RegisterPingerServer(p2p.grpcServer, srv)
	p2pproto.RegisterTesterServer(p2p.grpcServer, srv)
	p2pproto.RegisterAppsServer(p2p.grpcServer, srv)
	p2pproto.RegisterInstanceServer(p2p.grpcServer, srv)

	// serve grpc server over libp2p host
	grpcListener := p2pgrpc.NewListener(context.Background(), p2p.host, protosRPCProtocol)
	go func() {
		err := p2p.grpcServer.Serve(grpcListener)
		if err != nil {
			log.Error("grpc serve error: ", err)
			panic(err)
		}
	}()

	err := p2p.host.Network().Listen()
	if err != nil {
		return func() error { return nil }, fmt.Errorf("failed to listen: %w", err)
	}

	stopper := func() error {
		log.Debug("Stopping p2p server")
		p2p.grpcServer.GracefulStop()
		return p2p.host.Close()
	}
	return stopper, nil

}

// NewManager creates and returns a new p2p manager
func NewManager(key *pcrypto.Key, appManager AppManager, initMode bool, externalDB ExternalDB) (*P2P, error) {
	p2p := &P2P{
		peers:       util.NewMap[string, *rpcPeer](),
		appManager:  appManager,
		newPeerChan: make(chan peer.AddrInfo),
		initMode:    initMode,

		externalDB: externalDB,
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

	log.Debugf("Using host with ID '%s'", host.ID().String())
	return p2p, nil
}
