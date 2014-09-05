package main

import (
	"flag"
	"github.com/coreos/go-etcd/etcd"
	"log"
	"os"
)

var hostIP = "localhost"

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
	etcdAddr := "http://" + os.Getenv("ETCD_ADDR")
	if len(etcdAddr) == 0 {
		etcdAddr = "http://127.0.0.1:4001"
	}
	etcdClient := etcd.NewClient([]string{etcdAddr})
	dockerClient, _ := NewDockerClient(os.Getenv("DOCKER_HOST"))
	scheduler := NewScheduler(dockerClient, etcdClient)
	register := NewRegister(dockerClient, etcdClient)

	q1 := scheduler.StartSchedulingLoop()
	q2 := register.StartDockerEventLoop()
	select {
	case <-q1:
	case <-q2:
	}
}
