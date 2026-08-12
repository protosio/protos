package apic

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/protosio/protos/internal/app"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/invitations"
	"github.com/protosio/protos/internal/network"
	networkmodule "github.com/protosio/protos/internal/network/module"
	"github.com/protosio/protos/internal/p2p"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/provisioners"
	"github.com/protosio/protos/internal/release"
	"github.com/protosio/protos/internal/tasks"
	"github.com/protosio/protos/internal/user"
	"github.com/protosio/protos/internal/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var log = util.GetLogger("grpcAPI")

type Backend struct {
	protosClient     *Services
	commitIdentities *commitIdentityResolver
}

func NewBackend(protosClient *Services) pbApic.ProtosClientApiServer {
	backend := &Backend{
		protosClient:     protosClient,
		commitIdentities: newCommitIdentityResolver(protosClient),
	}
	backend.registerTaskStreamsIfConfigured()
	return backend
}

type Services struct {
	DB             *db.DB
	Manager        *user.Manager
	KeyManager     *pcrypto.Manager
	AppManager     *app.Manager
	NetworkManager *network.Manager
	NetworkControl NetworkController
	CloudManager   *provisioners.Manager
	P2PManager     *p2p.P2P
	TaskManager    *tasks.Manager
	Invites        *invitations.Manager
	CanProvision   bool
	WorkDir        string
	Capabilities   string
	P2PPort        int

	InitFunc        func(username string, name string, organisation string) error
	MarkInitialized func()
	ReleaseFetch    func() (release.Releases, error)
}

type NetworkController interface {
	EnableNetwork(context.Context) error
	DisableNetwork(context.Context) error
	NetworkRuntimeStatus(context.Context) NetworkRuntimeStatus
	NetworkState(context.Context) (networkmodule.State, error)
}

type NetworkRuntimeStatus struct {
	Supported      bool
	DesiredEnabled bool
	Enabled        bool
	State          string
	Message        string
	NetworkState   *networkmodule.State
}

func (s *Services) Init(username string, name string, organisation string) error {
	if s.InitFunc == nil {
		return fmt.Errorf("init is not available")
	}
	return s.InitFunc(username, name, organisation)
}

func (s *Services) MarkInitializedIfNeeded() {
	if s.MarkInitialized != nil {
		s.MarkInitialized()
	}
}

func (s *Services) GetProtosAvailableReleases() (release.Releases, error) {
	if s.ReleaseFetch == nil {
		return release.Releases{}, fmt.Errorf("release lookup is not available")
	}
	return s.ReleaseFetch()
}

func errorLoggingUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	resp, err := handler(ctx, req)
	if err != nil {
		log.Errorf("method %s: %v", info.FullMethod, err)
		err = publishedWriteOutcomeStatus(err)
	}
	return resp, err
}

// publishedWriteOutcomeStatus preserves the exact tracking identity for an
// ordinary mutation whose publisher returned an error after allocating an
// event/root receipt. The empty stage is intentional: the receipt is an
// observation address, not proof that local acceptance completed. A
// FailedPrecondition status avoids presenting the outcome as a transient RPC
// failure that is safe for clients to replay automatically.
func publishedWriteOutcomeStatus(err error) error {
	confirmation, ok := db.PublishedWriteConfirmationFromError(err)
	if !ok {
		return err
	}
	detail := &pbApic.WriteConfirmation{
		EventId:           confirmation.Receipt.EventID,
		PublishedRootHash: confirmation.Receipt.PublishedRootHash,
	}
	st := status.New(
		codes.FailedPrecondition,
		"write outcome unresolved; observe the exact receipt before retrying",
	)
	withDetails, detailErr := st.WithDetails(detail)
	if detailErr != nil {
		log.Errorf("attach unresolved write receipt: %v", detailErr)
		return st.Err()
	}
	return withDetails.Err()
}

func recoveryUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if p := recover(); p != nil {
			log.Errorf("[PANIC] %s\n----------------\n%s----------------", p, string(debug.Stack()))
			err = status.Error(codes.Internal, "Internal error. Please check client logs")
		}
	}()
	return handler(ctx, req)
}

func recoveryStreamInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	defer func() {
		if p := recover(); p != nil {
			log.Errorf("[PANIC] %s\n----------------\n%s----------------", p, string(debug.Stack()))
			err = status.Error(codes.Internal, "Internal error. Please check client logs")
		}
	}()
	return handler(srv, stream)
}

func NewGRPCServer(protosClient *Services) *grpc.Server {
	srv := grpc.NewServer(
		grpc.ChainStreamInterceptor(recoveryStreamInterceptor),
		grpc.ChainUnaryInterceptor(errorLoggingUnaryInterceptor, recoveryUnaryInterceptor),
	)
	pbApic.RegisterProtosClientApiServer(srv, NewBackend(protosClient))
	return srv
}

func StartGRPCServer(dataPath string, version string, protosClient *Services) (func() error, error) {

	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve home directory: %w", err)
	}

	if dataPath == "~" {
		dataPath = homedir
	} else if strings.HasPrefix(dataPath, "~/") {
		dataPath = filepath.Join(homedir, dataPath[2:])
	}

	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create API socket directory: %w", err)
	}

	unixSocketFile := filepath.Join(dataPath, "protos.socket")
	if err := os.Remove(unixSocketFile); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to remove stale local socket: %w", err)
	}
	l, err := net.Listen("unix", unixSocketFile)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on local socket: %w", err)
	}

	srv := NewGRPCServer(protosClient)

	log.Info("starting gRPC server at unix://", unixSocketFile)
	go func() {
		if err := srv.Serve(l); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Errorf("Failed to serve gRPC service: %s", err.Error())
		}
	}()

	stopper := func() error {
		log.Info("stopping gRPC server")
		srv.GracefulStop()
		_ = os.Remove(unixSocketFile)
		return nil
	}
	return stopper, nil
}
