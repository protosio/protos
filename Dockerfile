FROM golang:1.13 AS build

WORKDIR /go/src/github.com
ENV GOPATH=/go PATH=$PATH:/go/bin
RUN git clone --single-branch --branch master https://github.com/protosio/protos.git
WORKDIR /go/src/github.com/protos
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-w -extldflags "-static"' -o bin/protosd cmd/protos/protos.go
RUN mkdir /root/tmp


FROM alpine:3.10.3
WORKDIR /
COPY --from=build /go/src/github.com/protos/bin/protosd /usr/local/bin/protosd
RUN chmod +x /usr/local/bin/protosd
RUN mkdir /var/protos && mkdir /var/protos-containerd
COPY protos.yaml /etc/protos.yaml

ENTRYPOINT ["/usr/local/bin/protosd", "--loglevel", "debug", "--config", "/etc/protos.yaml", "init"]
