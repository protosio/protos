package apic

import (
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/protosio/protos/internal/util"
	"google.golang.org/grpc"
)

var log = util.GetLogger("grpcAPI")

type Backend struct {
	pbApic.UnimplementedAppServiceServer
	mu   *sync.RWMutex
	apps []*pbApic.App
}

func (b *Backend) GetApps(_ *empty.Empty, srv pbApic.AppService_GetAppsServer) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, app := range b.apps {
		err := srv.Send(app)
		if err != nil {
			return err
		}
	}

	return nil
}

var _ pbApic.AppServiceServer = (*Backend)(nil)

func StartGRPCServer(socketPath string) (func() error, error) {

	// create protos run dir
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		err := os.Mkdir(socketPath, 0755)
		if err != nil {
			return nil, fmt.Errorf("Failed to create protos dir '%s': %w", socketPath, err)
		}
	}

	l, err := net.Listen("unix", socketPath+"/protos.socket")
	if err != nil {
		return nil, fmt.Errorf("Failed to listen on local socket: %w", err)
	}

	srv := grpc.NewServer()
	pbApic.RegisterAppServiceServer(srv, &Backend{
		mu: &sync.RWMutex{},
	})

	log.Info("Starting gRPC server at unix://", socketPath)
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
