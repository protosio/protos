package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/helpers"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	noise "github.com/libp2p/go-libp2p-noise"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	"github.com/multiformats/go-multiaddr"
	"github.com/protosio/protos/internal/ssh"
	"github.com/protosio/protos/internal/util"
	"github.com/segmentio/ksuid"
)

var log = util.GetLogger("p2p")

const protosRequestProtocol = "/protos/request/0.0.1"
const protosResponseProtocol = "/protos/response/0.0.1"

type Handler interface {
	Do(data interface{}) (interface{}, error)
}

type payloadRequest struct {
	ID   string
	Type string
	Data interface{}
}

type payloadResponse struct {
	ID    string
	Error string
	Data  json.RawMessage
}

type request struct {
	resp      chan []byte
	err       chan error
	closeSig  chan interface{}
	startTime time.Time
}

type requests struct {
	*sync.RWMutex
	reqs map[string]*request
}

// Server is good
type Server struct {
	*InitProtocol
}

type P2P struct {
	host     host.Host
	srv      *Server
	handlers map[string]Handler
	reqs     *requests
}

func (p2p *P2P) getHandler(msgType string) (Handler, error) {
	if handler, found := p2p.handlers[msgType]; found {
		return handler, nil
	}
	return nil, fmt.Errorf("Handler for method '%s' not found", msgType)
}

func (p2p *P2P) addHandler(methodName string, handler Handler) {
	p2p.handlers[methodName] = handler
}

func (p2p *P2P) getRequest(id string) (*request, error) {
	p2p.reqs.RLock()
	defer p2p.reqs.RUnlock()
	if req, found := p2p.reqs.reqs[id]; found {
		return req, nil
	}
	return nil, fmt.Errorf("Could not find request with id '%s'", id)
}

func (p2p *P2P) addRequest(id string, req *request) {
	p2p.reqs.Lock()
	defer p2p.reqs.Unlock()
	p2p.reqs.reqs[id] = req
}

func (p2p *P2P) deleteRequest(id string) {
	p2p.reqs.RLock()
	defer p2p.reqs.RUnlock()
	delete(p2p.reqs.reqs, id)
}

func (p2p *P2P) streamRequestHandler(s network.Stream) {

	reqMsg := payloadRequest{}
	buf, err := ioutil.ReadAll(s)
	if err != nil {
		s.Reset()
		log.Println(err)
		return
	}
	s.Close()

	// unmarshal it
	err = json.Unmarshal(buf, &reqMsg)
	if err != nil {
		log.Println(err)
		return
	}

	log.Infof("Remote request '%s' from peer '%s'", reqMsg.Type, s.Conn().RemotePeer().String())

	respMsg := payloadResponse{
		ID: reqMsg.ID,
	}

	handler, err := p2p.getHandler(reqMsg.Type)
	if err != nil {
		log.Errorf("Failed to process request '%s' from '%s': %s", reqMsg.ID, s.Conn().RemotePeer().String(), err.Error())

		respMsg.Error = err.Error()

		// encode the response
		jsonResp, err := json.Marshal(respMsg)
		if err != nil {
			log.Errorf("Failed to encode response for request '%s' from '%s': %s", reqMsg.ID, s.Conn().RemotePeer().String(), err.Error())
			return
		}

		err = p2p.sendMsg(s.Conn().RemotePeer(), protosResponseProtocol, jsonResp)
		if err != nil {
			log.Errorf("Failed to send response to '%s': %s", s.Conn().RemotePeer().String(), err.Error())
			return
		}
		return
	}

	handlerResponse, err := handler.Do(reqMsg.Data)
	if err != nil {
		log.Errorf("Failed to process request '%s' from '%s': %s", reqMsg.ID, s.Conn().RemotePeer().String(), err.Error())
		return
	}

	// encode the returned handler response
	jsonHandler, err := json.Marshal(handlerResponse)
	if err != nil {
		log.Errorf("Failed to encode response for request '%s' from '%s': %s", reqMsg.ID, s.Conn().RemotePeer().String(), err.Error())
		return
	}

	// add response data or error
	if err != nil {
		respMsg.Error = err.Error()
	} else {
		respMsg.Data = jsonHandler
	}

	// encode the full response
	jsonResp, err := json.Marshal(respMsg)
	if err != nil {
		log.Errorf("Failed to encode response for request '%s' from '%s': %s", reqMsg.ID, s.Conn().RemotePeer().String(), err.Error())
		return
	}

	// send the response
	err = p2p.sendMsg(s.Conn().RemotePeer(), protosResponseProtocol, jsonResp)
	if err != nil {
		log.Errorf("Failed to send response to '%s': %s", s.Conn().RemotePeer().String(), err.Error())
		return
	}
}

func (p2p *P2P) streamResponseHandler(s network.Stream) {

	msg := &payloadResponse{}
	buf, err := ioutil.ReadAll(s)
	if err != nil {
		s.Reset()
		log.Errorf("Failed to read from peer '%s' stream: %s", s.Conn().RemotePeer().String(), err.Error())
		return
	}
	s.Close()

	// unmarshal it
	err = json.Unmarshal(buf, msg)
	if err != nil {
		log.Errorf("Failed to decode response from '%s': %s", s.Conn().RemotePeer().String(), err.Error())
		return
	}

	log.Infof("Received response '%s' from peer '%s'", msg.ID, s.Conn().RemotePeer().String())

	req, err := p2p.getRequest(msg.ID)
	if err != nil {
		log.Errorf("Failed to process response '%s' from '%s': %s", msg.ID, s.Conn().RemotePeer().String(), err.Error())
		return
	}

	// if the closeSig channel is closed, the request has been processed, so we return without sending the timeout error and closing the chans
	select {
	case <-req.closeSig:
		return
	default:
	}

	close(req.closeSig)

	if msg.Error != "" {
		req.err <- fmt.Errorf("Error returned by '%s': %s", s.Conn().RemotePeer().String(), msg.Error)
	} else {
		req.resp <- msg.Data
	}

	close(req.resp)
	close(req.err)
}

func (p2p *P2P) sendRequest(id peer.ID, msgType string, requestData interface{}, responseData interface{}) error {
	reqMsg := &payloadRequest{
		ID:   ksuid.New().String(),
		Type: msgType,
		Data: requestData,
	}

	// encode the request
	jsonReq, err := json.Marshal(reqMsg)
	if err != nil {
		return fmt.Errorf("Failed to encode request '%s' for peer '%s': %s", reqMsg.ID, id.String(), err.Error())
	}

	// create the request
	req := &request{
		resp:      make(chan []byte),
		err:       make(chan error),
		closeSig:  make(chan interface{}),
		startTime: time.Now(),
	}
	p2p.addRequest(reqMsg.ID, req)

	// send the request
	p2p.sendMsg(id, protosRequestProtocol, jsonReq)
	if err != nil {
		return fmt.Errorf("Failed to encode request '%s' for peer '%s': %s", reqMsg.ID, id.String(), err.Error())
	}

	go func() {
		// we sleep for the timeout period
		time.Sleep(time.Second * 5)

		// if the closeSig channel is closed, the request has been processed, so we return without sending the timeout error and closing the chans
		select {
		case <-req.closeSig:
			fmt.Println("timeout not used")
			return
		default:
		}

		// we close the closeSig channel so any response from the handler is discarded
		close(req.closeSig)

		req.err <- fmt.Errorf("Timeout waiting for request '%s'", reqMsg.ID)
		close(req.resp)
		close(req.err)
	}()

	// wait for response or error and return it, while also deleting the request
	defer p2p.deleteRequest(reqMsg.ID)
	select {
	case resp := <-req.resp:
		err := json.Unmarshal(resp, responseData)
		if err != nil {
			return fmt.Errorf("Failed to decode response payload: %v", err)
		}
		return nil
	case err := <-req.err:
		return err
	}

}

func (p2p *P2P) sendMsg(id peer.ID, protocolType protocol.ID, jsonMsg []byte) error {
	s, err := p2p.host.NewStream(context.Background(), id, protocolType)
	if err != nil {
		return err
	}

	// send the data
	_, err = s.Write(jsonMsg)
	if err != nil {
		return fmt.Errorf("Error while writing to stream: %v", err)
	}

	// close the stream and wait for the other side to close their half.
	err = helpers.FullClose(s)
	if err != nil {
		s.Reset()
		return fmt.Errorf("Error while closing stream: %v", err)
	}

	return nil
}

// Listen starts listening for p2p connections
func (p2p *P2P) Listen() (func() error, error) {
	log.Info("Starting p2p server")

	err := p2p.host.Network().Listen()
	if err != nil {
		return func() error { return nil }, fmt.Errorf("Failed to listen: %w", err)
	}

	// we register the handler for the init method
	p2p.addHandler("init", p2p.srv.InitProtocol)

	stopper := func() error {
		log.Debug("Stopping p2p server")
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

// GetSrv returns the server that implements all the remote handlers
func (p2p *P2P) GetSrv() *Server {
	return p2p.srv
}

// NewManager creates and returns a new p2p manager
func NewManager(port int, key *ssh.Key) (*P2P, error) {
	p2p := &P2P{
		handlers: map[string]Handler{},
		reqs:     &requests{&sync.RWMutex{}, map[string]*request{}},
	}

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

	p2p.host = host
	p2p.srv = &Server{
		NewInitProtocol(p2p),
	}

	p2p.host.SetStreamHandler(protosRequestProtocol, p2p.streamRequestHandler)
	p2p.host.SetStreamHandler(protosResponseProtocol, p2p.streamResponseHandler)

	return p2p, nil
}
