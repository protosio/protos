package apic

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/protosio/protos/internal/app"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/network"
	"github.com/protosio/protos/internal/p2p"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/provisioners"
	"github.com/protosio/protos/internal/release"
	"github.com/protosio/protos/internal/user"
	"github.com/protosio/protos/internal/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var log = util.GetLogger("grpcAPI")

type Backend struct {
	protosClient *Services
}

func NewBackend(protosClient *Services) pbApic.ProtosClientApiServer {
	return &Backend{protosClient: protosClient}
}

type Services struct {
	DB             *db.DB
	Manager        *user.Manager
	KeyManager     *pcrypto.Manager
	AppManager     *app.Manager
	NetworkManager *network.Manager
	CloudManager   *provisioners.Manager
	P2PManager     *p2p.P2P
	CanProvision   bool

	InitFunc     func(username string, name string, organization string) error
	ReleaseFetch func() (release.Releases, error)
}

func (s *Services) Init(username string, name string, organization string) error {
	if s.InitFunc == nil {
		return fmt.Errorf("init is not available")
	}
	return s.InitFunc(username, name, organization)
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
	}
	return resp, err
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

	unixSocketFile := dataPath + "/protos.socket"
	l, err := net.Listen("unix", unixSocketFile)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on local socket: %w", err)
	}

	srv := NewGRPCServer(protosClient)

	log.Info("starting gRPC server at unix://", unixSocketFile)
	go func() {
		if err := srv.Serve(l); err != nil {
			log.Fatalf("Failed to serve gRPC service: %s", err.Error())
		}
	}()

	stopper := func() error {
		log.Info("stopping gRPC server")
		srv.GracefulStop()
		return nil
	}
	return stopper, nil
}
