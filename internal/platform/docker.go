package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/docker/distribution"

	"github.com/protosio/protos/internal/core"
	"github.com/protosio/protos/internal/task"
	"github.com/protosio/protos/internal/util"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	volumetypes "github.com/docker/docker/api/types/volume"
	docker "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/heroku/docker-registry-client/registry"
	"github.com/pkg/errors"
)

const (
	protosNetwork = "protosnet"
)

type downloadEvent struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	Error          string `json:"error"`
	Progress       string `json:"progress"`
	ProgressDetail struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"progressDetail"`
}

type imageLayer struct {
	id         string
	size       int64
	downloaded int64
	extracted  int64
}

type downloadProgress struct {
	layers            map[string]imageLayer
	t                 core.Task
	totalSize         int64
	percentage        int
	weight            int
	initialPercentage int
}

// DockerContainer represents a container
type DockerContainer struct {
	ID       string
	IP       string
	Status   string
	ExitCode int
	p        *dockerPlatform
}

type dockerPlatform struct {
	client *docker.Client
}

// ConnectDocker connects to the Docker daemon
func (dp *dockerPlatform) Connect() {
	log.Info("Connecting to the docker daemon")
	var err error

	dp.client, err = docker.NewEnvClient()
	if err != nil {
		log.Fatalf("Failed to connect to Docker daemon: '%s'", err.Error())
	}
}

// combineEnv takes a map of environment variables and transforms them into a list of environment variables
func combineEnv(params map[string]string) []string {
	var env []string
	for id, val := range params {
		env = append(env, id+"="+val)
	}
	return env
}

func (dp *dockerPlatform) allocateContainerIP() (string, error) {
	protosNet, err := dp.GetNetwork(protosNetwork)
	if err != nil {
		return "", errors.Wrap(err, "Failed to allocate IP for container")
	}

	if len(protosNet.IPAM.Config) == 0 {
		return "", fmt.Errorf("Failed to allocate IP for container: no network config for network %s(%s)", protosNet.Name, protosNet.ID)
	}

	_, protosSubnet, err := net.ParseCIDR(protosNet.IPAM.Config[0].Subnet)
	if err != nil {
		return "", errors.Wrap(err, "Failed to allocate IP for container")
	}

	gateway := net.ParseIP(protosNet.IPAM.Config[0].Gateway)
	allocatedIPs := []net.IP{}
	allocatedIPs = append(allocatedIPs, gateway)

	for _, ipConfg := range protosNet.Containers {
		cntIP, _, err := net.ParseCIDR(ipConfg.IPv4Address)
		if err != nil {
			return "", errors.Wrap(err, "Failed to allocate IP for container")
		}
		allocatedIPs = append(allocatedIPs, cntIP)
	}

	allIPs := util.AllNetworkIPs(*protosSubnet)
	for _, ip := range allIPs {
		if util.IPinList(ip, allocatedIPs) {
			continue
		}
		return ip.String(), nil
	}

	return "", fmt.Errorf("Failed to allocate IP for container: all IPs have been allocated")
}

//
// Docker network operations
//

// Create Network creates the Protos Docker network
func (dp *dockerPlatform) CreateNetwork(name string) (types.NetworkResource, error) {
	var net types.NetworkResource
	netResponse, err := dp.client.NetworkCreate(context.Background(), name, types.NetworkCreate{
		CheckDuplicate: true,
		Driver:         "bridge",
		EnableIPv6:     false,
		Internal:       false,
	})
	if err != nil {
		return net, errors.Wrapf(err, "Failed to create network '%s'", name)
	}
	net, err = dp.client.NetworkInspect(context.Background(), netResponse.ID, types.NetworkInspectOptions{})
	if err != nil {
		return net, errors.Wrapf(err, "Failed to create network '%s'", name)
	}
	return net, nil
}

// GetNetwork returns a Docker network based on its name
func (dp *dockerPlatform) GetNetwork(name string) (types.NetworkResource, error) {
	var net types.NetworkResource
	networks, err := dp.client.NetworkList(context.Background(), types.NetworkListOptions{Filters: filters.NewArgs(filters.KeyValuePair{Key: "name", Value: name})})
	if err != nil {
		return net, errors.Wrapf(err, "Failed to retrieve network '%s'", name)
	}
	if len(networks) == 0 {
		return net, util.NewTypedError("Could not find network "+name, core.ErrNetworkNotFound)
	}
	// Although networklist and networkinspect both return NetworkResource, the list doesn't populate all the fields of the structure so another network inspect call is needed
	net, err = dp.client.NetworkInspect(context.Background(), networks[0].ID, types.NetworkInspectOptions{})
	if err != nil {
		return net, util.NewTypedError("Could not find network "+name, core.ErrNetworkNotFound)
	}
	return net, nil
}

//
// Docker volume operations
//

// GetOrCreateVolume returns a volume, either an existing one or a new one
func (dp *dockerPlatform) GetOrCreateVolume(volumeID string, persistencePath string) (string, error) {
	// volume := DockerVolume{PersistencePath: persistencePath}

	if volumeID != "" {
		log.Debugf("Retrieving Docker volume '%s'", volumeID)
		dockerVolume, err := dp.client.VolumeInspect(context.Background(), volumeID)
		if err != nil {
			return "", err
		}
		return dockerVolume.Name, nil
	}
	log.Debug("Creating new Docker volume")
	dockerVolume, err := dp.client.VolumeCreate(context.Background(), volumetypes.VolumeCreateBody{Labels: map[string]string{"protos": "0.0.1", "persistencePath": persistencePath}})
	if err != nil {
		return "", err
	}
	log.Debugf("Created docker volume '%s'", dockerVolume.Name)
	return dockerVolume.Name, nil
}

// RemoveVolume removes a Docker volume
func (dp *dockerPlatform) RemoveVolume(volumeID string) error {
	log.Debugf("Removing Docker volume '%s'", volumeID)
	return dp.client.VolumeRemove(context.Background(), volumeID, false)
}

//
// Docker container operations
//

// NewContainer creates and returns a docker container reference
func (dp *dockerPlatform) NewContainer(name string, appid string, imageid string, volumeid string, volumeMountPath string, publicPorts []util.Port, installerParams map[string]string) (core.PlatformRuntimeUnit, error) {
	if imageid == "" {
		return nil, errors.New("Docker imageid is empty")
	}
	log.Debugf("Creating container '%s' from image '%s'", name, imageid)
	var ports []string
	for _, port := range publicPorts {
		ports = append(ports, "0.0.0.0:"+strconv.Itoa(port.Nr)+":"+strconv.Itoa(port.Nr)+"/"+string(port.Type))
	}
	exposedPorts, portBindings, err := nat.ParsePortSpecs(ports)
	if err != nil {
		return &DockerContainer{}, err
	}

	envvars := map[string]string{}
	for k, v := range installerParams {
		envvars[k] = v
	}
	envvars["APPID"] = appid

	// mounting container volumes
	mounts := []mount.Mount{}
	if volumeid != "" && volumeMountPath != "" {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeVolume,
			Source: volumeid,
			Target: volumeMountPath,
		})
	}

	containerConfig := &container.Config{
		Image:        gconfig.AppStoreHost + "/" + imageid,
		ExposedPorts: exposedPorts,
		Env:          combineEnv(envvars),
	}
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Mounts:       mounts,
	}
	if gconfig.InternalIP != "" {
		hostConfig.ExtraHosts = []string{"protos:" + gconfig.InternalIP}
	} else {
		hostConfig.Links = []string{"protos"}
	}

	containerIP, err := dp.allocateContainerIP()
	if err != nil {
		return &DockerContainer{}, errors.Wrap(err, "Failed to create container")
	}

	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			protosNetwork: &network.EndpointSettings{
				NetworkID: protosNetwork,
				IPAddress: containerIP,
			},
		},
	}

	dcnt, err := dp.client.ContainerCreate(context.Background(), containerConfig, hostConfig, networkConfig, name)
	if err != nil {
		return &DockerContainer{}, err
	}
	cnt := DockerContainer{ID: dcnt.ID, p: dp}
	err = cnt.Update()
	if err != nil {
		return &DockerContainer{}, err
	}

	return &cnt, nil

}

// GetDockerContainer retrieves and returns a docker container based on the id
func (dp *dockerPlatform) GetDockerContainer(id string) (core.PlatformRuntimeUnit, error) {
	cnt := DockerContainer{ID: id, p: dp}
	err := cnt.Update()
	if err != nil {
		return &DockerContainer{}, util.ErrorContainsTransform(errors.Wrapf(err, "Error retrieving Docker container %s", id), "No such container", core.ErrContainerNotFound)
	}
	return &cnt, nil
}

// GetAllDockerContainers retrieves all docker containers
func (dp *dockerPlatform) GetAllDockerContainers() (map[string]core.PlatformRuntimeUnit, error) {

	cnts := map[string]core.PlatformRuntimeUnit{}

	containers, err := dp.client.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return cnts, nil
	}

	for _, container := range containers {
		cnts[container.ID] = &DockerContainer{ID: container.ID, p: dp, Status: container.Status, IP: container.NetworkSettings.Networks[protosNetwork].IPAddress}
	}

	return cnts, nil
}

// Update reads the container and updates the struct fields
func (cnt *DockerContainer) Update() error {
	container, err := cnt.p.client.ContainerInspect(context.Background(), cnt.ID)
	if err != nil {
		return errors.Wrapf(err, "Error retrieving container '%s'", cnt.ID)
	}
	if network, ok := container.NetworkSettings.Networks[protosNetwork]; ok {
		cnt.IP = network.IPAddress
	}
	cnt.Status = container.State.Status
	cnt.ExitCode = container.State.ExitCode
	return nil
}

// Start starts a Docker container
func (cnt *DockerContainer) Start() error {
	log.Debugf("Starting container '%s'", cnt.ID)
	err := cnt.p.client.ContainerStart(context.Background(), cnt.ID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	statusCh, errCh := cnt.p.client.ContainerWait(ctx, cnt.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			if err.Error() == "context deadline exceeded" {
				return nil
			}
			errors.Wrapf(err, "Error while waiting for container '%s'", cnt.ID)
		}
	case <-statusCh:
		out, err := cnt.p.client.ContainerLogs(ctx, cnt.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
		if err != nil {
			panic(err)
		}
		buf := new(bytes.Buffer)
		buf.ReadFrom(out)
		allOutput := buf.String()
		var output string
		if len(allOutput) > 300 {
			output = allOutput[0:300]
		} else {
			output = allOutput
		}
		return fmt.Errorf("unexpected container termination: %s", output)
	}
	return nil
}

// Stop stops a Docker container
func (cnt *DockerContainer) Stop() error {
	stopTimeout := time.Duration(10) * time.Second
	err := cnt.p.client.ContainerStop(context.Background(), cnt.ID, &stopTimeout)
	if err != nil {
		return err
	}
	return nil
}

// Remove removes a Docker container
func (cnt *DockerContainer) Remove() error {
	err := cnt.p.client.ContainerRemove(context.Background(), cnt.ID, types.ContainerRemoveOptions{})
	if err != nil {
		return err
	}
	return nil
}

// GetID returns the ID of the container, as a string
func (cnt *DockerContainer) GetID() string {
	return cnt.ID
}

// GetIP returns the IP of the container, as a string
func (cnt *DockerContainer) GetIP() string {
	return cnt.IP
}

// GetStatus returns the status of the container, as a string
func (cnt *DockerContainer) GetStatus() string {
	return cnt.Status
}

// GetExitCode returns the exit code of the container, as an int
func (cnt *DockerContainer) GetExitCode() int {
	return cnt.ExitCode
}

//
// Image pull progress reporting
//

func (dp *downloadProgress) updatePercentage() {
	downloaded := int64(0)
	extracted := int64(0)
	for _, layer := range dp.layers {
		downloaded = downloaded + layer.downloaded
		extracted = extracted + layer.extracted
	}
	downloadedPercentage := (downloaded * 100) / dp.totalSize
	extractedPercentage := (extracted * 100) / dp.totalSize
	newPercentage := int(((downloadedPercentage * 4) + extractedPercentage) / 5)
	if newPercentage != dp.percentage {
		dp.percentage = newPercentage
		dp.t.SetPercentage(dp.initialPercentage + ((dp.percentage * dp.weight) / 100))
		dp.t.Save()
	}
}

func (dp *downloadProgress) complete() {
	dp.t.SetPercentage(dp.initialPercentage + dp.weight)
	dp.t.SetState("Finished downloading Docker image")
	dp.t.Save()
}

func (dp *downloadProgress) processEvent(event downloadEvent) {
	if layer, found := dp.layers[event.ID]; found {
		switch event.Status {
		case "Already exists":
			layer.downloaded = layer.size
			layer.extracted = layer.size
		case "Downloading":
			layer.downloaded = event.ProgressDetail.Current
		case "Extracting":
			layer.extracted = event.ProgressDetail.Current
		}
		dp.layers[event.ID] = layer
		dp.updatePercentage()
	}
}

func (dp *downloadProgress) addLayers(layers []distribution.Descriptor) {
	for _, mlayer := range layers {
		l := imageLayer{id: mlayer.Digest.Encoded()[0:12], size: mlayer.Size}
		dp.layers[l.id] = l
		dp.totalSize = dp.totalSize + l.size
	}
}

//
// Docker image operations
//

// DataPath returns the path inside the container where data is persisted
func (dp *dockerPlatform) GetDockerImageDataPath(image types.ImageInspect) (string, error) {
	vlen := len(image.Config.Volumes)
	if vlen == 0 {
		return "", nil
	} else if vlen > 1 {
		return "", errors.Errorf("Docker image '%s' has too many volumes", image.ID)
	}
	persistentPath := ""
	for k := range image.Config.Volumes {
		persistentPath = k
		break
	}
	return persistentPath, nil
}

// GetDockerImage returns a docker image by id, if it is labeled for protos
func (dp *dockerPlatform) GetDockerImage(id string) (types.ImageInspect, error) {
	log.Debugf("Retrieving Docker image '%s'", id)
	repoImage := gconfig.AppStoreHost + "/" + id
	image, _, err := dp.client.ImageInspectWithRaw(context.Background(), repoImage)
	if err != nil {
		return types.ImageInspect{}, util.ErrorContainsTransform(errors.Wrapf(err, "Error retrieving Docker image '%s'", id), "No such image", core.ErrImageNotFound)
	}

	if _, valid := image.Config.Labels["protos"]; valid == false {
		return types.ImageInspect{}, errors.Errorf("Image '%s' is missing the protos label", id)
	}

	if len(image.RepoTags) == 0 {
		image.RepoTags = append(image.RepoTags, "n/a")
	}

	return image, nil

}

// GetAllDockerImages returns all docker images
func (dp *dockerPlatform) GetAllDockerImages() (map[string]types.ImageSummary, error) {

	imgs := map[string]types.ImageSummary{}
	images, err := dp.client.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		return map[string]types.ImageSummary{}, err
	}

	for _, image := range images {
		if _, valid := image.Labels["protos"]; valid == false {
			continue
		}

		if len(image.RepoTags) == 0 {
			image.RepoTags = append(image.RepoTags, "n/a")
		} else if image.RepoTags[0] == "<none>:<none>" {
			image.RepoTags[0] = "n/a"
		}
		imgs[image.ID] = image
	}

	return imgs, nil

}

// RemoveDockerImage removes a docker image
func (dp *dockerPlatform) RemoveDockerImage(id string) error {
	_, err := dp.client.ImageRemove(context.Background(), id, types.ImageRemoveOptions{PruneChildren: true})
	if err != nil {
		return err
	}
	return nil
}

// PullDockerImage pulls a docker image from the Protos app store
func (dp *dockerPlatform) PullDockerImage(t core.Task, id string, installerName string, installerVersion string) error {
	repoImage := gconfig.AppStoreHost + "/" + id
	progress := &downloadProgress{t: t, layers: make(map[string]imageLayer), weight: 85, initialPercentage: t.GetPercentage()}
	regClient, err := registry.New(fmt.Sprintf("https://%s/", gconfig.AppStoreHost), "", "")
	if err != nil {
		return errors.Wrapf(err, "Failed to pull image '%s' from app store", id)
	}

	if strings.Contains(id, "@") == false {
		return fmt.Errorf("Failed to pull image from app store: invalid image id: '%s'", id)
	}

	idparts := strings.Split(id, "@")
	manifest, err := regClient.ManifestV2(idparts[0], idparts[1])
	if err != nil {
		return errors.Wrapf(err, "Failed to pull image '%s' from app store", id)
	}
	progress.addLayers(manifest.Layers)

	events, err := dp.client.ImageCreate(context.Background(), repoImage, types.ImageCreateOptions{})
	if err != nil {
		return errors.Wrapf(err, "Failed to pull image '%s' from app store", id)
	}

	var e downloadEvent
	d := json.NewDecoder(events)
	for {
		select {
		case <-t.Dying():
			err := events.Close()
			if err != nil {
				log.Error(errors.Wrap(err, "Failed to close the image pull while the task was canceled"))
			}
			return task.ErrKilledByUser
		default:
		}
		if err := d.Decode(&e); err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		progress.processEvent(e)
	}
	progress.complete()

	log.WithField("proc", t.GetID()).Debugf("Pulled image %s successfully", id)
	if e.Error != "" {
		return errors.Errorf("Failed to pull image '%s' from app store: %s", id, e.Error)
	}

	return nil
}
