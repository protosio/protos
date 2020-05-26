FROM golang:1.14.3 AS build

WORKDIR /go/src/github.com
ENV GOPATH=/go PATH=$PATH:/go/bin
ADD ./src /go/src/github.com/protos
WORKDIR /go/src/github.com/protos
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-w -extldflags "-static"' -o bin/protosd cmd/protosd/protos.go
RUN mkdir /root/tmp


FROM alpine:3.10.5
WORKDIR /
COPY --from=build /go/src/github.com/protos/bin/protosd /usr/local/bin/protosd
COPY --from=build /go/src/github.com/protos/configs/protosd.yaml /etc/protosd.yaml
RUN chmod +x /usr/local/bin/protosd
RUN mkdir /var/protos && mkdir /var/protos-containerd

ENTRYPOINT ["/usr/local/bin/protosd", "--loglevel", "debug", "--config", "/etc/protos.yaml", "daemon"]
