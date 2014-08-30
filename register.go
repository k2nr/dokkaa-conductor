package main

import (
	_ "encoding/json"
	"github.com/coreos/go-etcd/etcd"
	"github.com/fsouza/go-dockerclient"
	"log"
)

type Register interface {
	StartDockerEventLoop() chan struct{}
	Add(id DockerContainerID) error
	Delete(id DockerContainerID) error
}

type register struct {
	dockerClient DockerInterface
	etcdClient   *etcd.Client
}

func NewRegister(dc DockerInterface, etcdc *etcd.Client) Register {
	return &register{
		dockerClient: dc,
		etcdClient:   etcdc,
	}
}

func (r register) StartDockerEventLoop() chan struct{} {
	c := make(chan *docker.APIEvents)
	r.dockerClient.AddEventListener(c)
	quit := make(chan struct{})

	go func() {
		defer func() { quit <- struct{}{} }()
		for event := range c {
			switch event.Status {
			case "start":
				log.Printf("container started: %+v\n", event)
				r.Add(DockerContainerID(event.ID))
			case "die":
				log.Printf("container stopped: %+v\n", event)
				r.Delete(DockerContainerID(event.ID))
			}
		}
	}()
	return quit
}

func (r register) Add(id DockerContainerID) error {
	_, err := r.dockerClient.InspectContainer(string(id))
	if err != nil {
		return err
	}
	path := rootPath() + "/containers/" + string(id)
	_, err = r.etcdClient.Set(path, "", 0)
	return err
}

func (r register) Delete(id DockerContainerID) error {
	path := rootPath() + "/containers/" + string(id)
	_, err := r.etcdClient.Delete(path, false)
	if err != nil {
		return err
	}
	return err
}

func rootPath() string {
	return "/hosts/" + hostIP + "/"
}
