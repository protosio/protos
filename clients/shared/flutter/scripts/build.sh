#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

printf 'Use clients/macos/scripts/build.sh or clients/ios/scripts/build.sh from the repo root.\n' >&2
printf 'Shared package helpers live in %s.\n' "${SCRIPT_DIR}" >&2
exit 2
