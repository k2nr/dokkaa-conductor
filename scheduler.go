package main

import (
	"github.com/coreos/go-etcd/etcd"
	"github.com/fsouza/go-dockerclient"
	"log"
	"strconv"
)

type Scheduler interface {
	Schedule(ma *Manifest) error
	StartSchedulingLoop() chan struct{}
}

type scheduler struct {
	dockerClient DockerInterface
	etcdClient   *etcd.Client
}

type manifestRunner struct {
	manifest     *Manifest
	dockerClient DockerInterface
	runHist      map[string]DockerContainerID
}

func newManifestRunner(manifest *Manifest, dc DockerInterface) *manifestRunner {
	return &manifestRunner{
		manifest:     manifest,
		dockerClient: dc,
		runHist:      map[string]DockerContainerID{},
	}
}

func (mr manifestRunner) runAll() {
	for name, _ := range mr.manifest.Containers {
		mr.run(name)
	}
}

func (mr manifestRunner) run(name string) {
	if _, ok := mr.runHist[name]; ok {
		return
	}
	container := mr.manifest.Containers[name]

	// run linked containers first
	if len(container.Links) > 0 {
		for _, linkName := range container.Links {
			mr.run(linkName)
		}
	}

	opts := buildRunOptions(name, container)
	runner := NewDockerRunner(mr.dockerClient)
	containerID, err := runner.Run(container.Image, opts)
	if err == nil {
		log.Printf("%s is running: %s\n", name, containerID)
		mr.runHist[name] = containerID
	}
}

func NewScheduler(dc DockerInterface, etcdc *etcd.Client) Scheduler {
	return &scheduler{
		dockerClient: dc,
		etcdClient:   etcdc,
	}
}

func (s scheduler) StartSchedulingLoop() chan struct{} {
	recv := make(chan *etcd.Response)
	quit := make(chan struct{})

	go s.etcdClient.Watch("/apps/", 0, true, recv, nil)
	go func() {
		defer func() { quit <- struct{}{} }()
		for n := range recv {
			if n != nil {
				val := n.Node.Value
				m := NewManifest(val)
				log.Printf("%+v\n", m)
				err := s.Schedule(&m)
				assert(err)
			}
		}
	}()
	return quit
}

func (s scheduler) Schedule(ma *Manifest) error {
	for _, container := range ma.Containers {
		image := container.Image
		err := s.pullImage(image)
		if err != nil {
			return err
		}
	}

	mr := newManifestRunner(ma, s.dockerClient)
	mr.runAll()

	return nil
}

func (s scheduler) pullImage(image string) error {
	puller := NewDockerPuller(s.dockerClient)
	return puller.Pull(image)
}

func buildRunOptions(name string, container Container) DockerRunOptions {
	exposedPorts := buildExposedPorts(container.Ports)
	return DockerRunOptions{
		ContainerName: name,
		ContainerConfig: &docker.Config{
			ExposedPorts: exposedPorts,
		},
		HostConfig: &docker.HostConfig{
			PublishAllPorts: true,
		},
	}
}

func buildExposedPorts(ports []int) map[docker.Port]struct{} {
	exposedPorts := map[docker.Port]struct{}{}
	for _, p := range ports {
		exposedPorts[docker.Port(strconv.Itoa(p)+"/tcp")] = struct{}{}
	}

	return exposedPorts
}
