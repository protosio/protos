#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
TOOL_DIR="${APP_DIR}/.build/tools/AXorcist"
AXORC="${TOOL_DIR}/.build/release/axorc"

if [[ ! -d "${TOOL_DIR}" ]]; then
  echo "AXorcist checkout not found at ${TOOL_DIR}" >&2
  echo "Clone https://github.com/steipete/AXorcist.git there, then rerun this script." >&2
  exit 2
fi

if [[ ! -x "${AXORC}" ]]; then
  swift build --package-path "${TOOL_DIR}" -c release --product axorc >/dev/null
fi

exec "${AXORC}" "$@"
