#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLIENT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
SHARED_DIR="$(cd "${CLIENT_DIR}/../shared/flutter" && pwd)"

"${SHARED_DIR}/scripts/generate-protobuf.sh"
flutter --version >/dev/null
flutter pub get --directory "${CLIENT_DIR}"

(
	cd "${CLIENT_DIR}"
	flutter build ios "$@"
)
