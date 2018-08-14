package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	volumetypes "github.com/docker/docker/api/types/volume"
	docker "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/nustiueudinastea/protos/util"
	"github.com/pkg/errors"
)

const protosNetwork = "protosnet"

// DockerContainer represents a container
type DockerContainer struct {
	ID     string
	IP     string
	Status string
}

// DockerVolume represents a persistent disk volume
type DockerVolume struct {
	ID              string
	PersistencePath string
}

var dockerClient *docker.Client

// ConnectDocker connects to the Docker daemon
func ConnectDocker() {
	log.Info("Connecting to the docker daemon")
	var err error

	dockerClient, err = docker.NewEnvClient()
	if err != nil {
		log.Fatal(err)
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

//
// Docker network operations
//

// CreateDockerNetwork creates a Docker network
func CreateDockerNetwork(name string) (string, error) {
	netResponse, err := dockerClient.NetworkCreate(context.Background(), name, types.NetworkCreate{
		CheckDuplicate: true,
		Driver:         "bridge",
		EnableIPv6:     false,
		Internal:       false,
	})
	if err != nil {
		return "", errors.Wrap(err, "Failed to create network "+name)
	}
	return netResponse.ID, nil
}

// DockerNetworkExists checks if a Docker network exists and returns a bool
func DockerNetworkExists(id string) bool {
	_, err := dockerClient.NetworkInspect(context.Background(), id, types.NetworkInspectOptions{})
	if err != nil {
		return false
	}
	return true
}

//
// Docker volume operations
//

// GetOrCreateDockerVolume returns a volume, either an existing one or a new one
func GetOrCreateDockerVolume(volumeID string, persistencePath string) (*DockerVolume, error) {
	volume := DockerVolume{PersistencePath: persistencePath}
	if volumeID != "" {
		log.Debug("Retrieving Docker volume " + volumeID)
		dockerVolume, err := dockerClient.VolumeInspect(context.Background(), volumeID)
		if err != nil {
			return nil, err
		}
		volume.ID = dockerVolume.Name
		return &volume, nil
	}
	log.Debug("Creating new Docker volume")
	dockerVolume, err := dockerClient.VolumeCreate(context.Background(), volumetypes.VolumeCreateBody{Labels: map[string]string{"protos": "0.0.1"}})
	if err != nil {
		return nil, err
	}
	log.Debug("Created docker volume " + dockerVolume.Name)
	volume.ID = dockerVolume.Name
	return &volume, nil
}

// RemoveDockerVolume removes a Docker volume
func RemoveDockerVolume(volumeID string) error {
	log.Debug("Removing Docker volume " + volumeID)
	return dockerClient.VolumeRemove(context.Background(), volumeID, false)
}

//
// Docker container operations
//

// NewDockerContainer creates and returns a docker container reference
func NewDockerContainer(name string, appid string, imageid string, volume *DockerVolume, publicPorts []util.Port, installerParams map[string]string) (*DockerContainer, error) {
	if imageid == "" {
		return nil, errors.New("Docker imageid is empty")
	}
	log.Debug("Creating container " + name + " from image " + imageid)
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
	if volume != nil {
		mounts = append(mounts, mount.Mount{
			Type:   mount.TypeVolume,
			Source: volume.ID,
			Target: volume.PersistencePath,
		})
	}

	containerConfig := &container.Config{
		Image:        imageid,
		ExposedPorts: exposedPorts,
		Env:          combineEnv(envvars),
	}
	hostConfig := &container.HostConfig{
		Links:        []string{"protos"},
		PortBindings: portBindings,
		Mounts:       mounts,
	}
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"protosnet": &network.EndpointSettings{
				NetworkID: protosNetwork,
			},
		},
	}

	dcnt, err := dockerClient.ContainerCreate(context.Background(), containerConfig, hostConfig, networkConfig, name)
	if err != nil {
		return &DockerContainer{}, err
	}
	cnt := DockerContainer{ID: dcnt.ID}
	err = cnt.Update()
	if err != nil {
		return &DockerContainer{}, err
	}

	return &cnt, nil

}

// GetDockerContainer retrieves and returns a docker container based on the id
func GetDockerContainer(id string) (*DockerContainer, error) {
	cnt := DockerContainer{ID: id}
	err := cnt.Update()
	if err != nil {
		return &DockerContainer{}, err
	}
	return &cnt, nil
}

// GetAllDockerContainers retrieves all docker containers
func GetAllDockerContainers() (map[string]*DockerContainer, error) {

	cnts := map[string]*DockerContainer{}

	containers, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return cnts, nil
	}

	for _, container := range containers {
		cnts[container.ID] = &DockerContainer{ID: container.ID, Status: container.Status, IP: container.NetworkSettings.Networks[protosNetwork].IPAddress}
	}

	return cnts, nil
}

// Update reads the container and updates the struct fields
func (cnt *DockerContainer) Update() error {
	container, err := dockerClient.ContainerInspect(context.Background(), cnt.ID)
	if err != nil {
		return errors.Wrap(err, "Error retrieving container "+cnt.ID)
	}
	if network, ok := container.NetworkSettings.Networks[protosNetwork]; ok {
		cnt.IP = network.IPAddress
	}
	cnt.Status = container.State.Status
	return nil
}

// Start starts a Docker container
func (cnt *DockerContainer) Start() error {
	log.Debug("Starting container " + cnt.ID)
	err := dockerClient.ContainerStart(context.Background(), cnt.ID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	statusCh, errCh := dockerClient.ContainerWait(ctx, cnt.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			if err.Error() == "context deadline exceeded" {
				return nil
			} else {
				errors.Wrap(err, "Error while waiting for container")
			}
		}
	case <-statusCh:
		out, err := dockerClient.ContainerLogs(ctx, cnt.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
		if err != nil {
			panic(err)
		}
		buf := new(bytes.Buffer)
		buf.ReadFrom(out)
		return fmt.Errorf("unexpected container termination: %s", buf.String())
	}
	return nil
}

// Stop stops a Docker container
func (cnt *DockerContainer) Stop() error {
	stopTimeout := time.Duration(10) * time.Second
	err := dockerClient.ContainerStop(context.Background(), cnt.ID, &stopTimeout)
	if err != nil {
		return err
	}
	return nil
}

// Remove removes a Docker container
func (cnt *DockerContainer) Remove() error {
	err := dockerClient.ContainerRemove(context.Background(), cnt.ID, types.ContainerRemoveOptions{})
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

//
// Docker image operations
//

// GetDockerImageDataPath returns the path inside the container where data is persisted
func GetDockerImageDataPath(image types.ImageInspect) (string, error) {
	vlen := len(image.Config.Volumes)
	if vlen == 0 {
		return "", nil
	} else if vlen > 1 {
		return "", errors.New("Docker image " + image.ID + " has too many volumes")
	}
	persistentPath := ""
	for k := range image.Config.Volumes {
		persistentPath = k
		break
	}
	return persistentPath, nil
}

// GetDockerImage returns a docker image by id, if it is labeled for protos
func GetDockerImage(id string) (types.ImageInspect, error) {
	image, _, err := dockerClient.ImageInspectWithRaw(context.Background(), id)
	if err != nil {
		return types.ImageInspect{}, err
	}

	if _, valid := image.Config.Labels["protos"]; valid == false {
		return types.ImageInspect{}, errors.New("Image " + id + " is missing the protos label")
	}

	if len(image.RepoTags) == 0 {
		image.RepoTags = append(image.RepoTags, "n/a")
	}

	return image, nil

}

// GetAllDockerImages returns all docker images
func GetAllDockerImages() (map[string]types.ImageSummary, error) {

	imgs := map[string]types.ImageSummary{}
	images, err := dockerClient.ImageList(context.Background(), types.ImageListOptions{})
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
func RemoveDockerImage(id string) error {
	_, err := dockerClient.ImageRemove(context.Background(), id, types.ImageRemoveOptions{PruneChildren: true})
	if err != nil {
		return err
	}
	return nil
}

// PullDockerImage pulls a docker image from the Protos app store
func PullDockerImage(name string, tag string) error {
	repoImage := gconfig.AppStoreHost + "/" + name
	imageStr := repoImage + ":" + tag
	events, err := dockerClient.ImagePull(context.Background(), imageStr, types.ImagePullOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to pull image from app store")
	}

	type Event struct {
		Status         string `json:"status"`
		Error          string `json:"error"`
		Progress       string `json:"progress"`
		ProgressDetail struct {
			Current int `json:"current"`
			Total   int `json:"total"`
		} `json:"progressDetail"`
	}

	var event *Event
	d := json.NewDecoder(events)
	for {
		if err := d.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
	}

	log.Debugf("Pulled image %s with status: %s:", name, event.Status)
	if event.Error != "" {
		return errors.New("Failed to pull image from app store: " + event.Error)
	}

	err = dockerClient.ImageTag(context.Background(), imageStr, name+":"+tag)
	if err != nil {
		return errors.Wrap(err, "Something went wrong while re-tagging Docker image")
	}
	_, err = dockerClient.ImageRemove(context.Background(), imageStr, types.ImageRemoveOptions{})
	if err != nil {
		return errors.Wrap(err, "Something went wront while removing old Docker image tag")
	}
	return nil
}
