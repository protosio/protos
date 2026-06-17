FROM golang:1.26.4-alpine3.23 AS build

WORKDIR /src
COPY go.mod go.sum ./
COPY cmd/protos-e2e-probe ./cmd/protos-e2e-probe
RUN GOMAXPROCS=2 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/protos-e2e-probe ./cmd/protos-e2e-probe

FROM alpine:3.23
RUN apk add --no-cache ca-certificates
COPY --from=build /out/protos-e2e-probe /usr/local/bin/protos-e2e-probe
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/protos-e2e-probe"]
