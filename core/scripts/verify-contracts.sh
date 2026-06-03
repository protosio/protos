#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

HOF="$("$ROOT_DIR/scripts/ensure-hof.sh")"
tmp_dir="$(mktemp -d ".tmp-contracts.XXXXXX")"
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

generate_proto_tmp() {
  local cue_file="$1"
  local output="$2"
  "$HOF" gen "$cue_file" -O "$tmp_dir" -T "contracts/hof/proto.tmpl:contract=$output" --no-format
}

generate_proto_tmp contracts/proto/apic/v1/apic.cue apic.proto
generate_proto_tmp contracts/proto/hostagent/v1/hostagent.cue hostagent.proto
generate_proto_tmp contracts/proto/p2p/v1/app.cue app.proto
generate_proto_tmp contracts/proto/p2p/v1/instance.cue instance.proto
generate_proto_tmp contracts/proto/p2p/v1/peerdb.cue peerDB.proto
generate_proto_tmp contracts/proto/p2p/v1/pinger.cue pinger.proto

"$HOF" gen internal/db/contracts/sql/protos/v0_0/contract.cue \
  -O "$tmp_dir" \
  -T internal/db/contracts/hof/sql.tmpl:contract=protos_01_tables_sql --no-format
"$HOF" gen internal/db/contracts/sql/protos/v0_0/contract.cue \
  -O "$tmp_dir" \
  -T internal/db/contracts/hof/sql_undo.tmpl:contract=protos_01_tables_undo_sql --no-format
"$HOF" gen internal/db/contracts/sql/protos/v0_0/contract.cue \
  -O "$tmp_dir" \
  -T internal/db/contracts/hof/go_sq_models.tmpl:contract=models.go --no-format
"$HOF" gen internal/db/contracts/sql/protos/v0_0/contract.cue \
  -O "$tmp_dir" \
  -T internal/db/contracts/hof/go_sql_version.tmpl:contract=protos_v0_0_gen.go --no-format
"$HOF" gen internal/db/contracts/sql/protos/catalog.cue \
  -O "$tmp_dir" \
  -T internal/db/contracts/hof/go_sql_catalog.tmpl:catalog=protos_catalog_gen.go --no-format

gofmt -w "$tmp_dir/models.go" "$tmp_dir/protos_v0_0_gen.go" "$tmp_dir/protos_catalog_gen.go"

diff -u apic/proto/apic.proto "$tmp_dir/apic.proto"
diff -u internal/hostagent/proto/hostagent.proto "$tmp_dir/hostagent.proto"
diff -u internal/p2p/proto/app.proto "$tmp_dir/app.proto"
diff -u internal/p2p/proto/instance.proto "$tmp_dir/instance.proto"
diff -u internal/p2p/proto/peerDB.proto "$tmp_dir/peerDB.proto"
diff -u internal/p2p/proto/pinger.proto "$tmp_dir/pinger.proto"
diff -u internal/db/migrations/protos_01_tables.sql "$tmp_dir/protos_01_tables_sql"
diff -u internal/db/migrations/protos_01_tables.undo.sql "$tmp_dir/protos_01_tables_undo_sql"
diff -u internal/db/models.go "$tmp_dir/models.go"
diff -u internal/db/contracts/sql/protos/v0_0/contract_gen.go "$tmp_dir/protos_v0_0_gen.go"
diff -u internal/db/contracts/sql/protos/catalog_gen.go "$tmp_dir/protos_catalog_gen.go"
