#!/bin/bash

[ -z "$PROTOS_FRONTEND_PATH" ] && echo "Please set environment variable PROTOS_FRONTEND_PATH" && exit 1;

docker run \
       --rm \
       -ti \
       --privileged \
       -v "$PWD":/go/src/github.com/nustiueudinastea/protos \
       -v /opt/protos:/opt/protos \
       -v "$PROTOS_FRONTEND_PATH":/protosfrontend \
       -v /var/run/docker.sock:/var/run/docker.sock \
       -w /go/src/github.com/nustiueudinastea/protos \
       -p 8080:8080 \
       --name protos \
       --hostname protos \
       golang:1.9.4 \
       go run cmd/protos/protos.go --loglevel debug --config protos.dev.yaml daemon