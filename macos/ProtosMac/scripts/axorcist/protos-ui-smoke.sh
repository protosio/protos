#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
OUT_DIR="${APP_DIR}/.build/axorcist-smoke"
mkdir -p "${OUT_DIR}"

trust_json="$(printf '%s' '{"commandId":"trust","command":"isProcessTrusted"}' | "${SCRIPT_DIR}/axorc.sh" --stdin)"
echo "${trust_json}" | tee "${OUT_DIR}/trust.json" >/dev/null
if [[ "${trust_json}" != *'"trusted":true'* ]]; then
  echo "AXorcist is not trusted for Accessibility." >&2
  exit 2
fi

feature_json="$(printf '%s' '{"commandId":"feature","command":"isAXFeatureEnabled"}' | "${SCRIPT_DIR}/axorc.sh" --stdin)"
echo "${feature_json}" | tee "${OUT_DIR}/feature.json" >/dev/null
if [[ "${feature_json}" != *'"enabled":true'* ]]; then
  echo "macOS Accessibility API is not enabled." >&2
  exit 2
fi

printf '%s' '{"commandId":"tree","command":"collectAll","application":"io.protos.macos","attributes":["AXRole","AXTitle","AXDescription","AXValue","AXEnabled","AXHelp","AXPlaceholderValue"],"maxDepth":6}' \
  | "${SCRIPT_DIR}/axorc.sh" --stdin > "${OUT_DIR}/axorc-tree.json"

"${SCRIPT_DIR}/protos-ui-smoke.swift" walk-sidebar | tee "${OUT_DIR}/walk-sidebar.log"
"${SCRIPT_DIR}/protos-ui-smoke.swift" network-negative | tee "${OUT_DIR}/network-negative.log"
"${SCRIPT_DIR}/protos-ui-smoke.swift" dump > "${OUT_DIR}/final-tree.txt"

echo "AXorcist smoke artifacts: ${OUT_DIR}"
