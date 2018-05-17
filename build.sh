#!/bin/bash

docker run \
       --rm \
       -ti \
       --privileged \
       -v "$PWD":/go/src/github.com/nustiueudinastea/protos \
       -w /go/src/github.com/nustiueudinastea/protos \
       -p 8080:8080 \
       --name protos \
       --hostname protos \
       golang:1.9.4 \
       go build -o bin/protos cmd/protos/protos.go