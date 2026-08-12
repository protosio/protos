//go:build darwin

package localmacos

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"

	hostagentpb "github.com/protosio/protos/internal/hostagent/proto"
	"github.com/protosio/protos/internal/provisioners"
)

type fakeLocalMacOSHostAgent struct {
	state        *hostagentpb.VMObservedState
	statusErr    error
	applyState   *hostagentpb.VMObservedState
	applyErr     error
	statusCalls  int
	applyCalls   int
	desiredState string
}

func (fake *fakeLocalMacOSHostAgent) Close() error { return nil }

func (fake *fakeLocalMacOSHostAgent) ApplyVM(_ string, _ string, desiredState string, _ *hostagentpb.VMConfig) (*hostagentpb.VMObservedState, error) {
	fake.applyCalls++
	fake.desiredState = desiredState
	return fake.applyState, fake.applyErr
}

func (fake *fakeLocalMacOSHostAgent) VMStatus(_, _, _ string) (*hostagentpb.VMObservedState, error) {
	fake.statusCalls++
	return fake.state, fake.statusErr
}

func (fake *fakeLocalMacOSHostAgent) ListVMs(string) ([]*hostagentpb.VMObservedState, error) {
	return nil, nil
}

func (fake *fakeLocalMacOSHostAgent) VMLogs(_, _, _ string) (string, error) {
	return "", nil
}

func newTestLocalMacOS(t *testing.T, hostAgent *fakeLocalMacOSHostAgent) *localMacOS {
	t.Helper()
	lm := &localMacOS{
		vmDir: t.TempDir(),
		newHostAgentClient: func() (localMacOSHostAgent, error) {
			return hostAgent, nil
		},
	}
	if err := lm.Init(); err != nil {
		t.Fatal(err)
	}
	return lm
}

func writeLocalMacOSInstanceArtifact(t *testing.T, lm *localMacOS, id string) string {
	t.Helper()
	dir := lm.instanceDir(id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "live-artifact"), []byte("must remain until host-agent deletion succeeds"), 0600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestDeleteInstanceHostAgentStatusFailurePreservesArtifacts(t *testing.T) {
	hostAgent := &fakeLocalMacOSHostAgent{statusErr: errors.New("status endpoint unavailable")}
	lm := newTestLocalMacOS(t, hostAgent)
	instanceDir := writeLocalMacOSInstanceArtifact(t, lm, "vm-live")

	err := lm.DeleteInstance("vm-live", localMacOSLocation)
	if err == nil || !errors.Is(err, hostAgent.statusErr) {
		t.Fatalf("DeleteInstance error = %v, want status failure", err)
	}
	if hostAgent.applyCalls != 0 {
		t.Fatalf("host-agent apply calls = %d, want 0 after inconclusive status", hostAgent.applyCalls)
	}
	if _, err := os.Stat(instanceDir); err != nil {
		t.Fatalf("instance artifacts were removed while VM may be live: %v", err)
	}
}

func TestDeleteInstanceHostAgentDeleteFailurePreservesArtifacts(t *testing.T) {
	config := &localMacOSVMConfig{ID: "vm-live", Name: "live"}
	hostAgent := &fakeLocalMacOSHostAgent{
		state:      &hostagentpb.VMObservedState{Id: config.ID, Status: provisioners.ServerStateStopped, Config: vmConfigToProto(config)},
		applyErr:   errors.New("delete apply failed"),
		applyState: &hostagentpb.VMObservedState{Id: config.ID, Status: "error"},
	}
	lm := newTestLocalMacOS(t, hostAgent)
	instanceDir := writeLocalMacOSInstanceArtifact(t, lm, config.ID)

	err := lm.DeleteInstance(config.ID, localMacOSLocation)
	if err == nil || !errors.Is(err, hostAgent.applyErr) {
		t.Fatalf("DeleteInstance error = %v, want host-agent delete failure", err)
	}
	if hostAgent.applyCalls != 1 || hostAgent.desiredState != "deleted" {
		t.Fatalf("host-agent delete calls = %d state=%q, want one deleted apply", hostAgent.applyCalls, hostAgent.desiredState)
	}
	if _, err := os.Stat(instanceDir); err != nil {
		t.Fatalf("instance artifacts were removed after failed host-agent delete: %v", err)
	}
}

func TestDeleteInstanceProvenAbsentRemovesResidualArtifacts(t *testing.T) {
	hostAgent := &fakeLocalMacOSHostAgent{state: &hostagentpb.VMObservedState{Status: provisioners.ServerStateStopped}}
	lm := newTestLocalMacOS(t, hostAgent)
	instanceDir := writeLocalMacOSInstanceArtifact(t, lm, "vm-absent")

	if err := lm.DeleteInstance("vm-absent", localMacOSLocation); err != nil {
		t.Fatal(err)
	}
	if hostAgent.statusCalls != 2 {
		t.Fatalf("host-agent status calls = %d, want ID and name absence checks", hostAgent.statusCalls)
	}
	if hostAgent.applyCalls != 0 {
		t.Fatalf("host-agent apply calls = %d, want 0 for proven absence", hostAgent.applyCalls)
	}
	if _, err := os.Stat(instanceDir); !os.IsNotExist(err) {
		t.Fatalf("residual instance artifacts still exist, stat error = %v", err)
	}
}

func TestDeleteVolumeWithoutMetadataRemovesCanonicalAttachedVolume(t *testing.T) {
	config := &localMacOSVMConfig{
		ID:   "vm-stopped",
		Name: "stopped",
		Volumes: []localMacOSAttachedVolume{
			{ID: "vol-retry", Name: "data", Path: "stale-path.raw", SizeMiB: 1},
		},
	}
	hostAgent := &fakeLocalMacOSHostAgent{
		state: &hostagentpb.VMObservedState{Id: config.ID, Status: provisioners.ServerStateStopped, Config: vmConfigToProto(config)},
	}
	lm := newTestLocalMacOS(t, hostAgent)
	volumePath := filepath.Join(lm.volumesDir(), "vol-retry.raw")
	if err := os.WriteFile(volumePath, []byte("volume"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(lm.volumeMetadataPath("vol-retry")); !os.IsNotExist(err) {
		t.Fatalf("test requires missing metadata, stat error = %v", err)
	}

	info, err := lm.GetInstanceInfo(config.ID, localMacOSLocation)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Volumes) != 1 || info.Volumes[0].VolumeID != "vol-retry" {
		t.Fatalf("host-agent config volumes = %#v, want attached vol-retry", info.Volumes)
	}
	if err := lm.DeleteVolume("vol-retry", localMacOSLocation); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(volumePath); !os.IsNotExist(err) {
		t.Fatalf("canonical volume remained after restart-idempotent deletion, stat error = %v", err)
	}
}

func TestAllocateLocalMacOSStaticIPUsesObservedNATNetwork(t *testing.T) {
	_, network, err := net.ParseCIDR("192.168.138.0/23")
	if err != nil {
		t.Fatal(err)
	}
	gateway := net.ParseIP("192.168.139.3")
	ip, err := allocateLocalMacOSStaticIP(network, gateway, "vm-test", nil)
	if err != nil {
		t.Fatal(err)
	}
	parsed := net.ParseIP(ip)
	if parsed == nil || !network.Contains(parsed) {
		t.Fatalf("allocated IP %q outside network %s", ip, network.String())
	}
	if ip == gateway.String() {
		t.Fatalf("allocated gateway IP %q", ip)
	}
	if ip == "192.168.64.192" {
		t.Fatalf("allocated old hard-coded subnet IP %q", ip)
	}
}

func TestAllocateLocalMacOSStaticIPSkipsUsedAddresses(t *testing.T) {
	_, network, err := net.ParseCIDR("192.168.64.0/30")
	if err != nil {
		t.Fatal(err)
	}
	ip, err := allocateLocalMacOSStaticIP(network, net.ParseIP("192.168.64.1"), "vm-test", map[string]struct{}{
		"192.168.64.2": {},
	})
	if err == nil {
		t.Fatalf("allocated %q from exhausted network", ip)
	}
}

func TestWriteJSONFileReplacesExistingFileAtomically(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metadata.json")

	if err := writeJSONFile(path, map[string]any{"status": "running"}); err != nil {
		t.Fatalf("write initial JSON: %v", err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatalf("chmod initial JSON: %v", err)
	}

	if err := writeJSONFile(path, map[string]any{"status": "stopped"}); err != nil {
		t.Fatalf("replace JSON: %v", err)
	}

	var got struct {
		Status string `json:"status"`
	}
	if err := readJSONFile(path, &got); err != nil {
		t.Fatalf("read replaced JSON: %v", err)
	}
	if got.Status != "stopped" {
		t.Fatalf("status = %q, want stopped", got.Status)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat replaced JSON: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0600 {
		t.Fatalf("mode = %v, want 0600", mode)
	}
}
