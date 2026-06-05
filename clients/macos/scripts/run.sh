#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLIENT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${CLIENT_DIR}/../.." && pwd)"

if [ "${PROTOS_MACOS_TASK_ENTRY:-}" != "1" ] && [ "${PROTOS_ALLOW_DIRECT_CLIENT_SCRIPT:-}" != "1" ]; then
	echo "Use task -t clients/macos/Taskfile.yml run from the repo root." >&2
	exit 2
fi

if pgrep -f '/ProtosFlutter.app/Contents/MacOS/ProtosFlutter' >/dev/null; then
	echo "Protos macOS app is already running. Stop it before starting another copy." >&2
	exit 1
fi
if pgrep -f 'flutter run -d macos' >/dev/null; then
	echo "A macOS Flutter run is already active. Stop it before starting another copy." >&2
	exit 1
fi

export PROTOS_FLUTTER_DATA_DIR="${HOME}/.protos"
export PROTOS_FLUTTER_CONFIG_FILE="${HOME}/.protos/protos.yaml"
export PROTOS_FLUTTER_CAPABILITIES="${PROTOS_FLUTTER_CAPABILITIES:-default}"
export PROTOS_FLUTTER_LOG_LEVEL="${PROTOS_FLUTTER_LOG_LEVEL:-info}"
export PROTOS_HOSTAGENT_BIN="${PROTOS_HOSTAGENT_BIN:-${REPO_ROOT}/bin/protos-hostagent}"

mkdir -p "${PROTOS_FLUTTER_DATA_DIR}"
chmod 700 "${PROTOS_FLUTTER_DATA_DIR}"

flutter pub get --directory "${CLIENT_DIR}" >/dev/null
(
	cd "${CLIENT_DIR}"
	log_file="$(mktemp "${TMPDIR:-/tmp}/protos-macos-run.XXXXXX")"
	set +e
	flutter run -d macos "$@" 2>&1 | tee "${log_file}"
	status="${PIPESTATUS[0]}"
	set -e
	if grep -q 'Protos macOS startup failed:' "${log_file}"; then
		rm -f "${log_file}"
		exit 1
	fi
	rm -f "${log_file}"
	exit "${status}"
)
