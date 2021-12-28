package apic

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime/debug"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/protosio/protos/internal/protosc"
	"github.com/protosio/protos/internal/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var log = util.GetLogger("grpcAPI")

type Backend struct {
	pbApic.UnimplementedProtosClientApiServer
	protosClient *protosc.ProtosClient
}

func StartGRPCServer(socketPath string, dataPath string, version string) (func() error, error) {

	protosClient, err := protosc.New(dataPath, version)
	if err != nil {
		return nil, fmt.Errorf("failed to create Protos client: %w", err)
	}

	// create protos run dir
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		err := os.Mkdir(socketPath, 0755)
		if err != nil {
			return nil, fmt.Errorf("failed to create protos dir '%s': %w", socketPath, err)
		}
	}

	unixSocketFile := socketPath + "/protos.socket"
	l, err := net.Listen("unix", unixSocketFile)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on local socket: %w", err)
	}

	recoveryOpt := grpc_recovery.WithRecoveryHandlerContext(
		func(ctx context.Context, p interface{}) error {
			log.Errorf("[PANIC] %s\n----------------\n%s----------------", p, string(debug.Stack()))
			return status.Error(codes.Internal, "Internal error. Please check client logs")
		},
	)

	srv := grpc.NewServer(
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_recovery.StreamServerInterceptor(recoveryOpt),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_recovery.UnaryServerInterceptor(recoveryOpt),
		)),
	)
	pbApic.RegisterProtosClientApiServer(srv, &Backend{
		protosClient: protosClient,
	})

	log.Info("Starting gRPC server at unix://", unixSocketFile)
	go func() {
		if err := srv.Serve(l); err != nil {
			log.Fatalf("Failed to serve gRPC service: %w", err)
		}
	}()

	stopper := func() error {
		log.Info("Stopping gRPC server")
		srv.GracefulStop()
		err = protosClient.NetworkManager.Down()
		if err != nil {
			log.Error(err)
		}
		return nil
	}
	return stopper, nil
}
