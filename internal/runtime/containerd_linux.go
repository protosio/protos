package runtime

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/containerd/containerd/platforms"
	"github.com/dennwc/btrfs"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/network"
)

const (
	protosNamespace string = "protos"
)

var pltfrm *containerdPlatform

type containerdPlatform struct {
	endpoint       string
	logsPath       string
	volumesPath    string
	initSignal     chan net.IP
	networkManager *network.Manager
	client         *containerd.Client
	initLock       *sync.RWMutex
}

func createContainerdRuntimePlatform(networkManager *network.Manager, runtimeUnixSocket string) *containerdPlatform {

	cfg := config.Get()

	if pltfrm == nil {
		pltfrm = &containerdPlatform{
			endpoint:       runtimeUnixSocket,
			logsPath:       cfg.WorkDir + "/logs",
			volumesPath:    cfg.WorkDir + "/volumes",
			initSignal:     make(chan net.IP, 1),
			initLock:       &sync.RWMutex{},
			networkManager: networkManager,
		}
	}

	return pltfrm
}

func (cdp *containerdPlatform) Init() error {

	if _, err := os.Stat(cdp.logsPath); os.IsNotExist(err) {
		err := os.Mkdir(cdp.logsPath, os.ModeDir)
		if err != nil {
			return fmt.Errorf("failed to initialize platform. Failed to create logs directory: %s", err.Error())
		}
	}

	if _, err := os.Stat(cdp.volumesPath); os.IsNotExist(err) {
		err := os.Mkdir(cdp.volumesPath, os.ModeDir)
		if err != nil {
			return fmt.Errorf("failed to initialize platform. Failed to create volumes directory: %s", err.Error())
		}
	}

	var err error

	cdp.initLock.Lock()

	if cdp.client == nil {
		log.Infof("Connecting to the containerd daemon using endpoint '%s'", cdp.endpoint)
		cdp.client, err = containerd.New(cdp.endpoint)
		if err != nil {
			return errors.Wrap(err, "Failed to initialize containerd runtime. Failed to connect, make sure you are running as root and the runtime has been started")
		}
	}

	cdp.initLock.Unlock()
	return nil
}

func (cdp *containerdPlatform) NewSandbox(name string, appID string, imageRef string, persistence bool, installerParams map[string]string) (RuntimeSandbox, error) {
	ctx := namespaces.WithNamespace(context.Background(), protosNamespace)
	pru := &containerdSandbox{p: cdp, containerID: appID}

	image, err := cdp.client.GetImage(ctx, imageRef)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve image '%s' from containerd: %w", imageRef, err)
	}

	opts := []oci.SpecOpts{
		oci.WithImageConfig(image),
		oci.WithEnv([]string{fmt.Sprintf("APPID=%s", appID)}),
	}

	if persistence {
		err = cdp.getOrCreateVolume(appID)
		if err != nil {
			return nil, fmt.Errorf("failed to create volume for sandbox '%s': %w", appID, err)
		}

		volumePath := cdp.volumesPath + "/" + appID
		mounts := []specs.Mount{{
			Type:        "none",
			Destination: "/data",
			Source:      volumePath,
			Options:     []string{"rbind"},
		}}

		opts = append(opts, oci.WithMounts(mounts))
	}

	log.Debugf("Creating containerd sandbox '%s' from image '%s'", name, imageRef)
	cnt, err := cdp.client.NewContainer(
		ctx,
		appID,
		containerd.WithImage(image),
		containerd.WithImageStopSignal(image, "SIGTERM"),
		containerd.WithSnapshotter("btrfs"),
		containerd.WithNewSnapshot(appID, image),
		containerd.WithNewSpec(opts...),
		containerd.WithContainerLabels(map[string]string{"platform": protosNamespace, "appID": appID, "appName": name}),
	)
	if err != nil {
		return pru, fmt.Errorf("failed to create container '%s' for app '%s': %w", name, appID, err)
	}

	pru.cnt = cnt
	log.Debugf("Created sandbox for app '%s'", appID)
	return pru, nil
}

func (cdp *containerdPlatform) GetImage(imageRef string) (PlatformImage, error) {
	ctx := namespaces.WithNamespace(context.Background(), protosNamespace)

	image, err := cdp.client.GetImage(ctx, imageRef)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve image '%s' from containerd: %w", imageRef, err)
	}

	cs := cdp.client.ContentStore()
	architectures, err := images.Platforms(ctx, cs, image.Target())
	if err != nil {
		return nil, fmt.Errorf("could not retrieve supported platforms for image '%s': %w", imageRef, err)
	}
	archFound := false
	for _, architecture := range architectures {
		if architecture.Architecture == runtime.GOARCH {
			archFound = true
			break
		}
	}
	if !archFound {
		return nil, fmt.Errorf("image '%s' with arch '%s' not found", imageRef, runtime.GOARCH)
	}

	pi := &platformImage{
		id:     imageRef,
		labels: image.Labels(),
	}

	return pi, nil
}

func (cdp *containerdPlatform) ImageExistsLocally(id string) (bool, error) {
	_, err := cdp.GetImage(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check local image for installer %s: %w", id, err)
	}

	return true, nil
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

func (cdp *containerdPlatform) GetSandbox(id string) (RuntimeSandbox, error) {
	ctx := namespaces.WithNamespace(context.Background(), protosNamespace)
	if id == "" {
		return nil, ErrSandboxNotFound
	}

	cnt, err := cdp.client.LoadContainer(ctx, id)
	if err != nil {
		return nil, ErrSandboxNotFound
	}

	return &containerdSandbox{p: cdp, cnt: cnt, containerID: id}, nil
}

func (cdp *containerdPlatform) GetAllSandboxes() (map[string]RuntimeSandbox, error) {
	ctx := namespaces.WithNamespace(context.Background(), protosNamespace)

	containers := map[string]RuntimeSandbox{}

	cnts, err := cdp.client.Containers(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to retrieve containers")
	}

	for _, cnt := range cnts {
		containers[cnt.ID()] = &containerdSandbox{p: cdp, cnt: cnt, containerID: cnt.ID()}
	}

	return containers, nil
}

func (cdp *containerdPlatform) GetHWStats() (HardwareStats, error) {
	return getHWStatus()
}

func (cdp *containerdPlatform) PullImage(imageRef string) error {
	ctx := namespaces.WithNamespace(context.Background(), protosNamespace)

	image, err := cdp.client.Pull(ctx, imageRef, containerd.WithPullUnpack, containerd.WithPlatform(platforms.DefaultString()), containerd.WithPullSnapshotter("btrfs"))
	if err != nil {
		return fmt.Errorf("failed to pull image '%s' from app store: %w", imageRef, err)
	}

	cs := cdp.client.ContentStore()
	architectures, err := images.Platforms(ctx, cs, image.Target())
	if err != nil {
		return fmt.Errorf("could not retrieve supported platforms for image '%s': %w", imageRef, err)
	}

	archFound := false
	for _, architecture := range architectures {
		if architecture.Architecture == runtime.GOARCH {
			archFound = true
			break
		}
	}
	if !archFound {
		return fmt.Errorf("could not find '%s' arch for image '%s'", runtime.GOARCH, imageRef)
	}

	return nil
}

func (cdp *containerdPlatform) RemoveImage(id string) error {
	return nil
}

//
// Volumes methods
//

func (cdp *containerdPlatform) getOrCreateVolume(id string) error {
	volumePath := cdp.volumesPath + "/" + id

	info, err := os.Stat(volumePath)
	if err == nil {
		if info.IsDir() {
			isSubVolume, err := btrfs.IsSubVolume(volumePath)
			if err != nil {
				return fmt.Errorf("could not check data volume for sandbox '%s': %w", id, err)
			}
			if isSubVolume {
				return nil
			} else {
				return fmt.Errorf("can't create volume for sandbox '%s'(%s): directory exists but is not a btrfs subvolume", id, volumePath)
			}
		} else {
			return fmt.Errorf("can't create volume for sandbox '%s'(%s): path exists but is not a directory", id, volumePath)
		}
	}

	if os.IsNotExist(err) {
		err = btrfs.CreateSubVolume(volumePath)
		if err != nil {
			return fmt.Errorf("could not create data volume for sandbox '%s'(%s): %w", id, volumePath, err)
		}
	} else {
		return fmt.Errorf("error while creating data volume for sandbox '%s'(%s): %w", id, volumePath, err)
	}

	return nil
}

func (cdp *containerdPlatform) volumeExists(id string) bool {
	volumePath := cdp.volumesPath + "/" + id
	info, err := os.Stat(volumePath)
	if err != nil {
		return false
	}

	if !info.IsDir() {
		return false
	}

	isSubVolume, err := btrfs.IsSubVolume(volumePath)
	if err != nil {
		return false
	}
	if !isSubVolume {
		return false
	}

	return true
}

func (cdp *containerdPlatform) removeVolume(id string) error {
	volumePath := cdp.volumesPath + "/" + id
	err := btrfs.DeleteSubVolume(volumePath)
	if err != nil {
		return fmt.Errorf("could not delete data volume for sandbox '%s'(%s): %w", id, volumePath, err)
	}
	return nil
}

func (cdp *containerdPlatform) createVolumeSnapshot(sourceVolumeID string, name string) error {
	volumePath := cdp.volumesPath + "/" + sourceVolumeID
	snapshotPath := cdp.volumesPath + "/" + name
	err := btrfs.SnapshotSubVolume(volumePath, snapshotPath, false)
	if err != nil {
		return fmt.Errorf("could not create snapshot for volume '%s'(%s): %w", sourceVolumeID, volumePath, err)
	}
	return nil
}

//
// struct and methods that satisfy RuntimeSandbox
//

// containerdSandbox represents a container
type containerdSandbox struct {
	p   *containerdPlatform
	cnt containerd.Container

	containerID string
}

// Update reads the container and updates the struct fields
func (cnt *containerdSandbox) Update() error {
	return nil
}

// Start starts a containerd sandbox
func (cnt *containerdSandbox) Start(ip net.IP) error {
	ctx := namespaces.WithNamespace(context.Background(), protosNamespace)

	var task containerd.Task
	var err error
	task, err = cnt.cnt.Task(ctx, nil)
	if err != nil {
		if !errors.Is(err, errdefs.ErrNotFound) {
			return fmt.Errorf("failed to start sandbox '%s': %w", cnt.containerID, err)
		}
	} else {
		status, err := task.Status(ctx)
		if err != nil {
			return fmt.Errorf("failed to start sandbox '%s': %w", cnt.containerID, err)
		}

		if status.Status == containerd.Running {
			return nil
		} else if status.Status == containerd.Stopped || status.Status == containerd.Created {
			err = cnt.Stop()
			if err != nil {
				return fmt.Errorf("failed to start sandbox '%s': %w", cnt.containerID, err)
			}
		} else {
			return fmt.Errorf("failed to start sandbox '%s': task in invalid state '%s'", cnt.containerID, status.Status)
		}

	}

	logFilePath := fmt.Sprintf("%s/%s.log", cnt.p.logsPath, cnt.containerID)
	task, err = cnt.cnt.NewTask(ctx, cio.LogFile(logFilePath))
	if err != nil {
		return fmt.Errorf("failed to create task for app '%s': %w", cnt.containerID, err)
	}

	netNSpath := fmt.Sprintf("/proc/%d/ns/net", task.Pid())
	err = cnt.p.networkManager.CreateNamespacedInterface(netNSpath, ip)
	if err != nil {
		return fmt.Errorf("failed to create task for app '%s': %w", cnt.containerID, err)
	}

	if err := task.Start(ctx); err != nil {
		return fmt.Errorf("failed to start sandbox '%s': %w", cnt.containerID, err)
	}

	task.IO()

	return nil
}

// Stop stops a containerd sandbox
func (cnt *containerdSandbox) Stop() error {
	ctx := namespaces.WithNamespace(context.Background(), protosNamespace)

	task, err := cnt.cnt.Task(ctx, nil)
	if err != nil {
		if errors.Is(err, errdefs.ErrNotFound) {
			// if container has no task it means it's stopped
			return nil

		}
		return fmt.Errorf("failed to stop sandbox '%s': %w", cnt.containerID, err)
	}

	status, err := task.Status(ctx)
	if err != nil {
		return fmt.Errorf("failed to stop sandbox '%s': %w", cnt.containerID, err)
	}

	if status.Status != containerd.Stopped {
		if err := task.Kill(ctx, syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to stop sandbox '%s': %w", cnt.containerID, err)
		}

		exitStatusC, err := task.Wait(ctx)
		if err != nil {
			return fmt.Errorf("failed to stop sandbox '%s': %w", cnt.containerID, err)
		}

		status := <-exitStatusC
		code, _, err := status.Result()
		if err != nil {
			return fmt.Errorf("failed to stop sandbox '%s': %w", cnt.containerID, err)
		}
		if code != 0 {
			log.Warnf("App '%s' exited with code '%d'", cnt.containerID, code)
		}
	}

	_, err = task.Delete(ctx)
	if err != nil {
		return fmt.Errorf("error while stopping sandbox '%s': %w", cnt.containerID, err)
	}

	return nil
}

// Remove removes a containerd sandbox
func (cnt *containerdSandbox) Remove() error {
	ctx := namespaces.WithNamespace(context.Background(), protosNamespace)

	err := cnt.Stop()
	if err != nil {
		return fmt.Errorf("error while removing sandbox '%s': %w", cnt.containerID, err)
	}

	err = cnt.cnt.Delete(ctx)
	if err != nil {
		return fmt.Errorf("error while removing sandbox '%s': %w", cnt.containerID, err)
	}

	if cnt.p.volumeExists(cnt.containerID) {
		err = cnt.p.removeVolume(cnt.containerID)
		if err != nil {
			return fmt.Errorf("error while removing sandbox '%s': %w", cnt.containerID, err)
		}
	}

	err = os.Remove(fmt.Sprintf("%s/%s.log", cnt.p.logsPath, cnt.containerID))
	if err != nil {
		log.Warn("Failed to delete log file for sandbox '%s': %w", cnt.containerID, err)
	}

	return nil
}

// GetID returns the ID of the container, as a string
func (cnt *containerdSandbox) GetID() string {
	return cnt.containerID
}

// GetStatus returns the status of the container, as a string
func (cnt *containerdSandbox) GetStatus() string {
	ctx := namespaces.WithNamespace(context.Background(), protosNamespace)

	task, err := cnt.cnt.Task(ctx, nil)
	if err != nil {
		if errors.Is(err, errdefs.ErrNotFound) {
			// if container has no task it means it's stopped
			return string(containerd.Stopped)

		}
		return "UNKNOWN"
	}

	status, err := task.Status(ctx)
	if err != nil {
		return "UNKNOWN"
	}

	return string(status.Status)
}

// GetLogs returns the logs of the container
func (cnt *containerdSandbox) GetLogs() ([]byte, error) {
	logFilePath := fmt.Sprintf("%s/%s.log", cnt.p.logsPath, cnt.containerID)
	logs, err := os.ReadFile(logFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read logs for sandbox '%s': %w", cnt.containerID, err)
	}
	return logs, nil
}

// GetExitCode returns the exit code of the container, as an int
func (cnt *containerdSandbox) GetExitCode() int {
	return 0
}
