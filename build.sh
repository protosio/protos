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
       golang:1.9.4 \
       /bin/bash -c "go get -u github.com/golang/dep/cmd/dep && dep ensure && go build -o bin/protos cmd/protos/protos.go"
