#!/usr/bin/env bash
set -euo pipefail

HOF_VERSION="${HOF_VERSION:-v0.6.10}"
GOBIN_DIR="${GOBIN:-$(go env GOPATH)/bin}"
HOF_BIN="${HOF_BIN:-$GOBIN_DIR/hof}"

if command -v hof >/dev/null 2>&1; then
  command -v hof
  exit 0
fi
if [ -x "$HOF_BIN" ]; then
  printf '%s\n' "$HOF_BIN"
  exit 0
fi

mkdir -p "$GOBIN_DIR"
if GOBIN="$GOBIN_DIR" go install "github.com/hofstadter-io/hof/cmd/hof@$HOF_VERSION"; then
  printf '%s\n' "$HOF_BIN"
  exit 0
fi

tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/protos-hof.XXXXXX")"
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

git clone --depth 1 --branch "$HOF_VERSION" https://github.com/hofstadter-io/hof "$tmp_dir/hof" >/dev/null
cd "$tmp_dir/hof"

if [ -f script/runtime/exe_next.go ] && ! grep -q "func (nopTestDeps) ModulePath" script/runtime/exe_next.go; then
  cat >> script/runtime/exe_next.go <<'EOF'

func (nopTestDeps) ModulePath() string { return "" }
EOF
fi

go build -o "$HOF_BIN" ./cmd/hof
printf '%s\n' "$HOF_BIN"
