package apic

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	pbApic "github.com/protosio/protos/apic/proto"
	hostagentipc "github.com/protosio/protos/internal/hostagent/ipc"
	hostagentpb "github.com/protosio/protos/internal/hostagent/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const hostAgentStatusTimeout = 900 * time.Millisecond

func (b *Backend) GetSystemStatus(ctx context.Context, _ *pbApic.GetSystemStatusRequest) (*pbApic.GetSystemStatusResponse, error) {
	services := b.protosClient
	status := &pbApic.SystemStatus{
		CoreStatus:   "running",
		WorkDir:      services.WorkDir,
		Capabilities: services.Capabilities,
		P2PPort:      int32(services.P2PPort),
	}
	if shouldReportHostAgent(runtime.GOOS) {
		status.HostAgent = hostAgentConnectionStatus(ctx)
	}

	if services.WorkDir != "" {
		socket := filepath.Join(services.WorkDir, "protos.socket")
		status.Endpoints = append(status.Endpoints, socketEndpoint("api-socket", socket))
	}

	status.Endpoints = append(status.Endpoints, &pbApic.CoreEndpoint{
		Kind:    "embedded-api",
		Address: "in-process",
		Active:  true,
		Message: "Flutter bridge",
	})

	if services.P2PPort > 0 {
		status.Endpoints = append(status.Endpoints, &pbApic.CoreEndpoint{
			Kind:    "p2p-port",
			Address: fmt.Sprintf("udp/%d", services.P2PPort),
			Active:  services.P2PManager != nil,
		})
	}
	if services.P2PManager != nil {
		peerID := services.P2PManager.PeerID()
		for _, addr := range services.P2PManager.ListenAddresses() {
			endpoint := &pbApic.CoreEndpoint{
				Kind:    "p2p-listen",
				Address: addr,
				Active:  true,
			}
			if peerID != "" {
				endpoint.Message = "peer " + peerID
			}
			status.Endpoints = append(status.Endpoints, endpoint)
		}
	}

	return &pbApic.GetSystemStatusResponse{Status: status}, nil
}

func shouldReportHostAgent(goos string) bool {
	return goos != "ios"
}

func socketEndpoint(kind string, path string) *pbApic.CoreEndpoint {
	endpoint := &pbApic.CoreEndpoint{
		Kind:    kind,
		Address: path,
	}
	info, err := os.Stat(path)
	if err != nil {
		endpoint.Message = err.Error()
		return endpoint
	}
	if info.Mode()&os.ModeSocket == 0 {
		endpoint.Message = "path exists but is not a socket"
		return endpoint
	}
	endpoint.Active = true
	endpoint.Message = "listening"
	return endpoint
}

func hostAgentConnectionStatus(ctx context.Context) *pbApic.HostAgentConnectionStatus {
	socket := hostagentipc.SocketPath()
	status := &pbApic.HostAgentConnectionStatus{Socket: socket}
	if _, err := os.Stat(socket); err != nil {
		status.Message = err.Error()
		return status
	}

	rpcCtx, cancel := context.WithTimeout(ctx, hostAgentStatusTimeout)
	defer cancel()

	conn, err := grpc.NewClient("unix://"+socket, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		status.Message = err.Error()
		return status
	}
	defer conn.Close()

	if _, err := hostagentpb.NewHostAgentClient(conn).Status(rpcCtx, &hostagentpb.StatusRequest{}); err != nil {
		status.Message = err.Error()
		return status
	}
	status.Connected = true
	status.Message = "reachable"
	return status
}
