package p2p

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/attic-labs/noms/go/chunks"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	noise "github.com/libp2p/go-libp2p-noise"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/protosio/protos/internal/ssh"
	"github.com/protosio/protos/internal/util"
	"github.com/segmentio/ksuid"
)

var log = util.GetLogger("p2p")

const protosRPCProtocol = "/protos/rpc/0.0.1"
const protosUpdatesTopic = "/protos/updates/0.0.1"

type RPCMsgType string

const RPCRequest RPCMsgType = "request"
const RPCResponse RPCMsgType = "response"

type DBSyncer interface {
	Sync(peerID string, dataset string, head string)
	HasCS(peerID string) bool
	AddRemoteCS(peerID string, cs chunks.ChunkStore)
}

type emptyReq struct{}
type emptyResp struct{}

type Handler struct {
	Func          func(data interface{}) (interface{}, error)
	RequestStruct interface{}
}

type rpcMsg struct {
	ID      string
	Type    RPCMsgType
	Payload json.RawMessage
}

type payloadRequest struct {
	Type string
	Data json.RawMessage
}

type payloadResponse struct {
	Error string
	Data  json.RawMessage
}

type payloadPubSub struct {
	ID      string
	Dataset string
	Head    string
}

type requestTracker struct {
	resp      chan []byte
	err       chan error
	closeSig  chan interface{}
	startTime time.Time
}

// Client is a remote p2p client
type Client struct {
	*ClientInit
	chunks.ChunkStore
	*ClientPing
}

type P2P struct {
	host         host.Host
	handlers     map[string]*Handler
	reqs         cmap.ConcurrentMap
	peerWriters  cmap.ConcurrentMap
	subscription *pubsub.Subscription
	topic        *pubsub.Topic
	dbSyncer     DBSyncer
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

func (p2p *P2P) newRPCStreamHandler(s network.Stream) {
	log.Debugf("New RPC stream for peer '%s'", s.Conn().RemotePeer().String())
	writeQueue := make(chan rpcMsg, 200)
	p2p.peerWriters.Set(s.Conn().RemotePeer().String(), writeQueue)
	go p2p.rpcReader(s, writeQueue)
	go p2p.rpcWriter(s, writeQueue)
}

func (p2p *P2P) rpcReader(s network.Stream, writeQueue chan rpcMsg) {
	stdReader := bufio.NewReader(s)
	for {
		buf, err := stdReader.ReadBytes('\n')
		if err != nil {
			if !strings.Contains(err.Error(), "stream reset") {
				log.Errorf("Connection error with peer '%s': %s", s.Conn().RemotePeer().String(), err.Error())
				return
			}
			s.Reset()
			log.Debugf("Remote peer '%s' closed the connection", s.Conn().RemotePeer().String())
			return
		}

		// we process the request in a separate routine
		go func(msgBytes []byte) {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("Exception whie processing incoming p2p RPC msg from '%s': %v", s.Conn().RemotePeer().String(), r)
				}
			}()

			msg := rpcMsg{}
			err = json.Unmarshal(msgBytes, &msg)
			if err != nil {
				log.Errorf("Failed to decode RPC message from '%s': %s", s.Conn().RemotePeer().String(), err.Error())
				return
			}

			if msg.Type == RPCRequest {
				// unmarshal remote request
				reqMsg := payloadRequest{}
				err = json.Unmarshal(msg.Payload, &reqMsg)
				if err != nil {
					log.Errorf("Failed to decode request from '%s': %s", s.Conn().RemotePeer().String(), err.Error())
					return
				}
				p2p.requestHandler(msg.ID, s.Conn().RemotePeer().String(), reqMsg, writeQueue)
			} else if msg.Type == RPCResponse {
				// unmarshal remote request
				respMsg := payloadResponse{}
				err = json.Unmarshal(msg.Payload, &respMsg)
				if err != nil {
					log.Errorf("Failed to decode response from '%s': %s", s.Conn().RemotePeer().String(), err.Error())
					return
				}
				p2p.responseHandler(msg.ID, s.Conn().RemotePeer().String(), respMsg)
			} else {
				log.Errorf("Wrong RPC message type from '%s': '%s'", s.Conn().RemotePeer().String(), msg.Type)
			}
		}(buf)
	}
}

func (p2p *P2P) rpcWriter(s network.Stream, writeQueue chan rpcMsg) {
	for {
		msg := <-writeQueue

		// encode the full response
		jsonMsg, err := json.Marshal(msg)
		if err != nil {
			log.Errorf("Failed to encode msg '%s'(%s) for '%s': %s", msg.ID, msg.Type, s.Conn().RemotePeer().String(), err.Error())
			continue
		}

		jsonMsg = append(jsonMsg, '\n')
		_, err = s.Write(jsonMsg)
		if err != nil {
			log.Errorf("Failed to send msg '%s'(%s) to '%s': %s", msg.ID, msg.Type, s.Conn().RemotePeer().String(), err.Error())
			continue
		}
	}
}

func (p2p *P2P) requestHandler(id string, peerID string, request payloadRequest, writeQueue chan rpcMsg) {
	log.Tracef("Remote request '%s' from peer '%s': %v", id, peerID, request)

	msg := rpcMsg{
		ID:   id,
		Type: RPCResponse,
	}

	response := payloadResponse{}

	// find handler
	handler, err := p2p.getHandler(request.Type)
	if err != nil {
		log.Errorf("Failed to process request '%s' from '%s': %s", id, peerID, err.Error())
		response.Error = err.Error()

		// encode the response
		jsonResp, err := json.Marshal(response)
		if err != nil {
			log.Errorf("Failed to encode response for request '%s' from '%s': %s", id, peerID, err.Error())
			return
		}
		msg.Payload = jsonResp
		writeQueue <- msg
		return
	}

	// execute handler method
	data := handler.RequestStruct
	err = json.Unmarshal(request.Data, &data)
	if err != nil {
		response.Error = fmt.Errorf("failed to decode data struct: %s", err.Error()).Error()

		// encode the response
		jsonResp, err := json.Marshal(response)
		if err != nil {
			log.Errorf("Failed to encode response for request '%s' from '%s': %s", id, peerID, err.Error())
			return
		}

		msg.Payload = jsonResp
		writeQueue <- msg
		return
	}

	var jsonHandlerResponse []byte
	handlerResponse, err := handler.Func(data)
	if err != nil {
		log.Errorf("Failed to process request '%s' from '%s': %s", id, peerID, err.Error())
	} else {
		// encode the returned handler response
		jsonHandlerResponse, err = json.Marshal(handlerResponse)
		if err != nil {
			log.Errorf("Failed to encode response for request '%s' from '%s': %s", id, peerID, err.Error())
		}
	}

	// add response data or error
	if err != nil {
		response.Error = fmt.Sprintf("Internal error: %s", err)
	} else {
		response.Data = jsonHandlerResponse
	}

	// encode the response
	jsonResp, err := json.Marshal(response)
	if err != nil {
		log.Errorf("Failed to encode response for request '%s' from '%s': %s", id, peerID, err.Error())
		return
	}
	msg.Payload = jsonResp
	log.Tracef("Sending response for msg '%s' to peer '%s': %v", id, peerID, response)

	// send the response
	writeQueue <- msg
}

func (p2p *P2P) responseHandler(id string, peerID string, response payloadResponse) {
	log.Tracef("Received response '%s' from peer '%s': %v", id, peerID, response)

	reqInteface, found := p2p.reqs.Get(id)
	if !found {
		log.Errorf("Failed to process response '%s' from '%s': request not found", id, peerID)
		return
	}

	req := reqInteface.(*requestTracker)

	// if the closeSig channel is closed, the request has timed out, so we return without sending the response received
	select {
	case <-req.closeSig:
		return
	default:
	}

	close(req.closeSig)

	if response.Error != "" {
		req.err <- fmt.Errorf("error returned by '%s': %s", peerID, response.Error)
	} else {
		req.resp <- response.Data
	}

	close(req.resp)
	close(req.err)
}

func (p2p *P2P) sendRequest(id peer.ID, msgType string, requestData interface{}, responseData interface{}) error {
	msg := rpcMsg{
		ID:   ksuid.New().String(),
		Type: RPCRequest,
	}

	// encode the request data
	jsonReqData, err := json.Marshal(requestData)
	if err != nil {
		return fmt.Errorf("failed to encode data for request '%s' for peer '%s': %s", msg.ID, id.String(), err.Error())
	}

	request := &payloadRequest{
		Type: msgType,
		Data: jsonReqData,
	}

	// encode the request
	jsonReq, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to encode request '%s' for peer '%s': %s", msg.ID, id.String(), err.Error())
	}
	msg.Payload = jsonReq

	// create the request tracker
	reqTracker := &requestTracker{
		resp:      make(chan []byte),
		err:       make(chan error),
		closeSig:  make(chan interface{}),
		startTime: time.Now(),
	}
	p2p.reqs.Set(msg.ID, reqTracker)

	log.Tracef("Sending request '%s' to '%s': %s", msgType, id.String(), string(jsonReq))

	writer, found := p2p.peerWriters.Get(id.String())
	if !found {
		return fmt.Errorf("failed to send request '%s' for peer '%s': peer writer not found", msg.ID, id.String())
	}

	writeQueue := writer.(chan rpcMsg)
	// send the request
	writeQueue <- msg

	go func() {
		// we sleep for the timeout period
		time.Sleep(time.Second * 5)

		// if the closeSig channel is closed, the request has been processed, so we return without sending the timeout error and closing the chans
		select {
		case <-reqTracker.closeSig:
			return
		default:
		}

		// we close the closeSig channel so any response from the handler is discarded
		close(reqTracker.closeSig)

		reqTracker.err <- fmt.Errorf("timeout waiting for request '%s'(%s) to peer '%s'", msg.ID, request.Type, id.String())
		close(reqTracker.resp)
		close(reqTracker.err)
	}()

	// wait for response or error and return it, while also deleting the request
	defer p2p.reqs.Remove(msg.ID)
	select {
	case resp := <-reqTracker.resp:
		err := json.Unmarshal(resp, responseData)
		if err != nil {
			return fmt.Errorf("failed to decode response payload: %v", err)
		}
		return nil
	case err := <-reqTracker.err:
		return err
	}

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

// AddPeer adds a peer to the p2p manager
func (p2p *P2P) AddPeer(pubKey []byte, destHost string) (string, error) {
	pk, err := crypto.UnmarshalEd25519PublicKey(pubKey)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshall public key: %w", err)
	}
	peerID, err := peer.IDFromPublicKey(pk)
	if err != nil {
		return "", fmt.Errorf("failed to create peer ID from public key: %w", err)
	}

	destinationString := fmt.Sprintf("/ip4/%s/tcp/10500/p2p/%s", destHost, peerID.String())
	maddr, err := multiaddr.NewMultiaddr(destinationString)
	if err != nil {
		return "", fmt.Errorf("failed to create multi address: %w", err)
	}

	peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return "", fmt.Errorf("failed to extrat info from address: %w", err)
	}

	log.Debugf("Adding peer id '%s'", peerInfo.ID.String())

	p2p.host.Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, 10*time.Second)
	return string(peerInfo.ID), nil
}

// GetClient returns the remote client that can reach all remote handlers
func (p2p *P2P) GetClient(peerID string) (client *Client, err error) {

	pID, err := peer.IDFromString(peerID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse peer ID from string: %w", err)
	}

	s, err := p2p.host.NewStream(context.Background(), pID, protosRPCProtocol)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream with peer '%s': %w", peerID, err)
	}
	p2p.newRPCStreamHandler(s)

	// err should be nil, and if there is a panic, we change it in the defer function
	// this is required because noms implements control flow using panics
	err = nil
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("exception whie building p2p client: %v", r)
			// if strings.Contains(err.Error(), "timeout waiting for request") {
			// 	err = fmt.Errorf("timeout waiting for p2p client")
			// }
		}
	}()

	client = &Client{
		NewRemoteInit(p2p, pID),
		NewRemoteChunkStore(p2p, pID),
		NewRemotePing(p2p, pID),
	}

	_, err = client.Ping()
	if err != nil {
		return nil, err
	}

	return client, err
}

// GetClient returns the remote client that can reach all remote handlers
func (p2p *P2P) GetClientForPeer(peer peer.ID) (client *Client, err error) {

	// err should be nil, and if there is a panic, we change it in the defer function
	// this is required because noms implements control flow using panics
	err = nil
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("exception whie building p2p client: %v", r)
			// if strings.Contains(err.Error(), "timeout waiting for request") {
			// 	err = fmt.Errorf("timeout waiting for p2p client")
			// }
		}
	}()

	client = &Client{
		NewRemoteInit(p2p, peer),
		NewRemoteChunkStore(p2p, peer),
		NewRemotePing(p2p, peer),
	}

	_, err = client.Ping()
	if err != nil {
		return nil, err
	}

	return client, err
}

func (p2p *P2P) pubSubMsgHandler() func() error {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("Exception whie processing incoming p2p message: %v", r)
			}
		}()

		for {
			msg, err := p2p.subscription.Next(ctx)
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					log.Errorf("Failed to retrieve pub sub message: %s", err.Error())
				}
				return
			}
			peerID := msg.ReceivedFrom.String()
			if msg.ReceivedFrom == p2p.host.ID() {
				continue
			}

			var data payloadPubSub
			err = json.Unmarshal(msg.Data, &data)
			if err != nil {
				log.Errorf("Failed to decode pub sub message from '%s': %w", peerID, err.Error())
				continue
			}
			log.Debugf("Peer '%s' advertised dataset '%s' head '%s'", peerID, data.Dataset, data.Head)

			if !p2p.dbSyncer.HasCS(peerID) {
				p2pClient, err := p2p.GetClientForPeer(msg.ReceivedFrom)
				if err != nil {
					log.Errorf("Failed to retrieve p2p client for '%s': %s", peerID, err.Error())
					continue
				}
				p2p.dbSyncer.AddRemoteCS(peerID, p2pClient)
			}

			p2p.dbSyncer.Sync(peerID, data.Dataset, data.Head)
		}
	}()

	stopper := func() error {
		log.Debug("Stopping p2p server")
		cancel()
		return p2p.host.Close()
	}
	return stopper
}

// StartServer starts listening for p2p connections
func (p2p *P2P) StartServer(metaConfigurator MetaConfigurator, userCreator UserCreator, cs chunks.ChunkStore) (func() error, error) {
	log.Info("Starting p2p server")

	p2pPing := &HandlersPing{}
	p2pInit := &HandlersInit{p2p: p2p, metaConfigurator: metaConfigurator, userCreator: userCreator}
	p2pChunkStore := &HandlersChunkStore{p2p: p2p, cs: cs}

	// we register handler methods which should be accessible from the client
	// ping handler
	p2p.addHandler(pingHandler, &Handler{Func: p2pPing.PerformPing, RequestStruct: &PingReq{}})
	// init handler
	p2p.addHandler(initHandler, &Handler{Func: p2pInit.PerformInit, RequestStruct: &InitReq{}})
	// db handlers
	p2p.addHandler(getRootHandler, &Handler{Func: p2pChunkStore.getRoot, RequestStruct: &emptyReq{}})
	p2p.addHandler(setRootHandler, &Handler{Func: p2pChunkStore.setRoot, RequestStruct: &setRootReq{}})
	p2p.addHandler(writeValueHandler, &Handler{Func: p2pChunkStore.writeValue, RequestStruct: &writeValueReq{}})
	p2p.addHandler(getStatsSummaryHandler, &Handler{Func: p2pChunkStore.getStatsSummary, RequestStruct: &emptyReq{}})
	p2p.addHandler(getRefsHandler, &Handler{Func: p2pChunkStore.getRefs, RequestStruct: &getRefsReq{}})
	p2p.addHandler(hasRefsHandler, &Handler{Func: p2pChunkStore.hasRefs, RequestStruct: &hasRefsReq{}})

	err := p2p.host.Network().Listen()
	if err != nil {
		return func() error { return nil }, fmt.Errorf("failed to listen: %w", err)
	}

	stopper := p2p.pubSubMsgHandler()
	return stopper, nil

}

func (p2p *P2P) Broadcast(dataset string, head string) error {
	data := payloadPubSub{
		ID:      ksuid.New().String(),
		Dataset: dataset,
		Head:    head,
	}
	msgBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return p2p.topic.Publish(context.Background(), msgBytes)
}

// NewManager creates and returns a new p2p manager
func NewManager(port int, key *ssh.Key, dbSyncer DBSyncer) (*P2P, error) {
	p2p := &P2P{
		handlers:    map[string]*Handler{},
		reqs:        cmap.New(),
		peerWriters: cmap.New(),
		dbSyncer:    dbSyncer,
	}

	prvKey, err := crypto.UnmarshalEd25519PrivateKey(key.Private())
	if err != nil {
		return nil, err
	}

	host, err := libp2p.New(
		libp2p.Identity(prvKey),
		libp2p.ListenAddrStrings(
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port),
			fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", port),
		),
		libp2p.Security(noise.ID, noise.New),
		libp2p.DefaultTransports,
		libp2p.ConnectionManager(connmgr.NewConnManager(100, 400, time.Minute)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to setup p2p host: %w", err)
	}

	p2p.host = host
	p2p.host.SetStreamHandler(protosRPCProtocol, p2p.newRPCStreamHandler)

	pubSub, err := pubsub.NewFloodSub(context.Background(), host)
	if err != nil {
		return nil, fmt.Errorf("failed to setup PubSub channel: %w", err)
	}

	p2p.topic, err = pubSub.Join(protosUpdatesTopic)
	if err != nil {
		return nil, fmt.Errorf("failed to join PubSub topic '%s': %w", protosUpdatesTopic, err)
	}

	p2p.subscription, err = p2p.topic.Subscribe()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to PubSub topic '%s': %w", protosUpdatesTopic, err)
	}

	log.Debugf("Using host with ID '%s'", host.ID().String())
	return p2p, nil
}
