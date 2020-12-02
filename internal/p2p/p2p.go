package p2p

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/helpers"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	noise "github.com/libp2p/go-libp2p-noise"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	"github.com/multiformats/go-multiaddr"
	"github.com/protosio/protos/internal/ssh"
	"github.com/protosio/protos/internal/util"
)

var log = util.GetLogger("p2p")

func writeStream(rw *bufio.ReadWriter, data []byte) {
	rw.Write(data)
	rw.Flush()
}

func handleStream(s network.Stream) {
	fmt.Println("Got a stream from: ", s.Conn().RemotePeer())
	fmt.Println("Proto: ", s.Protocol())

	buf, err := ioutil.ReadAll(s)
	if err != nil {
		log.Errorf("Failed to read from stream: %s", err.Error())
		return
	}
	s.Close()
	fmt.Println(string(buf))
}

type P2P struct {
	host host.Host
}

// Listen starts listening for p2p connections
func (p2p *P2P) Listen() (func() error, error) {
	log.Info("Starting p2p server")

	err := p2p.host.Network().Listen()
	if err != nil {
		return func() error { return nil }, fmt.Errorf("Failed to listen: %w", err)
	}

	stopper := func() error {
		log.Debug("Stopping down DNS server")
		return p2p.host.Close()
	}
	return stopper, nil

}

// Connect to a p2p node
func (p2p *P2P) Connect(id string) error {
	peerID, err := peer.IDFromString(id)
	if err != nil {
		return fmt.Errorf("Failed to parse peer ID from string: %w", err)
	}

	log.Infof("Connecting to peer ID '%s'", peerID.String())

	str, err := p2p.host.NewStream(context.Background(), peerID, syncProtocolID)
	if err != nil {
		return fmt.Errorf("Failed to start stream: %w", err)
	}

	_, err = str.Write([]byte("tester\n"))
	if err != nil {
		return fmt.Errorf("Failed to write to stream: %w", err)
	}

	err = helpers.FullClose(str)
	if err != nil {
		str.Reset()
		return err
	}

	return nil
}

// AddPeer adds a peer to the p2p manager
func (p2p *P2P) AddPeer(pubKey []byte, dest string) (string, error) {
	pk, err := crypto.UnmarshalEd25519PublicKey(pubKey)
	if err != nil {
		return "", fmt.Errorf("Failed to unmarshall public key: %w", err)
	}
	peerID, err := peer.IDFromPublicKey(pk)
	if err != nil {
		return "", fmt.Errorf("Failed to create peer ID from public key: %w", err)
	}

	destinationString := fmt.Sprintf("/ip4/%s/tcp/10500/p2p/%s", dest, peerID.String())
	maddr, err := multiaddr.NewMultiaddr(destinationString)
	if err != nil {
		return "", fmt.Errorf("Failed to create multi address: %w", err)
	}

	peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return "", fmt.Errorf("Failed to extrat info from address: %w", err)
	}

	log.Debugf("Adding peer id '%s'", peerInfo.ID.String())

	p2p.host.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, peerstore.PermanentAddrTTL)
	return string(peerInfo.ID), nil
}

// NewManager creates and returns a new p2p manager
func NewManager(port int, key *ssh.Key) (*P2P, error) {
	p2p := &P2P{}

	prvKey, err := crypto.UnmarshalEd25519PrivateKey(key.Private())
	if err != nil {
		return p2p, err
	}

	host, err := libp2p.New(context.Background(),
		libp2p.Identity(prvKey),
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port),
			fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", port),
		),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Transport(libp2pquic.NewTransport),
		libp2p.DefaultTransports,
		libp2p.ConnectionManager(connmgr.NewConnManager(100, 400, time.Minute)),
	)
	if err != nil {
		return p2p, err
	}

	log.Debugf("Using host with ID '%s'", host.ID().String())

	host.SetStreamHandler(syncProtocolID, handleStream)
	p2p.host = host

	return p2p, nil
}
