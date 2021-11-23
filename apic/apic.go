package apic

import (
	"fmt"
	"net"
	"os"

	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/protosio/protos/internal/protosc"
	"github.com/protosio/protos/internal/util"
	"google.golang.org/grpc"
)

var log = util.GetLogger("grpcAPI")

type Backend struct {
	pbApic.UnimplementedProtosClientApiServer
	protosClient *protosc.ProtosClient
}

func StartGRPCServer(socketPath string, dataPath string, version string) (func() error, error) {

	protosClient, err := protosc.New(dataPath, version)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Protos client: %w", err)
	}

	// create protos run dir
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		err := os.Mkdir(socketPath, 0755)
		if err != nil {
			return nil, fmt.Errorf("Failed to create protos dir '%s': %w", socketPath, err)
		}
	}

	unixSocketFile := socketPath + "/protos.socket"
	l, err := net.Listen("unix", unixSocketFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to listen on local socket: %w", err)
	}

	srv := grpc.NewServer()
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
		return nil
	}
	return stopper, nil
}
