package daemon

import (
	docker "github.com/docker/docker/client"
)

var dockerClient *docker.Client

func connectDocker() {
	log.Info("Connecting to the docker daemon")
	var err error

	dockerClient, err = docker.NewEnvClient()
	if err != nil {
		log.Fatal(err)
	}
}
