//go:build darwin

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/protosio/protos/integration-tests/e2eapic"
	localmacos "github.com/protosio/protos/provisioners/local_macos"
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
	}
	cancel()

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	providerName := "local-e2e-" + suffix
	imageName := "mactest-e2e-" + suffix
	vmDir := filepath.Join(workDir, "local-macos-vms")
	deadline := time.Now().Add(timeout)

	var cleanupInstances []string
	cleanupImage := false
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cleanupCancel()
		for i := len(cleanupInstances) - 1; i >= 0; i-- {
			_, _ = client.RemoveInstance(cleanupCtx, &pbApic.RemoveInstanceRequest{Name: cleanupInstances[i]})
		}
		if cleanupImage {
			_, _ = client.RemoveProvisionerImage(cleanupCtx, &pbApic.RemoveProvisionerImageRequest{
				ImageName:       imageName,
				ProvisionerName: providerName,
				Location:        "local",
			})
		}
		_, _ = client.RemoveProvisioner(cleanupCtx, &pbApic.RemoveProvisionerRequest{Name: providerName})
	}()

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

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Minute)
	if _, err := client.UploadProvisionerImage(ctx, &pbApic.UploadProvisionerImageRequest{
		ImagePath:       imagePath,
		ImageName:       imageName,
		ProvisionerName: providerName,
		Location:        "local",
	}); err != nil {
		cancel()
		return fmt.Errorf("upload local image: %w", err)
	}
	cancel()
	cleanupImage = true

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
		cleanupInstances = append(cleanupInstances, instanceName)

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
		if err := verifyLocalImageRegistry(deadline, client, deployed, appImage, suffix); err != nil {
			return err
		}
	}

	return nil
}

func verifyLocalImageRegistry(deadline time.Time, client pbApic.ProtosClientApiClient, deployed []*pbApic.CloudInstance, appImage string, suffix string) error {
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
		return err
	}
	if err := e2eapic.WaitForRemoteImageContentReady(deadline, client, seedVM.GetName(), appImage); err != nil {
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
		return err
	}
	if err := e2eapic.WaitForRemoteP2PImageLabel(deadline, client, pullVM.GetName(), appImage); err != nil {
		return err
	}
	fmt.Printf("Protos P2P image resolution verified: seed=%s puller=%s image=%s\n", seedVM.GetName(), pullVM.GetName(), appImage)
	return nil
}
