//go:build darwin

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

	"github.com/Masterminds/semver/v3"
	protosapp "github.com/protosio/protos/internal/app"
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
	imagePath := flag.String("image", "../cloud-provisioning/targets/output/mactest", "LinuxKit image directory or ISO path")
	workDir := flag.String("workdir", "", "temporary Protos workdir")
	keep := flag.Bool("keep", false, "keep temporary state after the run")
	timeout := flag.Duration("timeout", 6*time.Minute, "overall verification timeout")
	machineType := flag.String("machine", "vz-2c-2g", "local macOS machine type")
	instanceCount := flag.Int("instances", 2, "number of local macOS VMs to deploy and verify")
	configureNetwork := flag.Bool("network", true, "configure the host network module through protos-hostagent")
	appImage := flag.String("app-image", "", "optional app image used to verify Protos P2P image resolution across local VMs")
	flag.Parse()

	if err := run(*imagePath, *workDir, *keep, *timeout, *machineType, *instanceCount, *configureNetwork, *appImage); err != nil {
		fmt.Fprintf(os.Stderr, "e2e failed: %v\n", err)
		os.Exit(1)
	}
}

func run(imagePath string, workDir string, keep bool, timeout time.Duration, machineType string, instanceCount int, configureNetwork bool, appImage string) error {
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
		if _, err := cloudManager.DeployInstance(instanceName, providerName, "local", rel, machineType); err != nil {
			return fmt.Errorf("deploy local macOS instance %s: %w", instanceName, err)
		}
		cleanupInstances = append(cleanupInstances, instanceName)
		instance, err := waitForInstanceReady(deadline, cloudManager, instanceName)
		if err != nil {
			return err
		}
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

	if strings.TrimSpace(appImage) != "" {
		if err := verifyLocalImageRegistry(deadline, store, cloudManager, userManager, p2pManager, networkManager, deployed, appImage, suffix); err != nil {
			return err
		}
	}

	return nil
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
			routes, err := appRoutes(store, instances)
			if err != nil {
				lastErr = fmt.Errorf("load app routes: %w", err)
				time.Sleep(2 * time.Second)
				continue
			}
			if err := networkManager.ConfigurePeers(instances, devices, routes, nil); err != nil {
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

func verifyLocalImageRegistry(
	deadline time.Time,
	store *db.DB,
	cloudManager *provisioners.Manager,
	userManager *user.Manager,
	p2pManager *p2p.P2P,
	networkManager *protosnetwork.Manager,
	deployed []provisioners.InstanceInfo,
	appImage string,
	suffix string,
) error {
	if len(deployed) < 2 {
		return fmt.Errorf("image registry verification requires at least two local VMs")
	}
	appImage = strings.TrimSpace(appImage)
	if appImage == "" {
		return fmt.Errorf("app image is empty")
	}

	appManager := protosapp.CreateManager("", nil, store)
	seedVM := deployed[0]
	pullVM := deployed[1]

	seedApp, err := appManager.Create(appImage, "image-seed-"+suffix, seedVM.ID, false, nil)
	if err != nil {
		return fmt.Errorf("create seed app: %w", err)
	}
	if err := appManager.Start(seedApp.Name); err != nil {
		return fmt.Errorf("start seed app: %w", err)
	}
	if err := reconcileLocalTopology(store, cloudManager, userManager, p2pManager, networkManager); err != nil {
		return err
	}
	if err := waitForLocalHeadFinalized(deadline, store, "seed app start"); err != nil {
		return err
	}
	clients, err := reconnectPeersUntil(deadline, store, cloudManager, userManager, p2pManager, networkManager, instanceNames(deployed))
	if err != nil {
		return err
	}
	if err := waitForAllRemoteHeads(deadline, store, clients, deployed); err != nil {
		return err
	}
	if err := waitForRemoteAppStatus(deadline, clients[seedVM.Name], seedVM.Name, seedApp.Name, "running"); err != nil {
		return err
	}
	clients, err = reconnectPeersUntil(deadline, store, cloudManager, userManager, p2pManager, networkManager, instanceNames(deployed))
	if err != nil {
		return err
	}
	if err := waitForRemoteImageContentReady(deadline, clients[seedVM.Name], seedVM.Name, appImage); err != nil {
		return err
	}

	pullApp, err := appManager.Create(appImage, "image-pull-"+suffix, pullVM.ID, false, nil)
	if err != nil {
		return fmt.Errorf("create pull app: %w", err)
	}
	if err := appManager.Start(pullApp.Name); err != nil {
		return fmt.Errorf("start pull app: %w", err)
	}
	if err := reconcileLocalTopology(store, cloudManager, userManager, p2pManager, networkManager); err != nil {
		return err
	}
	if err := waitForLocalHeadFinalized(deadline, store, "pull app start"); err != nil {
		return err
	}
	clients, err = reconnectPeersUntil(deadline, store, cloudManager, userManager, p2pManager, networkManager, instanceNames(deployed))
	if err != nil {
		return err
	}
	if err := waitForAllRemoteHeads(deadline, store, clients, deployed); err != nil {
		return err
	}
	if err := waitForRemoteAppStatus(deadline, clients[pullVM.Name], pullVM.Name, pullApp.Name, "running"); err != nil {
		return err
	}
	if err := waitForRemoteP2PImageLabel(deadline, clients[pullVM.Name], pullVM.Name, appImage); err != nil {
		return err
	}
	fmt.Printf("Protos P2P image resolution verified: seed=%s puller=%s image=%s\n", seedVM.Name, pullVM.Name, appImage)
	return nil
}

func reconcileLocalTopology(
	store *db.DB,
	cloudManager *provisioners.Manager,
	userManager *user.Manager,
	p2pManager *p2p.P2P,
	networkManager *protosnetwork.Manager,
) error {
	instances, devices, err := declarativePeers(store, cloudManager, userManager)
	if err != nil {
		return err
	}
	if networkManager != nil {
		routes, err := appRoutes(store, instances)
		if err != nil {
			return fmt.Errorf("load app routes: %w", err)
		}
		if err := networkManager.ConfigurePeers(instances, devices, routes, nil); err != nil {
			return fmt.Errorf("configure network peers: %w", err)
		}
	}
	if err := p2pManager.ConfigurePeers(membership.Machines(instances, devices)); err != nil {
		return err
	}
	if err := store.ReconcileWitnesses(context.Background(), membership.WitnessCandidates(instances, devices)); err != nil {
		return err
	}
	return nil
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

func waitForLocalHeadFinalized(deadline time.Time, store *db.DB, label string) error {
	target, err := store.GetLastCommit("main")
	if err != nil {
		return fmt.Errorf("get local %s head: %w", label, err)
	}
	var lastErr error
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
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("local authored head did not finalize for %s: %w", label, lastErr)
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
		} else if st, ok := status.FromError(err); ok {
			lastErr = fmt.Errorf("%s", st.Message())
		} else {
			lastErr = err
		}
		if time.Since(lastReport) >= 30*time.Second {
			lastReport = time.Now()
			fmt.Printf("waiting for remote app status: instance=%s app=%s err=%v\n", instanceName, appName, lastErr)
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
