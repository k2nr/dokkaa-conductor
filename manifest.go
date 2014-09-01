package main

import (
	"encoding/json"
)

type Port struct {
	HostPort      int
	ContainerPort int
}

type Container struct {
	Image    string
	Name     string
	Scale    int
	Env      map[string]string
	Links    []string
	Command  []string
	Dns      []string
	Hostname string
	Ports    []int
}

type Manifest struct {
	AppName   string
	ContainerName string
	Container Container
}

func NewManifest(app, container, val string) *Manifest {
	m := Manifest{
		AppName: app,
		ContainerName: container,
	}
	var c Container
	err := json.Unmarshal([]byte(val), &c)
	if err == nil {
		m.Container = c
		m.Container.Name = app + "---" + container
	}

	return &m
}

func (m *Manifest) keyRoot() string {
	a := m.AppName
	c := m.ContainerName
	return "/apps/" + a + "/" + c + "/"
}

func (m *Manifest) ManifestKey() string {
	return m.keyRoot() + "manifest"
}

func (m *Manifest) HostsKey() string {
	return m.keyRoot() + "hosts"
}
