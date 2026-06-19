#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SHARED_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
GENERATED_DIR="${SHARED_DIR}/lib/src/generated"

tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/protos-flutter-protobuf.XXXXXX")"
cleanup() {
	rm -rf "${tmp_dir}"
}
trap cleanup EXIT

PROTOS_FLUTTER_PROTO_OUT_DIR="${tmp_dir}" "${SCRIPT_DIR}/generate-protobuf.sh"

diff -ru "${GENERATED_DIR}/apic/proto" "${tmp_dir}/apic/proto"
