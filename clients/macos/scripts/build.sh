#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLIENT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
SHARED_DIR="$(cd "${CLIENT_DIR}/../shared/flutter" && pwd)"

if [ "${PROTOS_MACOS_TASK_ENTRY:-}" != "1" ] && [ "${PROTOS_ALLOW_DIRECT_CLIENT_SCRIPT:-}" != "1" ]; then
	printf 'Use task -t clients/macos/Taskfile.yml build from the repo root.\n' >&2
	exit 2
fi

"${SHARED_DIR}/scripts/generate-protobuf.sh"
flutter --version >/dev/null
flutter pub get --directory "${CLIENT_DIR}"

(
	cd "${CLIENT_DIR}"
	flutter build macos "$@"
)

CONFIG="Release"
for arg in "$@"; do
	case "${arg}" in
		--debug) CONFIG="Debug" ;;
		--profile) CONFIG="Profile" ;;
	esac
done

APP_BUNDLE="$(find "${CLIENT_DIR}/build/macos/Build/Products/${CONFIG}" -maxdepth 1 -name '*.app' -print -quit)"
printf '%s\n' "${APP_BUNDLE}"
