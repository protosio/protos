package platform

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	guuid "github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/core"
	"github.com/protosio/protos/internal/util"
	"google.golang.org/grpc"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

const (
	unixProtocol                     = "unix"
	defaultGRPCTimeout               = 5 * time.Second
	defaultSandboxTerminationTimeout = 5
)

func dial(addr string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout(unixProtocol, addr, timeout)
}

func parseEndpointWithFallbackProtocol(endpoint string, fallbackProtocol string) (protocol string, addr string, err error) {
	if protocol, addr, err = parseEndpoint(endpoint); err != nil && protocol == "" {
		fallbackEndpoint := fallbackProtocol + "://" + endpoint
		protocol, addr, err = parseEndpoint(fallbackEndpoint)
		if err == nil {
			log.Warningf("Using %q as endpoint is deprecated, please consider using full url format %q.", endpoint, fallbackEndpoint)
		}
	}
	return
}

func parseEndpoint(endpoint string) (string, string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", "", err
	}

	switch u.Scheme {
	case "tcp":
		return "tcp", u.Host, nil

	case "unix":
		return "unix", u.Path, nil

	case "":
		return "", "", fmt.Errorf("using %q as endpoint is deprecated, please consider using full url format", endpoint)

	default:
		return u.Scheme, "", fmt.Errorf("protocol %q not supported", u.Scheme)
	}
}

type containerdPlatform struct {
	endpoint      string
	appStoreHost  string
	inContainer   bool
	runtimeClient pb.RuntimeServiceClient
	imageClient   pb.ImageServiceClient
}

func createContainerdRuntimePlatform(runtimeUnixSocket string, appStoreHost string, inContainer bool) *containerdPlatform {
	return &containerdPlatform{
		endpoint:     runtimeUnixSocket,
		appStoreHost: appStoreHost,
		inContainer:  inContainer,
	}
}

func (cdp *containerdPlatform) Init() (string, error) {
	log.Infof("Connecting to the containerd daemon using endpoint '%s'", cdp.endpoint)
	protocol, addr, err := parseEndpointWithFallbackProtocol(cdp.endpoint, unixProtocol)
	if err != nil {
		return "", err
	}
	if protocol != unixProtocol {
		return "", errors.New("unix socket is the only supported socket for the containerd endpoint")
	}

	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(defaultTimeout), grpc.WithDialer(dial))
	if err != nil {
		return "", errors.Wrap(err, "failed to connect, make sure you are running as root and the runtime has been started")
	}
	cdp.runtimeClient = pb.NewRuntimeServiceClient(conn)
	cdp.imageClient = pb.NewImageServiceClient(conn)

	return "", nil
}
func (cdp *containerdPlatform) GetSandbox(id string) (core.PlatformRuntimeUnit, error) {
	if id == "" {
		return nil, util.NewTypedError("containerd sandbox not found", core.ErrContainerNotFound)
	}
	pru := &containerdSandbox{p: cdp}
	podStatus, err := cdp.runtimeClient.PodSandboxStatus(context.Background(), &pb.PodSandboxStatusRequest{PodSandboxId: id})
	if err != nil {
		return pru, util.ErrorContainsTransform(errors.Wrapf(err, "Error retrieving containerd sandbox %s", id), "does not exist", core.ErrContainerNotFound)
	}
	pru.podID = podStatus.Status.Id
	pru.podStatus = podStatus.Status.State.String()
	pru.IP = podStatus.Status.Network.Ip
	return pru, nil
}

func (cdp *containerdPlatform) GetAllSandboxes() (map[string]core.PlatformRuntimeUnit, error) {
	return map[string]core.PlatformRuntimeUnit{}, nil
}

func (cdp *containerdPlatform) GetImage(id string) (core.PlatformImage, error) {
	imageResponse, err := cdp.imageClient.ImageStatus(context.Background(), &pb.ImageStatusRequest{Image: &pb.ImageSpec{Image: id}})
	if err != nil {
		return &platformImage{}, errors.Wrapf(err, "Could not retrieve image '%s' from containerd", id)
	}
	return &platformImage{id: imageResponse.Image.Id, repoTags: imageResponse.Image.RepoTags}, nil
}

func (cdp *containerdPlatform) GetAllImages() (map[string]core.PlatformImage, error) {
	images := map[string]core.PlatformImage{}

	imagesResponse, err := cdp.imageClient.ListImages(context.Background(), &pb.ListImagesRequest{})
	if err != nil {
		return images, errors.Wrap(err, "Could not retrieve images from containerd")
	}

	for _, img := range imagesResponse.Images {
		image := platformImage{
			id:       img.Id,
			repoTags: img.RepoTags,
		}
		images[image.id] = &image
	}

	return images, nil
}

func (cdp *containerdPlatform) PullImage(task core.Task, id string, name string, version string) error {
	piRequest := &pb.PullImageRequest{
		Image: &pb.ImageSpec{Image: id},
	}
	piResponse, err := cdp.imageClient.PullImage(context.Background(), piRequest)
	if err != nil {
		return err
	}
	log.Infof("Downloaded image '%s'", piResponse.GetImageRef())
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

func (cdp *containerdPlatform) NewSandbox(name string, appID string, imageID string, volumeID string, volumeMountPath string, publicPorts []util.Port, installerParams map[string]string) (core.PlatformRuntimeUnit, error) {
	pru := &containerdSandbox{p: cdp}

	log.Debugf("Creating containerd sandbox '%s' from image '%s'", name, imageID)

	// create pod
	podConfig := &pb.PodSandboxConfig{
		Hostname: name,
		Metadata: &pb.PodSandboxMetadata{
			Name:      name + "-sandbox",
			Namespace: "default",
			Uid:       guuid.New().String(),
			Attempt:   1,
		},
		Linux: &pb.LinuxPodSandboxConfig{},
	}
	podResponse, err := cdp.runtimeClient.RunPodSandbox(context.Background(), &pb.RunPodSandboxRequest{Config: podConfig})
	if err != nil {
		return pru, errors.Wrapf(err, "Failed to create sandbox '%s' for app '%s'", name, appID)
	}
	pru.podID = podResponse.PodSandboxId
	podStatus, err := cdp.runtimeClient.PodSandboxStatus(context.Background(), &pb.PodSandboxStatusRequest{PodSandboxId: podResponse.PodSandboxId})
	if err != nil {
		return pru, errors.Wrapf(err, "Failed to create sandbox '%s' for app '%s'", name, appID)
	}
	pru.IP = podStatus.Status.Network.Ip

	// create container in pod
	containerRequest := &pb.CreateContainerRequest{
		PodSandboxId: pru.podID,
		Config: &pb.ContainerConfig{
			Image:    &pb.ImageSpec{Image: imageID},
			Metadata: &pb.ContainerMetadata{Name: name, Attempt: 1}},
		SandboxConfig: podConfig,
	}
	containerResponse, err := cdp.runtimeClient.CreateContainer(context.Background(), containerRequest)
	if err != nil {
		return pru, errors.Wrapf(err, "Failed to create sandbox '%s' for app '%s'", name, appID)
	}
	pru.containerID = containerResponse.ContainerId
	statusResponse, err := cdp.runtimeClient.ContainerStatus(context.Background(), &pb.ContainerStatusRequest{ContainerId: pru.containerID})
	if err != nil {
		return pru, errors.Wrapf(err, "Failed to create sandbox '%s' for app '%s'", name, appID)
	}
	pru.podStatus = statusResponse.Status.State.String()

	return pru, nil
}

func (cdp *containerdPlatform) GetHWStats() (core.HardwareStats, error) {
	return getHWStatus()
}

// containerdSandbox represents a container
type containerdSandbox struct {
	p *containerdPlatform

	podID           string
	containerID     string
	IP              string
	podStatus       string
	containerStatus string
	exitCode        int
}

// Update reads the container and updates the struct fields
func (cnt *containerdSandbox) Update() error {
	return nil
}

// Start starts a containerd container
func (cnt *containerdSandbox) Start() error {
	return nil
}

// Stop stops a containerd container
func (cnt *containerdSandbox) Stop() error {
	return nil
}

// Remove removes a containerd container
func (cnt *containerdSandbox) Remove() error {
	// retrieve all containers for pod
	listResponse, err := cnt.p.runtimeClient.ListContainers(context.Background(), &pb.ListContainersRequest{Filter: &pb.ContainerFilter{PodSandboxId: cnt.podID}})
	if err != nil {
		return errors.Wrapf(err, "Failed to remove sandbox '%s'", cnt.podID)
	}
	// gracefully stop and remove containers for pod
	for _, pcnt := range listResponse.Containers {
		_, err := cnt.p.runtimeClient.StopContainer(context.Background(), &pb.StopContainerRequest{ContainerId: pcnt.Id, Timeout: defaultSandboxTerminationTimeout})
		if err != nil {
			log.Warnf("Failed to stop container '%s' for pod '%s': %s", pcnt.Id, cnt.podID, err.Error())
		}
		_, err = cnt.p.runtimeClient.RemoveContainer(context.Background(), &pb.RemoveContainerRequest{ContainerId: pcnt.Id})
		if err != nil {
			log.Warnf("Failed to remove container '%s' for pod '%s': %s", pcnt.Id, cnt.podID, err.Error())
		}
	}
	_, err = cnt.p.runtimeClient.StopPodSandbox(context.Background(), &pb.StopPodSandboxRequest{PodSandboxId: cnt.podID})
	if err != nil {
		return errors.Wrapf(err, "Failed to remove sandbox '%s'", cnt.podID)
	}
	_, err = cnt.p.runtimeClient.RemovePodSandbox(context.Background(), &pb.RemovePodSandboxRequest{PodSandboxId: cnt.podID})
	if err != nil {
		return errors.Wrapf(err, "Failed to remove sandbox '%s'", cnt.podID)
	}
	return nil
}

// GetID returns the ID of the container, as a string
func (cnt *containerdSandbox) GetID() string {
	return cnt.podID
}

// GetIP returns the IP of the container, as a string
func (cnt *containerdSandbox) GetIP() string {
	return cnt.IP
}

// GetStatus returns the status of the container, as a string
func (cnt *containerdSandbox) GetStatus() string {
	statusResponse, err := cnt.p.runtimeClient.ContainerStatus(context.Background(), &pb.ContainerStatusRequest{ContainerId: cnt.containerID})
	if err != nil {
		log.Error(errors.Wrapf(err, "Failed to get status for container '%s'", cnt.containerID))
		return statusUnknown
	}
	cnt.containerStatus = statusResponse.Status.State.String()
	cnt.exitCode = int(statusResponse.Status.ExitCode)
	return containerdToAppStatus(cnt.containerStatus, cnt.exitCode)
}

// GetExitCode returns the exit code of the container, as an int
func (cnt *containerdSandbox) GetExitCode() int {
	statusResponse, err := cnt.p.runtimeClient.ContainerStatus(context.Background(), &pb.ContainerStatusRequest{ContainerId: cnt.containerID})
	if err != nil {
		log.Error(errors.Wrapf(err, "Failed to get status for container '%s'", cnt.containerID))
		return 255
	}
	cnt.containerStatus = statusResponse.Status.State.String()
	cnt.exitCode = int(statusResponse.Status.ExitCode)
	return cnt.exitCode
}

//
// helper methods
//

func containerdToAppStatus(status string, exitCode int) string {
	switch status {
	case "CONTAINER_CREATED":
		return statusStopped
	case "CONTAINER_EXITED":
		if exitCode == 0 {
			return statusStopped
		}
		return statusFailed
	case "CONTAINER_RUNNING":
		return statusRunning
	case "CONTAINER_UNKNOWN":
		return statusUnknown
	default:
		return statusUnknown
	}
}
