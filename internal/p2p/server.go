package p2p

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	p2pgrpc "github.com/birros/go-libp2p-grpc"
	"github.com/go-playground/validator/v10"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/p2p/proto"
	"github.com/protosio/protos/internal/pcrypto"
	"google.golang.org/grpc"
)

var _ proto.PingerServer = (*Server)(nil)
var _ proto.PeerDBServer = (*Server)(nil)
var _ proto.AppsServer = (*Server)(nil)
var _ proto.InstanceServer = (*Server)(nil)

type ExternalDB interface {
	AddPeer(peerID string, conn *grpc.ClientConn) error
	RemovePeer(peerID string) error
	GetAllCommits() ([]db.Commit, error)
	ExecSQLAndCommit(statement string, commitMsg string) (string, error)
	GetLastCommit(branch string) (db.Commit, error)
	InitFromPeer(peerID string, bootstrapPeers []string) error
	EnableGRPCServers(server *grpc.Server) error
	Initialized() bool
}

type initMachine struct {
	id        string
	publicIP  string
	publicKey string
}

func (im *initMachine) GetID() string {
	return im.id
}
func (im *initMachine) GetPublicKey() string {
	return im.publicKey
}
func (im *initMachine) GetPublicIP() string {
	return im.publicIP
}
func (im *initMachine) GetName() string {
	return im.id
}

type Server struct {
	DB  ExternalDB
	p2p *P2P
}

func (s *Server) Ping(ctx context.Context, req *proto.PingRequest) (*proto.PingResponse, error) {
	_, ok := p2pgrpc.RemotePeerFromContext(ctx)
	if !ok {
		return nil, errors.New("no AuthInfo in context")
	}

	res := &proto.PingResponse{
		Pong: "Ping: " + req.Ping + "!",
	}
	return res, nil
}

func (s *Server) ExecSQL(ctx context.Context, req *proto.ExecSQLRequest) (*proto.ExecSQLResponse, error) {
	commit, err := s.DB.ExecSQLAndCommit(req.Statement, req.Msg)
	if err != nil {
		return nil, err
	}
	return &proto.ExecSQLResponse{Result: "", Commit: commit}, nil
}

func (s *Server) GetAllCommits(context.Context, *proto.GetAllCommitsRequest) (*proto.GetAllCommitsResponse, error) {
	commits, err := s.DB.GetAllCommits()
	if err != nil {
		return nil, err
	}

	res := &proto.GetAllCommitsResponse{}
	for _, commit := range commits {
		respCommit := proto.Commit{
			Hash:      commit.Hash,
			Committer: commit.Committer,
			Message:   commit.Message,
		}
		res.Commits = append(res.Commits, &respCommit)
	}

	return res, nil
}

func (s *Server) GetHead(context.Context, *proto.GetHeadRequest) (*proto.GetHeadResponse, error) {
	commit, err := s.DB.GetLastCommit("main")
	if err != nil {
		return nil, err
	}
	return &proto.GetHeadResponse{Commit: commit.Hash}, nil
}

// HandlerGetInstanceLogs retrieves logs for the local instance
func (s *Server) GetLogs(context.Context, *proto.GetLogsRequest) (*proto.GetLogsResponse, error) {

	logs, err := os.ReadFile("/var/log/protos.log")
	if err != nil {
		return nil, fmt.Errorf("failed to read protos logs: %w", err)
	}

	encodedLogs := base64.StdEncoding.EncodeToString(logs)

	return &proto.GetLogsResponse{Logs: encodedLogs}, nil
}

// HandlerGetInstancePeers retrieves the peers for the local instance
func (s *Server) GetPeers(context.Context, *proto.GetPeersRequest) (*proto.GetPeersResponse, error) {

	peers := map[string]string{}
	for id, machine := range s.p2p.machines.Snapshot() {
		peerStatus := string(PeerStatusDesired)
		if state, found := s.p2p.peerStates.Get(id); found && state.Status != "" {
			peerStatus = string(state.Status)
		}
		if client, found := s.p2p.clients.Get(id); found && client != nil {
			peerStatus = string(PeerStatusConnected)
		}
		if machine.GetName() != "" {
			peers[machine.GetName()] = peerStatus
			continue
		}
		peers[id] = peerStatus
	}

	return &proto.GetPeersResponse{Peers: peers}, nil
}

// HandlerInit does the initialisation on the server side
func (s *Server) Init(ctx context.Context, req *proto.InitRequest) (*proto.InitResponse, error) {

	validate := validator.New()
	err := validate.Struct(req)
	if err != nil {
		return nil, fmt.Errorf("failed to validate init request: %w", err)
	}

	pubKey, err := pcrypto.CreatePublicKeyFromBase64(req.OriginDevicePublicKey)
	if err != nil {
		return nil, fmt.Errorf("cannot perform initialization: %w", err)
	}

	im := initMachine{
		id:        pubKey.GetID(),
		publicKey: req.OriginDevicePublicKey,
	}

	_, err = s.p2p.AddPeer(&im)
	if err != nil {
		return nil, fmt.Errorf("failed to add init device as rpc client: %w", err)
	}

	err = s.DB.InitFromPeer(im.GetID(), req.OriginSwarmionAddrs)
	if err != nil {
		err = fmt.Errorf("failed to initialize database from peer %s(%s): %w", im.GetID(), pubKey.GetID(), err)
		log.Error(err.Error())
		return nil, err
	}

	go func() {
		time.Sleep(500 * time.Millisecond)
		s.p2p.restartServerSignal <- im
		close(s.p2p.restartServerSignal)
	}()

	return &proto.InitResponse{Architecture: runtime.GOARCH}, nil
}

func (s *Server) GetAppLogs(ctx context.Context, req *proto.GetAppLogsRequest) (*proto.GetAppLogsResponse, error) {

	logs, err := s.p2p.appManager.GetLogs(req.AppName)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve logs for app '%s': %w", req.AppName, err)
	}

	encodedLogs := base64.StdEncoding.EncodeToString(logs)
	return &proto.GetAppLogsResponse{Logs: encodedLogs}, nil
}

func (s *Server) GetAppStatus(ctx context.Context, req *proto.GetAppStatusRequest) (*proto.GetAppStatusResponse, error) {

	status, err := s.p2p.appManager.GetStatus(req.AppName)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve status for app '%s': %w", req.AppName, err)
	}

	return &proto.GetAppStatusResponse{Status: status}, nil
}
