//go:build darwin

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/protosio/protos/integration-tests/e2eapic"
	hostagentclient "github.com/protosio/protos/internal/hostagent/client"
	localmacos "github.com/protosio/protos/provisioners/local_macos"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func main() {
	imagePath := flag.String("image", "../cloud-provisioning/targets/output/mactest", "LinuxKit image directory or ISO path")
	flutterApp := flag.String("flutter-app", "", "built Protos macOS Flutter app bundle")
	hostAgentBin := flag.String("hostagent-bin", "../bin/protos-hostagent", "protos-hostagent binary used by the Flutter node")
	workDir := flag.String("workdir", "", "temporary Protos workdir")
	keep := flag.Bool("keep", false, "keep temporary state after the run")
	timeout := flag.Duration("timeout", 6*time.Minute, "overall verification timeout")
	machineType := flag.String("machine", "vz-2c-2g", "local macOS machine type")
	instanceCount := flag.Int("instances", 2, "number of local macOS VMs to deploy and verify")
	configureNetwork := flag.Bool("network", true, "configure the host network module through protos-hostagent")
	appImage := flag.String("app-image", "", "optional app image used to verify Protos P2P image resolution across local VMs")
	flag.Parse()

	if err := run(*imagePath, *flutterApp, *hostAgentBin, *workDir, *keep, *timeout, *machineType, *instanceCount, *configureNetwork, *appImage); err != nil {
		fmt.Fprintf(os.Stderr, "e2e failed: %v\n", err)
		os.Exit(1)
	}
}

func run(imagePath string, flutterApp string, hostAgentBin string, workDir string, keep bool, timeout time.Duration, machineType string, instanceCount int, configureNetwork bool, appImage string) error {
	if instanceCount < 1 {
		return fmt.Errorf("instances must be greater than zero")
	}
	imagePath, err := filepath.Abs(imagePath)
	if err != nil {
		return err
	}
	if workDir == "" {
		workDir, err = os.MkdirTemp("/tmp", "protos-local-macos-e2e-*")
		if err != nil {
			return err
		}
	} else {
		workDir, err = filepath.Abs(workDir)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(workDir, 0755); err != nil {
			return err
		}
	}
	if !keep {
		defer os.RemoveAll(workDir)
	}
	if configureNetwork {
		cleanupStaleLocalE2EVMRunners(hostAgentBin)
		stopExistingHostAgentDaemon(hostAgentBin)
	}

	node, err := e2eapic.StartFlutterNode(e2eapic.FlutterNodeOptions{
		WorkDir:          workDir,
		AppPath:          flutterApp,
		HostAgentBin:     hostAgentBin,
		ConfigureNetwork: configureNetwork,
	})
	if err != nil {
		return err
	}
	defer func() {
		_ = node.Stop()
	}()
	client := node.Client

	startupCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	if err := e2eapic.Init(startupCtx, client, "e2e", "E2E", "E2E"); err != nil {
		cancel()
		return fmt.Errorf("initialize Flutter local node: %w", err)
	}
	if configureNetwork {
		if err := e2eapic.StartHostAgent(startupCtx, client); err != nil {
			cancel()
			return fmt.Errorf("start host agent through APIC: %w", err)
		}
		cleanupStaleLocalE2EVMs(workDir)
	}
	cancel()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	providerName := "local-e2e-" + suffix
	imageName := "mactest-e2e-" + suffix
	vmDir := filepath.Join(workDir, "local-macos-vms")
	deadline := time.Now().Add(timeout)

	cleanup := &localMacOSE2ECleanup{
		client:       client,
		providerName: providerName,
		imageName:    imageName,
		vmDir:        vmDir,
	}
	defer cleanup.run()
	stopSignalCleanup := installSignalCleanup(cleanup, node, workDir, keep)
	defer stopSignalCleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	if _, err := client.AddProvisioner(ctx, &pbApic.AddProvisionerRequest{
		Name:        providerName,
		Type:        localmacos.Type.String(),
		Credentials: map[string]string{"VM_DIR": vmDir},
	}); err != nil {
		cancel()
		return fmt.Errorf("add local macOS provisioner: %w", err)
	}
	cancel()

	if _, err := e2eapic.UploadProvisionerImage(deadline, client, imagePath, imageName, providerName, "local", 10*time.Minute); err != nil {
		return fmt.Errorf("upload local image: %w", err)
	}
	cleanup.markImageUploaded()

	var deployed []*pbApic.CloudInstance
	for i := 0; i < instanceCount; i++ {
		instanceName := fmt.Sprintf("vm-e2e-%s-%d", suffix, i+1)
		ctx, cancel = context.WithTimeout(context.Background(), 2*time.Minute)
		if _, err := client.DeployInstance(ctx, &pbApic.DeployInstanceRequest{
			Name:          instanceName,
			CloudName:     providerName,
			CloudLocation: "local",
			MachineType:   machineType,
			DevImg:        imageName,
		}); err != nil {
			cancel()
			return fmt.Errorf("deploy local macOS instance %s: %w", instanceName, err)
		}
		cancel()
		cleanup.addInstance(instanceName)

		instance, err := e2eapic.WaitForInstanceReady(deadline, client, instanceName)
		if err != nil {
			return err
		}
		deployed = append(deployed, instance)
		fmt.Printf("deployed instance: name=%s id=%s ip=%s wg=%s arch=%s status=%s\n", instance.GetName(), instance.GetVmId(), instance.GetPublicIp(), e2eapic.WireGuardIPv6(instance), instance.GetArchitecture(), instance.GetStatus())

		if err := e2eapic.WaitForLocalCheckpoint(deadline, client, "local VM deploy"); err != nil {
			return err
		}
	}

	if configureNetwork {
		if err := e2eapic.WaitForHostWireGuardPings(deadline, deployed); err != nil {
			return err
		}
	}
	if len(deployed) > 1 {
		if err := e2eapic.WaitForRemoteFullMesh(deadline, client, deployed); err != nil {
			return err
		}
	}
	if err := e2eapic.WaitForAllRemoteHeads(deadline, client, deployed); err != nil {
		return err
	}

	if strings.TrimSpace(appImage) != "" {
		if err := verifyLocalImageRegistry(deadline, client, workDir, deployed, appImage, suffix); err != nil {
			return err
		}
	}

	return nil
}

type localMacOSE2ECleanup struct {
	once sync.Once
	mu   sync.Mutex

	client       pbApic.ProtosClientApiClient
	providerName string
	imageName    string
	vmDir        string

	instances     []string
	imageUploaded bool
}

func (c *localMacOSE2ECleanup) addInstance(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.instances = append(c.instances, name)
}

func (c *localMacOSE2ECleanup) markImageUploaded() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.imageUploaded = true
}

func (c *localMacOSE2ECleanup) run() {
	c.once.Do(func() {
		c.mu.Lock()
		instances := append([]string(nil), c.instances...)
		imageUploaded := c.imageUploaded
		client := c.client
		providerName := c.providerName
		imageName := c.imageName
		vmDir := c.vmDir
		c.mu.Unlock()

		if client != nil {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cleanupCancel()
			cleanupDeadline := time.Now().Add(5 * time.Minute)
			for i := len(instances) - 1; i >= 0; i-- {
				resp, err := client.RemoveInstance(cleanupCtx, &pbApic.RemoveInstanceRequest{Name: instances[i]})
				if err != nil {
					fmt.Fprintf(os.Stderr, "warn cleanup instance %s through APIC failed: %v\n", instances[i], err)
					continue
				}
				if taskID := strings.TrimSpace(resp.GetTaskId()); taskID != "" {
					if _, _, err := e2eapic.WaitForTaskSucceededWithEvents(cleanupDeadline, client, taskID); err != nil {
						fmt.Fprintf(os.Stderr, "warn cleanup instance %s task %s failed: %v\n", instances[i], taskID, err)
					}
				}
			}
			if imageUploaded {
				if _, err := client.RemoveProvisionerImage(cleanupCtx, &pbApic.RemoveProvisionerImageRequest{
					ImageName:       imageName,
					ProvisionerName: providerName,
					Location:        "local",
				}); err != nil {
					fmt.Fprintf(os.Stderr, "warn cleanup provisioner image %s through APIC failed: %v\n", imageName, err)
				}
			}
			if _, err := client.RemoveProvisioner(cleanupCtx, &pbApic.RemoveProvisionerRequest{Name: providerName}); err != nil {
				fmt.Fprintf(os.Stderr, "warn cleanup provisioner %s through APIC failed: %v\n", providerName, err)
			}
		}
		if err := cleanupHostAgentVMs(vmDir); err != nil {
			fmt.Fprintf(os.Stderr, "warn cleanup local VMs through host-agent failed: %v\n", err)
		}
	})
}

func installSignalCleanup(cleanup *localMacOSE2ECleanup, node *e2eapic.FlutterNode, workDir string, keep bool) func() {
	sigCh := make(chan os.Signal, 1)
	stopCh := make(chan struct{})
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case sig := <-sigCh:
			fmt.Fprintf(os.Stderr, "received %s, cleaning up local e2e resources\n", sig)
			if cleanup != nil {
				cleanup.run()
			}
			if node != nil {
				_ = node.Stop()
			}
			if !keep {
				_ = os.RemoveAll(workDir)
			}
			os.Exit(signalExitCode(sig))
		case <-stopCh:
		}
	}()
	return func() {
		signal.Stop(sigCh)
		close(stopCh)
	}
}

func signalExitCode(sig os.Signal) int {
	switch sig {
	case syscall.SIGTERM:
		return 143
	default:
		return 130
	}
}

func cleanupStaleLocalE2EVMRunners(hostAgentBin string) {
	hostAgentBin = strings.TrimSpace(hostAgentBin)
	if hostAgentBin == "" {
		return
	}
	abs, err := filepath.Abs(hostAgentBin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn resolve host-agent for stale local e2e runner cleanup: %v\n", err)
		return
	}
	cmd := exec.Command("sudo", "-n", abs, "--cleanup-vm-runners-manifest-prefix", localE2ETempDirPrefix)
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "warn stale local e2e runner cleanup failed: %v\n%s", err, string(out))
	} else if len(out) > 0 {
		fmt.Print(string(out))
	}
}

func stopExistingHostAgentDaemon(hostAgentBin string) {
	hostAgentBin = strings.TrimSpace(hostAgentBin)
	if hostAgentBin == "" {
		return
	}
	abs, err := filepath.Abs(hostAgentBin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn resolve host-agent for daemon restart: %v\n", err)
		return
	}
	cmd := exec.Command("sudo", "-n", abs, "--stop-existing")
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "warn stop existing host-agent daemon failed: %v\n%s", err, string(out))
	} else if len(out) > 0 {
		fmt.Print(string(out))
	}
}

const localE2ETempDirPrefix = "/tmp/protos-local-macos-e2e-"

func cleanupStaleLocalE2EVMs(currentWorkDir string) {
	matches := map[string]struct{}{}
	for _, prefix := range pathPrefixVariants(localE2ETempDirPrefix) {
		pattern := prefix + "*"
		found, err := filepath.Glob(pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn list stale local e2e workdirs for %s: %v\n", pattern, err)
			continue
		}
		for _, match := range found {
			matches[match] = struct{}{}
		}
	}
	currentWorkDirs := pathPrefixVariants(currentWorkDir)
	for workDir := range matches {
		if equivalentPath(workDir, currentWorkDirs) {
			continue
		}
		vmDir := filepath.Join(workDir, "local-macos-vms")
		if _, err := os.Stat(vmDir); err != nil {
			continue
		}
		if err := cleanupHostAgentVMs(vmDir); err != nil {
			fmt.Fprintf(os.Stderr, "warn cleanup stale local e2e VMs in %s failed: %v\n", vmDir, err)
		}
	}
}

func cleanupHostAgentVMs(vmDir string) error {
	vmDir = strings.TrimSpace(vmDir)
	if vmDir == "" {
		return nil
	}
	if _, err := os.Stat(vmDir); err != nil {
		return nil
	}
	client, err := hostagentclient.New()
	if err != nil {
		return err
	}
	defer client.Close()

	states, err := client.ListVMs(vmDir)
	if err != nil {
		return err
	}
	var cleanupErr error
	for i := len(states) - 1; i >= 0; i-- {
		state := states[i]
		if state.GetStatus() == "error" {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("%s: %s", state.GetId(), state.GetMessage()))
			continue
		}
		id := strings.TrimSpace(state.GetId())
		if id == "" && state.GetConfig() != nil {
			id = strings.TrimSpace(state.GetConfig().GetId())
		}
		if id == "" {
			continue
		}
		if _, err := client.ApplyVM(id, vmDir, "deleted", nil); err != nil {
			cleanupErr = errors.Join(cleanupErr, err)
		}
	}
	return cleanupErr
}

func equivalentPath(path string, candidates []string) bool {
	for _, candidate := range candidates {
		for _, variant := range pathPrefixVariants(path) {
			if variant == candidate {
				return true
			}
		}
	}
	return false
}

func pathPrefixVariants(path string) []string {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	abs, err := filepath.Abs(path)
	if err == nil {
		path = abs
	}
	cleaned := filepath.Clean(path)
	variants := []string{cleaned}
	if strings.HasPrefix(cleaned, "/tmp/") {
		variants = append(variants, "/private"+cleaned)
	}
	if strings.HasPrefix(cleaned, "/private/tmp/") {
		variants = append(variants, strings.TrimPrefix(cleaned, "/private"))
	}
	return uniqueStrings(variants)
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func verifyLocalImageRegistry(deadline time.Time, client pbApic.ProtosClientApiClient, workDir string, deployed []*pbApic.CloudInstance, appImage string, suffix string) error {
	if len(deployed) < 2 {
		return fmt.Errorf("image registry verification requires at least two local VMs")
	}
	appImage = strings.TrimSpace(appImage)
	if appImage == "" {
		return fmt.Errorf("app image is empty")
	}
	seedVM := deployed[0]
	pullVM := deployed[1]
	seedAppName := "image-seed-" + suffix
	pullAppName := "image-pull-" + suffix

	if err := e2eapic.CreateAndStartApp(deadline, client, appImage, seedAppName, seedVM.GetVmId()); err != nil {
		return fmt.Errorf("create/start seed app: %w", err)
	}
	if err := e2eapic.WaitForLocalCheckpoint(deadline, client, "seed app start"); err != nil {
		return err
	}
	if err := e2eapic.WaitForAllRemoteHeads(deadline, client, deployed); err != nil {
		return err
	}
	if _, err := e2eapic.WaitForAppStatus(deadline, client, seedVM.GetName(), seedAppName, "running"); err != nil {
		captureImageRegistryDiagnostics(deadline, client, workDir, seedVM.GetName(), seedAppName, err)
		return err
	}
	if err := e2eapic.WaitForRemoteImageContentReady(deadline, client, seedVM.GetName(), appImage); err != nil {
		captureImageRegistryDiagnostics(deadline, client, workDir, seedVM.GetName(), seedAppName, err)
		return err
	}

	if err := e2eapic.CreateAndStartApp(deadline, client, appImage, pullAppName, pullVM.GetVmId()); err != nil {
		return fmt.Errorf("create/start pull app: %w", err)
	}
	if err := e2eapic.WaitForLocalCheckpoint(deadline, client, "pull app start"); err != nil {
		return err
	}
	if err := e2eapic.WaitForAllRemoteHeads(deadline, client, deployed); err != nil {
		return err
	}
	if _, err := e2eapic.WaitForAppStatus(deadline, client, pullVM.GetName(), pullAppName, "running"); err != nil {
		captureImageRegistryDiagnostics(deadline, client, workDir, pullVM.GetName(), pullAppName, err)
		return err
	}
	if err := e2eapic.WaitForRemoteP2PImageLabel(deadline, client, pullVM.GetName(), appImage); err != nil {
		captureImageRegistryDiagnostics(deadline, client, workDir, pullVM.GetName(), pullAppName, err)
		return err
	}
	fmt.Printf("Protos P2P image resolution verified: seed=%s puller=%s image=%s\n", seedVM.GetName(), pullVM.GetName(), appImage)
	return nil
}

func captureImageRegistryDiagnostics(deadline time.Time, client pbApic.ProtosClientApiClient, workDir string, instanceName string, appName string, cause error) {
	if client == nil || strings.TrimSpace(workDir) == "" {
		return
	}
	dir := filepath.Join(workDir, "e2e-diagnostics")
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "warn create e2e diagnostics dir: %v\n", err)
		return
	}
	path := filepath.Join(dir, sanitizeDiagnosticName(instanceName+"-"+appName)+".txt")
	var out strings.Builder
	fmt.Fprintf(&out, "captured_at: %s\n", time.Now().UTC().Format(time.RFC3339Nano))
	fmt.Fprintf(&out, "instance: %s\n", instanceName)
	fmt.Fprintf(&out, "app: %s\n", appName)
	fmt.Fprintf(&out, "cause: %v\n\n", cause)

	ctx, cancel := diagnosticContext(deadline, 15*time.Second)
	apps, err := client.GetApps(ctx, &pbApic.GetAppsRequest{})
	cancel()
	appendDiagnosticProto(&out, "GetApps", apps, err)

	ctx, cancel = diagnosticContext(deadline, 15*time.Second)
	sql, err := client.ExecuteSql(ctx, &pbApic.ExecuteSqlRequest{
		Sql:     "select name, instance_id, desired_status, installer_ref from apps",
		MaxRows: 50,
	})
	cancel()
	appendDiagnosticProto(&out, "ExecuteSql apps", sql, err)

	ctx, cancel = diagnosticContext(deadline, 15*time.Second)
	tasks, err := client.GetTasks(ctx, &pbApic.GetTasksRequest{
		Instance:   instanceName,
		Stream:     "apps.runtime.reconcile",
		MaxResults: 20,
	})
	cancel()
	appendDiagnosticProto(&out, "GetTasks apps.runtime.reconcile", tasks, err)

	ctx, cancel = diagnosticContext(deadline, 30*time.Second)
	logs, err := client.GetInstanceLogs(ctx, &pbApic.GetInstanceLogsRequest{Name: instanceName})
	cancel()
	if err != nil {
		fmt.Fprintf(&out, "\n## GetInstanceLogs error\n%v\n", err)
	} else {
		fmt.Fprintf(&out, "\n## GetInstanceLogs\n%s\n", logs.GetLogs())
	}

	if err := os.WriteFile(path, []byte(out.String()), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "warn write e2e diagnostics %s: %v\n", path, err)
		return
	}
	fmt.Printf("captured image registry diagnostics: %s\n", path)
}

func appendDiagnosticProto(out *strings.Builder, label string, msg proto.Message, err error) {
	fmt.Fprintf(out, "## %s\n", label)
	if err != nil {
		fmt.Fprintf(out, "error: %v\n\n", err)
		return
	}
	if msg == nil {
		out.WriteString("nil\n\n")
		return
	}
	data, marshalErr := protojson.MarshalOptions{
		Multiline:       true,
		Indent:          "  ",
		EmitUnpopulated: false,
	}.Marshal(msg)
	if marshalErr != nil {
		fmt.Fprintf(out, "marshal error: %v\n\n", marshalErr)
		return
	}
	out.Write(data)
	out.WriteString("\n\n")
}

func sanitizeDiagnosticName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "diagnostics"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return b.String()
}

func diagnosticContext(deadline time.Time, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	target := time.Now().Add(timeout)
	if !deadline.IsZero() && deadline.Before(target) {
		target = deadline
	}
	if time.Until(target) <= 0 {
		return context.WithTimeout(context.Background(), time.Second)
	}
	return context.WithDeadline(context.Background(), target)
}
