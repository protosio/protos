#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

HOF="$("$ROOT_DIR/scripts/ensure-hof.sh")"

"$ROOT_DIR/scripts/generate-protobuf.sh"

"$HOF" gen internal/db/contracts/sql/protos/v0_0/contract.cue \
  -T internal/db/contracts/hof/sql.tmpl:contract=internal/db/migrations/protos_01_tables_sql --no-format
mv internal/db/migrations/protos_01_tables_sql internal/db/migrations/protos_01_tables.sql
"$HOF" gen internal/db/contracts/sql/protos/v0_0/contract.cue \
  -T internal/db/contracts/hof/sql_undo.tmpl:contract=internal/db/migrations/protos_01_tables_undo_sql --no-format
mv internal/db/migrations/protos_01_tables_undo_sql internal/db/migrations/protos_01_tables.undo.sql
"$HOF" gen internal/db/contracts/sql/protos/v0_1/contract.cue \
  -T internal/db/contracts/hof/go_sq_models.tmpl:contract=internal/db/models.go --no-format
"$HOF" gen internal/db/contracts/sql/protos/v0_1/contract.cue \
  -T internal/db/contracts/hof/go_sql_version.tmpl:contract=internal/db/contracts/sql/protos/v0_1/contract_gen.go --no-format
"$HOF" gen internal/db/contracts/sql/protos/v0_0/contract.cue \
  -T internal/db/contracts/hof/go_sql_version.tmpl:contract=internal/db/contracts/sql/protos/v0_0/contract_gen.go --no-format
"$HOF" gen internal/db/contracts/sql/protos/catalog.cue \
  -T internal/db/contracts/hof/go_sql_catalog.tmpl:catalog=internal/db/contracts/sql/protos/catalog_gen.go --no-format

gofmt -w internal/db/models.go \
	internal/db/contracts/sql/protos/v0_0/contract_gen.go \
	internal/db/contracts/sql/protos/v0_1/contract_gen.go \
	internal/db/contracts/sql/protos/catalog_gen.go
