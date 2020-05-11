#!/bin/bash

[ -z "$PROTOS_FRONTEND_PATH" ] && echo "Please set environment variable PROTOS_FRONTEND_PATH" && exit 1;

docker network create protosnet &> /dev/null

docker run \
       --rm \
       -ti \
       --privileged \
       -v "$PWD/../":/go/src/github.com/protosio/protos \
       -v /opt/protos:/opt/protos \
       -v "$PROTOS_FRONTEND_PATH":/protosfrontend \
       -v /var/run/docker.sock:/var/run/docker.sock \
       -w /go/src/github.com/protosio/protos \
       -e GOFLAGS=-mod=vendor \
       -p 8080:8080 \
       -p 8443:8443 \
       --name protos \
       --hostname protos \
       --network protosnet \
       golang:1.14 \
       go run --race cmd/protosd/protos.go --loglevel debug --config configs/protos.dev.yaml --dev daemon
