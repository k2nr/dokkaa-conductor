package main

import (
	"encoding/json"
	"github.com/coreos/go-etcd/etcd"
	"github.com/fsouza/go-dockerclient"
	"log"
	"regexp"
	"strconv"
	"time"
)

const (
	ambassadorName = "ambassador"
)

type Scheduler interface {
	Schedule(ma *Manifest) error
	StartSchedulingLoop() chan struct{}
}

type scheduler struct {
	dockerClient DockerInterface
	etcdClient   EtcdInterface
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
	opts := mr.buildRunOptions(mr.manifest.Container)
	mr.dockerClient.RemoveContainer(docker.RemoveContainerOptions{
		ID:    container.Name,
		Force: true,
	})
	runner := NewDockerRunner(mr.dockerClient)
	containerID, err := runner.Run(container.Image, opts)
	if err != nil {
		log.Printf("error: %+v\n", err)
		return err
	}
	log.Printf("%s is running: %s\n", container.Name, containerID)
	return nil
}

func NewScheduler(dc DockerInterface, etcdc EtcdInterface) Scheduler {
	return &scheduler{
		dockerClient: dc,
		etcdClient:   etcdc,
	}
}

func (s scheduler) WatchAppChanges() {
	watcher := NewEtcdWatcher(s.etcdClient)
	recv := watcher.Watch("/apps", true)
	for n := range recv {
		action := n.Action
		appName, containerName, _ := keySubMatch(n.Node.Key)
		val := n.Node.Value
		m, err := NewManifest(appName, containerName, val)
		if err != nil {
			log.Println(err)
			continue
		}
		log.Printf("etcd received: action=%s", action)

		switch action {
		case "set":
			cls := NewCluster(s.etcdClient)
			order, err := cls.HostLoadOrder(hostIP)
			if err != nil {
				log.Println(err)
				continue
			}
			time.Sleep(time.Second * time.Duration(order))
			acquired, _ := s.acquire(m)
			if acquired {
				log.Printf("acquired: %+v\n", m)
				s.Schedule(m)
			}
		case "delete":
			name := m.Container.Name
			err := s.dockerClient.StopContainer(name, 60)
			if err != nil {
				log.Printf("error: %+v\n", err)
			}
			_, err = s.dockerClient.WaitContainer(name)
			if err != nil {
				log.Printf("error: %+v\n", err)
			}
			opts := docker.RemoveContainerOptions{
				ID:            name,
				RemoveVolumes: true,
				Force:         false,
			}
			err = s.dockerClient.RemoveContainer(opts)
			if err != nil {
				log.Printf("error: %+v\n", err)
			}
		}
	}
}

func (s scheduler) StartSchedulingLoop() chan struct{} {
	quit := make(chan struct{})

	go func() {
		defer close(quit)
		s.WatchAppChanges()
	}()
	return quit
}

func keySubMatch(key string) (appName, containerName string, err error) {
	r, _ := regexp.Compile("/apps/([^/]+)/([^/]+)/manifest$")
	submatch := r.FindStringSubmatch(key)
	if len(submatch) == 0 {
		return "", "", nil
	}
	return submatch[1], submatch[2], nil
}

func (s scheduler) getHosts(manifest *Manifest) ([]string, uint64, error) {
	key := manifest.HostsKey()
	resp, err := s.etcdClient.Get(key, false, false)
	if err != nil {
		log.Printf("error: %+v\n", err)
		return nil, 0, err
	}
	var hosts []string
	err = json.Unmarshal([]byte(resp.Node.Value), &hosts)
	return hosts, resp.Node.ModifiedIndex, err
}

func (s scheduler) acquire(manifest *Manifest) (bool, error) {
	for i := 0; i < 3; i++ {
		hosts, modifiedIndex, err := s.getHosts(manifest)
		if err != nil {
			log.Printf("error: %+v\n", err.(*etcd.EtcdError))
			//			return false, err
		}

		included := false
		for _, h := range hosts {
			if h == hostIP {
				included = true
				break
			}
		}

		if !included {
			if len(hosts) >= manifest.Container.Scale {
				log.Println("already acquired by other hosts. scale=", manifest.Container.Scale, " hosts=", hosts)
				return false, nil
			}

			hosts = append(hosts, hostIP)

			hostsStr, err := json.Marshal(hosts)
			if err != nil {
				log.Printf("error: %+v\n", err)
				return false, err
			}
			if modifiedIndex == 0 {
				_, err = s.etcdClient.Create(
					manifest.HostsKey(),
					string(hostsStr),
					0)
			} else {
				_, err = s.etcdClient.CompareAndSwap(
					manifest.HostsKey(),
					string(hostsStr),
					0,
					"",
					modifiedIndex)
			}
			if err == nil {
				return true, nil
			} else {
			}
		} else {
			// TODO: check running containers and run manifest's container if the container not running
			return true, nil
		}
	}

	return false, nil
}

func (s scheduler) Schedule(ma *Manifest) error {
	image := ma.Container.Image
	err := s.pullImage(image)
	if err != nil {
		log.Printf("error: %+v\n", err)
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

func (mr manifestRunner) buildRunOptions(container *Container) DockerRunOptions {
	name := container.Name
	env := buildEnv(container.Env)
	var ports []int
	for _, p := range container.Services {
		ports = append(ports, p)
	}
	exposedPorts := buildExposedPorts(ports)
	var links []string
	c, _ := mr.dockerClient.InspectContainer(ambassadorName)
	if c != nil {
		links = append(links, ambassadorName+":backends")
	}
	return DockerRunOptions{
		ContainerName: name,
		ContainerConfig: &docker.Config{
			Env:          env,
			ExposedPorts: exposedPorts,
			Cmd:          container.Command,
		},
		HostConfig: &docker.HostConfig{
			PublishAllPorts: true,
			Links:           links,
		},
	}
}

func buildEnv(env map[string]string) []string {
	res := []string{}
	for k, v := range env {
		res = append(res, k+"="+v)
	}

	return res
}

func buildExposedPorts(ports []int) map[docker.Port]struct{} {
	exposedPorts := map[docker.Port]struct{}{}
	for _, p := range ports {
		exposedPorts[docker.Port(strconv.Itoa(p)+"/tcp")] = struct{}{}
	}

	return exposedPorts
}
