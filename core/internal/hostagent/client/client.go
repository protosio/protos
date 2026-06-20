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
	return NewWithSocket(hostagentipc.SocketPath())
}

func NewWithSocket(socket string) (*Client, error) {
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

func (c *Client) ApplyVM(id string, rootDir string, desiredState string, config *hostagentpb.VMConfig) (*hostagentpb.VMObservedState, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRPCTimeout)
	defer cancel()

	resp, err := c.client.Apply(ctx, &hostagentpb.ApplyRequest{
		DesiredState: &hostagentpb.HostDesiredState{
			Vms: []*hostagentpb.VMDesiredState{
				{
					Id:           id,
					RootDir:      rootDir,
					DesiredState: desiredState,
					Config:       config,
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

func (c *Client) VMStatus(id string, name string, rootDir string) (*hostagentpb.VMObservedState, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRPCTimeout)
	defer cancel()

	resp, err := c.client.Status(ctx, &hostagentpb.StatusRequest{
		Vms: []*hostagentpb.VMRef{
			{
				Id:      id,
				Name:    name,
				RootDir: rootDir,
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

func (c *Client) ListVMs(rootDir string) ([]*hostagentpb.VMObservedState, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRPCTimeout)
	defer cancel()

	resp, err := c.client.Status(ctx, &hostagentpb.StatusRequest{
		RootDir: rootDir,
		ListVms: true,
	})
	if err != nil {
		return nil, fmt.Errorf("host agent status via %s: %w", c.socket, err)
	}
	return resp.GetVms(), nil
}

func (c *Client) VMLogs(id string, name string, rootDir string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultRPCTimeout)
	defer cancel()

	resp, err := c.client.VMLogs(ctx, &hostagentpb.VMLogsRequest{
		Vm: &hostagentpb.VMRef{
			Id:      id,
			Name:    name,
			RootDir: rootDir,
		},
	})
	if err != nil {
		return "", fmt.Errorf("host agent VM logs via %s: %w", c.socket, err)
	}
	return resp.GetLogs(), nil
}

func (c *Client) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := c.client.Shutdown(ctx, &hostagentpb.ShutdownRequest{}); err != nil {
		return fmt.Errorf("host agent shutdown via %s: %w", c.socket, err)
	}
	return nil
}
