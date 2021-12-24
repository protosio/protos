package platform

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/auth"
	"github.com/protosio/protos/internal/util"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const (
	protosNamespace string = "protos"
)

var pltfrm *containerdPlatform

type containerdPlatform struct {
	endpoint          string
	appStoreHost      string
	internalInterface string
	wireguardIP       net.IP
	logsPath          string
	initSignal        chan net.IP
	network           net.IPNet
	key               wgtypes.Key
	client            *containerd.Client
	initLock          *sync.RWMutex
}

func createContainerdRuntimePlatform(runtimeUnixSocket string, appStoreHost string, inContainer bool, key wgtypes.Key, logsPath string) *containerdPlatform {
	if pltfrm == nil {
		pltfrm = &containerdPlatform{
			endpoint:     runtimeUnixSocket,
			appStoreHost: appStoreHost,
			logsPath:     logsPath,
			initSignal:   make(chan net.IP, 1),
			key:          key,
			network:      net.IPNet{},
			initLock:     &sync.RWMutex{},
		}
	}

	return pltfrm
}

func (cdp *containerdPlatform) Init(network net.IPNet, devices []auth.UserDevice) error {

	if _, err := os.Stat(cdp.logsPath); os.IsNotExist(err) {
		err := os.Mkdir(cdp.logsPath, os.ModeDir)
		if err != nil {
			return fmt.Errorf("failed to initialize platform. Failed to create logs directory: %s", err.Error())
		}
	}

	var err error

	cdp.initLock.Lock()

	if cdp.internalInterface == "" {
		internalInterface, wireguardIP, err := initNetwork(network, devices, cdp.key)
		if err != nil {
			return fmt.Errorf("can't initialize network: %s", err.Error())
		}
		cdp.internalInterface = internalInterface
		cdp.wireguardIP = wireguardIP
	}

	if cdp.client == nil {
		log.Infof("Connecting to the containerd daemon using endpoint '%s'", cdp.endpoint)
		cdp.client, err = containerd.New(cdp.endpoint)
		if err != nil {
			return errors.Wrap(err, "Failed to initialize containerd runtime. Failed to connect, make sure you are running as root and the runtime has been started")
		}
	}

	cdp.network = network

	cdp.initLock.Unlock()
	return nil
}

func (cdp *containerdPlatform) NewSandbox(name string, appID string, imageID string, volumeID string, volumeMountPath string, publicPorts []util.Port, installerParams map[string]string) (PlatformRuntimeUnit, error) {
	ctx := namespaces.WithNamespace(context.Background(), protosNamespace)
	pru := &containerdSandbox{p: cdp}

	repoImage := cdp.appStoreHost + "/" + imageID
	image, err := cdp.client.GetImage(ctx, repoImage)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not retrieve image '%s' from containerd", imageID)
	}

	opts := []oci.SpecOpts{
		oci.WithImageConfig(image),
		oci.WithEnv([]string{fmt.Sprintf("APPID=%s", appID)}),
	}

	log.Debugf("Creating containerd sandbox '%s' from image '%s'", name, imageID)
	cnt, err := cdp.client.NewContainer(
		ctx,
		appID,
		containerd.WithNewSnapshot(appID+"-snapshot", image),
		containerd.WithNewSpec(opts...),
		containerd.WithContainerLabels(map[string]string{"platform": protosNamespace, "appID": appID, "appName": name}),
	)
	if err != nil {
		return pru, errors.Wrapf(err, "Failed to create container '%s' for app '%s'", name, appID)
	}

	pru.containerID = appID
	logFilePath := fmt.Sprintf("%s/%s.log", cdp.logsPath, appID)
	pru.task, err = cnt.NewTask(ctx, cio.LogFile(logFilePath))
	if err != nil {
		return pru, errors.Wrapf(err, "Failed to create task '%s' for app '%s'", name, appID)
	}

	netNSpath := fmt.Sprintf("/proc/%d/ns/net", pru.task.Pid())
	usedIPs, err := cdp.getAllIPs()
	if err != nil {
		return pru, fmt.Errorf("failed to allocate IP for app '%s': %v", appID, err)
	}

	newIP, err := allocateIP(cdp.network, usedIPs)
	if err != nil {
		return pru, fmt.Errorf("failed to allocate IP for app '%s': %v", appID, err)
	}

	err = configureInterface(netNSpath, newIP, cdp.network, cdp.wireguardIP)
	if err != nil {
		return pru, fmt.Errorf("failed to configure network interface for app '%s': %v", appID, err)
	}

	log.Debugf("Created task for containerd sandbox '%s', with PID '%d' and ip '%s'", appID, pru.task.Pid(), newIP.String())

	return pru, nil
}

func (cdp *containerdPlatform) GetImage(id string) (PlatformImage, error) {
	ctx := namespaces.WithNamespace(context.Background(), protosNamespace)

	repoImage := cdp.appStoreHost + "/" + id
	image, err := cdp.client.GetImage(ctx, repoImage)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "Failed to retrieve image '%s' from containerd", id)
	}

	_, normalizedID, err := normalizeRepoDigest([]string{id})
	if err != nil {
		return nil, errors.Wrapf(err, "Could not retrieve image '%s' from containerd. Failed to normalize digest", id)
	}

	pi := &platformImage{
		id:      id,
		localID: normalizedID,
		// repoTags: imageResponse.Metadata().Labels,
		labels: image.Labels(),
	}

	return pi, nil
}

func (cdp *containerdPlatform) GetAllImages() (map[string]PlatformImage, error) {
	ctx := namespaces.WithNamespace(context.Background(), protosNamespace)
	images := map[string]PlatformImage{}

	listImagesResponse, err := cdp.client.ListImages(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Could not retrieve images from containerd")
	}

	for _, img := range listImagesResponse {
		image := platformImage{
			localID: img.Name(),
			labels:  img.Labels(),
		}
		images[image.id] = &image
	}

	return images, nil
}

func (cdp *containerdPlatform) GetSandbox(id string) (PlatformRuntimeUnit, error) {
	ctx := namespaces.WithNamespace(context.Background(), protosNamespace)
	if id == "" {
		return nil, util.NewTypedError("Container ID can't be empty", ErrContainerNotFound)
	}

	cnt, err := cdp.client.LoadContainer(ctx, id)
	if err != nil {
		return nil, util.NewTypedError("Container not found", ErrContainerNotFound)
	}

	task, err := cnt.Task(ctx, nil)
	if err != nil {
		return nil, util.NewTypedError("Task sandbox not found", ErrContainerNotFound)
	}

	return &containerdSandbox{p: cdp, task: task, cnt: cnt, containerID: id}, nil
}

func (cdp *containerdPlatform) GetAllSandboxes() (map[string]PlatformRuntimeUnit, error) {
	ctx := namespaces.WithNamespace(context.Background(), protosNamespace)

	containers := map[string]PlatformRuntimeUnit{}

	cnts, err := cdp.client.Containers(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to retrieve containers")
	}

	for _, cnt := range cnts {
		task, err := cnt.Task(ctx, nil)
		if err != nil {
			continue
		}
		containers[cnt.ID()] = &containerdSandbox{p: cdp, task: task, cnt: cnt, containerID: cnt.ID()}
	}

	return containers, nil
}

func (cdp *containerdPlatform) getAllIPs() (map[string]bool, error) {
	ctx := namespaces.WithNamespace(context.Background(), protosNamespace)

	ips := map[string]bool{}

	cnts, err := cdp.client.Containers(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to retrieve IPs")
	}

	for _, cnt := range cnts {
		task, err := cnt.Task(ctx, nil)
		if err != nil {
			continue
		}
		netNSPath := fmt.Sprintf("/proc/%d/ns/net", task.Pid())
		ip, err := getNetNSInterfaceIP(netNSPath, cdp.network)
		if err != nil {
			log.Errorf("Failed to retrieve IP for cnt '%s': %s")
		}
		if ip != nil {
			ips[ip.String()] = true
		}
	}

	return ips, nil
}

func (cdp *containerdPlatform) GetHWStats() (HardwareStats, error) {
	return getHWStatus()
}

func (cdp *containerdPlatform) PullImage(id string, name string, version string) error {
	ctx := namespaces.WithNamespace(context.Background(), protosNamespace)

	repoImage := cdp.appStoreHost + "/" + id
	_, err := cdp.client.Pull(ctx, repoImage, containerd.WithPullUnpack)
	if err != nil {
		return errors.Wrapf(err, "Failed to pull image '%s' from app store", id)
	}
	return nil
}

func (cdp *containerdPlatform) RemoveImage(id string) error {
	return nil
}

func (cdp *containerdPlatform) GetOrCreateVolume(id string, path string) (string, error) {
	return "", nil
}

func (cdp *containerdPlatform) RemoveVolume(id string) error {
	return nil
}

//
// struct and methods that satisfy PlatformRuntimeUnit
//

// containerdSandbox represents a container
type containerdSandbox struct {
	p    *containerdPlatform
	task containerd.Task
	cnt  containerd.Container

	containerID string
	IP          string
}

// Update reads the container and updates the struct fields
func (cnt *containerdSandbox) Update() error {
	return nil
}

// Start starts a containerd sandbox
func (cnt *containerdSandbox) Start() error {
	ctx := namespaces.WithNamespace(context.Background(), protosNamespace)
	if err := cnt.task.Start(ctx); err != nil {
		return errors.Wrapf(err, "Failed to start sandbox '%s'", cnt.containerID)
	}
	return nil
}

// Stop stops a containerd sandbox
func (cnt *containerdSandbox) Stop() error {
	ctx := namespaces.WithNamespace(context.Background(), protosNamespace)
	if err := cnt.task.Kill(ctx, syscall.SIGTERM); err != nil {
		return err
	}

	exitStatusC, err := cnt.task.Wait(ctx)
	if err != nil {
		return err
	}

	status := <-exitStatusC
	code, _, err := status.Result()
	if err != nil {
		return errors.Wrapf(err, "Failed to stop sandbox '%s'", cnt.containerID)
	}
	if code != 0 {
		log.Warnf("App '%s' exited with code '%d'", cnt.containerID, code)
	}

	_, err = cnt.task.Delete(ctx)
	if err != nil {
		return errors.Wrapf(err, "Error while stopping sandbox '%s'", cnt.containerID)
	}

	err = cnt.cnt.Delete(ctx)
	if err != nil {
		return errors.Wrapf(err, "Error while stopping sandbox '%s'", cnt.containerID)
	}

	err = os.Remove(fmt.Sprintf("%s/%s.log", cnt.p.logsPath, cnt.containerID))
	if err != nil {
		return errors.Wrapf(err, "Error while stopping sandbox '%s'", cnt.containerID)
	}

	return nil
}

// Remove removes a containerd sandbox
func (cnt *containerdSandbox) Remove() error {
	return nil
}

// GetID returns the ID of the container, as a string
func (cnt *containerdSandbox) GetID() string {
	return cnt.containerID
}

// GetIP returns the IP of the container, as a string
func (cnt *containerdSandbox) GetIP() string {
	return cnt.IP
}

// GetStatus returns the status of the container, as a string
func (cnt *containerdSandbox) GetStatus() string {
	ctx := namespaces.WithNamespace(context.Background(), protosNamespace)
	status, err := cnt.task.Status(ctx)
	if err != nil {
		return "UNKNOWN"
	}

	return string(status.Status)
}

// GetExitCode returns the exit code of the container, as an int
func (cnt *containerdSandbox) GetExitCode() int {
	return 0
}
