package main

import (
	_ "encoding/json"
	"log"
	"strings"

	"github.com/fsouza/go-dockerclient"
)

type Register interface {
	StartDockerEventLoop() chan struct{}
	Add(id DockerContainerID) error
	Delete(id DockerContainerID) error
}

type register struct {
	dockerClient DockerInterface
	etcdClient   EtcdInterface
}

func NewRegister(dc DockerInterface, etcdc EtcdInterface) Register {
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
		defer close(quit)
		for event := range c {
			switch event.Status {
			case "start":
				r.Add(DockerContainerID(event.ID))
			case "die":
				r.Delete(DockerContainerID(event.ID))
			}
		}
		log.Println("docker loop ended")
	}()
	return quit
}

func (r register) Add(id DockerContainerID) error {
	container, err := r.dockerClient.InspectContainer(string(id))
	if err != nil {
		log.Println("register: ", err)
		return err
	}
	if strings.HasPrefix(container.Name, "__") {
		// containers whose name starts with "__" doesn't be registered
		return nil
	}
	path := rootPath() + "containers/" + string(id)
	_, err = r.etcdClient.Set(path, "", 0)

	services, err := Services(container)
	if err != nil {
		log.Println("register: ", err)
		return err
	}
	for _, s := range services {
		err = s.Register(r.etcdClient)
		if err != nil {
			log.Println("register: ", err)
		}
	}
	return nil
}

func (r register) Delete(id DockerContainerID) error {
	path := rootPath() + "containers/" + string(id)
	_, err := r.etcdClient.Delete(path, false)
	if err != nil {
		return err
	}

	container, err := r.dockerClient.InspectContainer(string(id))
	if err != nil {
		log.Println("register: ", err)
		return err
	}
	services, err := Services(container)
	for _, s := range services {
		err = s.Delete(r.etcdClient)
		if err != nil {
			log.Println("register: ", err)
		}
	}
	return err
}

func rootPath() string {
	return "/hosts/" + hostIP + "/"
}
