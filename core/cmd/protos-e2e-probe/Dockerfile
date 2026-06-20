FROM golang:1.26.4-alpine3.23 AS build

WORKDIR /src
COPY go.mod go.sum ./
COPY cmd/protos-e2e-probe ./cmd/protos-e2e-probe
RUN GOMAXPROCS=2 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/protos-e2e-probe ./cmd/protos-e2e-probe

FROM alpine:3.23
ARG PROTOS_E2E_PROBE_PAYLOAD_MIB=0
RUN apk add --no-cache ca-certificates
RUN if [ "$PROTOS_E2E_PROBE_PAYLOAD_MIB" -gt 0 ]; then \
      mkdir -p /opt/protos-e2e-payload && \
      dd if=/dev/urandom of=/opt/protos-e2e-payload/payload.bin bs=1M count="$PROTOS_E2E_PROBE_PAYLOAD_MIB"; \
    fi
COPY --from=build /out/protos-e2e-probe /usr/local/bin/protos-e2e-probe
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/protos-e2e-probe"]
