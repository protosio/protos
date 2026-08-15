#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

(
  cd contracts/proto/history
  shasum -a 256 -c SHA256SUMS
)

HOF="$("$ROOT_DIR/scripts/ensure-hof.sh")"

cue_json_value() {
  local cue_file="$1"
  local expression="$2"
  "$HOF" eval "$cue_file" -e "$expression" --out json | tr -d '[:space:]'
}

cue_schema_version() {
  local cue_file="$1"
  local major
  local minor
  major="$(cue_json_value "$cue_file" 'lineage.schemas[0].version[0]')"
  minor="$(cue_json_value "$cue_file" 'lineage.schemas[0].version[1]')"
  printf '"%s.%s"' "$major" "$minor"
}

require_contract_value() {
  local description="$1"
  local actual="$2"
  local expected="$3"
  if [ "$actual" != "$expected" ]; then
    printf '%s: expected %s, got %s\n' "$description" "$expected" "$actual" >&2
    exit 1
  fi
}

verify_breaking_proto_transition() {
  local label="$1"
  local archived="$2"
  local current="$3"
  local expected_lineage="$4"
  local expected_from="$5"
  local expected_to="$6"

  local archived_lineage_id
  local archived_lineage_name
  local archived_to
  local current_lineage_id
  local current_lineage_name
  local current_from
  local current_to

  archived_lineage_id="$(cue_json_value "$archived" contract.migration.lineage_id)"
  archived_lineage_name="$(cue_json_value "$archived" lineage.name)"
  archived_to="$(cue_json_value "$archived" contract.migration.to_version)"
  current_lineage_id="$(cue_json_value "$current" contract.migration.lineage_id)"
  current_lineage_name="$(cue_json_value "$current" lineage.name)"
  current_from="$(cue_json_value "$current" contract.migration.from_version)"
  current_to="$(cue_json_value "$current" contract.migration.to_version)"

  require_contract_value "$label archived migration lineage" "$archived_lineage_id" "\"$expected_lineage\""
  require_contract_value "$label archived schema lineage" "$archived_lineage_name" "$archived_lineage_id"
  require_contract_value "$label current migration lineage" "$current_lineage_id" "$archived_lineage_id"
  require_contract_value "$label current schema lineage" "$current_lineage_name" "$current_lineage_id"
  require_contract_value "$label archived transition target" "$archived_to" "\"$expected_from\""
  require_contract_value "$label archived schema version" "$(cue_schema_version "$archived")" "$archived_to"
  require_contract_value "$label current transition source" "$current_from" "$archived_to"
  require_contract_value "$label current transition target" "$current_to" "\"$expected_to\""
  require_contract_value "$label current schema version" "$(cue_schema_version "$current")" "$current_to"
  require_contract_value "$label transition compatibility" "$(cue_json_value "$current" contract.migration.compatibility)" '"breaking"'
  require_contract_value "$label backward compatibility" "$(cue_json_value "$current" contract.migration.backward_compatible)" 'false'
  require_contract_value "$label forward compatibility" "$(cue_json_value "$current" contract.migration.forward_compatible)" 'false'
  require_contract_value "$label migration lenses" "$(cue_json_value "$current" lineage.lenses)" '[]'
}

verify_breaking_proto_transition \
  "APIC protobuf" \
  contracts/proto/history/apic/v0_0/apic.cue \
  contracts/proto/apic/v1/apic.cue \
  protos.client_api \
  0.0 \
  0.1
verify_breaking_proto_transition \
  "P2P instance protobuf" \
  contracts/proto/history/p2p_instance/v0_0/instance.cue \
  contracts/proto/p2p/v1/instance.cue \
  protos.p2p.instance \
  0.0 \
  0.1

tmp_dir="$(mktemp -d ".tmp-contracts.XXXXXX")"
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

expected_db_migrations="internal/db/migrations/protos_01_tables.sql
internal/db/migrations/protos_01_tables.undo.sql
internal/db/migrations/protos_02_instance_lifecycle_owner.sql
internal/db/migrations/protos_02_instance_lifecycle_owner.undo.sql"
actual_db_migrations="$(find internal/db/migrations -maxdepth 1 -type f -name '*.sql' | sort)"
if [ "$actual_db_migrations" != "$expected_db_migrations" ]; then
  {
    echo "DB migrations differ from the reviewed migration lineage."
    echo "Expected:"
    echo "$expected_db_migrations"
    echo "Actual:"
    echo "$actual_db_migrations"
  } >&2
  exit 1
fi

generate_proto_tmp() {
  local cue_file="$1"
  local output="$2"
  "$HOF" gen "$cue_file" -O "$tmp_dir" -T "contracts/hof/proto.tmpl:contract=$output" --no-format
}

generate_proto_tmp contracts/proto/apic/v1/apic.cue apic.proto
generate_proto_tmp contracts/proto/hostagent/v1/hostagent.cue hostagent.proto
generate_proto_tmp contracts/proto/p2p/v1/app.cue app.proto
generate_proto_tmp contracts/proto/p2p/v1/image.cue image.proto
generate_proto_tmp contracts/proto/p2p/v1/instance.cue instance.proto
generate_proto_tmp contracts/proto/p2p/v1/peerdb.cue peerDB.proto
generate_proto_tmp contracts/proto/p2p/v1/pinger.cue pinger.proto

"$HOF" gen internal/db/contracts/sql/protos/v0_0/contract.cue \
  -O "$tmp_dir" \
  -T internal/db/contracts/hof/sql.tmpl:contract=protos_01_tables_sql --no-format
"$HOF" gen internal/db/contracts/sql/protos/v0_0/contract.cue \
  -O "$tmp_dir" \
  -T internal/db/contracts/hof/sql_undo.tmpl:contract=protos_01_tables_undo_sql --no-format
"$HOF" gen internal/db/contracts/sql/protos/v0_1/contract.cue \
  -O "$tmp_dir" \
  -T internal/db/contracts/hof/go_sq_models.tmpl:contract=models.go --no-format
"$HOF" gen internal/db/contracts/sql/protos/v0_1/contract.cue \
  -O "$tmp_dir" \
  -T internal/db/contracts/hof/go_sql_version.tmpl:contract=protos_v0_1_gen.go --no-format
"$HOF" gen internal/db/contracts/sql/protos/v0_0/contract.cue \
  -O "$tmp_dir" \
  -T internal/db/contracts/hof/go_sql_version.tmpl:contract=protos_v0_0_gen.go --no-format
"$HOF" gen internal/db/contracts/sql/protos/catalog.cue \
  -O "$tmp_dir" \
  -T internal/db/contracts/hof/go_sql_catalog.tmpl:catalog=protos_catalog_gen.go --no-format

gofmt -w "$tmp_dir/models.go" "$tmp_dir/protos_v0_0_gen.go" "$tmp_dir/protos_v0_1_gen.go" "$tmp_dir/protos_catalog_gen.go"

diff -u apic/proto/apic.proto "$tmp_dir/apic.proto"
diff -u internal/hostagent/proto/hostagent.proto "$tmp_dir/hostagent.proto"
diff -u internal/p2p/proto/app.proto "$tmp_dir/app.proto"
diff -u internal/p2p/proto/image.proto "$tmp_dir/image.proto"
diff -u internal/p2p/proto/instance.proto "$tmp_dir/instance.proto"
diff -u internal/p2p/proto/peerDB.proto "$tmp_dir/peerDB.proto"
diff -u internal/p2p/proto/pinger.proto "$tmp_dir/pinger.proto"
diff -u internal/db/migrations/protos_01_tables.sql "$tmp_dir/protos_01_tables_sql"
diff -u internal/db/migrations/protos_01_tables.undo.sql "$tmp_dir/protos_01_tables_undo_sql"
diff -u internal/db/models.go "$tmp_dir/models.go"
diff -u internal/db/contracts/sql/protos/v0_0/contract_gen.go "$tmp_dir/protos_v0_0_gen.go"
diff -u internal/db/contracts/sql/protos/v0_1/contract_gen.go "$tmp_dir/protos_v0_1_gen.go"
diff -u internal/db/contracts/sql/protos/catalog_gen.go "$tmp_dir/protos_catalog_gen.go"
