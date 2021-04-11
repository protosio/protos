package platform

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"net"
// 	"net/url"
// 	"os"
// 	"time"

// 	"github.com/pkg/errors"
// 	"github.com/protosio/protos/internal/auth"
// 	"github.com/protosio/protos/internal/util"
// 	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
// 	"google.golang.org/grpc"
// 	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
// )

// const (
// 	unixProtocol                     = "unix"
// 	defaultGRPCTimeout               = 5 * time.Second
// 	defaultSandboxTerminationTimeout = 5
// 	logDirectory                     = "/var/protos-containerd/applogs"
// )

// type imageInfo struct {
// 	ImageSpec struct {
// 		Config struct {
// 			Labels map[string]string `json:"Labels"`
// 		} `json:"config"`
// 	} `json:"imageSpec"`
// }

// func dial(addr string, timeout time.Duration) (net.Conn, error) {
// 	return net.DialTimeout(unixProtocol, addr, timeout)
// }

// func parseEndpointWithFallbackProtocol(endpoint string, fallbackProtocol string) (protocol string, addr string, err error) {
// 	if protocol, addr, err = parseEndpoint(endpoint); err != nil && protocol == "" {
// 		fallbackEndpoint := fallbackProtocol + "://" + endpoint
// 		protocol, addr, err = parseEndpoint(fallbackEndpoint)
// 		if err == nil {
// 			log.Warningf("Using %q as endpoint is deprecated, please consider using full url format %q.", endpoint, fallbackEndpoint)
// 		}
// 	}
// 	return
// }

// func parseEndpoint(endpoint string) (string, string, error) {
// 	u, err := url.Parse(endpoint)
// 	if err != nil {
// 		return "", "", err
// 	}

// 	switch u.Scheme {
// 	case "tcp":
// 		return "tcp", u.Host, nil

// 	case "unix":
// 		return "unix", u.Path, nil

// 	case "":
// 		return "", "", fmt.Errorf("using %q as endpoint is deprecated, please consider using full url format", endpoint)

// 	default:
// 		return u.Scheme, "", fmt.Errorf("protocol %q not supported", u.Scheme)
// 	}
// }

// func convertPort(port util.Port) *pb.PortMapping {
// 	newPort := &pb.PortMapping{}
// 	switch port.Type {
// 	case util.TCP:
// 		newPort.Protocol = 0
// 	case util.UDP:
// 		newPort.Protocol = 1
// 	case util.SCTP:
// 		newPort.Protocol = 2
// 	}
// 	newPort.ContainerPort = int32(port.Nr)
// 	newPort.HostPort = int32(port.Nr)

// 	return newPort
// }

// type criPlatform struct {
// 	endpoint          string
// 	appStoreHost      string
// 	inContainer       bool
// 	runtimeClient     pb.RuntimeServiceClient
// 	imageClient       pb.ImageServiceClient
// 	dnsServer         string
// 	internalInterface string
// 	initSignal        chan net.IP
// 	key               wgtypes.Key
// 	conn              *grpc.ClientConn
// }

// func createCRIRuntimePlatform(runtimeUnixSocket string, appStoreHost string, inContainer bool, key wgtypes.Key) *criPlatform {
// 	return &criPlatform{
// 		endpoint:     runtimeUnixSocket,
// 		appStoreHost: appStoreHost,
// 		inContainer:  inContainer,
// 		initSignal:   make(chan net.IP, 1),
// 		key:          key,
// 	}
// }

// // func (cdp *criPlatform) initCni() error {
// // 	podConfig := &pb.PodSandboxConfig{
// // 		Hostname: "init",
// // 		Metadata: &pb.PodSandboxMetadata{
// // 			Name:      "init",
// // 			Namespace: "default",
// // 			Attempt:   1,
// // 		},
// // 		Linux:        &pb.LinuxPodSandboxConfig{},
// // 		LogDirectory: logDirectory,
// // 	}
// // 	pod, err := cdp.runtimeClient.RunPodSandbox(context.Background(), &pb.RunPodSandboxRequest{Config: podConfig})
// // 	if err != nil {
// // 		return errors.Wrapf(err, "Failed to create init pod")
// // 	}

// // 	_, err = cdp.runtimeClient.StopPodSandbox(context.Background(), &pb.StopPodSandboxRequest{PodSandboxId: pod.PodSandboxId})
// // 	if err != nil {
// // 		return errors.Wrapf(err, "Failed to stop and remove init sandbox")
// // 	}
// // 	_, err = cdp.runtimeClient.RemovePodSandbox(context.Background(), &pb.RemovePodSandboxRequest{PodSandboxId: pod.PodSandboxId})
// // 	if err != nil {
// // 		return errors.Wrapf(err, "Failed to stop and remove init sandbox")
// // 	}
// // 	return nil
// // }

// func (cdp *criPlatform) Init(network net.IPNet, devices []auth.UserDevice) error {
// 	internalInterface, err := initNetwork(network, devices, cdp.key)
// 	if err != nil {
// 		return fmt.Errorf("Can't initialize network: %s", err.Error())
// 	}
// 	cdp.internalInterface = internalInterface

// 	log.Infof("Connecting to the cri daemon using endpoint '%s'", cdp.endpoint)
// 	protocol, addr, err := parseEndpointWithFallbackProtocol(cdp.endpoint, unixProtocol)
// 	if err != nil {
// 		return err
// 	}
// 	if protocol != unixProtocol {
// 		return errors.New("Failed to initialize containerd runtime. Unix socket is the only supported socket for the containerd endpoint")
// 	}

// 	cdp.conn, err = grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(defaultGRPCTimeout), grpc.WithDialer(dial))
// 	if err != nil {
// 		return errors.Wrap(err, "Failed to initialize containerd runtime. Failed to connect, make sure you are running as root and the runtime has been started")
// 	}
// 	cdp.runtimeClient = pb.NewRuntimeServiceClient(cdp.conn)
// 	cdp.imageClient = pb.NewImageServiceClient(cdp.conn)

// 	return nil
// }

// func (cdp *criPlatform) NewSandbox(name string, appID string, imageID string, volumeID string, volumeMountPath string, publicPorts []util.Port, installerParams map[string]string) (PlatformRuntimeUnit, error) {
// 	pru := &criSandbox{p: cdp}

// 	img, err := cdp.GetImage(imageID)
// 	if err != nil {
// 		return pru, errors.Wrapf(err, "Failed to create sandbox '%s' for app '%s'", name, appID)
// 	}
// 	if img == nil {
// 		return pru, errors.Errorf("Failed to create sandbox for app '%s'(%s): image '%s' not found locally", name, appID, imageID)
// 	}

// 	localImg := img.(*platformImage)

// 	log.Debugf("Creating containerd sandbox '%s' from image '%s'", name, imageID)

// 	podPorts := []*pb.PortMapping{}
// 	for _, port := range publicPorts {
// 		podPorts = append(podPorts, convertPort(port))
// 	}

// 	// create pod
// 	podConfig := &pb.PodSandboxConfig{
// 		Hostname: name,
// 		Metadata: &pb.PodSandboxMetadata{
// 			Name:      name + "-sandbox",
// 			Namespace: "protos",
// 			Attempt:   1,
// 		},
// 		Labels:       map[string]string{"platform": "protos", "appID": appID, "appName": name},
// 		PortMappings: podPorts,
// 		DnsConfig:    &pb.DNSConfig{Servers: []string{cdp.dnsServer}, Searches: []string{"protos.local"}},
// 		Linux:        &pb.LinuxPodSandboxConfig{},
// 		LogDirectory: logDirectory,
// 	}
// 	podResponse, err := cdp.runtimeClient.RunPodSandbox(context.Background(), &pb.RunPodSandboxRequest{Config: podConfig})
// 	if err != nil {
// 		return pru, errors.Wrapf(err, "Failed to create sandbox '%s' for app '%s'", name, appID)
// 	}
// 	pru.podID = podResponse.PodSandboxId
// 	podStatus, err := cdp.runtimeClient.PodSandboxStatus(context.Background(), &pb.PodSandboxStatusRequest{PodSandboxId: podResponse.PodSandboxId})
// 	if err != nil {
// 		return pru, errors.Wrapf(err, "Failed to create sandbox '%s' for app '%s'", name, appID)
// 	}
// 	pru.IP = podStatus.Status.Network.Ip

// 	// create app environment variables

// 	envvars := []*pb.KeyValue{{Key: "APPID", Value: appID}}
// 	for k, v := range installerParams {
// 		envvars = append(envvars, &pb.KeyValue{Key: k, Value: v})
// 	}

// 	logFile := pru.podID + ".log"
// 	logFilePath := logDirectory + "/" + logFile

// 	_, err = os.Stat(logFilePath)
// 	if os.IsNotExist(err) {
// 		file, err := os.Create(logFilePath)
// 		if err != nil {
// 			log.Fatal(err)
// 			return pru, errors.Wrapf(err, "Failed to create log file for sandbox '%s' for app '%s'", name, appID)
// 		}
// 		defer file.Close()
// 	}

// 	// create container in pod
// 	containerRequest := &pb.CreateContainerRequest{
// 		PodSandboxId: pru.podID,
// 		Config: &pb.ContainerConfig{
// 			Image:    &pb.ImageSpec{Image: localImg.localID},
// 			Metadata: &pb.ContainerMetadata{Name: name, Attempt: 1},
// 			LogPath:  logFile,
// 			Envs:     envvars,
// 		},
// 		SandboxConfig: podConfig,
// 	}
// 	containerResponse, err := cdp.runtimeClient.CreateContainer(context.Background(), containerRequest)
// 	if err != nil {
// 		err2 := pru.Remove()
// 		if err2 != nil {
// 			log.Warnf("Failed to clean up on containerd sandbox creation failure: %s", err2.Error())
// 		}
// 		return &criSandbox{p: cdp}, errors.Wrapf(err, "Failed to create sandbox '%s' for app '%s'", name, appID)
// 	}
// 	pru.containerID = containerResponse.ContainerId
// 	statusResponse, err := cdp.runtimeClient.ContainerStatus(context.Background(), &pb.ContainerStatusRequest{ContainerId: pru.containerID})
// 	if err != nil {
// 		return pru, errors.Wrapf(err, "Failed to create sandbox '%s' for app '%s'", name, appID)
// 	}
// 	pru.podStatus = statusResponse.Status.State.String()

// 	return pru, nil
// }

// func (cdp *criPlatform) GetSandbox(appID string) (PlatformRuntimeUnit, error) {
// 	if appID == "" {
// 		return nil, util.NewTypedError("containerd sandbox not found", ErrContainerNotFound)
// 	}
// 	pru := &criSandbox{p: cdp}

// 	listPodsResponse, err := cdp.runtimeClient.ListPodSandbox(context.Background(), &pb.ListPodSandboxRequest{Filter: &pb.PodSandboxFilter{LabelSelector: map[string]string{"platform": "protos", "appID": appID}}})
// 	if err != nil {
// 		return pru, util.ErrorContainsTransform(errors.Wrapf(err, "Error retrieving containerd sandbox for app %s", appID), "does not exist", ErrContainerNotFound)
// 	}

// 	if len(listPodsResponse.Items) == 0 {
// 		return pru, fmt.Errorf("Could not find sandbox '%s'", appID)
// 	}

// 	podID := listPodsResponse.Items[0].Id
// 	podStatus, err := cdp.runtimeClient.PodSandboxStatus(context.Background(), &pb.PodSandboxStatusRequest{PodSandboxId: podID})
// 	if err != nil {
// 		return pru, util.ErrorContainsTransform(errors.Wrapf(err, "Error retrieving containerd sandbox %s", podID), "does not exist", ErrContainerNotFound)
// 	}
// 	pru.podID = podStatus.Status.Id
// 	pru.podStatus = podStatus.Status.State.String()
// 	pru.IP = podStatus.Status.Network.Ip
// 	cntListResponse, err := cdp.runtimeClient.ListContainers(context.Background(), &pb.ListContainersRequest{Filter: &pb.ContainerFilter{PodSandboxId: podStatus.Status.Id}})
// 	if err != nil {
// 		return pru, util.ErrorContainsTransform(errors.Wrapf(err, "Error retrieving containerd sandbox %s", podID), "does not exist", ErrContainerNotFound)
// 	}
// 	nrContainers := len(cntListResponse.Containers)
// 	if nrContainers != 1 {
// 		return pru, errors.Wrapf(err, "Containerd sandbox %s, has '%d' containers instead of 1", podID, len(cntListResponse.Containers))
// 	}
// 	pru.containerID = cntListResponse.Containers[0].Id
// 	// get status and save exit code
// 	statusResponse, err := cdp.runtimeClient.ContainerStatus(context.Background(), &pb.ContainerStatusRequest{ContainerId: pru.containerID})
// 	if err != nil {
// 		return nil, errors.Wrapf(err, "Error retrieving containerd sandbox %s", podID)
// 	}
// 	pru.containerStatus = statusResponse.Status.State.String()
// 	pru.exitCode = int(statusResponse.Status.ExitCode)
// 	return pru, nil
// }

// func (cdp *criPlatform) GetAllSandboxes() (map[string]PlatformRuntimeUnit, error) {
// 	return map[string]PlatformRuntimeUnit{}, nil
// }

// func (cdp *criPlatform) GetImage(id string) (PlatformImage, error) {

// 	_, normalizedID, err := normalizeRepoDigest([]string{id})
// 	if err != nil {
// 		return nil, errors.Wrapf(err, "Could not retrieve image '%s' from containerd", id)
// 	}

// 	imagesResponse, err := cdp.imageClient.ListImages(context.Background(), &pb.ListImagesRequest{})
// 	if err != nil {
// 		return nil, errors.Wrapf(err, "Could not retrieve image '%s' from containerd", id)
// 	}

// 	for _, img := range imagesResponse.Images {
// 		imgName, imgDigest, err := normalizeRepoDigest(img.RepoDigests)
// 		if err != nil {
// 			log.Warnf("Image '%s'[%s] has invalid repo digest: %s", img.Id, imgName, err.Error())
// 			continue
// 		}
// 		if normalizedID == imgDigest {
// 			// retrieve detailed info
// 			imageResponse, err := cdp.imageClient.ImageStatus(context.Background(), &pb.ImageStatusRequest{Image: &pb.ImageSpec{Image: img.Id}, Verbose: true})
// 			if err != nil || imageResponse.Image == nil {
// 				return nil, errors.Wrapf(err, "Could not retrieve image '%s' from containerd", id)
// 			}
// 			// image not found
// 			if imageResponse.Image == nil {
// 				return nil, nil
// 			}
// 			var imageInfo imageInfo
// 			err = json.Unmarshal([]byte(imageResponse.Info["info"]), &imageInfo)
// 			if err != nil {
// 				return &platformImage{}, errors.Wrapf(err, "Could not retrieve image '%s' from containerd", id)
// 			}
// 			image := platformImage{
// 				id:       id,
// 				localID:  img.Id,
// 				repoTags: img.RepoTags,
// 				labels:   imageInfo.ImageSpec.Config.Labels,
// 			}
// 			return &image, nil
// 		}
// 	}

// 	return nil, nil
// }

// func (cdp *criPlatform) GetAllImages() (map[string]PlatformImage, error) {
// 	images := map[string]PlatformImage{}

// 	imagesResponse, err := cdp.imageClient.ListImages(context.Background(), &pb.ListImagesRequest{})
// 	if err != nil {
// 		return images, errors.Wrap(err, "Could not retrieve images from containerd")
// 	}

// 	for _, img := range imagesResponse.Images {
// 		imgName, imgDigest, err := normalizeRepoDigest(img.RepoDigests)
// 		if err != nil {
// 			log.Warnf("Image '%s'[%s] has invalid repo digest: %s", img.Id, imgName, err.Error())
// 			continue
// 		}

// 		image := platformImage{
// 			id:       imgDigest,
// 			localID:  img.Id,
// 			repoTags: img.RepoTags,
// 		}
// 		images[image.id] = &image
// 	}

// 	return images, nil
// }

// func (cdp *criPlatform) PullImage(id string, name string, version string) error {
// 	repoImage := cdp.appStoreHost + "/" + id
// 	piRequest := &pb.PullImageRequest{
// 		Image: &pb.ImageSpec{Image: repoImage},
// 	}
// 	_, err := cdp.imageClient.PullImage(context.Background(), piRequest)
// 	if err != nil {
// 		return errors.Wrapf(err, "Failed to pull image '%s' from app store", id)
// 	}
// 	log.Infof("Downloaded image '%s'", id)
// 	return nil
// }

// func (cdp *criPlatform) RemoveImage(id string) error {
// 	return nil
// }

// func (cdp *criPlatform) GetOrCreateVolume(id string, path string) (string, error) {
// 	return "", nil
// }

// func (cdp *criPlatform) RemoveVolume(id string) error {
// 	return nil
// }

// func (cdp *criPlatform) CleanUpSandbox(id string) error {
// 	// remove logs
// 	logFile := logDirectory + "/" + id + ".log"
// 	log.Info("Removing log file ", logFile)
// 	err := os.Remove(logFile)
// 	if err != nil {
// 		return errors.Wrapf(err, "Failed to remove log file for sandbox '%s'", id)
// 	}
// 	return nil
// }

// func (cdp *criPlatform) GetHWStats() (HardwareStats, error) {
// 	return getHWStatus()
// }

// //
// // struct and methods that satisfy PlatformRuntimeUnit
// //

// // criSandbox represents a container
// type criSandbox struct {
// 	p *criPlatform

// 	podID           string
// 	containerID     string
// 	IP              string
// 	podStatus       string
// 	containerStatus string
// 	exitCode        int
// }

// // Update reads the container and updates the struct fields
// func (cnt *criSandbox) Update() error {
// 	return nil
// }

// // Start starts a containerd sandbox
// func (cnt *criSandbox) Start() error {
// 	_, err := cnt.p.runtimeClient.StartContainer(context.Background(), &pb.StartContainerRequest{ContainerId: cnt.containerID})
// 	if err != nil {
// 		return errors.Wrapf(err, "Failed to start sandbox '%s'", cnt.podID)
// 	}
// 	return nil
// }

// // Stop stops a containerd sandbox
// func (cnt *criSandbox) Stop() error {
// 	// stop container with default period
// 	_, err := cnt.p.runtimeClient.StopContainer(context.Background(), &pb.StopContainerRequest{ContainerId: cnt.containerID, Timeout: defaultSandboxTerminationTimeout})
// 	if err != nil {
// 		return errors.Wrapf(err, "Failed to stop sandbox '%s'", cnt.podID)
// 	}
// 	// get status and save exit code
// 	statusResponse, err := cnt.p.runtimeClient.ContainerStatus(context.Background(), &pb.ContainerStatusRequest{ContainerId: cnt.containerID})
// 	if err != nil {
// 		return errors.Wrapf(err, "Failed to stop sandbox '%s'", cnt.podID)
// 	}
// 	cnt.containerStatus = statusResponse.Status.State.String()
// 	cnt.exitCode = int(statusResponse.Status.ExitCode)

// 	// remove container
// 	_, err = cnt.p.runtimeClient.RemoveContainer(context.Background(), &pb.RemoveContainerRequest{ContainerId: cnt.containerID})
// 	if err != nil {
// 		return errors.Wrapf(err, "Failed to stop sandbox '%s'", cnt.podID)
// 	}
// 	cnt.containerID = ""

// 	// remove pod
// 	_, err = cnt.p.runtimeClient.StopPodSandbox(context.Background(), &pb.StopPodSandboxRequest{PodSandboxId: cnt.podID})
// 	if err != nil {
// 		return errors.Wrapf(err, "Failed to stop sandbox '%s'", cnt.podID)
// 	}
// 	_, err = cnt.p.runtimeClient.RemovePodSandbox(context.Background(), &pb.RemovePodSandboxRequest{PodSandboxId: cnt.podID})
// 	if err != nil {
// 		return errors.Wrapf(err, "Failed to stop sandbox '%s'", cnt.podID)
// 	}
// 	cnt.podID = ""
// 	return nil
// }

// // Remove removes a containerd sandbox
// func (cnt *criSandbox) Remove() error {
// 	// retrieve all containers for pod
// 	listResponse, err := cnt.p.runtimeClient.ListContainers(context.Background(), &pb.ListContainersRequest{Filter: &pb.ContainerFilter{PodSandboxId: cnt.podID}})
// 	if err != nil {
// 		return errors.Wrapf(err, "Failed to remove sandbox '%s'", cnt.podID)
// 	}
// 	// gracefully stop and remove containers for pod
// 	for _, pcnt := range listResponse.Containers {
// 		_, err := cnt.p.runtimeClient.StopContainer(context.Background(), &pb.StopContainerRequest{ContainerId: pcnt.Id, Timeout: defaultSandboxTerminationTimeout})
// 		if err != nil {
// 			log.Warnf("Failed to stop container '%s' for pod '%s': %s", pcnt.Id, cnt.podID, err.Error())
// 		}
// 		_, err = cnt.p.runtimeClient.RemoveContainer(context.Background(), &pb.RemoveContainerRequest{ContainerId: pcnt.Id})
// 		if err != nil {
// 			log.Warnf("Failed to remove container '%s' for pod '%s': %s", pcnt.Id, cnt.podID, err.Error())
// 		}
// 	}
// 	_, err = cnt.p.runtimeClient.StopPodSandbox(context.Background(), &pb.StopPodSandboxRequest{PodSandboxId: cnt.podID})
// 	if err != nil {
// 		return errors.Wrapf(err, "Failed to remove sandbox '%s'", cnt.podID)
// 	}
// 	_, err = cnt.p.runtimeClient.RemovePodSandbox(context.Background(), &pb.RemovePodSandboxRequest{PodSandboxId: cnt.podID})
// 	if err != nil {
// 		return errors.Wrapf(err, "Failed to remove sandbox '%s'", cnt.podID)
// 	}

// 	return nil
// }

// // GetID returns the ID of the container, as a string
// func (cnt *criSandbox) GetID() string {
// 	return cnt.podID
// }

// // GetIP returns the IP of the container, as a string
// func (cnt *criSandbox) GetIP() string {
// 	return cnt.IP
// }

// // GetStatus returns the status of the container, as a string
// func (cnt *criSandbox) GetStatus() string {
// 	if cnt.containerID != "" {
// 		statusResponse, err := cnt.p.runtimeClient.ContainerStatus(context.Background(), &pb.ContainerStatusRequest{ContainerId: cnt.containerID})
// 		if err != nil {
// 			log.Error(errors.Wrapf(err, "Failed to get status for container '%s'", cnt.containerID))
// 			return statusUnknown
// 		}
// 		cnt.containerStatus = statusResponse.Status.State.String()
// 		cnt.exitCode = int(statusResponse.Status.ExitCode)
// 		return containerdToAppStatus(cnt.containerStatus, cnt.exitCode)
// 	}
// 	return containerdToAppStatus(cnt.containerStatus, cnt.exitCode)
// }

// // GetExitCode returns the exit code of the container, as an int
// func (cnt *criSandbox) GetExitCode() int {
// 	statusResponse, err := cnt.p.runtimeClient.ContainerStatus(context.Background(), &pb.ContainerStatusRequest{ContainerId: cnt.containerID})
// 	if err != nil {
// 		log.Error(errors.Wrapf(err, "Failed to get status for container '%s'", cnt.containerID))
// 		return 255
// 	}
// 	cnt.containerStatus = statusResponse.Status.State.String()
// 	cnt.exitCode = int(statusResponse.Status.ExitCode)
// 	return cnt.exitCode
// }

// //
// // helper methods
// //

// func containerdToAppStatus(status string, exitCode int) string {
// 	switch status {
// 	case "CONTAINER_CREATED":
// 		return statusStopped
// 	case "CONTAINER_EXITED":
// 		if exitCode == 0 {
// 			return statusStopped
// 		}
// 		return statusFailed
// 	case "CONTAINER_RUNNING":
// 		return statusRunning
// 	case "CONTAINER_UNKNOWN":
// 		return statusUnknown
// 	default:
// 		return statusUnknown
// 	}
// }
