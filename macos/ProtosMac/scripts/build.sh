#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${APP_DIR}/../.." && pwd)"
SWIFT_PROTOBUF_VERSION="1.38.0"
GO_TAGS="${GO_TAGS:-dolt_purego_zstd,gms_pure_go}"
export MACOSX_DEPLOYMENT_TARGET="${MACOSX_DEPLOYMENT_TARGET:-15.0}"

mkdir -p "${APP_DIR}/.build/go"

env CGO_ENABLED=1 go build \
	-tags "${GO_TAGS}" \
	-buildmode=c-archive \
	-o "${APP_DIR}/.build/go/libprotos.a" \
	"${REPO_ROOT}/cmd/protos-macos-bridge"

TOOL_DIR="${APP_DIR}/.build/tools/swift-protobuf-${SWIFT_PROTOBUF_VERSION}"
if [[ ! -d "${TOOL_DIR}/.git" ]]; then
	rm -rf "${TOOL_DIR}"
	git clone --depth 1 --branch "${SWIFT_PROTOBUF_VERSION}" \
		https://github.com/apple/swift-protobuf.git \
		"${TOOL_DIR}"
fi
swift package --package-path "${APP_DIR}" config set-mirror \
	--original https://github.com/apple/swift-protobuf.git \
	--mirror "file://${TOOL_DIR}"

PROTOC_GEN_SWIFT="${PROTOC_GEN_SWIFT:-}"
if [[ -z "${PROTOC_GEN_SWIFT}" ]]; then
	PROTOC_GEN_SWIFT="$(command -v protoc-gen-swift || true)"
fi
if [[ -z "${PROTOC_GEN_SWIFT}" ]]; then
	swift build --package-path "${TOOL_DIR}" -c release --product protoc-gen-swift
	PROTOC_GEN_SWIFT="${TOOL_DIR}/.build/release/protoc-gen-swift"
fi

PATH="$(dirname "${PROTOC_GEN_SWIFT}"):${PATH}" protoc \
	-I "${REPO_ROOT}" \
	--swift_out="${APP_DIR}/Sources/ProtosMacApp/Generated" \
	"${REPO_ROOT}/apic/proto/apic.proto"

swift build --package-path "${APP_DIR}" "$@"
BIN_DIR="$(swift build --package-path "${APP_DIR}" --show-bin-path "$@")"

APP_BUNDLE="${APP_DIR}/.build/Protos.app"
rm -rf "${APP_BUNDLE}"
mkdir -p "${APP_BUNDLE}/Contents/MacOS" "${APP_BUNDLE}/Contents/Resources"
cp "${BIN_DIR}/ProtosMacApp" "${APP_BUNDLE}/Contents/MacOS/Protos"
cp "${APP_DIR}/Info.plist" "${APP_BUNDLE}/Contents/Info.plist"
codesign --force --sign - "${APP_BUNDLE}" >/dev/null 2>&1 || true

printf '%s\n' "${APP_BUNDLE}"
