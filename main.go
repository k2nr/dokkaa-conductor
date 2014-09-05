package main

import (
	"flag"
	"github.com/coreos/go-etcd/etcd"
	"log"
	"os"
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

func main() {
	flag.Parse()
	hostIP = getopt("HOST_IP", "127.0.0.1")
	etcdAddr := "http://" + getopt("ETCD_ADDR", "127.0.0.1:4001")
	etcdClient := etcd.NewClient([]string{etcdAddr})
	dockerClient, _ := NewDockerClient(getopt("DOCKER_HOST", "unix:///var/run/docker.sock"))
	scheduler := NewScheduler(dockerClient, etcdClient)
	register := NewRegister(dockerClient, etcdClient)

	q1 := scheduler.StartSchedulingLoop()
	q2 := register.StartDockerEventLoop()
	select {
	case <-q1:
	case <-q2:
	}
}
