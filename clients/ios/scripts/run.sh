#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLIENT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
SHARED_DIR="$(cd "${CLIENT_DIR}/../shared/flutter" && pwd)"

if [ "${PROTOS_IOS_TASK_ENTRY:-}" != "1" ] && [ "${PROTOS_ALLOW_DIRECT_CLIENT_SCRIPT:-}" != "1" ]; then
	printf 'Use task -t clients/ios/Taskfile.yml run or run:no-tunnel from the repo root.\n' >&2
	exit 2
fi

tunnel_args=()
flutter_args=()
while [ "$#" -gt 0 ]; do
	case "$1" in
		--with-tunnel)
			tunnel_args+=("$1")
			shift
			;;
		--without-tunnel|--no-tunnel)
			tunnel_args+=("$1")
			shift
			;;
		*)
			flutter_args+=("$1")
			shift
			;;
	esac
done

"${SHARED_DIR}/scripts/generate-protobuf.sh"
flutter --version >/dev/null
flutter pub get --directory "${CLIENT_DIR}"

(
	cd "${CLIENT_DIR}"
	"${SCRIPT_DIR}/with-tunnel-mode.sh" "${tunnel_args[@]}" -- flutter run "${flutter_args[@]}"
)
