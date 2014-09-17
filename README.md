[![Build Status](https://travis-ci.org/k2nr/dokkaa-conductor.svg?branch=master)](https://travis-ci.org/k2nr/dokkaa-conductor)

# Dokkaa Conductor

This is a part of [Dokkaa project](https://github.com/k2nr/dokkaa).
dokkaa-conductor is docker cluster management tool powered by [etcd](https://github.com/coreos/etcd).

# Installation

Mostly you don't use dokkaa-conductor directly.
Use [dokkaacfg](https://github.com/k2nr/dokkaacfg) to launch all dokkaa environment.

For those who want to use dokkaa-conductor solely, here is the usage.

```
$ docker run --name conductor -v /var/run/docker.sock:/var/run/docker.sock -e HOST_IP=<host public IP> -e DOCKER_HOST=unix:///var/run/docker.sock -e ETCD_ADDR=<etcd IP>:4001 k2nr/dokkaa-conductor
```

# How It Works

dokkaa-conductor watches etcd and run/stop docker container, announce service using [skydns](https://github.com/skynetservices/skydns).

# Contributing

# License

MIT
