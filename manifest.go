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
	Scale    int
	Env      map[string]string
	Links    []string
	Command  []string
	Dns      []string
	Hostname string
	Ports    []int
}

type Manifest struct {
	Id         string
	Containers map[string]Container
}

func NewManifest(s string) Manifest {
	var m Manifest
	json.Unmarshal([]byte(s), &m)
	return m
}
