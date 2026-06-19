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
	"sort"
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
	closed    bool
	shutdown  func()
	activeVMs map[string]struct{}
}

func NewServer(networkModules ...networkmodule.Module) *Server {
	server := &Server{activeVMs: map[string]struct{}{}}
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
	if s.closed {
		return nil, fmt.Errorf("host agent is shutting down")
	}

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
	if s.closed {
		return nil, fmt.Errorf("host agent is shutting down")
	}

	vmCapacity := len(req.GetVms())
	if req.GetListVms() {
		vmCapacity = 8
	}
	resp := &hostagentpb.StatusResponse{Vms: make([]*hostagentpb.VMObservedState, 0, vmCapacity)}
	if req.GetListVms() {
		resp.Vms = append(resp.Vms, s.listVMs(req.GetRootDir())...)
	}
	for _, vm := range req.GetVms() {
		resp.Vms = append(resp.Vms, s.vmStatus(vm))
	}
	if req.GetNetwork() {
		resp.Network = s.networkStatus("")
	}
	return resp, nil
}

func (s *Server) SetShutdownFunc(shutdown func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shutdown = shutdown
}

func (s *Server) Shutdown(ctx context.Context, req *hostagentpb.ShutdownRequest) (*hostagentpb.ShutdownResponse, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return &hostagentpb.ShutdownResponse{Message: "host agent is already shutting down"}, nil
	}
	shutdown := s.shutdown
	s.mu.Unlock()

	if shutdown == nil {
		return &hostagentpb.ShutdownResponse{Message: "host agent shutdown is not configured"}, nil
	}
	go shutdown()
	return &hostagentpb.ShutdownResponse{Message: "host agent shutdown requested"}, nil
}

func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	var err error
	if stopErr := s.stopActiveVMsLocked(); stopErr != nil {
		err = errors.Join(err, stopErr)
	}

	if s.network == nil {
		return err
	}

	if downErr := s.network.Down(); downErr != nil {
		err = errors.Join(err, fmt.Errorf("tear down network: %w", downErr))
	}
	s.networkUp = false
	if closeErr := s.network.Close(); closeErr != nil {
		err = errors.Join(err, fmt.Errorf("close network module: %w", closeErr))
	}
	s.network = nil
	return err
}

func (s *Server) applyVM(vm *hostagentpb.VMDesiredState) *hostagentpb.VMObservedState {
	id := strings.TrimSpace(firstNonEmpty(vm.GetId(), vm.GetConfig().GetId()))
	desired := strings.ToLower(strings.TrimSpace(vm.GetDesiredState()))
	if desired == "" {
		desired = desiredStateStopped
	}

	manifestPath, manifest, err := s.prepareVMManifest(vm)
	if err != nil {
		return observedError(id, err.Error())
	}

	switch desired {
	case desiredStateConfig:
		return observedFromManifest(manifestPath, manifest, "")
	case desiredStateRunning:
		return s.startVM(manifestPath)
	case desiredStateStopped:
		return s.stopVM(id, manifestPath)
	case desiredStateDeleted:
		state := s.stopVM(id, manifestPath)
		if state.GetStatus() == stateError {
			return state
		}
		if err := os.Remove(manifestPath); err != nil && !os.IsNotExist(err) {
			return observedError(id, fmt.Sprintf("remove VM manifest: %v", err))
		}
		state.Status = desiredStateDeleted
		return state
	default:
		return observedError(id, fmt.Sprintf("unknown desired state %q", vm.GetDesiredState()))
	}
}

func (s *Server) prepareVMManifest(vm *hostagentpb.VMDesiredState) (string, vmManifest, error) {
	if vm.GetConfig() != nil {
		return writeConfiguredVMManifest(vm.GetRootDir(), vm.GetConfig())
	}
	ref := &hostagentpb.VMRef{Id: vm.GetId(), RootDir: vm.GetRootDir()}
	manifestPath, manifest, err := resolveVMManifest(ref)
	if err != nil {
		return "", vmManifest{}, err
	}
	return manifestPath, manifest, nil
}

func (s *Server) startVM(manifestPath string) *hostagentpb.VMObservedState {
	manifest, err := readManifest(manifestPath)
	if err != nil {
		return observedError("", err.Error())
	}
	if manifest.PID > 0 && processAlive(manifest.PID) {
		if manifest.Status != stateRunning {
			_ = updateManifest(manifestPath, map[string]any{manifestStatusField: stateRunning})
			manifest.Status = stateRunning
		}
		s.trackActiveVMLocked(manifestPath)
		return observedFromManifest(manifestPath, manifest, "")
	}

	if err := updateManifest(manifestPath, map[string]any{
		manifestPIDField:    0,
		manifestStatusField: stateChanging,
	}); err != nil {
		return observedError(manifest.ID, err.Error())
	}

	executable, err := os.Executable()
	if err != nil {
		return observedError(manifest.ID, fmt.Sprintf("resolve host agent executable: %v", err))
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return observedError(manifest.ID, fmt.Sprintf("resolve host agent executable symlink: %v", err))
	}
	if err := ensureVMRunnerEntitled(executable); err != nil {
		return observedError(manifest.ID, err.Error())
	}

	console, err := os.OpenFile(filepath.Join(filepath.Dir(manifestPath), consoleLogFile), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return observedError(manifest.ID, fmt.Sprintf("open VM console log: %v", err))
	}
	defer console.Close()

	cmd := exec.Command(executable, legacyRunVMFlag, legacyManifestFlag, manifestPath)
	cmd.Stdin = nil
	cmd.Stdout = console
	cmd.Stderr = console
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	closeParentFilesOnExec(console.Fd())
	if err := cmd.Start(); err != nil {
		return observedError(manifest.ID, fmt.Sprintf("start VM process: %v", err))
	}

	pid := cmd.Process.Pid
	if err := updateManifest(manifestPath, map[string]any{
		manifestPIDField:    pid,
		manifestStatusField: stateRunning,
	}); err != nil {
		return observedError(manifest.ID, err.Error())
	}
	go waitVMProcess(manifestPath, pid, cmd)
	s.trackActiveVMLocked(manifestPath)

	manifest.PID = pid
	manifest.Status = stateRunning
	return observedFromManifest(manifestPath, manifest, "")
}

func (s *Server) stopVM(id string, manifestPath string) *hostagentpb.VMObservedState {
	manifest, err := readManifest(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &hostagentpb.VMObservedState{Id: id, Status: stateStopped}
		}
		return observedError(id, err.Error())
	}
	if id == "" {
		id = manifest.ID
	}
	if manifest.PID == 0 || !processAlive(manifest.PID) {
		_ = updateManifest(manifestPath, map[string]any{
			manifestPIDField:    0,
			manifestStatusField: stateStopped,
		})
		s.untrackActiveVMLocked(manifestPath)
		manifest.PID = 0
		manifest.Status = stateStopped
		return observedFromManifest(manifestPath, manifest, "")
	}

	_ = updateManifest(manifestPath, map[string]any{manifestStatusField: stateChanging})
	process, err := os.FindProcess(manifest.PID)
	if err != nil {
		return observedError(id, fmt.Sprintf("find VM process %d: %v", manifest.PID, err))
	}
	if err := process.Signal(syscall.SIGTERM); err != nil && processAlive(manifest.PID) {
		return observedError(id, fmt.Sprintf("stop VM process %d: %v", manifest.PID, err))
	}
	if !waitForProcessExit(manifest.PID, gracefulStopTimeout) {
		if err := process.Signal(syscall.SIGTERM); err != nil && processAlive(manifest.PID) {
			return observedError(id, fmt.Sprintf("force stop VM process %d: %v", manifest.PID, err))
		}
	}
	if !waitForProcessExit(manifest.PID, forcedStopTimeout) {
		if err := process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			return observedError(id, fmt.Sprintf("kill VM process %d: %v", manifest.PID, err))
		}
		_ = waitForProcessExit(manifest.PID, forcedStopTimeout)
	}

	if err := updateManifest(manifestPath, map[string]any{
		manifestPIDField:    0,
		manifestStatusField: stateStopped,
	}); err != nil {
		return observedError(id, err.Error())
	}
	s.untrackActiveVMLocked(manifestPath)
	manifest.PID = 0
	manifest.Status = stateStopped
	return observedFromManifest(manifestPath, manifest, "")
}

func (s *Server) vmStatus(ref *hostagentpb.VMRef) *hostagentpb.VMObservedState {
	id := strings.TrimSpace(ref.GetId())
	manifestPath, manifest, err := resolveVMManifest(ref)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &hostagentpb.VMObservedState{Id: id, Status: stateStopped}
		}
		return observedError(id, err.Error())
	}
	return s.observedVMStatus(manifestPath, manifest)
}

func (s *Server) observedVMStatus(manifestPath string, manifest vmManifest) *hostagentpb.VMObservedState {
	current, err := readManifest(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &hostagentpb.VMObservedState{Id: manifest.ID, Status: stateStopped}
		}
		return observedError(manifest.ID, err.Error())
	}
	manifest = current
	if manifest.PID > 0 && processAlive(manifest.PID) {
		if manifest.Status != stateRunning {
			_ = updateManifest(manifestPath, map[string]any{manifestStatusField: stateRunning})
			manifest.Status = stateRunning
		}
		s.trackActiveVMLocked(manifestPath)
		return observedFromManifest(manifestPath, manifest, "")
	}
	if manifest.PID != 0 || manifest.Status == stateRunning || manifest.Status == stateChanging {
		_ = updateManifest(manifestPath, map[string]any{
			manifestPIDField:    0,
			manifestStatusField: stateStopped,
		})
		s.untrackActiveVMLocked(manifestPath)
		manifest.PID = 0
		manifest.Status = stateStopped
	}
	return observedFromManifest(manifestPath, manifest, "")
}

func (s *Server) stopActiveVMsLocked() error {
	if len(s.activeVMs) == 0 {
		return nil
	}
	paths := make([]string, 0, len(s.activeVMs))
	for manifestPath := range s.activeVMs {
		paths = append(paths, manifestPath)
	}
	sort.Strings(paths)

	var err error
	for _, manifestPath := range paths {
		state := s.stopVM("", manifestPath)
		if state.GetStatus() == stateError {
			err = errors.Join(err, fmt.Errorf("stop VM %s: %s", state.GetId(), state.GetMessage()))
		}
	}
	return err
}

func (s *Server) trackActiveVMLocked(manifestPath string) {
	manifestPath = strings.TrimSpace(manifestPath)
	if manifestPath == "" {
		return
	}
	if s.activeVMs == nil {
		s.activeVMs = map[string]struct{}{}
	}
	s.activeVMs[manifestPath] = struct{}{}
}

func (s *Server) untrackActiveVMLocked(manifestPath string) {
	if s.activeVMs == nil {
		return
	}
	delete(s.activeVMs, manifestPath)
}

func (s *Server) listVMs(rootDir string) []*hostagentpb.VMObservedState {
	rootDir = strings.TrimSpace(rootDir)
	if rootDir == "" {
		return []*hostagentpb.VMObservedState{observedError("", "root_dir is required")}
	}
	entries, err := os.ReadDir(hostAgentVMInstancesDir(rootDir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return []*hostagentpb.VMObservedState{observedError("", err.Error())}
	}
	states := make([]*hostagentpb.VMObservedState, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := hostAgentVMManifestPath(rootDir, entry.Name())
		manifest, err := readManifest(manifestPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			states = append(states, observedError(entry.Name(), err.Error()))
			continue
		}
		states = append(states, s.observedVMStatus(manifestPath, manifest))
	}
	return states
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
			peers, err := networkPeersFromProto(desired.GetInstances(), desired.GetDevices(), desired.GetExitRoutes())
			if err != nil {
				return s.networkStatus(err.Error())
			}
			if err := s.network.ConfigurePeers(config, peers); err != nil {
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
			peers, err := networkPeersFromProto(desired.GetInstances(), desired.GetDevices(), desired.GetExitRoutes())
			if err != nil {
				return s.networkStatus(err.Error())
			}
			if err := s.network.ConfigurePeers(config, peers); err != nil {
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
	var state *hostagentpb.NetworkState
	if s.network != nil {
		moduleName = s.network.Name()
		if observed, err := s.network.State(); err == nil {
			state = networkStateToProto(observed)
		} else if message == "" {
			message = fmt.Sprintf("inspect network state: %v", err)
		}
	}
	return &hostagentpb.NetworkObservedState{
		Module:  moduleName,
		Up:      s.networkUp,
		Message: message,
		State:   state,
	}
}

func networkStateToProto(state networkmodule.State) *hostagentpb.NetworkState {
	out := &hostagentpb.NetworkState{
		Module:        state.Module,
		Up:            state.Up,
		InterfaceName: state.InterfaceName,
		Messages:      append([]string(nil), state.Messages...),
	}
	for _, item := range state.Interfaces {
		out.Interfaces = append(out.Interfaces, &hostagentpb.NetworkInterface{
			Name:       item.Name,
			Type:       item.Type,
			Index:      int32(item.Index),
			Mtu:        int32(item.MTU),
			Up:         item.Up,
			Master:     item.Master,
			MacAddress: item.MacAddress,
			Kind:       item.Kind,
		})
	}
	for _, item := range state.Addresses {
		out.Addresses = append(out.Addresses, &hostagentpb.NetworkAddress{
			InterfaceName: item.InterfaceName,
			Cidr:          item.CIDR,
			Scope:         item.Scope,
		})
	}
	for _, item := range state.Routes {
		out.Routes = append(out.Routes, &hostagentpb.NetworkRoute{
			InterfaceName: item.InterfaceName,
			Destination:   item.Destination,
			Gateway:       item.Gateway,
			Source:        item.Source,
			Family:        item.Family,
			Table:         item.Table,
			Protocol:      item.Protocol,
			Scope:         item.Scope,
			Priority:      item.Priority,
			Kind:          item.Kind,
		})
	}
	for _, item := range state.WireGuardPeers {
		out.WireguardPeers = append(out.WireguardPeers, &hostagentpb.WireGuardPeer{
			PublicKey:       item.PublicKey,
			Endpoint:        item.Endpoint,
			AllowedIps:      append([]string(nil), item.AllowedIPs...),
			LatestHandshake: item.LatestHandshake,
			RxBytes:         item.RxBytes,
			TxBytes:         item.TxBytes,
		})
	}
	for _, table := range state.FirewallTables {
		tableProto := &hostagentpb.FirewallTable{
			Family: table.Family,
			Name:   table.Name,
		}
		for _, chain := range table.Chains {
			chainProto := &hostagentpb.FirewallChain{
				Name:     chain.Name,
				Type:     chain.Type,
				Hook:     chain.Hook,
				Priority: chain.Priority,
			}
			for _, rule := range chain.Rules {
				chainProto.Rules = append(chainProto.Rules, &hostagentpb.FirewallRule{
					Expressions: append([]string(nil), rule.Expressions...),
					Packets:     rule.Packets,
					Bytes:       rule.Bytes,
				})
			}
			tableProto.Chains = append(tableProto.Chains, chainProto)
		}
		out.FirewallTables = append(out.FirewallTables, tableProto)
	}
	for _, item := range state.DNS {
		out.Dns = append(out.Dns, &hostagentpb.DNSState{
			Scope:   item.Scope,
			Domain:  item.Domain,
			Servers: append([]string(nil), item.Servers...),
			Port:    int32(item.Port),
			Active:  item.Active,
			Source:  item.Source,
		})
	}
	return out
}

func hasNetworkPeers(desired *hostagentpb.NetworkDesiredState) bool {
	return len(desired.GetInstances()) > 0 || len(desired.GetDevices()) > 0 || len(desired.GetExitRoutes()) > 0
}

func networkConfigFromProto(config *hostagentpb.NetworkConfig) (networkmodule.Config, error) {
	if config == nil {
		return networkmodule.Config{}, fmt.Errorf("network config is required")
	}

	addr, err := netip.ParseAddr(config.GetIpv6Address())
	if err != nil {
		return networkmodule.Config{}, fmt.Errorf("invalid IPv6 address %q: %w", config.GetIpv6Address(), err)
	}
	ipv4Addr := netip.Addr{}
	if config.GetIpv4Address() != "" {
		ipv4Addr, err = netip.ParseAddr(config.GetIpv4Address())
		if err != nil {
			return networkmodule.Config{}, fmt.Errorf("invalid IPv4 address %q: %w", config.GetIpv4Address(), err)
		}
	}

	return networkmodule.Config{
		IPv6Address:         addr,
		IPv4Address:         ipv4Addr,
		LocalPeerID:         config.GetLocalPeerId(),
		WireGuardPrivateKey: config.GetWireguardPrivateKey(),
		Domain:              config.GetDomain(),
	}, nil
}

func networkPeersFromProto(instances []*hostagentpb.InstancePeer, devices []*hostagentpb.DevicePeer, exitRoutes []*hostagentpb.ExitRoute) (networkmodule.Peers, error) {
	peers := networkmodule.Peers{
		Instances:  make([]networkmodule.InstancePeer, 0, len(instances)),
		Devices:    make([]networkmodule.DevicePeer, 0, len(devices)),
		ExitRoutes: make([]networkmodule.ExitRoute, 0, len(exitRoutes)),
	}

	for _, instance := range instances {
		ipv4Addr := netip.Addr{}
		if instance.GetIpv4Address() != "" {
			addr, err := netip.ParseAddr(instance.GetIpv4Address())
			if err != nil {
				return networkmodule.Peers{}, fmt.Errorf("invalid IPv4 address %q for instance peer %q: %w", instance.GetIpv4Address(), instance.GetName(), err)
			}
			ipv4Addr = addr
		}
		routes := make([]netip.Addr, 0, len(instance.GetRoutes()))
		for _, route := range instance.GetRoutes() {
			addr, err := netip.ParseAddr(route)
			if err != nil {
				return networkmodule.Peers{}, fmt.Errorf("invalid route %q for instance peer %q: %w", route, instance.GetName(), err)
			}
			routes = append(routes, addr)
		}
		peers.Instances = append(peers.Instances, networkmodule.InstancePeer{
			ID:          instance.GetId(),
			Name:        instance.GetName(),
			PublicKey:   instance.GetPublicKey(),
			PublicIP:    instance.GetPublicIp(),
			IPv4Address: ipv4Addr,
			Routes:      routes,
		})
	}

	for _, device := range devices {
		ipv4Addr := netip.Addr{}
		if device.GetIpv4Address() != "" {
			addr, err := netip.ParseAddr(device.GetIpv4Address())
			if err != nil {
				return networkmodule.Peers{}, fmt.Errorf("invalid IPv4 address %q for device peer %q: %w", device.GetIpv4Address(), device.GetName(), err)
			}
			ipv4Addr = addr
		}
		peers.Devices = append(peers.Devices, networkmodule.DevicePeer{
			ID:          device.GetId(),
			Name:        device.GetName(),
			PublicKey:   device.GetPublicKey(),
			IPv4Address: ipv4Addr,
		})
	}

	for _, route := range exitRoutes {
		peers.ExitRoutes = append(peers.ExitRoutes, networkmodule.ExitRoute{
			ID:         route.GetId(),
			DeviceID:   route.GetDeviceId(),
			InstanceID: route.GetInstanceId(),
			CIDRs:      route.GetCidrs(),
		})
	}

	return peers, nil
}

type vmManifest struct {
	ID                  string          `json:"id"`
	Name                string          `json:"name"`
	ImageID             string          `json:"image_id"`
	Location            string          `json:"location"`
	MachineType         string          `json:"machine_type"`
	Cores               uint32          `json:"cores"`
	MemoryMiB           uint32          `json:"memory_mib"`
	InitOriginPublicKey string          `json:"init_origin_public_key"`
	PublicIP            string          `json:"public_ip"`
	MACAddress          string          `json:"mac_address"`
	Status              string          `json:"status"`
	PID                 int             `json:"pid,omitempty"`
	KernelPath          string          `json:"kernel_path"`
	InitrdPath          string          `json:"initrd_path"`
	CmdlinePath         string          `json:"cmdline_path"`
	RootDiskPath        string          `json:"root_disk_path,omitempty"`
	BootISOPath         string          `json:"boot_iso_path,omitempty"`
	MetadataISO         string          `json:"metadata_iso,omitempty"`
	Network             vmNetworkConfig `json:"network,omitempty"`
	Volumes             []vmVolume      `json:"volumes,omitempty"`
}

type vmNetworkConfig struct {
	Interface    string   `json:"interface"`
	IPAddress    string   `json:"ip_address"`
	PrefixLength int32    `json:"prefix_length"`
	Gateway      string   `json:"gateway"`
	DNSServers   []string `json:"dns_servers,omitempty"`
}

type vmVolume struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	SizeMiB int32  `json:"size_mib"`
}

func readManifest(path string) (vmManifest, error) {
	var m vmManifest
	data, err := os.ReadFile(path)
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return m, fmt.Errorf("decode VM manifest %s: %w", path, err)
	}
	return m, nil
}

func writeConfiguredVMManifest(rootDir string, config *hostagentpb.VMConfig) (string, vmManifest, error) {
	manifestPath, err := manifestPathForConfig(rootDir, config)
	if err != nil {
		return "", vmManifest{}, err
	}
	manifest := manifestFromProto(config)
	if manifest.ID == "" {
		return "", vmManifest{}, fmt.Errorf("VM id is required")
	}
	existing, err := readManifest(manifestPath)
	if err == nil {
		manifest.PID = existing.PID
		manifest.Status = existing.Status
	} else if !os.IsNotExist(err) {
		return "", vmManifest{}, err
	}
	if manifest.Status == "" {
		manifest.Status = stateStopped
	}
	if err := writeManifest(manifestPath, manifest); err != nil {
		return "", vmManifest{}, err
	}
	return manifestPath, manifest, nil
}

func writeManifest(path string, manifest vmManifest) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(filepath.Dir(path), ".manifest-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if info, err := os.Stat(path); err == nil {
		if err := preserveManifestMetadata(tmpPath, info); err != nil {
			_ = os.Remove(tmpPath)
			return err
		}
	} else if os.IsNotExist(err) {
		if err := os.Chmod(tmpPath, 0644); err != nil {
			_ = os.Remove(tmpPath)
			return err
		}
	} else {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func resolveVMManifest(ref *hostagentpb.VMRef) (string, vmManifest, error) {
	rootDir := strings.TrimSpace(ref.GetRootDir())
	if rootDir == "" {
		return "", vmManifest{}, fmt.Errorf("root_dir is required")
	}
	if id := strings.TrimSpace(ref.GetId()); id != "" {
		manifestPath := hostAgentVMManifestPath(rootDir, id)
		manifest, err := readManifest(manifestPath)
		return manifestPath, manifest, err
	}
	name := strings.TrimSpace(ref.GetName())
	if name == "" {
		return "", vmManifest{}, fmt.Errorf("VM id or name is required")
	}
	entries, err := os.ReadDir(hostAgentVMInstancesDir(rootDir))
	if err != nil {
		return "", vmManifest{}, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := hostAgentVMManifestPath(rootDir, entry.Name())
		manifest, err := readManifest(manifestPath)
		if err != nil {
			continue
		}
		if manifest.Name == name {
			return manifestPath, manifest, nil
		}
	}
	return "", vmManifest{}, os.ErrNotExist
}

func manifestPathForConfig(rootDir string, config *hostagentpb.VMConfig) (string, error) {
	if config == nil {
		return "", fmt.Errorf("VM config is required")
	}
	id, err := cleanVMID(config.GetId())
	if err != nil {
		return "", err
	}
	if artifactDir := strings.TrimSpace(config.GetArtifactDir()); artifactDir != "" {
		return filepath.Join(artifactDir, "manifest.json"), nil
	}
	rootDir = strings.TrimSpace(rootDir)
	if rootDir == "" {
		return "", fmt.Errorf("root_dir or config.artifact_dir is required")
	}
	return hostAgentVMManifestPath(rootDir, id), nil
}

func hostAgentVMInstancesDir(rootDir string) string {
	return filepath.Join(rootDir, "instances")
}

func hostAgentVMManifestPath(rootDir string, id string) string {
	cleanID, err := cleanVMID(id)
	if err != nil {
		cleanID = strings.TrimSpace(id)
	}
	return filepath.Join(hostAgentVMInstancesDir(rootDir), cleanID, "manifest.json")
}

func cleanVMID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("VM id is required")
	}
	if id == "." || id == ".." || strings.ContainsAny(id, `/\`) {
		return "", fmt.Errorf("invalid VM id %q", id)
	}
	return id, nil
}

func manifestFromProto(config *hostagentpb.VMConfig) vmManifest {
	if config == nil {
		return vmManifest{}
	}
	manifest := vmManifest{
		ID:                  strings.TrimSpace(config.GetId()),
		Name:                strings.TrimSpace(config.GetName()),
		ImageID:             strings.TrimSpace(config.GetImageId()),
		Location:            strings.TrimSpace(config.GetLocation()),
		MachineType:         strings.TrimSpace(config.GetMachineType()),
		Cores:               config.GetCores(),
		MemoryMiB:           config.GetMemoryMib(),
		InitOriginPublicKey: strings.TrimSpace(config.GetInitOriginPublicKey()),
		PublicIP:            strings.TrimSpace(config.GetPublicIp()),
		MACAddress:          strings.TrimSpace(config.GetMacAddress()),
		KernelPath:          strings.TrimSpace(config.GetKernelPath()),
		InitrdPath:          strings.TrimSpace(config.GetInitrdPath()),
		CmdlinePath:         strings.TrimSpace(config.GetCmdlinePath()),
		RootDiskPath:        strings.TrimSpace(config.GetRootDiskPath()),
		BootISOPath:         strings.TrimSpace(config.GetBootIsoPath()),
		MetadataISO:         strings.TrimSpace(config.GetMetadataIso()),
	}
	if network := config.GetNetwork(); network != nil {
		manifest.Network = vmNetworkConfig{
			Interface:    strings.TrimSpace(network.GetInterface()),
			IPAddress:    strings.TrimSpace(network.GetIpAddress()),
			PrefixLength: network.GetPrefixLength(),
			Gateway:      strings.TrimSpace(network.GetGateway()),
			DNSServers:   append([]string(nil), network.GetDnsServers()...),
		}
	}
	for _, volume := range config.GetVolumes() {
		if volume == nil {
			continue
		}
		manifest.Volumes = append(manifest.Volumes, vmVolume{
			ID:      strings.TrimSpace(volume.GetId()),
			Name:    strings.TrimSpace(volume.GetName()),
			Path:    strings.TrimSpace(volume.GetPath()),
			SizeMiB: volume.GetSizeMib(),
		})
	}
	return manifest
}

func protoFromManifest(manifestPath string, manifest vmManifest) *hostagentpb.VMConfig {
	config := &hostagentpb.VMConfig{
		Id:                  manifest.ID,
		Name:                manifest.Name,
		ImageId:             manifest.ImageID,
		Location:            manifest.Location,
		MachineType:         manifest.MachineType,
		Cores:               manifest.Cores,
		MemoryMib:           manifest.MemoryMiB,
		InitOriginPublicKey: manifest.InitOriginPublicKey,
		PublicIp:            manifest.PublicIP,
		MacAddress:          manifest.MACAddress,
		KernelPath:          manifest.KernelPath,
		InitrdPath:          manifest.InitrdPath,
		CmdlinePath:         manifest.CmdlinePath,
		RootDiskPath:        manifest.RootDiskPath,
		BootIsoPath:         manifest.BootISOPath,
		MetadataIso:         manifest.MetadataISO,
		ArtifactDir:         filepath.Dir(manifestPath),
	}
	config.Network = &hostagentpb.VMNetworkConfig{
		Interface:    manifest.Network.Interface,
		IpAddress:    manifest.Network.IPAddress,
		PrefixLength: manifest.Network.PrefixLength,
		Gateway:      manifest.Network.Gateway,
		DnsServers:   append([]string(nil), manifest.Network.DNSServers...),
	}
	for _, volume := range manifest.Volumes {
		config.Volumes = append(config.Volumes, &hostagentpb.VMVolume{
			Id:      volume.ID,
			Name:    volume.Name,
			Path:    volume.Path,
			SizeMib: volume.SizeMiB,
		})
	}
	return config
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
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

func observedFromManifest(manifestPath string, manifest vmManifest, message string) *hostagentpb.VMObservedState {
	return &hostagentpb.VMObservedState{
		Id:       manifest.ID,
		Status:   manifest.Status,
		Pid:      int32(manifest.PID),
		PublicIp: manifest.PublicIP,
		Message:  message,
		Config:   protoFromManifest(manifestPath, manifest),
	}
}

func observedError(id string, message string) *hostagentpb.VMObservedState {
	return &hostagentpb.VMObservedState{
		Id:      id,
		Status:  stateError,
		Message: message,
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
