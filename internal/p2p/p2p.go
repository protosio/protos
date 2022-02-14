package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
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
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/util"
	"github.com/segmentio/ksuid"
)

var log = util.GetLogger("p2p")

type rpcMsgType string
type pubsubMsgType string

const (
	protosRPCProtocol                 = "/protos/rpc/0.0.1"
	protosUpdatesTopic                = "/protos/updates/0.0.1"
	rpcRequest          rpcMsgType    = "request"
	rpcResponse         rpcMsgType    = "response"
	pubsubBroadcastHead pubsubMsgType = "broadcasthead"
	pubsubRequestHead   pubsubMsgType = "requesthead"
	p2pPort             uint          = 10500
)

type DBSyncer interface {
	Sync(peerID string, dataset string, head string)
	GetAllDatasetsHeads() map[string]string
}

type AppManager interface {
	GetLogs(name string) ([]byte, error)
}

type Machine interface {
	GetPublicKey() []byte
	GetPublicIP() string
	GetName() string
}

type rpcMsgProcessor struct {
	WriteQueue chan rpcMsg
	Stop       context.CancelFunc
}

type rpcPeer struct {
	machine Machine
	client  *Client
}

type emptyReq struct{}
type emptyResp struct{}

type rpcHandler struct {
	Func          func(peer peer.ID, data interface{}) (interface{}, error)
	RequestStruct interface{}
}

type pubsubHandler struct {
	Func          func(peer peer.ID, data interface{}) error
	PayloadStruct interface{}
}

type pubsubMsg struct {
	ID      string
	Type    pubsubMsgType
	Payload json.RawMessage
}

type rpcMsg struct {
	ID      string
	Type    rpcMsgType
	Payload json.RawMessage
}

type rpcPayloadRequest struct {
	Type string
	Data json.RawMessage
}

type rpcPayloadResponse struct {
	Error string
	Data  json.RawMessage
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
	*ClientDB

	peer peer.ID
}

func (c *Client) GetCS() chunks.ChunkStore {
	return c.ChunkStore
}

type P2P struct {
	*ClientPubSub
	*ClientAppManager

	host             host.Host
	rpcHandlers      map[string]*rpcHandler
	pubsubHandlers   map[pubsubMsgType]*pubsubHandler
	reqs             cmap.ConcurrentMap
	rpcMsgProcessors cmap.ConcurrentMap
	peers            cmap.ConcurrentMap
	subscription     *pubsub.Subscription
	topic            *pubsub.Topic
	dbSyncer         DBSyncer
	appManager       AppManager
	initMode         bool
}

func (p2p *P2P) getRPCHandler(msgType string) (*rpcHandler, error) {
	if handler, found := p2p.rpcHandlers[msgType]; found {
		return handler, nil
	}
	return nil, fmt.Errorf("RPC handler for method '%s' not found", msgType)
}

func (p2p *P2P) addRPCHandler(methodName string, handler *rpcHandler) {
	p2p.rpcHandlers[methodName] = handler
}

func (p2p *P2P) getPubSubHandler(msgType pubsubMsgType) (*pubsubHandler, error) {
	if handler, found := p2p.pubsubHandlers[msgType]; found {
		return handler, nil
	}
	return nil, fmt.Errorf("PubSub handler for msg type '%s' not found", msgType)
}

func (p2p *P2P) addPubSubHandler(msgType pubsubMsgType, handler *pubsubHandler) {
	p2p.pubsubHandlers[msgType] = handler
}

func (p2p *P2P) newRPCStreamHandler(s network.Stream) {
	log.Debugf("Starting RPC stream processor for peer '%s'", s.Conn().RemotePeer().String())
	writeQueue := make(chan rpcMsg, 200)
	ctx, cancel := context.WithCancel(context.Background())
	p2p.rpcMsgProcessors.Set(s.Conn().RemotePeer().String(), &rpcMsgProcessor{WriteQueue: writeQueue, Stop: cancel})
	go p2p.rpcMsgReader(s, writeQueue, ctx)
	go p2p.rpcMsgWriter(s, writeQueue, ctx)
}

func (p2p *P2P) rpcMsgReader(s network.Stream, writeQueue chan rpcMsg, ctx context.Context) {
	// we process the request in a separate routine
	msgProcessor := func(msgBytes []byte) {
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("Exception whie processing incoming p2p RPC msg from '%s': %v", s.Conn().RemotePeer().String(), r)
			}
		}()

		msg := rpcMsg{}
		err := json.Unmarshal(msgBytes, &msg)
		if err != nil {
			log.Errorf("Failed to decode RPC message from '%s': %s", s.Conn().RemotePeer().String(), err.Error())
			return
		}

		if msg.Type == rpcRequest {
			// unmarshal remote request
			reqMsg := rpcPayloadRequest{}
			err = json.Unmarshal(msg.Payload, &reqMsg)
			if err != nil {
				log.Errorf("Failed to decode request from '%s': %s", s.Conn().RemotePeer().String(), err.Error())
				return
			}
			p2p.requestHandler(msg.ID, s.Conn().RemotePeer(), reqMsg, writeQueue)
		} else if msg.Type == rpcResponse {
			// unmarshal remote request
			respMsg := rpcPayloadResponse{}
			err = json.Unmarshal(msg.Payload, &respMsg)
			if err != nil {
				log.Errorf("Failed to decode response from '%s': %s", s.Conn().RemotePeer().String(), err.Error())
				return
			}
			p2p.responseHandler(msg.ID, s.Conn().RemotePeer(), respMsg)
		} else {
			log.Errorf("Wrong RPC message type from '%s': '%s'", s.Conn().RemotePeer().String(), msg.Type)
		}
	}

	readerChan := util.DelimReader(s, '\n')
	for {
		select {
		case bytes := <-readerChan:
			if len(bytes) == 0 {
				continue
			}
			go msgProcessor(bytes)
		case <-ctx.Done():
			log.Debugf("Stopping RPC msg reader for peer '%s'", s.Conn().RemotePeer().String())
			return
		}
	}
}

func (p2p *P2P) rpcMsgWriter(s network.Stream, writeQueue chan rpcMsg, ctx context.Context) {
	for {
		select {
		case msg := <-writeQueue:
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
		case <-ctx.Done():
			log.Debugf("Stopping RPC msg writer for peer '%s'", s.Conn().RemotePeer().String())
			return
		}

	}
}

func (p2p *P2P) requestHandler(id string, peerID peer.ID, request rpcPayloadRequest, writeQueue chan rpcMsg) {
	log.Tracef("Remote request '%s' from peer '%s': %v", id, peerID.String(), request)

	msg := rpcMsg{
		ID:   id,
		Type: rpcResponse,
	}

	response := rpcPayloadResponse{}

	// find handler
	handler, err := p2p.getRPCHandler(request.Type)
	if err != nil {
		log.Errorf("Failed to process request '%s' from '%s': %s", id, peerID.String(), err.Error())
		response.Error = err.Error()

		// encode the response
		jsonResp, err := json.Marshal(response)
		if err != nil {
			log.Errorf("Failed to encode response for request '%s' from '%s': %s", id, peerID.String(), err.Error())
			return
		}
		msg.Payload = jsonResp
		writeQueue <- msg
		return
	}

	// execute handler method
	data := reflect.New(reflect.ValueOf(handler.RequestStruct).Elem().Type()).Interface()
	err = json.Unmarshal(request.Data, &data)
	if err != nil {
		response.Error = fmt.Errorf("failed to decode data struct: %s", err.Error()).Error()

		// encode the response
		jsonResp, err := json.Marshal(response)
		if err != nil {
			log.Errorf("Failed to encode response for request '%s' from '%s': %s", id, peerID.String(), err.Error())
			return
		}

		msg.Payload = jsonResp
		writeQueue <- msg
		return
	}

	var jsonHandlerResponse []byte
	handlerResponse, err := handler.Func(peerID, data)
	if err != nil {
		log.Errorf("Failed to process request '%s' from '%s': %s", id, peerID.String(), err.Error())
	} else {
		// encode the returned handler response
		jsonHandlerResponse, err = json.Marshal(handlerResponse)
		if err != nil {
			log.Errorf("Failed to encode response for request '%s' from '%s': %s", id, peerID.String(), err.Error())
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
		log.Errorf("Failed to encode response for request '%s' from '%s': %s", id, peerID.String(), err.Error())
		return
	}
	msg.Payload = jsonResp
	log.Tracef("Sending response for msg '%s' to peer '%s': %v", id, peerID.String(), response)

	// send the response
	writeQueue <- msg
}

func (p2p *P2P) responseHandler(id string, peerID peer.ID, response rpcPayloadResponse) {
	log.Tracef("Received response '%s' from peer '%s': %v", id, peerID.String(), response)

	reqInteface, found := p2p.reqs.Get(id)
	if !found {
		log.Errorf("Failed to process response '%s' from '%s': request not found", id, peerID.String())
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
		req.err <- fmt.Errorf("error returned by '%s': %s", peerID.String(), response.Error)
	} else {
		req.resp <- response.Data
	}

	close(req.resp)
	close(req.err)
}

func (p2p *P2P) sendRequest(peerID peer.ID, msgType string, requestData interface{}, responseData interface{}) error {
	msg := rpcMsg{
		ID:   ksuid.New().String(),
		Type: rpcRequest,
	}

	// encode the request data
	jsonReqData, err := json.Marshal(requestData)
	if err != nil {
		return fmt.Errorf("failed to encode data for request '%s' for peer '%s': %s", msg.ID, peerID.String(), err.Error())
	}

	request := &rpcPayloadRequest{
		Type: msgType,
		Data: jsonReqData,
	}

	// encode the request
	jsonReq, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to encode request '%s' for peer '%s': %s", msg.ID, peerID.String(), err.Error())
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

	log.Tracef("Sending request '%s' to '%s': %s", msgType, peerID.String(), string(jsonReq))

	rpcMsgProcessorI, found := p2p.rpcMsgProcessors.Get(peerID.String())
	if !found {
		return fmt.Errorf("failed to send request '%s' for peer '%s': peer writer not found", msg.ID, peerID.String())
	}

	msgProcessor := rpcMsgProcessorI.(*rpcMsgProcessor)
	// send the request
	msgProcessor.WriteQueue <- msg

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

		reqTracker.err <- fmt.Errorf("timeout waiting for request '%s'(%s) to peer '%s'", msg.ID, request.Type, peerID.String())
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

func (p2p *P2P) pubsubMsgProcessor() func() error {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		log.Debug("Starting PubSub processor")
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

			go func(data []byte, peerID string) {
				defer func() {
					if r := recover(); r != nil {
						log.Errorf("Exception whie processing incoming p2p message from peer '%s': %v", peerID, r)
					}
				}()

				var pubsubMsg pubsubMsg
				err = json.Unmarshal(data, &pubsubMsg)
				if err != nil {
					log.Errorf("Failed to decode pub sub message from '%s': %v", peerID, err.Error())
					return
				}

				handler, err := p2p.getPubSubHandler(pubsubMsg.Type)
				if err != nil {
					log.Errorf("Failed to process message from '%s': %v", peerID, err.Error())
					return
				}

				payload := reflect.New(reflect.ValueOf(handler.PayloadStruct).Elem().Type()).Interface()
				err = json.Unmarshal(pubsubMsg.Payload, &payload)
				if err != nil {
					log.Errorf("Failed to process message from '%s': %v", peerID, err.Error())
					return
				}

				err = handler.Func(msg.ReceivedFrom, payload)
				if err != nil {
					log.Errorf("Failed calling pubsub handler for message '%s' from '%s': %v", pubsubMsg.Type, peerID, err.Error())
					return
				}

			}(msg.Data, peerID)
		}
	}()

	stopper := func() error {
		log.Debug("Stopping PubSub processor")
		cancel()
		return nil
	}
	return stopper
}

func (p2p *P2P) BroadcastMsg(msgType pubsubMsgType, data interface{}) error {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	msg := pubsubMsg{
		ID:      ksuid.New().String(),
		Type:    msgType,
		Payload: dataBytes,
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p2p.topic.Publish(context.Background(), msgBytes)
}

// GetPeerID adds a peer to the p2p manager
func (p2p *P2P) PubKeyToPeerID(pubKey []byte) (string, error) {
	pk, err := crypto.UnmarshalEd25519PublicKey(pubKey)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshall public key: %v", err)
	}
	peerID, err := peer.IDFromPublicKey(pk)
	if err != nil {
		return "", fmt.Errorf("failed to create peer ID from public key: %v", err)
	}
	return peerID.String(), nil
}

// AddPeer adds a peer to the p2p manager
func (p2p *P2P) AddPeer(machine Machine) (*Client, error) {
	pk, err := crypto.UnmarshalEd25519PublicKey(machine.GetPublicKey())
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshall public key: %v", err)
	}
	peerID, err := peer.IDFromPublicKey(pk)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer ID from public key: %v", err)
	}

	destinationString := ""
	if machine.GetPublicIP() != "" {
		destinationString = fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", machine.GetPublicIP(), p2pPort, peerID.String())
	} else {
		destinationString = fmt.Sprintf("/p2p/%s", peerID.String())
	}
	maddr, err := multiaddr.NewMultiaddr(destinationString)
	if err != nil {
		return nil, fmt.Errorf("failed to create multi address: %v", err)
	}

	peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return nil, fmt.Errorf("failed to extract info from address: %v", err)
	}
	rpcpeer := &rpcPeer{machine: machine}
	p2p.peers.Set(peerID.String(), rpcpeer)

	log.Debugf("Adding peer id '%s'(%s) at ip '%s'", machine.GetName(), peerInfo.ID.String(), machine.GetPublicIP())

	err = p2p.host.Connect(context.Background(), *peerInfo)
	if err != nil {
		log.Errorf("Failed to connect to peer '%s': %s", peerID.String(), err.Error())
	}

	client, err := p2p.createClientForPeer(peerID)
	if err != nil {
		return nil, fmt.Errorf("failed to add peer '%s': %v", peerID.String(), err)
	}
	rpcpeer.client = client

	return client, nil
}

// ConfigurePeers configures all the peers passed as arguemnt
func (p2p *P2P) ConfigurePeers(machines []Machine) error {
	currentPeers := map[string]peer.ID{}
	log.Debugf("Configuring p2p peers")

	// add new peers
	for _, machine := range machines {
		if len(machine.GetPublicKey()) == 0 {
			continue
		}

		pk, err := crypto.UnmarshalEd25519PublicKey(machine.GetPublicKey())
		if err != nil {
			log.Errorf("Failed to configure peer: %s", err.Error())
			continue
		}

		peerID, err := peer.IDFromPublicKey(pk)
		if err != nil {
			log.Errorf("Failed to configure peer: %s", err.Error())
			continue
		}

		if p2p.host.ID().String() == peerID.String() {
			continue
		}

		rpcpeerI, found := p2p.peers.Get(peerID.String())
		if !found {
			log.Debugf("Adding new peer '%s'(%s) at '%s'", machine.GetName(), peerID.String(), machine.GetPublicIP())
			destinationString := ""
			if machine.GetPublicIP() != "" {
				destinationString = fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", machine.GetPublicIP(), p2pPort, peerID.String())
			} else {
				destinationString = fmt.Sprintf("/p2p/%s", peerID.String())
			}
			maddr, err := multiaddr.NewMultiaddr(destinationString)
			if err != nil {
				log.Errorf("Failed to create multiaddress for peer '%s': %s", peerID.String(), err.Error())
				continue
			}

			peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
			if err != nil {
				log.Errorf("Failed parse peer info from multiaddress for peer '%s': %s", peerID.String(), err.Error())
				continue
			}

			rpcpeer := &rpcPeer{machine: machine}
			p2p.peers.Set(peerID.String(), rpcpeer)
			if machine.GetPublicIP() != "" {
				err = p2p.host.Connect(context.Background(), *peerInfo)
				if err != nil {
					log.Errorf("Failed to connect to peer '%s': %s", peerID.String(), err.Error())
				}
			}
		} else if rpcpeer := rpcpeerI.(*rpcPeer); rpcpeer.machine == nil {
			log.Infof("Replacing machine info for peer '%s'", machine.GetName())
			rpcpeer.machine = machine
		} else if rpcpeer := rpcpeerI.(*rpcPeer); rpcpeer.machine.GetName() == initMachineName {
			log.Infof("Replacing machine info for peer '%s' that triggerd initialisation", machine.GetName())
			rpcpeer.machine = machine
		}
		currentPeers[peerID.String()] = peerID
	}

	// delete old peers
	for rpcpeerItem := range p2p.peers.IterBuffered() {
		rpcpeer := rpcpeerItem.Val.(*rpcPeer)
		if _, found := currentPeers[rpcpeerItem.Key]; !found {
			name := "unknown"
			if rpcpeer.machine != nil {
				name = rpcpeer.machine.GetName()
			}
			if name == initMachineName {
				continue
			}
			log.Debugf("Removing old peer '%s'(%s)", name, rpcpeerItem.Key)
			p2p.peers.Remove(rpcpeerItem.Key)
			if rpcpeer.client != nil {
				err := p2p.host.Network().ClosePeer(rpcpeer.client.peer)
				if err != nil {
					log.Debugf("Failed to disconnect from old peer '%s'(%s)", rpcpeerItem.Key, name)
				}
			}
		}
	}

	return nil
}

func (p2p *P2P) GetCSClient(peerID string) (db.ChunkStoreClient, error) {
	rpcpeerI, found := p2p.peers.Get(peerID)
	if !found {
		return nil, fmt.Errorf("could not find peer '%s'", peerID)
	}
	rpcpeer := rpcpeerI.(*rpcPeer)
	if rpcpeer.client == nil {
		return nil, fmt.Errorf("could not find RPC client for peer '%s'", peerID)
	}

	return rpcpeer.client, nil
}

// getRPCPeer returns the rpc client for a peer
func (p2p *P2P) getRPCPeer(peerID peer.ID) (*rpcPeer, error) {
	rpcpeerI, found := p2p.peers.Get(peerID.String())
	rpcpeer := rpcpeerI.(*rpcPeer)
	if found {
		return rpcpeer, nil
	}
	return nil, fmt.Errorf("could not find RPC peer '%s'", peerID.String())
}

// createClientForPeer returns the remote client that can reach all remote handlers
func (p2p *P2P) createClientForPeer(peerID peer.ID) (client *Client, err error) {

	// err should be nil, and if there is a panic, we change it in the defer function
	// this is required because noms implements control flow using panics
	err = nil
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("exception whie building p2p client: %v", r)
			if strings.Contains(err.Error(), "timeout waiting for request") {
				err = fmt.Errorf("timeout waiting for p2p client")
			}
		}
	}()

	client = &Client{
		ClientInit: &ClientInit{p2p: p2p, peerID: peerID},
		ClientPing: &ClientPing{p2p: p2p, peerID: peerID},
		ClientDB:   &ClientDB{p2p: p2p, peerID: peerID},
		peer:       peerID,
	}

	tries := 0
	for {
		_, err = client.Ping()
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

	cls := NewRemoteChunkStore(p2p, peerID)
	client.ChunkStore = cls

	return client, nil
}

func (p2p *P2P) peerDiscoveryProcessor() func() error {
	ticker := time.NewTicker(10 * time.Second)
	stopSignal := make(chan struct{})
	go func() {
		log.Info("Starting peer discovery processor")
		for {
			select {
			case <-ticker.C:
				for rpcpeerItem := range p2p.peers.IterBuffered() {
					rpcpeer := rpcpeerItem.Val.(*rpcPeer)
					if rpcpeer.client == nil && rpcpeer.machine.GetPublicIP() != "" {
						pk, err := crypto.UnmarshalEd25519PublicKey(rpcpeer.machine.GetPublicKey())
						if err != nil {
							log.Errorf("Failed to configure peer: %s", err.Error())
							continue
						}

						peerID, err := peer.IDFromPublicKey(pk)
						if err != nil {
							log.Errorf("Failed to configure peer: %s", err.Error())
							continue
						}

						destinationString := fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", rpcpeer.machine.GetPublicIP(), p2pPort, peerID.String())
						maddr, err := multiaddr.NewMultiaddr(destinationString)
						if err != nil {
							log.Errorf("Failed to create multiaddress for peer '%s': %s", peerID.String(), err.Error())
							continue
						}

						peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
						if err != nil {
							log.Errorf("Failed parse peer info from multiaddress for peer '%s': %s", peerID.String(), err.Error())
							continue
						}

						err = p2p.host.Connect(context.Background(), *peerInfo)
						if err != nil {
							continue
						}
					}
				}
			case <-stopSignal:
				log.Debugf("Stopping peer discovery processor")
			}
		}
	}()
	stopper := func() error {
		stopSignal <- struct{}{}
		return nil
	}
	return stopper
}

func (p2p *P2P) newConnectionHandler(netw network.Network, conn network.Conn) {
	go func() {
		if conn.Stat().Transient {
			return
		}

		var rpcpeer *rpcPeer
		if p2p.initMode {
			rpcpeer = &rpcPeer{machine: &initMachine{name: initMachineName}}
		} else {
			rpcpeerI, found := p2p.peers.Get(conn.RemotePeer().String())
			if !found {
				log.Errorf("Peer '%s' not found locally while creating client", conn.RemotePeer().String())
				conn.Close()
				return
			}
			rpcpeer = rpcpeerI.(*rpcPeer)
		}

		log.Debugf("New connection with peer '%s'(%s). Creating client", rpcpeer.machine.GetName(), conn.RemotePeer().String())
		if conn.Stat().Direction == network.DirOutbound {
			s, err := p2p.host.NewStream(context.Background(), conn.RemotePeer(), protocol.ID(protosRPCProtocol))
			if err != nil {
				log.Errorf("Could not create stream to new peer '%s'(%s): %s", rpcpeer.machine.GetName(), conn.RemotePeer().String(), err.Error())
				conn.Close()
				return
			}
			p2p.newRPCStreamHandler(s)
		}

		tries := 0
		for {
			streams := conn.GetStreams()
			if len(streams) != 0 {
				break
			}

			time.Sleep(500 * time.Millisecond)
			if tries == 19 {
				log.Errorf("Time out. Could not find stream for new peer '%s'(%s)", rpcpeer.machine.GetName(), conn.RemotePeer().String())
				conn.Close()
				return
			}
			tries++
		}

		tries = 0
		for {
			_, found := p2p.rpcMsgProcessors.Get(conn.RemotePeer().String())
			if found {
				break
			}

			time.Sleep(100 * time.Millisecond)
			if tries == 19 {
				log.Errorf("Time out. Could not find writer for new peer '%s'(%s)", rpcpeer.machine.GetName(), conn.RemotePeer().String())
				conn.Close()
				return
			}
			tries++
		}

		rpcClient, err := p2p.createClientForPeer(conn.RemotePeer())
		if err != nil {
			log.Errorf("Failed to create client for new peer '%s'(%s): %s", rpcpeer.machine.GetName(), conn.RemotePeer().String(), err.Error())
			conn.Close()
			return
		}
		rpcpeer.client = rpcClient
	}()
}

func (p2p *P2P) closeConnectionHandler(netw network.Network, conn network.Conn) {
	if conn.Stat().Transient {
		return
	}

	log.Debugf("Removing client. Connection closed with peer '%s'.", conn.RemotePeer().String())

	p2p.host.Peerstore().ClearAddrs(conn.RemotePeer())
	rpcpeerI, found := p2p.peers.Get(conn.RemotePeer().String())
	if found {
		rpcpeer := rpcpeerI.(*rpcPeer)
		rpcpeer.client = nil
	}

	rpcMsgProcessorI, found := p2p.rpcMsgProcessors.Get(conn.RemotePeer().String())
	if !found {
		log.Errorf("RPC msg processor for peer '%s' not found locally while cleaning up client", conn.RemotePeer().String())
	} else {
		msgProcessor := rpcMsgProcessorI.(*rpcMsgProcessor)
		msgProcessor.Stop()
	}

}

// StartServer starts listening for p2p connections
func (p2p *P2P) StartServer(metaConfigurator MetaConfigurator, cs chunks.ChunkStore) (func() error, error) {
	log.Info("Starting p2p server")

	p2pPing := &HandlersPing{}
	p2pInit := &HandlersInit{p2p: p2p, metaConfigurator: metaConfigurator}
	p2pDB := &HandlersDB{dbSyncer: p2p.dbSyncer}
	p2pChunkStore := &HandlersChunkStore{p2p: p2p, cs: cs}

	p2pPubSub := &pubSub{p2p: p2p, dbSyncer: p2p.dbSyncer}

	// register RPC handler methods which should be accessible from the client
	// ping handler
	p2p.addRPCHandler(pingHandler, &rpcHandler{Func: p2pPing.HandlerPing, RequestStruct: &PingReq{}})
	// init handler
	p2p.addRPCHandler(initHandler, &rpcHandler{Func: p2pInit.HandlerInit, RequestStruct: &InitReq{}})
	// db handlers
	p2p.addRPCHandler(sendDatasetsHeadsHandler, &rpcHandler{Func: p2pDB.SendDatasetsHeadsHandler, RequestStruct: &SendDatasetsHeadsReq{}})
	// db sync handlers
	p2p.addRPCHandler(getRootHandler, &rpcHandler{Func: p2pChunkStore.getRoot, RequestStruct: &emptyReq{}})
	p2p.addRPCHandler(setRootHandler, &rpcHandler{Func: p2pChunkStore.setRoot, RequestStruct: &setRootReq{}})
	p2p.addRPCHandler(writeValueHandler, &rpcHandler{Func: p2pChunkStore.writeValue, RequestStruct: &writeValueReq{}})
	p2p.addRPCHandler(getStatsSummaryHandler, &rpcHandler{Func: p2pChunkStore.getStatsSummary, RequestStruct: &emptyReq{}})
	p2p.addRPCHandler(getRefsHandler, &rpcHandler{Func: p2pChunkStore.getRefs, RequestStruct: &getRefsReq{}})
	p2p.addRPCHandler(hasRefsHandler, &rpcHandler{Func: p2pChunkStore.hasRefs, RequestStruct: &hasRefsReq{}})

	// register PubSub hanlder methods
	p2p.addPubSubHandler(pubsubBroadcastHead, &pubsubHandler{Func: p2pPubSub.BroadcastHeadHandler, PayloadStruct: &pubsubPayloadBroadcastHead{}})
	p2p.addPubSubHandler(pubsubRequestHead, &pubsubHandler{Func: p2pPubSub.BroadcastRequestHeadHandler, PayloadStruct: &emptyReq{}})

	err := p2p.host.Network().Listen()
	if err != nil {
		return func() error { return nil }, fmt.Errorf("failed to listen: %v", err)
	}

	pubsubStopper := p2p.pubsubMsgProcessor()
	peerDiscoveryStopper := p2p.peerDiscoveryProcessor()

	stopper := func() error {
		log.Debug("Stopping p2p server")
		pubsubStopper()
		peerDiscoveryStopper()
		return p2p.host.Close()
	}
	return stopper, nil

}

// NewManager creates and returns a new p2p manager
func NewManager(key *pcrypto.Key, dbSyncer DBSyncer, appManager AppManager, initMode bool) (*P2P, error) {
	p2p := &P2P{
		rpcHandlers:      map[string]*rpcHandler{},
		pubsubHandlers:   map[pubsubMsgType]*pubsubHandler{},
		reqs:             cmap.New(),
		rpcMsgProcessors: cmap.New(),
		peers:            cmap.New(),
		dbSyncer:         dbSyncer,
		appManager:       appManager,
		initMode:         initMode,
	}

	p2p.ClientPubSub = &ClientPubSub{p2p: p2p}

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
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", p2pPort),
			fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", p2pPort),
		),
		libp2p.Security(noise.ID, noise.New),
		libp2p.DefaultTransports,
		libp2p.ConnectionManager(con),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to setup p2p host: %v", err)
	}

	p2p.host = host
	p2p.host.SetStreamHandler(protocol.ID(protosRPCProtocol), p2p.newRPCStreamHandler)
	pubSub, err := pubsub.NewFloodSub(context.Background(), host)
	if err != nil {
		return nil, fmt.Errorf("failed to setup PubSub channel: %v", err)
	}

	nb := network.NotifyBundle{
		ConnectedF:    p2p.newConnectionHandler,
		DisconnectedF: p2p.closeConnectionHandler,
	}
	p2p.host.Network().Notify(&nb)

	p2p.topic, err = pubSub.Join(protosUpdatesTopic)
	if err != nil {
		return nil, fmt.Errorf("failed to join PubSub topic '%s': %v", protosUpdatesTopic, err)
	}

	p2p.subscription, err = p2p.topic.Subscribe()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to PubSub topic '%s': %v", protosUpdatesTopic, err)
	}

	log.Debugf("Using host with ID '%s'", host.ID().String())
	return p2p, nil
}
