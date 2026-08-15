//go:build darwin

package e2eapic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/protosio/protos/internal/pcrypto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const ProbeAppPort = 8080

type FlutterNodeOptions struct {
	WorkDir          string
	AppPath          string
	HostAgentBin     string
	ConfigureNetwork bool
	StartupTimeout   time.Duration
}

type FlutterNode struct {
	DataDir    string
	SocketPath string
	AppPath    string
	LogPath    string
	Client     pbApic.ProtosClientApiClient

	conn   *grpc.ClientConn
	cmd    *exec.Cmd
	exitCh chan error
}

type ReplicationExpectation struct {
	Label      string
	Priorities map[string]int
}

type ProbeAppResponse struct {
	OK             bool   `json:"ok"`
	ID             string `json:"id"`
	Target         string `json:"target"`
	StatusCode     int    `json:"status_code"`
	BytesRead      int    `json:"bytes_read"`
	DurationMillis int64  `json:"duration_ms"`
	Error          string `json:"error"`
}

func StartFlutterNode(opts FlutterNodeOptions) (*FlutterNode, error) {
	if strings.TrimSpace(opts.WorkDir) == "" {
		return nil, fmt.Errorf("workdir is required")
	}
	timeout := opts.StartupTimeout
	if timeout <= 0 {
		timeout = 45 * time.Second
	}
	dataDir := filepath.Join(opts.WorkDir, "flutter-node")
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, err
	}
	socketPath := filepath.Join(dataDir, "protos.socket")
	if len(socketPath) > 100 {
		return nil, fmt.Errorf("APIC socket path is too long for macOS (%d bytes): %s", len(socketPath), socketPath)
	}
	_ = os.Remove(socketPath)

	appPath, err := ResolveMacOSApp(opts.AppPath)
	if err != nil {
		return nil, err
	}
	executable, err := macOSAppExecutable(appPath)
	if err != nil {
		return nil, err
	}
	logPath := filepath.Join(opts.WorkDir, "flutter-node.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return nil, err
	}

	capabilities := "default"
	if !opts.ConfigureNetwork {
		capabilities = "default,no-network"
	}
	hostAgentBin := strings.TrimSpace(opts.HostAgentBin)
	if hostAgentBin == "" {
		hostAgentBin = defaultHostAgentBin()
	}
	if hostAgentBin != "" {
		if abs, absErr := filepath.Abs(hostAgentBin); absErr == nil {
			hostAgentBin = abs
		}
	}

	cmd := exec.Command(executable)
	cmd.Dir = filepath.Dir(executable)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Env = append(os.Environ(),
		"PROTOS_FLUTTER_DATA_DIR="+dataDir,
		"PROTOS_FLUTTER_CONFIG_FILE="+filepath.Join(dataDir, "protos.yaml"),
		"PROTOS_FLUTTER_CAPABILITIES="+capabilities,
		"PROTOS_FLUTTER_LOG_LEVEL=debug",
	)
	if hostAgentBin != "" {
		cmd.Env = append(cmd.Env, "PROTOS_HOSTAGENT_BIN="+hostAgentBin)
	}
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, err
	}
	_ = logFile.Close()

	node := &FlutterNode{
		DataDir:    dataDir,
		SocketPath: socketPath,
		AppPath:    appPath,
		LogPath:    logPath,
		cmd:        cmd,
		exitCh:     make(chan error, 1),
	}
	go func() {
		node.exitCh <- cmd.Wait()
	}()
	if err := node.waitForAPI(timeout); err != nil {
		_ = node.Stop()
		return nil, err
	}
	return node, nil
}

func (n *FlutterNode) Stop() error {
	if n == nil {
		return nil
	}
	if n.conn != nil {
		_ = n.conn.Close()
	}
	if n.cmd == nil || n.cmd.Process == nil {
		return nil
	}
	select {
	case <-n.exitCh:
		return nil
	default:
	}
	_ = n.cmd.Process.Signal(os.Interrupt)
	select {
	case <-n.exitCh:
		return nil
	case <-time.After(5 * time.Second):
		_ = n.cmd.Process.Kill()
		<-n.exitCh
		return nil
	}
}

func (n *FlutterNode) waitForAPI(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		select {
		case err := <-n.exitCh:
			return fmt.Errorf("Flutter node exited before API became ready: %w; log tail:\n%s", err, logTail(n.LogPath))
		default:
		}
		if _, err := os.Stat(n.SocketPath); err != nil {
			lastErr = err
			if failure := startupFailureLog(n.LogPath); failure != "" {
				return fmt.Errorf("Flutter node failed before API became ready: %s; log tail:\n%s", failure, logTail(n.LogPath))
			}
			time.Sleep(500 * time.Millisecond)
			continue
		}
		conn, err := grpc.NewClient("unix://"+n.SocketPath, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			lastErr = err
			time.Sleep(500 * time.Millisecond)
			continue
		}
		client := pbApic.NewProtosClientApiClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_, err = client.GetSystemStatus(ctx, &pbApic.GetSystemStatusRequest{})
		cancel()
		if err == nil {
			n.conn = conn
			n.Client = client
			return nil
		}
		_ = conn.Close()
		lastErr = err
		if failure := startupFailureLog(n.LogPath); failure != "" {
			return fmt.Errorf("Flutter node failed before API became ready: %s; log tail:\n%s", failure, logTail(n.LogPath))
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("Flutter node API did not become ready at %s: %w; log tail:\n%s", n.SocketPath, lastErr, logTail(n.LogPath))
}

func ResolveMacOSApp(value string) (string, error) {
	if strings.TrimSpace(value) != "" {
		abs, err := filepath.Abs(value)
		if err != nil {
			return "", err
		}
		if info, err := os.Stat(abs); err == nil && info.IsDir() {
			return abs, nil
		}
		return "", fmt.Errorf("macOS app bundle not found at %s", abs)
	}
	if env := strings.TrimSpace(os.Getenv("PROTOS_FLUTTER_APP_PATH")); env != "" {
		return ResolveMacOSApp(env)
	}
	coreDir := coreDir()
	patterns := []string{
		filepath.Join(coreDir, "..", "clients", "macos", "build", "macos", "Build", "Products", "Debug", "*.app"),
		filepath.Join(coreDir, "..", "clients", "macos", "build", "macos", "Build", "Products", "Profile", "*.app"),
		filepath.Join(coreDir, "..", "clients", "macos", "build", "macos", "Build", "Products", "Release", "*.app"),
	}
	var matches []string
	for _, pattern := range patterns {
		found, err := filepath.Glob(pattern)
		if err != nil {
			return "", err
		}
		matches = append(matches, found...)
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("built macOS Flutter app was not found; run task -t clients/macos/Taskfile.yml build")
	}
	sort.Slice(matches, func(i, j int) bool {
		left, _ := os.Stat(matches[i])
		right, _ := os.Stat(matches[j])
		if left == nil || right == nil {
			return matches[i] < matches[j]
		}
		return left.ModTime().After(right.ModTime())
	})
	return matches[0], nil
}

func macOSAppExecutable(appPath string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(appPath, "Contents", "MacOS", "*"))
	if err != nil {
		return "", err
	}
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil || info.IsDir() || info.Mode()&0111 == 0 {
			continue
		}
		return match, nil
	}
	return "", fmt.Errorf("no executable found in %s", filepath.Join(appPath, "Contents", "MacOS"))
}

func Init(ctx context.Context, client pbApic.ProtosClientApiClient, username string, name string, organisation string) error {
	_, err := client.Init(ctx, &pbApic.InitRequest{
		Username:     username,
		Name:         name,
		Organisation: organisation,
	})
	return err
}

func StartHostAgent(ctx context.Context, client pbApic.ProtosClientApiClient) error {
	_, err := client.StartHostAgent(ctx, &pbApic.StartHostAgentRequest{})
	return err
}

func WaitForInstanceReady(deadline time.Time, client pbApic.ProtosClientApiClient, name string) (*pbApic.CloudInstance, error) {
	var lastStatus string
	for time.Now().Before(deadline) {
		ctx, cancel := contextWithDeadline(deadline, 10*time.Second)
		resp, err := client.GetInstance(ctx, &pbApic.GetInstanceRequest{Name: name})
		cancel()
		if err == nil && resp.GetInstance() != nil {
			instance := resp.GetInstance()
			lastStatus = instance.GetStatus()
			if strings.TrimSpace(instance.GetPublicKey()) != "" {
				return instance, nil
			}
			if terminalStatus(instance.GetStatus()) {
				return nil, fmt.Errorf("deployment for %s ended with status %q", name, instance.GetStatus())
			}
		} else if err != nil {
			lastStatus = err.Error()
		}
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("instance %s did not become ready before deadline; last status: %s", name, lastStatus)
}

func WaitForInstanceAbsent(deadline time.Time, client pbApic.ProtosClientApiClient, instance *pbApic.CloudInstance) error {
	var lastErr error
	for time.Now().Before(deadline) {
		instances, err := ListInstances(deadline, client)
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
			continue
		}
		if !containsInstance(instances, instance.GetName()) && !containsInstance(instances, instance.GetVmId()) {
			return nil
		}
		lastErr = fmt.Errorf("instance still present: %s/%s", instance.GetName(), instance.GetVmId())
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("instance %s remained after deletion: %w", instance.GetName(), lastErr)
}

func WaitForInstanceStatus(deadline time.Time, client pbApic.ProtosClientApiClient, name string, status string) (*pbApic.CloudInstance, error) {
	var lastStatus string
	for time.Now().Before(deadline) {
		ctx, cancel := contextWithDeadline(deadline, 10*time.Second)
		resp, err := client.GetInstance(ctx, &pbApic.GetInstanceRequest{Name: name})
		cancel()
		if err == nil && resp.GetInstance() != nil {
			instance := resp.GetInstance()
			lastStatus = instance.GetStatus()
			if strings.EqualFold(strings.TrimSpace(instance.GetStatus()), strings.TrimSpace(status)) {
				return instance, nil
			}
		} else if err != nil {
			lastStatus = err.Error()
		}
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("instance %s did not reach status %q before deadline; last status: %s", name, status, lastStatus)
}

func WaitForNoInstances(deadline time.Time, client pbApic.ProtosClientApiClient) error {
	var lastErr error
	for time.Now().Before(deadline) {
		instances, err := ListInstances(deadline, client)
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
			continue
		}
		if len(instances) == 0 {
			return nil
		}
		lastErr = fmt.Errorf("instances still present: %v", InstanceNames(instances))
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("instances remained after e2e deprovision: %w", lastErr)
}

func ListInstances(deadline time.Time, client pbApic.ProtosClientApiClient) ([]*pbApic.CloudInstance, error) {
	ctx, cancel := contextWithDeadline(deadline, 20*time.Second)
	defer cancel()
	resp, err := client.GetInstances(ctx, &pbApic.GetInstancesRequest{})
	if err != nil {
		return nil, err
	}
	return resp.GetInstances(), nil
}

func WaitForRemoteFullMesh(deadline time.Time, client pbApic.ProtosClientApiClient, instances []*pbApic.CloudInstance) error {
	peerIDs, err := peerIDsByInstanceName(instances)
	if err != nil {
		return err
	}
	var lastErr error
	var lastReport time.Time
	for time.Now().Before(deadline) {
		ready := true
		for _, source := range instances {
			sourcePeerID := peerIDs[source.GetName()]
			state, err := RuntimeState(deadline, client, source.GetName())
			if err != nil {
				lastErr = fmt.Errorf("get remote runtime state for %s: %w", source.GetName(), err)
				ready = false
				break
			}
			fmt.Printf("remote runtime peers for %s: physical=%v routed=%v participating=%v logical=%v\n", source.GetName(), state.GetPhysicalConnectedPeers(), state.GetRoutedPeers(), state.GetParticipatingPeers(), state.GetLogicalPeers())
			for _, target := range instances {
				targetPeerID := peerIDs[target.GetName()]
				if targetPeerID == sourcePeerID {
					continue
				}
				if transportReady, missing := runtimePeerTransportReady(state, targetPeerID); !transportReady {
					lastErr = fmt.Errorf("%s runtime transport is not ready for %s (%s), missing %s: %s", source.GetName(), target.GetName(), targetPeerID, strings.Join(missing, ", "), RuntimeStateSummary(state))
					ready = false
					break
				}
			}
			if !ready {
				break
			}
		}
		if ready {
			return nil
		}
		if time.Since(lastReport) >= 30*time.Second {
			lastReport = time.Now()
			fmt.Printf("waiting for VM runtime full mesh: %v\n", lastErr)
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("VM full mesh did not become connected: %w", lastErr)
}

func WaitForRemotePeerConnection(deadline time.Time, client pbApic.ProtosClientApiClient, instances []*pbApic.CloudInstance, peerID string) error {
	var lastErr error
	var lastReport time.Time
	for time.Now().Before(deadline) {
		ready := true
		for _, source := range instances {
			state, err := RuntimeState(deadline, client, source.GetName())
			if err != nil {
				ready = false
				lastErr = fmt.Errorf("get remote runtime state for %s: %w", source.GetName(), err)
				break
			}
			fmt.Printf("remote runtime peers for %s: physical=%v routed=%v participating=%v logical=%v\n", source.GetName(), state.GetPhysicalConnectedPeers(), state.GetRoutedPeers(), state.GetParticipatingPeers(), state.GetLogicalPeers())
			if transportReady, missing := runtimePeerTransportReady(state, peerID); !transportReady {
				ready = false
				lastErr = fmt.Errorf("%s runtime transport is not ready for peer id %s, missing %s: %s", source.GetName(), peerID, strings.Join(missing, ", "), RuntimeStateSummary(state))
				break
			}
		}
		if ready {
			return nil
		}
		if time.Since(lastReport) >= 30*time.Second {
			lastReport = time.Now()
			fmt.Printf("waiting for remote runtime peer connection: %v\n", lastErr)
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("remote VM peer connection to %s did not become connected: %w", peerID, lastErr)
}

// WaitForLocalPeerConnections proves the other half of the NAT topology: the
// local origin sees every VM on the shared application-owned transport and in
// the Swarmion database scope. Pair this with WaitForRemotePeerConnection,
// which proves every VM sees the local origin over the same duplex connection.
func WaitForLocalPeerConnections(deadline time.Time, client pbApic.ProtosClientApiClient, instances []*pbApic.CloudInstance) error {
	peerIDs, err := peerIDsByInstanceName(instances)
	if err != nil {
		return err
	}
	var lastErr error
	var lastReport time.Time
	for time.Now().Before(deadline) {
		state, err := RuntimeState(deadline, client, "")
		if err != nil {
			lastErr = fmt.Errorf("get local runtime state: %w", err)
			time.Sleep(3 * time.Second)
			continue
		}
		missing := make([]string, 0)
		for _, instance := range instances {
			peerID := peerIDs[instance.GetName()]
			if ready, missingPlanes := runtimePeerTransportReady(state, peerID); !ready {
				missing = append(missing, instance.GetName()+" ("+peerID+": "+strings.Join(missingPlanes, ", ")+")")
			}
		}
		if len(missing) == 0 {
			return nil
		}
		lastErr = fmt.Errorf("local runtime does not see VM peers connected: %s; %s", strings.Join(missing, ", "), RuntimeStateSummary(state))
		if time.Since(lastReport) >= 30*time.Second {
			lastReport = time.Now()
			fmt.Printf("waiting for local outbound runtime connections: %v\n", lastErr)
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("local outbound runtime transport did not become ready: %w", lastErr)
}

func WaitForReplicationState(deadline time.Time, client pbApic.ProtosClientApiClient, expect ReplicationExpectation) error {
	var lastErr error
	var lastReport time.Time
	for time.Now().Before(deadline) {
		state, err := RuntimeState(deadline, client, "")
		if err != nil {
			lastErr = err
			reportReplicationProgress(&lastReport, expect.Label, nil, lastErr)
			time.Sleep(3 * time.Second)
			continue
		}
		if fatal := strings.TrimSpace(state.GetFatalState()); fatal != "" && fatal != "none" {
			return fmt.Errorf("runtime fatal state while waiting for %s: %s", expect.Label, fatal)
		}
		if err := AssertNoBlockingCompatibility(state); err != nil {
			return err
		}
		if err := AssertReplicationPriorities(state, expect.Priorities); err != nil {
			lastErr = err
			reportReplicationProgress(&lastReport, expect.Label, state, lastErr)
			time.Sleep(3 * time.Second)
			continue
		}
		checkpoint := state.GetCheckpointRootHash()
		durable := state.GetDurableMainRootHash()
		if checkpoint != "" && state.GetTentativeRootHash() == checkpoint && (durable == "" || durable == checkpoint) {
			fmt.Printf("checkpoint assertion ok: %s checkpoint=%s providers=%v routed=%v priority_peers=%v\n", expect.Label, checkpoint, state.GetStateProviders(), state.GetRoutedPeers(), sortedPeerPriorityKeys(expect.Priorities))
			return nil
		}
		lastErr = fmt.Errorf("checkpoint state did not converge for %q: checkpoint=%s tentative=%s durable=%s", expect.Label, checkpoint, state.GetTentativeRootHash(), durable)
		reportReplicationProgress(&lastReport, expect.Label, state, lastErr)
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("timed out waiting for checkpoint state %q: %w", expect.Label, lastErr)
}

func WaitForPeerRemoved(deadline time.Time, client pbApic.ProtosClientApiClient, peerID string) error {
	var lastErr error
	var lastReport time.Time
	for time.Now().Before(deadline) {
		state, err := RuntimeState(deadline, client, "")
		if err != nil {
			lastErr = runtimeStateQueryError(deadline, client, "", err)
		} else if stringSet(state.GetStateProviders())[peerID] {
			lastErr = fmt.Errorf("state providers still contain %s", peerID)
		} else if stringSet(state.GetRoutedPeers())[peerID] {
			lastErr = fmt.Errorf("routed peers still contain %s", peerID)
		} else if RuntimePeerStatus(state, peerID) != nil {
			lastErr = fmt.Errorf("runtime peer statuses still contain %s", peerID)
		} else if RuntimeCompatibility(state, peerID) != nil {
			lastErr = fmt.Errorf("runtime compatibility still contains %s", peerID)
		} else {
			return nil
		}
		if time.Since(lastReport) >= 30*time.Second {
			lastReport = time.Now()
			fmt.Printf("waiting for runtime peer removal: peer=%s err=%v\n", peerID, lastErr)
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("runtime peer %s remained after deletion: %w", peerID, lastErr)
}

func WaitForPeerRemovedFromRuntimes(deadline time.Time, client pbApic.ProtosClientApiClient, instances []*pbApic.CloudInstance, peerID string) error {
	var lastErr error
	var lastReport time.Time
	for time.Now().Before(deadline) {
		remaining := make([]string, 0, len(instances))
		for _, instance := range instances {
			if instance == nil {
				continue
			}
			state, err := RuntimeState(deadline, client, instance.GetName())
			if err != nil {
				remaining = append(remaining, instance.GetName()+": "+runtimeStateQueryError(deadline, client, instance.GetName(), err).Error())
				continue
			}
			if err := peerAbsentFromRuntimeState(state, peerID); err != nil {
				remaining = append(remaining, instance.GetName()+": "+err.Error())
			}
		}
		if len(remaining) == 0 {
			return nil
		}
		lastErr = fmt.Errorf("%s", strings.Join(remaining, "; "))
		if time.Since(lastReport) >= 30*time.Second {
			lastReport = time.Now()
			fmt.Printf("waiting for remote runtime peer removal: peer=%s err=%v\n", peerID, lastErr)
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("remote runtimes still observed peer %s: %w", peerID, lastErr)
}

func peerAbsentFromRuntimeState(state *pbApic.RuntimeState, peerID string) error {
	if stringSet(state.GetStateProviders())[peerID] {
		return fmt.Errorf("state providers still contain %s", peerID)
	}
	if stringSet(state.GetRoutedPeers())[peerID] {
		return fmt.Errorf("routed peers still contain %s", peerID)
	}
	if RuntimePeerStatus(state, peerID) != nil {
		return fmt.Errorf("runtime peer statuses still contain %s", peerID)
	}
	if RuntimeCompatibility(state, peerID) != nil {
		return fmt.Errorf("runtime compatibility still contains %s", peerID)
	}
	return nil
}

func RuntimeState(deadline time.Time, client pbApic.ProtosClientApiClient, instance string) (*pbApic.RuntimeState, error) {
	return RuntimeStateWithOptions(deadline, client, instance, false)
}

func RuntimeStateWithOptions(deadline time.Time, client pbApic.ProtosClientApiClient, instance string, allowStale bool) (*pbApic.RuntimeState, error) {
	ctx, cancel := contextWithDeadline(deadline, 15*time.Second)
	defer cancel()
	resp, err := client.GetRuntimeState(ctx, &pbApic.GetRuntimeStateRequest{Instance: instance, AllowStale: allowStale})
	if err != nil {
		return nil, err
	}
	if resp.GetState() == nil {
		return nil, fmt.Errorf("empty runtime state")
	}
	return resp.GetState(), nil
}

func runtimeStateQueryError(deadline time.Time, client pbApic.ProtosClientApiClient, instance string, err error) error {
	state, diagErr := RuntimeStateWithOptions(deadline, client, instance, true)
	if diagErr != nil {
		return fmt.Errorf("runtime state query/catch-up failed: %w; stale diagnostic failed: %w", err, diagErr)
	}
	return fmt.Errorf(
		"runtime state query/catch-up failed: %w; stale diagnostic consistency=%s read_error=%q summary={%s}",
		err,
		state.GetReadConsistency(),
		state.GetReadError(),
		RuntimeStateSummary(state),
	)
}

func AssertReplicationPriorities(state *pbApic.RuntimeState, expected map[string]int) error {
	if len(expected) == 0 {
		return nil
	}
	statuses := map[string]*pbApic.RuntimePeerStatus{}
	for _, status := range state.GetPeerStatuses() {
		statuses[status.GetPeerId()] = status
	}
	for peerID, want := range expected {
		status, found := statuses[peerID]
		if !found {
			return fmt.Errorf("missing runtime peer status for %s", peerID)
		}
		if int(status.GetReplicationPriority()) != want {
			return fmt.Errorf("replication priority for peer %s = %d, want %d", peerID, status.GetReplicationPriority(), want)
		}
		if strings.TrimSpace(status.GetReplicationDeviceClass()) == "" {
			return fmt.Errorf("replication device class for peer %s is empty", peerID)
		}
	}
	return nil
}

func AssertNoBlockingCompatibility(state *pbApic.RuntimeState) error {
	for _, item := range state.GetCompatibility() {
		if item.GetBlocking() {
			return fmt.Errorf("runtime compatibility blocked by peer %s: %s", item.GetPeerId(), item.GetReason())
		}
	}
	return nil
}

func WaitForLocalCheckpoint(deadline time.Time, client pbApic.ProtosClientApiClient, label string) error {
	var lastErr error
	var lastReport time.Time
	for time.Now().Before(deadline) {
		state, err := RuntimeState(deadline, client, "")
		if err != nil {
			lastErr = err
		} else {
			checkpoint := state.GetCheckpointRootHash()
			durable := state.GetDurableMainRootHash()
			if checkpoint != "" && state.GetTentativeRootHash() == checkpoint && (durable == "" || durable == checkpoint) {
				fmt.Printf("local head checkpointed for %s: %s\n", label, checkpoint)
				return nil
			}
			lastErr = fmt.Errorf("checkpoint=%s tentative=%s durable=%s", checkpoint, state.GetTentativeRootHash(), durable)
		}
		if time.Since(lastReport) >= 30*time.Second {
			lastReport = time.Now()
			fmt.Printf("waiting for local checkpoint: label=%s err=%v\n", label, lastErr)
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("local authored head did not checkpoint for %s: %w", label, lastErr)
}

func WaitForAllRemoteHeads(deadline time.Time, client pbApic.ProtosClientApiClient, instances []*pbApic.CloudInstance) error {
	var lastErr error
	var lastReport time.Time
	for time.Now().Before(deadline) {
		localState, err := RuntimeState(deadline, client, "")
		if err != nil {
			lastErr = err
			reportRemoteCheckpointWait(&lastReport, "", nil, lastErr)
			time.Sleep(3 * time.Second)
			continue
		}
		localRoot, err := checkpointedRuntimeRoot(localState)
		if err != nil {
			lastErr = fmt.Errorf("local runtime checkpoint not ready: %w", err)
			reportRemoteCheckpointWait(&lastReport, "", localState, lastErr)
			time.Sleep(3 * time.Second)
			continue
		}
		ready := true
		for _, instance := range instances {
			remoteState, err := RuntimeState(deadline, client, instance.GetName())
			if err != nil {
				lastErr = fmt.Errorf("%s get remote runtime checkpoint: %w", instance.GetName(), err)
				ready = false
				reportRemoteCheckpointWait(&lastReport, instance.GetName(), nil, lastErr)
				break
			}
			remoteRoot, err := checkpointedRuntimeRoot(remoteState)
			if err != nil {
				lastErr = fmt.Errorf("%s remote runtime checkpoint not ready: %w", instance.GetName(), err)
				ready = false
				reportRemoteCheckpointWait(&lastReport, instance.GetName(), remoteState, lastErr)
				break
			}
			if remoteRoot != localRoot {
				lastErr = fmt.Errorf("%s remote checkpoint root %q did not match local checkpoint root %q", instance.GetName(), remoteRoot, localRoot)
				ready = false
				reportRemoteCheckpointWait(&lastReport, instance.GetName(), remoteState, lastErr)
				break
			}
		}
		if ready {
			for _, instance := range instances {
				fmt.Printf("remote runtime checkpoint matched local root for %s: %s\n", instance.GetName(), localRoot)
			}
			return nil
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("remote runtime checkpoint did not match current local runtime checkpoint: %w", lastErr)
}

func checkpointedRuntimeRoot(state *pbApic.RuntimeState) (string, error) {
	if state == nil {
		return "", fmt.Errorf("empty runtime state")
	}
	if fatal := strings.TrimSpace(state.GetFatalState()); fatal != "" && fatal != "none" {
		return "", fmt.Errorf("fatal runtime state: %s", fatal)
	}
	if err := AssertNoBlockingCompatibility(state); err != nil {
		return "", err
	}
	root := strings.TrimSpace(state.GetCheckpointRootHash())
	if root == "" {
		return "", fmt.Errorf("checkpoint root is empty")
	}
	if tentative := strings.TrimSpace(state.GetTentativeRootHash()); tentative != root {
		return "", fmt.Errorf("tentative root %q did not match checkpoint root %q", tentative, root)
	}
	if durable := strings.TrimSpace(state.GetDurableMainRootHash()); durable != "" && durable != root {
		return "", fmt.Errorf("durable root %q did not match checkpoint root %q", durable, root)
	}
	return root, nil
}

func CreateAndStartApp(deadline time.Time, client pbApic.ProtosClientApiClient, image string, name string, instanceID string) error {
	ctx, cancel := contextWithDeadline(deadline, 30*time.Second)
	_, err := client.CreateApp(ctx, &pbApic.CreateAppRequest{
		Name:        name,
		InstallerId: image,
		InstanceId:  instanceID,
		Persistence: false,
	})
	cancel()
	if err != nil {
		return err
	}
	ctx, cancel = contextWithDeadline(deadline, 30*time.Second)
	_, err = client.StartApp(ctx, &pbApic.StartAppRequest{Name: name})
	cancel()
	return err
}

func WaitForAppStatus(deadline time.Time, client pbApic.ProtosClientApiClient, instanceName string, appName string, want string) (*pbApic.App, error) {
	var lastErr error
	var lastReport time.Time
	for time.Now().Before(deadline) {
		apps, err := ListApps(deadline, client)
		if err != nil {
			lastErr = err
		} else {
			for _, app := range apps {
				if app.GetName() != appName {
					continue
				}
				if instanceName != "" && app.GetInstanceName() != instanceName {
					continue
				}
				got := strings.TrimSpace(app.GetStatus())
				if strings.HasPrefix(got, want) {
					fmt.Printf("remote app status ok: instance=%s app=%s status=%s\n", app.GetInstanceName(), app.GetName(), got)
					return app, nil
				}
				lastErr = fmt.Errorf("status=%q, want prefix %q", got, want)
				break
			}
			if lastErr == nil {
				lastErr = fmt.Errorf("app %s on %s not found", appName, instanceName)
			}
		}
		if time.Since(lastReport) >= 30*time.Second {
			lastReport = time.Now()
			fmt.Printf("waiting for remote app status: instance=%s app=%s err=%v\n", instanceName, appName, lastErr)
		}
		time.Sleep(5 * time.Second)
	}
	return nil, fmt.Errorf("app %s on %s did not reach status %q: %w", appName, instanceName, want, lastErr)
}

func ListApps(deadline time.Time, client pbApic.ProtosClientApiClient) ([]*pbApic.App, error) {
	ctx, cancel := contextWithDeadline(deadline, 15*time.Second)
	defer cancel()
	resp, err := client.GetApps(ctx, &pbApic.GetAppsRequest{})
	if err != nil {
		return nil, err
	}
	return resp.GetApps(), nil
}

func WaitForNoApps(deadline time.Time, client pbApic.ProtosClientApiClient) error {
	var lastErr error
	for time.Now().Before(deadline) {
		apps, err := ListApps(deadline, client)
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
			continue
		}
		if len(apps) == 0 {
			return nil
		}
		lastErr = fmt.Errorf("apps still present: %v", AppNames(apps))
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("apps remained after e2e deprovision: %w", lastErr)
}

func WaitForNoAppsForInstance(deadline time.Time, client pbApic.ProtosClientApiClient, instanceIDs ...string) error {
	ids := stringSet(instanceIDs)
	var lastErr error
	for time.Now().Before(deadline) {
		apps, err := ListApps(deadline, client)
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
			continue
		}
		var names []string
		for _, app := range apps {
			if ids[app.GetInstanceName()] {
				names = append(names, app.GetName())
			}
		}
		if len(names) == 0 {
			return nil
		}
		lastErr = fmt.Errorf("apps still reference instance %v: %v", instanceIDs, names)
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("apps remained after instance %v deletion: %w", instanceIDs, lastErr)
}

type UploadInstanceImageArchiveResult struct {
	TaskID           string
	Task             *pbApic.Task
	Events           []*pbApic.TaskEvent
	ProgressUpdates  []*pbApic.TaskProgressUpdate
	Instance         string
	ImageRef         string
	TargetDigest     string
	Platform         string
	BytesUploaded    uint64
	ArchiveSizeBytes uint64
}

func UploadImageArchive(deadline time.Time, client pbApic.ProtosClientApiClient, instanceName string, archivePath string, imageRef string) (UploadInstanceImageArchiveResult, error) {
	absPath, err := filepath.Abs(archivePath)
	if err != nil {
		return UploadInstanceImageArchiveResult{}, err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return UploadInstanceImageArchiveResult{}, fmt.Errorf("stat seed image archive %s: %w", absPath, err)
	}
	if !info.Mode().IsRegular() {
		return UploadInstanceImageArchiveResult{}, fmt.Errorf("seed image archive %s is not a regular file", absPath)
	}

	ctx, cancel := contextWithDeadline(deadline, 30*time.Second)
	resp, err := client.UploadInstanceImageArchive(ctx, &pbApic.UploadInstanceImageArchiveRequest{
		Instance:    instanceName,
		ArchivePath: absPath,
		ImageRef:    imageRef,
	})
	cancel()
	if err != nil {
		return UploadInstanceImageArchiveResult{}, err
	}
	taskID := strings.TrimSpace(resp.GetTaskId())
	if taskID == "" {
		return UploadInstanceImageArchiveResult{}, fmt.Errorf("image archive upload response did not include a task id")
	}
	fmt.Printf("queued remote seed image archive upload task: instance=%s image=%s path=%s size=%s task=%s\n", instanceName, imageRef, absPath, formatByteCount(uint64(info.Size())), taskID)
	task, events, updates, err := WaitForTaskSucceededWithProgress(deadline, client, taskID)
	if err != nil {
		return UploadInstanceImageArchiveResult{}, err
	}
	result := uploadInstanceImageArchiveTaskResult(task)
	result.TaskID = taskID
	result.Task = task
	result.Events = events
	result.ProgressUpdates = updates
	if result.Instance == "" {
		result.Instance = instanceName
	}
	if result.ImageRef == "" {
		result.ImageRef = imageRef
	}
	if result.ArchiveSizeBytes == 0 {
		result.ArchiveSizeBytes = uint64(info.Size())
	}
	fmt.Printf("remote seed image archive loaded: instance=%s image=%s digest=%s platform=%s bytes=%s\n", result.Instance, result.ImageRef, result.TargetDigest, result.Platform, formatByteCount(result.BytesUploaded))
	return result, nil
}

func formatByteCount(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	value := float64(bytes)
	for _, suffix := range []string{"KiB", "MiB", "GiB", "TiB"} {
		value /= unit
		if value < unit {
			return fmt.Sprintf("%.1f%s", value, suffix)
		}
	}
	return fmt.Sprintf("%.1fPiB", value/unit)
}

func WaitForRemoteImageContentReady(deadline time.Time, client pbApic.ProtosClientApiClient, instanceName string, imageRef string) error {
	var lastErr error
	var lastReport time.Time
	for time.Now().Before(deadline) {
		ctx, cancel := contextWithDeadline(deadline, 30*time.Second)
		resp, err := client.GetInstanceImage(ctx, &pbApic.GetInstanceImageRequest{Instance: instanceName, ImageRef: imageRef, IncludeContent: true})
		cancel()
		if err == nil && resp.GetFound() && resp.GetHasContent() && resp.GetTarget() != nil && len(resp.GetDescriptors()) > 0 {
			fmt.Printf("remote image content ready: instance=%s image=%s digest=%s blobs=%d\n", instanceName, imageRef, resp.GetTargetDigest(), len(resp.GetDescriptors()))
			return nil
		}
		if err != nil {
			lastErr = err
		} else if !resp.GetFound() {
			lastErr = fmt.Errorf("image %s is not present on %s", imageRef, instanceName)
		} else if !resp.GetHasContent() {
			lastErr = fmt.Errorf("image content %s is not present on %s", imageRef, instanceName)
		} else if resp.GetTarget() == nil {
			lastErr = fmt.Errorf("image content target is empty")
		} else {
			lastErr = fmt.Errorf("image content descriptor list is empty")
		}
		if time.Since(lastReport) >= 30*time.Second {
			lastReport = time.Now()
			fmt.Printf("waiting for remote image content: instance=%s image=%s err=%v\n", instanceName, imageRef, lastErr)
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("image content was not ready on %s for %s: %w", instanceName, imageRef, lastErr)
}

func WaitForRemoteP2PImageLabel(deadline time.Time, client pbApic.ProtosClientApiClient, instanceName string, imageRef string) error {
	var lastErr error
	var lastReport time.Time
	for time.Now().Before(deadline) {
		ctx, cancel := contextWithDeadline(deadline, 10*time.Second)
		resp, err := client.GetInstanceImage(ctx, &pbApic.GetInstanceImageRequest{Instance: instanceName, ImageRef: imageRef})
		cancel()
		if err == nil && resp.GetFound() && resp.GetLabels()["protos.io/image.source"] == "p2p" {
			fmt.Printf("remote P2P image label observed: instance=%s image=%s digest=%s\n", instanceName, imageRef, resp.GetTargetDigest())
			return nil
		}
		if err != nil {
			lastErr = err
		} else if !resp.GetFound() {
			lastErr = fmt.Errorf("image %s is not present", imageRef)
		} else {
			lastErr = fmt.Errorf("image %s label protos.io/image.source=%q, want p2p", imageRef, resp.GetLabels()["protos.io/image.source"])
		}
		if time.Since(lastReport) >= 30*time.Second {
			lastReport = time.Now()
			fmt.Printf("waiting for remote P2P image label: instance=%s image=%s err=%v\n", instanceName, imageRef, lastErr)
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("%s did not report Protos P2P image source for %s: %w", instanceName, imageRef, lastErr)
}

func WaitForHostWireGuardPings(deadline time.Time, instances []*pbApic.CloudInstance) error {
	var lastErr error
	for time.Now().Before(deadline) {
		allReachable := true
		for _, instance := range instances {
			ip := WireGuardIPv6(instance)
			if ip == "" {
				return fmt.Errorf("instance %s has no derivable WireGuard IPv6 address", instance.GetName())
			}
			ctx, cancel := contextWithDeadline(deadline, 8*time.Second)
			output, err := exec.CommandContext(ctx, "ping6", "-c", "3", "-n", ip).CombinedOutput()
			cancel()
			if err != nil {
				allReachable = false
				lastErr = fmt.Errorf("ping %s (%s): %w: %s", instance.GetName(), ip, err, strings.TrimSpace(string(output)))
				break
			}
			fmt.Printf("host WireGuard ping ok: %s %s\n", instance.GetName(), ip)
		}
		if allReachable {
			return nil
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("host could not reach all VM WireGuard IPv6 addresses: %w", lastErr)
}

func WaitForContainerHTTPToApp(deadline time.Time, source *pbApic.CloudInstance, sourceApp *pbApic.App, targetApp *pbApic.App) error {
	if sourceApp == nil || targetApp == nil {
		return fmt.Errorf("source and target apps are required")
	}
	if strings.TrimSpace(sourceApp.GetIp()) == "" {
		return fmt.Errorf("source app %s has no overlay IP", sourceApp.GetName())
	}
	if strings.TrimSpace(targetApp.GetIp()) == "" {
		return fmt.Errorf("target app %s has no overlay IP", targetApp.GetName())
	}
	var lastErr error
	var lastReport time.Time
	httpClient := &http.Client{Timeout: 15 * time.Second}
	for time.Now().Before(deadline) {
		targetURL := AppProbeURL(targetApp, "/")
		probeURL := AppProbeURL(sourceApp, "/probe") + "?target=" + url.QueryEscape(targetURL) + "&timeout_ms=8000&max_bytes=4096"
		ctx, cancel := contextWithDeadline(deadline, 15*time.Second)
		resp, err := httpGetProbe(ctx, httpClient, probeURL)
		cancel()
		if err == nil && resp.OK {
			fmt.Printf("app connectivity ok: source_instance=%s source_app=%s target_app=%s target_ip=%s bytes=%d status=%d\n", source.GetName(), sourceApp.GetName(), targetApp.GetName(), targetApp.GetIp(), resp.BytesRead, resp.StatusCode)
			return nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("probe returned ok=false status=%d error=%q", resp.StatusCode, resp.Error)
		}
		if time.Since(lastReport) >= 30*time.Second {
			lastReport = time.Now()
			fmt.Printf("waiting for app connectivity: source_instance=%s source_app=%s target_app=%s err=%v\n", source.GetName(), sourceApp.GetName(), targetApp.GetName(), lastErr)
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("app %s on %s could not reach app %s over the overlay: %w", sourceApp.GetName(), source.GetName(), targetApp.GetName(), lastErr)
}

func AppProbeURL(app *pbApic.App, path string) string {
	return fmt.Sprintf("http://[%s]:%d%s", app.GetIp(), ProbeAppPort, path)
}

func PeerIDForInstance(instance *pbApic.CloudInstance) (string, error) {
	peerID, err := pcrypto.PeerIDFromPublicKeyString(instance.GetPublicKey())
	if err != nil {
		return "", fmt.Errorf("derive peer id for instance %s: %w", instance.GetName(), err)
	}
	return peerID, nil
}

func peerIDsByInstanceName(instances []*pbApic.CloudInstance) (map[string]string, error) {
	out := make(map[string]string, len(instances))
	for _, instance := range instances {
		peerID, err := PeerIDForInstance(instance)
		if err != nil {
			return nil, err
		}
		out[instance.GetName()] = peerID
	}
	return out, nil
}

func WireGuardIPv6(instance *pbApic.CloudInstance) string {
	if ip := strings.TrimSpace(instance.GetInternalIp()); ip != "" {
		return ip
	}
	key, err := pcrypto.CreatePublicKeyFromBase64(instance.GetPublicKey())
	if err != nil {
		return ""
	}
	return key.IPv6Address().String()
}

func InstanceNames(instances []*pbApic.CloudInstance) []string {
	names := make([]string, 0, len(instances))
	for _, instance := range instances {
		names = append(names, instance.GetName())
	}
	return names
}

func AppNames(apps []*pbApic.App) []string {
	names := make([]string, 0, len(apps))
	for _, app := range apps {
		names = append(names, app.GetName())
	}
	return names
}

func RuntimePeerStatus(state *pbApic.RuntimeState, peerID string) *pbApic.RuntimePeerStatus {
	if state == nil {
		return nil
	}
	for _, peerStatus := range state.GetPeerStatuses() {
		if peerStatus.GetPeerId() == peerID {
			return peerStatus
		}
	}
	return nil
}

// runtimePeerTransportReady requires all three independent readiness planes:
// a live application-owned host connection, a Swarmion route over that host,
// and participation in the database scope. Logical overlay membership is
// intentionally not considered physical reachability.
func runtimePeerTransportReady(state *pbApic.RuntimeState, peerID string) (bool, []string) {
	if state == nil || strings.TrimSpace(peerID) == "" {
		return false, []string{"physical", "routed", "participating"}
	}
	status := RuntimePeerStatus(state, peerID)
	physical := stringSet(state.GetPhysicalConnectedPeers())[peerID] || status != nil && status.GetPhysicalConnected()
	routed := stringSet(state.GetRoutedPeers())[peerID] || status != nil && status.GetRouted()
	participating := stringSet(state.GetParticipatingPeers())[peerID] || status != nil && status.GetParticipating()

	missing := make([]string, 0, 3)
	if !physical {
		missing = append(missing, "physical")
	}
	if !routed {
		missing = append(missing, "routed")
	}
	if !participating {
		missing = append(missing, "participating")
	}
	return len(missing) == 0, missing
}

func RuntimeCompatibility(state *pbApic.RuntimeState, peerID string) *pbApic.RuntimeCompatibility {
	if state == nil {
		return nil
	}
	for _, item := range state.GetCompatibility() {
		if item.GetPeerId() == peerID {
			return item
		}
	}
	return nil
}

func RuntimeStateSummary(state *pbApic.RuntimeState) string {
	if state == nil {
		return "nil"
	}
	trace := state.GetContentSyncTrace()
	if len(trace) > 8 {
		trace = trace[len(trace)-8:]
	}
	return fmt.Sprintf(
		"peer=%s providers=%v physical=%v routed=%v participating=%v logical=%v logical_target=%d read_consistency=%s read_error=%q checkpoint_root=%s protocol_root=%s durable_root=%s event_receipt_content_dissent_observations=%d pending_checkpoint=%t checkpoint_error=%q refresh_pending=%t refresh_error=%q fatal=%q trace=%v",
		state.GetPeerId(),
		state.GetStateProviders(),
		state.GetPhysicalConnectedPeers(),
		state.GetRoutedPeers(),
		state.GetParticipatingPeers(),
		state.GetLogicalPeers(),
		state.GetLogicalPeerTarget(),
		state.GetReadConsistency(),
		state.GetReadError(),
		state.GetCheckpointRootHash(),
		state.GetProtocolCheckpointRootHash(),
		state.GetDurableMainRootHash(),
		state.GetEventReceiptContentDissentObservations(),
		state.GetRuntimeCheckpointPending(),
		state.GetRuntimeCheckpointLastError(),
		state.GetRuntimeRefreshPending(),
		state.GetRuntimeRefreshLastError(),
		state.GetFatalState(),
		trace,
	)
}

func Minutes(timeout time.Duration) int32 {
	if timeout <= 0 {
		return 0
	}
	return int32((timeout + time.Minute - 1) / time.Minute)
}

type UploadProvisionerImageResult struct {
	ImageID         string
	TaskID          string
	Task            *pbApic.Task
	Events          []*pbApic.TaskEvent
	ProgressUpdates []*pbApic.TaskProgressUpdate
}

func UploadProvisionerImage(deadline time.Time, client pbApic.ProtosClientApiClient, imagePath string, imageName string, provisionerName string, location string, timeout time.Duration) (string, error) {
	result, err := UploadProvisionerImageDetailed(deadline, client, imagePath, imageName, provisionerName, location, timeout)
	if err != nil {
		return "", err
	}
	return result.ImageID, nil
}

func UploadProvisionerImageDetailed(deadline time.Time, client pbApic.ProtosClientApiClient, imagePath string, imageName string, provisionerName string, location string, timeout time.Duration) (UploadProvisionerImageResult, error) {
	ctx, cancel := contextWithDeadline(deadline, 30*time.Second)
	resp, err := client.UploadProvisionerImage(ctx, &pbApic.UploadProvisionerImageRequest{
		ImagePath:       imagePath,
		ImageName:       imageName,
		ProvisionerName: provisionerName,
		Location:        location,
		Timeout:         Minutes(timeout),
	})
	cancel()
	if err != nil {
		return UploadProvisionerImageResult{}, err
	}
	taskID := strings.TrimSpace(resp.GetTaskId())
	if taskID == "" {
		return UploadProvisionerImageResult{}, fmt.Errorf("upload image response did not include a task id")
	}
	fmt.Printf("queued image upload task: image=%s provisioner=%s location=%s task=%s\n", imageName, provisionerName, location, taskID)
	task, events, updates, err := WaitForTaskSucceededWithProgress(deadline, client, taskID)
	if err != nil {
		return UploadProvisionerImageResult{}, err
	}
	imageID := uploadTaskImageID(task)
	if imageID == "" {
		return UploadProvisionerImageResult{}, fmt.Errorf("upload task %s completed without image_id result: %s", taskID, task.GetResultJson())
	}
	return UploadProvisionerImageResult{
		ImageID:         imageID,
		TaskID:          taskID,
		Task:            task,
		Events:          events,
		ProgressUpdates: updates,
	}, nil
}

func WaitForTaskSucceededWithEvents(deadline time.Time, client pbApic.ProtosClientApiClient, taskID string) (*pbApic.Task, []*pbApic.TaskEvent, error) {
	task, events, _, err := waitForTaskSucceeded(deadline, client, taskID, true)
	return task, events, err
}

func WaitForTaskSucceededWithProgress(deadline time.Time, client pbApic.ProtosClientApiClient, taskID string) (*pbApic.Task, []*pbApic.TaskEvent, []*pbApic.TaskProgressUpdate, error) {
	return waitForTaskSucceeded(deadline, client, taskID, true)
}

func waitForTaskSucceeded(deadline time.Time, client pbApic.ProtosClientApiClient, taskID string, includeEvents bool) (*pbApic.Task, []*pbApic.TaskEvent, []*pbApic.TaskProgressUpdate, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, nil, nil, fmt.Errorf("task id is empty")
	}
	ctx, cancel := contextWithDeadline(deadline, 0)
	defer cancel()
	stream, err := client.WatchTask(ctx, &pbApic.WatchTaskRequest{
		Id:                  taskID,
		IncludeSnapshot:     true,
		IncludeEvents:       includeEvents,
		HeartbeatIntervalMs: 10000,
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("watch task %s: %w", taskID, err)
	}

	var latestStatus string
	var progressUpdates []*pbApic.TaskProgressUpdate
	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			task, events, err := fetchTaskResult(deadline, client, taskID, latestStatus, includeEvents)
			return task, events, progressUpdates, err
		}
		if err != nil {
			return nil, nil, nil, fmt.Errorf("watch task %s: %w", taskID, err)
		}
		if task := resp.GetTask(); task != nil && task.GetId() != "" {
			latestStatus = task.GetStatus()
			fmt.Printf("%s task snapshot: id=%s stream=%s status=%s progress=%d message=%s\n", time.Now().UTC().Format(time.RFC3339Nano), task.GetId(), task.GetStream(), task.GetStatus(), task.GetProgress(), task.GetMessage())
		}
		if update := resp.GetUpdate(); update != nil {
			latestStatus = update.GetStatus()
			progressUpdates = append(progressUpdates, update)
			fmt.Printf("%s task progress: id=%s status=%s progress=%d durable=%t message=%s%s\n", time.Now().UTC().Format(time.RFC3339Nano), update.GetTaskId(), update.GetStatus(), update.GetProgress(), update.GetDurable(), update.GetMessage(), taskProgressDetailsSummary(update.GetDetailsJson()))
		}
		if taskTerminalStatus(latestStatus) {
			task, events, err := fetchTaskResult(deadline, client, taskID, latestStatus, includeEvents)
			return task, events, progressUpdates, err
		}
	}
}

func fetchTaskResult(deadline time.Time, client pbApic.ProtosClientApiClient, taskID string, latestStatus string, includeEvents bool) (*pbApic.Task, []*pbApic.TaskEvent, error) {
	ctx, cancel := contextWithDeadline(deadline, 10*time.Second)
	defer cancel()
	resp, err := client.GetTask(ctx, &pbApic.GetTaskRequest{Id: taskID, IncludeEvents: includeEvents})
	if err != nil {
		return nil, nil, fmt.Errorf("get task %s: %w", taskID, err)
	}
	task := resp.GetTask()
	if task == nil || task.GetId() == "" {
		return nil, nil, fmt.Errorf("task %s not found", taskID)
	}
	if task.GetStatus() != "succeeded" {
		if taskFailedStatus(task.GetStatus()) {
			return nil, nil, fmt.Errorf("task %s ended with status %s: %s", taskID, task.GetStatus(), taskErrorSummary(task))
		}
		if latestStatus != "" {
			return nil, nil, fmt.Errorf("task %s stream ended before success; latest=%s current=%s", taskID, latestStatus, task.GetStatus())
		}
		return nil, nil, fmt.Errorf("task %s is %s, not succeeded", taskID, task.GetStatus())
	}
	return task, resp.GetEvents(), nil
}

func uploadTaskImageID(task *pbApic.Task) string {
	if task == nil || strings.TrimSpace(task.GetResultJson()) == "" {
		return ""
	}
	var result struct {
		ImageID string `json:"image_id"`
	}
	if err := json.Unmarshal([]byte(task.GetResultJson()), &result); err != nil {
		return ""
	}
	return strings.TrimSpace(result.ImageID)
}

func uploadInstanceImageArchiveTaskResult(task *pbApic.Task) UploadInstanceImageArchiveResult {
	if task == nil || strings.TrimSpace(task.GetResultJson()) == "" {
		return UploadInstanceImageArchiveResult{}
	}
	var result struct {
		Instance         string `json:"instance"`
		ImageRef         string `json:"image_ref"`
		TargetDigest     string `json:"target_digest"`
		Platform         string `json:"platform"`
		BytesUploaded    uint64 `json:"bytes_uploaded"`
		ArchiveSizeBytes uint64 `json:"archive_size_bytes"`
	}
	if err := json.Unmarshal([]byte(task.GetResultJson()), &result); err != nil {
		return UploadInstanceImageArchiveResult{}
	}
	return UploadInstanceImageArchiveResult{
		Instance:         strings.TrimSpace(result.Instance),
		ImageRef:         strings.TrimSpace(result.ImageRef),
		TargetDigest:     strings.TrimSpace(result.TargetDigest),
		Platform:         strings.TrimSpace(result.Platform),
		BytesUploaded:    result.BytesUploaded,
		ArchiveSizeBytes: result.ArchiveSizeBytes,
	}
}

func taskTerminalStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "succeeded", "failed", "cancelled":
		return true
	default:
		return false
	}
}

func taskFailedStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed", "cancelled":
		return true
	default:
		return false
	}
}

func taskErrorSummary(task *pbApic.Task) string {
	message := strings.TrimSpace(task.GetMessage())
	errMessage := strings.TrimSpace(task.GetErrorMessage())
	if errMessage == "" {
		return message
	}
	if message == "" || message == "failed" {
		return errMessage
	}
	return message + ": " + errMessage
}

func taskProgressDetailsSummary(detailsJSON string) string {
	detailsJSON = strings.TrimSpace(detailsJSON)
	if detailsJSON == "" || detailsJSON == "{}" {
		return ""
	}
	var values map[string]any
	if err := json.Unmarshal([]byte(detailsJSON), &values); err != nil {
		return " details=" + detailsJSON
	}
	var parts []string
	for _, key := range []string{
		"percent",
		"bytes_uploaded",
		"archive_size_bytes",
		"chunk_bytes",
		"chunk_duration_ms",
		"bytes_per_second",
		"elapsed_ms",
		"instance",
		"image_ref",
	} {
		value, found := values[key]
		if !found || value == nil {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%v", key, value))
	}
	if len(parts) == 0 {
		return " details=" + detailsJSON
	}
	return " details={" + strings.Join(parts, " ") + "}"
}

func contextWithDeadline(deadline time.Time, max time.Duration) (context.Context, context.CancelFunc) {
	timeout := time.Until(deadline)
	if timeout <= 0 {
		timeout = time.Millisecond
	}
	if max > 0 && timeout > max {
		timeout = max
	}
	return context.WithTimeout(context.Background(), timeout)
}

func containsInstance(instances []*pbApic.CloudInstance, id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	for _, instance := range instances {
		if instance.GetName() == id || instance.GetVmId() == id {
			return true
		}
	}
	return false
}

func terminalStatus(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	if idx := strings.Index(status, ":"); idx >= 0 {
		status = strings.TrimSpace(status[:idx])
	}
	switch status {
	case "failed", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

func reportReplicationProgress(lastReport *time.Time, label string, state *pbApic.RuntimeState, err error) {
	if !lastReport.IsZero() && time.Since(*lastReport) < 30*time.Second {
		return
	}
	*lastReport = time.Now()
	if state == nil {
		fmt.Printf("waiting for checkpoint state: label=%s err=%v\n", label, err)
		return
	}
	fmt.Printf("waiting for checkpoint state: label=%s err=%v %s\n", label, err, RuntimeStateSummary(state))
}

func reportRemoteCheckpointWait(lastReport *time.Time, instanceName string, state *pbApic.RuntimeState, err error) {
	if !lastReport.IsZero() && time.Since(*lastReport) < 30*time.Second {
		return
	}
	*lastReport = time.Now()
	if state != nil {
		fmt.Printf("waiting for remote runtime checkpoint: instance=%s err=%v runtime={%s}\n", instanceName, err, RuntimeStateSummary(state))
		return
	}
	fmt.Printf("waiting for remote runtime checkpoint: instance=%s err=%v\n", instanceName, err)
}

func sortedPeerPriorityKeys(priorities map[string]int) []string {
	out := make([]string, 0, len(priorities))
	for peerID := range priorities {
		if strings.TrimSpace(peerID) != "" {
			out = append(out, peerID)
		}
	}
	sort.Strings(out)
	return out
}

func stringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out[value] = true
		}
	}
	return out
}

func httpGetProbe(ctx context.Context, client *http.Client, probeURL string) (ProbeAppResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, nil)
	if err != nil {
		return ProbeAppResponse{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return ProbeAppResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return ProbeAppResponse{}, err
	}
	var out ProbeAppResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return ProbeAppResponse{}, fmt.Errorf("decode probe response status=%d body=%q: %w", resp.StatusCode, strings.TrimSpace(string(body)), err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return out, fmt.Errorf("probe endpoint returned status=%d ok=%v error=%q", resp.StatusCode, out.OK, out.Error)
	}
	return out, nil
}

func coreDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func defaultHostAgentBin() string {
	return filepath.Join(coreDir(), "..", "bin", "protos-hostagent")
}

func logTail(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return err.Error()
	}
	if len(data) > 16*1024 {
		data = data[len(data)-16*1024:]
	}
	return string(data)
}

func startupFailureLog(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Protos macOS startup failed:") {
			return line
		}
		if strings.Contains(line, "failed to listen on local socket:") {
			return line
		}
	}
	return ""
}
