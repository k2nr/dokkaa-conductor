package main

import (
	"flag"
	"log"
	"os"

	"github.com/coreos/go-etcd/etcd"
)

var (
	hostIP string
)

func getopt(name, def string) string {
	if env := os.Getenv(name); env != "" {
		return env
	}
	return def
}

func assert(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func newEtcdClient() EtcdInterface {
	etcdAddr := "http://" + getopt("ETCD_ADDR", "127.0.0.1:4001")
	return etcd.NewClient([]string{etcdAddr})
}

func newDockerClient() DockerInterface {
	dockerClient, _ := NewDockerClient(getopt("DOCKER_HOST", "unix:///var/run/docker.sock"))
	return dockerClient
}

func main() {
	flag.Parse()
	hostIP = getopt("HOST_IP", "127.0.0.1")
	scheduler := NewScheduler(newDockerClient(), newEtcdClient())
	register := NewRegister(newDockerClient(), newEtcdClient())

	q1 := scheduler.StartSchedulingLoop()
	q2 := register.StartDockerEventLoop()
	select {
	case <-q1:
	case <-q2:
	}
}
