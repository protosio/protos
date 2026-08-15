# Durable operations on Swarmion

Swarmion accepts and publishes a successful write locally before checkpointing,
durable materialization, and remote convergence finish. Backend workflows must
therefore use an operation identity and its exact event/root receipt; a Dolt
root alone is not an operation identity.

## Instance-delete recovery

Instance-delete tasks use the `immutable_operation_facts` recovery model:

1. The queued task contains a stable random operation key, intent digest,
   original `AuthorPeerID`, and the expected delete invariant.
2. The final business delete and one deterministic `operation_effect` row in
   `task_operation_facts` publish in the same `Execute` request. Seeing that
   fact therefore proves which logical delete produced the SQL state.
3. The validated publication address and exact event/root receipt are recorded
   as a deterministic `published_operation_receipt` fact. Its payload contains
   only event/root and operation/author/intent identity, so every peer using
   the current public receipt contract derives identical content.
4. Task status, progress, owner, timestamps, checkpoint observations, and proof
   results remain mutable projections. They are never part of an immutable
   fact and are not recovery authority.

This removes the post-delete mutable receipt checkpoint (formerly called T92)
from the correctness path. A recovering peer waits for the original operation
binding, observes that exact event through `applied_durably`, verifies that the
atomic effect fact is present, and resumes the projection. It does not publish
another delete.

The receipt fact has one current encoding and is not projected into the mutable
task payload. Recovery resolves the immutable operation and reconstructs mutable
observations from the exact receipt instead of trusting task progress state.

The unversioned recovery model, fact kinds, fact-row identity domain, delete
operation identity domain, and peer-drain authorization identity domain form a
fresh persisted-state boundary. Tasks and facts written with the retired
domains are deliberately unsupported and are not migrated through compatibility
branches. Such state must be rebuilt from a fresh repository; it must never be
treated as authoritative absence or permission to replay a destructive action.
A foreign operation miss is likewise not authoritative absence: no timeout
permits replay. Restore the original author history or perform an explicit,
application-verified operator migration/fence before taking ownership of such a
task.

The `task_operation_facts` table is part of the generated initial schema for new
repositories. Later schema additions use explicit migrations; repositories are
adopted only when their recorded schema matches a supported migration boundary
and otherwise fail closed.

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
2. Passively call `WaitReceipt` with an `OtherPeerRetention` requirement of one
   peer and the `OtherPeerRetained` condition.
3. Report `local_accepted` or `other_peer_available` as distinct stages.

An availability success proves one other peer retained the live event/root
closure. It does not prove a checkpoint, canonical application, content
coverage, quorum, or global convergence. A bounded unavailable result is an
accepted write whose replication is not yet provable; it never authorizes a
replay. Fresh single-peer/bootstrap repositories return at `local_accepted`
without waiting for an impossible second-peer proof. For ordinary unscoped
requests, Swarmion selects the current database-scoped logical peers at each
observation boundary. `EligiblePeerIDs` is that topology snapshot, while
`Peers` contains receipt evidence only and may still be empty for a newly
authored receipt. `NoCurrentEligiblePeers` is the explicit weak single-peer
outcome; it is not other-peer availability and grants no replay or destructive
workflow authority. Use stable `ReasonCode` values for control flow, never the
diagnostic `Reason` prose. Pass explicit `PeerIDs` only for intentionally fixed
application-owned replication targets.

Migrations and provider deletion remain explicit exceptions. Migrations consume
a checkpoint snapshot. Provider deletion keeps immutable authorization `P` and
final deletion `D` on exact `applied_durably` receipts with their `AS OF`
business-invariant checks.

Use Swarmion's public `OperationAddress`/`OperationResolution` and
`ObserveReceipt`/`WaitReceipt` helpers with a caller-owned deadline. They do
not hold the SQL workspace while waiting.

## SQL adapter expectations

The public runtime does not expose a mutable `*sql.DB`. `QuerySQL` returns
buffered rows for scoped reads, and this backend's `database/sql` adapter is
read-only. Every mutation enters `DatabaseRuntime.Execute` with an immutable
`OperationIdentity`; its `PublicationOutcome` is the only retry/publication
authority.

An accepted or recorded no-change `PublicationOutcome` consumes the stable
identity and carries an exact receipt; a replay resolves that same outcome and
does not rerun the SQL body. Rejected-safe-to-retry is the sole mutation retry
authority. Unresolved, inconclusive, unavailable, unknown, and terminal
outcomes remain non-authorizing and must be recovered through the complete
outcome/address contract.

## Dependency updates

The `protocol`, `runtime`, `transports`, `transport-adapters`, `cue`, and
`declarative` Swarmion modules are one release unit. Pin all six to the same
immutable pseudo-version and validate their resolved module origins with
`GOWORK=off` before updating this backend.
