#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SHARED_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${SHARED_DIR}/../../.." && pwd)"
CORE_DIR="${REPO_ROOT}/core"
GO_TAGS="${GO_TAGS:-dolt_purego_zstd,gms_pure_go}"

build_macos() {
	local app_dir="$1"
	OUT_DIR="${app_dir}/macos/Runner/Frameworks"
	OUT_LIB="${OUT_DIR}/libprotos.dylib"

	mkdir -p "${OUT_DIR}"

	(
		cd "${CORE_DIR}"
		env CGO_ENABLED=1 go build \
			-tags "${GO_TAGS}" \
			-buildmode=c-shared \
			-o "${OUT_LIB}" \
			./cmd/protos-ffi-bridge
	)

	install_name_tool -id "@rpath/libprotos.dylib" "${OUT_LIB}" 2>/dev/null || true
	codesign --force --sign - "${OUT_LIB}" >/dev/null 2>&1 || true

	printf '%s\n' "${OUT_LIB}"
}

build_ios() {
	local app_dir="$1"
	local platform="${PLATFORM_NAME:-iphoneos}"
	local sdk="iphoneos"
	local min_flag="-miphoneos-version-min=14.0"
	if [[ "${platform}" == "iphonesimulator" ]]; then
		sdk="iphonesimulator"
		min_flag="-mios-simulator-version-min=14.0"
	fi

	local arch="${PROTOS_IOS_ARCH:-${CURRENT_ARCH:-arm64}}"
	if [[ "${arch}" == "undefined_arch" ]]; then
		arch="arm64"
	fi
	if [[ "${arch}" == *" "* ]]; then
		arch="${arch%% *}"
	fi

	local goarch="arm64"
	case "${arch}" in
		arm64) goarch="arm64" ;;
		x86_64) goarch="amd64" ;;
		*)
			printf 'unsupported iOS architecture: %s\n' "${arch}" >&2
			exit 1
			;;
	esac

	local sdk_path
	sdk_path="$(xcrun --sdk "${sdk}" --show-sdk-path)"
	local cc_path
	cc_path="$(xcrun --sdk "${sdk}" --find clang)"
	local out_dir="${app_dir}/ios/Runner/Frameworks/${platform}"
	local out_lib="${out_dir}/libprotos.a"
	local ios_go_tags="${IOS_GO_TAGS:-${GO_TAGS}}"

	mkdir -p "${out_dir}"

	(
		cd "${CORE_DIR}"
		env \
			GOOS=ios \
			GOARCH="${goarch}" \
			CGO_ENABLED=1 \
			CC="${cc_path}" \
			CGO_CFLAGS="${min_flag} -arch ${arch} -isysroot ${sdk_path}" \
			CGO_LDFLAGS="${min_flag} -arch ${arch} -isysroot ${sdk_path}" \
			go build \
				-tags "${ios_go_tags}" \
				-buildmode=c-archive \
				-o "${out_lib}" \
				./cmd/protos-ffi-bridge
	)

	printf '%s\n' "${out_lib}"
}

TARGET="${1:-macos}"
APP_DIR="${2:-${PROTOS_FLUTTER_APP_DIR:-$(pwd)}}"
APP_DIR="$(cd "${APP_DIR}" && pwd)"

case "${TARGET}" in
	macos) build_macos "${APP_DIR}" ;;
	ios) build_ios "${APP_DIR}" ;;
	*)
		printf 'usage: %s [macos|ios] [client-app-dir]\n' "$0" >&2
		exit 2
		;;
esac
