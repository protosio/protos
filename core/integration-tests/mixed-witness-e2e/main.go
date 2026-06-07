//go:build darwin

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"flag"

	"github.com/Masterminds/semver/v3"
	protosapp "github.com/protosio/protos/internal/app"
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
	label         string
	required      []string
	allowed       []string
	count         int
	requireActive bool
	ranks         map[string]int
}

type appConnectivityPair struct {
	hetznerApp  *protosapp.App
	scalewayApp *protosapp.App
}

const (
	localDeviceName      = "local-laptop-e2e"
	defaultProbeAppImage = "docker.io/protosio/protos-e2e-probe:latest"
	probeAppPort         = 8080
)

type probeAppResponse struct {
	OK             bool   `json:"ok"`
	ID             string `json:"id"`
	Target         string `json:"target"`
	StatusCode     int    `json:"status_code"`
	BytesRead      int    `json:"bytes_read"`
	DurationMillis int64  `json:"duration_ms"`
	Error          string `json:"error"`
}

func main() {
	imagePath := flag.String("image", "../cloud-provisioning/targets/output/mactest", "local macOS LinuxKit image directory or ISO path")
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
	seedImageArchive := flag.String("seed-image-archive", "", "optional local image tar or tar.gz archive to copy to the Hetzner seed VM and load before app start")
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
		appImage:           *appImage,
		seedImageArchive:   *seedImageArchive,
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
	appImage           string
	seedImageArchive   string
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
	if err := userManager.AddDevice(admin.Username, localDeviceName, localKey); err != nil {
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
			_ = networkManager.ConfigurePeers(nil, nil, nil, nil)
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
	taskStop := cloudManager.StartTaskRunner(context.Background(), 500*time.Millisecond)
	defer func() {
		_ = taskStop()
	}()

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
	removeCleanupInstance := func(name string) {
		for i, cleanupName := range cleanupInstances {
			if cleanupName != name {
				continue
			}
			cleanupInstances = append(cleanupInstances[:i], cleanupInstances[i+1:]...)
			return
		}
	}

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
		if _, err := cloudManager.DeployInstance(name, provider, location, rel, machine); err != nil {
			return provisioners.InstanceInfo{}, err
		}
		cleanupInstances = append(cleanupInstances, name)
		instance, err := waitForInstanceReady(deadline, cloudManager, name)
		if err != nil {
			return provisioners.InstanceInfo{}, err
		}
		deployed = append(deployed, instance)
		fmt.Printf("deployed instance: name=%s id=%s kind=%s provider=%s ip=%s arch=%s status=%s\n", instance.Name, instance.ID, instance.Kind, provider, instance.PublicIP, instance.Architecture, instance.Status)
		return instance, nil
	}

	localVM1, err := deploy("local-vm-1-"+suffix, localProviderName, "local", cfg.localMachine)
	if err != nil {
		return fmt.Errorf("deploy first local macOS VM: %w", err)
	}
	localVM1PeerID, err := instancePeerID(localVM1)
	if err != nil {
		return err
	}
	if err := waitForWitnessState(deadline, store, cloudManager, userManager, p2pManager, networkManager, witnessExpectation{
		label:    "first local VM stays fallback below laptop",
		required: []string{localKey.GetID()},
		allowed:  []string{localKey.GetID(), localVM1PeerID},
		count:    1,
		ranks: map[string]int{
			localKey.GetID(): localClientRank,
			localVM1PeerID:   localVMRank,
		},
	}); err != nil {
		return err
	}

	localVM2, err := deploy("local-vm-2-"+suffix, localProviderName, "local", cfg.localMachine)
	if err != nil {
		return fmt.Errorf("deploy second local macOS VM: %w", err)
	}
	localVM2PeerID, err := instancePeerID(localVM2)
	if err != nil {
		return err
	}
	if err := waitForWitnessState(deadline, store, cloudManager, userManager, p2pManager, networkManager, witnessExpectation{
		label:    "local laptop remains above local VMs",
		required: []string{localKey.GetID()},
		allowed:  []string{localKey.GetID(), localVM1PeerID, localVM2PeerID},
		count:    1,
		ranks: map[string]int{
			localKey.GetID(): localClientRank,
			localVM1PeerID:   localVMRank,
			localVM2PeerID:   localVMRank,
		},
	}); err != nil {
		return err
	}

	hetznerVM, err := deploy("hetzner-vm-"+suffix, hetznerProviderName, cfg.hetznerLocation, cfg.hetznerMachine)
	if err != nil {
		return fmt.Errorf("deploy Hetzner VM: %w", err)
	}
	hetznerPeerID, err := instancePeerID(hetznerVM)
	if err != nil {
		return err
	}
	if err := waitForWitnessState(deadline, store, cloudManager, userManager, p2pManager, networkManager, witnessExpectation{
		label:    "single Hetzner cloud VM waits as eligible standby behind laptop",
		required: []string{localKey.GetID()},
		allowed:  []string{hetznerPeerID, localKey.GetID(), localVM1PeerID, localVM2PeerID},
		count:    1,
		ranks: map[string]int{
			localKey.GetID(): localClientRank,
			localVM1PeerID:   localVMRank,
			localVM2PeerID:   localVMRank,
			hetznerPeerID:    cloudVMRank,
		},
	}); err != nil {
		return err
	}

	scalewayVM, err := deploy("scaleway-vm-"+suffix, scalewayProviderName, cfg.scalewayLocation, cfg.scalewayMachine)
	if err != nil {
		return fmt.Errorf("deploy Scaleway VM: %w", err)
	}
	scalewayPeerID, err := instancePeerID(scalewayVM)
	if err != nil {
		return err
	}
	if err := waitForWitnessState(deadline, store, cloudManager, userManager, p2pManager, networkManager, witnessExpectation{
		label:         "two cloud VMs become active witnesses and laptop adopts child epoch",
		required:      []string{hetznerPeerID, scalewayPeerID},
		allowed:       []string{hetznerPeerID, scalewayPeerID},
		count:         2,
		requireActive: true,
		ranks: map[string]int{
			localKey.GetID(): localClientRank,
			localVM1PeerID:   localVMRank,
			localVM2PeerID:   localVMRank,
			hetznerPeerID:    cloudVMRank,
			scalewayPeerID:   cloudVMRank,
		},
	}); err != nil {
		return err
	}

	clients, err := reconnectPeersUntil(deadline, store, cloudManager, userManager, p2pManager, networkManager, instanceNames(deployed))
	if err != nil {
		return err
	}
	if err := waitForRemotePeerConnection(deadline, clients, deployed, localKey.GetID(), localDeviceName); err != nil {
		return err
	}
	if err := waitForRemoteSwarmionConnection(deadline, clients, deployed, localKey.GetID()); err != nil {
		return err
	}
	if err := waitForAllRemoteHeads(deadline, store, clients, deployed); err != nil {
		return fmt.Errorf("post-witness handoff DB sync failed: %w", err)
	}

	appPair, err := createCloudAppConnectivityPair(cfg, deadline, store, cloudManager, userManager, p2pManager, networkManager, deployed, hetznerVM, scalewayVM, suffix)
	if err != nil {
		return err
	}
	if err := waitForLocalHeadFinalized(deadline, store, "cloud app deployment writes"); err != nil {
		return err
	}

	clients, err = reconnectPeersUntil(deadline, store, cloudManager, userManager, p2pManager, networkManager, instanceNames(deployed))
	if err != nil {
		return err
	}
	if err := waitForRemotePeerConnection(deadline, clients, deployed, localKey.GetID(), localDeviceName); err != nil {
		return err
	}
	if err := waitForRemoteSwarmionConnection(deadline, clients, deployed, localKey.GetID()); err != nil {
		return err
	}
	if err := waitForAllRemoteHeads(deadline, store, clients, deployed); err != nil {
		return err
	}
	if err := verifyCloudAppConnectivity(deadline, clients, hetznerVM, scalewayVM, appPair); err != nil {
		return err
	}

	if err := deleteInstanceAndVerify(deadline, store, cloudManager, userManager, p2pManager, networkManager, localVM1, removeCleanupInstance); err != nil {
		return err
	}
	if err := waitForWitnessState(deadline, store, cloudManager, userManager, p2pManager, networkManager, witnessExpectation{
		label:         "cloud witness pair remains after first local VM deletion",
		required:      []string{hetznerPeerID, scalewayPeerID},
		allowed:       []string{hetznerPeerID, scalewayPeerID},
		count:         2,
		requireActive: true,
		ranks: map[string]int{
			localKey.GetID(): localClientRank,
			localVM2PeerID:   localVMRank,
			hetznerPeerID:    cloudVMRank,
			scalewayPeerID:   cloudVMRank,
		},
	}); err != nil {
		return err
	}

	if err := deleteInstanceAndVerify(deadline, store, cloudManager, userManager, p2pManager, networkManager, localVM2, removeCleanupInstance); err != nil {
		return err
	}
	if err := waitForWitnessState(deadline, store, cloudManager, userManager, p2pManager, networkManager, witnessExpectation{
		label:         "cloud witness pair remains after second local VM deletion",
		required:      []string{hetznerPeerID, scalewayPeerID},
		allowed:       []string{hetznerPeerID, scalewayPeerID},
		count:         2,
		requireActive: true,
		ranks: map[string]int{
			localKey.GetID(): localClientRank,
			hetznerPeerID:    cloudVMRank,
			scalewayPeerID:   cloudVMRank,
		},
	}); err != nil {
		return err
	}

	if err := deleteInstanceAndVerify(deadline, store, cloudManager, userManager, p2pManager, networkManager, hetznerVM, removeCleanupInstance); err != nil {
		return err
	}
	if err := waitForWitnessState(deadline, store, cloudManager, userManager, p2pManager, networkManager, witnessExpectation{
		label:    "single remaining cloud VM falls back behind laptop after Hetzner deletion",
		required: []string{localKey.GetID()},
		allowed:  []string{localKey.GetID(), scalewayPeerID},
		count:    1,
		ranks: map[string]int{
			localKey.GetID(): localClientRank,
			scalewayPeerID:   cloudVMRank,
		},
	}); err != nil {
		return err
	}

	if err := deleteInstanceAndVerify(deadline, store, cloudManager, userManager, p2pManager, networkManager, scalewayVM, removeCleanupInstance); err != nil {
		return err
	}
	if err := waitForWitnessState(deadline, store, cloudManager, userManager, p2pManager, networkManager, witnessExpectation{
		label:    "local laptop remains after all VM deletion",
		required: []string{localKey.GetID()},
		allowed:  []string{localKey.GetID()},
		count:    1,
		ranks: map[string]int{
			localKey.GetID(): localClientRank,
		},
	}); err != nil {
		return err
	}
	if err := waitForNoInstances(deadline, cloudManager); err != nil {
		return err
	}
	if err := waitForNoApps(deadline, store); err != nil {
		return err
	}

	return nil
}

func waitForLocalHeadFinalized(deadline time.Time, store *db.DB, label string) error {
	target, err := store.GetLastCommit("main")
	if err != nil {
		return fmt.Errorf("get local %s head: %w", label, err)
	}
	var lastErr error
	var lastReport time.Time
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := store.CatchUpFinalized(ctx, label)
		cancel()
		if err != nil {
			lastErr = err
		} else {
			head, headErr := store.GetLastCommit("main")
			if headErr != nil {
				lastErr = headErr
			} else if head.Hash == target.Hash {
				fmt.Printf("local head finalized for %s: %s\n", label, target.Hash)
				return nil
			} else {
				lastErr = fmt.Errorf("local head %q did not match authored head %q", head.Hash, target.Hash)
			}
		}
		if time.Since(lastReport) >= 30*time.Second {
			lastReport = time.Now()
			if status, ok := store.SwarmionStatus(); ok {
				fmt.Printf("waiting for local finalized head: label=%s err=%v active=%v finalized=%s failures=%d last_failure=%s\n", label, lastErr, status.ActiveWitnessIDs, status.FinalizedRootHash.String(), status.ProtocolFailureCount, status.LastProtocolFailureDetail)
			} else {
				fmt.Printf("waiting for local finalized head: label=%s err=%v\n", label, lastErr)
			}
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("local authored head did not finalize for %s: %w", label, lastErr)
}

func ensureHostAgentAvailable() error {
	socket := hostagentipc.SocketPath()
	if _, err := os.Stat(socket); err != nil {
		return fmt.Errorf("host agent socket %s is not available; start the host agent through the Protos StartHostAgent API before running this harness: %w", socket, err)
	}
	conn, err := net.DialTimeout("unix", socket, 2*time.Second)
	if err != nil {
		return fmt.Errorf("host agent socket %s is not reachable; start or restart the host agent through the Protos StartHostAgent API before running this harness: %w", socket, err)
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
		if err := assertProtocolWitnessRanks(status.EligibleWitnessRanks, protocolWitnessRanks(expect)); err != nil {
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
		peerID, err := instancePeerID(instance)
		if err != nil {
			return err
		}
		observed[peerID] = instance.WitnessRank
	}
	for _, device := range devices {
		peerID, err := db.PeerIDFromPublicKeyString(device.PublicKey)
		if err != nil {
			return err
		}
		observed[peerID] = device.WitnessRank
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

func protocolWitnessRanks(expect witnessExpectation) map[string]int {
	if len(expect.ranks) == 0 || len(expect.required) == 0 {
		return nil
	}
	expected := make(map[string]int, len(expect.required))
	for _, peerID := range expect.required {
		rank, found := expect.ranks[peerID]
		if !found {
			continue
		}
		expected[peerID] = rank
	}
	return expected
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
		routes, err := appRoutes(store, instances)
		if err != nil {
			return nil, nil, fmt.Errorf("load app routes: %w", err)
		}
		if err := networkManager.ConfigurePeers(instances, devices, routes, nil); err != nil {
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

func appRoutes(store *db.DB, instances []provisioners.InstanceInfo) ([]protosnetwork.AppRoute, error) {
	apps, err := protosapp.CreateManager("", nil, store).GetAll()
	if err != nil {
		return nil, err
	}
	activeInstanceIDs := make(map[string]struct{}, len(instances))
	for _, instance := range instances {
		activeInstanceIDs[instance.ID] = struct{}{}
	}
	routes := make([]protosnetwork.AppRoute, 0, len(apps))
	for _, app := range apps {
		if app.IP == nil || strings.TrimSpace(app.InstanceID) == "" {
			continue
		}
		if _, found := activeInstanceIDs[app.InstanceID]; !found {
			continue
		}
		routes = append(routes, protosnetwork.AppRoute{
			InstanceID: app.InstanceID,
			IP:         app.IP,
		})
	}
	return routes, nil
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
	if expect.requireActive {
		return nil, "", false
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

func instancePeerID(instance provisioners.InstanceInfo) (string, error) {
	peerID, err := instance.GetPeerID()
	if err != nil {
		return "", fmt.Errorf("derive peer id for instance %s: %w", instance.Name, err)
	}
	return peerID, nil
}

func waitForInstanceReady(deadline time.Time, cloudManager *provisioners.Manager, name string) (provisioners.InstanceInfo, error) {
	var lastStatus string
	for time.Now().Before(deadline) {
		instance, err := cloudManager.GetInstance(name)
		if err == nil {
			lastStatus = instance.Status
			if strings.TrimSpace(instance.PublicKey) != "" {
				return instance, nil
			}
			if strings.Contains(strings.ToLower(instance.Status), "failed") || strings.Contains(strings.ToLower(instance.Status), "cancelled") {
				return provisioners.InstanceInfo{}, fmt.Errorf("deployment for %s ended with status %q", name, instance.Status)
			}
		} else {
			lastStatus = err.Error()
		}
		time.Sleep(2 * time.Second)
	}
	return provisioners.InstanceInfo{}, fmt.Errorf("instance %s did not become ready before deadline; last status: %s", name, lastStatus)
}

func deleteInstanceAndVerify(
	deadline time.Time,
	store *db.DB,
	cloudManager *provisioners.Manager,
	userManager *user.Manager,
	p2pManager *p2p.P2P,
	networkManager *protosnetwork.Manager,
	instance provisioners.InstanceInfo,
	removeCleanup func(string),
) error {
	fmt.Printf("deleting instance: name=%s id=%s kind=%s provider=%s\n", instance.Name, instance.ID, instance.Kind, instance.KindID)
	if err := cloudManager.DeleteInstance(instance.Name); err != nil {
		return fmt.Errorf("delete instance %s: %w", instance.Name, err)
	}
	if removeCleanup != nil {
		removeCleanup(instance.Name)
	}
	if err := waitForInstanceAbsent(deadline, cloudManager, instance); err != nil {
		return err
	}
	if err := waitForNoAppsForInstance(deadline, store, instance.ID); err != nil {
		return err
	}
	if _, _, err := reconcileTopology(store, cloudManager, userManager, p2pManager, networkManager); err != nil {
		return err
	}
	peerID, err := instancePeerID(instance)
	if err != nil {
		return err
	}
	if err := waitForSwarmionPeerRemoved(deadline, store, peerID); err != nil {
		return err
	}
	fmt.Printf("delete assertion ok: name=%s id=%s\n", instance.Name, instance.ID)
	return nil
}

func waitForInstanceAbsent(deadline time.Time, cloudManager *provisioners.Manager, instance provisioners.InstanceInfo) error {
	var lastErr error
	for time.Now().Before(deadline) {
		nameAbsent, nameErr := instanceLookupAbsent(cloudManager, instance.Name)
		idAbsent, idErr := instanceLookupAbsent(cloudManager, instance.ID)
		if nameAbsent && idAbsent {
			return nil
		}
		lastErr = fmt.Errorf("instance still present by name err=%q by id err=%q", nameErr, idErr)
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("instance %s remained in the declarative DB after deletion: %w", instance.Name, lastErr)
}

func instanceLookupAbsent(cloudManager *provisioners.Manager, id string) (bool, string) {
	_, err := cloudManager.GetInstance(id)
	if err == nil {
		return false, "<nil>"
	}
	return true, err.Error()
}

func waitForNoAppsForInstance(deadline time.Time, store *db.DB, instanceID string) error {
	var lastErr error
	for time.Now().Before(deadline) {
		apps, err := protosapp.CreateManager("", nil, store).GetAll()
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
			continue
		}
		var names []string
		for _, app := range apps {
			if app.InstanceID == instanceID {
				names = append(names, app.Name)
			}
		}
		if len(names) == 0 {
			return nil
		}
		lastErr = fmt.Errorf("apps still reference instance %s: %v", instanceID, names)
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("apps remained after instance %s deletion: %w", instanceID, lastErr)
}

func waitForSwarmionPeerRemoved(deadline time.Time, store *db.DB, peerID string) error {
	var lastErr error
	var lastReport time.Time
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		catchUpErr := store.CatchUpFinalized(ctx, "mixed e2e verify deprovision")
		peerStatuses, peerStatusErr := store.SwarmionPeerStatus(ctx)
		cancel()
		if catchUpErr != nil {
			lastErr = catchUpErr
		} else if peerStatusErr != nil {
			lastErr = peerStatusErr
		} else if err := assertPeerAbsentFromPeerTable(store, peerID); err != nil {
			lastErr = err
		} else {
			status, ok := store.SwarmionStatus()
			if !ok {
				lastErr = fmt.Errorf("swarmion status is unavailable")
			} else if err := assertPeerAbsentFromSwarmionStatus(status, peerStatuses, peerID); err != nil {
				lastErr = err
			} else {
				return nil
			}
		}
		if time.Since(lastReport) >= 30*time.Second {
			lastReport = time.Now()
			if status, ok := store.SwarmionStatus(); ok {
				fmt.Printf("waiting for swarmion peer removal: peer=%s err=%v active=%v eligible=%v providers=%v connected=%v\n", peerID, lastErr, status.ActiveWitnessIDs, status.EligibleWitnessIDs, status.StateProviders, status.ConnectedPeers)
			} else {
				fmt.Printf("waiting for swarmion peer removal: peer=%s err=%v\n", peerID, lastErr)
			}
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("swarmion peer %s remained after deletion: %w", peerID, lastErr)
}

func assertPeerAbsentFromPeerTable(store *db.DB, peerID string) error {
	peerIDs, err := db.GetPeerIDs(store)
	if err != nil {
		return err
	}
	if _, found := peerIDs[peerID]; found {
		return fmt.Errorf("peer table still contains %s", peerID)
	}
	return nil
}

func assertPeerAbsentFromSwarmionStatus(status swarmionapp.Status, peerStatuses []swarmionapp.PeerStatus, peerID string) error {
	if stringSet(status.ActiveWitnessIDs)[peerID] {
		return fmt.Errorf("active witnesses still contain %s", peerID)
	}
	if stringSet(status.EligibleWitnessIDs)[peerID] {
		return fmt.Errorf("eligible witnesses still contain %s", peerID)
	}
	if rank, found := status.EligibleWitnessRanks[peerID]; found && rank > 0 {
		return fmt.Errorf("eligible witness rank for %s is still %d", peerID, rank)
	}
	if stringSet(status.StateProviders)[peerID] {
		return fmt.Errorf("state providers still contain %s", peerID)
	}
	if stringSet(status.ConnectedPeers)[peerID] {
		return fmt.Errorf("connected peers still contain %s", peerID)
	}
	for _, peerStatus := range peerStatuses {
		if peerStatus.PeerID != peerID {
			continue
		}
		if peerStatus.Connected || peerStatus.Dialable || peerStatus.StateProvider || peerStatus.Witness || peerStatus.EligibleWitness {
			return fmt.Errorf("swarmion peer status still has active flags for %s: connected=%t dialable=%t provider=%t witness=%t eligible=%t", peerID, peerStatus.Connected, peerStatus.Dialable, peerStatus.StateProvider, peerStatus.Witness, peerStatus.EligibleWitness)
		}
		return fmt.Errorf("swarmion peer status still contains %s without active flags", peerID)
	}
	return nil
}

func waitForNoInstances(deadline time.Time, cloudManager *provisioners.Manager) error {
	var lastErr error
	for time.Now().Before(deadline) {
		instances, err := cloudManager.GetInstancesWithUpdatedStatus()
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
			continue
		}
		if len(instances) == 0 {
			return nil
		}
		names := make([]string, 0, len(instances))
		for _, instance := range instances {
			names = append(names, instance.Name)
		}
		lastErr = fmt.Errorf("instances still present: %v", names)
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("instances remained after e2e deprovision: %w", lastErr)
}

func waitForNoApps(deadline time.Time, store *db.DB) error {
	var lastErr error
	for time.Now().Before(deadline) {
		apps, err := protosapp.CreateManager("", nil, store).GetAll()
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
			continue
		}
		if len(apps) == 0 {
			return nil
		}
		names := make([]string, 0, len(apps))
		for _, app := range apps {
			names = append(names, app.Name)
		}
		lastErr = fmt.Errorf("apps still present: %v", names)
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("apps remained after e2e deprovision: %w", lastErr)
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
	instances = provisioners.ActiveInstances(instances)
	devices, err := userManager.GetAllDevices(false)
	if err != nil {
		return nil, nil, err
	}
	devices = membership.FilterDevices(devices, peerIDs)
	return instances, devices, nil
}

func waitForRemotePeerConnection(deadline time.Time, clients map[string]*p2p.Client, instances []provisioners.InstanceInfo, peerID string, peerName string) error {
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
			status := peers.Peers[peerID]
			if status != string(p2p.PeerStatusConnected) && peerName != "" {
				status = peers.Peers[peerName]
			}
			if status != string(p2p.PeerStatusConnected) {
				ready = false
				lastErr = fmt.Errorf("%s sees peer id %s / name %s as %q", source.Name, peerID, peerName, status)
				break
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
	return fmt.Errorf("remote VM peer connection to %s (%s) did not become connected: %w", peerName, peerID, lastErr)
}

func waitForRemoteSwarmionConnection(deadline time.Time, clients map[string]*p2p.Client, instances []provisioners.InstanceInfo, peerID string) error {
	var lastErr error
	var lastReport time.Time
	for time.Now().Before(deadline) {
		ready := true
		for _, source := range instances {
			client := clients[source.Name]
			if client == nil {
				ready = false
				lastErr = fmt.Errorf("missing p2p client for %s", source.Name)
				break
			}
			state, err := remoteRuntimeState(client)
			if err != nil {
				ready = false
				lastErr = fmt.Errorf("get remote runtime state for %s: %w", source.Name, err)
				break
			}
			peerStatus := runtimePeerStatus(state, peerID)
			connected := stringSet(state.GetConnectedPeers())[peerID]
			if peerStatus != nil {
				connected = connected || peerStatus.GetConnected()
			}
			if !connected {
				ready = false
				lastErr = fmt.Errorf("%s swarmion transport does not see peer %s connected: %s", source.Name, peerID, runtimeStateSummary(state))
				break
			}
		}
		if ready {
			return nil
		}
		if time.Since(lastReport) >= 30*time.Second {
			lastReport = time.Now()
			fmt.Printf("waiting for remote swarmion connection: %v\n", lastErr)
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("remote Swarmion connection to %s did not become connected: %w", peerID, lastErr)
}

func waitForAllRemoteHeads(deadline time.Time, store *db.DB, clients map[string]*p2p.Client, instances []provisioners.InstanceInfo) error {
	var lastErr error
	var lastReport time.Time
	for time.Now().Before(deadline) {
		localHead, err := currentLocalHead(deadline, store)
		if err != nil {
			lastErr = err
			reportRemoteHeadWait(lastReport, "", nil, lastErr)
			lastReport = time.Now()
			time.Sleep(3 * time.Second)
			continue
		}

		ready := true
		for _, instance := range instances {
			client := clients[instance.Name]
			if client == nil {
				return fmt.Errorf("missing p2p client for %s", instance.Name)
			}
			remoteHead, err := remoteHead(client)
			if err != nil {
				lastErr = fmt.Errorf("%s get remote DB head: %w", instance.Name, err)
				ready = false
				if time.Since(lastReport) >= 30*time.Second {
					reportRemoteHeadWait(lastReport, instance.Name, client, lastErr)
					lastReport = time.Now()
				}
				break
			}
			if remoteHead != localHead.Hash {
				lastErr = fmt.Errorf("%s remote head %q did not match local head %q", instance.Name, remoteHead, localHead.Hash)
				ready = false
				if time.Since(lastReport) >= 30*time.Second {
					reportRemoteHeadWait(lastReport, instance.Name, client, lastErr)
					lastReport = time.Now()
				}
				break
			}
		}
		if ready {
			for _, instance := range instances {
				fmt.Printf("remote DB head matched local head for %s: %s\n", instance.Name, localHead.Hash)
			}
			return nil
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("remote DB head did not match current local DB head: %w", lastErr)
}

func createCloudAppConnectivityPair(
	cfg harnessConfig,
	deadline time.Time,
	store *db.DB,
	cloudManager *provisioners.Manager,
	userManager *user.Manager,
	p2pManager *p2p.P2P,
	networkManager *protosnetwork.Manager,
	deployed []provisioners.InstanceInfo,
	hetznerVM provisioners.InstanceInfo,
	scalewayVM provisioners.InstanceInfo,
	suffix string,
) (*appConnectivityPair, error) {
	appImage := strings.TrimSpace(cfg.appImage)
	if appImage == "" {
		return nil, fmt.Errorf("app image cannot be empty")
	}

	appManager := protosapp.CreateManager("", nil, store)
	if strings.TrimSpace(cfg.seedImageArchive) != "" {
		clients, err := reconnectPeersUntil(deadline, store, cloudManager, userManager, p2pManager, networkManager, instanceNames(deployed))
		if err != nil {
			return nil, err
		}
		if err := loadRemoteSeedImageArchive(deadline, clients[hetznerVM.Name], hetznerVM, cfg.seedImageArchive, appImage); err != nil {
			return nil, err
		}
	}

	hetznerApp, err := appManager.Create(appImage, "app-hetzner-"+suffix, hetznerVM.ID, false, nil)
	if err != nil {
		return nil, fmt.Errorf("create Hetzner app: %w", err)
	}
	fmt.Printf(
		"created cloud P2P image seed app: image=%s seed_instance=%s seed_app=%s ip=%s\n",
		appImage,
		hetznerVM.Name,
		hetznerApp.Name,
		hetznerApp.IPString(),
	)

	if err := appManager.Start(hetznerApp.Name); err != nil {
		return nil, fmt.Errorf("request Hetzner app start: %w", err)
	}
	clients, err := syncRemoteTopologyAfterAppWrite(deadline, store, cloudManager, userManager, p2pManager, networkManager, deployed, "cloud image seed app start")
	if err != nil {
		return nil, err
	}
	if err := waitForRemoteAppStatus(deadline, clients[hetznerVM.Name], hetznerVM.Name, hetznerApp.Name, "running"); err != nil {
		return nil, err
	}
	clients, err = reconnectPeersUntil(deadline, store, cloudManager, userManager, p2pManager, networkManager, instanceNames(deployed))
	if err != nil {
		return nil, err
	}
	if err := waitForRemoteImageContentReady(deadline, clients[hetznerVM.Name], hetznerVM.Name, appImage); err != nil {
		return nil, err
	}

	scalewayApp, err := appManager.Create(appImage, "app-scaleway-"+suffix, scalewayVM.ID, false, nil)
	if err != nil {
		return nil, fmt.Errorf("create Scaleway app: %w", err)
	}
	fmt.Printf(
		"created cloud P2P image pull app: image=%s pull_instance=%s pull_app=%s ip=%s\n",
		appImage,
		scalewayVM.Name,
		scalewayApp.Name,
		scalewayApp.IPString(),
	)
	if err := appManager.Start(scalewayApp.Name); err != nil {
		return nil, fmt.Errorf("request Scaleway app start: %w", err)
	}
	clients, err = syncRemoteTopologyAfterAppWrite(deadline, store, cloudManager, userManager, p2pManager, networkManager, deployed, "cloud image pull app start")
	if err != nil {
		return nil, err
	}
	if err := waitForRemoteAppStatus(deadline, clients[scalewayVM.Name], scalewayVM.Name, scalewayApp.Name, "running"); err != nil {
		return nil, err
	}
	if err := waitForRemoteP2PImageLabel(deadline, clients[scalewayVM.Name], scalewayVM.Name, appImage); err != nil {
		return nil, err
	}
	fmt.Printf(
		"cloud P2P image resolution verified: seed=%s puller=%s image=%s\n",
		hetznerVM.Name,
		scalewayVM.Name,
		appImage,
	)

	return &appConnectivityPair{hetznerApp: hetznerApp, scalewayApp: scalewayApp}, nil
}

func loadRemoteSeedImageArchive(
	deadline time.Time,
	client *p2p.Client,
	instance provisioners.InstanceInfo,
	archivePath string,
	imageRef string,
) error {
	if client == nil {
		return fmt.Errorf("missing p2p client for %s", instance.Name)
	}
	archivePath = strings.TrimSpace(archivePath)
	if archivePath == "" {
		return nil
	}
	absPath, err := filepath.Abs(archivePath)
	if err != nil {
		return err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("stat seed image archive %s: %w", absPath, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("seed image archive %s is not a regular file", absPath)
	}

	file, err := os.Open(absPath)
	if err != nil {
		return err
	}
	defer file.Close()

	uploadID := fmt.Sprintf("seed-%d", time.Now().UnixNano())
	timeout := time.Until(deadline)
	if timeout <= 0 {
		return fmt.Errorf("deadline passed before seed image archive load")
	}
	if timeout > 10*time.Minute {
		timeout = 10 * time.Minute
	}
	deadline = time.Now().Add(timeout)
	buf := make([]byte, 1024*1024)
	var offset uint64
	for {
		n, readErr := file.Read(buf)
		eof := readErr == io.EOF
		if readErr != nil && !eof {
			return fmt.Errorf("read seed image archive %s: %w", absPath, readErr)
		}
		if n == 0 && !eof {
			continue
		}
		callTimeout := time.Until(deadline)
		if callTimeout <= 0 {
			return fmt.Errorf("deadline passed while uploading seed image archive to %s", instance.Name)
		}
		if callTimeout > 2*time.Minute {
			callTimeout = 2 * time.Minute
		}
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		resp, err := client.UploadImageArchiveChunk(ctx, &p2pproto.UploadImageArchiveChunkRequest{
			UploadId: uploadID,
			ImageRef: imageRef,
			Offset:   offset,
			Data:     append([]byte(nil), buf[:n]...),
			Eof:      eof,
		})
		cancel()
		if err != nil {
			if st, ok := status.FromError(err); ok {
				return fmt.Errorf("upload seed image archive to %s: %s", instance.Name, st.Message())
			}
			return fmt.Errorf("upload seed image archive to %s: %w", instance.Name, err)
		}
		offset = resp.GetReceivedBytes()
		if !eof {
			continue
		}
		if !resp.GetLoaded() {
			return fmt.Errorf("seed image archive upload to %s reached EOF without loading image", instance.Name)
		}
		fmt.Printf(
			"remote seed image archive loaded: instance=%s image=%s digest=%s platform=%s bytes=%d\n",
			instance.Name,
			resp.GetImageRef(),
			resp.GetTargetDigest(),
			resp.GetPlatform(),
			offset,
		)
		return nil
	}
}

func syncRemoteTopologyAfterAppWrite(
	deadline time.Time,
	store *db.DB,
	cloudManager *provisioners.Manager,
	userManager *user.Manager,
	p2pManager *p2p.P2P,
	networkManager *protosnetwork.Manager,
	instances []provisioners.InstanceInfo,
	label string,
) (map[string]*p2p.Client, error) {
	if _, _, err := reconcileTopology(store, cloudManager, userManager, p2pManager, networkManager); err != nil {
		return nil, fmt.Errorf("reconcile app routes after %s: %w", label, err)
	}
	if err := waitForLocalHeadFinalized(deadline, store, label); err != nil {
		return nil, err
	}
	clients, err := reconnectPeersUntil(deadline, store, cloudManager, userManager, p2pManager, networkManager, instanceNames(instances))
	if err != nil {
		return nil, err
	}
	if err := waitForAllRemoteHeads(deadline, store, clients, instances); err != nil {
		return nil, fmt.Errorf("%s DB sync failed: %w", label, err)
	}
	return clients, nil
}

func verifyCloudAppConnectivity(
	deadline time.Time,
	clients map[string]*p2p.Client,
	hetznerVM provisioners.InstanceInfo,
	scalewayVM provisioners.InstanceInfo,
	appPair *appConnectivityPair,
) error {
	if appPair == nil || appPair.hetznerApp == nil || appPair.scalewayApp == nil {
		return fmt.Errorf("cloud app connectivity pair is required")
	}
	hetznerApp := appPair.hetznerApp
	scalewayApp := appPair.scalewayApp

	if err := waitForRemoteAppStatus(deadline, clients[hetznerVM.Name], hetznerVM.Name, hetznerApp.Name, "running"); err != nil {
		return err
	}
	if err := waitForRemoteAppStatus(deadline, clients[scalewayVM.Name], scalewayVM.Name, scalewayApp.Name, "running"); err != nil {
		return err
	}

	if err := waitForContainerHTTPToApp(deadline, hetznerVM, hetznerApp, scalewayApp); err != nil {
		return err
	}
	if err := waitForContainerHTTPToApp(deadline, scalewayVM, scalewayApp, hetznerApp); err != nil {
		return err
	}
	return nil
}

func waitForRemoteAppStatus(deadline time.Time, client *p2p.Client, instanceName string, appName string, want string) error {
	if client == nil {
		return fmt.Errorf("missing p2p client for %s", instanceName)
	}
	var lastErr error
	var lastReport time.Time
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		statusResp, err := client.GetAppStatus(ctx, &p2pproto.GetAppStatusRequest{AppName: appName})
		cancel()
		if err == nil {
			got := strings.TrimSpace(statusResp.GetStatus())
			if got == want {
				fmt.Printf("remote app status ok: instance=%s app=%s status=%s\n", instanceName, appName, got)
				return nil
			}
			lastErr = fmt.Errorf("status=%q, want %q", got, want)
			if time.Since(lastReport) >= 30*time.Second {
				lastReport = time.Now()
				fmt.Printf("waiting for remote app status: instance=%s app=%s %v\n", instanceName, appName, lastErr)
			}
		} else {
			if st, ok := status.FromError(err); ok {
				lastErr = fmt.Errorf("%s", st.Message())
			} else {
				lastErr = err
			}
			if time.Since(lastReport) >= 30*time.Second {
				lastReport = time.Now()
				fmt.Printf("waiting for remote app status: instance=%s app=%s err=%v\n", instanceName, appName, lastErr)
			}
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("app %s on %s did not reach status %q: %w", appName, instanceName, want, lastErr)
}

func waitForRemoteImageContentReady(deadline time.Time, client *p2p.Client, instanceName string, imageRef string) error {
	if client == nil {
		return fmt.Errorf("missing p2p client for %s", instanceName)
	}
	var lastErr error
	var lastReport time.Time
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		describeResp, describeErr := client.DescribeImage(ctx, &p2pproto.DescribeImageRequest{ImageRef: imageRef})
		contentResp, contentErr := client.GetImageContent(ctx, &p2pproto.GetImageContentRequest{ImageRef: imageRef})
		cancel()
		if describeErr == nil && contentErr == nil && describeResp.GetFound() && contentResp.GetFound() && contentResp.GetTarget() != nil && len(contentResp.GetDescriptors()) > 0 {
			fmt.Printf(
				"remote image content ready: instance=%s image=%s digest=%s blobs=%d\n",
				instanceName,
				imageRef,
				describeResp.GetTargetDigest(),
				len(contentResp.GetDescriptors()),
			)
			return nil
		}
		if describeErr != nil {
			if st, ok := status.FromError(describeErr); ok {
				lastErr = fmt.Errorf("describe: %s", st.Message())
			} else {
				lastErr = fmt.Errorf("describe: %w", describeErr)
			}
		} else if !describeResp.GetFound() {
			lastErr = fmt.Errorf("image %s is not present on %s", imageRef, instanceName)
		} else if contentErr != nil {
			if st, ok := status.FromError(contentErr); ok {
				lastErr = fmt.Errorf("content: %s", st.Message())
			} else {
				lastErr = fmt.Errorf("content: %w", contentErr)
			}
		} else if !contentResp.GetFound() {
			lastErr = fmt.Errorf("image content %s is not present on %s", imageRef, instanceName)
		} else if contentResp.GetTarget() == nil {
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

func waitForRemoteP2PImageLabel(deadline time.Time, client *p2p.Client, instanceName string, imageRef string) error {
	if client == nil {
		return fmt.Errorf("missing p2p client for %s", instanceName)
	}
	var lastErr error
	var lastReport time.Time
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		imageResp, err := client.DescribeImage(ctx, &p2pproto.DescribeImageRequest{ImageRef: imageRef})
		cancel()
		if err != nil {
			if st, ok := status.FromError(err); ok {
				lastErr = fmt.Errorf("%s", st.Message())
			} else {
				lastErr = err
			}
			time.Sleep(5 * time.Second)
			continue
		}
		if !imageResp.GetFound() {
			lastErr = fmt.Errorf("image %s is not present", imageRef)
			time.Sleep(5 * time.Second)
			continue
		}
		if imageResp.GetLabels()["protos.io/image.source"] == "p2p" {
			fmt.Printf("remote P2P image label observed: instance=%s image=%s digest=%s\n", instanceName, imageRef, imageResp.GetTargetDigest())
			return nil
		}
		lastErr = fmt.Errorf("image %s label protos.io/image.source=%q, want p2p", imageRef, imageResp.GetLabels()["protos.io/image.source"])
		if time.Since(lastReport) >= 30*time.Second {
			lastReport = time.Now()
			fmt.Printf("waiting for remote P2P image label: instance=%s image=%s err=%v\n", instanceName, imageRef, lastErr)
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("%s did not report Protos P2P image source for %s: %w", instanceName, imageRef, lastErr)
}

func waitForContainerHTTPToApp(
	deadline time.Time,
	source provisioners.InstanceInfo,
	sourceApp *protosapp.App,
	targetApp *protosapp.App,
) error {
	if sourceApp == nil || targetApp == nil {
		return fmt.Errorf("source and target apps are required")
	}
	if sourceApp.IP == nil {
		return fmt.Errorf("source app %s has no overlay IP", sourceApp.Name)
	}
	if targetApp.IP == nil {
		return fmt.Errorf("target app %s has no overlay IP", targetApp.Name)
	}
	var lastErr error
	var lastReport time.Time
	client := &http.Client{Timeout: 15 * time.Second}
	for time.Now().Before(deadline) {
		targetURL := appProbeURL(targetApp, "/")
		probeURL := appProbeURL(sourceApp, "/probe") + "?target=" + url.QueryEscape(targetURL) + "&timeout_ms=8000&max_bytes=4096"
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		resp, err := httpGetProbe(ctx, client, probeURL)
		cancel()
		if err == nil && resp.OK {
			fmt.Printf("app connectivity ok: source_instance=%s source_app=%s target_app=%s target_ip=%s bytes=%d status=%d\n", source.Name, sourceApp.Name, targetApp.Name, targetApp.IP.String(), resp.BytesRead, resp.StatusCode)
			return nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("probe returned ok=false status=%d error=%q", resp.StatusCode, resp.Error)
		}
		if time.Since(lastReport) >= 30*time.Second {
			lastReport = time.Now()
			fmt.Printf("waiting for app connectivity: source_instance=%s source_app=%s target_app=%s err=%v\n", source.Name, sourceApp.Name, targetApp.Name, lastErr)
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("app %s on %s could not reach app %s over the overlay: %w", sourceApp.Name, source.Name, targetApp.Name, lastErr)
}

func appProbeURL(app *protosapp.App, path string) string {
	return fmt.Sprintf("http://[%s]:%d%s", app.IP.String(), probeAppPort, path)
}

func httpGetProbe(ctx context.Context, client *http.Client, probeURL string) (probeAppResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, nil)
	if err != nil {
		return probeAppResponse{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return probeAppResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return probeAppResponse{}, err
	}
	var out probeAppResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return probeAppResponse{}, fmt.Errorf("decode probe response status=%d body=%q: %w", resp.StatusCode, strings.TrimSpace(string(body)), err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return out, fmt.Errorf("probe endpoint returned status=%d ok=%v error=%q", resp.StatusCode, out.OK, out.Error)
	}
	return out, nil
}

func instanceNames(instances []provisioners.InstanceInfo) []string {
	names := make([]string, 0, len(instances))
	for _, instance := range instances {
		names = append(names, instance.Name)
	}
	return names
}

func currentLocalHead(deadline time.Time, store *db.DB) (db.Commit, error) {
	timeout := time.Until(deadline)
	if timeout <= 0 {
		return db.Commit{}, fmt.Errorf("deadline passed before reading local DB head")
	}
	if timeout > 10*time.Second {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	err := store.CatchUpFinalized(ctx, "mixed witness remote DB head check")
	cancel()
	if err != nil {
		return db.Commit{}, fmt.Errorf("catch up local finalized DB head: %w", err)
	}
	head, err := store.GetLastCommit("main")
	if err != nil {
		return db.Commit{}, fmt.Errorf("get local DB head: %w", err)
	}
	return head, nil
}

func remoteHead(client *p2p.Client) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	head, err := client.GetHead(ctx, &p2pproto.GetHeadRequest{})
	cancel()
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return "", fmt.Errorf("%s", st.Message())
		}
		return "", err
	}
	return strings.TrimSpace(head.GetCommit()), nil
}

func reportRemoteHeadWait(lastReport time.Time, instanceName string, client *p2p.Client, err error) {
	if !lastReport.IsZero() && time.Since(lastReport) < 30*time.Second {
		return
	}
	if client == nil {
		fmt.Printf("waiting for remote DB head: err=%v\n", err)
		return
	}
	if state, stateErr := remoteRuntimeState(client); stateErr == nil {
		fmt.Printf("waiting for remote DB head: instance=%s err=%v runtime={%s}\n", instanceName, err, runtimeStateSummary(state))
	} else {
		fmt.Printf("waiting for remote DB head: instance=%s err=%v runtime_err=%v\n", instanceName, err, stateErr)
	}
}

func remoteRuntimeState(client *p2p.Client) (*p2pproto.RuntimeState, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetRuntimeState(ctx, &p2pproto.GetRuntimeStateRequest{})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			return nil, fmt.Errorf("%s", st.Message())
		}
		return nil, err
	}
	if resp.GetState() == nil {
		return nil, fmt.Errorf("empty runtime state")
	}
	return resp.GetState(), nil
}

func runtimePeerStatus(state *p2pproto.RuntimeState, peerID string) *p2pproto.RuntimePeerStatus {
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

func runtimeStateSummary(state *p2pproto.RuntimeState) string {
	if state == nil {
		return "nil"
	}
	trace := state.GetContentSyncTrace()
	if len(trace) > 8 {
		trace = trace[len(trace)-8:]
	}
	return fmt.Sprintf(
		"peer=%s active=%v eligible=%v providers=%v connected=%v finalized_root=%s protocol_root=%s durable_root=%s epoch=%s pending_finalized=%t finalized_error=%q refresh_pending=%t refresh_error=%q fatal=%q trace=%v",
		state.GetPeerId(),
		state.GetActiveWitnessIds(),
		state.GetEligibleWitnessIds(),
		state.GetStateProviders(),
		state.GetConnectedPeers(),
		state.GetFinalizedRootHash(),
		state.GetProtocolFinalizedRootHash(),
		state.GetDurableMainRootHash(),
		state.GetActiveEpochId(),
		state.GetRuntimeFinalizedPending(),
		state.GetRuntimeFinalizedLastError(),
		state.GetRuntimeRefreshPending(),
		state.GetRuntimeRefreshLastError(),
		state.GetFatalState(),
		trace,
	)
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
