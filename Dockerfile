FROM golang:1.26.3-alpine3.23 AS build

ARG PROTOS_BUILD_DIR=.
ARG PROTOS_NATIVE_GO_TAGS=dolt_purego_zstd,gms_pure_go
WORKDIR /go/src/github.com
ENV GOPATH=/go PATH=$PATH:/go/bin
ADD . /go/src/github.com/protos
WORKDIR /go/src/github.com/protos/${PROTOS_BUILD_DIR}
RUN CGO_ENABLED=0 GOOS=linux go build -tags "${PROTOS_NATIVE_GO_TAGS}" -ldflags '-w' -o bin/protosd cmd/protosd/protosd.go
RUN mkdir /root/tmp


FROM alpine:3.23
WORKDIR /
RUN apk add --no-cache ca-certificates \
	&& mkdir /opt/protos /var/protos /var/protos-containerd
ARG PROTOS_BUILD_DIR=.
COPY --from=build /go/src/github.com/protos/${PROTOS_BUILD_DIR}/bin/protosd /opt/protos/protosd
COPY --from=build /go/src/github.com/protos/${PROTOS_BUILD_DIR}/configs/protosd.yaml /opt/protos/protosd.yaml
RUN chmod +x /opt/protos/protosd

ENTRYPOINT ["/opt/protos/protosd", "--loglevel", "debug", "--config", "/opt/protos/protosd.yaml", "daemon"]
