package main

import (
	"github.com/coreos/go-etcd/etcd"
)

type EtcdInterface interface {
	CompareAndSwap(key string, value string, ttl uint64, prevValue string, prevIndex uint64) (*etcd.Response, error)
	CompareAndDelete(key string, prevValue string, prevIndex uint64) (*etcd.Response, error)
	Create(key string, value string, ttl uint64) (*etcd.Response, error)
	Delete(key string, recursive bool) (*etcd.Response, error)
	DeleteDir(key string) (*etcd.Response, error)
	Get(key string, sort, recursive bool) (*etcd.Response, error)
	GetCluster() []string
	MarshalJSON() ([]byte, error)
	OpenCURL()
	Set(key string, value string, ttl uint64) (*etcd.Response, error)
	SyncCluster() bool
	UnmarshalJSON(b []byte) error
	Update(key string, value string, ttl uint64) (*etcd.Response, error)
	Watch(prefix string, waitIndex uint64, recursive bool, receiver chan *etcd.Response, stop chan bool) (*etcd.Response, error)
}
