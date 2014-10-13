package main

import (
	"encoding/json"
	"log"
	"path"
	"strconv"
	"strings"

	"github.com/fsouza/go-dockerclient"
)

type Service interface {
	Register(cli EtcdInterface) error
	Delete(cli EtcdInterface) error
}

type service struct {
	Name     string
	App      string
	Port     string
	HostPort string
	Role     string
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
	roleMap := map[string]string{}

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
		if strings.HasPrefix(parts[0], "DOKKAA_ROLE_") {
			name := strings.TrimPrefix(parts[0], "DOKKAA_ROLE_")
			name = strings.ToLower(name)
			roleMap[name] = parts[1]
		}
	}

	services := []Service{}
	for name, port := range serviceMap {
		portBinding, ok := container.NetworkSettings.Ports[docker.Port(port+"/tcp")]
		if !ok {
			log.Println("No port binding for ", port+"/tcp")
			continue
		}
		hostPort := portBinding[0].HostPort
		hostPort = strings.TrimSuffix(hostPort, "/tcp")
		hostPort = strings.TrimSuffix(hostPort, "/udp")
		s := &service{
			App:      appName,
			Name:     name,
			Port:     port,
			HostPort: hostPort,
			Role:     roleMap[name],
		}
		services = append(services, s)
	}

	return services, nil
}

func (s *service) appPath() string {
	base := path.Join("/", "skydns", "local", "skydns")
	return path.Join(base, s.App)
}

func (s *service) servicePath() string {
	return path.Join(s.appPath(), s.Name)
}

func (s *service) webPath() string {
	return path.Join("/", "skydns", "local", "skydns", "web", s.App)
}

func (s *service) Register(cli EtcdInterface) error {
	port, _ := strconv.Atoi(s.HostPort)
	ann := &Announcement{
		Host: hostIP,
		Port: port,
	}
	value, _ := json.Marshal(ann)
	cli.Set(s.servicePath(), string(value), 0)

	if s.Role == "web" {
		cli.Set(s.webPath(), string(value), 0)
	}

	return nil
}

func (s *service) Delete(cli EtcdInterface) error {
	_, err := cli.Delete(s.servicePath(), false)
	return err
}
