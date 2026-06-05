#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLIENT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
SHARED_DIR="$(cd "${CLIENT_DIR}/../shared/flutter" && pwd)"

if [ "${PROTOS_IOS_TASK_ENTRY:-}" != "1" ] && [ "${PROTOS_ALLOW_DIRECT_CLIENT_SCRIPT:-}" != "1" ]; then
	printf 'Use task -t clients/ios/Taskfile.yml build or build:no-tunnel from the repo root.\n' >&2
	exit 2
fi

tunnel_args=()
flutter_args=()
tunnel_mode="with"
while [ "$#" -gt 0 ]; do
	case "$1" in
		--with-tunnel)
			tunnel_mode="with"
			tunnel_args+=("$1")
			shift
			;;
		--without-tunnel|--no-tunnel)
			tunnel_mode="without"
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

normalize_no_tunnel_app_bundle() {
	mode="release"
	for arg in "${flutter_args[@]}"; do
		case "${arg}" in
			--debug) mode="debug" ;;
			--profile) mode="profile" ;;
			--release) mode="release" ;;
		esac
	done
	case "${mode}" in
		debug) app_dir="${CLIENT_DIR}/build/ios/Debug-iphoneos/Protos.app" ;;
		profile) app_dir="${CLIENT_DIR}/build/ios/Profile-iphoneos/Protos.app" ;;
		release) app_dir="${CLIENT_DIR}/build/ios/Release-iphoneos/Protos.app" ;;
		*) printf 'unsupported iOS build mode: %s\n' "${mode}" >&2; exit 2 ;;
	esac
	expected_app_dir="${CLIENT_DIR}/build/ios/iphoneos/Protos.app"
	if [ ! -d "${app_dir}" ]; then
		printf 'built application bundle not found at %s\n' "${app_dir}" >&2
		exit 1
	fi
	rm -rf "${expected_app_dir}"
	mkdir -p "$(dirname "${expected_app_dir}")"
	cp -R "${app_dir}" "${expected_app_dir}"
}

(
	cd "${CLIENT_DIR}"
	if [ "${tunnel_mode}" = "without" ]; then
		log_file="$(mktemp "${TMPDIR:-/tmp}/protos-ios-build.XXXXXX.log")"
		set +e
		"${SCRIPT_DIR}/with-tunnel-mode.sh" "${tunnel_args[@]}" -- flutter build ios "${flutter_args[@]}" 2>&1 | tee "${log_file}"
		status="${PIPESTATUS[0]}"
		set -e
		if [ "${status}" -ne 0 ]; then
			if grep -q 'Build succeeded but the expected app at .* not found' "${log_file}"; then
				normalize_no_tunnel_app_bundle
				rm -f "${log_file}"
				exit 0
			fi
			rm -f "${log_file}"
			exit "${status}"
		fi
		normalize_no_tunnel_app_bundle
		rm -f "${log_file}"
		exit 0
	fi
	"${SCRIPT_DIR}/with-tunnel-mode.sh" "${tunnel_args[@]}" -- flutter build ios "${flutter_args[@]}"
)
