package main

import (
	"testing"
	"os"
	"fmt"
	"github.com/coreos/go-etcd/etcd"
)

func newScheduler() Scheduler {
	dockerCli,_ := NewDockerClient(os.Getenv("DOCKER_HOST"))
	etcdCli := etcd.NewClient([]string{"http://127.0.0.1:4001"})
	return NewScheduler(dockerCli, etcdCli)
}

func TestGetClusterIPs(t *testing.T) {
	s := newScheduler()
	ips := s.GetClusterIPs()
	fmt.Println(ips)
}
