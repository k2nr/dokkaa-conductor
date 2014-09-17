package main

import (
	"log"
	"net/url"
	"regexp"
	"strings"

	"github.com/coreos/go-etcd/etcd"
)

type Cluster interface {
	GetClusterIPs() []string
	HostLoadOrder(ip string) (int, error)
}

type cluster struct {
	etcd EtcdInterface
}

func NewCluster(e EtcdInterface) Cluster {
	return &cluster{
		etcd: e,
	}
}

func (c cluster) GetClusterIPs() []string {
	ips := []string{}
	c.etcd.SyncCluster()
	machines := c.etcd.GetCluster()
	for _, m := range machines {
		u, _ := url.Parse(m)
		ip := strings.Split(u.Host, ":")[0]
		ips = append(ips, ip)
	}
	return ips
}

func (c cluster) HostLoadOrder(ip string) (int, error) {
	hostRanks, err := c.getHostRanks()
	if err != nil {
		// in order to ignore /hosts key not found error
		// TODO: ignore only not found error
		return 0, nil
	}

	order := 0
	thisHostCnt := hostRanks[hostIP]
	ips := c.GetClusterIPs()
	for _, ip := range ips {
		if ip == hostIP {
			continue
		}
		n, ok := hostRanks[ip]
		if !ok {
			n = 0
		}
		if n < thisHostCnt {
			order++
		}
	}
	log.Printf("ranks: %d", hostRanks)
	log.Printf("order: %d", order)
	return order, nil
}

func (c cluster) getHostRanks() (map[string]int, error) {
	resp, err := c.etcd.Get("/hosts", false, true)
	if err != nil {
		return nil, err
	}

	hostRanks := map[string]int{}
	hostNodes := resp.Node.Nodes
	rHost, _ := regexp.Compile("/hosts/([^/]+)$")
	for _, node := range hostNodes {
		submatch := rHost.FindStringSubmatch(node.Key)
		host := submatch[1]
		var containersNode *etcd.Node
		for _, nn := range node.Nodes {
			if nn.Key == "/hosts/"+host+"/containers" {
				containersNode = nn
				break
			}
		}
		hostRanks[host] = len(containersNode.Nodes)
	}

	return hostRanks, nil
}
