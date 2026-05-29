#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${APP_DIR}/../.." && pwd)"
OUT_DIR="${APP_DIR}/macos/Runner/Frameworks"
OUT_LIB="${OUT_DIR}/libprotos.dylib"
GO_TAGS="${GO_TAGS:-dolt_purego_zstd,gms_pure_go}"

mkdir -p "${OUT_DIR}"

env CGO_ENABLED=1 go build \
	-tags "${GO_TAGS}" \
	-buildmode=c-shared \
	-o "${OUT_LIB}" \
	"${REPO_ROOT}/cmd/protos-macos-bridge"

install_name_tool -id "@rpath/libprotos.dylib" "${OUT_LIB}" 2>/dev/null || true
codesign --force --sign - "${OUT_LIB}" >/dev/null 2>&1 || true

printf '%s\n' "${OUT_LIB}"

