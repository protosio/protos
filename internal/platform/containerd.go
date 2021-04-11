package platform

import (
	"context"
	"fmt"
	"net"
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

type containerdPlatform struct {
	endpoint          string
	appStoreHost      string
	dnsServer         string
	internalInterface string
	initSignal        chan net.IP
	key               wgtypes.Key
	client            *containerd.Client
}

func createContainerdRuntimePlatform(runtimeUnixSocket string, appStoreHost string, inContainer bool, key wgtypes.Key) *containerdPlatform {
	return &containerdPlatform{
		endpoint:     runtimeUnixSocket,
		appStoreHost: appStoreHost,
		initSignal:   make(chan net.IP, 1),
		key:          key,
	}
}

func (cdp *containerdPlatform) Init(network net.IPNet, devices []auth.UserDevice) error {
	internalInterface, err := initNetwork(network, devices, cdp.key)
	if err != nil {
		return fmt.Errorf("Can't initialize network: %s", err.Error())
	}
	cdp.internalInterface = internalInterface

	log.Infof("Connecting to the containerd daemon using endpoint '%s'", cdp.endpoint)
	cdp.client, err = containerd.New(cdp.endpoint)
	if err != nil {
		return errors.Wrap(err, "Failed to initialize containerd runtime. Failed to connect, make sure you are running as root and the runtime has been started")
	}

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

	log.Debugf("Creating containerd sandbox '%s' from image '%s'", name, imageID)
	cnt, err := cdp.client.NewContainer(
		ctx,
		appID,
		containerd.WithNewSnapshot(imageID+"-snapshot", image),
		containerd.WithNewSpec(oci.WithImageConfig(image)),
		containerd.WithContainerLabels(map[string]string{"platform": protosNamespace, "appID": appID, "appName": name}),
	)
	if err != nil {
		return pru, errors.Wrapf(err, "Failed to create sandbox '%s' for app '%s'", name, appID)
	}
	pru.containerID = appID
	pru.task, err = cnt.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		return pru, errors.Wrapf(err, "Failed to create sandbox '%s' for app '%s'", name, appID)
	}

	return pru, nil
}

func (cdp *containerdPlatform) GetImage(id string) (PlatformImage, error) {
	ctx := namespaces.WithNamespace(context.Background(), protosNamespace)

	repoImage := cdp.appStoreHost + "/" + id
	image, err := cdp.client.GetImage(ctx, repoImage)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not retrieve image '%s' from containerd", id)
	}

	_, normalizedID, err := normalizeRepoDigest([]string{id})
	if err != nil {
		return nil, errors.Wrapf(err, "Could not retrieve image '%s' from containerd", id)
	}

	pi := &platformImage{
		id:      id,
		localID: normalizedID,
		// repoTags: imageResponse.Metadata().Labels,
		labels: image.Labels(),
	}

	return pi, nil
}

func (cdp *containerdPlatform) CleanUpSandbox(id string) error {
	// remove logs
	return nil
}

func (cdp *containerdPlatform) GetAllImages() (map[string]PlatformImage, error) {
	ctx := namespaces.WithNamespace(context.Background(), protosNamespace)
	images := map[string]PlatformImage{}

	listImagesResponse, err := cdp.client.ListImages(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Could not retrieve images from containerd")
	}

	for _, img := range listImagesResponse {
		fmt.Println(img)
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
		return nil, util.NewTypedError("containerd sandbox not found", ErrContainerNotFound)
	}

	cnts, err := cdp.client.Containers(ctx, id)
	if err != nil {
		return nil, util.NewTypedError("containerd sandbox not found", ErrContainerNotFound)
	}
	if len(cnts) == 0 {
		return nil, util.NewTypedError("containerd sandbox not found", ErrContainerNotFound)
	}

	cnt := cnts[0]
	task, err := cnt.Task(ctx, nil)
	if err != nil {
		return nil, util.NewTypedError("containerd sandbox not found", ErrContainerNotFound)
	}

	return &containerdSandbox{p: cdp, task: task}, nil
}

func (cdp *containerdPlatform) GetAllSandboxes() (map[string]PlatformRuntimeUnit, error) {
	return map[string]PlatformRuntimeUnit{}, nil
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
	p *containerdPlatform

	containerID     string
	IP              string
	containerStatus string
	exitCode        int
	task            containerd.Task
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
	code, exitedAt, err := status.Result()
	if err != nil {
		return errors.Wrapf(err, "Failed to stop sandbox '%s'", cnt.containerID)
	}
	fmt.Println(code, " ", exitedAt)
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
