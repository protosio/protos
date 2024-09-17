package apic

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

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
	protosClient *protosc.ProtosClient
}

func errorLoggingUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	resp, err := handler(ctx, req)
	if err != nil {
		log.Errorf("method %s: %v", info.FullMethod, err)
	}
	return resp, err
}

func StartGRPCServer(dataPath string, version string, protosClient *protosc.ProtosClient) (func() error, error) {

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
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				grpc.UnaryServerInterceptor(errorLoggingUnaryInterceptor),
				grpc_recovery.UnaryServerInterceptor(recoveryOpt),
			),
		),
	)
	pbApic.RegisterProtosClientApiServer(srv, &Backend{
		protosClient: protosClient,
	})

	log.Info("starting gRPC server at unix://", unixSocketFile)
	go func() {
		if err := srv.Serve(l); err != nil {
			log.Fatalf("Failed to serve gRPC service: %s", err.Error())
		}
	}()

	stopper := func() error {
		log.Info("stopping gRPC server")
		srv.GracefulStop()
		if protosClient.NetworkManager != nil {
			log.Info("bringing down network")
			err = protosClient.NetworkManager.Down()
			if err != nil {
				log.Error(err)
			}
		}
		return nil
	}
	return stopper, nil
}
