#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

HOF="$("$ROOT_DIR/scripts/ensure-hof.sh")"

generate_proto_contract() {
  local cue_file="$1"
  local proto_file="$2"
  "$HOF" gen "$cue_file" -T "contracts/hof/proto.tmpl:contract=$proto_file" --no-format
}

generate_proto_contract contracts/proto/apic/v1/apic.cue apic/proto/apic.proto
generate_proto_contract contracts/proto/p2p/v1/app.cue internal/p2p/proto/app.proto
generate_proto_contract contracts/proto/p2p/v1/instance.cue internal/p2p/proto/instance.proto
generate_proto_contract contracts/proto/p2p/v1/peerdb.cue internal/p2p/proto/peerDB.proto
generate_proto_contract contracts/proto/p2p/v1/pinger.cue internal/p2p/proto/pinger.proto

protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative,require_unimplemented_servers=false \
  apic/proto/apic.proto \
  internal/p2p/proto/app.proto \
  internal/p2p/proto/instance.proto \
  internal/p2p/proto/peerDB.proto \
  internal/p2p/proto/pinger.proto
