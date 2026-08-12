# Durable operations on Swarmion

Swarmion accepts and publishes a successful write locally before checkpointing,
durable materialization, and remote convergence finish. Backend workflows must
therefore use an operation identity and its exact event/root receipt; a Dolt
root alone is not an operation identity.

## Instance-delete recovery

New instance-delete tasks use `immutable_operation_facts_v1`:

1. The queued task contains a stable random operation key, intent digest,
   original `AuthorPeerID`, and the expected delete invariant.
2. The final business delete and one deterministic `operation_effect` row in
   `task_operation_facts` commit in the same operation transaction. Seeing that
   fact therefore proves which logical delete produced the SQL state.
3. The exact event ID, published root, event digest, author, and intent are
   recorded as a deterministic `operation_receipt` fact. Any peer that learns
   the original author-scoped Swarmion binding derives identical content.
4. Task status, progress, owner, timestamps, checkpoint observations, and proof
   results remain mutable projections. They are never part of an immutable
   fact and are not recovery authority.

This removes the post-delete mutable receipt checkpoint (formerly called T92)
from the correctness path. A recovering peer waits for the original operation
binding, observes that exact event through `applied_durably`, verifies that the
atomic effect fact is present, and resumes the projection. It does not publish
another delete.

Tasks created before this model are rejected as unsupported rather than routed
through the removed mutable-checkpoint implementation. A foreign operation miss
is not authoritative absence: no timeout permits replay. Restore the original
author history or perform an explicit, application-verified operator
migration/fence before taking ownership of such a task.

The `task_operation_facts` table is part of the regenerated v0.0 snapshot for
new repositories. Later schema additions use explicit versioned migrations;
legacy repositories are adopted only when their recorded schema matches a
supported migration boundary and otherwise fail closed.

## Receipt and content semantics

- `Commit` proves local acceptance/publication, not checkpointing, durability,
  remote convergence, or an application invariant.
- `pending`, `parked_*`, and foreign `unavailable` must be observed; they never
  authorize republishing. Only a result explicitly marked safe to publish does.
- `applied_durably` proves checkpoint application. It is independent of full
  historical-root coverage. `content_dissent` is a passive, revisitable
  observation: it is not durable success, parking, or a reason to replay an
  operation or start an automatic catch-up loop.
- Destructive workflows must query their business invariant `AS OF` the durable
  checkpoint commit returned for the exact receipt.

For ordinary desired-state writes, task enqueue/progress, app lifecycle, route
configuration, and VM deployment state, use receipt availability instead of
waiting for `applied_durably`:

1. Publish and retain the exact event/root receipt.
2. Passively observe `WaitReceiptAvailability` with `MinimumOtherPeers: 1`.
3. Report `local_accepted` or `other_peer_available` as distinct stages.

An availability success proves one other peer retained the live event/root
closure. It does not prove a checkpoint, canonical application, content
coverage, quorum, or global convergence. A bounded unavailable result is an
accepted write whose replication is not yet provable; it never authorizes a
replay. Fresh single-peer/bootstrap repositories return at `local_accepted`
without waiting for an impossible second-peer proof.

Migrations and provider deletion remain explicit exceptions. Migrations consume
a checkpoint snapshot. Provider deletion keeps immutable authorization `P` and
final deletion `D` on exact `applied_durably` receipts with their `AS OF`
business-invariant checks.

Use Swarmion's public operation-receipt and exact-event wait helpers with a
caller-owned deadline. They do not hold the SQL workspace while waiting.

## SQL adapter expectations

`SQLDB()` exposes a serialized mutable workspace, not independent database
sessions. Swarmion buffers result rows and provides several adapter connections
so an unclosed result no longer pins all unrelated calls, but workspace
statements are still serialized. Inspect `SQLCapabilities()` rather than
assuming transaction queries, isolation selection, named parameters, or
`LastInsertId` support.

Use `RunOperationTransaction` for idempotent mutations. Its typed error reports
the failing phase, commit status, and rollback result. The backend consumes
those fields directly for exact transaction metrics; only a custom runner that
discards `OperationTransactionError`, or `database/sql` reporting that cleanup
already happened without revealing the driver rollback outcome, increments the
opaque-lifecycle counter.

Backend operations with caller-stable keys always select
`OperationNoChangePolicyRecordReceipt`. If their SQL body changes no content,
the same-root event and exact receipt still consume the key, and a replay skips
the body. This prevents a delete/update that is a no-op today from executing
later under the same key after database state changes. Ordinary convenience
updates/deletes keep conventional SQL semantics instead: a no-op succeeds with
no event or receipt, using Swarmion's explicit transaction no-change
observation rather than inferring the outcome from receipt absence.

Inspect and compare-and-swap discard ordinary drafts through the public draft
API; never issue raw `DOLT_RESET('--hard')` against the shared workspace.

## Dependency updates

The `protocol`, `runtime`, `transports`, `transport-adapters`, `cue`, and
`declarative` Swarmion modules are one release unit. Pin all six to the same
immutable pseudo-version and validate their resolved module origins with
`GOWORK=off` before updating this backend.
