package p2p

import (
	"bufio"
	"context"
	"crypto/ed25519"
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
	// fmt.Fprintln(s, "Hello Friend!")
}

type P2P struct {
	host host.Host
}

// Listen starts listening for p2p connections
func (p2p *P2P) Listen() error {
	log.Info("Starting server")

	err := p2p.host.Network().Listen()
	if err != nil {
		return fmt.Errorf("Failed to listen: %w", err)
	}

	// Hang forever
	<-make(chan struct{})
	return nil
}

// Connect to a p2p node
func (p2p *P2P) Connect(dest string) error {
	log.Info("Starting client")

	// pubKey, err := crypto.UnmarshalEd25519PublicKey([]byte(pubKeyString))
	// if err != nil {
	// 	return fmt.Errorf("Failed to unmarshall pub key: %w", err)
	// }

	// pid, err := peer.IDFromPublicKey(pubKey)
	// if err != nil {
	// 	return fmt.Errorf("Failed to get ID from pub key: %w", err)
	// }

	maddr, err := multiaddr.NewMultiaddr(dest)
	if err != nil {
		return fmt.Errorf("Failed to create multi address: %w", err)
	}

	peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("Failed to extrat info from address: %w", err)
	}

	str, err := p2p.host.NewStream(context.Background(), peerInfo.ID, syncProtocolID)
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
func (p2p *P2P) AddPeer(pubKeyString string, dest string) error {
	maddr, err := multiaddr.NewMultiaddr(dest)
	if err != nil {
		return fmt.Errorf("Failed to create multi address: %w", err)
	}

	peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("Failed to extrat info from address: %w", err)
	}

	// pubKey, err := crypto.UnmarshalEd25519PublicKey([]byte(pubKeyString))
	// if err != nil {
	// 	return err
	// }

	// pid, err := peer.IDFromPublicKey(pubKey)
	// if err != nil {
	// 	return err
	// }

	p2p.host.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, peerstore.PermanentAddrTTL)
	return nil
}

// NewManager creates and returns a new p2p manager
func NewManager(port int, privKey ed25519.PrivateKey) (*P2P, error) {
	p2p := &P2P{}

	prvKey, err := crypto.UnmarshalEd25519PrivateKey(privKey)
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

	host.SetStreamHandler(syncProtocolID, handleStream)
	p2p.host = host

	return p2p, nil
}
