package apic

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	pbApic "github.com/protosio/protos/apic/proto"
	hostagentcontrol "github.com/protosio/protos/internal/hostagent/control"
	hostagentipc "github.com/protosio/protos/internal/hostagent/ipc"
	hostagentpb "github.com/protosio/protos/internal/hostagent/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	grpcstatus "google.golang.org/grpc/status"
)

const hostAgentStatusTimeout = 900 * time.Millisecond

func (b *Backend) GetSystemStatus(ctx context.Context, _ *pbApic.GetSystemStatusRequest) (*pbApic.GetSystemStatusResponse, error) {
	services := b.protosClient
	networkStatus := &pbApic.NetworkRuntimeStatus{Supported: false, State: "unsupported"}
	if services.NetworkControl != nil {
		networkStatus = networkRuntimeStatusToProto(services.NetworkControl.NetworkRuntimeStatus(ctx))
	}
	status := &pbApic.SystemStatus{
		CoreStatus:         "running",
		WorkDir:            services.WorkDir,
		Capabilities:       services.Capabilities,
		P2PPort:            int32(services.P2PPort),
		NetworkEnabled:     networkStatus.GetEnabled(),
		HostAgentSupported: shouldReportHostAgent(runtime.GOOS),
		Network:            networkStatus,
	}
	if status.GetHostAgentSupported() {
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

func (b *Backend) StartHostAgent(ctx context.Context, _ *pbApic.StartHostAgentRequest) (*pbApic.StartHostAgentResponse, error) {
	if !shouldReportHostAgent(runtime.GOOS) {
		return nil, grpcstatus.Errorf(codes.Unimplemented, "host-agent is not available on %s", runtime.GOOS)
	}
	if status := hostAgentConnectionStatus(ctx); status.GetConnected() {
		if b.protosClient.NetworkControl != nil {
			if err := b.protosClient.NetworkControl.EnableNetwork(ctx); err != nil {
				return nil, err
			}
		}
		return &pbApic.StartHostAgentResponse{Status: status}, nil
	}
	if err := hostagentcontrol.Start(ctx, hostagentcontrol.StartOptions{
		SocketUID: -1,
		SocketGID: -1,
	}); err != nil {
		return nil, err
	}
	status, err := waitForHostAgent(ctx)
	if err != nil {
		return nil, err
	}
	if b.protosClient.NetworkControl != nil {
		if err := b.protosClient.NetworkControl.EnableNetwork(ctx); err != nil {
			return nil, err
		}
	}
	return &pbApic.StartHostAgentResponse{Status: status}, nil
}

func (b *Backend) StopHostAgent(ctx context.Context, _ *pbApic.StopHostAgentRequest) (*pbApic.StopHostAgentResponse, error) {
	if !shouldReportHostAgent(runtime.GOOS) {
		return nil, grpcstatus.Errorf(codes.Unimplemented, "host-agent is not available on %s", runtime.GOOS)
	}
	if status := hostAgentConnectionStatus(ctx); !status.GetConnected() {
		return &pbApic.StopHostAgentResponse{Status: status}, nil
	}
	if b.protosClient.NetworkControl != nil {
		if err := b.protosClient.NetworkControl.DisableNetwork(ctx); err != nil {
			return nil, err
		}
	}
	stopErr := hostagentcontrol.Stop(ctx)
	status, waitErr := waitForHostAgentStopped(ctx)
	if waitErr == nil {
		return &pbApic.StopHostAgentResponse{Status: status}, nil
	}
	if stopErr != nil {
		return nil, stopErr
	}
	return nil, waitErr
}

func (b *Backend) SetNetworkEnabled(ctx context.Context, in *pbApic.SetNetworkEnabledRequest) (*pbApic.SetNetworkEnabledResponse, error) {
	if b.protosClient.NetworkControl == nil {
		return nil, grpcstatus.Error(codes.Unimplemented, "network control is not available")
	}
	var err error
	if in.GetEnabled() {
		err = b.protosClient.NetworkControl.EnableNetwork(ctx)
	} else {
		err = b.protosClient.NetworkControl.DisableNetwork(ctx)
	}
	status := networkRuntimeStatusToProto(b.protosClient.NetworkControl.NetworkRuntimeStatus(ctx))
	if err != nil {
		return &pbApic.SetNetworkEnabledResponse{Status: status}, err
	}
	return &pbApic.SetNetworkEnabledResponse{Status: status}, nil
}

func networkRuntimeStatusToProto(status NetworkRuntimeStatus) *pbApic.NetworkRuntimeStatus {
	out := &pbApic.NetworkRuntimeStatus{
		Supported:      status.Supported,
		DesiredEnabled: status.DesiredEnabled,
		Enabled:        status.Enabled,
		State:          status.State,
		Message:        status.Message,
	}
	if status.NetworkState != nil {
		out.NetworkState = networkStateToProto(*status.NetworkState)
	}
	return out
}

func shouldReportHostAgent(goos string) bool {
	return goos == "darwin"
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

func waitForHostAgent(ctx context.Context) (*pbApic.HostAgentConnectionStatus, error) {
	var last *pbApic.HostAgentConnectionStatus
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(6 * time.Second)
	defer timeout.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			last = hostAgentConnectionStatus(ctx)
			if last.GetConnected() {
				return last, nil
			}
		case <-timeout.C:
			if last == nil {
				last = hostAgentConnectionStatus(ctx)
			}
			return nil, fmt.Errorf("host-agent did not become reachable: %s", last.GetMessage())
		}
	}
}

func waitForHostAgentStopped(ctx context.Context) (*pbApic.HostAgentConnectionStatus, error) {
	var last *pbApic.HostAgentConnectionStatus
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(6 * time.Second)
	defer timeout.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			last = hostAgentConnectionStatus(ctx)
			if !last.GetConnected() {
				return last, nil
			}
		case <-timeout.C:
			if last == nil {
				last = hostAgentConnectionStatus(ctx)
			}
			return nil, fmt.Errorf("host-agent did not stop: %s", last.GetMessage())
		}
	}
}
