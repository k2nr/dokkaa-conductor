package main

import (
	"errors"
	"testing"

	"github.com/dchest/uniuri"
	"github.com/fsouza/go-dockerclient"
)

type dockerMock struct {
	host         string
	pulledImages []docker.PullImageOptions
	containers   []*docker.Container
}

func (d *dockerMock) ListContainers(options docker.ListContainersOptions) ([]docker.APIContainers, error) {
	panic("")
}

func (d *dockerMock) InspectContainer(id string) (*docker.Container, error) {
	panic("")
}

func (d *dockerMock) CreateContainer(docker.CreateContainerOptions) (*docker.Container, error) {
	container := &docker.Container{
		ID: uniuri.New(),
	}
	d.containers = append(d.containers, container)
	return container, nil
}

func (d *dockerMock) StartContainer(id string, hostConfig *docker.HostConfig) error {
	for _, c := range d.containers {
		if c.ID == id {
			return nil
		}
	}
	return errors.New("No Such Container")
}

func (d *dockerMock) StopContainer(id string, timeout uint) error {
	panic("")
}

func (d *dockerMock) RemoveContainer(opts docker.RemoveContainerOptions) error {
	panic("")
}

func (d *dockerMock) WaitContainer(id string) (int, error) {
	panic("")
}

func (d *dockerMock) PullImage(opts docker.PullImageOptions, auth docker.AuthConfiguration) error {
	d.pulledImages = append(d.pulledImages, opts)
	return nil
}

func (d *dockerMock) AddEventListener(listener chan<- *docker.APIEvents) error {
	panic("")
}

func TestRun(t *testing.T) {
	runner := NewDockerRunner(&dockerMock{})
	options := DockerRunOptions{
		ContainerName:   "test-container",
		ContainerConfig: &docker.Config{},
		HostConfig:      &docker.HostConfig{},
	}
	_, err := runner.Run("ubuntu:14.04", options)
	if err != nil {
		t.Error(err)
	}
}

func TestPull(t *testing.T) {
	expects := map[string][2]string{
		"ubuntu:14.04": [2]string{"ubuntu", "14.04"},
		"ubuntu":       [2]string{"ubuntu", "latest"},
	}
	for input, result := range expects {
		d := dockerMock{}
		puller := NewDockerPuller(&d)
		puller.Pull(input)
		image := d.pulledImages[0]
		if image.Repository != result[0] {
			t.Error(image)
		}
		if image.Tag != result[1] {
			t.Error(image)
		}
	}
}
