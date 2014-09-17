package main

import (
	"fmt"
	"testing"

	"github.com/coreos/go-etcd/etcd"
)

func newCluster() Cluster {
	etcdCli := etcd.NewClient([]string{"http://127.0.0.1:4001"})
	return NewCluster(etcdCli)
}

func TestGetClusterIPs(t *testing.T) {
	c := newCluster()
	ips := c.GetClusterIPs()
	fmt.Println(ips)
}
