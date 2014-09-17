package main

import (
	"encoding/json"
	"github.com/coreos/go-etcd/etcd"
	"github.com/fsouza/go-dockerclient"
	"log"
	"math"
	"regexp"
	"strconv"
	"time"
)

const (
	ambassadorName = "__ambassador"
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
			s.release(m)
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

type Host struct {
	Addr   string `json:"addr"`
	Status string `json:"status"`
}

func keySubMatch(key string) (appName, containerName string, err error) {
	r, _ := regexp.Compile("/apps/([^/]+)/([^/]+)/manifest$")
	submatch := r.FindStringSubmatch(key)
	if len(submatch) == 0 {
		return "", "", nil
	}
	return submatch[1], submatch[2], nil
}

func (s scheduler) getHosts(manifest *Manifest) ([]string, error) {
	key := manifest.HostsDirKey()
	resp, err := s.etcdClient.Get(key, true, true)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	var hosts []string
	for _, v := range resp.Node.Nodes {
		var h Host
		json.Unmarshal([]byte(v.Value), &h)
		hosts = append(hosts, h.Addr)
	}
	return hosts, err
}

func (s scheduler) hostsIncluded(manifest *Manifest) (bool, error) {
	scale := manifest.Container.Scale
	hosts, err := s.getHosts(manifest)
	if err != nil {
		// error is returned if hosts/ not found. for this error, we have to continue as if there's no error
		return false, nil
	}
	slice := int(math.Min(float64(scale), float64(len(hosts))))
	hosts = hosts[:int(slice)]
	included := false
	if err == nil {
		for _, h := range hosts {
			if h == hostIP {
				included = true
				break
			}
		}
	}
	limitExceeded := !included && len(hosts) >= scale
	if limitExceeded {
		log.Println("already acquired by other hosts. scale=", manifest.Container.Scale, " hosts=", hosts)
	}
	return included, nil
}

func (s scheduler) acquire(manifest *Manifest) (bool, error) {
	// check if this host is already included
	included, err := s.hostsIncluded(manifest)
	if included {
		return true, nil
	}

	// Set Host
	hs, err := json.Marshal(Host{
		Addr:   hostIP,
		Status: "creating",
	})
	if err != nil {
		log.Println(err)
		return false, err
	}
	_, err = s.etcdClient.CreateInOrder(manifest.HostsDirKey(), string(hs), 0)
	if err != nil {
		log.Println(err.(*etcd.EtcdError))
		return false, err
	}

	// Check acquired order exceeds scale limit
	included, err = s.hostsIncluded(manifest)
	return included, err
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

func (s scheduler) release(manifest *Manifest) error {
	key := manifest.HostsDirKey()
	resp, err := s.etcdClient.Get(key, true, true)
	if err != nil {
		log.Println(err)
		return err
	}
	for _, n := range resp.Node.Nodes {
		var h Host
		err = json.Unmarshal([]byte(n.Value), &h)
		if err != nil {
			continue
		}
		if h.Addr == hostIP {
			s.etcdClient.Delete(n.Key, false)
		}
	}
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
