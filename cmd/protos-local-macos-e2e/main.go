package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/semver"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/membership"
	protosnetwork "github.com/protosio/protos/internal/network"
	"github.com/protosio/protos/internal/p2p"
	p2pproto "github.com/protosio/protos/internal/p2p/proto"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/provisioners"
	"github.com/protosio/protos/internal/release"
	"github.com/protosio/protos/internal/user"
	"github.com/protosio/protos/provisioners/local_macos"
	"github.com/protosio/protos/provisioners/scaleway"
	"google.golang.org/grpc/status"
)

type noopAppManager struct{}

func (noopAppManager) GetLogs(string) ([]byte, error) {
	return nil, fmt.Errorf("logs are not available in the e2e harness")
}

func (noopAppManager) GetStatus(string) (string, error) {
	return "", fmt.Errorf("status is not available in the e2e harness")
}

func main() {
	imagePath := flag.String("image", "provisioner-images/targets/output/mactest", "LinuxKit image directory or ISO path")
	workDir := flag.String("workdir", "", "temporary Protos workdir")
	keep := flag.Bool("keep", false, "keep temporary state after the run")
	timeout := flag.Duration("timeout", 6*time.Minute, "overall verification timeout")
	machineType := flag.String("machine", "vz-2c-2g", "local macOS machine type")
	instanceCount := flag.Int("instances", 2, "number of local macOS VMs to deploy and verify")
	configureNetwork := flag.Bool("network", true, "configure the host network module through protos-vm-hostagent")
	flag.Parse()

	if err := run(*imagePath, *workDir, *keep, *timeout, *machineType, *instanceCount, *configureNetwork); err != nil {
		fmt.Fprintf(os.Stderr, "e2e failed: %v\n", err)
		os.Exit(1)
	}
}

func run(imagePath string, workDir string, keep bool, timeout time.Duration, machineType string, instanceCount int, configureNetwork bool) error {
	if instanceCount < 1 {
		return fmt.Errorf("instances must be greater than zero")
	}
	imagePath, err := filepath.Abs(imagePath)
	if err != nil {
		return err
	}
	if workDir == "" {
		workDir, err = os.MkdirTemp("", "protos-local-macos-e2e-*")
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
	admin, err := userManager.CreateUser("e2e", "E2E", true)
	if err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}
	if err := userManager.AddDevice(admin.Username, "local-e2e", localKey); err != nil {
		return fmt.Errorf("add current device: %w", err)
	}

	var networkManager *protosnetwork.Manager
	if configureNetwork {
		networkManager, err = protosnetwork.NewManager()
		if err != nil {
			return fmt.Errorf("create network manager: %w", err)
		}
		if err := networkManager.Init(localKey, config.Get().InternalDomain); err != nil {
			_ = networkManager.Close()
			return fmt.Errorf("initialize network manager through protos-vm-hostagent: %w", err)
		}
		defer func() {
			_ = networkManager.ConfigurePeers(nil, nil)
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
		scaleway.NewFactory(),
		localmacos.NewFactory(),
	)
	if err != nil {
		return fmt.Errorf("create provisioner manager: %w", err)
	}

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	providerName := "local-e2e-" + suffix
	imageName := "mactest-e2e-" + suffix
	vmDir := filepath.Join(workDir, "local-macos-vms")

	var cleanupInstances []string
	cleanupImage := false
	defer func() {
		for i := len(cleanupInstances) - 1; i >= 0; i-- {
			_ = cloudManager.DeleteInstance(cleanupInstances[i])
		}
		if cleanupImage {
			provider, err := cloudManager.GetProvider(providerName)
			if err == nil {
				if imageProvider, ok := provider.(provisioners.ImageProvisioner); ok {
					_ = imageProvider.RemoveImage(imageName, "local")
				}
			}
		}
		_ = cloudManager.DeleteProvisioner(providerName)
	}()

	if err := cloudManager.AddProvisioner(providerName, localmacos.Type.String(), map[string]string{"VM_DIR": vmDir}); err != nil {
		return fmt.Errorf("add local macOS provisioner: %w", err)
	}
	if err := cloudManager.UploadLocalImage(imagePath, imageName, providerName, "local", 0); err != nil {
		return fmt.Errorf("upload local image: %w", err)
	}
	cleanupImage = true

	rel := release.Release{
		Version:     imageName,
		CloudImages: map[string]release.CloudImage{},
	}

	deadline := time.Now().Add(timeout)
	var deployed []provisioners.InstanceInfo
	for i := 0; i < instanceCount; i++ {
		instanceName := fmt.Sprintf("vm-e2e-%s-%d", suffix, i+1)
		instance, err := cloudManager.DeployInstance(instanceName, providerName, "local", rel, machineType)
		if err != nil {
			return fmt.Errorf("deploy local macOS instance %s: %w", instanceName, err)
		}
		cleanupInstances = append(cleanupInstances, instanceName)
		deployed = append(deployed, instance)
		fmt.Printf("deployed instance: name=%s id=%s ip=%s wg=%s arch=%s status=%s\n", instance.Name, instance.ID, instance.PublicIP, wireGuardIPv6(instance), instance.Architecture, instance.Status)

		if _, err := reconnectPeersUntil(deadline, store, cloudManager, userManager, p2pManager, networkManager, instanceNames(deployed)); err != nil {
			return err
		}
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
	if len(deployed) > 1 {
		if err := waitForRemoteFullMesh(deadline, clients, deployed); err != nil {
			return err
		}
	}

	localHead, err := store.GetLastCommit("main")
	if err != nil {
		return fmt.Errorf("get local DB head: %w", err)
	}
	for _, instance := range deployed {
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

func reconnectPeersUntil(deadline time.Time, store *db.DB, cloudManager *provisioners.Manager, userManager *user.Manager, p2pManager *p2p.P2P, networkManager *protosnetwork.Manager, wantNames []string) (map[string]*p2p.Client, error) {
	var lastErr error
	for time.Now().Before(deadline) {
		instances, devices, err := declarativePeers(store, cloudManager, userManager)
		if err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
			continue
		}
		if networkManager != nil {
			if err := networkManager.ConfigurePeers(instances, devices); err != nil {
				lastErr = fmt.Errorf("configure network peers: %w", err)
				time.Sleep(2 * time.Second)
				continue
			}
		}
		if err := p2pManager.ConfigurePeers(membership.Machines(instances, devices)); err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
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
		time.Sleep(2 * time.Second)
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
