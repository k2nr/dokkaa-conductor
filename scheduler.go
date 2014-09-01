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
		log.Printf("error: %+v\n", err)
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
				action := n.Action
				appName, containerName, _ := keySubMatch(n.Node.Key)
				val := n.Node.Value
				m := NewManifest(appName, containerName, val)

				switch action {
				case "set":
					order := myHostLoadOrder(hostIP)
					time.Sleep(time.Second * time.Duration(order))
					acquired, _ := s.acquire(m)
					if acquired {
						log.Printf("acquired: %+v\n", m)
						s.Schedule(m)
					}
				case "delete":
					name := appName + "---" + containerName
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
		log.Println("schedule loop ended")
	}()
	return quit
}

func myHostLoadOrder(ip string) int {
	// TODO
	return 0
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
		log.Printf("modified index: %d", modifiedIndex)
		if err != nil {
			log.Printf("error: %+v\n", err.(*etcd.EtcdError))
//			return false, err
		}

		if len(hosts) >= manifest.Container.Scale {
			log.Printf("hosts >= scale", err)
			return false, nil
		}

		included := false
		for _, h := range hosts {
			log.Println(h)
			if h == hostIP {
				included = true
				break
			}
		}
		log.Printf("included: %+v\n", included)

		if !included {
			hosts = append(hosts, hostIP)
			log.Printf("hosts: %+v\n", hosts)

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
				log.Println("acquired")
				return true, nil
			} else {
				log.Printf("err: %+v\n", err)
			}
		} else {
			// TODO: check running containers and run manifest's container if the container not running
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
