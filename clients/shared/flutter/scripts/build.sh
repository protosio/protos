#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

printf 'Use task -t clients/macos/Taskfile.yml build or task -t clients/ios/Taskfile.yml build:no-tunnel from the repo root.\n' >&2
printf 'Shared package helpers live in %s.\n' "${SCRIPT_DIR}" >&2
exit 2
