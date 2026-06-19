package p2p

import (
	"context"
	"runtime/debug"

	p2pgrpc "github.com/birros/go-libp2p-grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newP2PGRPCServer() *grpc.Server {
	return grpc.NewServer(
		p2pgrpc.WithP2PCredentials(),
		grpc.ChainUnaryInterceptor(p2pErrorLoggingUnaryInterceptor, p2pRecoveryUnaryInterceptor),
		grpc.ChainStreamInterceptor(p2pRecoveryStreamInterceptor),
	)
}

func p2pErrorLoggingUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	resp, err := handler(ctx, req)
	if err != nil {
		log.Errorf("p2p method %s: %v", info.FullMethod, err)
	}
	return resp, err
}

func p2pRecoveryUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if p := recover(); p != nil {
			log.Errorf("[P2P PANIC] %s\n----------------\n%s----------------", p, string(debug.Stack()))
			err = status.Error(codes.Internal, "internal p2p error")
		}
	}()
	return handler(ctx, req)
}

func p2pRecoveryStreamInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	defer func() {
		if p := recover(); p != nil {
			log.Errorf("[P2P PANIC] %s\n----------------\n%s----------------", p, string(debug.Stack()))
			err = status.Error(codes.Internal, "internal p2p error")
		}
	}()
	return handler(srv, stream)
}
