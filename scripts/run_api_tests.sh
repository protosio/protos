#!/usr/bin/env bash
set -euo pipefail

env \
	CGO_ENABLED="${CGO_ENABLED:-0}" \
	GOFLAGS="${GOFLAGS:--tags=dolt_purego_zstd,gms_pure_go}" \
	go test ./api_tests
