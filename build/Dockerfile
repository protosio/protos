FROM golang:1.17.3-alpine3.14 AS build

WORKDIR /go/src/github.com
ENV GOPATH=/go PATH=$PATH:/go/bin
ADD ./src /go/src/github.com/protos
WORKDIR /go/src/github.com/protos
RUN apk --update add git
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-w -extldflags "-static"' -o bin/protosd cmd/protosd/protos.go
RUN mkdir /root/tmp


FROM alpine:3.14.3
WORKDIR /
RUN mkdir /opt/protos /var/protos /var/protos-containerd
COPY --from=build /go/src/github.com/protos/bin/protosd /opt/protos/protosd
COPY --from=build /go/src/github.com/protos/configs/protosd.yaml /opt/protos/protosd.yaml
RUN chmod +x /opt/protos/protosd

ENTRYPOINT ["/opt/protos/protosd", "--loglevel", "debug", "--config", "/opt/protos/protosd.yaml", "daemon"]
