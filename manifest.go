package main

import (
	"fmt"
	"strings"
	"strconv"
	"encoding/json"
)

const (
	backendsPortStart = 10000
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
	Services map[string]int
}

type Manifest struct {
	AppName       string
	ContainerName string
	Container     Container
}

func NewManifest(app, container, val string) *Manifest {
	m := Manifest{
		AppName:       app,
		ContainerName: container,
	}
	var c Container
	err := json.Unmarshal([]byte(val), &c)
	if err == nil {
		m.Container = c
		m.Container.Name = app + "---" + container
		m.Container.Env = map[string]string{}
		m.Container.Env["DOKKAA_APP_NAME"] = app
		for k, v := range m.Container.Services {
			m.Container.Env["DOKKAA_SERVICE_" + k] = strconv.Itoa(v)
		}
		for i, l := range m.Container.Links {
			port := backendsPortStart + i
			m.Container.Env[fmt.Sprintf("BACKENDS_%d", port)] = l + "." + app + ".skydns.local"
			m.Container.Env[strings.ToUpper(l) + "_ADDR"] = "backends"
			m.Container.Env[strings.ToUpper(l) + "_PORT"] = strconv.Itoa(port)
		}
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
