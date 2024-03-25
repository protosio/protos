package p2p

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"

	p2pgrpc "github.com/birros/go-libp2p-grpc"
	"github.com/go-playground/validator/v10"
	"github.com/nustiueudinastea/doltswarm"
	"github.com/protosio/protos/internal/p2p/proto"
	"google.golang.org/grpc"
)

var _ proto.PingerServer = (*Server)(nil)
var _ proto.TesterServer = (*Server)(nil)

type ExternalDB interface {
	AddPeer(peerID string, conn *grpc.ClientConn) error
	RemovePeer(peerID string) error
	GetAllCommits() ([]doltswarm.Commit, error)
	ExecAndCommit(query string, commitMsg string) (string, error)
	GetLastCommit(branch string) (doltswarm.Commit, error)
}

// MetaConfigurator allows for the configuration of the meta package
type MetaConfigurator interface {
	SetNetwork(network net.IPNet) net.IP
	SetInstanceName(name string)
}

type initMachine struct {
	name      string
	publicIP  string
	publicKey string
}

func (im *initMachine) GetPublicKey() string {
	return im.publicKey
}
func (im *initMachine) GetPublicIP() string {
	return im.publicIP
}
func (im *initMachine) GetName() string {
	return im.name
}

type Server struct {
	DB               ExternalDB
	p2p              *P2P
	metaConfigurator MetaConfigurator
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
	commit, err := s.DB.ExecAndCommit(req.Statement, req.Msg)
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
		res.Commits = append(res.Commits, commit.Hash)
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

	for id, rpcpeer := range s.p2p.peers.Snapshot() {
		client := rpcpeer.GetClient()
		machine := rpcpeer.GetMachine()
		peerName := fmt.Sprintf("unknown(%s)", id)
		peerStatus := "disconnected"
		if machine != nil {
			peerName = fmt.Sprintf("%s(%s)", machine.GetName(), id)
		}
		if client != nil {
			peerStatus = "connected"
		}
		peers[peerName] = peerStatus
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

	_, network, err := net.ParseCIDR(req.Network)
	if err != nil {
		return nil, fmt.Errorf("cannot perform initialization, network '%s' is invalid: %w", req.Network, err)
	}

	im := &initMachine{
		name:      initMachineName,
		publicKey: req.OriginDevicePublicKey,
	}

	s.p2p.initMode = false
	_, err = s.p2p.AddPeer(im)
	if err != nil {
		return nil, fmt.Errorf("failed to add init device as rpc client: %w", err)
	}

	s.metaConfigurator.SetInstanceName(req.InstanceName)
	ipNet := s.metaConfigurator.SetNetwork(*network)

	return &proto.InitResponse{InstanceIp: ipNet.String(), Architecture: runtime.GOARCH}, nil
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
