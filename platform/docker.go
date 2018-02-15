package platform

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	docker "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

const protosNetwork = "bridge"

// DockerContainer represents
type DockerContainer struct {
	ID     string
	IP     string
	Status string
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
// Docker container operation
//

// NewDockerContainer creates and returns a docker container reference
func NewDockerContainer(name string, appid string, imageid string, ports string, installerParams map[string]string) (*DockerContainer, error) {
	log.Debug("Creating container " + name + " from image " + imageid)
	var publicports []string
	for _, v := range strings.Split(ports, ",") {
		publicports = append(publicports, "0.0.0.0:"+v+":"+v+"/tcp")
	}
	exposedPorts, portBindings, err := nat.ParsePortSpecs(publicports)
	if err != nil {
		return &DockerContainer{}, err
	}

	envvars := map[string]string{}
	for k, v := range installerParams {
		envvars[k] = v
	}
	envvars["APPID"] = appid

	containerConfig := &container.Config{
		Image:        imageid,
		ExposedPorts: exposedPorts,
		Env:          combineEnv(envvars),
	}
	hostConfig := &container.HostConfig{
		Links:        []string{"protos"},
		PortBindings: portBindings,
	}

	dcnt, err := dockerClient.ContainerCreate(context.Background(), containerConfig, hostConfig, nil, name)
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
		return errors.New("Error retrieving container " + cnt.ID + ": " + err.Error())
	}
	cnt.IP = container.NetworkSettings.Networks[protosNetwork].IPAddress
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
// Docker image operation
//

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
