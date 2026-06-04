#!/usr/bin/env bash
set -euo pipefail

usage() {
	printf 'usage: %s [--with-tunnel|--without-tunnel] -- <flutter build/run command>\n' "$0" >&2
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLIENT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
PROJECT_FILE="${CLIENT_DIR}/ios/Runner.xcodeproj/project.pbxproj"

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
	clean_no_tunnel_app_bundles
	perl -0pi -e '
		s/\n\t\t\t\tC0DE20312F00000100000001 \/\* Embed App Extensions \*\/,//;
		s/\n\t\t\t\tC0DE20812F00000100000001 \/\* PBXTargetDependency \*\/,//;
	' "${PROJECT_FILE}"
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
