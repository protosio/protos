//go:build darwin

package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"flag"

	"github.com/Masterminds/semver"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	hostagentipc "github.com/protosio/protos/internal/hostagent/ipc"
	"github.com/protosio/protos/internal/membership"
	protosnetwork "github.com/protosio/protos/internal/network"
	"github.com/protosio/protos/internal/p2p"
	p2pproto "github.com/protosio/protos/internal/p2p/proto"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/provisioners"
	"github.com/protosio/protos/internal/release"
	"github.com/protosio/protos/internal/user"
	"github.com/protosio/protos/provisioners/hetzner"
	"github.com/protosio/protos/provisioners/local_macos"
	"github.com/protosio/protos/provisioners/scaleway"
	"google.golang.org/grpc/status"
	swarmionapp "swarmion.dev/runtime/app"
)

type noopAppManager struct{}

func (noopAppManager) GetLogs(string) ([]byte, error) {
	return nil, fmt.Errorf("logs are not available in the e2e harness")
}

func (noopAppManager) GetStatus(string) (string, error) {
	return "", fmt.Errorf("status is not available in the e2e harness")
}

type witnessExpectation struct {
	label    string
	required []string
	allowed  []string
	count    int
	ranks    map[string]int
}

func main() {
	imagePath := flag.String("image", "provisioner-images/targets/output/mactest", "local macOS LinuxKit image directory or ISO path")
	workDir := flag.String("workdir", "", "temporary Protos workdir")
	keep := flag.Bool("keep", false, "keep temporary state after the run")
	timeout := flag.Duration("timeout", 45*time.Minute, "overall verification timeout")
	imageUploadTimeout := flag.Duration("image-upload-timeout", 30*time.Minute, "per-provider cloud image upload timeout")
	localMachine := flag.String("local-machine", "vz-2c-2g", "local macOS machine type")
	hetznerImage := flag.String("hetzner-image", "provisioner-images/targets/output/hetzner/hetzner-efi-fat16.img", "Hetzner FAT16 EFI disk image to upload")
	hetznerMachine := flag.String("hetzner-machine", "cpx22", "Hetzner machine type")
	hetznerLocation := flag.String("hetzner-location", "hel1", "Hetzner location")
	hetznerEnv := flag.String("hetzner-env", ".env-hetzner", "Hetzner credential env file")
	scalewayImage := flag.String("scaleway-image", "provisioner-images/targets/output/scaleway/scaleway-efi.iso", "Scaleway EFI image to upload")
	scalewayMachine := flag.String("scaleway-machine", "DEV1-S", "Scaleway machine type")
	scalewayLocation := flag.String("scaleway-location", "fr-par-1", "Scaleway zone")
	scalewayEnv := flag.String("scaleway-env", ".env-scaleway", "Scaleway credential env file")
	configureNetwork := flag.Bool("network", true, "configure the host network module through protos-hostagent")
	flag.Parse()

	cfg := harnessConfig{
		imagePath:          *imagePath,
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
		configureNetwork:   *configureNetwork,
	}
	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "mixed witness e2e failed: %v\n", err)
		os.Exit(1)
	}
}

type harnessConfig struct {
	imagePath          string
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
	configureNetwork   bool
}

func run(cfg harnessConfig) error {
	if err := ensureHostAgentAvailable(); err != nil {
		return err
	}

	imagePath, err := filepath.Abs(cfg.imagePath)
	if err != nil {
		return err
	}
	hetznerImagePath, err := uploadImagePath(cfg.hetznerImage, "hetzner-efi-fat16.img")
	if err != nil {
		return fmt.Errorf("resolve Hetzner image: %w", err)
	}
	scalewayImagePath, err := uploadImagePath(cfg.scalewayImage, "scaleway-efi.iso")
	if err != nil {
		return fmt.Errorf("resolve Scaleway image: %w", err)
	}
	workDir := cfg.workDir
	if workDir == "" {
		workDir, err = os.MkdirTemp("", "protos-mixed-witness-e2e-*")
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

	version, err := semver.NewVersion("0.1.0-dev.23")
	if err != nil {
		return err
	}
	config.New(workDir, version)

	localKey, err := pcrypto.GetLocalKey(workDir)
	if err != nil {
		return fmt.Errorf("get local key: %w", err)
	}
	store, err := db.Open(workDir, config.DBName, localKey)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer store.Close()
	if err := store.Init(); err != nil {
		return fmt.Errorf("init db: %w", err)
	}

	keyManager := pcrypto.CreateManager(store)
	userManager := user.CreateManager(store, keyManager)
	admin, err := userManager.CreateUser("e2e", "Mixed Witness E2E", true)
	if err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}
	if err := userManager.AddDevice(admin.Username, "local-laptop-e2e", localKey); err != nil {
		return fmt.Errorf("add current device: %w", err)
	}

	var networkManager *protosnetwork.Manager
	if cfg.configureNetwork {
		networkManager, err = protosnetwork.NewManager()
		if err != nil {
			return fmt.Errorf("create network manager: %w", err)
		}
		if err := networkManager.Init(localKey, config.Get().InternalDomain); err != nil {
			_ = networkManager.Close()
			return fmt.Errorf("initialize network manager through protos-hostagent: %w", err)
		}
		defer func() {
			_ = networkManager.ConfigurePeers(nil, nil, nil)
			_ = networkManager.Close()
		}()
	}

	p2pManager, err := p2p.NewManager(localKey, noopAppManager{}, store, config.Get().P2PPort)
	if err != nil {
		return fmt.Errorf("create p2p manager: %w", err)
	}
	p2pStop, err := p2pManager.StartServer()
	if err != nil {
		return fmt.Errorf("start p2p server: %w", err)
	}
	defer func() {
		_ = p2pStop()
	}()

	cloudManager, err := provisioners.CreateManager(
		store,
		userManager,
		keyManager,
		p2pManager,
		hetzner.NewFactory(),
		scaleway.NewFactory(),
		localmacos.NewFactory(),
	)
	if err != nil {
		return fmt.Errorf("create provisioner manager: %w", err)
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	localProviderName := "local-e2e-" + suffix
	hetznerProviderName := "hetzner-e2e-" + suffix
	scalewayProviderName := "scaleway-e2e-" + suffix
	imageName := "mixed-witness-e2e-" + suffix
	vmDir := filepath.Join(workDir, "local-macos-vms")

	var cleanupInstances []string
	var cleanupImages []imageRef
	var cleanupProviders []string
	defer func() {
		for i := len(cleanupInstances) - 1; i >= 0; i-- {
			_ = cloudManager.DeleteInstance(cleanupInstances[i])
		}
		for _, image := range cleanupImages {
			removeImage(cloudManager, image)
		}
		for _, providerName := range cleanupProviders {
			_ = cloudManager.DeleteProvisioner(providerName)
		}
	}()

	if err := cloudManager.AddProvisioner(localProviderName, localmacos.Type.String(), map[string]string{"VM_DIR": vmDir}); err != nil {
		return fmt.Errorf("add local macOS provisioner: %w", err)
	}
	cleanupProviders = append(cleanupProviders, localProviderName)
	if err := cloudManager.UploadLocalImage(imagePath, imageName, localProviderName, "local", cfg.imageUploadTimeout); err != nil {
		return fmt.Errorf("upload local macOS image: %w", err)
	}
	cleanupImages = append(cleanupImages, imageRef{provider: localProviderName, name: imageName, location: "local"})

	hetznerCredentials, err := hetznerAuth(cfg.hetznerEnv)
	if err != nil {
		return err
	}
	if err := cloudManager.AddProvisioner(hetznerProviderName, hetzner.Type.String(), hetznerCredentials); err != nil {
		return fmt.Errorf("add Hetzner provisioner: %w", err)
	}
	cleanupProviders = append(cleanupProviders, hetznerProviderName)
	if err := cloudManager.UploadLocalImage(hetznerImagePath, imageName, hetznerProviderName, cfg.hetznerLocation, cfg.imageUploadTimeout); err != nil {
		return fmt.Errorf("upload Hetzner image: %w", err)
	}
	cleanupImages = append(cleanupImages, imageRef{provider: hetznerProviderName, name: imageName, location: cfg.hetznerLocation})

	scalewayCredentials, err := scalewayAuth(cfg.scalewayEnv)
	if err != nil {
		return err
	}
	if err := cloudManager.AddProvisioner(scalewayProviderName, scaleway.Type.String(), scalewayCredentials); err != nil {
		return fmt.Errorf("add Scaleway provisioner: %w", err)
	}
	cleanupProviders = append(cleanupProviders, scalewayProviderName)
	if err := cloudManager.UploadLocalImage(scalewayImagePath, imageName, scalewayProviderName, cfg.scalewayLocation, cfg.imageUploadTimeout); err != nil {
		return fmt.Errorf("upload Scaleway image: %w", err)
	}
	cleanupImages = append(cleanupImages, imageRef{provider: scalewayProviderName, name: imageName, location: cfg.scalewayLocation})

	rel := release.Release{
		Version:     imageName,
		CloudImages: map[string]release.CloudImage{},
	}
	deadline := time.Now().Add(cfg.timeout)
	localClientRank := db.DefaultWitnessRankForDeviceType(db.WitnessDeviceTypeLocalUserClient)
	localVMRank := db.DefaultWitnessRankForDeviceType(db.WitnessDeviceTypeLocalVM)
	cloudVMRank := db.DefaultWitnessRankForDeviceType(db.WitnessDeviceTypeCloudVM)

	if err := waitForWitnessState(deadline, store, cloudManager, userManager, p2pManager, networkManager, witnessExpectation{
		label:    "initial local laptop",
		required: []string{localKey.GetID()},
		allowed:  []string{localKey.GetID()},
		count:    1,
		ranks: map[string]int{
			localKey.GetID(): localClientRank,
		},
	}); err != nil {
		return err
	}

	var deployed []provisioners.InstanceInfo
	deploy := func(name string, provider string, location string, machine string) (provisioners.InstanceInfo, error) {
		instance, err := cloudManager.DeployInstance(name, provider, location, rel, machine)
		if err != nil {
			return provisioners.InstanceInfo{}, err
		}
		cleanupInstances = append(cleanupInstances, name)
		deployed = append(deployed, instance)
		fmt.Printf("deployed instance: name=%s id=%s kind=%s provider=%s ip=%s arch=%s status=%s\n", instance.Name, instance.ID, instance.Kind, provider, instance.PublicIP, instance.Architecture, instance.Status)
		return instance, nil
	}

	localVM1, err := deploy("local-vm-1-"+suffix, localProviderName, "local", cfg.localMachine)
	if err != nil {
		return fmt.Errorf("deploy first local macOS VM: %w", err)
	}
	if err := waitForWitnessState(deadline, store, cloudManager, userManager, p2pManager, networkManager, witnessExpectation{
		label:    "first local VM stays fallback below laptop",
		required: []string{localKey.GetID()},
		allowed:  []string{localKey.GetID(), localVM1.ID},
		count:    1,
		ranks: map[string]int{
			localKey.GetID(): localClientRank,
			localVM1.ID:      localVMRank,
		},
	}); err != nil {
		return err
	}

	localVM2, err := deploy("local-vm-2-"+suffix, localProviderName, "local", cfg.localMachine)
	if err != nil {
		return fmt.Errorf("deploy second local macOS VM: %w", err)
	}
	if err := waitForWitnessState(deadline, store, cloudManager, userManager, p2pManager, networkManager, witnessExpectation{
		label:    "local laptop remains above local VMs",
		required: []string{localKey.GetID()},
		allowed:  []string{localKey.GetID(), localVM1.ID, localVM2.ID},
		count:    1,
		ranks: map[string]int{
			localKey.GetID(): localClientRank,
			localVM1.ID:      localVMRank,
			localVM2.ID:      localVMRank,
		},
	}); err != nil {
		return err
	}

	hetznerVM, err := deploy("hetzner-vm-"+suffix, hetznerProviderName, cfg.hetznerLocation, cfg.hetznerMachine)
	if err != nil {
		return fmt.Errorf("deploy Hetzner VM: %w", err)
	}
	if err := waitForWitnessState(deadline, store, cloudManager, userManager, p2pManager, networkManager, witnessExpectation{
		label:    "single Hetzner cloud VM waits as eligible standby behind laptop",
		required: []string{localKey.GetID()},
		allowed:  []string{hetznerVM.ID, localKey.GetID(), localVM1.ID, localVM2.ID},
		count:    1,
		ranks: map[string]int{
			localKey.GetID(): localClientRank,
			localVM1.ID:      localVMRank,
			localVM2.ID:      localVMRank,
			hetznerVM.ID:     cloudVMRank,
		},
	}); err != nil {
		return err
	}

	scalewayVM, err := deploy("scaleway-vm-"+suffix, scalewayProviderName, cfg.scalewayLocation, cfg.scalewayMachine)
	if err != nil {
		return fmt.Errorf("deploy Scaleway VM: %w", err)
	}
	if err := waitForWitnessState(deadline, store, cloudManager, userManager, p2pManager, networkManager, witnessExpectation{
		label:    "two cloud VMs become active witnesses",
		required: []string{hetznerVM.ID, scalewayVM.ID},
		allowed:  []string{hetznerVM.ID, scalewayVM.ID},
		count:    2,
		ranks: map[string]int{
			localKey.GetID(): localClientRank,
			localVM1.ID:      localVMRank,
			localVM2.ID:      localVMRank,
			hetznerVM.ID:     cloudVMRank,
			scalewayVM.ID:    cloudVMRank,
		},
	}); err != nil {
		return err
	}

	clients, err := reconnectPeersUntil(deadline, store, cloudManager, userManager, p2pManager, networkManager, instanceNames(deployed))
	if err != nil {
		return err
	}
	if networkManager != nil {
		if err := waitForHostWireGuardPings(deadline, deployed); err != nil {
			return err
		}
	}
	if err := waitForRemoteFullMesh(deadline, clients, deployed); err != nil {
		return err
	}
	if err := waitForAllRemoteHeads(deadline, store, clients, deployed); err != nil {
		return err
	}

	return nil
}

func ensureHostAgentAvailable() error {
	socket := hostagentipc.SocketPath()
	if _, err := os.Stat(socket); err != nil {
		return fmt.Errorf("host agent socket %s is not available; start protos-hostagent locally with sudo before running this harness: %w", socket, err)
	}
	conn, err := net.DialTimeout("unix", socket, 2*time.Second)
	if err != nil {
		return fmt.Errorf("host agent socket %s is not reachable; start or restart protos-hostagent locally with sudo before running this harness: %w", socket, err)
	}
	return conn.Close()
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

type imageRef struct {
	provider string
	name     string
	location string
}

func removeImage(cloudManager *provisioners.Manager, image imageRef) {
	defer func() {
		if recovered := recover(); recovered != nil {
			fmt.Printf("cleanup image panic ignored: provider=%s name=%s location=%s panic=%v\n", image.provider, image.name, image.location, recovered)
		}
	}()
	provider, err := cloudManager.GetProvider(image.provider)
	if err != nil {
		fmt.Printf("cleanup image provider lookup failed: provider=%s name=%s err=%v\n", image.provider, image.name, err)
		return
	}
	if err := provider.Init(); err != nil {
		fmt.Printf("cleanup image provider init failed: provider=%s name=%s err=%v\n", image.provider, image.name, err)
		return
	}
	imageProvider, ok := provider.(provisioners.ImageProvisioner)
	if !ok {
		fmt.Printf("cleanup image provider has no image API: provider=%s name=%s\n", image.provider, image.name)
		return
	}
	if err := imageProvider.RemoveImage(image.name, image.location); err != nil {
		fmt.Printf("cleanup image failed: provider=%s name=%s location=%s err=%v\n", image.provider, image.name, image.location, err)
	}
}

func waitForWitnessState(
	deadline time.Time,
	store *db.DB,
	cloudManager *provisioners.Manager,
	userManager *user.Manager,
	p2pManager *p2p.P2P,
	networkManager *protosnetwork.Manager,
	expect witnessExpectation,
) error {
	var lastErr error
	var lastReport time.Time
	reportProgress := func(status *swarmionapp.Status, err error) {
		if time.Since(lastReport) < 30*time.Second {
			return
		}
		lastReport = time.Now()
		if status == nil {
			fmt.Printf("waiting for witness transition: label=%s err=%v\n", expect.label, err)
			return
		}
		fmt.Printf(
			"waiting for witness transition: label=%s err=%v active=%v eligible=%v ranks=%v connected=%v providers=%v finalized=%s failures=%d last_failure=%s\n",
			expect.label,
			err,
			status.ActiveWitnessIDs,
			status.EligibleWitnessIDs,
			status.EligibleWitnessRanks,
			status.ConnectedPeers,
			status.StateProviders,
			status.FinalizedRootHash.String(),
			status.ProtocolFailureCount,
			status.LastProtocolFailureDetail,
		)
	}
	for time.Now().Before(deadline) {
		instances, devices, err := reconcileTopology(store, cloudManager, userManager, p2pManager, networkManager)
		if err != nil {
			lastErr = err
			reportProgress(nil, lastErr)
			time.Sleep(3 * time.Second)
			continue
		}
		if err := assertPersistedWitnessRanks(instances, devices, expect.ranks); err != nil {
			lastErr = err
			reportProgress(nil, lastErr)
			time.Sleep(3 * time.Second)
			continue
		}

		status, ok := store.SwarmionStatus()
		if !ok {
			lastErr = fmt.Errorf("swarmion status is unavailable")
			reportProgress(nil, lastErr)
			time.Sleep(3 * time.Second)
			continue
		}
		if status.Fatal != nil {
			return fmt.Errorf("swarmion fatal state while waiting for %s: %s", expect.label, status.Fatal.State)
		}
		if err := assertNoBlockingCompatibility(store); err != nil {
			return err
		}
		if err := assertProtocolWitnessRanks(status.EligibleWitnessRanks, expect.ranks); err != nil {
			lastErr = err
			reportProgress(&status, lastErr)
			time.Sleep(3 * time.Second)
			continue
		}
		if observed, epochID, ok := witnessStatusMatches(status, expect); ok {
			fmt.Printf("witness assertion ok: %s witnesses=%v eligible=%v root=%s epoch=%s\n", expect.label, observed, status.EligibleWitnessIDs, status.FinalizedRootHash.String(), epochID)
			return nil
		}
		lastErr = fmt.Errorf("active witnesses %v did not match %s", status.ActiveWitnessIDs, describeExpectation(expect))
		reportProgress(&status, lastErr)
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("timed out waiting for witness transition %q: %w", expect.label, lastErr)
}

func assertPersistedWitnessRanks(instances []provisioners.InstanceInfo, devices []user.UserDevice, expected map[string]int) error {
	if len(expected) == 0 {
		return nil
	}
	observed := make(map[string]int, len(instances)+len(devices))
	for _, instance := range instances {
		observed[instance.ID] = instance.WitnessRank
	}
	for _, device := range devices {
		observed[device.ID] = device.WitnessRank
	}
	for peerID, want := range expected {
		got, found := observed[peerID]
		if !found {
			return fmt.Errorf("missing persisted witness rank for peer %s", peerID)
		}
		if got != want {
			return fmt.Errorf("persisted witness rank for peer %s = %d, want %d", peerID, got, want)
		}
	}
	return nil
}

func assertProtocolWitnessRanks(observed map[string]int, expected map[string]int) error {
	if len(expected) == 0 {
		return nil
	}
	for peerID, want := range expected {
		got, found := observed[peerID]
		if !found {
			return fmt.Errorf("missing finalized protocol witness rank for peer %s", peerID)
		}
		if got != want {
			return fmt.Errorf("finalized protocol witness rank for peer %s = %d, want %d", peerID, got, want)
		}
	}
	return nil
}

func reconcileTopology(
	store *db.DB,
	cloudManager *provisioners.Manager,
	userManager *user.Manager,
	p2pManager *p2p.P2P,
	networkManager *protosnetwork.Manager,
) ([]provisioners.InstanceInfo, []user.UserDevice, error) {
	instances, devices, err := declarativePeers(store, cloudManager, userManager)
	if err != nil {
		return nil, nil, err
	}
	if networkManager != nil {
		if err := networkManager.ConfigurePeers(instances, devices, nil); err != nil {
			return nil, nil, fmt.Errorf("configure network peers: %w", err)
		}
	}
	if err := p2pManager.ConfigurePeers(membership.Machines(instances, devices)); err != nil {
		return nil, nil, err
	}
	if err := store.ReconcileWitnesses(context.Background(), membership.WitnessCandidates(instances, devices)); err != nil {
		return nil, nil, err
	}
	return instances, devices, nil
}

func assertNoBlockingCompatibility(store *db.DB) error {
	compatibility, err := store.SwarmionCompatibility(context.Background())
	if err != nil {
		return err
	}
	for _, item := range compatibility {
		if item.Blocking {
			return fmt.Errorf("swarmion compatibility blocked by peer %s: %s", item.PeerID, item.Reason)
		}
	}
	return nil
}

func witnessSetMatches(active []string, expect witnessExpectation) bool {
	if len(active) != expect.count {
		return false
	}
	activeSet := stringSet(active)
	allowed := stringSet(expect.allowed)
	for _, peerID := range expect.required {
		if !activeSet[peerID] {
			return false
		}
	}
	for _, peerID := range active {
		if !allowed[peerID] {
			return false
		}
	}
	return true
}

func witnessStatusMatches(status swarmionapp.Status, expect witnessExpectation) ([]string, string, bool) {
	if witnessSetMatches(status.ActiveWitnessIDs, expect) {
		return append([]string(nil), status.ActiveWitnessIDs...), status.ActiveEpochID, true
	}
	if len(expect.required) != expect.count {
		return nil, "", false
	}
	witnesses, epochID, ok := db.WitnessFormationInStatus(status, expect.required)
	if !ok || !witnessSetMatches(witnesses, expect) {
		return nil, "", false
	}
	return witnesses, epochID, true
}

func describeExpectation(expect witnessExpectation) string {
	return fmt.Sprintf("expectation %q count=%d required=%v allowed=%v", expect.label, expect.count, expect.required, expect.allowed)
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

func reconnectPeersUntil(
	deadline time.Time,
	store *db.DB,
	cloudManager *provisioners.Manager,
	userManager *user.Manager,
	p2pManager *p2p.P2P,
	networkManager *protosnetwork.Manager,
	wantNames []string,
) (map[string]*p2p.Client, error) {
	var lastErr error
	for time.Now().Before(deadline) {
		if _, _, err := reconcileTopology(store, cloudManager, userManager, p2pManager, networkManager); err != nil {
			lastErr = err
			time.Sleep(3 * time.Second)
			continue
		}
		clients := map[string]*p2p.Client{}
		allFound := true
		for _, name := range wantNames {
			client, err := p2pManager.GetClient(name)
			if err != nil || client == nil {
				lastErr = err
				allFound = false
				break
			}
			clients[name] = client
		}
		if allFound {
			return clients, nil
		}
		time.Sleep(3 * time.Second)
	}
	return nil, fmt.Errorf("remote peers did not become reachable: %w", lastErr)
}

func declarativePeers(store *db.DB, cloudManager *provisioners.Manager, userManager *user.Manager) ([]provisioners.InstanceInfo, []user.UserDevice, error) {
	peerIDs, err := db.GetPeerIDs(store)
	if err != nil {
		return nil, nil, err
	}
	instances, err := cloudManager.GetInstances(true)
	if err != nil {
		return nil, nil, err
	}
	instances = membership.FilterInstances(instances, peerIDs)
	devices, err := userManager.GetAllDevices(false)
	if err != nil {
		return nil, nil, err
	}
	devices = membership.FilterDevices(devices, peerIDs)
	return instances, devices, nil
}

func waitForRemoteFullMesh(deadline time.Time, clients map[string]*p2p.Client, instances []provisioners.InstanceInfo) error {
	var lastErr error
	for time.Now().Before(deadline) {
		ready := true
		for _, source := range instances {
			client := clients[source.Name]
			if client == nil {
				ready = false
				lastErr = fmt.Errorf("missing p2p client for %s", source.Name)
				break
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			peers, err := client.GetPeers(ctx, &p2pproto.GetPeersRequest{})
			cancel()
			if err != nil {
				ready = false
				lastErr = fmt.Errorf("get remote peers for %s: %w", source.Name, err)
				break
			}
			fmt.Printf("remote p2p peers for %s: %v\n", source.Name, peers.Peers)
			for _, target := range instances {
				if target.Name == source.Name {
					continue
				}
				if peers.Peers[target.Name] != string(p2p.PeerStatusConnected) {
					ready = false
					lastErr = fmt.Errorf("%s sees %s as %q", source.Name, target.Name, peers.Peers[target.Name])
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
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("VM full mesh did not become connected: %w", lastErr)
}

func waitForHostWireGuardPings(deadline time.Time, instances []provisioners.InstanceInfo) error {
	var lastErr error
	for time.Now().Before(deadline) {
		allReachable := true
		for _, instance := range instances {
			ip := wireGuardIPv6(instance)
			if ip == "" {
				return fmt.Errorf("instance %s has no derivable WireGuard IPv6 address", instance.Name)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			output, err := exec.CommandContext(ctx, "ping6", "-c", "3", "-n", ip).CombinedOutput()
			cancel()
			if err != nil {
				allReachable = false
				lastErr = fmt.Errorf("ping %s (%s): %w: %s", instance.Name, ip, err, strings.TrimSpace(string(output)))
				break
			}
			fmt.Printf("host WireGuard ping ok: %s %s\n", instance.Name, ip)
		}
		if allReachable {
			return nil
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("host could not reach all VM WireGuard IPv6 addresses: %w", lastErr)
}

func waitForAllRemoteHeads(deadline time.Time, store *db.DB, clients map[string]*p2p.Client, instances []provisioners.InstanceInfo) error {
	localHead, err := store.GetLastCommit("main")
	if err != nil {
		return fmt.Errorf("get local DB head: %w", err)
	}
	for _, instance := range instances {
		client := clients[instance.Name]
		if client == nil {
			return fmt.Errorf("missing p2p client for %s", instance.Name)
		}
		if err := waitForRemoteHead(deadline, client, localHead.Hash); err != nil {
			return fmt.Errorf("%s did not sync DB head: %w", instance.Name, err)
		}
		fmt.Printf("remote DB head matched local head for %s: %s\n", instance.Name, localHead.Hash)
	}
	return nil
}

func instanceNames(instances []provisioners.InstanceInfo) []string {
	names := make([]string, 0, len(instances))
	for _, instance := range instances {
		names = append(names, instance.Name)
	}
	return names
}

func wireGuardIPv6(instance provisioners.InstanceInfo) string {
	key, err := pcrypto.CreatePublicKeyFromBase64(instance.PublicKey)
	if err != nil {
		return ""
	}
	return key.IPv6Address().String()
}

func waitForRemoteHead(deadline time.Time, client *p2p.Client, expected string) error {
	var lastErr error
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		head, err := client.GetHead(ctx, &p2pproto.GetHeadRequest{})
		cancel()
		if err == nil && strings.TrimSpace(head.Commit) == expected {
			return nil
		}
		if err != nil {
			if st, ok := status.FromError(err); ok {
				lastErr = fmt.Errorf("%s", st.Message())
			} else {
				lastErr = err
			}
		} else {
			lastErr = fmt.Errorf("remote head %q did not match local head %q", head.Commit, expected)
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("remote DB head did not match local DB head: %w", lastErr)
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
