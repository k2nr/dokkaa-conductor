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
	ID        string
	Container Container
}

func NewManifest(app, container, val string) *Manifest {
	var m Manifest
	json.Unmarshal([]byte(val), &m)
	m.ID = app
	m.Container.Name = container

	return &m
}
