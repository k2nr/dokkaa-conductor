package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const (
	backendsPortStart = 10000
)

type Port struct {
	HostPort      int
	ContainerPort int
}

type Srv struct {
	Port int
	Role string
}

type Container struct {
	Image    string
	Name     string
	Scale    int
	Env      map[string]string
	Links    []string
	Command  []string
	Services map[string]Srv
}

type Manifest struct {
	AppName       string
	ContainerName string
	Container     *Container
}

func NewManifest(app, container, val string) (*Manifest, error) {
	m := Manifest{
		AppName:       app,
		ContainerName: container,
	}
	var c Container
	err := json.Unmarshal([]byte(val), &c)
	if err != nil {
		return &m, err
	}
	m.Container = &c
	m.Container.Name = app + "---" + container
	if m.Container.Scale == 0 {
		m.Container.Scale = 1
	}
	m.Container.Env = map[string]string{}
	m.Container.Env["DOKKAA_APP_NAME"] = app
	for k, s := range m.Container.Services {
		m.Container.Env["DOKKAA_SERVICE_"+k] = strconv.Itoa(s.Port)
		if s.Role != "" {
			m.Container.Env["DOKKAA_ROLE_"+k] = s.Role
		}
	}
	for i, l := range m.Container.Links {
		port := backendsPortStart + i
		m.Container.Env[fmt.Sprintf("BACKENDS_%d", port)] = l + "." + app + ".skydns.local"
		s := "SERVICE_" + strings.ToUpper(l)
		m.Container.Env[s+"_ADDR"] = "backends"
		m.Container.Env[s+"_PORT"] = strconv.Itoa(port)
	}

	return &m, nil
}

func (m *Manifest) keyRoot() string {
	a := m.AppName
	c := m.ContainerName
	return "/apps/" + a + "/" + c + "/"
}

func (m *Manifest) ManifestKey() string {
	return m.keyRoot() + "manifest"
}

func (m *Manifest) HostsDirKey() string {
	return m.keyRoot() + "hosts"
}
