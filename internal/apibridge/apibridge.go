package apibridge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/Masterminds/semver/v3"
	"github.com/protosio/protos/apic"
	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/protosio/protos/internal/protosd"
	"github.com/protosio/protos/internal/util"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	DefaultVersion      = "0.1.0-dev.23"
	defaultBufConnSize  = 16 * 1024 * 1024
	defaultCapabilities = "default,no-api,no-network"
)

type StartConfig struct {
	ConfigFile   string `json:"config_file"`
	DataDir      string `json:"data_dir"`
	Capabilities string `json:"capabilities"`
	LogLevel     string `json:"log_level"`
}

type Bridge struct {
	mu       sync.Mutex
	node     *protosd.Node
	server   *grpc.Server
	listener *bufconn.Listener
	conn     *grpc.ClientConn
}

func Start(ctx context.Context, rawConfig []byte) (*Bridge, error) {
	var cfg StartConfig
	if len(rawConfig) > 0 && strings.TrimSpace(string(rawConfig)) != "" {
		if err := json.Unmarshal(rawConfig, &cfg); err != nil {
			return nil, fmt.Errorf("decode bridge config: %w", err)
		}
	}
	if strings.TrimSpace(cfg.ConfigFile) == "" {
		cfg.ConfigFile = "protos.yaml"
	}
	if strings.TrimSpace(cfg.Capabilities) == "" {
		cfg.Capabilities = defaultCapabilities
	}
	if strings.TrimSpace(cfg.LogLevel) != "" {
		level, err := logrus.ParseLevel(cfg.LogLevel)
		if err != nil {
			return nil, err
		}
		util.SetLogLevel(level)
	}

	version, err := semver.NewVersion(DefaultVersion)
	if err != nil {
		return nil, err
	}
	node, err := protosd.NewNode(cfg.ConfigFile, version, protosd.Options{
		DataDir:      cfg.DataDir,
		Capabilities: cfg.Capabilities,
	})
	if err != nil {
		return nil, err
	}
	if err := node.Start(); err != nil {
		node.Close()
		return nil, err
	}

	listener := bufconn.Listen(defaultBufConnSize)
	server := apic.NewGRPCServer(node.APIServices())
	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && serveErr != grpc.ErrServerStopped {
			util.GetLogger("apibridge").Errorf("embedded API server stopped unexpectedly: %v", serveErr)
		}
	}()

	conn, err := grpc.NewClient(
		"passthrough:///protos-embedded",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		server.Stop()
		_ = listener.Close()
		node.Close()
		return nil, err
	}

	return &Bridge{
		node:     node,
		server:   server,
		listener: listener,
		conn:     conn,
	}, nil
}

func (b *Bridge) Call(ctx context.Context, method string, request []byte) ([]byte, error) {
	if b == nil {
		return nil, fmt.Errorf("bridge is not started")
	}
	methodDesc, fullMethod, err := resolveMethod(method)
	if err != nil {
		return nil, err
	}

	req := dynamicpb.NewMessage(methodDesc.Input())
	if len(request) > 0 {
		if err := proto.Unmarshal(request, req); err != nil {
			return nil, fmt.Errorf("decode %s request: %w", methodDesc.Name(), err)
		}
	}

	resp := dynamicpb.NewMessage(methodDesc.Output())
	b.mu.Lock()
	conn := b.conn
	b.mu.Unlock()
	if conn == nil {
		return nil, fmt.Errorf("bridge is stopped")
	}
	if err := conn.Invoke(ctx, fullMethod, req, resp); err != nil {
		return nil, err
	}
	encoded, err := proto.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("encode %s response: %w", methodDesc.Name(), err)
	}
	return encoded, nil
}

func (b *Bridge) WatchChanges(ctx context.Context, request []byte, emit func([]byte) bool) error {
	if b == nil {
		return fmt.Errorf("bridge is not started")
	}
	if emit == nil {
		return fmt.Errorf("watch callback is required")
	}

	var req pbApic.WatchChangesRequest
	if len(request) > 0 {
		if err := proto.Unmarshal(request, &req); err != nil {
			return fmt.Errorf("decode WatchChanges request: %w", err)
		}
	}

	b.mu.Lock()
	conn := b.conn
	b.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("bridge is stopped")
	}

	stream, err := pbApic.NewProtosClientApiClient(conn).WatchChanges(ctx, &req)
	if err != nil {
		return err
	}
	for {
		resp, err := stream.Recv()
		if err != nil {
			if err == io.EOF || ctx.Err() != nil {
				return nil
			}
			return err
		}
		encoded, err := proto.Marshal(resp)
		if err != nil {
			return fmt.Errorf("encode WatchChanges response: %w", err)
		}
		if !emit(encoded) {
			return nil
		}
	}
}

func (b *Bridge) Stop() error {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	conn := b.conn
	server := b.server
	listener := b.listener
	node := b.node
	b.conn = nil
	b.server = nil
	b.listener = nil
	b.node = nil
	b.mu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}
	if server != nil {
		server.GracefulStop()
	}
	if listener != nil {
		_ = listener.Close()
	}
	if node != nil {
		node.Close()
	}
	return nil
}

func resolveMethod(method string) (protoreflect.MethodDescriptor, string, error) {
	method = strings.TrimSpace(method)
	if method == "" {
		return nil, "", fmt.Errorf("API method is required")
	}
	parts := strings.Split(method, "/")
	methodName := parts[len(parts)-1]
	if methodName == "" && len(parts) > 1 {
		methodName = parts[len(parts)-2]
	}
	methodName = strings.TrimSpace(methodName)
	if dot := strings.LastIndex(methodName, "."); dot >= 0 {
		methodName = methodName[dot+1:]
	}

	service := pbApic.File_apic_proto_apic_proto.Services().ByName("ProtosClientApi")
	if service == nil {
		return nil, "", fmt.Errorf("ProtosClientApi descriptor is unavailable")
	}
	methodDesc := service.Methods().ByName(protoreflect.Name(methodName))
	if methodDesc == nil {
		return nil, "", fmt.Errorf("unknown Protos API method %q", method)
	}
	fullMethod := fmt.Sprintf("/%s.%s/%s", service.ParentFile().Package(), service.Name(), methodDesc.Name())
	return methodDesc, fullMethod, nil
}
