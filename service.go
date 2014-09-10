package main

import (
	"encoding/json"
	"github.com/coreos/go-etcd/etcd"
	"github.com/fsouza/go-dockerclient"
	"path"
	"strconv"
	"strings"
)

type Service interface {
	Register(cli *etcd.Client) error
	Delete(cli *etcd.Client) error
}

type service struct {
	Name     string
	App      string
	Port     string
	HostPort string
}

type Announcement struct {
	Host     string `json:"host"`
	Port     int    `json:"port,omitempty"`
	Priority int    `json:"priority,omitempty"`
	Weight   int    `json:"weight,omitempty"`
	TTL      int    `json:"ttl,omitempty"`
}

func Services(container *docker.Container) ([]Service, error) {
	serviceMap := map[string]string{}
	var appName string
	for _, e := range container.Config.Env {
		parts := strings.SplitN(e, "=", 2)
		switch parts[0] {
		case "DOKKAA_APP_NAME":
			appName = parts[1]
		}
		if strings.HasPrefix(parts[0], "DOKKAA_SERVICE_") {
			name := strings.TrimPrefix(parts[0], "DOKKAA_SERVICE_")
			name = strings.ToLower(name)
			serviceMap[name] = parts[1]
		}
	}

	services := []Service{}
	for name, port := range serviceMap {
		hostPort := container.HostConfig.PortBindings[docker.Port(port+"/tcp")][0].HostPort
		hostPort = strings.TrimSuffix(hostPort, "/tcp")
		hostPort = strings.TrimSuffix(hostPort, "/udp")
		s := &service{
			App:      appName,
			Name:     name,
			Port:     port,
			HostPort: hostPort,
		}
		services = append(services, s)
	}

	return services, nil
}

func (s *service) path() string {
	base := path.Join("/", "skydns", "local", "skydns")
	return path.Join(base, s.App, s.Name)
}

func (s *service) Register(cli *etcd.Client) error {
	port, _ := strconv.Atoi(s.HostPort)
	ann := &Announcement{
		Host: hostIP,
		Port: port,
	}
	value, _ := json.Marshal(ann)
	cli.Set(s.path(), string(value), 0)

	return nil
}

func (s *service) Delete(cli *etcd.Client) error {
	_, err := cli.Delete(s.path(), false)
	return err
}
