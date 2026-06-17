//go:build darwin

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/protosio/protos/integration-tests/e2eapic"
	"github.com/protosio/protos/provisioners/hetzner"
	localmacos "github.com/protosio/protos/provisioners/local_macos"
	"github.com/protosio/protos/provisioners/scaleway"
)

const (
	defaultProbeAppImage = "docker.io/protosio/protos-e2e-probe:latest"

	localClientPriority = 50
	localVMPriority     = 30
	cloudVMPriority     = 100
)

type appConnectivityPair struct {
	hetznerApp  *pbApic.App
	scalewayApp *pbApic.App
}

func main() {
	imagePath := flag.String("image", "../cloud-provisioning/targets/output/mactest", "local macOS LinuxKit image directory or ISO path")
	flutterApp := flag.String("flutter-app", "", "built Protos macOS Flutter app bundle")
	hostAgentBin := flag.String("hostagent-bin", "../bin/protos-hostagent", "protos-hostagent binary used by the Flutter node")
	workDir := flag.String("workdir", "", "temporary Protos workdir")
	keep := flag.Bool("keep", false, "keep temporary state after the run")
	timeout := flag.Duration("timeout", 45*time.Minute, "overall verification timeout")
	imageUploadTimeout := flag.Duration("image-upload-timeout", 30*time.Minute, "per-provider cloud image upload timeout")
	localMachine := flag.String("local-machine", "vz-2c-2g", "local macOS machine type")
	hetznerImage := flag.String("hetzner-image", "../cloud-provisioning/targets/output/hetzner/hetzner-bios.img", "Hetzner BIOS raw disk image to upload")
	hetznerMachine := flag.String("hetzner-machine", "cpx11", "Hetzner machine type")
	hetznerLocation := flag.String("hetzner-location", "ash", "Hetzner location")
	hetznerEnv := flag.String("hetzner-env", ".env-hetzner", "Hetzner credential env file")
	scalewayImage := flag.String("scaleway-image", "../cloud-provisioning/targets/output/scaleway/scaleway-efi.iso", "Scaleway EFI image to upload")
	scalewayMachine := flag.String("scaleway-machine", "DEV1-S", "Scaleway machine type")
	scalewayLocation := flag.String("scaleway-location", "fr-par-1", "Scaleway zone")
	scalewayEnv := flag.String("scaleway-env", ".env-scaleway", "Scaleway credential env file")
	appImage := flag.String("app-image", defaultProbeAppImage, "container image used for app connectivity checks; must run protos-e2e-probe on port 8080")
	seedImageArchive := flag.String("seed-image-archive", "", "optional local image tar or tar.gz archive to upload to the Hetzner seed VM before app start")
	summaryArtifact := flag.String("summary-artifact", "", "path for the mixed-cloud run summary JSON artifact")
	configureNetwork := flag.Bool("network", true, "configure the host network module through protos-hostagent")
	flag.Parse()

	cfg := harnessConfig{
		imagePath:          *imagePath,
		flutterApp:         *flutterApp,
		hostAgentBin:       *hostAgentBin,
		workDir:            *workDir,
		keep:               *keep,
		timeout:            *timeout,
		imageUploadTimeout: *imageUploadTimeout,
		localMachine:       *localMachine,
		hetznerImage:       *hetznerImage,
		hetznerMachine:     *hetznerMachine,
		hetznerLocation:    *hetznerLocation,
		hetznerEnv:         *hetznerEnv,
		scalewayImage:      *scalewayImage,
		scalewayMachine:    *scalewayMachine,
		scalewayLocation:   *scalewayLocation,
		scalewayEnv:        *scalewayEnv,
		appImage:           *appImage,
		seedImageArchive:   *seedImageArchive,
		summaryArtifact:    *summaryArtifact,
		configureNetwork:   *configureNetwork,
	}
	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "mixed cloud e2e failed: %v\n", err)
		os.Exit(1)
	}
}

type harnessConfig struct {
	imagePath          string
	flutterApp         string
	hostAgentBin       string
	workDir            string
	keep               bool
	timeout            time.Duration
	imageUploadTimeout time.Duration
	localMachine       string
	hetznerImage       string
	hetznerMachine     string
	hetznerLocation    string
	hetznerEnv         string
	scalewayImage      string
	scalewayMachine    string
	scalewayLocation   string
	scalewayEnv        string
	appImage           string
	seedImageArchive   string
	summaryArtifact    string
	configureNetwork   bool
}

type imageRef struct {
	provider string
	name     string
	location string
	id       string
}

type mixedCloudRunSummary struct {
	Path             string                      `json:"-"`
	StartedAt        string                      `json:"started_at"`
	FinishedAt       string                      `json:"finished_at,omitempty"`
	Status           string                      `json:"status"`
	Error            string                      `json:"error,omitempty"`
	WorkDir          string                      `json:"work_dir"`
	Suffix           string                      `json:"suffix"`
	ImageName        string                      `json:"image_name"`
	AppImage         string                      `json:"app_image"`
	SeedImageArchive string                      `json:"seed_image_archive,omitempty"`
	Providers        []mixedCloudSummaryProvider `json:"providers"`
	Images           []mixedCloudSummaryImage    `json:"images"`
	Instances        []mixedCloudSummaryInstance `json:"instances"`
	Cleanup          []mixedCloudSummaryCleanup  `json:"cleanup"`
}

type mixedCloudSummaryProvider struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Location string `json:"location,omitempty"`
	Status   string `json:"status"`
}

type mixedCloudSummaryImage struct {
	Name     string `json:"name"`
	ID       string `json:"id,omitempty"`
	Provider string `json:"provider"`
	Location string `json:"location"`
	Status   string `json:"status"`
}

type mixedCloudSummaryInstance struct {
	Name         string `json:"name"`
	ID           string `json:"id,omitempty"`
	PeerID       string `json:"peer_id,omitempty"`
	Provider     string `json:"provider"`
	Location     string `json:"location"`
	MachineType  string `json:"machine_type"`
	PublicIP     string `json:"public_ip,omitempty"`
	Architecture string `json:"architecture,omitempty"`
	Status       string `json:"status"`
}

type mixedCloudSummaryCleanup struct {
	ResourceType string `json:"resource_type"`
	Name         string `json:"name"`
	ID           string `json:"id,omitempty"`
	Provider     string `json:"provider,omitempty"`
	Location     string `json:"location,omitempty"`
	Status       string `json:"status"`
	Error        string `json:"error,omitempty"`
}

func newMixedCloudRunSummary(path string, workDir string, suffix string, imageName string, cfg harnessConfig) (*mixedCloudRunSummary, error) {
	if strings.TrimSpace(path) == "" {
		path = filepath.Join(os.TempDir(), "protos-mixed-cloud-e2e-summary-"+suffix+".json")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return &mixedCloudRunSummary{
		Path:             absPath,
		StartedAt:        time.Now().UTC().Format(time.RFC3339Nano),
		Status:           "running",
		WorkDir:          workDir,
		Suffix:           suffix,
		ImageName:        imageName,
		AppImage:         cfg.appImage,
		SeedImageArchive: cfg.seedImageArchive,
	}, nil
}

func (summary *mixedCloudRunSummary) addProvider(name string, typ string, location string) {
	if summary == nil {
		return
	}
	summary.Providers = append(summary.Providers, mixedCloudSummaryProvider{Name: name, Type: typ, Location: location, Status: "configured"})
}

func (summary *mixedCloudRunSummary) setProviderStatus(name string, status string) {
	if summary == nil {
		return
	}
	for i := range summary.Providers {
		if summary.Providers[i].Name == name {
			summary.Providers[i].Status = status
			return
		}
	}
}

func (summary *mixedCloudRunSummary) addImage(provider string, name string, location string, id string) {
	if summary == nil {
		return
	}
	summary.Images = append(summary.Images, mixedCloudSummaryImage{Name: name, ID: id, Provider: provider, Location: location, Status: "uploaded"})
}

func (summary *mixedCloudRunSummary) setImageStatus(provider string, name string, location string, status string) {
	if summary == nil {
		return
	}
	for i := range summary.Images {
		image := summary.Images[i]
		if image.Provider == provider && image.Name == name && image.Location == location {
			summary.Images[i].Status = status
			return
		}
	}
}

func (summary *mixedCloudRunSummary) addInstance(instance *pbApic.CloudInstance, machineType string, peerID string, status string) {
	if summary == nil || instance == nil {
		return
	}
	summary.Instances = append(summary.Instances, mixedCloudSummaryInstance{
		Name:         instance.GetName(),
		ID:           instance.GetVmId(),
		PeerID:       peerID,
		Provider:     instance.GetCloudName(),
		Location:     instance.GetLocation(),
		MachineType:  machineType,
		PublicIP:     instance.GetPublicIp(),
		Architecture: instance.GetArchitecture(),
		Status:       status,
	})
}

func (summary *mixedCloudRunSummary) setInstanceStatus(name string, status string) {
	if summary == nil {
		return
	}
	for i := range summary.Instances {
		if summary.Instances[i].Name == name {
			summary.Instances[i].Status = status
			return
		}
	}
}

func (summary *mixedCloudRunSummary) recordCleanup(resourceType string, name string, id string, provider string, location string, err error) {
	if summary == nil {
		return
	}
	item := mixedCloudSummaryCleanup{
		ResourceType: resourceType,
		Name:         name,
		ID:           id,
		Provider:     provider,
		Location:     location,
		Status:       "removed",
	}
	if err != nil {
		item.Status = "failed"
		item.Error = err.Error()
	}
	summary.Cleanup = append(summary.Cleanup, item)
}

func (summary *mixedCloudRunSummary) finish(runErr error) {
	if summary == nil {
		return
	}
	summary.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if runErr != nil {
		summary.Status = "failed"
		summary.Error = runErr.Error()
		return
	}
	summary.Status = "passed"
}

func (summary *mixedCloudRunSummary) write() error {
	if summary == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(summary.Path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(summary.Path, data, 0644); err != nil {
		return err
	}
	fmt.Printf("mixed cloud e2e summary artifact: %s\n", summary.Path)
	return nil
}

func run(cfg harnessConfig) (runErr error) {
	imagePath, err := filepath.Abs(cfg.imagePath)
	if err != nil {
		return err
	}
	hetznerImagePath, err := uploadImagePath(cfg.hetznerImage, "hetzner-bios.img")
	if err != nil {
		return fmt.Errorf("resolve Hetzner image: %w", err)
	}
	scalewayImagePath, err := uploadImagePath(cfg.scalewayImage, "scaleway-efi.iso")
	if err != nil {
		return fmt.Errorf("resolve Scaleway image: %w", err)
	}
	workDir := cfg.workDir
	if workDir == "" {
		workDir, err = os.MkdirTemp("/tmp", "protos-mixed-cloud-e2e-*")
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
	if !cfg.keep {
		defer os.RemoveAll(workDir)
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	localProviderName := "local-e2e-" + suffix
	hetznerProviderName := "hetzner-e2e-" + suffix
	scalewayProviderName := "scaleway-e2e-" + suffix
	imageName := "mixed-cloud-e2e-" + suffix
	vmDir := filepath.Join(workDir, "local-macos-vms")
	summary, err := newMixedCloudRunSummary(cfg.summaryArtifact, workDir, suffix, imageName, cfg)
	if err != nil {
		return fmt.Errorf("create mixed cloud run summary: %w", err)
	}
	defer func() {
		summary.finish(runErr)
		if err := summary.write(); err != nil {
			if runErr == nil {
				runErr = fmt.Errorf("write mixed cloud summary artifact: %w", err)
				return
			}
			fmt.Fprintf(os.Stderr, "failed to write mixed cloud summary artifact: %v\n", err)
		}
	}()

	node, err := e2eapic.StartFlutterNode(e2eapic.FlutterNodeOptions{
		WorkDir:          workDir,
		AppPath:          cfg.flutterApp,
		HostAgentBin:     cfg.hostAgentBin,
		ConfigureNetwork: cfg.configureNetwork,
	})
	if err != nil {
		return err
	}
	defer func() {
		_ = node.Stop()
	}()
	client := node.Client

	startupCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	if err := e2eapic.Init(startupCtx, client, "e2e", "Mixed Cloud E2E", "Mixed Cloud E2E"); err != nil {
		cancel()
		return fmt.Errorf("initialize Flutter local node: %w", err)
	}
	if cfg.configureNetwork {
		if err := e2eapic.StartHostAgent(startupCtx, client); err != nil {
			cancel()
			return fmt.Errorf("start host agent through APIC: %w", err)
		}
	}
	cancel()

	deadline := time.Now().Add(cfg.timeout)

	var cleanupInstances []string
	var cleanupImages []imageRef
	var cleanupProviders []string
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cleanupCancel()
		for i := len(cleanupInstances) - 1; i >= 0; i-- {
			cleanupName := cleanupInstances[i]
			_, err := client.RemoveInstance(cleanupCtx, &pbApic.RemoveInstanceRequest{Name: cleanupName})
			summary.recordCleanup("instance", cleanupName, "", "", "", err)
		}
		for _, image := range cleanupImages {
			_, err := client.RemoveProvisionerImage(cleanupCtx, &pbApic.RemoveProvisionerImageRequest{
				ImageName:       image.name,
				ProvisionerName: image.provider,
				Location:        image.location,
			})
			summary.recordCleanup("image", image.name, image.id, image.provider, image.location, err)
			if err == nil {
				summary.setImageStatus(image.provider, image.name, image.location, "removed")
			}
		}
		for _, providerName := range cleanupProviders {
			_, err := client.RemoveProvisioner(cleanupCtx, &pbApic.RemoveProvisionerRequest{Name: providerName})
			summary.recordCleanup("provisioner", providerName, "", providerName, "", err)
			if err == nil {
				summary.setProviderStatus(providerName, "removed")
			}
		}
	}()
	removeCleanupInstance := func(name string) {
		for i, cleanupName := range cleanupInstances {
			if cleanupName != name {
				continue
			}
			cleanupInstances = append(cleanupInstances[:i], cleanupInstances[i+1:]...)
			return
		}
	}

	if err := addProvisioner(client, localProviderName, localmacos.Type.String(), map[string]string{"VM_DIR": vmDir}); err != nil {
		return fmt.Errorf("add local macOS provisioner: %w", err)
	}
	summary.addProvider(localProviderName, localmacos.Type.String(), "local")
	cleanupProviders = append(cleanupProviders, localProviderName)
	localImageID, err := uploadProvisionerImage(client, imagePath, imageName, localProviderName, "local", cfg.imageUploadTimeout)
	if err != nil {
		return fmt.Errorf("upload local macOS image: %w", err)
	}
	summary.addImage(localProviderName, imageName, "local", localImageID)
	cleanupImages = append(cleanupImages, imageRef{provider: localProviderName, name: imageName, location: "local", id: localImageID})

	hetznerCredentials, err := hetznerAuth(cfg.hetznerEnv)
	if err != nil {
		return err
	}
	if err := addProvisioner(client, hetznerProviderName, hetzner.Type.String(), hetznerCredentials); err != nil {
		return fmt.Errorf("add Hetzner provisioner: %w", err)
	}
	summary.addProvider(hetznerProviderName, hetzner.Type.String(), cfg.hetznerLocation)
	cleanupProviders = append(cleanupProviders, hetznerProviderName)
	hetznerImageID, err := uploadProvisionerImage(client, hetznerImagePath, imageName, hetznerProviderName, cfg.hetznerLocation, cfg.imageUploadTimeout)
	if err != nil {
		return fmt.Errorf("upload Hetzner image: %w", err)
	}
	summary.addImage(hetznerProviderName, imageName, cfg.hetznerLocation, hetznerImageID)
	cleanupImages = append(cleanupImages, imageRef{provider: hetznerProviderName, name: imageName, location: cfg.hetznerLocation, id: hetznerImageID})

	scalewayCredentials, err := scalewayAuth(cfg.scalewayEnv)
	if err != nil {
		return err
	}
	if err := addProvisioner(client, scalewayProviderName, scaleway.Type.String(), scalewayCredentials); err != nil {
		return fmt.Errorf("add Scaleway provisioner: %w", err)
	}
	summary.addProvider(scalewayProviderName, scaleway.Type.String(), cfg.scalewayLocation)
	cleanupProviders = append(cleanupProviders, scalewayProviderName)
	scalewayImageID, err := uploadProvisionerImage(client, scalewayImagePath, imageName, scalewayProviderName, cfg.scalewayLocation, cfg.imageUploadTimeout)
	if err != nil {
		return fmt.Errorf("upload Scaleway image: %w", err)
	}
	summary.addImage(scalewayProviderName, imageName, cfg.scalewayLocation, scalewayImageID)
	cleanupImages = append(cleanupImages, imageRef{provider: scalewayProviderName, name: imageName, location: cfg.scalewayLocation, id: scalewayImageID})

	localState, err := e2eapic.RuntimeState(deadline, client, "")
	if err != nil {
		return fmt.Errorf("read local runtime state: %w", err)
	}
	localPeerID := localState.GetPeerId()
	if strings.TrimSpace(localPeerID) == "" {
		return fmt.Errorf("local runtime state did not expose a peer id")
	}
	if err := e2eapic.WaitForReplicationState(deadline, client, e2eapic.ReplicationExpectation{
		Label: "initial local client",
		Priorities: map[string]int{
			localPeerID: localClientPriority,
		},
	}); err != nil {
		return err
	}

	var deployed []*pbApic.CloudInstance
	deploy := func(name string, provider string, location string, machine string) (*pbApic.CloudInstance, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		_, err := client.DeployInstance(ctx, &pbApic.DeployInstanceRequest{
			Name:          name,
			CloudName:     provider,
			CloudLocation: location,
			MachineType:   machine,
			DevImg:        imageName,
		})
		cancel()
		if err != nil {
			return nil, err
		}
		cleanupInstances = append(cleanupInstances, name)
		instance, err := e2eapic.WaitForInstanceReady(deadline, client, name)
		if err != nil {
			return nil, err
		}
		deployed = append(deployed, instance)
		peerID, peerErr := e2eapic.PeerIDForInstance(instance)
		if peerErr != nil {
			return nil, peerErr
		}
		summary.addInstance(instance, machine, peerID, "ready")
		fmt.Printf("deployed instance: name=%s id=%s provider=%s ip=%s arch=%s status=%s\n", instance.GetName(), instance.GetVmId(), provider, instance.GetPublicIp(), instance.GetArchitecture(), instance.GetStatus())
		return instance, nil
	}
	recordInstanceDeleted := func(instance *pbApic.CloudInstance) {
		if instance == nil {
			return
		}
		summary.setInstanceStatus(instance.GetName(), "deleted")
		summary.recordCleanup("instance", instance.GetName(), instance.GetVmId(), instance.GetCloudName(), instance.GetLocation(), nil)
	}

	localVM1, err := deploy("local-vm-1-"+suffix, localProviderName, "local", cfg.localMachine)
	if err != nil {
		return fmt.Errorf("deploy first local macOS VM: %w", err)
	}
	localVM1PeerID, err := e2eapic.PeerIDForInstance(localVM1)
	if err != nil {
		return err
	}
	if err := e2eapic.WaitForReplicationState(deadline, client, e2eapic.ReplicationExpectation{
		Label: "first local VM stays fallback below laptop",
		Priorities: map[string]int{
			localPeerID:    localClientPriority,
			localVM1PeerID: localVMPriority,
		},
	}); err != nil {
		return err
	}

	localVM2, err := deploy("local-vm-2-"+suffix, localProviderName, "local", cfg.localMachine)
	if err != nil {
		return fmt.Errorf("deploy second local macOS VM: %w", err)
	}
	localVM2PeerID, err := e2eapic.PeerIDForInstance(localVM2)
	if err != nil {
		return err
	}
	if err := e2eapic.WaitForReplicationState(deadline, client, e2eapic.ReplicationExpectation{
		Label: "local laptop remains above local VMs",
		Priorities: map[string]int{
			localPeerID:    localClientPriority,
			localVM1PeerID: localVMPriority,
			localVM2PeerID: localVMPriority,
		},
	}); err != nil {
		return err
	}

	hetznerVM, err := deploy("hetzner-vm-"+suffix, hetznerProviderName, cfg.hetznerLocation, cfg.hetznerMachine)
	if err != nil {
		return fmt.Errorf("deploy Hetzner VM: %w", err)
	}
	hetznerPeerID, err := e2eapic.PeerIDForInstance(hetznerVM)
	if err != nil {
		return err
	}
	if err := e2eapic.WaitForReplicationState(deadline, client, e2eapic.ReplicationExpectation{
		Label: "single Hetzner cloud VM records cloud priority metadata",
		Priorities: map[string]int{
			localPeerID:    localClientPriority,
			localVM1PeerID: localVMPriority,
			localVM2PeerID: localVMPriority,
			hetznerPeerID:  cloudVMPriority,
		},
	}); err != nil {
		return err
	}

	scalewayVM, err := deploy("scaleway-vm-"+suffix, scalewayProviderName, cfg.scalewayLocation, cfg.scalewayMachine)
	if err != nil {
		return fmt.Errorf("deploy Scaleway VM: %w", err)
	}
	scalewayPeerID, err := e2eapic.PeerIDForInstance(scalewayVM)
	if err != nil {
		return err
	}
	if err := e2eapic.WaitForReplicationState(deadline, client, e2eapic.ReplicationExpectation{
		Label: "two cloud VMs record cloud priority metadata",
		Priorities: map[string]int{
			localPeerID:    localClientPriority,
			localVM1PeerID: localVMPriority,
			localVM2PeerID: localVMPriority,
			hetznerPeerID:  cloudVMPriority,
			scalewayPeerID: cloudVMPriority,
		},
	}); err != nil {
		return err
	}

	if err := e2eapic.WaitForRemotePeerConnection(deadline, client, deployed, localPeerID); err != nil {
		return err
	}
	if err := e2eapic.WaitForRemoteRuntimeConnection(deadline, client, deployed, localPeerID); err != nil {
		return err
	}
	if err := e2eapic.WaitForAllRemoteHeads(deadline, client, deployed); err != nil {
		return fmt.Errorf("post-checkpoint DB sync failed: %w", err)
	}

	appPair, err := createCloudAppConnectivityPair(cfg, deadline, client, deployed, hetznerVM, scalewayVM, suffix)
	if err != nil {
		return err
	}
	if err := e2eapic.WaitForLocalCheckpoint(deadline, client, "cloud app deployment writes"); err != nil {
		return err
	}
	if err := e2eapic.WaitForRemotePeerConnection(deadline, client, deployed, localPeerID); err != nil {
		return err
	}
	if err := e2eapic.WaitForRemoteRuntimeConnection(deadline, client, deployed, localPeerID); err != nil {
		return err
	}
	if err := e2eapic.WaitForAllRemoteHeads(deadline, client, deployed); err != nil {
		return err
	}
	if err := verifyCloudAppConnectivity(deadline, hetznerVM, scalewayVM, appPair); err != nil {
		return err
	}

	if err := stopInstanceAndVerifyPeerShutdown(deadline, client, localVM1, []*pbApic.CloudInstance{localVM2, hetznerVM, scalewayVM}); err != nil {
		return err
	}
	summary.setInstanceStatus(localVM1.GetName(), "stopped-peer-removed")

	if err := deleteInstanceAndVerify(deadline, client, localVM1, removeCleanupInstance); err != nil {
		return err
	}
	recordInstanceDeleted(localVM1)
	if err := e2eapic.WaitForReplicationState(deadline, client, e2eapic.ReplicationExpectation{
		Label: "cloud-priority metadata remains after first local VM deletion",
		Priorities: map[string]int{
			localPeerID:    localClientPriority,
			localVM2PeerID: localVMPriority,
			hetznerPeerID:  cloudVMPriority,
			scalewayPeerID: cloudVMPriority,
		},
	}); err != nil {
		return err
	}

	if err := deleteInstanceAndVerify(deadline, client, localVM2, removeCleanupInstance); err != nil {
		return err
	}
	recordInstanceDeleted(localVM2)
	if err := e2eapic.WaitForReplicationState(deadline, client, e2eapic.ReplicationExpectation{
		Label: "cloud-priority metadata remains after second local VM deletion",
		Priorities: map[string]int{
			localPeerID:    localClientPriority,
			hetznerPeerID:  cloudVMPriority,
			scalewayPeerID: cloudVMPriority,
		},
	}); err != nil {
		return err
	}

	if err := deleteInstanceAndVerify(deadline, client, hetznerVM, removeCleanupInstance); err != nil {
		return err
	}
	recordInstanceDeleted(hetznerVM)
	if err := e2eapic.WaitForReplicationState(deadline, client, e2eapic.ReplicationExpectation{
		Label: "single remaining cloud VM keeps cloud priority metadata after Hetzner deletion",
		Priorities: map[string]int{
			localPeerID:    localClientPriority,
			scalewayPeerID: cloudVMPriority,
		},
	}); err != nil {
		return err
	}

	if err := deleteInstanceAndVerify(deadline, client, scalewayVM, removeCleanupInstance); err != nil {
		return err
	}
	recordInstanceDeleted(scalewayVM)
	if err := e2eapic.WaitForReplicationState(deadline, client, e2eapic.ReplicationExpectation{
		Label: "local client remains after all VM deletion",
		Priorities: map[string]int{
			localPeerID: localClientPriority,
		},
	}); err != nil {
		return err
	}
	if err := e2eapic.WaitForNoInstances(deadline, client); err != nil {
		return err
	}
	if err := e2eapic.WaitForNoApps(deadline, client); err != nil {
		return err
	}

	return nil
}

func stopInstanceAndVerifyPeerShutdown(deadline time.Time, client pbApic.ProtosClientApiClient, instance *pbApic.CloudInstance, observers []*pbApic.CloudInstance) error {
	peerID, err := e2eapic.PeerIDForInstance(instance)
	if err != nil {
		return err
	}
	fmt.Printf("stopping instance to verify daemon peer shutdown: name=%s id=%s peer=%s\n", instance.GetName(), instance.GetVmId(), peerID)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	_, err = client.StopInstance(ctx, &pbApic.StopInstanceRequest{Name: instance.GetName()})
	cancel()
	if err != nil {
		return fmt.Errorf("stop instance %s: %w", instance.GetName(), err)
	}
	if _, err := e2eapic.WaitForInstanceStatus(deadline, client, instance.GetName(), "stopped"); err != nil {
		return err
	}
	if err := e2eapic.WaitForPeerRemoved(deadline, client, peerID); err != nil {
		return err
	}
	if err := e2eapic.WaitForPeerRemovedFromRuntimes(deadline, client, observers, peerID); err != nil {
		return err
	}
	fmt.Printf("daemon shutdown peer removal verified: name=%s peer=%s\n", instance.GetName(), peerID)
	return nil
}

func addProvisioner(client pbApic.ProtosClientApiClient, name string, typ string, credentials map[string]string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	_, err := client.AddProvisioner(ctx, &pbApic.AddProvisionerRequest{
		Name:        name,
		Type:        typ,
		Credentials: credentials,
	})
	return err
}

func uploadProvisionerImage(client pbApic.ProtosClientApiClient, imagePath string, imageName string, provider string, location string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout + 5*time.Minute)
	if timeout <= 0 {
		deadline = time.Now().Add(35 * time.Minute)
	}
	return e2eapic.UploadProvisionerImage(deadline, client, imagePath, imageName, provider, location, timeout)
}

func createCloudAppConnectivityPair(
	cfg harnessConfig,
	deadline time.Time,
	client pbApic.ProtosClientApiClient,
	deployed []*pbApic.CloudInstance,
	hetznerVM *pbApic.CloudInstance,
	scalewayVM *pbApic.CloudInstance,
	suffix string,
) (*appConnectivityPair, error) {
	appImage := strings.TrimSpace(cfg.appImage)
	if appImage == "" {
		return nil, fmt.Errorf("app image cannot be empty")
	}
	if strings.TrimSpace(cfg.seedImageArchive) != "" {
		if err := e2eapic.UploadImageArchive(deadline, client, hetznerVM.GetName(), cfg.seedImageArchive, appImage); err != nil {
			return nil, err
		}
	}

	hetznerAppName := "app-hetzner-" + suffix
	if err := e2eapic.CreateAndStartApp(deadline, client, appImage, hetznerAppName, hetznerVM.GetVmId()); err != nil {
		return nil, fmt.Errorf("create/start Hetzner app: %w", err)
	}
	if err := syncAfterAppWrite(deadline, client, deployed, "cloud image seed app start"); err != nil {
		return nil, err
	}
	hetznerApp, err := e2eapic.WaitForAppStatus(deadline, client, hetznerVM.GetName(), hetznerAppName, "running")
	if err != nil {
		return nil, err
	}
	if err := e2eapic.WaitForRemoteImageContentReady(deadline, client, hetznerVM.GetName(), appImage); err != nil {
		return nil, err
	}
	if err := e2eapic.WaitForRemoteFullMesh(deadline, client, []*pbApic.CloudInstance{hetznerVM, scalewayVM}); err != nil {
		return nil, fmt.Errorf("cloud seed/puller p2p mesh did not become ready: %w", err)
	}

	scalewayAppName := "app-scaleway-" + suffix
	if err := e2eapic.CreateAndStartApp(deadline, client, appImage, scalewayAppName, scalewayVM.GetVmId()); err != nil {
		return nil, fmt.Errorf("create/start Scaleway app: %w", err)
	}
	if err := syncAfterAppWrite(deadline, client, deployed, "cloud image pull app start"); err != nil {
		return nil, err
	}
	scalewayApp, err := e2eapic.WaitForAppStatus(deadline, client, scalewayVM.GetName(), scalewayAppName, "running")
	if err != nil {
		return nil, err
	}
	if err := e2eapic.WaitForRemoteP2PImageLabel(deadline, client, scalewayVM.GetName(), appImage); err != nil {
		return nil, err
	}
	fmt.Printf("cloud P2P image resolution verified: seed=%s puller=%s image=%s\n", hetznerVM.GetName(), scalewayVM.GetName(), appImage)
	return &appConnectivityPair{hetznerApp: hetznerApp, scalewayApp: scalewayApp}, nil
}

func syncAfterAppWrite(deadline time.Time, client pbApic.ProtosClientApiClient, instances []*pbApic.CloudInstance, label string) error {
	if err := e2eapic.WaitForLocalCheckpoint(deadline, client, label); err != nil {
		return err
	}
	if err := e2eapic.WaitForAllRemoteHeads(deadline, client, instances); err != nil {
		return fmt.Errorf("%s DB sync failed: %w", label, err)
	}
	return nil
}

func verifyCloudAppConnectivity(deadline time.Time, hetznerVM *pbApic.CloudInstance, scalewayVM *pbApic.CloudInstance, appPair *appConnectivityPair) error {
	if appPair == nil || appPair.hetznerApp == nil || appPair.scalewayApp == nil {
		return fmt.Errorf("cloud app connectivity pair is required")
	}
	if err := e2eapic.WaitForContainerHTTPToApp(deadline, hetznerVM, appPair.hetznerApp, appPair.scalewayApp); err != nil {
		return err
	}
	if err := e2eapic.WaitForContainerHTTPToApp(deadline, scalewayVM, appPair.scalewayApp, appPair.hetznerApp); err != nil {
		return err
	}
	return nil
}

func deleteInstanceAndVerify(deadline time.Time, client pbApic.ProtosClientApiClient, instance *pbApic.CloudInstance, removeCleanup func(string)) error {
	fmt.Printf("deleting instance: name=%s id=%s provider=%s\n", instance.GetName(), instance.GetVmId(), instance.GetCloudName())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	_, err := client.RemoveInstance(ctx, &pbApic.RemoveInstanceRequest{Name: instance.GetName()})
	cancel()
	if err != nil {
		return fmt.Errorf("delete instance %s: %w", instance.GetName(), err)
	}
	if removeCleanup != nil {
		removeCleanup(instance.GetName())
	}
	if err := e2eapic.WaitForInstanceAbsent(deadline, client, instance); err != nil {
		return err
	}
	if err := e2eapic.WaitForNoAppsForInstance(deadline, client, instance.GetName(), instance.GetVmId()); err != nil {
		return err
	}
	peerID, err := e2eapic.PeerIDForInstance(instance)
	if err != nil {
		return err
	}
	if err := e2eapic.WaitForPeerRemoved(deadline, client, peerID); err != nil {
		return err
	}
	fmt.Printf("delete assertion ok: name=%s id=%s\n", instance.GetName(), instance.GetVmId())
	return nil
}

func uploadImagePath(imagePath string, defaultFile string) (string, error) {
	imagePath = strings.TrimSpace(imagePath)
	if imagePath == "" {
		return "", fmt.Errorf("image path is empty")
	}
	absPath, err := filepath.Abs(imagePath)
	if err != nil {
		return "", err
	}

	return imageFilePath(absPath, defaultFile)
}

func imageFilePath(imagePath string, defaultFile string) (string, error) {
	info, err := os.Stat(imagePath)
	if err != nil {
		return "", fmt.Errorf("stat image path: %w", err)
	}
	if !info.IsDir() {
		return imagePath, nil
	}

	base := filepath.Base(imagePath)
	candidates := []string{
		filepath.Join(imagePath, defaultFile),
		filepath.Join(imagePath, base+"-disk.img"),
		filepath.Join(imagePath, base+"-efi.iso"),
		filepath.Join(imagePath, "disk.img"),
		filepath.Join(imagePath, "efi.iso"),
		filepath.Join(imagePath, base+"-root.raw"),
		filepath.Join(imagePath, "root.raw"),
	}
	for _, candidate := range candidates {
		if isRegularFile(candidate) {
			return candidate, nil
		}
	}

	matches, err := filepath.Glob(filepath.Join(imagePath, "*-disk.img"))
	if err != nil {
		return "", err
	}
	for _, match := range matches {
		if isRegularFile(match) {
			return match, nil
		}
	}
	matches, err = filepath.Glob(filepath.Join(imagePath, "*-root.raw"))
	if err != nil {
		return "", err
	}
	for _, match := range matches {
		if isRegularFile(match) {
			return match, nil
		}
	}

	return "", fmt.Errorf("image directory %s does not contain an uploadable image file", imagePath)
}

func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func hetznerAuth(path string) (map[string]string, error) {
	env, err := loadEnvFile(path)
	if err != nil {
		return nil, err
	}
	auth := map[string]string{
		"API_KEY": envValue(env, "API_KEY"),
	}
	if err := requireAuth(path, auth, "API_KEY"); err != nil {
		return nil, err
	}
	return auth, nil
}

func scalewayAuth(path string) (map[string]string, error) {
	env, err := loadEnvFile(path)
	if err != nil {
		return nil, err
	}
	auth := map[string]string{
		"ORGANISATION_ID": firstNonEmpty(
			envValue(env, "ORGANISATION_ID"),
			envValue(env, "ORG_ID"),
			envValue(env, "SCW_DEFAULT_ORGANIZATION_ID"),
			scalewayConfigValue("default_organization_id"),
			scalewayConfigValue("organization_id"),
		),
		"ACCESS_KEY": envValue(env, "ACCESS_KEY"),
		"SECRET_KEY": envValue(env, "SECRET_KEY"),
	}
	if projectID := firstNonEmpty(
		envValue(env, "PROJECT_ID"),
		envValue(env, "SCW_DEFAULT_PROJECT_ID"),
		envValue(env, "SCW_PROJECT_ID"),
		scalewayConfigValue("default_project_id"),
		scalewayConfigValue("project_id"),
	); projectID != "" {
		auth["PROJECT_ID"] = projectID
	}
	if err := requireAuth(path, auth, "ORGANISATION_ID", "ACCESS_KEY", "SECRET_KEY"); err != nil {
		return nil, err
	}
	return auth, nil
}

func loadEnvFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read env file %s: %w", path, err)
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(strings.TrimPrefix(key, "export "))
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if key != "" {
			values[key] = value
		}
	}
	return values, nil
}

func envValue(values map[string]string, key string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return strings.TrimSpace(values[key])
}

func scalewayConfigValue(key string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(home, ".config", "scw", "config.yaml"))
	if err != nil {
		return ""
	}
	prefix := key + ":"
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			return strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, prefix)), `"'`)
		}
	}
	return ""
}

func requireAuth(path string, values map[string]string, keys ...string) error {
	var missing []string
	for _, key := range keys {
		if strings.TrimSpace(values[key]) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("env file %s is missing required keys: %s", path, strings.Join(missing, ", "))
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
