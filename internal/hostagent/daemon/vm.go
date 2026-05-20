package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	hostagentpb "github.com/protosio/protos/internal/hostagent/proto"
	networkmodule "github.com/protosio/protos/internal/network/module"
)

const (
	desiredStateRunning = "running"
	desiredStateStopped = "stopped"
	desiredStateDeleted = "deleted"
	desiredStateUp      = "up"
	desiredStateDown    = "down"
	desiredStateConfig  = "configured"
	stateError          = "error"
	stateRunning        = "running"
	stateStopped        = "stopped"
	stateChanging       = "changing"

	consoleLogFile      = "console.log"
	gracefulStopTimeout = 30 * time.Second
	forcedStopTimeout   = 15 * time.Second
	stopPollInterval    = 500 * time.Millisecond
	legacyRunVMFlag     = "--run-vm"
	legacyManifestFlag  = "-manifest"
	manifestStatusField = "status"
	manifestPIDField    = "pid"
)

type Server struct {
	hostagentpb.UnimplementedHostAgentServer

	mu        sync.Mutex
	network   networkmodule.Module
	networkUp bool
}

func NewServer(networkModules ...networkmodule.Module) *Server {
	server := &Server{}
	if len(networkModules) > 0 {
		server.network = networkModules[0]
	}
	return server
}

func (s *Server) Apply(ctx context.Context, req *hostagentpb.ApplyRequest) (*hostagentpb.ApplyResponse, error) {
	desired := req.GetDesiredState()
	if desired == nil {
		return nil, fmt.Errorf("desired state is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	resp := &hostagentpb.ApplyResponse{Vms: make([]*hostagentpb.VMObservedState, 0, len(desired.GetVms()))}
	for _, vm := range desired.GetVms() {
		resp.Vms = append(resp.Vms, s.applyVM(vm))
	}
	if desired.GetNetwork() != nil {
		resp.Network = s.applyNetwork(desired.GetNetwork())
	}
	return resp, nil
}

func (s *Server) Status(ctx context.Context, req *hostagentpb.StatusRequest) (*hostagentpb.StatusResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resp := &hostagentpb.StatusResponse{Vms: make([]*hostagentpb.VMObservedState, 0, len(req.GetVms()))}
	for _, vm := range req.GetVms() {
		resp.Vms = append(resp.Vms, s.vmStatus(vm.GetId(), vm.GetManifestPath()))
	}
	if req.GetNetwork() {
		resp.Network = s.networkStatus("")
	}
	return resp, nil
}

func (s *Server) applyVM(vm *hostagentpb.VMDesiredState) *hostagentpb.VMObservedState {
	id := strings.TrimSpace(vm.GetId())
	manifestPath := strings.TrimSpace(vm.GetManifestPath())
	if manifestPath == "" {
		return observedError(id, manifestPath, "manifest path is required")
	}

	switch strings.ToLower(strings.TrimSpace(vm.GetDesiredState())) {
	case desiredStateRunning:
		return s.startVM(id, manifestPath)
	case desiredStateStopped:
		return s.stopVM(id, manifestPath)
	case desiredStateDeleted:
		state := s.stopVM(id, manifestPath)
		if state.GetStatus() == stateError {
			return state
		}
		state.Status = desiredStateDeleted
		return state
	default:
		return observedError(id, manifestPath, fmt.Sprintf("unknown desired state %q", vm.GetDesiredState()))
	}
}

func (s *Server) startVM(id string, manifestPath string) *hostagentpb.VMObservedState {
	manifest, err := readManifest(manifestPath)
	if err != nil {
		return observedError(id, manifestPath, err.Error())
	}
	if id == "" {
		id = manifest.ID
	}
	if manifest.PID > 0 && processAlive(manifest.PID) {
		if manifest.Status != stateRunning {
			_ = updateManifest(manifestPath, map[string]any{manifestStatusField: stateRunning})
			manifest.Status = stateRunning
		}
		return observedFromManifest(id, manifestPath, manifest, "")
	}

	if err := updateManifest(manifestPath, map[string]any{
		manifestPIDField:    0,
		manifestStatusField: stateChanging,
	}); err != nil {
		return observedError(id, manifestPath, err.Error())
	}

	executable, err := os.Executable()
	if err != nil {
		return observedError(id, manifestPath, fmt.Sprintf("resolve host agent executable: %v", err))
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return observedError(id, manifestPath, fmt.Sprintf("resolve host agent executable symlink: %v", err))
	}

	console, err := os.OpenFile(filepath.Join(filepath.Dir(manifestPath), consoleLogFile), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return observedError(id, manifestPath, fmt.Sprintf("open VM console log: %v", err))
	}
	defer console.Close()

	cmd := exec.Command(executable, legacyRunVMFlag, legacyManifestFlag, manifestPath)
	cmd.Stdin = nil
	cmd.Stdout = console
	cmd.Stderr = console
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	closeParentFilesOnExec(console.Fd())
	if err := cmd.Start(); err != nil {
		return observedError(id, manifestPath, fmt.Sprintf("start VM process: %v", err))
	}

	pid := cmd.Process.Pid
	if err := updateManifest(manifestPath, map[string]any{
		manifestPIDField:    pid,
		manifestStatusField: stateRunning,
	}); err != nil {
		return observedError(id, manifestPath, err.Error())
	}
	go waitVMProcess(manifestPath, pid, cmd)

	manifest.PID = pid
	manifest.Status = stateRunning
	return observedFromManifest(id, manifestPath, manifest, "")
}

func (s *Server) stopVM(id string, manifestPath string) *hostagentpb.VMObservedState {
	manifest, err := readManifest(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &hostagentpb.VMObservedState{Id: id, ManifestPath: manifestPath, Status: stateStopped}
		}
		return observedError(id, manifestPath, err.Error())
	}
	if id == "" {
		id = manifest.ID
	}
	if manifest.PID == 0 || !processAlive(manifest.PID) {
		_ = updateManifest(manifestPath, map[string]any{
			manifestPIDField:    0,
			manifestStatusField: stateStopped,
		})
		manifest.PID = 0
		manifest.Status = stateStopped
		return observedFromManifest(id, manifestPath, manifest, "")
	}

	_ = updateManifest(manifestPath, map[string]any{manifestStatusField: stateChanging})
	process, err := os.FindProcess(manifest.PID)
	if err != nil {
		return observedError(id, manifestPath, fmt.Sprintf("find VM process %d: %v", manifest.PID, err))
	}
	if err := process.Signal(syscall.SIGTERM); err != nil && processAlive(manifest.PID) {
		return observedError(id, manifestPath, fmt.Sprintf("stop VM process %d: %v", manifest.PID, err))
	}
	if !waitForProcessExit(manifest.PID, gracefulStopTimeout) {
		if err := process.Signal(syscall.SIGTERM); err != nil && processAlive(manifest.PID) {
			return observedError(id, manifestPath, fmt.Sprintf("force stop VM process %d: %v", manifest.PID, err))
		}
	}
	if !waitForProcessExit(manifest.PID, forcedStopTimeout) {
		if err := process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			return observedError(id, manifestPath, fmt.Sprintf("kill VM process %d: %v", manifest.PID, err))
		}
		_ = waitForProcessExit(manifest.PID, forcedStopTimeout)
	}

	if err := updateManifest(manifestPath, map[string]any{
		manifestPIDField:    0,
		manifestStatusField: stateStopped,
	}); err != nil {
		return observedError(id, manifestPath, err.Error())
	}
	manifest.PID = 0
	manifest.Status = stateStopped
	return observedFromManifest(id, manifestPath, manifest, "")
}

func (s *Server) vmStatus(id string, manifestPath string) *hostagentpb.VMObservedState {
	manifest, err := readManifest(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &hostagentpb.VMObservedState{Id: id, ManifestPath: manifestPath, Status: stateStopped}
		}
		return observedError(id, manifestPath, err.Error())
	}
	if id == "" {
		id = manifest.ID
	}
	if manifest.PID > 0 && processAlive(manifest.PID) {
		if manifest.Status != stateRunning {
			_ = updateManifest(manifestPath, map[string]any{manifestStatusField: stateRunning})
			manifest.Status = stateRunning
		}
		return observedFromManifest(id, manifestPath, manifest, "")
	}
	if manifest.PID != 0 || manifest.Status == stateRunning || manifest.Status == stateChanging {
		_ = updateManifest(manifestPath, map[string]any{
			manifestPIDField:    0,
			manifestStatusField: stateStopped,
		})
		manifest.PID = 0
		manifest.Status = stateStopped
	}
	return observedFromManifest(id, manifestPath, manifest, "")
}

func (s *Server) applyNetwork(desired *hostagentpb.NetworkDesiredState) *hostagentpb.NetworkObservedState {
	if s.network == nil {
		return &hostagentpb.NetworkObservedState{Up: false, Message: "network module is not configured"}
	}
	state := strings.ToLower(strings.TrimSpace(desired.GetDesiredState()))
	if state == "" {
		state = desiredStateUp
	}

	switch state {
	case desiredStateUp:
		config, err := networkConfigFromProto(desired.GetConfig())
		if err != nil {
			return s.networkStatus(err.Error())
		}
		if err := s.network.Up(config); err != nil {
			return s.networkStatus(err.Error())
		}
		s.networkUp = true
		if desired.GetReconcilePeers() || hasNetworkPeers(desired) {
			if err := s.network.ConfigurePeers(config, networkPeersFromProto(desired.GetInstances(), desired.GetDevices())); err != nil {
				return s.networkStatus(err.Error())
			}
		}
		if err := s.applyNamespacedInterfaces(config, desired.GetNamespacedInterfaces()); err != nil {
			return s.networkStatus(err.Error())
		}
		return s.networkStatus("")
	case desiredStateConfig, "configure", "reconcile":
		config, err := networkConfigFromProto(desired.GetConfig())
		if err != nil {
			return s.networkStatus(err.Error())
		}
		if desired.GetReconcilePeers() || hasNetworkPeers(desired) {
			if err := s.network.ConfigurePeers(config, networkPeersFromProto(desired.GetInstances(), desired.GetDevices())); err != nil {
				return s.networkStatus(err.Error())
			}
		}
		if err := s.applyNamespacedInterfaces(config, desired.GetNamespacedInterfaces()); err != nil {
			return s.networkStatus(err.Error())
		}
		s.networkUp = true
		return s.networkStatus("")
	case desiredStateDown:
		if err := s.network.Down(); err != nil {
			return s.networkStatus(err.Error())
		}
		s.networkUp = false
		return s.networkStatus("")
	default:
		return s.networkStatus(fmt.Sprintf("unknown network desired state %q", desired.GetDesiredState()))
	}
}

func (s *Server) applyNamespacedInterfaces(config networkmodule.Config, interfaces []*hostagentpb.NamespacedInterface) error {
	if len(interfaces) == 0 {
		return nil
	}
	namespacedModule, ok := s.network.(networkmodule.NamespacedInterfaceModule)
	if !ok {
		return fmt.Errorf("network module %q does not support namespaced interfaces", s.network.Name())
	}
	for _, iface := range interfaces {
		ip := net.ParseIP(iface.GetIp())
		if ip == nil {
			return fmt.Errorf("invalid namespaced interface IP %q", iface.GetIp())
		}
		if err := namespacedModule.CreateNamespacedInterface(config, iface.GetNetnsPath(), ip); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) networkStatus(message string) *hostagentpb.NetworkObservedState {
	moduleName := ""
	if s.network != nil {
		moduleName = s.network.Name()
	}
	return &hostagentpb.NetworkObservedState{
		Module:  moduleName,
		Up:      s.networkUp,
		Message: message,
	}
}

func hasNetworkPeers(desired *hostagentpb.NetworkDesiredState) bool {
	return len(desired.GetInstances()) > 0 || len(desired.GetDevices()) > 0
}

func networkConfigFromProto(config *hostagentpb.NetworkConfig) (networkmodule.Config, error) {
	if config == nil {
		return networkmodule.Config{}, fmt.Errorf("network config is required")
	}

	addr, err := netip.ParseAddr(config.GetIpv6Address())
	if err != nil {
		return networkmodule.Config{}, fmt.Errorf("invalid IPv6 address %q: %w", config.GetIpv6Address(), err)
	}

	return networkmodule.Config{
		IPv6Address:         addr,
		WireGuardPrivateKey: config.GetWireguardPrivateKey(),
		Domain:              config.GetDomain(),
	}, nil
}

func networkPeersFromProto(instances []*hostagentpb.InstancePeer, devices []*hostagentpb.DevicePeer) networkmodule.Peers {
	peers := networkmodule.Peers{
		Instances: make([]networkmodule.InstancePeer, 0, len(instances)),
		Devices:   make([]networkmodule.DevicePeer, 0, len(devices)),
	}

	for _, instance := range instances {
		peers.Instances = append(peers.Instances, networkmodule.InstancePeer{
			ID:        instance.GetId(),
			Name:      instance.GetName(),
			PublicKey: instance.GetPublicKey(),
			PublicIP:  instance.GetPublicIp(),
		})
	}

	for _, device := range devices {
		peers.Devices = append(peers.Devices, networkmodule.DevicePeer{
			Name:      device.GetName(),
			PublicKey: device.GetPublicKey(),
		})
	}

	return peers
}

type manifest struct {
	ID       string `json:"id"`
	PublicIP string `json:"public_ip"`
	Status   string `json:"status"`
	PID      int    `json:"pid,omitempty"`
}

func readManifest(path string) (manifest, error) {
	var m manifest
	data, err := os.ReadFile(path)
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return m, fmt.Errorf("decode VM manifest %s: %w", path, err)
	}
	return m, nil
}

func updateManifest(path string, updates map[string]any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	var values map[string]any
	if err := json.Unmarshal(data, &values); err != nil {
		return fmt.Errorf("decode VM manifest %s: %w", path, err)
	}
	for key, value := range updates {
		values[key] = value
	}
	updated, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return fmt.Errorf("encode VM manifest %s: %w", path, err)
	}
	updated = append(updated, '\n')

	tmp, err := os.CreateTemp(filepath.Dir(path), ".manifest-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(updated); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := preserveManifestMetadata(tmpPath, info); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func preserveManifestMetadata(path string, info os.FileInfo) error {
	if info == nil {
		return nil
	}
	if err := os.Chmod(path, info.Mode().Perm()); err != nil {
		return fmt.Errorf("preserve manifest mode: %w", err)
	}
	if os.Geteuid() != 0 {
		return nil
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}
	if err := os.Chown(path, int(stat.Uid), int(stat.Gid)); err != nil {
		return fmt.Errorf("preserve manifest owner: %w", err)
	}
	return nil
}

func observedFromManifest(id string, manifestPath string, manifest manifest, message string) *hostagentpb.VMObservedState {
	if id == "" {
		id = manifest.ID
	}
	return &hostagentpb.VMObservedState{
		Id:           id,
		ManifestPath: manifestPath,
		Status:       manifest.Status,
		Pid:          int32(manifest.PID),
		PublicIp:     manifest.PublicIP,
		Message:      message,
	}
}

func observedError(id string, manifestPath string, message string) *hostagentpb.VMObservedState {
	return &hostagentpb.VMObservedState{
		Id:           id,
		ManifestPath: manifestPath,
		Status:       stateError,
		Message:      message,
	}
}

func waitVMProcess(manifestPath string, pid int, cmd *exec.Cmd) {
	_ = cmd.Wait()
	current, err := readManifest(manifestPath)
	if err != nil {
		return
	}
	if current.PID == pid {
		_ = updateManifest(manifestPath, map[string]any{
			manifestPIDField:    0,
			manifestStatusField: stateStopped,
		})
	}
}

func processAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}

func waitForProcessExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			return true
		}
		time.Sleep(stopPollInterval)
	}
	return !processAlive(pid)
}

func closeParentFilesOnExec(keep ...uintptr) {
	keepFDs := map[int]struct{}{
		0: {},
		1: {},
		2: {},
	}
	for _, fd := range keep {
		keepFDs[int(fd)] = struct{}{}
	}

	entries, err := os.ReadDir("/dev/fd")
	if err != nil {
		return
	}
	for _, entry := range entries {
		fd, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		if _, ok := keepFDs[fd]; ok {
			continue
		}
		syscall.CloseOnExec(fd)
	}
}
