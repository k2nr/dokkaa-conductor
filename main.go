package main

import (
	"flag"
	"github.com/coreos/go-etcd/etcd"
	"log"
	"os"
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
	etcdClient := etcd.NewClient([]string{"http://127.0.0.1:4001"})
	dockerClient, _ := NewDockerClient(os.Getenv("DOCKER_HOST"))
	scheduler := NewScheduler(dockerClient, etcdClient)

	recv := make(chan *etcd.Response)
	go etcdClient.Watch("/apps/", 0, true, recv, nil)
	for n := range recv {
		if n != nil {
			val := n.Node.Value
			m := NewManifest(val)
			log.Printf("%+v\n", m)
			err := scheduler.Schedule(&m)
			assert(err)
		}
	}
}
