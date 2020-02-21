#!/bin/bash

docker run \
       --rm \
       -ti \
       --privileged \
       -v "$PWD":/go/src/github.com/protosio/protos \
       -w /go/src/github.com/protosio/protos \
       -p 8080:8080 \
       --name protos \
       --hostname protos \
       golang:1.13 \
       /bin/bash -c "go mod tidy && go build -o bin/protosd cmd/protos/protos.go"
