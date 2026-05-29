#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${APP_DIR}/../.." && pwd)"
OUT_DIR="${APP_DIR}/lib/src/generated"

mkdir -p "${OUT_DIR}"

if ! command -v protoc-gen-dart >/dev/null 2>&1; then
	if [[ -x "${HOME}/.pub-cache/bin/protoc-gen-dart" ]]; then
		export PATH="${HOME}/.pub-cache/bin:${PATH}"
	else
		dart pub global activate protoc_plugin
		export PATH="${HOME}/.pub-cache/bin:${PATH}"
	fi
fi

protoc \
	-I "${REPO_ROOT}" \
	--dart_out="${OUT_DIR}" \
	"${REPO_ROOT}/apic/proto/apic.proto"

