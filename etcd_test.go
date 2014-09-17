package main

import "github.com/coreos/go-etcd/etcd"

type etcdMock struct {
	watchChan chan *etcd.Response
}

func (e *etcdMock) CreateInOrder(dir string, value string, ttl uint64) (*etcd.Response, error) {
	panic("")
}

func (e *etcdMock) Delete(key string, recursive bool) (*etcd.Response, error) {
	panic("")
}

func (e *etcdMock) Get(key string, sort, recursive bool) (*etcd.Response, error) {
	panic("")
}

func (e *etcdMock) GetCluster() []string {
	panic("")
}

func (e etcdMock) Set(key string, value string, ttl uint64) (*etcd.Response, error) {
	panic("")
}

func (e etcdMock) SyncCluster() bool {
	panic("")
}

func (e etcdMock) Watch(prefix string, waitIndex uint64, recursive bool, receiver chan *etcd.Response, stop chan bool) (*etcd.Response, error) {
	panic("")
}
