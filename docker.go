package main

import (
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"strings"
)

type DockerInterface interface {
	ListContainers(options docker.ListContainersOptions) ([]docker.APIContainers, error)
	InspectContainer(id string) (*docker.Container, error)
	CreateContainer(docker.CreateContainerOptions) (*docker.Container, error)
	StartContainer(id string, hostConfig *docker.HostConfig) error
	StopContainer(id string, timeout uint) error
	RemoveContainer(opts docker.RemoveContainerOptions) error
	WaitContainer(id string) (int, error)
	PullImage(opts docker.PullImageOptions, auth docker.AuthConfiguration) error
	AddEventListener(listener chan<- *docker.APIEvents) error
}

func NewDockerClient(host string) (DockerInterface, error) {
	if host == "" {
		host = "unix:///var/run/docker.sock"
	}

	return docker.NewClient(host)
}

type DockerContainerID string
type DockerImageID string

type DockerPuller interface {
	Pull(image string) error
}

type dockerPuller struct {
	client DockerInterface
}

func NewDockerPuller(client DockerInterface) DockerPuller {
	return dockerPuller{
		client: client,
	}
}

func (dp dockerPuller) Pull(image string) error {
	image, tag := parseImageName(image)

	if len(tag) == 0 {
		tag = "latest"
	}

	opts := docker.PullImageOptions{
		Repository: image,
		Tag:        tag,
	}

	return dp.client.PullImage(opts, docker.AuthConfiguration{})
}

func parseImageName(image string) (string, string) {
	tag := ""
	parts := strings.SplitN(image, "/", 2)
	repo := ""
	if len(parts) == 2 {
		repo = parts[0]
		image = parts[1]
	}
	parts = strings.SplitN(image, ":", 2)
	if len(parts) == 2 {
		image = parts[0]
		tag = parts[1]
	}
	if repo != "" {
		image = fmt.Sprintf("%s/%s", repo, image)
	}
	return image, tag
}

type DockerRunOptions struct {
	ContainerName   string
	ContainerConfig *docker.Config
	HostConfig      *docker.HostConfig
}

type DockerRunner interface {
	Run(image string, opts DockerRunOptions) (DockerContainerID, error)
}

type dockerRunner struct {
	client DockerInterface
}

func NewDockerRunner(client DockerInterface) DockerRunner {
	return dockerRunner{
		client: client,
	}
}

func (r dockerRunner) Run(image string, opts DockerRunOptions) (DockerContainerID, error) {
	createOpts := docker.CreateContainerOptions{
		Name:   opts.ContainerName,
		Config: opts.ContainerConfig,
	}
	createOpts.Config.Image = image

	container, err := r.client.CreateContainer(createOpts)
	if err != nil {
		return "", err
	}

	err = r.client.StartContainer(container.ID, opts.HostConfig)

	return DockerContainerID(container.ID), err
}
