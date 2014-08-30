package main

import (
	"github.com/coreos/go-etcd/etcd"
	"github.com/fsouza/go-dockerclient"
	"log"
	"strconv"
	"regexp"
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
}

func newManifestRunner(manifest *Manifest, dc DockerInterface) *manifestRunner {
	return &manifestRunner{
		manifest:     manifest,
		dockerClient: dc,
	}
}

func (mr manifestRunner) run() error {
	container := mr.manifest.Container
	opts := buildRunOptions(mr.manifest.Container)
	runner := NewDockerRunner(mr.dockerClient)
	containerID, err := runner.Run(container.Image, opts)
	if err != nil {
		return err
	}
	log.Printf("%s is running: %s\n", container.Name, containerID)
	return nil
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
//				action := n.Action
				appName, containerName, _ := keySubMatch(n.Node.Key)
				val := n.Node.Value
				m := NewManifest(appName, containerName, val)
				log.Printf("%+v\n", m)
				s.Schedule(m)
			}
		}
	}()
	return quit
}

func keySubMatch(key string) (appName, containerName string, err error) {
	r,_ := regexp.Compile("/apps/([^/]+)/?([^/]+)?$")
	submatch := r.FindStringSubmatch(key)
	if len(submatch) == 0 {
		return "", "", nil
	}
	return submatch[0], submatch[1], nil
}

func (s scheduler) Schedule(ma *Manifest) error {
	image := ma.Container.Image
	err := s.pullImage(image)
	if err != nil {
		return err
	}

	mr := newManifestRunner(ma, s.dockerClient)
	mr.run()

	return nil
}

func (s scheduler) pullImage(image string) error {
	puller := NewDockerPuller(s.dockerClient)
	return puller.Pull(image)
}

func buildRunOptions(container Container) DockerRunOptions {
	name := container.Name
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
