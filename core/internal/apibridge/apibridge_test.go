package apibridge

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	pbApic "github.com/protosio/protos/apic/proto"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestBridgeCallsProtobufAPI(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "protos.yaml")
	if err := os.WriteFile(configFile, []byte("p2pport: 0\n"), 0600); err != nil {
		t.Fatal(err)
	}

	rawConfig := []byte(fmt.Sprintf(
		`{"config_file":%q,"data_dir":%q,"capabilities":"none","log_level":"error"}`,
		configFile,
		filepath.Join(tempDir, "data"),
	))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	bridge, err := Start(ctx, rawConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := bridge.Stop(); err != nil {
			t.Errorf("stop bridge: %v", err)
		}
	}()

	request, err := proto.Marshal(&pbApic.GetSupportedProvisionersRequest{})
	if err != nil {
		t.Fatal(err)
	}
	responseBytes, err := bridge.Call(ctx, "GetSupportedProvisioners", request)
	if err != nil {
		t.Fatal(err)
	}

	var response pbApic.GetSupportedProvisionersResponse
	if err := proto.Unmarshal(responseBytes, &response); err != nil {
		t.Fatal(err)
	}
	if len(response.ProvisionerTypes) == 0 {
		t.Fatal("expected at least one supported provisioner")
	}

	if _, err := bridge.Call(ctx, "NoSuchMethod", nil); err == nil {
		t.Fatal("expected an error for unknown methods")
	}

	watchRequest, err := proto.Marshal(&pbApic.WatchChangesRequest{IncludeSnapshot: true})
	if err != nil {
		t.Fatal(err)
	}
	var watchResponseBytes []byte
	if err := bridge.WatchChanges(ctx, watchRequest, func(data []byte) bool {
		watchResponseBytes = append([]byte(nil), data...)
		return false
	}); err != nil {
		t.Fatal(err)
	}
	var watchResponse pbApic.WatchChangesResponse
	if err := proto.Unmarshal(watchResponseBytes, &watchResponse); err != nil {
		t.Fatal(err)
	}
	if watchResponse.Reason != "initial" || !watchResponse.RuntimeChanged {
		t.Fatalf("unexpected watch response: reason=%q runtime_changed=%v", watchResponse.Reason, watchResponse.RuntimeChanged)
	}

	taskWatchRequest, err := proto.Marshal(&pbApic.WatchTaskRequest{Id: "missing-task", IncludeSnapshot: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := bridge.WatchTask(ctx, taskWatchRequest, func(data []byte) bool {
		t.Fatalf("unexpected task watch response: %d bytes", len(data))
		return false
	}); err == nil {
		t.Fatal("expected WatchTask to return an error for an unknown task")
	}
}

func TestBridgeUserErrorUsesStatusMessage(t *testing.T) {
	err := bridgeUserError(grpcstatus.Error(codes.Unknown, "failed to deploy instance: failed to connect"))
	if got, want := err.Error(), "failed to deploy instance: failed to connect"; got != want {
		t.Fatalf("bridgeUserError() = %q, want %q", got, want)
	}
}
