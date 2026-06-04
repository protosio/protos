#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLIENT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
SHARED_DIR="$(cd "${CLIENT_DIR}/../shared/flutter" && pwd)"

tunnel_args=()
flutter_args=()
while [ "$#" -gt 0 ]; do
	case "$1" in
		--with-tunnel|--without-tunnel|--no-tunnel)
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
