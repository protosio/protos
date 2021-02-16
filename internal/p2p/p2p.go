package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sync"
	"time"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
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

type emptyReq struct{}
type emptyResp struct{}

type Handler struct {
	Func          func(data interface{}) (interface{}, error)
	RequestStruct interface{}
}

type payloadRequestClient struct {
	ID   string
	Type string
	Data interface{}
}

type payloadRequestServer struct {
	ID   string
	Type string
	Data json.RawMessage
}

type payloadResponse struct {
	ID    string
	Error string
	Data  json.RawMessage
}

type responseError struct {
	statusCode int
	err        error
}

func (r *responseError) Error() string {
	return r.err.Error()
}

func (r *responseError) StatusCode() int {
	return r.statusCode
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

// Client is a remote p2p client
type Client struct {
	*ClientInit
	chunks.ChunkStore
}

type P2P struct {
	host     host.Host
	handlers map[string]*Handler
	reqs     *requests
}

func (p2p *P2P) getHandler(msgType string) (*Handler, error) {
	if handler, found := p2p.handlers[msgType]; found {
		return handler, nil
	}
	return nil, fmt.Errorf("Handler for method '%s' not found", msgType)
}

func (p2p *P2P) addHandler(methodName string, handler *Handler) {
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

	reqMsg := payloadRequestServer{}
	buf, err := ioutil.ReadAll(s)
	if err != nil {
		s.Reset()
		log.Println(err)
		return
	}
	s.Close()

	// unmarshal remote request
	err = json.Unmarshal(buf, &reqMsg)
	if err != nil {
		log.Errorf("Failed to decode request from '%s': %s", s.Conn().RemotePeer().String(), err.Error())
		return
	}

	log.Tracef("Remote request '%s' from peer '%s': %s", reqMsg.Type, s.Conn().RemotePeer().String(), string(reqMsg.Data))

	respMsg := payloadResponse{
		ID: reqMsg.ID,
	}

	// find handler
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

	// execute handler method
	data := handler.RequestStruct
	err = json.Unmarshal(reqMsg.Data, &data)
	if err != nil {
		respMsg.Error = fmt.Errorf("Failed to decode data struct: %s", err.Error()).Error()

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

	var jsonHandlerResponse []byte
	handlerResponse, err := handler.Func(data)
	if err != nil {
		err = fmt.Errorf("Failed to process request '%s' from '%s': %s", reqMsg.ID, s.Conn().RemotePeer().String(), err.Error())
		log.Errorf(err.Error())
	} else {
		// encode the returned handler response
		jsonHandlerResponse, err = json.Marshal(handlerResponse)
		if err != nil {
			err = fmt.Errorf("Failed to encode response for request '%s' from '%s': %s", reqMsg.ID, s.Conn().RemotePeer().String(), err.Error())
			log.Errorf(err.Error())
		}
	}

	// add response data or error
	if err != nil {
		respMsg.Error = err.Error()
	} else {
		respMsg.Data = jsonHandlerResponse
	}

	// encode the full response
	jsonResp, err := json.Marshal(respMsg)
	if err != nil {
		log.Errorf("Failed to encode response for request '%s' from '%s': %s", reqMsg.ID, s.Conn().RemotePeer().String(), err.Error())
		return
	}

	log.Tracef("Sending response for msg '%s' to peer '%s': %s", respMsg.ID, s.Conn().RemotePeer().String(), string(jsonHandlerResponse))

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

	log.Tracef("Received response '%s' from peer '%s': %s", msg.ID, s.Conn().RemotePeer().String(), string(msg.Data))

	req, err := p2p.getRequest(msg.ID)
	if err != nil {
		log.Errorf("Failed to process response '%s' from '%s': %s", msg.ID, s.Conn().RemotePeer().String(), err.Error())
		return
	}

	// if the closeSig channel is closed, the request has timed out, so we return without sending the response received
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
	reqMsg := &payloadRequestClient{
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

	log.Tracef("Sending request '%s' to '%s': %s", msgType, id.String(), string(jsonReq))

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

	s.Close()

	// close the stream and wait for the other side to close their half.
	err = s.Close()
	if err != nil {
		s.Reset()
		return fmt.Errorf("Error while closing stream: %v", err)
	}

	return nil
}

// AddPeer adds a peer to the p2p manager
func (p2p *P2P) AddPeer(pubKey []byte, destHost string) (string, error) {
	pk, err := crypto.UnmarshalEd25519PublicKey(pubKey)
	if err != nil {
		return "", fmt.Errorf("Failed to unmarshall public key: %w", err)
	}
	peerID, err := peer.IDFromPublicKey(pk)
	if err != nil {
		return "", fmt.Errorf("Failed to create peer ID from public key: %w", err)
	}

	destinationString := fmt.Sprintf("/ip4/%s/tcp/10500/p2p/%s", destHost, peerID.String())
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

// GetClient returns the remote client that can reach all remote handlers
func (p2p *P2P) GetClient(peerID string) (*Client, error) {

	pID, err := peer.IDFromString(peerID)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse peer ID from string: %w", err)
	}

	return &Client{
		NewRemoteInit(p2p, pID),
		NewRemoteChunkStore(p2p, pID),
	}, nil
}

// StartServer starts listening for p2p connections
func (p2p *P2P) StartServer(metaConfigurator MetaConfigurator, userCreator UserCreator, cs chunks.ChunkStore) (func() error, error) {
	log.Info("Starting p2p server")

	p2pInit := &HandlersInit{p2p: p2p, metaConfigurator: metaConfigurator, userCreator: userCreator}
	p2pChunkStore := &HandlersChunkStore{cs: cs}

	// we register handler methods which should be accessible from the client
	p2p.addHandler(initHandler, &Handler{Func: p2pInit.PerformInit, RequestStruct: &InitReq{}})
	p2p.addHandler(getRootHandler, &Handler{Func: p2pChunkStore.getRoot, RequestStruct: &emptyReq{}})
	p2p.addHandler(setRootHandler, &Handler{Func: p2pChunkStore.setRoot, RequestStruct: &setRootReq{}})
	p2p.addHandler(writeValueHandler, &Handler{Func: p2pChunkStore.writeValue, RequestStruct: &writeValueReq{}})
	p2p.addHandler(getStatsSummaryHandler, &Handler{Func: p2pChunkStore.getStatsSummary, RequestStruct: &emptyReq{}})
	p2p.addHandler(getRefsHandler, &Handler{Func: p2pChunkStore.getRefs, RequestStruct: &getRefsReq{}})
	p2p.addHandler(hasRefsHandler, &Handler{Func: p2pChunkStore.hasRefs, RequestStruct: &hasRefsReq{}})

	err := p2p.host.Network().Listen()
	if err != nil {
		return func() error { return nil }, fmt.Errorf("Failed to listen: %w", err)
	}

	stopper := func() error {
		log.Debug("Stopping p2p server")
		return p2p.host.Close()
	}
	return stopper, nil

}

// NewManager creates and returns a new p2p manager
func NewManager(port int, key *ssh.Key) (*P2P, error) {
	p2p := &P2P{
		handlers: map[string]*Handler{},
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

	p2p.host = host
	p2p.host.SetStreamHandler(protosRequestProtocol, p2p.streamRequestHandler)
	p2p.host.SetStreamHandler(protosResponseProtocol, p2p.streamResponseHandler)

	log.Debugf("Using host with ID '%s'", host.ID().String())
	return p2p, nil
}
