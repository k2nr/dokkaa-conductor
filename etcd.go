package main

import (
	"github.com/coreos/go-etcd/etcd"
	"log"
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

type EtcdWatcher interface {
	Watch(prefix string, recursive bool) chan *etcd.Response
}

type etcdWatcher struct {
	client EtcdInterface
}

func NewEtcdWatcher(cli EtcdInterface) EtcdWatcher {
	return &etcdWatcher{
		client: cli,
	}
}

func (w *etcdWatcher) Watch(prefix string, recursive bool) chan *etcd.Response {
	wrapRecv := make(chan *etcd.Response)

	go func() {
	LOOP1:
		for {
			recv := make(chan *etcd.Response)
			stop := make(chan bool)

			go w.client.Watch(prefix, 0, recursive, recv, stop)

		LOOP2:
			for {
				select {
				case _, ok := <-wrapRecv:
					if !ok {
						break LOOP1
					}
				case r, ok := <-recv:
					if !ok {
						log.Println("watching loop ended. reconnecting.")
						close(stop)
						break LOOP2
					}
					if r != nil {
						wrapRecv <- r
					}
				}
			}
		}
	}()

	return wrapRecv
}
