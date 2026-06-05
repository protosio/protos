#!/usr/bin/env bash
set -euo pipefail

usage() {
	printf 'usage: %s [--with-tunnel|--without-tunnel] -- <flutter build/run command>\n' "$0" >&2
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLIENT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
PROJECT_FILE="${CLIENT_DIR}/ios/Runner.xcodeproj/project.pbxproj"

if [ "${PROTOS_IOS_TASK_ENTRY:-}" != "1" ] && [ "${PROTOS_ALLOW_DIRECT_CLIENT_SCRIPT:-}" != "1" ]; then
	printf 'Use task -t clients/ios/Taskfile.yml build/run tasks from the repo root.\n' >&2
	exit 2
fi

tunnel_mode="${PROTOS_IOS_TUNNEL:-with}"
while [ "$#" -gt 0 ]; do
	case "$1" in
		--with-tunnel)
			tunnel_mode="with"
			shift
			;;
		--without-tunnel|--no-tunnel)
			tunnel_mode="without"
			shift
			;;
		--)
			shift
			break
			;;
		-h|--help)
			usage
			exit 0
			;;
		*)
			break
			;;
	esac
done

if [ "$#" -eq 0 ]; then
	usage
	exit 2
fi

case "${tunnel_mode}" in
	with|1|true|yes)
		dart_define_value="true"
		;;
	without|0|false|no)
		tunnel_mode="without"
		dart_define_value="false"
		;;
	*)
		printf 'unsupported tunnel mode: %s\n' "${tunnel_mode}" >&2
		exit 2
		;;
esac

clean_no_tunnel_app_bundles() {
	rm -rf \
		"${CLIENT_DIR}/build/ios/iphoneos/Protos.app" \
		"${CLIENT_DIR}/build/ios/Debug-iphoneos/Protos.app" \
		"${CLIENT_DIR}/build/ios/Profile-iphoneos/Protos.app" \
		"${CLIENT_DIR}/build/ios/Release-iphoneos/Protos.app" \
		"${CLIENT_DIR}/ios/build/Debug-iphoneos/Protos.app" \
		"${CLIENT_DIR}/ios/build/Profile-iphoneos/Protos.app" \
		"${CLIENT_DIR}/ios/build/Release-iphoneos/Protos.app"
}

assert_no_tunnel_runner_links() {
	if grep -Eq '^[[:space:]]*C0DE20312F00000100000001 /\* Embed App Extensions \*/,[[:space:]]*$' "${PROJECT_FILE}"; then
		printf 'no-tunnel build still embeds the Packet Tunnel extension\n' >&2
		exit 1
	fi
	if grep -Eq '^[[:space:]]*C0DE20812F00000100000001 /\* PBXTargetDependency \*/,[[:space:]]*$' "${PROJECT_FILE}"; then
		printf 'no-tunnel build still depends on the Packet Tunnel target\n' >&2
		exit 1
	fi
}

backup_file=""
restore_project() {
	if [ -n "${backup_file}" ] && [ -f "${backup_file}" ]; then
		cp "${backup_file}" "${PROJECT_FILE}"
		rm -f "${backup_file}"
	fi
}

cleanup_and_restore() {
	restore_project
}

if [ "${tunnel_mode}" = "without" ]; then
	backup_file="$(mktemp)"
	cp "${PROJECT_FILE}" "${backup_file}"
	trap cleanup_and_restore EXIT
	if [ "${PROTOS_IOS_SKIP_NO_TUNNEL_CLEAN:-}" != "1" ]; then
		clean_no_tunnel_app_bundles
	fi
	perl -0pi -e '
		s/\n\t\t\t\tC0DE20312F00000100000001 \/\* Embed App Extensions \*\/,//;
		s/\n\t\t\t\tC0DE20812F00000100000001 \/\* PBXTargetDependency \*\/,//;
	' "${PROJECT_FILE}"
	assert_no_tunnel_runner_links
fi

cmd=("$@")
has_tunnel_define=0
for arg in "${cmd[@]}"; do
	case "${arg}" in
		--dart-define=PROTOS_IOS_TUNNEL=*)
			has_tunnel_define=1
			;;
	esac
done
if [ "${has_tunnel_define}" -eq 0 ]; then
	cmd+=("--dart-define=PROTOS_IOS_TUNNEL=${dart_define_value}")
fi

"${cmd[@]}"
