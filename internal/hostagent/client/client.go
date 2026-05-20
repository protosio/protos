package client

import (
	"context"
	"fmt"
	"time"

	hostagentipc "github.com/protosio/protos/internal/hostagent/ipc"
	hostagentpb "github.com/protosio/protos/internal/hostagent/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const defaultRPCTimeout = 4 * time.Minute

type Client struct {
	socket string
	conn   *grpc.ClientConn
	client hostagentpb.HostAgentClient
}

func New() (*Client, error) {
	socket := hostagentipc.SocketPath()
	conn, err := grpc.NewClient("unix://"+socket, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("host agent ipc: %w", err)
	}
	return &Client{
		socket: socket,
		conn:   conn,
		client: hostagentpb.NewHostAgentClient(conn),
	}, nil
}

func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) ApplyVM(id string, manifestPath string, desiredState string) (*hostagentpb.VMObservedState, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRPCTimeout)
	defer cancel()

	resp, err := c.client.Apply(ctx, &hostagentpb.ApplyRequest{
		DesiredState: &hostagentpb.HostDesiredState{
			Vms: []*hostagentpb.VMDesiredState{
				{
					Id:           id,
					ManifestPath: manifestPath,
					DesiredState: desiredState,
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("host agent apply via %s: %w", c.socket, err)
	}
	if len(resp.GetVms()) == 0 {
		return nil, fmt.Errorf("host agent apply via %s returned no VM state", c.socket)
	}
	state := resp.GetVms()[0]
	if state.GetMessage() != "" && state.GetStatus() == "error" {
		return state, fmt.Errorf("host agent apply failed for VM '%s': %s", id, state.GetMessage())
	}
	return state, nil
}

func (c *Client) VMStatus(id string, manifestPath string) (*hostagentpb.VMObservedState, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRPCTimeout)
	defer cancel()

	resp, err := c.client.Status(ctx, &hostagentpb.StatusRequest{
		Vms: []*hostagentpb.VMRef{
			{
				Id:           id,
				ManifestPath: manifestPath,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("host agent status via %s: %w", c.socket, err)
	}
	if len(resp.GetVms()) == 0 {
		return nil, fmt.Errorf("host agent status via %s returned no VM state", c.socket)
	}
	return resp.GetVms()[0], nil
}
