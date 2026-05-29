#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

"${SCRIPT_DIR}/generate-protobuf.sh"
flutter --version >/dev/null
flutter pub get --directory "${APP_DIR}"
flutter build macos "$@"

CONFIG="Release"
for arg in "$@"; do
	case "${arg}" in
		--debug) CONFIG="Debug" ;;
		--profile) CONFIG="Profile" ;;
	esac
done

APP_BUNDLE="$(find "${APP_DIR}/build/macos/Build/Products/${CONFIG}" -maxdepth 1 -name '*.app' -print -quit)"
printf '%s\n' "${APP_BUNDLE}"

