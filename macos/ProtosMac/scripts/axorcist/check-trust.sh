#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

printf '%s\n' '{"commandId":"trust","command":"isProcessTrusted"}' | "${SCRIPT_DIR}/axorc.sh" --stdin
printf '%s\n' '{"commandId":"feature","command":"isAXFeatureEnabled"}' | "${SCRIPT_DIR}/axorc.sh" --stdin
