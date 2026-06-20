package daemon

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	hostagentpb "github.com/protosio/protos/internal/hostagent/proto"
)

func TestCloseStopsActiveVMs(t *testing.T) {
	cmd := exec.Command("/bin/sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start child process: %v", err)
	}
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()
	defer func() {
		if processAlive(cmd.Process.Pid) {
			_ = cmd.Process.Kill()
		}
		select {
		case <-waitCh:
		default:
		}
	}()

	rootDir := t.TempDir()
	manifestPath := hostAgentVMManifestPath(rootDir, "test-vm")
	manifest := vmManifest{
		ID:     "test-vm",
		PID:    cmd.Process.Pid,
		Status: stateRunning,
	}
	if err := writeManifest(manifestPath, manifest); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	server := NewServer()
	if state := server.observedVMStatus(manifestPath, manifest); state.GetStatus() != stateRunning {
		t.Fatalf("expected running state, got %q: %s", state.GetStatus(), state.GetMessage())
	}
	if err := server.Close(); err != nil {
		t.Fatalf("close server: %v", err)
	}

	select {
	case <-waitCh:
	case <-time.After(5 * time.Second):
		t.Fatal("active VM process was not stopped")
	}

	current, err := readManifest(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if current.PID != 0 || current.Status != stateStopped {
		t.Fatalf("expected stopped manifest, got pid=%d status=%q", current.PID, current.Status)
	}
}

func TestListVMsSkipsInstanceDirectoriesWithoutManifest(t *testing.T) {
	rootDir := t.TempDir()
	manifestPath := hostAgentVMManifestPath(rootDir, "missing-manifest")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatalf("create instance directory: %v", err)
	}

	server := NewServer()
	states := server.listVMs(rootDir)
	if len(states) != 0 {
		t.Fatalf("expected no VM states, got %d: %#v", len(states), states)
	}
}

func TestVMLogsReadsConsoleLog(t *testing.T) {
	rootDir := t.TempDir()
	manifestPath := hostAgentVMManifestPath(rootDir, "test-vm")
	if err := writeManifest(manifestPath, vmManifest{
		ID:     "test-vm",
		Name:   "Test VM",
		Status: stateStopped,
	}); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(manifestPath), consoleLogFile), []byte("protosd log line\n"), 0644); err != nil {
		t.Fatalf("write console log: %v", err)
	}

	server := NewServer()
	resp, err := server.VMLogs(context.Background(), &hostagentpb.VMLogsRequest{
		Vm: &hostagentpb.VMRef{Id: "test-vm", RootDir: rootDir},
	})
	if err != nil {
		t.Fatalf("VMLogs: %v", err)
	}
	if resp.GetLogs() != "protosd log line\n" {
		t.Fatalf("logs = %q", resp.GetLogs())
	}
}
