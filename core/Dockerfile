# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.26.6-alpine3.23 AS build

RUN apk add --no-cache ca-certificates git openssh-client

ARG PROTOS_BUILD_DIR=.
ARG PROTOS_NATIVE_GO_TAGS=dolt_purego_zstd,gms_pure_go
ARG PROTOS_BUILD_GOMAXPROCS=2
ARG TARGETOS
ARG TARGETARCH
WORKDIR /go/src/github.com
ENV GOPATH=/go \
    PATH=$PATH:/go/bin \
    GIT_TERMINAL_PROMPT=0 \
    GOPRIVATE=github.com/nustiueudinastea/swarmion \
    GONOSUMDB=github.com/nustiueudinastea/swarmion
RUN mkdir -p -m 0700 /root/.ssh \
    && ssh-keyscan github.com >> /root/.ssh/known_hosts
ADD . /go/src/github.com/protos
WORKDIR /go/src/github.com/protos/${PROTOS_BUILD_DIR}
RUN --mount=type=ssh,required=false \
    --mount=type=secret,id=github_netrc,target=/root/.netrc,mode=0600,required=false \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    if [ ! -s /root/.netrc ] && [ -n "${SSH_AUTH_SOCK:-}" ]; then \
      git config --global url."ssh://git@github.com/".insteadOf "https://github.com/"; \
    fi; \
    target_os="${TARGETOS:-linux}"; \
    target_arch="${TARGETARCH:-$(go env GOARCH)}"; \
    GOMAXPROCS=${PROTOS_BUILD_GOMAXPROCS} CGO_ENABLED=0 GOOS="${target_os}" GOARCH="${target_arch}" \
      go build -tags "${PROTOS_NATIVE_GO_TAGS}" -ldflags '-w' -o bin/protosd cmd/protosd/protosd.go
RUN mkdir /root/tmp


FROM alpine:3.23
WORKDIR /
RUN apk add --no-cache ca-certificates \
	&& mkdir /opt/protos /var/protos /var/protos-containerd
ARG PROTOS_BUILD_DIR=.
COPY --from=build /go/src/github.com/protos/${PROTOS_BUILD_DIR}/bin/protosd /opt/protos/protosd
COPY --from=build /go/src/github.com/protos/${PROTOS_BUILD_DIR}/configs/protosd.yaml /opt/protos/protosd.yaml
RUN cat > /opt/protos/entrypoint.sh <<'EOF'
#!/bin/sh
set -u

mkdir -p /var/log /var/lib/protos/tmp
log_file=/var/log/protos.log
fifo="$(mktemp -u /tmp/protos-log.XXXXXX)"
mkfifo "$fifo"

console_sink=""
for candidate in /dev/console /dev/hvc0 /dev/ttyS0; do
  if [ -e "$candidate" ] && (printf '' > "$candidate") 2>/dev/null; then
    console_sink="$candidate"
    break
  fi
done

if [ -n "$console_sink" ]; then
  tee -a "$log_file" "$console_sink" < "$fifo" >/dev/null &
else
  tee -a "$log_file" < "$fifo" &
fi
tee_pid=$!

/opt/protos/protosd --loglevel debug --config /opt/protos/protosd.yaml > "$fifo" 2>&1
status=$?

wait "$tee_pid" >/dev/null 2>&1 || true
rm -f "$fifo"
exit "$status"
EOF
RUN chmod +x /opt/protos/protosd /opt/protos/entrypoint.sh

ENTRYPOINT ["/opt/protos/entrypoint.sh"]
